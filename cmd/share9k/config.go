package main

import (
	"io/ioutil"
	"strings"

	"github.com/go-yaml/yaml"
)

type Config struct {
	Up1         []*Upload1        `yaml:"up1_servers"`
	Keybindings map[string]string `yaml:"key_bindings"`
}

const DefaultConfig = `# Enter a custom Up1 configuration here!
# To improve security, you may use a custom "client_url" parameter on any of these,
# to supply your own securely hosted client.
up1_servers:
- api_key:     d684007d2a02c40402d6b14063a8fb4b
  name:        ikrypto.club Up1
  service_url: https://up.pg.ikrypto.club
- api_key:     59Mnk5nY6eCn4bi9GvfOXhMH54E7Bh6EMJXtyJfs
  name:        Riseup Up1
  service_url: https://share.riseup.net
- api_key:     35dfa184829b3a6ae805ef1847b1fe64
  name:        XWiki SAS Up1
  service_url: https://up1.xwikisas.com
key_bindings:
  SnapScreen:  Alt+Shift+X
  SnapRegion:  Alt+Shift+Z

`

func writeYaml(path string, data interface{}) {
	bt, _ := yaml.Marshal(data)
	str := string(bt)
	str = strings.Replace(str, "\n", "\r\n", -1)
	ioutil.WriteFile(path, bt, 0700)
}
