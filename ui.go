package main

import (
	"image"
	"io/fs"
	"log"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/disintegration/imaging"
)

func activate(app *gtk.Application) {
	Window = gtk.NewApplicationWindow(app)
	Window.SetTitle("linux-wallpaperengine Helper")

	FlowBox = gtk.NewFlowBox()
	FlowBox.SetSelectionMode(gtk.SelectionSingle)
	FlowBox.SetHomogeneous(true)
	FlowBox.SetColumnSpacing(12)
	FlowBox.SetRowSpacing(12)
	FlowBox.SetMaxChildrenPerLine(8)
	FlowBox.SetHAlign(gtk.AlignCenter)
	FlowBox.SetVAlign(gtk.AlignStart)
	FlowBox.SetMarginTop(10)
	FlowBox.SetMarginBottom(10)
	FlowBox.SetMarginStart(10)
	FlowBox.SetMarginEnd(10)

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
	scrolledWindow.SetChild(FlowBox)

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

	Window.SetChild(vBox)
	Window.SetDefaultSize(800, 600)
	Window.SetVisible(true)
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
		tempFilePath := path.Join(CacheDir, path.Base(path.Dir(imagePath)))
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
	FlowBox.RemoveAll()
	// remove all previous child-activated signal handlers
	// else we would have multiple handlers for the same signal
	// causing multiple wallpapers to be applied on click
	for _, id := range ImageClickSignalHandlers {
		FlowBox.HandlerDisconnect(id)
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
		Window.SetChild(errorLabel) // Set the error label as the sole child
		Window.SetVisible(true)
		return // Exit activation function
	}

	signalHandler := FlowBox.Connect("child-activated", func(box *gtk.FlowBox, child *gtk.FlowBoxChild) {
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
	ImageClickSignalHandlers = append(ImageClickSignalHandlers, signalHandler)

	if len(wallpapers) == 0 {
		// Display error message if no wallpapers are found
		errorLabel := gtk.NewLabel("No wallpapers found in the directory: " + wallpaperDir)
		errorLabel.SetHExpand(true)
		errorLabel.SetVExpand(true)
		errorLabel.SetHAlign(gtk.AlignCenter)
		errorLabel.SetVAlign(gtk.AlignCenter)
		Window.SetChild(errorLabel) // Set the error label as the sole child
		Window.SetVisible(true)
		return // Exit activation function
	}

	// sort by date modified, newest first
	sort.SliceStable(wallpapers, func(i, j int) bool {
		iInfo, iErr := wallpapers[i].Info()
		jInfo, jErr := wallpapers[j].Info()
		if iErr != nil || jErr != nil {
			log.Printf("Error getting info for wallpaper %s or %s: %v, %v", wallpapers[i].Name(), wallpapers[j].Name(), iErr, jErr)
			return false // keep original order if there's an error
		}
		return iInfo.ModTime().After(jInfo.ModTime())
	})

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

			FlowBox.Append(iconOverlay)
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
		_, err := runDetachedProcess("xdg-open", fullPath)
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
