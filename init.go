package main

import (
	"fmt"
	"image"
	_ "image/gif"  // For gif decoder
	_ "image/jpeg" // For jpeg decoder
	_ "image/png"  // For png decoder
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
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
	LinuxWallpaperEngineBin  string `toml:"linux_wallpaperengine_bin"`
	WallpaperEngineDir       string `toml:"wallpaper_engine_dir"`
	LastSetId 							 string `toml:"last_set_id"`
}

type PostProcessingStruct struct {
	Enabled        bool   `toml:"enabled"`
	ScreenshotFile string `toml:"screenshot_file"`
	PostCommand    string `toml:"post_command"`
}

type SavedUIStateStruct struct {
	Volume int64 `toml:"volume"`
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
			Volume: 100, // Default volume set to 100%
		},
	}
}

var Config *ConfigStruct
var cacheDir string

func main() {
	app := gtk.NewApplication("com.github.diamondburned.gotk4-examples.gtk4.simple", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() { activate(app) })

	// Ensure config directory exists
	configDir, err := ensureConfigDir()
	if err != nil {
		log.Fatalf("Failed to ensure config directory: %v", err)
		return
	}
	log.Printf("Config directory ensured at: %s", configDir)

	// Ensure cache directory exists
	cacheDir, err = ensureCacheDir()
	if err != nil {
		log.Fatalf("Failed to ensure cache directory: %v", err)
		return
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

		fmt.Println(Config)

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
		log.Printf("Restoring wallpaper from path: %s", wallpaperPath)
		success := applyWallpaper(wallpaperPath, float64(Config.SavedUIState.Volume)) // Convert volume to a percentage
	  if !success {
			log.Println("Failed to restore wallpaper.")
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
			return []string{}, nil // Process is not running
		}
		return []string{}, err // Some other error occurred
	}
	// If output is not empty, return the pids
	if strings.TrimSpace(string(output)) != "" {
		pids := strings.Fields(string(output))
		log.Printf("Process '%s' is running with PIDs: %v", processName, pids)
		return pids, nil // Process is running
	}
	return []string{}, nil // Process is not running
}

func applyWallpaper(wallpaperPath string, volume float64) bool {
	cmd := Config.Constants.LinuxWallpaperEngineBin + " --screen-root HDMI-A-1 --bg " + wallpaperPath

	if volume <= 1 {
		cmd += " --silent"
	} else {
		cmd += " --volume " + strconv.FormatFloat(volume, 'f', 0, 64)
	}

	cacheScreenshot := ""
	if Config.PostProcessing.Enabled {
		cacheScreenshot = Config.PostProcessing.ScreenshotFile // ~/.cache/linux-wallpaperengine-helper/screenshot.png

		cmd += " --screenshot " + cacheScreenshot
		log.Printf("Saving screenshot to: %s", cacheScreenshot)
	}

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
		defer devNull.Close() // Ensure /dev/null is closed after the command starts
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
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("Error opening image file %s: %v", imagePath, err)
		return
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Error decoding image %s: %v", imagePath, err)
		return
	}

	// Resize the image to the desired thumbnail size
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
		tempFilePath := path.Join(cacheDir, path.Base(path.Dir(imagePath)))
		cachedThumbnailPath := path.Join(tempFilePath, "thumbnail.png")

		if _, err := os.Stat(cachedThumbnailPath); os.IsNotExist(err) {
			// If the cached thumbnail does not exist, create it
			log.Printf("Cached thumbnail not found for %s, creating it...", imagePath)
			cacheImage(imagePath, cachedThumbnailPath)
		} else {
			log.Printf("Using cached thumbnail for %s", imagePath)
		}

		// Convert the image.Image to a GdkPixbuf (GoTk4's image format for display)
		// This conversion can be resource intensive for very large images,
		// so doing it after resizing is crucial.
		pixbuf,err := gdkpixbuf.NewPixbufFromFile(cachedThumbnailPath)
		if err != nil {
			log.Printf("Error creating GdkPixbuf from file %s: %v", cachedThumbnailPath, err)
			return
		}

		paintable := gdk.NewTextureForPixbuf(pixbuf)

		// Update the gtk.Image widget on the main GTK thread
		glib.IdleAdd(func() {
			targetImage.SetFromPaintable(paintable)
			targetImage.SetPixelSize(pixelSize) // Set the pixel size for the image
			targetImage.SetHAlign(gtk.AlignCenter)
			targetImage.SetVAlign(gtk.AlignCenter)
			targetImage.SetMarginTop(10)
			targetImage.SetMarginBottom(10)
			targetImage.SetMarginStart(10)
			targetImage.SetMarginEnd(10)
			targetImage.SetVisible(true) // Make sure the image is visible
		})
	}()
}

func activate(app *gtk.Application) {
	window := gtk.NewApplicationWindow(app)
	window.SetTitle("linux-wallpaperengine Helper")

	// make the bottom controls 
	controlBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	controlBar.SetHAlign(gtk.AlignCenter)
	controlBar.SetVAlign(gtk.AlignEnd)
	controlBar.SetMarginTop(10)
	controlBar.SetMarginBottom(10)
	controlBar.SetMarginStart(10)
	controlBar.SetMarginEnd(10)

	// volume control
	volumeContainer := gtk.NewBox(gtk.OrientationVertical, 0)
	volumeContainer.SetHExpand(true)
	volumeContainer.SetHAlign(gtk.AlignStart)
	volumeContainer.SetVAlign(gtk.AlignCenter)
	volumeLabel := gtk.NewLabel("Volume: " + strconv.FormatInt(Config.SavedUIState.Volume, 10) + "%")
	volumeLabel.SetHAlign(gtk.AlignCenter)
	volumeLabel.SetVAlign(gtk.AlignCenter)
	volumeSlider := gtk.NewScaleWithRange(gtk.OrientationHorizontal, 0, 100, 1)
	volumeSlider.SetValue(float64(Config.SavedUIState.Volume)) // Default volume to 100%
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

	flowBox := gtk.NewFlowBox()
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

	flowBox.Connect("child-activated", func(box *gtk.FlowBox, child *gtk.FlowBoxChild) {
		if child == nil {
			return // No child selected (though this is unlikely for 'child-activated')
		}
		// Retrieve the wallpaper name from the tooltip text of the activated image
		wallpaperName := child.Child().(*gtk.Image).TooltipText()
		if wallpaperName == "" {
			log.Println("[WARN] No wallpaper name found for the activated child.")
			return
		}
		log.Println("Applying wallpaper:", wallpaperName)
		// Here you would add the code to apply the wallpaper, e.g.:
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperName) // You might need to refine how you get the full path if wallpaperName is just the directory name
		applyWallpaper(fullWallpaperPath, volumeSlider.Value())
	})

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
					continue // Skip directories
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

			imageWidget := gtk.NewImageFromIconName("image-x-generic-symbolic")
			imageWidget.SetPixelSize(128) // Set a fixed size for the image
			imageWidget.SetHAlign(gtk.AlignCenter)
			imageWidget.SetVAlign(gtk.AlignCenter)
			imageWidget.SetMarginTop(10)
			imageWidget.SetMarginBottom(10)
			imageWidget.SetMarginStart(10)
			imageWidget.SetMarginEnd(10)
			imageWidget.SetTooltipText(wallpaper.Name())
			flowBox.Append(imageWidget)

			loadImageAsync(imageFile, imageWidget, 128)
		}
	}

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
