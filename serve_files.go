package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// WalkServeFilesResult serve file in Gin
type WalkServeFilesResult struct {
	Dirs  []string
	Files []string
}

// walk the first level in root DIR
func walkServeFiles(rootDir string) WalkServeFilesResult {
	result := WalkServeFilesResult{}

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

		// Only keep the first level at root
		relativePath, err := filepath.Rel(rootDir, path)

		if err != nil {
			log.Printf("[ERROR]  Cannot get the relative path for: %s\n", path)
		}

		if relativePath != name {
			return filepath.SkipDir
		}

		// Add to result

		if isDir {
			result.Dirs = append(result.Dirs, name)
		} else {
			result.Files = append(result.Files, name)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("error walking the path %q: %v\n", rootDir, err)
	}

	return result
}

func serveDirsAndFiles(router *gin.Engine, walkAppsResult *WalkAppsResult, startupInitDir string) {
	serveDirs := append([]string{}, walkAppsResult.AppDirs...)
	serveFiles := append([]string{}, globalSiteConfig.ServeStaticFiles...)

	if globalSiteConfig.ServeAllInDir {
		walkServeFilesResult := walkServeFiles(startupInitDir)

		serveDirs = append(serveDirs, walkServeFilesResult.Dirs...)
		serveFiles = append(serveFiles, walkServeFilesResult.Files...)
	}

	servedDirsMap := map[string]bool{}

	for _, appDir := range serveDirs {
		if _, ok := servedDirsMap[appDir]; !ok {
			router.Static("/"+appDir, path.Join(startupInitDir, appDir))
			servedDirsMap[appDir] = true
		}
	}

	servedFilesMap := map[string]bool{}

	for _, file := range serveFiles {
		if _, ok := servedFilesMap[file]; !ok {
			router.StaticFile("/"+file, path.Join(startupInitDir, file))
			servedFilesMap[file] = true
		}
	}
}
