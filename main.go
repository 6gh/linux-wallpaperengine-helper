package main

import (
	_ "image/gif"  // For gif decoder
	_ "image/jpeg" // For jpeg decoder
	_ "image/png"  // For png decoder
	"log"
	"os"
	"path"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var Config *ConfigStruct
var CacheDir string

func main() {
	// ensure ~/.config/linux-wallpaperengine-helper
	configDir, err := ensureConfigDir()
	if err != nil {
		log.Fatalf("Failed to ensure config directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Config directory ensured at: %s", configDir)

	// ensure ~/.cache/linux-wallpaperengine-helper
	CacheDir, err = ensureCacheDir()
	if err != nil {
		log.Fatalf("Failed to ensure cache directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Cache directory ensured at: %s", CacheDir)

	Config = NewDefaultConfig(configDir)

	configFile := path.Join(configDir, "config.toml")
	readOrCreateConfig(configFile, Config)

	// check for --restore
	restore := false
	for _, arg := range os.Args {
		if arg == "--restore" {
			restore = true
			log.Println("Restore mode activated, applying last set wallpaper...")
			break
		}
	}

	if restore {
		if Config.SavedUIState.LastSetId == "" {
			log.Println("No last set wallpaper ID found, cannot restore wallpaper.")
			os.Exit(1)
		}
		wallpaperPath, err := resolvePath(Config.Constants.WallpaperEngineDir)
		if err != nil {
			log.Printf("Failed to restore wallpaper; Error resolving Wallpaper Engine path: %v", err)
			os.Exit(1)
		}

		success := applyWallpaper(path.Join(wallpaperPath, Config.SavedUIState.LastSetId), float64(Config.SavedUIState.Volume))
		if !success {
			log.Println("Failed to restore wallpaper.")
			os.Exit(1)
		} else {
			log.Println("Wallpaper restored successfully.")
			os.Exit(0)
		}
	}

	app := gtk.NewApplication("dev._6gh.linux-wallpaperengine-helper", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	code := app.Run(os.Args)
	saveConfig()
	os.Exit(code)
}
