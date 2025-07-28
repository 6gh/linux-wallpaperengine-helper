package main

import (
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"golang.org/x/image/bmp"
)

var settingWallpaper bool = false

func createWallpaperCommand(wallpaperPath string, volume float64) (string, string) {
	cmd := Config.Constants.LinuxWallpaperEngineBin + " --screen-root HDMI-A-1 --bg " + wallpaperPath

	if volume <= 1 {
		cmd += " --silent"
	} else {
		cmd += " --volume " + strconv.FormatFloat(volume, 'f', 0, 64)
	}

	cacheScreenshot := ""
	if Config.PostProcessing.Enabled {
		cacheScreenshot = path.Join(CacheDir, "screenshot.png")

		cmd += " --screenshot " + cacheScreenshot
	}

	if Config.Constants.WallpaperEngineAssets != "" {
		cmd += " --assets-dir " + Config.Constants.WallpaperEngineAssets
	}

	return cmd, cacheScreenshot
}

func updateGUIStatusText(message string) {
	if StatusText != nil {
		glib.IdleAdd(func() {
			StatusText.SetText(message)
		})
	}
}

func replaceVariablesInCommand(command string, variables map[string]string) string {
	// do not modify the original command, return a new string with replacements
	output := command
	for key, value := range variables {
		output = regexp.MustCompile(`%` + key + `%`).ReplaceAllString(output, value)
	}
	return output
}

func applyWallpaper(wallpaperPath string, volume float64) bool {
	if settingWallpaper {
		log.Println("Another wallpaper is currently being set. Please wait before setting another.")
		return false
	}
	updateGUIStatusText("Starting linux-wallpaperengine...")
	settingWallpaper = true

	defer func() {
		updateGUIStatusText("Double-click a wallpaper to apply it.")
		settingWallpaper = false
	}()

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

	if Config.PostProcessing.Enabled {
		log.Println("Post-processing enabled, running post-processing...")

		if Config.PostProcessing.ArtificialDelay > 0 {
			updateGUIStatusText("Delaying post-processing...")
			log.Printf("Waiting for %d seconds before running post-processing...", Config.PostProcessing.ArtificialDelay)
			time.Sleep(time.Duration(Config.PostProcessing.ArtificialDelay) * time.Second)
		}
		updateGUIStatusText("Running post-processing...")

		if len(Config.PostProcessing.ScreenshotFiles) > 0 && len(Config.PostProcessing.ScreenshotFiles[0]) > 0 {
			for _, filePath := range Config.PostProcessing.ScreenshotFiles {
				if path.Ext(filePath) == "" {
					filePath += ".png" // ensure the file has a .png extension
				}

				if !slices.Contains([]string{".png", ".jpg", ".jpeg", ".bmp"}, path.Ext(filePath)) {
					log.Printf("Unsupported file format for post-processing: %s", filePath)
					continue // skip unsupported formats
				}

				source, err := os.Open(cacheScreenshot)
				if err != nil {
					log.Printf("Error opening screenshot file for post-processing: %v", err)
					continue
				}
				defer source.Close()

				dest, err := os.Create(filePath)
				if err != nil {
					log.Printf("Error creating destination file for post-processing: %v", err)
					continue
				}
				defer dest.Close()

				if path.Ext(filePath) == ".png" {
					// no need for transcoding, just copy the file
					log.Printf("Copying screenshot to: %s", filePath)

					if _, err := io.Copy(dest, source); err != nil {
						log.Printf("Error copying screenshot file: %v", err)
						continue
					}

					log.Printf("Copied screenshot to: %s", filePath)
					continue
				} else {
					// transcoding the screenshot to the specified file format
					log.Printf("Transcoding screenshot to: %s", filePath)

					img, err := png.Decode(source)
					if err != nil {
						log.Printf("Error decoding PNG for transcoding: %v", err)
						continue
					}

					switch ext := path.Ext(filePath); ext {
						case ".jpg", ".jpeg":
							err = jpeg.Encode(dest, img, nil)
							if err != nil {
								log.Printf("Error encoding JPEG for transcoding: %v", err)
							}
							log.Printf("Successfully transcoded screenshot to: %s", filePath)
						case ".bmp":
							err = bmp.Encode(dest, img)
							if err != nil {
								log.Printf("Error encoding BMP for transcoding: %v", err)
							}
							log.Printf("Successfully transcoded screenshot to: %s", filePath)
						default:
							log.Printf("Unsupported file format: %s", ext)
					}
				}
			}
		}

		if Config.PostProcessing.PostCommand != "" {
			postCmdStr := replaceVariablesInCommand(Config.PostProcessing.PostCommand, map[string]string{
				"screenshot":    cacheScreenshot,
				"wallpaperPath": wallpaperPath,
				"wallpaperId":   path.Base(wallpaperPath),
				"volume":        strconv.FormatFloat(volume, 'f', 0, 64),
			})

			log.Printf("Post-processing command: %s", postCmdStr)

			pid, err := runDetachedProcess("sh", "-c", postCmdStr)
			if err != nil {
				log.Printf("Error starting post-processing command '%s': %v", postCmdStr, err)
			} else {
				log.Printf("Successfully started post-processing command (PID: %d): %s", pid, postCmdStr)
			}
		}

		// set swww wallpaper if enabled
		if Config.PostProcessing.SetSWWW {
			setSWWW(cacheScreenshot)
		}
	}

	// Save the last set wallpaper ID
	Config.SavedUIState.LastSetId = path.Base(wallpaperPath)
	return true
}

func setSWWW(screenshotPath string) bool {
	runningDaemons, err := getRunningProcessPids("swww-daemon")
	
	if err != nil {
		log.Println("Couldn't check for swww-daemon running.", err)
		return false
	}

	if len(runningDaemons) < 1 {
		log.Println("swww-daemon not running, starting swww-daemon")
		pid, err := runDetachedProcess("swww-daemon")
		if err != nil {
			log.Println("swww-daemon couldn't be started.", err)
			return false
		}

		log.Printf("Started swww-daemon [PID: %v]", pid)
	}

	pid, err := runDetachedProcess("swww", "img", screenshotPath)
	if err != nil {
		log.Println("swww command couldn't be started.", err)
		return false
	}

	log.Printf("Started swww command [PID: %v]", pid)
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
