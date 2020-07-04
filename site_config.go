package main

import (
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// SiteConfig site config
type SiteConfig struct {
	Extra      MetadataExtra `yaml:"extra"`
	HTMLBegin  string        `yaml:"htmlBegin"`
	HTMLMiddle string        `yaml:"htmlMiddle"`
	HTMLEnd    string        `yaml:"htmlEnd"`

	ListenAddress     string   `yaml:"listenAddress"`
	StartupInitDir    string   `yaml:"startupInitDir"`
	EnableServeStatic bool     `yaml:"enableServeStatic"`
	ServeStaticFiles  []string `yaml:"serveStaticFiles"`
	ServeAllInDir     bool     `yaml:"serveAllInDir"`

	SessionSign     string   `yaml:"sessionSign"`
	ExtraKeysHidden []string `yaml:"extraKeysHidden"`

	ExtraKeysHiddenMap map[string]bool
}

var globalSiteConfig = SiteConfig{
	Extra: MetadataExtra{
		"defaultRoute": "/home",
	},
	HTMLBegin: `<!doctype html><html lang="en"><head><meta charset="utf-8"/>
<link rel="icon" href="/favicon.ico"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<meta name="theme-color" content="#000000"/>
<meta name="description" content="Web site for React Micro Frontends demo"/>
<link rel="apple-touch-icon" href="/logo192.png"/>
<title>React Micro Frontends</title>`,
	HTMLMiddle: `</head><body><noscript>You need to enable JavaScript to run this app.</noscript>
<div id="root"></div><script>var rmfMetadataJSONP = {apps:[], extra: {}};
function rmfMetadataCallback(data) { rmfMetadataJSONP = data }</script>`,
	HTMLEnd: `</body></html>`,

	ListenAddress:     "127.0.0.1:8080",
	StartupInitDir:    ".",
	EnableServeStatic: true,

	ServeStaticFiles: []string{
		"favicon.ico",
	},
	ServeAllInDir: false,

	SessionSign: "",
	ExtraKeysHidden: []string{
		"userGroup",
		"actPercent",
	},
}

// LoadSiteConfig Load site's config form YAML file
func LoadSiteConfig(filename string) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0600)
	defer file.Close()

	content, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Printf("[ERROR]  Cannot read file %s\n", filename)
		return
	}

	siteConfig := SiteConfig{}
	err = yaml.Unmarshal(content, &siteConfig)

	if err != nil {
		log.Printf("[ERROR]  Cannot convert file %s to SiteConfig\n", filename)
		return
	}
}

// MergeFrom Merge the config to 'conf' from 'other'
func (conf *SiteConfig) MergeFrom(other *SiteConfig) {
	for key, value := range other.Extra {
		conf.Extra[key] = value
	}

	if other.HTMLBegin != "" {
		conf.HTMLBegin = other.HTMLBegin
	}

	if other.HTMLMiddle != "" {
		conf.HTMLMiddle = other.HTMLMiddle
	}

	if other.HTMLEnd != "" {
		conf.HTMLEnd = other.HTMLEnd
	}

	if other.ListenAddress != "" {
		conf.ListenAddress = other.ListenAddress
	}

	conf.StartupInitDir = other.StartupInitDir
	conf.EnableServeStatic = other.EnableServeStatic

	if len(other.ServeStaticFiles) > 0 {
		conf.ServeStaticFiles = other.ServeStaticFiles
	}

	conf.ServeAllInDir = other.ServeAllInDir
	conf.SessionSign = other.SessionSign

	if len(other.ExtraKeysHidden) > 0 {
		conf.ExtraKeysHidden = other.ExtraKeysHidden
	}

	conf.UpdateExtraKeysHiddenMap()
}

// UpdateExtraKeysHiddenMap update the map of ExtraKeysHidden
func (conf *SiteConfig) UpdateExtraKeysHiddenMap() {
	conf.ExtraKeysHiddenMap = map[string]bool{}

	for _, key := range conf.ExtraKeysHidden {
		conf.ExtraKeysHiddenMap[key] = true
	}
}

// SafeExtra hidden some keys from user
func (conf *SiteConfig) SafeExtra(extra MetadataExtra) MetadataExtra {
	res := MetadataExtra{}

	for key, value := range extra {
		if _, ok := conf.ExtraKeysHiddenMap[key]; !ok {
			res[key] = value
		}
	}

	return res
}
