package main

import (
	"github.com/superp00t/etc/yo"
	"github.com/superp00t/keystroke"
)

var kstroke *keystroke.Processor

func loadKeybinds(kb map[string]string) {
	restart := false

	if kstroke == nil {
		kstroke = keystroke.New()
	} else {
		kstroke.Clear()
		restart = true
	}

	if kb["SnapRegion"] != "" {
		kstroke.On(kb["SnapRegion"], snapRegion)
	}

	if kb["SnapScreen"] != "" {
		kstroke.On(kb["SnapScreen"], screenShot)
	}

	if !restart {
		go func() {
			err := kstroke.Run()
			if err != nil {
				yo.Fatal(err)
			}
		}()
	}
}
