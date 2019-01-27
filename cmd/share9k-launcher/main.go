package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/superp00t/go-minisign"

	"github.com/superp00t/etc"
	"github.com/superp00t/etc/yo"
	"github.com/superp00t/vsn"
)

const signingKey = `RWQoR3JU+2TIxCwvlTnHul+NpftbHWynABY2z2IVF6imAPAWgB7EhBs0`
const host = "https://img.ikrypto.club"

func main() {
	yo.Main("update share9k", _main)
	yo.Init()
}

func get(url string) (string, error) {
	r, err := http.Get(host + url)
	if err != nil {
		return "", err
	}

	if r.StatusCode != 200 {
		return "", fmt.Errorf("got status %s", r.Status)
	}

	e := etc.NewBuffer()
	_, err = io.Copy(e, r.Body)
	if err != nil {
		return "", err
	}

	return e.ToString(), nil
}

func trim(v string) string {
	v = strings.Replace(v, " ", "", -1)
	v = strings.Replace(v, "\n", "", -1)
	v = strings.Replace(v, "\r", "", -1)
	v = strings.Replace(v, "\t", "", -1)
	return v
}

func loadVersion() (vsn.Version, error) {
	spath := etc.ParseSystemPath(os.Args[0])
	if len(spath) == 0 {
		ppath, err := os.Getwd()
		if err != nil {
			yo.Warn(err)
		}
		spath = append(etc.ParseSystemPath(ppath), spath...)
	}
	yo.Spew([]string(spath))
	path := spath[:len(spath)-2].Concat("version.txt")

	if path.IsExtant() == false {
		return "", fmt.Errorf("version file does not exist")
	}

	f, err := etc.FileController(path.Render())
	if err != nil {
		return "", err
	}

	str := trim(string(f.ReadRemainder()))

	return vsn.Version(str), nil
}

func getBin() etc.Path {
	spath := etc.ParseSystemPath(os.Args[0])
	path := spath[:len(spath)-1]
	return path
}

func _main(args []string) {
	vers, err := get("/share9k/version.txt")
	if err != nil {
		launch()
		return
	}

	serverVersion := vsn.Version(vers)
	installedVersion, err := loadVersion()
	if err != nil {
		yo.Warn(err)
		launch()
		return
	}

	if serverVersion.IsGreaterThan(installedVersion) {
		yo.Warnf("%s is greater than %s\n", serverVersion, installedVersion)

		pkg := fmt.Sprintf("/share9k/share9k-%s-%s-%s.zip", runtime.GOOS, runtime.GOARCH, serverVersion)

		_sig, err := get(pkg + ".minisig")
		if err != nil {
			launch()
			return
		}

		yo.Warn(_sig)

		pk, err := minisign.NewPublicKey(signingKey)
		if err != nil {
			die(err)
		}

		sig, err := minisign.DecodeSignature(_sig)
		if err != nil {
			die(err)
		}

		h, err := http.Get(host + pkg)
		if err != nil {
			launch()
			return
		}

		randomZip := etc.TmpDirectory().Concat(etc.GenerateRandomUUID().String()).Render() + ".zip"

		yo.Ok("Downloading to ", randomZip)

		f, err := etc.FileController(randomZip)
		if err != nil {
			die(err)
		}

		_, err = io.Copy(f, h.Body)
		if err != nil {
			die(err)
		}
		f.Close()

		ok, err := pk.VerifyFromFile(randomZip, sig)
		if !ok {
			yo.Fatalf("could not verify sig %s", err)
		}

		z, err := zip.OpenReader(randomZip)
		if err != nil {
			die(err)
		}

		path := getBin()
		path = path[:len(path)-1]

		for _, f := range z.File {
			name := etc.ParseUnixPath(f.FileHeader.Name)
			bound := path.Concat(name...).Render()
			if strings.HasSuffix(f.FileHeader.Name, "/") {
				os.MkdirAll(bound, 0700)
				continue
			}

			yo.Ok("writing file", bound)

			fl := etc.NewBuffer()

			flr, err := f.Open()
			if err != nil {
				die(err)
			}

			_, err = io.Copy(fl, flr)
			if err != nil {
				yo.Warn(err)
			}

			err = ioutil.WriteFile(bound, fl.Bytes(), 0700)
			if err != nil {
				yo.Warn(err)
			}

			flr.Close()
		}

		z.Close()
		os.Remove(randomZip)
	}

	launch()
}

func launch() {
	exe := getBin().Concat("share9k").Render()
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}

	yo.Ok("Launching", exe)

	cmd := exec.Command(exe)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	die(cmd.Run())
}

func die(err error) {
	yo.Warn(err)
	time.Sleep(20 * time.Second)
}
