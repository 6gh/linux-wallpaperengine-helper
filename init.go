package main

import (
	"image"
	_ "image/gif"  // For gif decoder
	_ "image/jpeg" // For jpeg decoder
	_ "image/png"  // For png decoder
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/disintegration/imaging"

	"github.com/pelletier/go-toml/v2"
)

type ConstantsStruct struct {
	LinuxWallpaperEngineBin  string `toml:"linux_wallpaperengine_bin"` // in case someone didnt install it in PATH
	WallpaperEngineDir       string `toml:"wallpaper_engine_dir"`      // in case you installed wallpaper engine in a different directory
	LastSetId 							 string `toml:"last_set_id"`               // the last set wallpaper ID, used for restoring the wallpaper TODO: add multi-monitor support
}

type PostProcessingStruct struct {
	Enabled        bool   `toml:"enabled"`         // whether any of this section is enabled
	ScreenshotFile string `toml:"screenshot_file"` // the file where the screenshot will be saved with the --screenshot flag
	PostCommand    string `toml:"post_command"`    // the command to run after the wallpaper is applied, with some placeholders
}

type SavedUIStateStruct struct {
	Broken     []string `toml:"broken"`    // user marked as "broken"; can be hidden from UI or shown at the end of the list
	Favorites  []string `toml:"favorites"` // user marked as "favorite"; shown at the top of the list
	HideBroken bool   `toml:"hide_broken"` // whether to hide broken wallpapers from the UI
	Volume     int64 `toml:"volume"`       // the volume level for the wallpaper engine, 0-100; 0 = --silent, > 0 = --volume <value>
}

type ConfigStruct struct {
	Constants ConstantsStruct `toml:"Constants"`
	PostProcessing PostProcessingStruct `toml:"PostProcessing"`
	SavedUIState SavedUIStateStruct `toml:"SavedUIState"`
}

func NewDefaultConfig(configDir string) *ConfigStruct {
	return &ConfigStruct{
		Constants: ConstantsStruct{
			LinuxWallpaperEngineBin:  "linux-wallpaperengine",
			WallpaperEngineDir:       path.Join(os.Getenv("HOME"), ".steam", "steam", "steamapps", "workshop", "content", "431960"),
			LastSetId:                "",
		},
		PostProcessing: PostProcessingStruct{
			Enabled:        false,
			ScreenshotFile: path.Join(configDir, "screenshot.png"),
			PostCommand:    "",
		},
		SavedUIState: SavedUIStateStruct{
			Broken:    []string{},
			Favorites: []string{},
			HideBroken: false,
			Volume: 100,
		},
	}
}

var Config *ConfigStruct
var cacheDir string
var flowBox *gtk.FlowBox
var window *gtk.ApplicationWindow
var imageClickSignalHandlers []glib.SignalHandle

func main() {
	app := gtk.NewApplication("dev._6gh.linux-wallpaperengine-helper", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	// ensure ~/.config/linux-wallpaperengine-helper
	configDir, err := ensureConfigDir()
	if err != nil {
		log.Fatalf("Failed to ensure config directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Config directory ensured at: %s", configDir)

	// ensure ~/.cache/linux-wallpaperengine-helper
	cacheDir, err = ensureCacheDir()
	if err != nil {
		log.Fatalf("Failed to ensure cache directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Cache directory ensured at: %s", cacheDir)

	Config = NewDefaultConfig(configDir)

	configFile := path.Join(configDir, "config.toml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Printf("Config file does not exist, creating default at: %s", configFile)
		
		content, err := toml.Marshal(Config)
		if err != nil {
			log.Fatalf("Failed to marshal config to TOML: %v", err)
			return
		}

		err = os.WriteFile(configFile, content, 0644)
		if err != nil {
			log.Fatalf("Failed to write config file: %v", err)
			return
		}

		log.Printf("Default config file created at: %s", configFile)
	} else {
		log.Printf("Config file already exists at: %s", configFile)
		content, err := os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
			return
		}
		err = toml.Unmarshal(content, &Config)
		if err != nil {
			log.Fatalf("Failed to unmarshal config file: %v", err)
			return
		}
		log.Printf("Config file loaded from: %s", configFile)
	}

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
		if Config.Constants.LastSetId == "" {
			log.Println("No last set wallpaper ID found, cannot restore wallpaper.")
			os.Exit(1)
		}
		wallpaperPath := path.Join(Config.Constants.WallpaperEngineDir, Config.Constants.LastSetId)

		if _, err := os.Stat(wallpaperPath); os.IsNotExist(err) {
			log.Printf("Could'nt find wallpaper at path: %s", wallpaperPath)
			log.Println("Cannot restore wallpaper, exiting.")
			os.Exit(1)
		}

		log.Printf("Restoring wallpaper from path: %s", wallpaperPath)
		success := applyWallpaper(wallpaperPath, float64(Config.SavedUIState.Volume))
	  if !success {
			log.Println("Failed to restore wallpaper, exiting.")
			os.Exit(1)
		} else {
			log.Println("Wallpaper restored successfully.")
			os.Exit(0)
		}
	}

	if code := app.Run(os.Args); code > 0 {
		saveConfig()
		os.Exit(code)
	} else {
		saveConfig()
		os.Exit(0)
	}
}

func ensureCacheDir() (string, error) {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		cacheDir = path.Join(os.Getenv("HOME"), ".cache")
	}

	wallpaperCacheDir := path.Join(cacheDir, "linux-wallpaperengine-helper")
	if _, err := os.Stat(wallpaperCacheDir); os.IsNotExist(err) {
		err = os.MkdirAll(wallpaperCacheDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create cache directory: %v", err)
			return "", err
		}
		log.Printf("Created cache directory: %s", wallpaperCacheDir)
		return wallpaperCacheDir, nil
	} else {
		log.Printf("Cache directory already exists: %s", wallpaperCacheDir)
		return wallpaperCacheDir, nil
	}
}

func ensureConfigDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = path.Join(os.Getenv("HOME"), ".config")
	}

	wallpaperConfigDir := path.Join(configDir, "linux-wallpaperengine-helper")
	if _, err := os.Stat(wallpaperConfigDir); os.IsNotExist(err) {
		err = os.MkdirAll(wallpaperConfigDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create config directory: %v", err)
			return "", err
		}
		log.Printf("Created config directory: %s", wallpaperConfigDir)
		return wallpaperConfigDir, nil
	} else {
		log.Printf("Config directory already exists: %s", wallpaperConfigDir)
		return wallpaperConfigDir, nil
	}
}

func isProcessRunning(processName string) ([]string, error) {
	cmd := exec.Command("pidof", processName)
	// if error code is 1, the process is not running
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return []string{}, nil 
		}
		return []string{}, err 
	}
	// If output is not empty, return the pids
	if strings.TrimSpace(string(output)) != "" {
		pids := strings.Fields(string(output))
		log.Printf("Process '%s' is running with PIDs: %v", processName, pids)
		return pids, nil
	}
	return []string{}, nil
}

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

	return cmd, cacheScreenshot
}

func applyWallpaper(wallpaperPath string, volume float64) bool {
	cmd, cacheScreenshot := createWallpaperCommand(wallpaperPath, volume)

	runningPids, err := isProcessRunning("linux-wallpaperengine");
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
	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		log.Printf("Warning: Could not open /dev/null for detaching process I/O: %v", err)
	} else {
		execCmd.Stdin = devNull
		execCmd.Stdout = devNull
		execCmd.Stderr = devNull
		defer devNull.Close()
	}

	err = execCmd.Start()
	if err != nil {
		log.Printf("Error starting wallpaper command '%s': %v", cmd, err)
		return false // exit if we cannot start the command, to prevent multiple instances taking up resources
	} else {
		log.Printf("Successfully started detached wallpaper command (PID: %d): %s", execCmd.Process.Pid, cmd)
	}

	if Config.PostProcessing.Enabled && cacheScreenshot != "" && Config.PostProcessing.PostCommand != "" {
		screenshotPattern := `%screenshot%`
		screenshotRegex := regexp.MustCompile(screenshotPattern)

		postCmdStr := Config.PostProcessing.PostCommand

		postCmdStr = screenshotRegex.ReplaceAllString(postCmdStr, cacheScreenshot)

		log.Printf("Post-processing enabled, running command: %s", postCmdStr)
		postCmd := exec.Command("sh", "-c", postCmdStr)
		postCmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true,
		}
		postCmd.Stdin = devNull
		postCmd.Stdout = devNull
		postCmd.Stderr = devNull

		err = postCmd.Start()
		if err != nil {
			log.Printf("Error starting post-processing command '%s': %v", postCmdStr, err)
		} else {
			log.Printf("Successfully started post-processing command (PID: %d): %s", postCmd.Process.Pid, postCmdStr)
		}
	}

	// Save the last set wallpaper ID
	Config.Constants.LastSetId = path.Base(wallpaperPath)
	return true
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

func cacheImage(imagePath string, cachedThumbnailPath string) {
	// TODO: add support for gifs
	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("Error opening image file %s: %v", imagePath, err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Error decoding image %s: %v", imagePath, err)
		return
	}

	// Resize the image to the desired thumbnail size (128x128 pixels)
	// this is to have a uniform look for all images
	thumbnail := imaging.Fit(img, 128, 128, imaging.Lanczos)

	cachedThumbnailDir := path.Dir(cachedThumbnailPath)

	err = os.MkdirAll(cachedThumbnailDir, 0755)
	if err != nil {
		log.Printf("Error creating directory %s: %v", cachedThumbnailDir, err)
		return
	}

	err = imaging.Save(thumbnail, cachedThumbnailPath)
	if err != nil {
		log.Printf("Error saving thumbnail to %s: %v", cachedThumbnailDir, err)
		return
	}

	log.Printf("Thumbnail saved to: %s", cachedThumbnailDir)
}

func loadImageAsync(imagePath string, targetImage *gtk.Image, pixelSize int) {
	go func() {
		// check for cached thumbnail first
		// TODO: add support for gifs
		tempFilePath := path.Join(cacheDir, path.Base(path.Dir(imagePath)))
		cachedThumbnailPath := path.Join(tempFilePath, "thumbnail.png") // ~/.cache/linux-wallpaperengine-helper/<wallpaper_id>/thumbnail.png

		if _, err := os.Stat(cachedThumbnailPath); os.IsNotExist(err) {
			// If the cached thumbnail does not exist, create it
			log.Printf("Cached thumbnail not found for %s, creating it...", imagePath)
			cacheImage(imagePath, cachedThumbnailPath)
		} else {
			log.Printf("Using cached thumbnail for %s", imagePath)
		}

		// convert to pixbuf
		pixbuf,err := gdkpixbuf.NewPixbufFromFile(cachedThumbnailPath)
		if err != nil {
			log.Printf("Error creating GdkPixbuf from file %s: %v", cachedThumbnailPath, err)
			return
		}

		// convert to paintable as image.SetFromPixbuf is deprecated
		paintable := gdk.NewTextureForPixbuf(pixbuf)

		// Update the gtk.Image widget on the main GTK thread
		glib.IdleAdd(func() {
			targetImage.SetFromPaintable(paintable)
			targetImage.SetPixelSize(pixelSize)
		})
	}()
}

func refreshImages() {
	flowBox.RemoveAll()
	// remove all previous child-activated signal handlers
	// else we would have multiple handlers for the same signal
	// causing multiple wallpapers to be applied on click
	for _, id := range imageClickSignalHandlers {
		flowBox.HandlerDisconnect(id)
	}

	// get all wallpapers from the wallpaperengine directory
	wallpaperDir := Config.Constants.WallpaperEngineDir
	wallpapers, err := os.ReadDir(wallpaperDir)
	if err != nil {
		// Display error message if directory cannot be read
		errorLabel := gtk.NewLabel("Error reading wallpaper directory: " + err.Error())
		errorLabel.SetHExpand(true)
		errorLabel.SetVExpand(true)
		errorLabel.SetHAlign(gtk.AlignCenter)
		errorLabel.SetVAlign(gtk.AlignCenter)
		window.SetChild(errorLabel) // Set the error label as the sole child
		window.SetVisible(true)
		return // Exit activation function
	}

	signalHandler := flowBox.Connect("child-activated", func(box *gtk.FlowBox, child *gtk.FlowBoxChild) {
		if child == nil {
			return
		}
		// Retrieve the wallpaper name from the tooltip text of the activated image
		wallpaperName := child.Child().(*gtk.Overlay).Child().(*gtk.Image).TooltipText()
		if wallpaperName == "" {
			log.Println("[WARN] No wallpaper name found for the activated child.")
			return
		}
		log.Println("Applying wallpaper:", wallpaperName)
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperName) 
		applyWallpaper(fullWallpaperPath, float64(Config.SavedUIState.Volume))
	})
	imageClickSignalHandlers = append(imageClickSignalHandlers, signalHandler)

	if len(wallpapers) == 0 {
		// Display error message if no wallpapers are found
		errorLabel := gtk.NewLabel("No wallpapers found in the directory: " + wallpaperDir)
		errorLabel.SetHExpand(true)
		errorLabel.SetVExpand(true)
		errorLabel.SetHAlign(gtk.AlignCenter)
		errorLabel.SetVAlign(gtk.AlignCenter)
		window.SetChild(errorLabel) // Set the error label as the sole child
		window.SetVisible(true)
		return // Exit activation function
	}

	// put favorites first
	sort.SliceStable(wallpapers, func(i, j int) bool {
		// sort favorites first
		if slices.Contains(Config.SavedUIState.Favorites, wallpapers[i].Name()) && !slices.Contains(Config.SavedUIState.Favorites, wallpapers[j].Name()) {
			return true // i is a favorite, j is not
		} else {
			return false // i is not a favorite, j is a favorite or both are not favorites
		}
	})

	if Config.SavedUIState.HideBroken {
		// filter out broken wallpapers if hideBroken is true
		wallpapers = func() []fs.DirEntry {
			filtered := make([]fs.DirEntry, 0, len(wallpapers))
			for _, wallpaper := range wallpapers {
				if !slices.Contains(Config.SavedUIState.Broken, wallpaper.Name()) {
					filtered = append(filtered, wallpaper)
				}
			}
			return filtered
		}()
	} else {
		// put broken wallpapers at the end
		sort.SliceStable(wallpapers, func(i, j int) bool {
			// sort broken wallpapers last
			if slices.Contains(Config.SavedUIState.Broken, wallpapers[i].Name()) && !slices.Contains(Config.SavedUIState.Broken, wallpapers[j].Name()) {
				return false // i is broken, j is not
			} else if !slices.Contains(Config.SavedUIState.Broken, wallpapers[i].Name()) && slices.Contains(Config.SavedUIState.Broken, wallpapers[j].Name()) {
				return true // i is not broken, j is broken
			} else {
				return false // both are either broken or not broken, keep original order
			}
		})
	}


	for _, wallpaper := range wallpapers {
		if wallpaper.IsDir() {
			// find image file in the wallpaper directory
			imageFiles, err := os.ReadDir(path.Join(wallpaperDir, wallpaper.Name()))
			if err != nil {
				log.Println("Error reading wallpaper subdirectory:", err)
				continue
			}
			var imageFile string
			for _, file := range imageFiles {
				if file.IsDir() {
					continue // Skip sub-directories
				}
				if file.Name() == "preview.jpg" || file.Name() == "preview.png" || file.Name() == "preview.gif" {
					imageFile = path.Join(wallpaperDir, wallpaper.Name(), file.Name())
					break // Found the preview image, no need to check further
				}
			}

			if imageFile == "" {
				log.Println("No preview image found for wallpaper:", wallpaper.Name())
				continue // Skip this wallpaper if no preview image is found
			}

			isFavorite := slices.Contains(Config.SavedUIState.Favorites, wallpaper.Name())
			isBroken := slices.Contains(Config.SavedUIState.Broken, wallpaper.Name())

			iconOverlay := gtk.NewOverlay() // in case we want to add an icon overlay later
			imageWidget := gtk.NewImageFromIconName("image-x-generic-symbolic")
			imageWidget.SetPixelSize(128)
			imageWidget.SetHAlign(gtk.AlignCenter)
			imageWidget.SetVAlign(gtk.AlignCenter)
			imageWidget.SetMarginTop(10)
			imageWidget.SetMarginBottom(10)
			imageWidget.SetMarginStart(10)
			imageWidget.SetMarginEnd(10)
			imageWidget.SetTooltipText(wallpaper.Name())

			if isFavorite {
				// if the wallpaper is a favorite, add a star icon to the top right of the image
				favoriteIcon := gtk.NewImageFromIconName("emote-love-symbolic")
				favoriteIcon.SetPixelSize(16)
				favoriteIcon.SetHAlign(gtk.AlignEnd)
				favoriteIcon.SetVAlign(gtk.AlignStart)
				iconOverlay.AddOverlay(favoriteIcon)
			}
			if isBroken {
				// if the wallpaper is marked as broken, add a warning icon to the top right of the image
				warningIcon := gtk.NewImageFromIconName("dialog-warning-symbolic")
				warningIcon.SetPixelSize(16)
				warningIcon.SetHAlign(gtk.AlignEnd)
				warningIcon.SetVAlign(gtk.AlignStart)
				iconOverlay.AddOverlay(warningIcon)
			}

			iconOverlay.SetChild(imageWidget)

			attachContextMenu(iconOverlay, wallpaper.Name(), isFavorite, isBroken)

			flowBox.Append(iconOverlay)
			loadImageAsync(imageFile, imageWidget, 128)
		}
	}
}

func attachContextMenu(imageWidget *gtk.Overlay, wallpaperName string, isFavorite bool, isBroken bool) {
	actionGroup := gio.NewSimpleActionGroup()

	// apply action
	applyAction := gio.NewSimpleAction("apply", nil)
	applyAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Println("Applying wallpaper:", wallpaperName)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperName)
		applyWallpaper(fullWallpaperPath, float64(Config.SavedUIState.Volume))
	})
	actionGroup.AddAction(&applyAction.Action)

	// toggle_favorite action
	toggleFavoriteAction := gio.NewSimpleAction("toggle_favorite", nil)
	toggleFavoriteAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		if isFavorite {
			log.Printf("Removing %s from favorites", wallpaperName)
			Config.SavedUIState.Favorites = slices.Delete(Config.SavedUIState.Favorites, slices.Index(Config.SavedUIState.Favorites, wallpaperName), slices.Index(Config.SavedUIState.Favorites, wallpaperName)+1)
		} else {
			log.Printf("Adding %s to favorites", wallpaperName)
			Config.SavedUIState.Favorites = append(Config.SavedUIState.Favorites, wallpaperName)
		}

		refreshImages()
	})
	actionGroup.AddAction(&toggleFavoriteAction.Action)

	// toggle_broken action
	toggleBrokenAction := gio.NewSimpleAction("toggle_broken", nil)
	toggleBrokenAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		if isBroken {
			log.Printf("Marking %s as not broken", wallpaperName)
			Config.SavedUIState.Broken = slices.Delete(Config.SavedUIState.Broken, slices.Index(Config.SavedUIState.Broken, wallpaperName), slices.Index(Config.SavedUIState.Broken, wallpaperName)+1)
		} else {
			log.Printf("Marking %s as broken", wallpaperName)
			Config.SavedUIState.Broken = append(Config.SavedUIState.Broken, wallpaperName)
		}

		refreshImages()
	})
	actionGroup.AddAction(&toggleBrokenAction.Action)

	// open_directory action
	openDirectoryAction := gio.NewSimpleAction("open_directory", nil)
	openDirectoryAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Printf("Opening directory for %s\n", wallpaperName)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullPath := path.Join(wallpaperDir, wallpaperName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			log.Printf("Wallpaper directory does not exist: %s", fullPath)
			return
		}
		cmd := exec.Command("xdg-open", fullPath)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // run in a new session
		}
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
		if err != nil {
			log.Printf("Warning: Could not open /dev/null for detaching process I/O: %v", err)
		} else {
			cmd.Stdin = devNull
			cmd.Stdout = devNull
			cmd.Stderr = devNull
			defer devNull.Close()
		}
		err = cmd.Start()
		if err != nil {
			log.Printf("Error opening directory %s: %v", fullPath, err)
			return
		}
		log.Printf("Opened directory for wallpaper %s: %s", wallpaperName, fullPath)
	})
	actionGroup.AddAction(&openDirectoryAction.Action)

	// copy_command action
	copyCommandAction := gio.NewSimpleAction("copy_command", nil)
	copyCommandAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Printf("Copying command for %s to clipboard", wallpaperName)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperName)
		cmd, _ := createWallpaperCommand(fullWallpaperPath, float64(Config.SavedUIState.Volume))
		clipboard := gdk.DisplayGetDefault().Clipboard()
		// if clipboard == nil {
		// 	log.Println("Error getting clipboard")
		// 	return
		// }
		clipboard.SetText(cmd)
		log.Printf("Command copied to clipboard: %s", cmd)
	})
	actionGroup.AddAction(&copyCommandAction.Action)

	imageWidget.InsertActionGroup(wallpaperName, actionGroup)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	imageWidget.AddController(gesture)
	gesture.ConnectReleased(func(nPress int, x, y float64) {
		if nPress == 1 { // ensure single click
			// context menu for the image
			contextMenuModel := gio.NewMenu()

			contextMenuModel.Append("Apply Wallpaper", wallpaperName + ".apply")
			if isFavorite {
				contextMenuModel.Append("Remove from Favorites", wallpaperName + ".toggle_favorite")
			} else {
				contextMenuModel.Append("Add to Favorites", wallpaperName + ".toggle_favorite")
			}
			if isBroken {
				contextMenuModel.Append("Unmark as Broken", wallpaperName + ".toggle_broken")
			} else {
				contextMenuModel.Append("Mark as Broken", wallpaperName + ".toggle_broken")
			}
			contextMenuModel.Append("Open Wallpaper Directory", wallpaperName + ".open_directory")
			contextMenuModel.Append("Copy Command to Clipboard", wallpaperName + ".copy_command")

			contextMenu := gtk.NewPopoverMenuFromModel(contextMenuModel)
			contextMenu.SetParent(imageWidget)

			// makes the popover appear at the clicked position
			rect := gdk.NewRectangle(int(x), int(y), 1, 1)
			contextMenu.SetPointingTo(&rect)

			// makes the popover appear at the bottom of the cursor
			contextMenu.SetPosition(gtk.PosBottom) 
			contextMenu.SetHasArrow(true) 

			contextMenu.Popup()
		}
	})

	log.Println("Context menu attached to image widget for wallpaper:", wallpaperName)
}

func activate(app *gtk.Application) {
	window = gtk.NewApplicationWindow(app)
	window.SetTitle("linux-wallpaperengine Helper")

	flowBox = gtk.NewFlowBox()
	flowBox.SetSelectionMode(gtk.SelectionSingle)
	flowBox.SetHomogeneous(true)
	flowBox.SetColumnSpacing(12)
	flowBox.SetRowSpacing(12)
	flowBox.SetMaxChildrenPerLine(8)
	flowBox.SetHAlign(gtk.AlignCenter)
	flowBox.SetVAlign(gtk.AlignStart)
	flowBox.SetMarginTop(10)
	flowBox.SetMarginBottom(10)
	flowBox.SetMarginStart(10)
	flowBox.SetMarginEnd(10)

	// make the bottom controls 
	controlBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	controlBar.SetHAlign(gtk.AlignCenter)
	controlBar.SetVAlign(gtk.AlignEnd)
	controlBar.SetMarginTop(10)
	controlBar.SetMarginBottom(10)
	controlBar.SetMarginStart(10)
	controlBar.SetMarginEnd(10)

	// refresh button
	refreshButton := gtk.NewButtonWithLabel("Refresh")
	refreshButton.SetHExpand(false)
	refreshButton.SetHAlign(gtk.AlignStart)
	refreshButton.SetVAlign(gtk.AlignCenter)
	refreshButton.Connect("clicked", func() {
		log.Println("Refreshing wallpapers...")
		refreshImages()
	})
	controlBar.Append(refreshButton)

	// volume control
	volumeContainer := gtk.NewBox(gtk.OrientationVertical, 0)
	volumeContainer.SetHExpand(true)
	volumeContainer.SetHAlign(gtk.AlignStart)
	volumeContainer.SetVAlign(gtk.AlignCenter)
	volumeLabel := gtk.NewLabel("Volume: " + strconv.FormatInt(Config.SavedUIState.Volume, 10) + "%")
	volumeLabel.SetHAlign(gtk.AlignCenter)
	volumeLabel.SetVAlign(gtk.AlignCenter)
	volumeSlider := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 0, 100, 1)
	volumeSlider.SetValue(float64(Config.SavedUIState.Volume))
	volumeSlider.SetHExpand(true)
	volumeSlider.SetVExpand(false)
	volumeSlider.SetHAlign(gtk.AlignCenter)
	volumeSlider.SetVAlign(gtk.AlignCenter)
	volumeSlider.SetSizeRequest(200, -1)
	volumeSlider.Connect("value-changed", func(slider *gtk.Scale) {
		value := slider.Value()
		volumeLabel.SetLabel("Volume: " + strconv.FormatFloat(value, 'f', 0, 64) + "%")
		Config.SavedUIState.Volume = int64(value)
	})
	volumeContainer.Append(volumeLabel)
	volumeContainer.Append(volumeSlider)
	controlBar.Append(volumeContainer)

	refreshImages()

	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolledWindow.SetMinContentHeight(600)
	scrolledWindow.SetMinContentWidth(800)
	scrolledWindow.SetVExpand(true)
	scrolledWindow.SetChild(flowBox)

	topText := gtk.NewLabel("Select a wallpaper to apply it.")
	topText.SetHAlign(gtk.AlignCenter)
	topText.SetVAlign(gtk.AlignStart)
	topText.SetMarginTop(10)
	topText.SetMarginBottom(10)
	topText.SetMarginStart(10)
	topText.SetMarginEnd(10)

	vBox := gtk.NewBox(gtk.OrientationVertical, 0)
	vBox.Append(topText)
	vBox.Append(scrolledWindow)
	vBox.Append(controlBar)

	window.SetChild(vBox)
	window.SetDefaultSize(800, 600)
	window.SetVisible(true)
}
