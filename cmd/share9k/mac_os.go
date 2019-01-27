package main

import (
	"C"
	"github.com/superp00t/etc"
	"os/exec"
	"github.com/go-yaml/yaml"
	"strings"
	"strconv"
)

type sysProfile struct {
	GD map[string]*GPU `yaml:"Graphics/Displays"`
}

type GPU struct {
	Displays map[string]*Display `yaml:"Displays"`
}

type Display struct {
	Main bool `yaml:"Main Display"`
	Resolution string `yaml:"Resolution"`
}

//export getScreenResMac
func getScreenResMac(x, y, w, h *C.int) {
	*x = C.int(0)
	*y = C.int(0)

	buffer := etc.NewBuffer()

	cmd := exec.Command("system_profiler", "SPDisplaysDataType")

	cmd.Stdout = buffer
	cmd.Stderr = buffer

	cmd.Run()

	var ss sysProfile
	yaml.Unmarshal(buffer.Bytes(), &ss)

	for _, v := range ss.GD {
		for _, d := range v.Displays {
			if d.Main {
				res := d.Resolution
				if strings.Contains(d.Resolution, " @") {
					res = strings.Split(d.Resolution, " @")[0]
					return
				}

				els := strings.Split(res, " x ")
				ww, _ := strconv.ParseInt(els[0], 0, 64)
				hh, _ := strconv.ParseInt(els[1], 0, 64)
				
				*w = C.int(int(ww))
				*h = C.int(int(hh))
				return
			}
		}
	}
	
} 