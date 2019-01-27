package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
	"golang.org/x/crypto/nacl/secretbox"
)

type img9k struct {
	encrypted   bool
	serviceName string
	service     string
}

func Img9k() *img9k {
	return &img9k{
		false,
		"IMG9K (Unencrypted)",
		"https://img.ikrypto.club",
	}
}

func EncryptedImg9k() *img9k {
	return &img9k{
		true,
		"IMG9K (Encrypted)",
		"https://img.ikrypto.club",
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (i *img9k) ServiceName() string {
	return i.serviceName
}

func (i *img9k) serviceHost() []byte {
	u, _ := url.Parse(i.serviceName)
	return []byte(u.Hostname())
}

func (i *img9k) Upload(contentType string, name string, data io.Reader) (string, error) {
	key := make([]byte, 32)
	rand.Read(key)

	if i.encrypted {
		buf := etc.NewBuffer()
		buf.WriteUString(contentType)
		io.Copy(buf, data)

		h := hmac.New(
			sha512.New,
			key,
		)

		hash := h.Sum(i.serviceHost())
		bkey := new([32]byte)
		bnonce := new([24]byte)

		copy(bkey[:], hash[:32])
		copy(bnonce[:], hash[32:56])

		box := secretbox.Seal(nil, buf.Bytes(), bnonce, bkey)
		data = etc.FromBytes(box)

		name = "encrypted.i9k"
		contentType = "application/octet-stream"
	}

	e := etc.NewBuffer()
	w := multipart.NewWriter(e)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`,
			escapeQuotes(name)))
	h.Set("Content-Type", contentType)
	file, err := w.CreatePart(h)
	if err != nil {
		return "", err
	}

	yo.Spew(h)

	_, err = io.Copy(file, data)
	if err != nil {
		yo.Fatal(err)
	}

	err = w.Close()
	if err != nil {
		yo.Fatal(err)
	}

	postURL := i.service + "/upload"
	if i.encrypted {
		postURL += "?x=1"
	}

	yo.Ok("Posting to", postURL)

	client :=
		&http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

	prx := &proxyReader{
		e,
		e.Size(),
		0,
	}

	req, err := http.NewRequest("POST", postURL, prx)
	req.Header.Set("User-Agent", "SHARE9K")
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err != nil {
		return "", err
	}

	ht, err := client.Do(req)
	if err != nil {
		return "", err
	}

	yo.Spew(ht.Header)

	if ht.StatusCode != 301 {
		str := etc.NewBuffer()
		io.Copy(str, ht.Body)
		yo.Warn(str.String())

		return "", fmt.Errorf("server returned %s", ht.Status)
	}

	url := i.service + ht.Header.Get("Location")

	if i.encrypted {
		encoded := base64.URLEncoding.EncodeToString(key)
		url += "#" + encoded
	}

	return url, nil
}
