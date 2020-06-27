package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/mssola/user_agent"
)

// MetadataExtra Extra metadata
type MetadataExtra struct {
	DefaultRoute string `json:"defaultRoute"`
}

// MetadataRender rmfRenders in package.json
type MetadataRender struct {
	RenderID     string `json:"renderId"`
	RoutePath    string `json:"routePath"`
	ComponentKey string `json:"componentKey"`
}

// MetadataApp App's metadata
type MetadataApp struct {
	ID           string           `json:"id"`
	Dependencies []string         `json:"dependencies"` // NOT implement yet
	Entries      []string         `json:"entries"`
	Renders      []MetadataRender `json:"renders"`
}

// Metadata the all metadata
type Metadata struct {
	Apps  []MetadataApp `json:"apps"`
	Extra MetadataExtra `json:"extra"`
}

// MetadataInfoForRequest Metadata info for user request
type MetadataInfoForRequest struct {
	PolyfillApp      MetadataApp
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
	GitRevision   GitRevision      `json:"gitRevision"`
	LibraryExport string           `json:"libraryExport"`
	PublicPath    string           `json:"publicPath"`
	Renders       []MetadataRender `json:"renders"`
	ServiceName   string           `json:"serviceName"`
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
		Renders:      manifest.Renders,
	}

	return &app
}

// GenerateIndexHTML Generate index Html for SPA
func (info *MetadataInfoForRequest) GenerateIndexHTML(userAgent string) string {
	// polyfill and framework
	styleLinks := ``
	scripts := GeneratePolyfillScriptTag(&info.PolyfillApp, userAgent)

	// framework
	for _, entry := range info.FrameworkApp.Entries {
		if strings.HasSuffix(strings.ToLower(entry), ".css") {
			styleLinks += `<link href="` + entry + `" rel="stylesheet">`
		} else if strings.HasSuffix(strings.ToLower(entry), ".js") {
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
<meta name="description" content="Web site for React Micro Frontends demo"/>
<link rel="apple-touch-icon" href="/logo192.png"/>
<title>React Micro Frontends</title>
` + styleLinks + `</head><body><noscript>You need to enable JavaScript to run this app.</noscript>
<div id="root"></div><script>var rmfMetadataJSONP = {apps:[], extra: {}};
function rmfMetadataCallback(data) { rmfMetadataJSONP = data }</script>
` + jsonpScript + inlineScripts + scripts + `</body></html>`
}

// GeneratePolyfillScriptTag Generate polyfill script tag on different Browser
func GeneratePolyfillScriptTag(polyfillApp *MetadataApp, userAgent string) string {
	ua := user_agent.New(userAgent)

	if ua.Bot() {
		// don't give polyfill to a bot
		return ""
	}

	mapEntries := mapPolyfillEntries(polyfillApp.Entries)
	key := ""

	browserName, browserVersion := ua.Browser()

	if browserName == "Internet Explorer" {
		ieVersion, err := strconv.ParseFloat(browserVersion, 32)

		if err != nil {
			ieVersion = 0.0
		}

		// IE11
		if ieVersion > 10.5 {
			if _, ok := mapEntries["polyfill-ie11"]; ok {
				key = "polyfill-ie11"
			}
		}

		// IE9
		if key == "" {
			if _, ok := mapEntries["polyfill-ie9"]; ok {
				key = "polyfill-ie9"
			}
		}
	}

	if key == "" {
		key = "polyfill"
	}

	if url, ok := mapEntries[key]; ok {
		return `<script src="` + url + `"></script>`
	}

	return ""
}

// return { "polyfill": "full-URL", "polyfill-ie9": "full-URL-ie9", "polyfill-ie11": "full-URL-ie11", }
func mapPolyfillEntries(entries []string) map[string]string {
	res := map[string]string{}

	for _, url := range entries {
		parts := strings.Split(url, "/")
		lastPart := parts[len(parts)-1]
		key := strings.SplitN(lastPart, ".", 2)[0]
		res[key] = url
	}

	return res
}
