package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// WalkAppsResult the result while walk files
type WalkAppsResult struct {
	AppDirs       []string
	ManifestFiles []string
}

const (
	appDirPrefix               = "rmf-"
	manifestFileName           = "rmf-manifest.json"
	frameworkServiceName       = "framework"
	frameworkRuntimeFilePrefix = "runtime-framework."
)

var listenAddress = "127.0.0.1:8080"
var publishRoot = "http://localhost:8080/"
var startupInitDir = "."
var serveStaticFiles = true

func init() {
	addr := os.Getenv("RMF_LISTEN_ADDRESS")

	if addr != "" {
		listenAddress = addr
	}

	url := os.Getenv("RMF_PUBLISH_ROOT")

	if url != "" {
		publishRoot = url
	}

	dir := os.Getenv("RMF_STARTUP_INIT_DIR")

	if dir != "" {
		startupInitDir = dir
	}

	serveStatic := os.Getenv("RMF_SERVE_STATIC_FILES")

	if serveStatic != "" {
		serveStaticFiles = serveStatic != "false"
	}
}

func walkAppFiles(rootDir string) WalkAppsResult {
	result := WalkAppsResult{}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		isDir := info.IsDir()
		name := info.Name()
		// fmt.Printf("Find path: %+v\n", path)

		// Only keep 'rmf-xxx-yyy' dir at root
		if isDir && (path != ".") && !strings.HasPrefix(path, appDirPrefix) {
			return filepath.SkipDir
		}

		// Find app's dir
		if isDir && path == name && strings.HasPrefix(name, appDirPrefix) {
			// fmt.Printf("Find app dir: %+v\n", name)
			result.AppDirs = append(result.AppDirs, name)
		}

		// Find 'rmf-manifest.json'
		if !isDir && name == manifestFileName {
			// fmt.Printf("Find manifest: %+v\n", path)
			result.ManifestFiles = append(result.ManifestFiles, path)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", rootDir, err)
	}

	return result
}

func main() {
	walkAppsResult := walkAppFiles(startupInitDir)
	// fmt.Printf("WalkAppsResult: %v\", walkAppsResult)
	cache := NewAppManifestCache()

	for _, filename := range walkAppsResult.ManifestFiles {
		cache.LoadAppManifest(path.Join(startupInitDir, filename))
	}

	cache.CacheFrameworkRuntimes(startupInitDir)

	// fmt.Printf("Cache ServiceManifests: %+v\n", cache.ServiceManifests)
	// fmt.Printf("Cache FrameworkRuntimes: %+v\n", cache.FrameworkRuntimes)

	router := gin.Default()

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	})

	router.GET("/api/metadata/info", func(c *gin.Context) {
		data := map[string]interface{}{
			"apps": []string{},
			"extra": map[string]interface{}{
				"defaultRoute": "/home",
			},
		}

		// /JSONP?callback=x
		// return ：x({\"foo\":\"bar\"})
		c.JSONP(http.StatusOK, data)
	})

	if serveStaticFiles {
		for _, appDir := range walkAppsResult.AppDirs {
			router.Static("/"+appDir, path.Join(startupInitDir, appDir))
		}
	}

	fmt.Println("Serve on: ", listenAddress)
	router.Run(listenAddress)
}
