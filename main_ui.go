package main

import (
	"encoding/json"
	"image"
	"log"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/core/glib"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/disintegration/imaging"
)

type WallpaperItem struct {
	WallpaperID   string
	WallpaperPath string
	CachedPath    string
	IsFavorite    bool
	IsBroken      bool
	ModTime       time.Time

	// get from project.json
	projectJson   ProjectJSON
}

type ProjectJSON struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Tags          []string `json:"tags"`
	PreviewImage  string   `json:"preview"`
}

var WallpaperItems []WallpaperItem
var ImageClickSignalHandlers []glib.SignalHandle
var FlowBox *gtk.FlowBox
var Window *gtk.ApplicationWindow

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

	topControlBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	topControlBar.SetHAlign(gtk.AlignCenter)
	topControlBar.SetVAlign(gtk.AlignStart)
	topControlBar.SetMarginTop(10)
	topControlBar.SetMarginBottom(10)
	topControlBar.SetMarginStart(10)
	topControlBar.SetMarginEnd(10)
	
	refreshButton := gtk.NewButtonWithLabel("Refresh")
	refreshButton.SetHAlign(gtk.AlignStart)
	refreshButton.SetVAlign(gtk.AlignCenter)
	refreshButton.Connect("clicked", func() {
		log.Println("Refreshing wallpapers...")
		refreshWallpaperItems()
	})
	topControlBar.Append(refreshButton)

	searchBar := gtk.NewSearchBar()
	searchBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	searchEntry := gtk.NewSearchEntry()
	searchEntry.SetPlaceholderText("Search wallpapers...")
	searchEntry.SetHAlign(gtk.AlignCenter)
	searchEntry.SetVAlign(gtk.AlignCenter)
	searchEntry.Connect("search-changed", func(entry *gtk.SearchEntry) {
		query := strings.ToLower(entry.Text())
		log.Printf("Search query: %s", query)
	})
	searchBox.Append(searchEntry)
	searchBar.SetChild(searchBox)
	searchBar.ConnectEntry(searchEntry)
	searchBar.SetSearchMode(true)
	topControlBar.Append(searchBar)

	sortByModel := gtk.NewStringList([]string{"Date (desc)", "Date (asc)", "Name (asc)", "Name (desc)"})
	sortByDropdown := gtk.NewDropDown(sortByModel, nil)
	sortByDropdown.SetHAlign(gtk.AlignStart)
	sortByDropdown.SetVAlign(gtk.AlignCenter)
	topControlBar.Append(sortByDropdown)

	randomButton := gtk.NewButtonWithLabel("Random")
	randomButton.SetHAlign(gtk.AlignStart)
	randomButton.SetVAlign(gtk.AlignCenter)
	// randomButton.Connect("clicked", func() {
	// 	applyRandomWallpaper()
	// })
	topControlBar.Append(randomButton)

	optionsButton := gtk.NewButtonWithLabel("Options")
	optionsButton.SetHAlign(gtk.AlignEnd)
	optionsButton.SetVAlign(gtk.AlignCenter)
	optionsButton.Connect("clicked", func() {
		log.Println("Opening options dialog...")
		showOptionsDialog()
	})
	topControlBar.Append(optionsButton)

	exitButton := gtk.NewButtonWithLabel("Exit")
	exitButton.SetHAlign(gtk.AlignEnd)
	exitButton.SetVAlign(gtk.AlignCenter)
	exitButton.Connect("clicked", func() {
		log.Println("Exiting application...")
		Window.Close()
	})
	topControlBar.Append(exitButton)
	
	bottomControlBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	bottomControlBar.SetHAlign(gtk.AlignCenter)
	bottomControlBar.SetVAlign(gtk.AlignEnd)
	bottomControlBar.SetMarginTop(10)
	bottomControlBar.SetMarginBottom(10)
	bottomControlBar.SetMarginStart(10)
	bottomControlBar.SetMarginEnd(10)

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
	bottomControlBar.Append(volumeContainer)

	refreshWallpaperItems()

	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	scrolledWindow.SetMinContentHeight(600)
	scrolledWindow.SetMinContentWidth(800)
	scrolledWindow.SetVExpand(true)
	scrolledWindow.SetChild(FlowBox)

	topText := gtk.NewLabel("Select a wallpaper to apply it.")
	topText.SetHAlign(gtk.AlignCenter)
	topText.SetVAlign(gtk.AlignStart)
	topText.SetMarginTop(4)
	topText.SetMarginBottom(4)
	topText.SetMarginStart(4)
	topText.SetMarginEnd(4)

	vBox := gtk.NewBox(gtk.OrientationVertical, 0)
	vBox.Append(topControlBar)
	vBox.Append(topText)
	vBox.Append(scrolledWindow)
	vBox.Append(bottomControlBar)

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

func sortWallpaperItems() {
	// sort by date modified, newest first
	sort.SliceStable(WallpaperItems, func(i, j int) bool {
		iModTime := WallpaperItems[i].ModTime
		jModTime := WallpaperItems[j].ModTime
		if iModTime.IsZero() || jModTime.IsZero() {
			log.Printf("Error getting last modified info for wallpaper %s or %s", WallpaperItems[i].WallpaperID, WallpaperItems[j].WallpaperID)
			return false // keep original order if there's an error
		}
		return iModTime.After(jModTime)
	})

	// put favorites first
	sort.SliceStable(WallpaperItems, func(i, j int) bool {
		if WallpaperItems[i].IsFavorite && !WallpaperItems[j].IsFavorite {
			return true 
		} else {
			return false
		}
	})

	if Config.SavedUIState.HideBroken {
		WallpaperItems = slices.DeleteFunc(WallpaperItems, func(item WallpaperItem) bool {
			return item.IsBroken
		})
	} else {
		// put broken wallpapers at the end
		sort.SliceStable(WallpaperItems, func(i, j int) bool {
			if WallpaperItems[i].IsBroken && !WallpaperItems[j].IsBroken {
				return false
			} else if !WallpaperItems[i].IsBroken && WallpaperItems[j].IsBroken {
				return true
			} else {
				return false
			}
		})
	}
}

func refreshWallpaperItems() {
	FlowBox.RemoveAll()
	WallpaperItems = []WallpaperItem{}

	// remove all previous child-activated signal handlers
	// else we would have multiple handlers for the same signal
	// causing multiple wallpapers to be applied on click
	for _, id := range ImageClickSignalHandlers {
		FlowBox.HandlerDisconnect(id)
	}

	// get all wallpapers from the wallpaperengine directory
	wallpaperDir, err := ensureDir(Config.Constants.WallpaperEngineDir)
	if err != nil {
		log.Printf("Error ensuring wallpaper directory: %v", err)
		// Display error message if directory cannot be ensured
		errorLabel := gtk.NewLabel("Error ensuring wallpaper directory: " + err.Error())
		errorLabel.SetHExpand(true)
		errorLabel.SetVExpand(true)
		errorLabel.SetHAlign(gtk.AlignCenter)
		errorLabel.SetVAlign(gtk.AlignCenter)
		Window.SetChild(errorLabel) // Set the error label as the sole child
		Window.SetVisible(true)
		return // Exit activation function
	}

	wallpaperFolders, err := os.ReadDir(wallpaperDir)
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
		// Retrieve the wallpaper ID from the tooltip text of the activated image
		wallpaperId := child.Child().(*gtk.Overlay).Child().(*gtk.Image).Name()
		if wallpaperId == "" {
			log.Println("[WARN] No wallpaper ID found for the activated child.")
			return
		}
		log.Println("Applying wallpaper:", wallpaperId)
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperId) 
		applyWallpaper(fullWallpaperPath, float64(Config.SavedUIState.Volume))
	})
	ImageClickSignalHandlers = append(ImageClickSignalHandlers, signalHandler)

	if len(wallpaperFolders) == 0 {
		// Display error message if no wallpapers are found
		errorLabel := gtk.NewLabel("No wallpapers found in the directory: " + wallpaperDir)
		errorLabel.SetHExpand(true)
		errorLabel.SetVExpand(true)
		errorLabel.SetHAlign(gtk.AlignCenter)
		errorLabel.SetVAlign(gtk.AlignCenter)
		Window.SetChild(errorLabel)
		Window.SetVisible(true)
		return
	}

	for _, wallpaperFolder := range wallpaperFolders {
		if !wallpaperFolder.IsDir() {
			log.Printf("Skipping non-directory entry: %s", wallpaperFolder.Name())
			continue
		}

		wallpaperPath := path.Join(wallpaperDir, wallpaperFolder.Name())

		projectJsonFilePath := path.Join(wallpaperPath, "project.json")
		projectJson := ProjectJSON{}
		data, err := os.ReadFile(projectJsonFilePath)
		if err != nil {
			log.Printf("Error reading project.json for wallpaper %s: %v", wallpaperFolder.Name(), err)
			continue
		}

		err = json.Unmarshal(data, &projectJson)
		if err != nil {
			log.Printf("Error reading project.json for wallpaper %s: %v", wallpaperFolder.Name(), err)
			projectJson = ProjectJSON{
				Title:        wallpaperFolder.Name(),
				Description:  "No description available",
				Tags:         []string{},
				PreviewImage: "",
			}
		}

		var cachedImagePath string
		if projectJson.PreviewImage == "" {
			cachedImagePath = ""
		} else {
			cachedImagePath = path.Join(CacheDir, wallpaperFolder.Name(), projectJson.PreviewImage)
		}

		var modTime time.Time
		info, err := wallpaperFolder.Info()
		if err != nil {
			log.Printf("Error getting info for wallpaper %s: %v", wallpaperFolder.Name(), err)
			modTime = time.Time{} // default to zero value if we cannot get the mod time
		} else {
			modTime = info.ModTime()
		}

		WallpaperItems = append(WallpaperItems, WallpaperItem{
			projectJson: projectJson,
			WallpaperID: wallpaperFolder.Name(),
			WallpaperPath: wallpaperPath,
			CachedPath: cachedImagePath,
			IsFavorite: slices.Contains(Config.SavedUIState.Favorites, wallpaperFolder.Name()),
			IsBroken: slices.Contains(Config.SavedUIState.Broken, wallpaperFolder.Name()),
			ModTime:   modTime,
		})
	}

	sortWallpaperItems()

	for _, wallpaperItem := range WallpaperItems {
		if wallpaperItem.CachedPath != "" {
			iconOverlay := gtk.NewOverlay() // in case we want to add an icon overlay later
			imageWidget := gtk.NewImageFromIconName("image-x-generic-symbolic")
			imageWidget.SetPixelSize(128)
			imageWidget.SetHAlign(gtk.AlignCenter)
			imageWidget.SetVAlign(gtk.AlignCenter)
			imageWidget.SetMarginTop(10)
			imageWidget.SetMarginBottom(10)
			imageWidget.SetMarginStart(10)
			imageWidget.SetMarginEnd(10)
			imageWidget.SetName(wallpaperItem.WallpaperID)
			imageWidget.SetTooltipText(wallpaperItem.projectJson.Title + "\n" + wallpaperItem.projectJson.Description + "\n" + strings.Join(wallpaperItem.projectJson.Tags, ", ") + "\n" + wallpaperItem.WallpaperID) // set the wallpaper ID as tooltip text

			if wallpaperItem.IsFavorite {
				// if the wallpaper is a favorite, add a heart icon to the top right of the image
				favoriteIcon := gtk.NewImageFromIconName("emote-love-symbolic")
				favoriteIcon.SetPixelSize(16)
				favoriteIcon.SetHAlign(gtk.AlignEnd)
				favoriteIcon.SetVAlign(gtk.AlignStart)
				iconOverlay.AddOverlay(favoriteIcon)
			}
			if wallpaperItem.IsBroken {
				// if the wallpaper is marked as broken, add a warning icon to the top right of the image
				warningIcon := gtk.NewImageFromIconName("dialog-warning-symbolic")
				warningIcon.SetPixelSize(16)
				warningIcon.SetHAlign(gtk.AlignEnd)
				warningIcon.SetVAlign(gtk.AlignStart)
				iconOverlay.AddOverlay(warningIcon)
			}

			iconOverlay.SetChild(imageWidget)

			attachContextMenu(iconOverlay, &wallpaperItem, wallpaperItem.IsFavorite, wallpaperItem.IsBroken)

			FlowBox.Append(iconOverlay)
			loadImageAsync(wallpaperItem.CachedPath, imageWidget, 128)
		}
	}
}

func attachContextMenu(imageWidget *gtk.Overlay, wallpaperItem *WallpaperItem, isFavorite bool, isBroken bool) {
	actionGroup := gio.NewSimpleActionGroup()

	// apply action
	applyAction := gio.NewSimpleAction("apply", nil)
	applyAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Println("Applying wallpaper:", wallpaperItem.WallpaperID)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperItem.WallpaperID)
		applyWallpaper(fullWallpaperPath, float64(Config.SavedUIState.Volume))
	})
	actionGroup.AddAction(&applyAction.Action)

	// toggle_favorite action
	toggleFavoriteAction := gio.NewSimpleAction("toggle_favorite", nil)
	toggleFavoriteAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		if isFavorite {
			log.Printf("Removing %s from favorites", wallpaperItem.WallpaperID)
			wallpaperItem.IsFavorite = false
			Config.SavedUIState.Favorites = slices.Delete(Config.SavedUIState.Favorites, slices.Index(Config.SavedUIState.Favorites, wallpaperItem.WallpaperID), slices.Index(Config.SavedUIState.Favorites, wallpaperItem.WallpaperID)+1)
		} else {
			log.Printf("Favoriting %s", wallpaperItem.WallpaperID)
			wallpaperItem.IsFavorite = true
			Config.SavedUIState.Favorites = append(Config.SavedUIState.Favorites, wallpaperItem.WallpaperID)
		}

		refreshWallpaperItems()
	})
	actionGroup.AddAction(&toggleFavoriteAction.Action)

	// toggle_broken action
	toggleBrokenAction := gio.NewSimpleAction("toggle_broken", nil)
	toggleBrokenAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		if isBroken {
			log.Printf("Marking %s as not broken", wallpaperItem.WallpaperID)
			wallpaperItem.IsBroken = false
			Config.SavedUIState.Broken = slices.Delete(Config.SavedUIState.Broken, slices.Index(Config.SavedUIState.Broken, wallpaperItem.WallpaperID), slices.Index(Config.SavedUIState.Broken, wallpaperItem.WallpaperID)+1)
		} else {
			log.Printf("Marking %s as broken", wallpaperItem.WallpaperID)
			wallpaperItem.IsBroken = true
			Config.SavedUIState.Broken = append(Config.SavedUIState.Broken, wallpaperItem.WallpaperID)
		}

		refreshWallpaperItems()
	})
	actionGroup.AddAction(&toggleBrokenAction.Action)

	// open_directory action
	openDirectoryAction := gio.NewSimpleAction("open_directory", nil)
	openDirectoryAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Printf("Opening directory for %s\n", wallpaperItem)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullPath := path.Join(wallpaperDir, wallpaperItem.WallpaperID)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			log.Printf("Wallpaper directory does not exist: %s", fullPath)
			return
		}
		_, err := runDetachedProcess("xdg-open", fullPath)
		if err != nil {
			log.Printf("Error opening directory %s: %v", fullPath, err)
			return
		}
		log.Printf("Opened directory for wallpaper %s: %s", wallpaperItem, fullPath)
	})
	actionGroup.AddAction(&openDirectoryAction.Action)

	// copy_command action
	copyCommandAction := gio.NewSimpleAction("copy_command", nil)
	copyCommandAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Printf("Copying command for %s to clipboard", wallpaperItem)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperItem.WallpaperID)
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

	imageWidget.InsertActionGroup(wallpaperItem.WallpaperID, actionGroup)

	gesture := gtk.NewGestureClick()
	gesture.SetButton(3)
	imageWidget.AddController(gesture)
	gesture.ConnectReleased(func(nPress int, x, y float64) {
		if nPress == 1 { // ensure single click
			// context menu for the image
			contextMenuModel := gio.NewMenu()

			contextMenuModel.Append("Apply Wallpaper", wallpaperItem.WallpaperID + ".apply")
			if isFavorite {
				contextMenuModel.Append("Remove from Favorites", wallpaperItem.WallpaperID + ".toggle_favorite")
			} else {
				contextMenuModel.Append("Add to Favorites", wallpaperItem.WallpaperID + ".toggle_favorite")
			}
			if isBroken {
				contextMenuModel.Append("Unmark as Broken", wallpaperItem.WallpaperID + ".toggle_broken")
			} else {
				contextMenuModel.Append("Mark as Broken", wallpaperItem.WallpaperID + ".toggle_broken")
			}
			contextMenuModel.Append("Open Wallpaper Directory", wallpaperItem.WallpaperID + ".open_directory")
			contextMenuModel.Append("Copy Command to Clipboard", wallpaperItem.WallpaperID + ".copy_command")

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

	log.Println("Context menu attached to image widget for wallpaper:", wallpaperItem.WallpaperID)
}
