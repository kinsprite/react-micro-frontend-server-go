package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"path"
	"strings"
	"sync"
	"time"
)

// AppManifestCache AppManifest Cache
type AppManifestCache struct {
	// FrameworkRuntimes map[string]string // entry URL to runtime JS contents
	// ServiceManifests  map[string][]*AppManifest // serverName to []*AppManifest
	// rwMtx             *sync.RWMutex
	FrameworkRuntimes sync.Map
	ServiceManifests  sync.Map
}

// NewAppManifestCache new an AppManifestCache
func NewAppManifestCache() *AppManifestCache {
	return &AppManifestCache{
		// FrameworkRuntimes: map[string]string{},
		// ServiceManifests:  map[string][]*AppManifest{},
		// rwMtx: new(sync.RWMutex),
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

	value, ok := cache.ServiceManifests.Load(manifest.ServiceName)

	appManifests := []*AppManifest{}

	if ok {
		appManifests = value.([]*AppManifest)
	}

	cache.ServiceManifests.Store(manifest.ServiceName, append(appManifests, &manifest))
	// fmt.Printf("manifest: %+v\n", manifest)
}

// CacheFrameworkRuntimes cache framework runtimes
func (cache *AppManifestCache) CacheFrameworkRuntimes(baseDir string) {
	value, ok := cache.ServiceManifests.Load(frameworkServiceName)

	if !ok {
		log.Printf("[ERROR]  Cannot find manifest for service '%s'\n", frameworkServiceName)
		return
	}

	appManifests := value.([]*AppManifest)

	for _, manifest := range appManifests {
		for _, entry := range manifest.Entrypoints {
			if strings.Contains(entry, frameworkRuntimeFilePrefix) {
				// fmt.Printf("Framework runtime entry: %+v\n", entry)
				contents, err := readRuntimeContent(baseDir, entry)

				if err == nil {
					cache.FrameworkRuntimes.Store(entry, contents)
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

	cache.ServiceManifests.Range(func(key, value interface{}) bool {
		serviceName := key.(string)
		manifests := value.([]*AppManifest)

		mLen := len(manifests)
		selIdx := 0

		if mLen > 1 {
			selIdx = r.Intn(mLen)
		}

		app := manifests[selIdx].ConvertToMetadataApp()

		if serviceName == frameworkServiceName {
			// fmt.Printf("Frame manifests BEFORE: %+v\n", *manifests[selIdx])
			cache.AppendFrameworkAppInfo(info, app, inlineRuntime)
			// fmt.Printf("Frame manifests AFTER: %+v\n", *manifests[selIdx])
		} else {
			info.OtherApps = append(info.OtherApps, *app)
		}

		return true
	})

	return info
}

// AppendFrameworkAppInfo Append Framework App Info
func (cache *AppManifestCache) AppendFrameworkAppInfo(
	info *MetadataInfoForRequest, frameApp *MetadataApp, inlineRuntime bool) {
	// fmt.Printf("FrameApp: %+v\n", frameApp)
	if !inlineRuntime {
		info.FrameworkApp = *frameApp
		return
	}

	for i, entry := range frameApp.Entries {
		if strings.Contains(entry, frameworkRuntimeFilePrefix) {
			content, ok := cache.FrameworkRuntimes.Load(entry)

			if ok {
				info.FrameworkRuntime = content.(string)
				frameAppEntries := append([]string{}, frameApp.Entries[:i]...)
				frameAppEntries = append(frameAppEntries, frameApp.Entries[i+1:]...)
				info.FrameworkApp = *frameApp
				info.FrameworkApp.Entries = frameAppEntries
				return
			}
		}
	}

	info.FrameworkApp = *frameApp
}
