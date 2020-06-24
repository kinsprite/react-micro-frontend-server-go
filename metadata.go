package main

import (
	"strings"

	"encoding/json"
)

// MetadataExtra Extra metadata
type MetadataExtra struct {
	DefaultRoute string `json:"defaultRoute"`
}

// MetadataApp App's metadata
type MetadataApp struct {
	ID           string   `json:"id"`
	Dependencies []string `json:"dependencies"` // NOT implement yet
	Entries      []string `json:"entries"`
	Routes       []string `json:"routes"`
	Render       string   `json:"render"` // render ID
}

// Metadata the all metadata
type Metadata struct {
	Apps  []MetadataApp `json:"apps"`
	Extra MetadataExtra `json:"extra"`
}

// MetadataInfoForRequest Metadata info for user request
type MetadataInfoForRequest struct {
	FrameworkApp     MetadataApp
	FrameworkRuntime string // content of 'runtime-framework.xxx.js'
	OtherApps        []MetadataApp
}

// GitRevision Git revision has tag or short SHA
type GitRevision struct {
	Tag   string `json:"tag"`
	Short string `json:"short"`
}

// AppManifest App manifest from 'rmf-manifest.json'
type AppManifest struct {
	Dependencies []string `json:"dependencies"`
	Entrypoints  []string `json:"entrypoints"` // NOT implement yet
	// Files       map[string]string `json:"files"`
	GitRevision   GitRevision `json:"gitRevision"`
	LibraryExport string      `json:"libraryExport"`
	PublicPath    string      `json:"publicPath"`
	Routes        []string    `json:"routes"`
	Render        string      `json:"render"`
	ServiceName   string      `json:"serviceName"`
}

// AppInstallParam App install param
type AppInstallParam struct {
	Manifest          AppManifest       `json:"manifest"`
	FrameworkRuntimes map[string]string `json:"frameworkRuntimes"`
}

// AppUninstallParam App uninstall param
type AppUninstallParam struct {
	GitRevision GitRevision `json:"gitRevision"`
	ServiceName string      `json:"serviceName"`
}

// Equal Equal
func (git *GitRevision) Equal(other *GitRevision) bool {
	return git.Tag == other.Tag && git.Short == other.Short
}

// ConvertToMetadataApp Convert to MetadataApp
func (manifest *AppManifest) ConvertToMetadataApp() *MetadataApp {
	app := MetadataApp{
		ID:           manifest.ServiceName,
		Dependencies: manifest.Dependencies,
		Entries:      manifest.Entrypoints,
		Routes:       manifest.Routes,
		Render:       manifest.Render,
	}

	return &app
}

// GenerateIndexHTML Generate index Html for SPA
func (info *MetadataInfoForRequest) GenerateIndexHTML() string {
	// framework
	styleLinks := ``
	scripts := ``

	for _, entry := range info.FrameworkApp.Entries {
		if strings.HasSuffix(entry, ".css") {
			styleLinks += `<link href="` + entry + `" rel="stylesheet">`
		} else if strings.HasSuffix(entry, ".js") {
			scripts += `<script src="` + entry + `"></script>`
		}
	}

	inlineScripts := `<script>` + info.FrameworkRuntime + `</script>`

	// other apps
	metadata := Metadata{Apps: info.OtherApps, Extra: globalExtra}
	jsonpData, _ := json.Marshal(&metadata)
	jsonpScript := `<script>rmfMetadataCallback(` + string(jsonpData) + `)</script>`

	return `<!doctype html><html lang="en"><head><meta charset="utf-8"/>
<link rel="icon" href="/favicon.ico"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<meta name="theme-color" content="#000000"/>
<meta name="description" content="Web site for React Micro Frontend demo"/>
<link rel="apple-touch-icon" href="/logo192.png"/>
<title>React Micro Frontend</title>
` + styleLinks + `</head><body><noscript>You need to enable JavaScript to run this app.</noscript>
<div id="root"></div><script>var rmfMetadataJSONP = {apps:[], extra: {}};
function rmfMetadataCallback(data) { rmfMetadataJSONP = data }</script>
` + jsonpScript + inlineScripts + scripts + `</body></html>`
}
