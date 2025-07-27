package main

import (
	"context"
	_ "image/gif"  // For gif decoder
	_ "image/jpeg" // For jpeg decoder
	_ "image/png"  // For png decoder
	"log"
	"os"
	"path"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/urfave/cli/v3"
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

	if (len(os.Args) > 1) {
		log.Println("Running as a CLI application")

		cmd := &cli.Command{
			Name: "linux-wallpaperengine-helper",
			Usage: "A really simple helper GUI app to apply wallpapers using linux-wallpaperengine",
			Commands: []*cli.Command{
				{
					Name: "restore",
					Aliases: []string{"r"},
					Usage: "Restore the last set wallpaper set in the config",
					Flags: []cli.Flag{
						&cli.BoolFlag{
							Name: "skip-post-processing",
							Aliases: []string{"sp"},
							Usage: "Skip post-processing step when restoring wallpaper",
						},
					},
					Action: func(ctx context.Context, c *cli.Command) error {
						oldPostProcessingEnabled := Config.PostProcessing.Enabled
						defer func() {
							log.Printf("Restoring PostProcessing.Enabled to %v", oldPostProcessingEnabled)
							Config.PostProcessing.Enabled = oldPostProcessingEnabled
						}()
						if oldPostProcessingEnabled && c.Bool("skip-post-processing") {
							log.Println("Skipping post-processing step as requested.")
							Config.PostProcessing.Enabled = false
						}

						if !restoreWallpaper() {
							log.Println("Failed to restore last set wallpaper.")
							return cli.Exit("Failed to restore last set wallpaper.", 1)
						}
						return nil
					},
				},
				{
					Name: "kill",
					Aliases: []string{"k"},
					Usage: "Kill any running linux-wallpaperengine process",
					Action: func(ctx context.Context, c *cli.Command) error {
						if err := tryKillProcesses("linux-wallpaperengine"); err != nil {
							log.Printf("Error trying to kill existing processes: %v", err)
							return cli.Exit("Failed to kill existing processes.", 1)
						}
						return nil
					},
				},
			},
		}

		if err := cmd.Run(context.Background(), os.Args); err != nil {
			log.Fatal(err)
    }
	} else {
		app := gtk.NewApplication("dev._6gh.linux-wallpaperengine-helper", gio.ApplicationFlagsNone)
		app.ConnectActivate(func() { activate(app) })

		code := app.Run(os.Args)
		saveConfig()
		os.Exit(code)
	}
}
