package main

import (
	"context"
	_ "image/gif"  // For gif decoder
	_ "image/jpeg" // For jpeg decoder
	_ "image/png"  // For png decoder
	"log"
	"os"
	"path"
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/urfave/cli/v3"
)

var CacheDir string
var Config *ConfigStruct
var ConfigDir string

func main() {
	// ensure ~/.config/linux-wallpaperengine-helper
	ConfigDir, err := ensureConfigDir()
	if err != nil {
		log.Fatalf("Failed to ensure config directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Config directory ensured at: %s", ConfigDir)

	Config = NewDefaultConfig(ConfigDir)

	configFile := path.Join(ConfigDir, "config.toml")
	readOrCreateConfig(configFile, Config)

	// ensure ~/.cache/linux-wallpaperengine-helper
	CacheDir, err = ensureCacheDir()
	if err != nil {
		log.Fatalf("Failed to ensure cache directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Cache directory ensured at: %s", CacheDir)

	if len(os.Args) > 1 {
		log.Println("Running as a CLI application")

		cmd := &cli.Command{
			Name:  "linux-wallpaperengine-helper",
			Usage: "A really simple helper GUI app to apply wallpapers using linux-wallpaperengine",
			Commands: []*cli.Command{
				{
					Name:    "restore",
					Aliases: []string{"r"},
					Usage:   "Restore the last set wallpaper set in the config",
					Flags: []cli.Flag{
						&cli.BoolWithInverseFlag{
							Name:     "post-processing",
							Usage:    "Override post-processing step, e.g. --post-processing or --no-post-processing. Setting this to false will skip post-processing entirely.",
							Category: "Post Processing",
							Required: false,
							OnlyOnce: true,
							Action: func(ctx context.Context, c *cli.Command, value bool) error {
								log.Printf("PostProcessing.Enabled set to %v", value)
								Config.PostProcessing.Enabled = value
								return nil
							},
						},
						&cli.DurationFlag{
							Name:     "artificial-delay",
							Aliases:  []string{"delay"},
							Usage:    "Override artificial delay in seconds to wait before post-processing, e.g. --artificial-delay=2s",
							Category: "Post Processing",
							Action: func(ctx context.Context, c *cli.Command, value time.Duration) error {
								log.Printf("PostProcessing.ArtificialDelay set to %vs", int64(value.Seconds()))
								Config.PostProcessing.ArtificialDelay = int64(value.Seconds())
								return nil
							},
						},
						&cli.StringSliceFlag{
							Name:      "screenshot",
							Usage:     "Override screenshot files to copy output screenshot to, e.g. --screenshot=/path/to/screenshot.png --screenshot=/path/to/another.jpg",
							TakesFile: true,
							Category:  "Post Processing",
							Action: func(ctx context.Context, c *cli.Command, value []string) error {
								log.Printf("PostProcessing.ScreenshotFiles set to %v", value)
								Config.PostProcessing.ScreenshotFiles = value
								return nil
							},
						},
						&cli.StringFlag{
							Name:     "post-command",
							Aliases:  []string{"command"},
							Usage:    "Override post-command to run, e.g. --post-command='your-command'",
							Category: "Post Processing",
							Action: func(ctx context.Context, c *cli.Command, value string) error {
								Config.PostProcessing.PostCommand = value
								return nil
							},
						},
						&cli.BoolWithInverseFlag{
							Name:     "swww",
							Usage:    "Override whether to set the wallpaper using swww after applying the wallpaper, e.g. --swww or --no-swww",
							Category: "Post Processing",
							Action: func(ctx context.Context, c *cli.Command, value bool) error {
								log.Printf("PostProcessing.SetSWWW set to %v", value)
								Config.PostProcessing.SetSWWW = value
								return nil
							},
						},
					},
					Action: func(ctx context.Context, c *cli.Command) error {
						if err := restoreWallpaper(); err != nil {
							log.Println("Failed to restore last set wallpaper:", err)
							return cli.Exit("Failed to restore last set wallpaper.", 1)
						}
						return nil
					},
				},
				{
					Name:    "kill",
					Aliases: []string{"k"},
					Usage:   "Kill any running linux-wallpaperengine process",
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
		app := gtk.NewApplication("dev._6gh.linux-wallpaperengine-helper", gio.ApplicationDefaultFlags)
		app.ConnectActivate(func() { activate(app) })

		code := app.Run(os.Args)
		saveConfig()
		os.Exit(code)
	}
}
