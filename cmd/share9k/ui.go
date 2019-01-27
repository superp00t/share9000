package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/superp00t/etc"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
	"github.com/superp00t/etc/yo"
	"github.com/zserge/webview"

	"github.com/h2non/filetype"
	"github.com/skratchdot/open-golang/open"
)

var w webview.WebView
var port int

func runUI() {
	var l net.Listener
	var err error

	for port = 30000; port < 65535; port++ {
		l, err = net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
		if err != nil {
			continue
		} else {
			goto success
		}
	}

	yo.Fatal(err, "port bind exhausted. Something is using up all your ports, probably.")

success:

	h := mux.NewRouter()
	h.PathPrefix("/").Handler(http.FileServer(assetFS()))

	go func() {
		yo.Fatal(http.Serve(l, h))
	}()

	w = webview.New(webview.Settings{
		Width:     200,
		Height:    600,
		Title:     "SHARE9K",
		Resizable: false,
		URL:       fmt.Sprintf("http://localhost:%d/", port),
		ExternalInvokeCallback: handleRPC,
	})

	defer w.Exit()
	w.Run()
}

// Updates the uploader selection list
func setUploaders() {
	if w == nil {
		return
	}

	activeIndex := 0
	for i, v := range Uploaders {
		if v.ServiceName() == _Opts.Service {
			activeIndex = i
		}
	}

	var s []string
	for _, v := range Uploaders {
		s = append(s, v.ServiceName())
	}

	obj, _ := json.Marshal(s)

	script := fmt.Sprintf(`(function(uploaders, activeIndex, review) {
			var list = document.getElementById("uploaders");
			while(list.firstChild) {
				list.removeChild(list.firstChild);
			}
			for (var i = 0; i < uploaders.length; i++) {
				var el = document.createElement("option");
				el.setAttribute("value", i.toString());
				el.innerHTML = uploaders[i];
				if (i == activeIndex) {
					el.setAttribute("selected", true);
				}
				list.appendChild(el);
			}
			var ru = document.getElementById("review-uploads");
			ru._checked = review;
			ru.checked = review;
	})(%s, %d, %t);`, string(obj), activeIndex, _Opts.ReviewUploads)
	yo.Ok(script)
	err := w.Eval(script)
	if err != nil {
		yo.Warn(err)
	}
}

func onExec() {
	yo.Ok("Page loaded")
	setUploaders()
}

func handleRPC(webv webview.WebView, str string) {
	var s []string

	json.Unmarshal([]byte(str), &s)

	yo.Ok(str)

	switch s[0] {
	case "pageload":
		webv.Dispatch(onExec)
	case "scrot":
		go screenShot()
	case "snip-scrot":
		go snapRegion()
	case "change-uploader":
		bstr, _ := strconv.ParseInt(s[1], 0, 64)
		chosenDL = Uploaders[int(bstr)]
		yo.Println("changed uploader to", chosenDL.ServiceName())
		setOpts(Opts{
			chosenDL.ServiceName(),
			_Opts.ReviewUploads,
		})
	case "file-select":
		filePath := webv.Dialog(webview.DialogTypeOpen, webview.DialogFlagFile, "Select file for upload", "")
		yo.Ok("Filepath:", filePath)
		if filePath != "" {
			fiStat, err := os.Stat(filePath)
			if err == nil {
				if fiStat.IsDir() == false {
					file, err := ioutil.ReadFile(filePath)
					if err != nil {
						yo.Fatal(err)
					}

					name := "share.bin"
					typ := "application/octet-stream"

					kind, err := filetype.Match(file)
					if err == nil {
						name = "share." + kind.Extension
						typ = kind.MIME.Value
					}

					uploadType = typ
					uploadSize = fiStat.Size()
					uploadName = name

					go uploadReader(uploadName, uploadType, uploadSize, etc.FromBytes(file))
				}
			}
		}
	case "edit-config":
		open.Run(etc.LocalDirectory().Concat("s9k_config.txt").Render())
	case "review-uploads":
		setOpts(Opts{
			_Opts.Service,
			s[1] == "true",
		})
		yo.Ok("Review uploads?", s[1])
	case "reload-config":
		loadConfig()
	}
}

var uploadSize int64
var uploadType, uploadName string
var uploadFile io.Reader
var base64Img string

func displayImage(title string, img image.Image) {
	imgData := resize.Thumbnail(600, 400, img, resize.Bicubic)
	rect := imgData.Bounds()
	// rgba := image.NewRGBA(rect)
	// draw.Draw(rgba, rect, imgData, rect.Min, draw.Src)

	buf := etc.NewBuffer()
	png.Encode(buf, imgData)

	thumbX := rect.Max.X - rect.Min.X
	thumbY := rect.Max.Y - rect.Min.Y

	thumbX += 32
	thumbY += 64

	yo.Ok("Thumbnail dimensions", thumbX, thumbY)

	base64Img = "data:image/png;base64," + url.QueryEscape(base64.StdEncoding.EncodeToString(buf.Bytes()))

	imgdisplay := webview.New(webview.Settings{
		Width:     thumbX,
		Height:    thumbY,
		Title:     title,
		Resizable: true,
		URL:       fmt.Sprintf("http://localhost:%d/img.html", port),
		ExternalInvokeCallback: handleDisplayRPC,
	})

	imgdisplay.Run()
	yo.Println("exiting...")
	imgdisplay.Exit()
}

func handleDisplayRPC(webv webview.WebView, str string) {
	var s []string

	json.Unmarshal([]byte(str), &s)

	switch s[0] {
	case "pageload":
		webv.Dispatch(func() {
			if err := webv.Eval(`document.getElementById("show-scrot").innerHTML = '<img src="` + base64Img + `"></img>';`); err != nil {
				yo.Warn(err)
			}
		})
	case "upload":
		yo.Println("terminating...")
		webv.Terminate()
		yo.Println("terminated")
		go uploadReader(uploadName, uploadType, uploadSize, uploadFile)
	}
}
