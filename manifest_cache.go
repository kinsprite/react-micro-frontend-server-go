package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path"
	"strings"
)

const (
	frameworkServiceName       = "framework"
	frameworkRuntimeFilePrefix = "runtime-framework."
)

// AppManifestCache AppManifest Cache
type AppManifestCache struct {
	FrameworkRuntimes map[string]string // URL to runtime JS contents
	ServiceManifests  map[string][]*AppManifest
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
				contents, err := readRuntimeContent(baseDir, entry)

				if err != nil {
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
