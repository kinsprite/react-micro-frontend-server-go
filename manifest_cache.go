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
	FrameworkRuntimes sync.Map // entry URL to runtime JS contents, as map[key string]string
	ServiceManifests  sync.Map // serverName to []*AppManifest, as map[key string] []*AppManifest
	ServiceMutexes    sync.Map //  serverName to *Mutex, for per app's Install and Uninstall
}

// NewAppManifestCache new an AppManifestCache
func NewAppManifestCache() *AppManifestCache {
	return &AppManifestCache{}
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

		if mLen == 0 {
			return true
		}

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

// InstallAppVersion Install an new App version after the static files have been deployed.
func (cache *AppManifestCache) InstallAppVersion(app *AppInstallParam) bool {
	mtxValue, _ := cache.ServiceMutexes.LoadOrStore(app.Manifest.ServiceName, &sync.Mutex{})
	mtx := mtxValue.(*sync.Mutex)

	mtx.Lock()
	defer mtx.Unlock()

	// Save the runtime thunk's content first
	for url, content := range app.FrameworkRuntimes {
		cache.FrameworkRuntimes.Store(url, content)
	}

	// Save the manifest
	value, ok := cache.ServiceManifests.Load(app.Manifest.ServiceName)
	appManifests := []*AppManifest{}

	if ok {
		// DON'T change the old slice
		appManifests = append(appManifests, value.([]*AppManifest)...)
	}

	appManifests = append(appManifests, &app.Manifest)
	cache.ServiceManifests.Store(app.Manifest.ServiceName, appManifests)

	return true
}

// UninstallAppVersion Uninstall an delployed App version. NOTE: Leave cache.FrameworkRuntimes unchanged.
func (cache *AppManifestCache) UninstallAppVersion(app *AppUninstallParam) bool {
	mtxValue, _ := cache.ServiceMutexes.LoadOrStore(app.ServiceName, &sync.Mutex{})
	mtx := mtxValue.(*sync.Mutex)

	mtx.Lock()
	defer mtx.Unlock()

	value, ok := cache.ServiceManifests.Load(app.ServiceName)

	if !ok {
		return false
	}

	appManifests := value.([]*AppManifest)

	// Find the app by GitRevision
	isFound := false
	newManifests := []*AppManifest{}
	last := 0

	for i, manifest := range appManifests {
		if manifest.GitRevision.Equal(&app.GitRevision) {
			isFound = true
			newManifests = append(newManifests, appManifests[last:i]...)
			last = i + 1
		}
	}

	if isFound {
		newManifests = append(newManifests, appManifests[last:]...)
		cache.ServiceManifests.Store(app.ServiceName, newManifests)
	}

	return isFound
}
