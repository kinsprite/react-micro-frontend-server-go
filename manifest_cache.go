package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"path"
	"strings"
	"time"
)

// AppManifestCache AppManifest Cache
type AppManifestCache struct {
	FrameworkRuntimes map[string]string // URL to runtime JS contents
	ServiceManifests  map[string][]*AppManifest
}

// NewAppManifestCache new an AppManifestCache
func NewAppManifestCache() *AppManifestCache {
	return &AppManifestCache{
		FrameworkRuntimes: map[string]string{},
		ServiceManifests:  map[string][]*AppManifest{},
	}
}

// LoadAppManifest cache each Manifest file
func (cache *AppManifestCache) LoadAppManifest(filename string) {
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Printf("[ERROR]  Cannot read file %s\n", filename)
		return
	}

	var manifest AppManifest
	err = json.Unmarshal(content, &manifest)

	if err != nil {
		log.Printf("[ERROR]  Unmarshal file %s to AppManifest\n", filename)
		return
	}

	appManifests := cache.ServiceManifests[manifest.ServiceName]
	cache.ServiceManifests[manifest.ServiceName] = append(appManifests, &manifest)
	// fmt.Printf("manifest: %+v\n", manifest)
}

// CacheFrameworkRuntimes cache framework runtimes
func (cache *AppManifestCache) CacheFrameworkRuntimes(baseDir string) {
	appManifests, ok := cache.ServiceManifests[frameworkServiceName]

	if !ok {
		log.Printf("[ERROR]  Cannot find manifest for service '%s'\n", frameworkServiceName)
		return
	}

	for _, manifest := range appManifests {
		for _, entry := range manifest.Entrypoints {
			if strings.Contains(entry, frameworkRuntimeFilePrefix) {
				// fmt.Printf("Framework runtime entry: %+v\n", entry)
				contents, err := readRuntimeContent(baseDir, entry)

				if err == nil {
					cache.FrameworkRuntimes[entry] = contents
				}
			}
		}
	}
}

func readRuntimeContent(baseDir string, entry string) (string, error) {
	entryParts := strings.Split(entry, "/")
	partsLen := len(entryParts)

	start := 0

	if partsLen > 3 {
		start = partsLen - 3
	}

	parts := append([]string{baseDir}, entryParts[start:partsLen]...)
	filename := path.Join(parts...)

	content, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Printf("[ERROR]  Cannot read file %s\n", filename)
		return "", err
	}

	return string(content[:]), err
}

// GenerateMetadata Generate Metadata for user request
func (cache *AppManifestCache) GenerateMetadata(isDev bool, inlineRuntime bool) *MetadataInfoForRequest {
	info := &MetadataInfoForRequest{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for serviceName, manifests := range cache.ServiceManifests {
		mLen := len(manifests)
		selIdx := 0

		if mLen > 1 {
			selIdx = r.Intn(mLen)
		}

		app := manifests[selIdx].ConvertToMetadataApp()

		if serviceName == frameworkServiceName {
			cache.AppendFrameworkAppInfo(info, app, inlineRuntime)
		} else {
			info.OtherApps = append(info.OtherApps, *app)
		}
	}

	return info
}

// AppendFrameworkAppInfo Append Framework App Info
func (cache *AppManifestCache) AppendFrameworkAppInfo(
	info *MetadataInfoForRequest, frameApp *MetadataApp, inlineRuntime bool) {
	if !inlineRuntime {
		info.FrameworkApp = *frameApp
		return
	}

	for i, entry := range frameApp.Entries {
		if strings.HasPrefix(entry, frameworkRuntimeFilePrefix) {
			if content, ok := cache.FrameworkRuntimes[entry]; ok {
				info.FrameworkRuntime = content
				frameApp.Entries = append(frameApp.Entries[:i], frameApp.Entries[i+1:]...)
				info.FrameworkApp = *frameApp
				return
			}
		}
	}
}
