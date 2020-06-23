package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var listenAddress = "127.0.0.1:8080"
var publishRoot = "http://localhost:8080/"
var startupInitDir = "."

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
}
func prepareTestDirTree(tree string) (string, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", fmt.Errorf("error creating temp directory: %v\n", err)
	}

	err = os.MkdirAll(filepath.Join(tmpDir, tree), 0755)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	return tmpDir, nil
}

func walkMain() []string {
	tmpDir := '.'
	subDirToSkip := ".git"
	apps := []string{}

	fmt.Println("On Unix:")
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		name := info.Name()

		if info.IsDir() && strings.HasPrefix(name, "rmf-") {
			if path == name {
				fmt.Printf("Find dir: %+v \n", name)
				apps = append(apps, name)
			} else {
				return filepath.SkipDir
			}
		}

		if info.IsDir() && info.Name() == subDirToSkip {
			fmt.Printf("skipping a dir without errors: %+v \n", info.Name())
			return filepath.SkipDir
		}
		fmt.Printf("visited file or dir: %q\n", path)
		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", tmpDir, err)
		return apps
	}

	return apps
}

func main() {
	apps := walkMain()
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
		// 将输出：x({\"foo\":\"bar\"})
		c.JSONP(http.StatusOK, data)
	})

	for _, app := range apps {
		router.Static("/"+app, startupInitDir+"/"+app)
	}

	fmt.Println("Serve on : ", listenAddress)
	router.Run(listenAddress)
}
