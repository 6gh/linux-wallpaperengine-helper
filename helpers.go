package main

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func resolvePath(pathString string) (string, error) {
	// this is because in the config file users can use ~, so ensure that we resolve it
	if strings.HasPrefix(pathString, "~/") {
		usr, err := user.Current()
		if err != nil {
			return "", err
		}
		
		pathString = filepath.Join(usr.HomeDir, pathString[2:])
	}

	if !filepath.IsAbs(pathString) {
		absPath, err := filepath.Abs(pathString)
		if err != nil {
			return "", err
		}
		pathString = absPath
	}

	return pathString, nil
}

func ensureDir(pathString string) (string, error) {
	pathString, err := resolvePath(pathString)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(pathString); os.IsNotExist(err) {
		err = os.MkdirAll(pathString, 0755)
		if err != nil {
			return "", err
		}
		return pathString, nil
	} else {
		return pathString, nil
	}
}

func ensureCacheDir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}

	wallpaperCacheDir := filepath.Join(cacheDir, "linux-wallpaperengine-helper")

	return ensureDir(wallpaperCacheDir)
}

func ensureConfigDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}

	wallpaperConfigDir := filepath.Join(configDir, "linux-wallpaperengine-helper")

	return ensureDir(wallpaperConfigDir)
}
