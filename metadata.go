package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/mssola/user_agent"
)

// MetadataExtra Extra metadata
type MetadataExtra map[string]string

// MetadataRender rmfRenders in package.json
type MetadataRender map[string]string

// MetadataApp App's metadata
type MetadataApp struct {
	ID           string           `json:"id"`
	Dependencies []string         `json:"dependencies"` // NOT implement yet
	Entries      []string         `json:"entries"`
	Renders      []MetadataRender `json:"renders"`
	Extra        MetadataExtra    `json:"extra"`
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
	Extra         MetadataExtra    `json:"extra"`
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
		Extra:        globalSiteConfig.SafeExtra(manifest.Extra),
	}

	return &app
}

// GenerateIndexHTML Generate index Html for SPA. return (HTML, ServerPushLink)
func (info *MetadataInfoForRequest) GenerateIndexHTML(userAgent string) (string, string) {
	resultHTML := strings.Builder{}
	resultHTML.Grow(6 * 1024)

	serverPushStyles := []string{}
	serverPushScripts := []string{}

	// polyfill and framework
	styleLinks := strings.Builder{}
	scripts := strings.Builder{}
	styleLinks.Grow(256)
	scripts.Grow(1024)

	if polyfillURL := GeneratePolyfillScriptURL(&info.PolyfillApp, userAgent); polyfillURL != "" {
		scripts.WriteString(`<script src="`)
		scripts.WriteString(polyfillURL)
		scripts.WriteString(`"></script>`)
		serverPushScripts = append(serverPushScripts, polyfillURL)
	}

	// framework
	for _, entry := range info.FrameworkApp.Entries {
		if strings.HasSuffix(strings.ToLower(entry), ".css") {
			styleLinks.WriteString(`<link href="`)
			styleLinks.WriteString(entry)
			styleLinks.WriteString(`" rel="stylesheet">`)
			serverPushStyles = append(serverPushStyles, entry)
		} else if strings.HasSuffix(strings.ToLower(entry), ".js") {
			scripts.WriteString(`<script src="`)
			scripts.WriteString(entry)
			scripts.WriteString(`"></script>`)
			serverPushScripts = append(serverPushScripts, entry)
		}
	}

	resultHTML.WriteString(globalSiteConfig.HTMLBegin)

	// Links in header
	resultHTML.WriteString(styleLinks.String())
	resultHTML.WriteString(globalSiteConfig.HTMLMiddle)

	// JSONP: other Apps and Extra
	metadata := Metadata{Apps: info.OtherApps, Extra: globalSiteConfig.Extra}
	jsonpData, _ := json.Marshal(&metadata)
	resultHTML.WriteString(`<script>rmfMetadataCallback(`)
	resultHTML.Write(jsonpData)
	resultHTML.WriteString(`)</script>`)

	// Inline framework runtime
	resultHTML.WriteString(`<script>`)
	resultHTML.WriteString(info.FrameworkRuntime)
	resultHTML.WriteString(`</script>`)

	// Polyfill and Framework scripts
	resultHTML.WriteString(scripts.String())

	// HTML End
	resultHTML.WriteString(globalSiteConfig.HTMLEnd)

	// HTML & Server Push
	return resultHTML.String(), GenerateSererPushLink(serverPushStyles, serverPushScripts)
}

// GeneratePolyfillScriptURL Generate polyfill script url on different Browser
func GeneratePolyfillScriptURL(polyfillApp *MetadataApp, userAgent string) string {
	ua := user_agent.New(userAgent)

	if ua.Bot() {
		// don't give polyfill to a bot
		return ""
	}

	mapEntries := mapPolyfillEntries(polyfillApp.Entries)
	key := ""

	setKeyIfValid := func(newKey string) {
		if _, ok := mapEntries[newKey]; ok {
			key = newKey
		}
	}

	browserName, browserVersion := ua.Browser()

	if browserName == "Internet Explorer" {
		ieVersion, err := strconv.ParseFloat(browserVersion, 32)

		if err != nil {
			ieVersion = 0.0
		}

		// IE11
		if ieVersion > 10.5 {
			setKeyIfValid("polyfill-ie11")
		}

		// IE9 as fallback
		if key == "" {
			setKeyIfValid("polyfill-ie9")
		}
	}

	// Modern browser
	if key == "" {
		setKeyIfValid("polyfill")
	}

	if key != "" {
		return mapEntries[key]
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

// GenerateSererPushLink Generate server push link in HTTP headers
func GenerateSererPushLink(serverPushStyles []string, serverPushScripts []string) string {
	// Server Push
	headerLink := strings.Builder{}
	headerLink.Grow(2048)

	addNewURL := func(url string, asType string) {
		if headerLink.Len() > 0 {
			headerLink.WriteString(", ")
		}

		// Format: </style.css>; as=style; rel=preload, </favicon.ico>; as=image; rel=preload
		headerLink.WriteString(`<`)
		headerLink.WriteString(url)
		headerLink.WriteString(`>; as=`)
		headerLink.WriteString(asType)
		headerLink.WriteString(`; rel=preload`)
	}

	for _, url := range serverPushStyles {
		addNewURL(url, "style")
	}

	for _, url := range serverPushScripts {
		addNewURL(url, "script")
	}

	return headerLink.String()
}
