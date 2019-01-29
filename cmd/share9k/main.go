package main

/*
#include <stdio.h>
#include "select.h"
#define _THREAD_SAFE 1
#cgo linux LDFLAGS: -lX11 -lX11 -lXi
#cgo windows pkg-config: sdl2
#cgo darwin CFLAGS: -I/usr/local/include
#cgo darwin LDFLAGS: -L/usr/local/lib -lSDL2
*/
import "C"

import (
	"fmt"
	"image/png"

	"os"
	"os/exec"

	"github.com/go-yaml/yaml"
	"github.com/zserge/webview"

	"encoding/json"
	"image"
	"io"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"

	"github.com/kbinani/screenshot"
)

func init() {
	// This is necessary for SDL logic,
	// which must be run in func main in order to be truly cross-platform
	runtime.LockOSThread()
}

type Opts struct {
	Service       string `yaml:"service"`
	ReviewUploads bool   `yaml:"review_uploads"`
}

var _Opts Opts

var constantUploaders = []FileService{
	Img9k(),
	EncryptedImg9k(),
	Imgur(),
}

var Uploaders = constantUploaders

var Res = []string{
	"640x480",
	"512x512",
}

var chosenRes string = Res[0]
var chosenDL = Uploaders[0]
var Cfg *Config

type FileService interface {
	ServiceName() string
	Upload(string, string, io.Reader) (string, error)
}

func setOpts(o Opts) {
	_Opts = o
	opts := etc.LocalDirectory().Concat(".s9k_opts").Render()
	writeYaml(opts, _Opts)
}

func loadConfig() {
	loc := etc.LocalDirectory()
	if !loc.IsExtant() {
		os.MkdirAll(loc.Render(), 0700)
	}
	config := loc.Concat("s9k_config.txt")
	if !config.IsExtant() {
		dc := DefaultConfig
		dc = strings.Replace(dc, "\n", "\r\n", -1)
		ioutil.WriteFile(config.Render(), []byte(dc), 0700)
	}

	Cfg = new(Config)

	b, err := ioutil.ReadFile(config.Render())
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(b, Cfg)
	if err != nil {
		panic(err)
	}
	loadKeybinds(Cfg.Keybindings)
	Uploaders = constantUploaders
	for _, v := range Cfg.Up1 {
		Uploaders = append(Uploaders, v)
	}

	setUploaders()
}

func parseInt(s string) int {
	i, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		yo.Fatal(err)
	}

	return int(i)
}

func main() {
	yo.AddSubroutine("select-region", []string{"aspect ratio width", "aspect ratio height"}, "snaps region", func(args []string) {
		arx := parseInt(args[0])
		ary := parseInt(args[1])

		var x, y, w, h C.int
		C.acquire_rectangle(C.int(arx), C.int(ary), &x, &y, &w, &h)

		fmt.Println(int(x), int(y), int(w), int(h))
	})

	yo.AddSubroutine("review-image", []string{"x", "y", "port", "uid"}, "review an image", func(args []string) {
		port = parseInt(args[2])
		race := make(chan bool)
		closed := false
		go func() {
			fmt.Println(<-race)
			os.Exit(0)
		}()

		urls := fmt.Sprintf("http://localhost:%d/img.html", port)

		w = webview.New(webview.Settings{
			Width:     parseInt(args[0]),
			Height:    parseInt(args[1]),
			Title:     "Review image",
			Resizable: true,
			URL:       urls,
			ExternalInvokeCallback: func(ev webview.WebView, str string) {
				var s []string
				json.Unmarshal([]byte(str), &s)

				switch s[0] {
				case "pageload":
					ev.Dispatch(func() {
						if err := ev.Eval(
							fmt.Sprintf(
								`document.getElementById("show-scrot").innerHTML = '<img src="http://localhost:%d/preview/%s"></img>';`, port, args[3])); err != nil {
							yo.Warn(err)
						}
					})
				case "upload":
					closed = true
					ev.Terminate()
					race <- true
				}
			},
		})

		defer w.Exit()
		w.Run()
		if !closed {
			race <- false
		}
		time.Sleep(300 * time.Hour)
	})

	yo.Main("launches GUI", func(args []string) {
		pth := etc.LocalDirectory().Concat(".s9k_opts")
		if !pth.IsExtant() {
			setOpts(Opts{
				ReviewUploads: true,
				Service:       "",
			})
		} else {
			b, err := ioutil.ReadFile(pth.Render())
			if err != nil {
				yo.Fatal(err)
			}

			yaml.Unmarshal(b, &_Opts)
			for _, v := range Uploaders {
				if v.ServiceName() == _Opts.Service {
					chosenDL = v
					break
				}
			}
		}

		loadConfig()

		runUI()
	})

	yo.Init()
}

func screenShot() {
	var x, y, w, h C.int
	C.acquire_full_desktop_info(&x, &y, &w, &h)

	rect := image.Rect(
		int(x),
		int(y),
		int(x)+int(w),
		int(y)+int(h),
	)

	displayUploadRect(rect)
}

func snapRegion() {
	output := etc.NewBuffer()
	cmd := exec.Command(os.Args[0], "select-region", "0", "0")
	cmd.Stdout = output
	cmd.Run()

	str := strings.Split(
		strings.TrimRight(output.ToString(), "\r\n "),
		" ",
	)

	x, y, w, h := parseInt(str[0]), parseInt(str[1]), parseInt(str[2]), parseInt(str[3])

	// var x, y, w, h C.int
	// C.acquire_rectangle(C.int(0), C.int(0), &x, &y, &w, &h)

	rect := image.Rect(
		x,
		y,
		x+w,
		y+h,
	)

	if w == 0 || h == 0 {
		return
	}

	go displayUploadRect(rect)
}

func displayUploadRect(rect image.Rectangle) {
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		yo.Fatal(err)
	}

	buf := etc.NewBuffer()
	png.Encode(buf, img)

	uploadFile = buf
	uploadName = "screenshot.png"
	uploadSize = buf.Size()
	uploadType = "image/png"

	if !_Opts.ReviewUploads {
		go uploadReader(uploadName, uploadType, uploadSize, uploadFile)
		return
	}

	displayImage("Share Screenshot", img)
}

type proxyReader struct {
	r    io.Reader
	sz   int64
	read int64
}

func (pr *proxyReader) fraction() float64 {
	return float64(pr.read) / float64(pr.sz)
}

func (pr *proxyReader) Read(b []byte) (int, error) {
	i, err := pr.r.Read(b)
	pr.read += int64(i)
	frac := pr.fraction()
	yo.Warn("Read ", i, "bytes", "(", frac, ")")
	go setFrac(int(frac * 100))
	return i, err
}

func uploadReader(filename, contentType string, size int64, data io.Reader) {
	go setFrac(0)
	status("Uploading...")
	url, err := chosenDL.Upload(contentType, filename, data)
	if err != nil {
		status(err.Error())
		return
	}

	clipboard.WriteAll(url)
	status("Copied URL to clipboard")
}

func stopRecording() {
}

func status(s string) {
	// _status.SetText(s)
	w.Dispatch(func() {
		w.Eval(`document.getElementById("status").textContent = "` + s + `";`)
	})
}

func setFrac(v int) {
	w.Dispatch(func() {
		yo.Warn("setting fraction to", v)
		js := `document.getElementById("prog").setAttribute("style", "width: ` + fmt.Sprintf("%d", v) + `%;");`
		yo.Ok("Evald: ", js)
		w.Eval(js)
	})
}
