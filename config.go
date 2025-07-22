package main

import (
	"log"
	"os"
	"path"

	"github.com/pelletier/go-toml/v2"
)

type ConstantsStruct struct {
	DiscardProcessLogs       bool   `toml:"discard_process_logs"      comment:"Whether to pipe detached processes' logs to /dev/null"`
	LinuxWallpaperEngineBin  string `toml:"linux_wallpaperengine_bin" comment:"The absolute path to the binary, in case the binary isn't in PATH"`
	WallpaperEngineDir       string `toml:"wallpaper_engine_dir"      comment:"The absolute path to the workshop content directory of Wallpaper Engine; where the wallpapers are stored"`
	WallpaperEngineAssets    string `toml:"wallpaper_engine_assets"   comment:"The absolute path to the assets directory of Wallpaper Engine; https://github.com/Almamu/linux-wallpaperengine#1-get-wallpaper-engine-assets"`
}

type PostProcessingStruct struct {
	Enabled        bool   `toml:"enabled"         comment:"Whether to enable post-processing features below"`
	ScreenshotFile string `toml:"screenshot_file" comment:"The file where the screenshot will be saved with the --screenshot flag"`
	PostCommand    string `toml:"post_command"    comment:"The command to run after the wallpaper is applied, with some placeholders"`
	SetSWWW        bool   `toml:"set_swww"        comment:"Whether to set the wallpaper using swww after applying the wallpaper; requires screenshot_file to be set and swww to be working"`
}

type SavedUIStateStruct struct {
	LastSetId  string   `toml:"last_set_id" comment:"The last set wallpaper ID, used for restoring the wallpaper"` // # TODO: add multi monitor support
	Volume     int64    `toml:"volume"      comment:"The volume level for the wallpaper engine, 0-100; 0 = --silent, > 0 = --volume <value>"`
	HideBroken bool     `toml:"hide_broken" comment:"Whether to hide broken wallpapers from the UI"`
	Broken     []string `toml:"broken"      comment:"Wallpapers marked as 'broken'; can be hidden from UI or shown at the end of the list"`
	Favorites  []string `toml:"favorites"   comment:"Wallpapers marked as 'favorite'; shown at the top of the list"`
}

type ConfigStruct struct {
	Constants      ConstantsStruct      `toml:"Constants"`
	PostProcessing PostProcessingStruct `toml:"PostProcessing"`
	SavedUIState   SavedUIStateStruct   `toml:"SavedUIState"`
}

func NewDefaultConfig(configDir string) *ConfigStruct {
	return &ConfigStruct{
		Constants: ConstantsStruct{
			DiscardProcessLogs:       true,
			LinuxWallpaperEngineBin:  "linux-wallpaperengine",
			WallpaperEngineDir:       path.Join(os.Getenv("HOME"), ".steam", "steam", "steamapps", "workshop", "content", "431960"),
			WallpaperEngineAssets:    "",
		},
		PostProcessing: PostProcessingStruct{
			Enabled:        false,
			ScreenshotFile: path.Join(configDir, "screenshot.png"),
			PostCommand:    "",
		},
		SavedUIState: SavedUIStateStruct{
			LastSetId: "",
			Broken:    []string{},
			Favorites: []string{},
			HideBroken: false,
			Volume: 100,
		},
	}
}

func readOrCreateConfig(configFile string, config *ConfigStruct) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("Config file does not exist, creating default at: %s", configFile)

		content, err := toml.Marshal(config)
		if err != nil {
			log.Fatalf("Failed to marshal config to TOML: %v", err)
			return err
		}

		err = os.WriteFile(configFile, content, 0644)
		if err != nil {
			log.Fatalf("Failed to write config file: %v", err)
			return err
		}

		log.Printf("Default config file created at: %s", configFile)
	} else {
		log.Printf("Config file already exists at: %s", configFile)
		content, err := os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
			return err
		}
		err = toml.Unmarshal(content, &config)
		if err != nil {
			log.Fatalf("Failed to unmarshal config file: %v", err)
			return err
		}
		log.Printf("Config file loaded from: %s", configFile)
	}

	return nil
}

func saveConfig() {
	configDir, err := ensureConfigDir()
	if err != nil {
		log.Printf("Failed to ensure config directory: %v", err)
		return
	}

	configFile := path.Join(configDir, "config.toml")
	content, err := toml.Marshal(Config)
	if err != nil {
		log.Printf("Failed to marshal config to TOML: %v", err)
		return
	}

	err = os.WriteFile(configFile, content, 0644)
	if err != nil {
		log.Printf("Failed to write config file: %v", err)
		return
	}

	log.Printf("Config saved to: %s", configFile)
}
