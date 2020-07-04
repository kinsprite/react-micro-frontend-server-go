package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	polyfillServiceName        = "polyfill"
	frameworkServiceName       = "framework"
	frameworkRuntimeFilePrefix = "runtime-framework."
	userGroupKey               = "userGroup"
	testerUserGroup            = "tester"
	defaultUserGroup           = ""
	userGroupsSplitSep         = ","
)

var manifestFileNameRegexp = regexp.MustCompile(`^rmf-manifest([.\-_].+)?\.json$`)

var siteConfigFile = ""

func init() {
	configFile := os.Getenv("RMF_SITE_CONFIG_FILE")

	if configFile != "" {
		siteConfigFile = configFile
	}
}

func parseFlags() {
	configFileFlag := flag.String("RMF_SITE_CONFIG_FILE", "", "Site's config from YAML file")
	flag.Parse()

	if *configFileFlag != "" {
		siteConfigFile = *configFileFlag
	}
}

func matchManifestFileName(fileName string) bool {
	return manifestFileNameRegexp.MatchString(fileName)
}

func walkAppFiles(rootDir string) WalkAppsResult {
	result := WalkAppsResult{}

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

		// Find 'rmf-manifest.json' or 'rmf-manifest.xxx.json'
		if !isDir && matchManifestFileName(name) {
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

func noCacheMiddleware(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Expires", "Thu, 01 Jan 1970 00:00:01 GMT")
	ctx.Next()
}

func main() {
	parseFlags()

	if siteConfigFile != "" {
		LoadSiteConfig(siteConfigFile)
	}

	globalSiteConfig.UpdateExtraKeysHiddenMap()

	walkAppsResult := walkAppFiles(globalSiteConfig.StartupInitDir)
	// fmt.Printf("WalkAppsResult: %v\", walkAppsResult)
	cache := NewAppManifestCache()

	for _, filename := range walkAppsResult.ManifestFiles {
		cache.LoadAppManifest(filename)
	}

	cache.CacheFrameworkRuntimes(globalSiteConfig.StartupInitDir)

	// fmt.Printf("Cache ServiceManifests: %+v\n", cache.ServiceManifests)
	// fmt.Printf("Cache FrameworkRuntimes: %+v\n", cache.FrameworkRuntimes)

	if globalSiteConfig.GinReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()
	sessionMiddleware := createSessionMiddleware()

	engine.GET("/healthz", noCacheMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	})

	metadataRouterGroup := engine.Group("/api/metadata").Use(sessionMiddleware, noCacheMiddleware)

	metadataRouterGroup.GET("/info", func(c *gin.Context) {
		userGroups := getUserGroups(c)
		info := cache.GenerateMetadata(GenMetadataParam{userGroups, true})

		c.JSONP(http.StatusOK, &Metadata{
			Apps:  info.OtherApps,
			Extra: globalSiteConfig.SafeExtra(globalSiteConfig.Extra),
		})
	})

	metadataRouterGroup.POST("/install-app-version", func(c *gin.Context) {
		var param AppInstallParam

		if err := c.BindJSON(&param); err != nil {
			return
		}

		ok := cache.InstallAppVersion(&param)

		c.JSON(http.StatusOK, gin.H{
			"install": ok,
		})
	})

	metadataRouterGroup.POST("/uninstall-app-version", func(c *gin.Context) {
		var param AppUninstallParam

		if err := c.BindJSON(&param); err != nil {
			return
		}

		ok := cache.UninstallAppVersion(&param)
		c.JSON(http.StatusOK, gin.H{
			"uninstall": ok,
		})
	})

	userRouterGroup := engine.Group("/api/user").Use(sessionMiddleware, noCacheMiddleware)

	userRouterGroup.GET("/is-tester", func(c *gin.Context) {
		isTester := false

		for _, group := range getUserGroups(c) {
			if group == testerUserGroup {
				isTester = true
			}
		}

		c.JSON(http.StatusOK, isTester)
	})

	userRouterGroup.POST("/login-as-tester", func(c *gin.Context) {
		var isTester bool

		if err := c.BindJSON(&isTester); err != nil {
			return
		}

		userGroup := testerUserGroup

		if !isTester {
			userGroup = defaultUserGroup
		}

		err := setUserGroups(c, []string{userGroup})

		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Status(http.StatusOK)
	})

	if globalSiteConfig.EnableServeStatic {
		// Fix invalid MIME type in windows
		mime.AddExtensionType(".js", "text/javascript")

		serveDirsAndFiles(engine, &walkAppsResult, globalSiteConfig.StartupInitDir)
	}

	// SPA
	engine.NoRoute(sessionMiddleware, noCacheMiddleware, func(c *gin.Context) {
		userGroups := getUserGroups(c)
		info := cache.GenerateMetadata(GenMetadataParam{userGroups, true})
		// fmt.Printf("INFO %+v\n", info)
		userAgent := c.Request.UserAgent()
		HTML, pushLink := info.GenerateIndexHTML(userAgent)

		if pushLink != "" {
			c.Writer.Header().Add("Link", pushLink)
		}

		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(HTML))
	})

	fmt.Println("Serve on: ", globalSiteConfig.ListenAddress)
	engine.Run(globalSiteConfig.ListenAddress)
}
