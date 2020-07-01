package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
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
	polyfillServiceName        = "polyfill"
	frameworkServiceName       = "framework"
	frameworkRuntimeFilePrefix = "runtime-framework."
)

var listenAddress = "127.0.0.1:8080"
var startupInitDir = "."
var serveStaticFiles = true
var siteConfigFile = ""

func init() {
	addr := os.Getenv("RMF_LISTEN_ADDRESS")

	if addr != "" {
		listenAddress = addr
	}

	dir := os.Getenv("RMF_STARTUP_INIT_DIR")

	if dir != "" {
		startupInitDir = dir
	}

	serveStatic := os.Getenv("RMF_SERVE_STATIC_FILES")

	if serveStatic != "" {
		serveStaticFiles = serveStatic != "false"
	}

	configFile := os.Getenv("RMF_SITE_CONFIG_FILE")

	if configFile != "" {
		siteConfigFile = configFile
	}
}

func parseFlags() {
	listeningAddressFlag := flag.String("RMF_LISTEN_ADDRESS", "", "Server listening address")
	startupInitDirFlag := flag.String("RMF_STARTUP_INIT_DIR", "", "Search micro frontend manifests in the dir")
	serveStaticFileFlag := flag.String("RMF_SERVE_STATIC_FILES", "", "Blog updating script file")
	configFileFlag := flag.String("RMF_SITE_CONFIG_FILE", "", "Site's config form YAML file")

	flag.Parse()

	if *listeningAddressFlag != "" {
		listenAddress = *listeningAddressFlag
	}

	if *startupInitDirFlag != "" {
		startupInitDir = *startupInitDirFlag
	}

	if *serveStaticFileFlag != "" {
		serveStaticFiles = *serveStaticFileFlag != "false"
	}

	if *configFileFlag != "" {
		siteConfigFile = *configFileFlag
	}
}

func walkAppFiles(rootDir string) WalkAppsResult {
	result := WalkAppsResult{}

	rootLen := len(rootDir)

	if rootDir[rootLen-1] != '/' && rootDir[rootLen-1] != '\\' {
		rootLen++
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		// just continue rootDir
		if path == rootDir {
			return nil
		}

		isDir := info.IsDir()
		name := info.Name()
		// fmt.Printf("Find path: %+v\n", path)

		// Only keep 'rmf-xxx-yyy' dir at root
		relativePath, err := filepath.Rel(rootDir, path)

		if err != nil {
			log.Printf("[ERROR]  Cannot get the relative path for: %s\n", path)
		}

		if isDir && !strings.HasPrefix(relativePath, appDirPrefix) {
			return filepath.SkipDir
		}

		// Find app's dir
		if isDir && relativePath == name && strings.HasPrefix(name, appDirPrefix) {
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

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func main() {
	parseFlags()

	if siteConfigFile != "" {
		LoadSiteConfig(siteConfigFile)
	}

	walkAppsResult := walkAppFiles(startupInitDir)
	// fmt.Printf("WalkAppsResult: %v\", walkAppsResult)
	cache := NewAppManifestCache()

	for _, filename := range walkAppsResult.ManifestFiles {
		cache.LoadAppManifest(filename)
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
		info := cache.GenerateMetadata(true, true)

		c.JSONP(http.StatusOK, &Metadata{
			Apps:  info.OtherApps,
			Extra: globalSiteConfig.Extra,
		})
	})

	router.POST("/api/metadata/install-app-version", func(c *gin.Context) {
		var param AppInstallParam

		if err := c.BindJSON(&param); err != nil {
			return
		}

		ok := cache.InstallAppVersion(&param)

		c.JSON(http.StatusOK, gin.H{
			"install": ok,
		})
	})

	router.POST("/api/metadata/uninstall-app-version", func(c *gin.Context) {
		var param AppUninstallParam

		if err := c.BindJSON(&param); err != nil {
			return
		}

		ok := cache.UninstallAppVersion(&param)
		c.JSON(http.StatusOK, gin.H{
			"uninstall": ok,
		})
	})

	if serveStaticFiles {
		// Fix invalid MIME type in windows
		mime.AddExtensionType(".js", "text/javascript")

		for _, appDir := range walkAppsResult.AppDirs {
			router.Static("/"+appDir, path.Join(startupInitDir, appDir))
		}

		for _, file := range globalSiteConfig.ServeStaticFiles {
			router.StaticFile("/"+file, path.Join(startupInitDir, file))
		}
	}

	// SPA
	router.NoRoute(func(c *gin.Context) {
		info := cache.GenerateMetadata(true, true)
		// fmt.Printf("INFO %+v\n", info)
		userAgent := c.Request.UserAgent()
		HTML, pushLink := info.GenerateIndexHTML(userAgent)

		if pushLink != "" {
			c.Writer.Header().Add("Link", pushLink)
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(HTML))
	})

	fmt.Println("Serve on: ", listenAddress)
	router.Run(listenAddress)
}
