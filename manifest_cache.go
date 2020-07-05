package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// GenMetadataParam the param for GenerateMetadataParam()
type GenMetadataParam struct {
	UserGroups      []string
	IsInlineRuntime bool
}

// AppFilterItem the app item found
type AppFilterItem struct {
	App               *AppManifest
	ActivationPercent int
}

// AppVersionMap map the version by `{GitRevision.GetVersionKey()}`
type AppVersionMap map[string]*AppManifest

// AppManifestCache AppManifest Cache
type AppManifestCache struct {
	FrameworkRuntimes sync.Map // entry URL to runtime JS contents, as map[key string]string
	ServiceManifests  sync.Map // serviceName to AppVersionMap, as map[key string]AppVersionMap
	ServiceMutexes    sync.Map // serviceName to *Mutex, for per app's Install, Uninstall and Update
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

	var appManifests AppVersionMap

	if value, ok := cache.ServiceManifests.Load(manifest.ServiceName); ok {
		appManifests = value.(AppVersionMap)
	} else {
		appManifests = AppVersionMap{}
		cache.ServiceManifests.Store(manifest.ServiceName, appManifests)
	}

	appManifests[manifest.GitRevision.GetVersionKey()] = &manifest
	// fmt.Printf("manifest: %+v\n", manifest)
}

// CacheFrameworkRuntimes cache framework runtimes
func (cache *AppManifestCache) CacheFrameworkRuntimes(baseDir string) {
	value, ok := cache.ServiceManifests.Load(frameworkServiceName)

	if !ok {
		log.Printf("[ERROR]  Cannot find manifest for service '%s'\n", frameworkServiceName)
		return
	}

	appManifests := value.(AppVersionMap)

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
	var content []byte
	var err error

	readRuntime := func(validPathParts int) bool {
		start := 0

		if partsLen > validPathParts {
			start = partsLen - validPathParts
		}

		parts := append([]string{baseDir}, entryParts[start:partsLen]...)
		filename := path.Join(parts...)

		if exist, _ := pathExists(filename); exist {
			content, err = ioutil.ReadFile(filename)

			if err == nil {
				return true
			}
		}

		return false
	}

	ok := readRuntime(3) || readRuntime(2) || readRuntime(1)

	if !ok {
		log.Printf("[ERROR]  Cannot read runtime content for %s\n", entry)
		return "", fmt.Errorf("Cannot read runtime content")
	}

	return string(content[:]), nil
}

func stringSliceContainsAny(a, b []string) bool {
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			if a[i] == b[j] {
				return true
			}
		}
	}

	return false
}

func calcActivationPercent(manifest *AppManifest) int {
	if value, ok := manifest.Extra[activationPercentKey]; ok {
		activationPercent, err := strconv.Atoi(value)

		if err != nil {
			activationPercent = 0
		} else if activationPercent < 0 {
			activationPercent = 0
		} else if activationPercent > 100 {
			activationPercent = 100
		}

		return activationPercent
	}

	// default: when missing 'activationPercent' in extra map
	return 100
}

func filterUserManifests(manifests AppVersionMap, userGroups []string) (
	matches []AppFilterItem, defaults []AppFilterItem) {
	matches = []AppFilterItem{}
	defaults = []AppFilterItem{}
	defaultGroups := []string{defaultUserGroup}

	for _, manifest := range manifests {
		groupsInExtra := defaultGroups

		if value, ok := manifest.Extra[userGroupKey]; ok {
			groupsInExtra = strings.Split(value, userGroupsSplitSep)
		}

		activationPercent := calcActivationPercent(manifest)

		if activationPercent < 1 {
			continue
		}

		item := AppFilterItem{App: manifest, ActivationPercent: activationPercent}

		if stringSliceContainsAny(groupsInExtra, userGroups) {
			matches = append(matches, item)
		} else if stringSliceContainsAny(groupsInExtra, defaultGroups) {
			defaults = append(defaults, item)
		}
	}

	return matches, defaults
}

func selectAppByActivationPercent(r *rand.Rand, manifests []AppFilterItem) int {
	selIdx := 0
	mLen := len(manifests)

	if mLen > 1 {
		sum := 0
		steps := make([]int, mLen)

		for i := 0; i < mLen; i++ {
			sum += manifests[i].ActivationPercent
			steps[i] = sum
		}

		selSum := r.Intn(sum)

		for i := 0; i < mLen; i++ {
			if selSum < steps[i] {
				return i
			}
		}
	}

	return selIdx
}

// GenerateMetadata Generate Metadata for user request
func (cache *AppManifestCache) GenerateMetadata(param GenMetadataParam) *MetadataInfoForRequest {
	info := &MetadataInfoForRequest{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	cache.ServiceManifests.Range(func(key, value interface{}) bool {
		serviceName := key.(string)
		matches, defaults := filterUserManifests(value.(AppVersionMap), param.UserGroups)
		manifests := defaults

		if len(matches) > 0 {
			manifests = matches
		}

		// guard for defaults is empty
		if len(manifests) == 0 {
			return true
		}

		selIdx := selectAppByActivationPercent(r, manifests)
		app := manifests[selIdx].App.ConvertToMetadataApp()

		if serviceName == polyfillServiceName {
			info.PolyfillApp = *app
		} else if serviceName == frameworkServiceName {
			cache.AppendFrameworkAppInfo(info, app, param.IsInlineRuntime)
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
	// lock AppVersionMap when changing it's content
	mtxValue, _ := cache.ServiceMutexes.LoadOrStore(app.Manifest.ServiceName, &sync.Mutex{})
	mtx := mtxValue.(*sync.Mutex)

	mtx.Lock()
	defer mtx.Unlock()

	// Save the runtime thunk's content first
	for url, content := range app.FrameworkRuntimes {
		cache.FrameworkRuntimes.Store(url, content)
	}

	// Save the manifest
	var appManifests AppVersionMap
	value, ok := cache.ServiceManifests.Load(app.Manifest.ServiceName)

	if ok {
		appManifests = value.(AppVersionMap)
	} else {
		appManifests = AppVersionMap{}
	}

	version := app.Manifest.GitRevision.GetVersionKey()
	appManifests[version] = &app.Manifest

	if !ok {
		cache.ServiceManifests.Store(app.Manifest.ServiceName, appManifests)
	}

	return true
}

// UninstallAppVersion Uninstall an deployed App version. NOTE: Leave cache.FrameworkRuntimes unchanged.
func (cache *AppManifestCache) UninstallAppVersion(app *AppUninstallParam) bool {
	value, ok := cache.ServiceManifests.Load(app.ServiceName)

	if !ok {
		return false
	}

	appManifests := value.(AppVersionMap)

	// lock AppVersionMap when changing it's content
	mtxValue, _ := cache.ServiceMutexes.LoadOrStore(app.ServiceName, &sync.Mutex{})
	mtx := mtxValue.(*sync.Mutex)

	mtx.Lock()
	defer mtx.Unlock()

	// Find the version and delete it
	version := app.GitRevision.GetVersionKey()
	_, isFound := appManifests[version]

	if isFound {
		delete(appManifests, version)
	}

	return isFound
}

// UpdateAppExtra Update multi deployed Apps' Extra
func (cache *AppManifestCache) UpdateAppExtra(params []AppUpdateExtraParam) bool {
	// group params by service name (as App ID)
	serviceMap := map[string][]*AppUpdateExtraParam{}

	for _, p := range params {
		slice, ok := serviceMap[p.ServiceName]

		if !ok {
			slice = []*AppUpdateExtraParam{}
		}

		slice = append(slice, &p)
		serviceMap[p.ServiceName] = slice
	}

	// update each App's Extra
	hasOK := false

	for serviceName, params := range serviceMap {
		if cache.UpdateOneAppExtra(serviceName, params) {
			hasOK = true
		}
	}

	return hasOK
}

// UpdateOneAppExtra Update one deployed App's Extra
func (cache *AppManifestCache) UpdateOneAppExtra(serviceName string, params []*AppUpdateExtraParam) bool {
	value, ok := cache.ServiceManifests.Load(serviceName)

	if !ok {
		return false
	}

	appVersionMap := value.(AppVersionMap)

	// lock AppVersionMap when changing it's content
	mtxValue, _ := cache.ServiceMutexes.LoadOrStore(serviceName, &sync.Mutex{})
	mtx := mtxValue.(*sync.Mutex)

	mtx.Lock()
	defer mtx.Unlock()

	// update each version in params
	hasOK := false

	for _, param := range params {
		version := param.GitRevision.GetVersionKey()

		if app, ok := appVersionMap[version]; ok {
			// Merge each K-V, not replace all
			for key, value := range param.Extra {
				app.Extra[key] = value
			}

			hasOK = true
		}
	}

	return hasOK
}
