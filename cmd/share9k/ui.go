package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"

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
	h.HandleFunc("/preview/{id}", func(rw http.ResponseWriter, r *http.Request) {
		u, _ := etc.ParseUUID(mux.Vars(r)["id"])
		i, ok := previews.Load(u)
		if ok {
			b := i.(*etc.Buffer).Bytes()
			rw.Header().Set("Content-Type", "image/png")
			rw.Write(b)
		}
	})
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

var previews = new(sync.Map)

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
		if runtime.GOOS == "windows" {
			cmd := exec.Command("C:\\Windows\\System32\\rundll32.exe", "url.dll,FileProtocolHandler", etc.LocalDirectory().Concat("s9k_config.txt").Render())
			err := cmd.Run()
			if err != nil {
				yo.Warn(err)
			}
		} else {
			open.Run(etc.LocalDirectory().Concat("s9k_config.txt").Render())
		}
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
	uid := etc.GenerateRandomUUID()
	yo.Ok("storing in preview", uid)
	previews.Store(uid, buf)

	thumbX := rect.Max.X - rect.Min.X
	thumbY := rect.Max.Y - rect.Min.Y

	thumbX += 32
	thumbY += 64

	yo.Ok("Thumbnail dimensions", thumbX, thumbY)

	fmt.Println(base64Img)

	cmd := exec.Command(os.Args[0], "review-image", ints(thumbX), ints(thumbY), ints(port), uid.String())
	output := etc.NewBuffer()
	cmd.Stderr = output
	cmd.Stdout = output
	cmd.Run()

	boolean := strings.TrimRight(output.ToString(), "\r\n")
	yo.Okf("returned %s\n", boolean)
	if boolean == "true" {
		go uploadReader(uploadName, uploadType, uploadSize, uploadFile)
	}

	previews.Delete(uid)
}

func ints(i int) string {
	return fmt.Sprintf("%d", i)
}
