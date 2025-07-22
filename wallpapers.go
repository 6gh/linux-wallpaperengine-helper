package main

import (
	"log"
	"path"
	"regexp"
	"strconv"
	"syscall"
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

	runningPids, err := getRunningProcessPids("linux-wallpaperengine");
	if err != nil {
		log.Println("Error checking if linux-wallpaperengine is running:", err)
		return false
	}
	
	if len(runningPids) > 0 {
		log.Println("linux-wallpaperengine is already running, killing old process(es)...")
		for _, pid := range runningPids {
			pidInt, err := strconv.Atoi(pid)
			if err != nil {
				log.Printf("Error converting PID '%s' to int: %v", pid, err)
				return false // exit if we cannot convert the PID to an int, to prevent multiple instances taking up resources
			}
			err = syscall.Kill(pidInt, syscall.SIGTERM)
			if err != nil {
				log.Printf("Error killing process with PID %d: %v", pidInt, err)
				return false // exit if we cannot kill the old process, to prevent multiple instances taking up resources
			} else {
				log.Printf("Successfully killed process with PID %d", pidInt)
			}
		}
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
		pid, err := runDetachedProcess("sh", "-c", postCmdStr)
		if err != nil {
			log.Printf("Error starting post-processing command '%s': %v", postCmdStr, err)
		} else {
			log.Printf("Successfully started post-processing command (PID: %d): %s", pid, postCmdStr)
		}

		// set swww wallpaper if enabled
		if Config.PostProcessing.SetSWWW {
			log.Printf("Set swww is enabled, running swww img command")
			pid, err := runDetachedProcess("swww", "img", cacheScreenshot)
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
