package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
)

// Helper function to resolve a path string.
//
// If the path starts with ~/, it replaces it with the user's home directory.
// If the path is not absolute, it converts it to an absolute path.
//
// Returns the resolved path or an error if it fails.
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

// Helper function to ensure a directory exists.
//
// Takes a path string, resolves it (see resolvePath func), and checks if the directory exists.
// If it does not exist, it creates the directory with 0755 permissions.
// If it does exist, it checks for the correct permissions and returns the path.
//
// Returns the path to the directory or an error if it fails.
func ensureDir(pathString string) (string, error) {
	pathString, err := resolvePath(pathString)
	if err != nil {
		return "", err
	}

	stat, err := os.Stat(pathString)

	if os.IsNotExist(err) {
		err = os.MkdirAll(pathString, 0755)
		if err != nil {
			return "", err
		}
		return pathString, nil
	} else {
		if stat.IsDir() {
			if stat.Mode().Perm() != 0755 {
				err = os.Chmod(pathString, 0755)
				if err != nil {
					return "", fmt.Errorf("failed to set 755 permissions for %s: %v", pathString, err)
				}
			}
		} else {
			return "", fmt.Errorf("%s exists but is not a directory", pathString)
		}
	}

	return pathString, nil
}

// Helper function to ensure the cache directory exists.
// Uses the $XDG_CACHE_HOME environment variable or defaults to $HOME/.cache, and
// appends "linux-wallpaperengine-helper" to it.
//
// Creates the cache directory if it does not exist. Returns the path to the cache directory or an error if it fails.
func ensureCacheDir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}

	wallpaperCacheDir := filepath.Join(cacheDir, "linux-wallpaperengine-helper")

	return ensureDir(wallpaperCacheDir)
}

// Helper function to ensure the config directory exists.
// Uses the $XDG_CONFIG_HOME environment variable or defaults to $HOME/.config, and appends "linux-wallpaperengine-helper" to it.
//
// Creates the config directory if it does not exist. Returns the path to the config directory or an error if it fails.
func ensureConfigDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}

	wallpaperConfigDir := filepath.Join(configDir, "linux-wallpaperengine-helper")

	return ensureDir(wallpaperConfigDir)
}

// Helper function to replace given variables in the string.
//
// The first parameter is the input string, the second is a map of variables to replace.
// Returns the resulting string
//
// The variables are replaced using regex with the format of %<variable_name>%.
// These should be provided in the format of a map[string]string as such:
//
//	map[string]string{
//		"screenshot":    "~/Pictures/screenshot.png",
//		"wallpaperPath": "/path/to/wallpaper",
//		"wallpaperId":   "1234567890",
//		"volume":        "50",
//	}
//
// You should not add % in the keys.
func replaceVariablesInString(input string, variables map[string]string) string {
	// do not modify the original command, return a new string with replacements
	output := input
	for key, value := range variables {
		output = regexp.MustCompile(`%`+key+`%`).ReplaceAllString(output, value)
	}
	return output
}

// Escapes special characters in a string for use in GTK markup.
func escapeMarkup(input string) string {
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	return input
}
