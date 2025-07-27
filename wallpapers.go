package main

import (
	"log"
	"math/rand"
	"path"
	"regexp"
	"strconv"
	"time"
)


func createWallpaperCommand(wallpaperPath string, volume float64) (string, string) {
	cmd := Config.Constants.LinuxWallpaperEngineBin + " --screen-root HDMI-A-1 --bg " + wallpaperPath

	if volume <= 1 {
		cmd += " --silent"
	} else {
		cmd += " --volume " + strconv.FormatFloat(volume, 'f', 0, 64)
	}

	cacheScreenshot := ""
	if Config.PostProcessing.Enabled && Config.PostProcessing.ScreenshotFile != "" {
		cacheScreenshot = Config.PostProcessing.ScreenshotFile // ~/.cache/linux-wallpaperengine-helper/screenshot.png

		cmd += " --screenshot " + cacheScreenshot
	}

        if Config.Constants.WallpaperEngineAssets != "" {
                cmd += " --assets-dir " + Config.Constants.WallpaperEngineAssets
        }

	return cmd, cacheScreenshot
}

func applyWallpaper(wallpaperPath string, volume float64) bool {
	cmd, cacheScreenshot := createWallpaperCommand(wallpaperPath, volume)

	err := tryKillProcesses("linux-wallpaperengine")
	if err != nil {
		log.Printf("Error trying to kill existing processes: %v", err)
		return false
	}

	log.Println("Executing command:", cmd)
	pid, err := runDetachedProcess("sh", "-c", cmd)
	if err != nil {
		log.Printf("Error starting wallpaper command '%s': %v", cmd, err)
		return false // exit if we cannot start the command, to prevent multiple instances taking up resources
	} else {
		log.Printf("Successfully started detached wallpaper command (PID: %d): %s", pid, cmd)
	}

	if Config.PostProcessing.Enabled && cacheScreenshot != "" && Config.PostProcessing.PostCommand != "" {
		screenshotPattern := `%screenshot%`
		screenshotRegex := regexp.MustCompile(screenshotPattern)

		postCmdStr := Config.PostProcessing.PostCommand

		postCmdStr = screenshotRegex.ReplaceAllString(postCmdStr, cacheScreenshot)

		log.Printf("Post-processing enabled, running command: %s", postCmdStr)
		if Config.PostProcessing.ArtificialDelay > 0 {
			log.Printf("Waiting for %d seconds before running post-processing...", Config.PostProcessing.ArtificialDelay)
			time.Sleep(time.Duration(Config.PostProcessing.ArtificialDelay) * time.Second)
		}
		pid, err := runDetachedProcess("sh", "-c", postCmdStr)
		if err != nil {
			log.Printf("Error starting post-processing command '%s': %v", postCmdStr, err)
		} else {
			log.Printf("Successfully started post-processing command (PID: %d): %s", pid, postCmdStr)
		}

		// set swww wallpaper if enabled
		if Config.PostProcessing.SetSWWW {
			log.Printf("Set swww is enabled, running swww img command")
			pid, err := runDetachedProcess("sleep", "2s", "&&", "swww", "img", cacheScreenshot)
			if err != nil {
				log.Printf("Error starting swww command: %v", err)
				return false // exit if we cannot start the swww command, to prevent multiple instances taking up resources
			} else {
				log.Printf("Successfully started swww command (PID: %d)", pid)
			}
		}
	}

	// Save the last set wallpaper ID
	Config.SavedUIState.LastSetId = path.Base(wallpaperPath)
	return true
}

func restoreWallpaper() bool {
	if Config.SavedUIState.LastSetId == "" {
		log.Println("No last set wallpaper ID found, cannot restore wallpaper.")
		return false
	}

	wallpaperPath, err := resolvePath(path.Join(Config.Constants.WallpaperEngineDir, Config.SavedUIState.LastSetId))
	if err != nil {
		log.Printf("Failed to restore wallpaper; Error resolving wallpaper path: %v", err)
		return false
	}

	log.Printf("Restoring last set wallpaper: %s", wallpaperPath)
	return applyWallpaper(wallpaperPath, float64(Config.SavedUIState.Volume))
}

func applyRandomWallpaper() bool {
	if len(WallpaperItems) == 0 {
		log.Println("No wallpapers available to apply.")
		return false
	}

	nonBrokenWallpapers := make([]WallpaperItem, 0)
	for _, item := range WallpaperItems {
		if !item.IsBroken {
			nonBrokenWallpapers = append(nonBrokenWallpapers, item)
		}
	}
	if len(nonBrokenWallpapers) == 0 {
		log.Println("No non-broken wallpapers available to apply.")
		return false
	}

	randomIndex := rand.Intn(len(nonBrokenWallpapers))
	wallpaper := nonBrokenWallpapers[randomIndex]

	log.Printf("Applying random wallpaper: %s", wallpaper.WallpaperID)
	return applyWallpaper(wallpaper.WallpaperPath, float64(Config.SavedUIState.Volume))
}
