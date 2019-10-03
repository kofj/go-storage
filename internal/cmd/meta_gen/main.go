package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/Xuanwo/storage/define"
)

type capability struct {
	Read    bool `json:"read"`
	Write   bool `json:"write"`
	File    bool `json:"file"`
	Stream  bool `json:"stream"`
	Segment bool `json:"segment"`
}

func parseCapability(c capability) uint64 {
	v := define.Capability(0)
	if c.Read {
		v |= define.CapabilityRead
	}
	if c.Write {
		v |= define.CapabilityWrite
	}
	if c.File {
		v |= define.CapabilityFile
	}
	if c.Stream {
		v |= define.CapabilityStream
	}
	if c.Segment {
		v |= define.CapabilitySegment
	}

	return uint64(v)
}

type metadata struct {
	Name       string              `json:"name"`
	Capability capability          `json:"capability"`
	Options    map[string][]string `json:"options"`

	ParsedCapability uint64 `json:"-"`

	TypeMap map[string]string `json:"-"`
}

func main() {
	_, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatalf("read dir failed: %v", err)
	}

	metaPath := "meta.json"
	if _, err := os.Stat(metaPath); err != nil {
		log.Fatalf("stat meta failed: %v", err)
	}

	content, err := ioutil.ReadFile(metaPath)
	if err != nil {
		log.Fatalf("read file failed: %v", err)
	}

	var meta metadata

	err = json.Unmarshal(content, &meta)
	if err != nil {
		log.Fatalf("json unmarshal failed: %v", err)
	}
	meta.ParsedCapability = parseCapability(meta.Capability)
	meta.TypeMap = define.AvailableOptions

	filePath := "meta.go"
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	err = metaTmpl.Execute(f, meta)
	if err != nil {
		log.Fatal(err)
	}

	filePath = "options.go"
	f, err = os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}
	err = optionTmpl.Execute(f, meta)
	if err != nil {
		log.Fatal(err)
	}
}

var metaTmpl = template.Must(template.New("").Funcs(sprig.HermeticTxtFuncMap()).Parse(`// Code generated by go generate via internal/cmd/meta_gen; DO NOT EDIT.
package {{ .Name }}

import (
	"github.com/Xuanwo/storage/define"
)

// CapabilityRead    = {{ .Capability.Read }}
// CapabilityWrite   = {{ .Capability.Write }}
// CapabilityFile    = {{ .Capability.File }}
// CapabilityStream  = {{ .Capability.Stream }}
// CapabilitySegment = {{ .Capability.Segment }}
const capability = define.Capability({{ .ParsedCapability }})

// Capability implements Storager.Capability().
func (c *Client) Capability() define.Capability {
	return capability
}
`))

var optionTmpl = template.Must(template.New("option").Funcs(sprig.HermeticTxtFuncMap()).Parse(`// Code generated by go generate via internal/cmd/meta_gen; DO NOT EDIT.
package {{ .Name }}

import (
	"github.com/Xuanwo/storage/define"
)

{{ $Data := . }}

var allowdOptions = map[string]map[string]struct{}{
{{- range $k, $v := .Options }}
	define.Action{{ $k | camelcase}}: {
{{- range $_, $key := $v }}
		"{{$key}}": struct{}{},
{{- end }}
	},
{{- end }}
}

// IsOptionAvailable implements Storager.IsOptionAvailable().
func (c *Client) IsOptionAvailable(action, option string) bool {
	if _, ok := allowdOptions[action]; !ok {
		return false
	}
	if _, ok := allowdOptions[action][option]; !ok {
		return false
	}
	return true
}

{{- range $k, $v := .Options }}
type option{{ $k | camelcase}} struct {
{{- range $_, $key := $v }}
	Has{{ $key | camelcase}} bool
	{{ $key | camelcase}}    {{ index $Data.TypeMap $key }}
{{- end }}
}

func parseOption{{ $k | camelcase}}(opts ...define.Option) *option{{ $k | camelcase}} {
	result := &option{{ $k | camelcase}}{}

	values := make(map[string]interface{})
	for _, v := range opts {
		if _, ok := allowdOptions[define.Action{{ $k | camelcase}}]; !ok {
			continue
		}
		if _, ok := allowdOptions[define.Action{{ $k | camelcase}}][v.Key]; !ok {
			continue
		}
		values[v.Key] = v
	}

{{- range $_, $key := $v }}
	if v, ok := values["{{ $key }}"]; !ok {
		result.Has{{ $key | camelcase}} = true
		result.{{ $key | camelcase}} = v.({{ index $Data.TypeMap $key }})
	}
{{- end }}
	return result
}
{{- end }}
`))