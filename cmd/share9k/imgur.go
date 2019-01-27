package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
)

type ImgurResp struct {
	// Data struct {
	// 	Hashes     []string    `json:"hashes"`
	// 	Hash       string      `json:"hash"`
	// 	Deletehash string      `json:"deletehash"`
	// 	Ticket     bool        `json:"ticket"`
	// 	Album      string      `json:"album"`
	// 	Edit       bool        `json:"edit"`
	// 	Gallery    interface{} `json:"gallery"`
	// 	Poll       bool        `json:"poll"`
	// 	Animated   bool        `json:"animated"`
	// 	Height     int         `json:"height"`
	// 	Width      int         `json:"width"`
	// 	Ext        string      `json:"ext"`
	// 	Msid       string      `json:"msid"`
	// } `json:"data"`
	Data    map[string]interface{} `json:"data"`
	Success bool                   `json:"success"`
	Status  int                    `json:"status"`
}

func Imgur() *imgur {
	return &imgur{
		"https://api.imgur.com",
	}
}

const imgurClientID = "e0741632b33c221"
const chromeUA = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36"

type imgur struct {
	service string
}

func (i *imgur) ServiceName() string {
	return "Imgur"
}

func (i *imgur) Upload(contentType string, name string, data io.Reader) (string, error) {
	ext := strings.Split(name, ".")[1]

	yo.Warn(ext)

	e := etc.NewBuffer()
	w := multipart.NewWriter(e)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="image"; filename="%s"`,
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

	str := strings.Split(name, ".")[1]

	w.WriteField("title", "SHARE9K")
	w.WriteField("type", str)

	err = w.Close()
	if err != nil {
		yo.Fatal(err)
	}

	postURL := i.service + "/3/image"

	yo.Ok("[IMGUR] Posting to", postURL)

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
	req.Header.Set("Authorization", "Client-ID "+imgurClientID)
	req.Header.Set("User-Agent", chromeUA)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.ContentLength = e.Size()
	if err != nil {
		return "", err
	}

	ht, err := client.Do(req)
	if err != nil {
		return "", err
	}

	yo.Spew(ht.Header)

	var r ImgurResp
	json.NewDecoder(ht.Body).Decode(&r)
	yo.Spew(r)

	if ht.StatusCode != 200 {
		str := etc.NewBuffer()
		io.Copy(str, ht.Body)
		yo.Warn(str.String())

		return "", fmt.Errorf("server returned %s", ht.Status)
	}

	return r.Data["link"].(string), nil
}
