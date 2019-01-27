package main

import (
	"crypto/aes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	accm "github.com/pschlump/AesCCM"
	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
)

type Upload1 struct {
	ApiKey  string `yaml:"api_key"`
	Client  string `yaml:"client_url,omitempty"`
	Name    string `yaml:"name"`
	Service string `yaml:"service_url"`
}

func Up1(c, s, apiKey, n string) *Upload1 {
	u := &Upload1{
		apiKey, c, n, s,
	}

	if c == "" {
		u.Client = s
	}
	return u
}

func (u *Upload1) baseURL() string {
	return trimSlash(u.Service)
}

func trimSlash(su string) string {
	if strings.HasSuffix(su, "/") {
		su = su[len(su)-1:]
	}
	return su
}

func (u *Upload1) client() string {
	if u.Client == "" {
		return u.baseURL()
	}

	return trimSlash(u.Client)
}

func (u *Upload1) ServiceName() string {
	return u.Name
}

type ccmParameters struct {
	key   []byte
	iv    []byte
	ident []byte
}

func deriveParams(seed []byte) *ccmParameters {
	_hsh := sha512.New()
	_hsh.Write(seed)
	hsh := _hsh.Sum(nil)

	return &ccmParameters{
		key:   hsh[:32],
		iv:    hsh[32:48],
		ident: hsh[48:64],
	}
}

func ccmEncrypt(file, seed []byte) ([]byte, []byte, error) {
	params := deriveParams(seed)

	aes, err := aes.NewCipher(params.key)
	if err != nil {
		return nil, nil, err
	}

	nonceSize := findIvLen(len(file))

	ccm, err := accm.NewCCM(aes, 8, nonceSize)
	if err != nil {
		return nil, nil, err
	}

	encrypted := ccm.Seal(nil, params.iv[:nonceSize], file, nil)

	return encrypted, params.ident, nil
}

func (u *Upload1) Upload(contentType string, name string, data io.Reader) (string, error) {
	buf := etc.NewBuffer()
	io.Copy(buf, data)

	header, _ := json.Marshal(
		struct {
			Mime string `json:"mime"`
			Name string `json:"name"`
		}{
			contentType,
			name,
		},
	)

	envelope := etc.NewBuffer()
	headerCode := []rune(string(header))

	for i := 0; i < len(headerCode); i++ {
		envelope.WriteBigUint16(uint16(headerCode[i]))
	}

	envelope.WriteBigUint16(0)
	envelope.Write(buf.Bytes())

	_seed := etc.NewBuffer()
	_seed.WriteRandom(16)
	seed := _seed.Bytes()

	ciphertext, ident, err := ccmEncrypt(envelope.Bytes(), seed)
	if err != nil {
		return "", err
	}

	envelope = nil

	e := etc.NewBuffer()
	w := multipart.NewWriter(e)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="file"`)

	h.Set("Content-Type", "text/plain")
	file, err := w.CreatePart(h)
	if err != nil {
		return "", err
	}

	idt := urlEncode(ident)
	file.Write(ciphertext)
	yo.Warn("Ident", idt)
	w.WriteField("api_key", u.ApiKey)
	w.WriteField("ident", idt)
	w.Close()

	ciphertext = nil

	client := &http.Client{}

	urlstr := u.baseURL() + "/up"
	yo.Ok("POST'ing to ", urlstr)

	prx := &proxyReader{
		e,
		e.Size(),
		0,
	}

	req, err := http.NewRequest("POST", u.baseURL()+"/up", prx)
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if err != nil {
		return "", err
	}

	ht, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if ht.StatusCode != 200 {
		return "", fmt.Errorf("server returned %s", ht.Status)
	}

	str := etc.NewBuffer()
	io.Copy(str, ht.Body)
	yo.Warn(str.String())

	return u.client() + "/#" + urlEncode(seed), nil
}

func urlEncode(str []byte) string {
	uri := base64.URLEncoding.EncodeToString(str)
	return strings.Replace(uri, "=", "", -1)
}

func findIvLen(bufferLength int) int {
	if bufferLength < 0xFFFF {
		return 15 - 2
	}

	if bufferLength < 0xFFFFFF {
		return 15 - 3
	}

	return 15 - 4
}
