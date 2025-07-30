package main

import (
	"encoding/json"
	"errors"
	"fmt"
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

var MainWindow *gtk.ApplicationWindow = nil
var ScrolledWindow *gtk.ScrolledWindow = nil
var SearchQuery string = ""
var SelectedWallpaperItemId string = ""
var StatusText *gtk.Label = nil
var WallpaperPropertiesBox *gtk.FlowBox = nil
var WallpaperList *gtk.FlowBox = nil

// Main function to create the GTK application and set up the main window.
func activate(app *gtk.Application) {
	MainWindow = gtk.NewApplicationWindow(app)
	MainWindow.SetTitle("Linux Wallpaper Engine Helper")
	setupIconStyling()

	//ANCHOR - Top control bar
	// This will contain the controls relating to the list or app itself
	// Example: refresh, search, sort, etc.

	topControlBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	topControlBar.SetHAlign(gtk.AlignCenter)
	topControlBar.SetVAlign(gtk.AlignStart)
	topControlBar.SetMarginTop(10)
	topControlBar.SetMarginBottom(10)
	topControlBar.SetMarginStart(10)
	topControlBar.SetMarginEnd(10)
	topControlBar.SetSpacing(4)

	refreshButton := gtk.NewButtonWithLabel("Refresh")
	refreshButton.SetHAlign(gtk.AlignStart)
	refreshButton.SetVAlign(gtk.AlignCenter)
	refreshButton.Connect("clicked", func() {
		log.Println("Refreshing wallpapers...")
		refreshWallpaperDisplay()
	})
	topControlBar.Append(refreshButton)

	searchBar := gtk.NewSearchBar()
	searchBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	searchEntry := gtk.NewSearchEntry()
	searchEntry.SetPlaceholderText("Search wallpapers...")
	searchEntry.SetHAlign(gtk.AlignCenter)
	searchEntry.SetVAlign(gtk.AlignCenter)
	searchEntry.Connect("search-changed", func(entry *gtk.SearchEntry) {
		SearchQuery = strings.ToLower(entry.Text())
		log.Printf("Search query: %s", SearchQuery)
		filterWallpapersBySearch(SearchQuery)
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
	sortByDropdown.SetSelected(0) // default to "Date (desc)"
	sortByDropdown.Connect("notify::selected", func() {
		selectedIndex := sortByDropdown.Selected()
		switch selectedIndex {
		case 0:
			Config.SavedUIState.SortBy = "date_desc"
		case 1:
			Config.SavedUIState.SortBy = "date_asc"
		case 2:
			Config.SavedUIState.SortBy = "name_asc"
		case 3:
			Config.SavedUIState.SortBy = "name_desc"
		default:
			log.Printf("Unknown sort criteria index: %d, defaulting to date_desc", selectedIndex)
			Config.SavedUIState.SortBy = "date_desc"
		}
		refreshWallpaperDisplay()
	})
	topControlBar.Append(sortByDropdown)

	randomButton := gtk.NewButtonWithLabel("Random")
	randomButton.SetHAlign(gtk.AlignStart)
	randomButton.SetVAlign(gtk.AlignCenter)
	randomButton.Connect("clicked", func() {
		applyRandomWallpaper()
	})
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
		MainWindow.Close()
	})
	topControlBar.Append(exitButton)

	//ANCHOR - Status text
	// This will contain information as to what the app is doing at the moment
	// Set this via updateGUIStatusText("Your message here")

	StatusText = gtk.NewLabel("Double-click a wallpaper to apply it.")
	StatusText.SetHAlign(gtk.AlignCenter)
	StatusText.SetVAlign(gtk.AlignStart)
	StatusText.SetMarginTop(4)
	StatusText.SetMarginBottom(4)
	StatusText.SetMarginStart(4)
	StatusText.SetMarginEnd(4)

	//ANCHOR - Bottom control bar
	// This will contain controls related to settings applied to the application process
	// Example: wallpaper volume, swww fit type, etc.

	bottomControlBar := gtk.NewBox(gtk.OrientationHorizontal, 0)
	bottomControlBar.SetHAlign(gtk.AlignCenter)
	bottomControlBar.SetVAlign(gtk.AlignEnd)
	bottomControlBar.SetMarginTop(10)
	bottomControlBar.SetMarginBottom(10)
	bottomControlBar.SetMarginStart(10)
	bottomControlBar.SetMarginEnd(10)
	bottomControlBar.SetSpacing(4)

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

	//ANCHOR - Wallpaper list
	// This will contain the list of wallpapers

	WallpaperList = gtk.NewFlowBox()
	WallpaperList.SetSelectionMode(gtk.SelectionSingle)
	WallpaperList.SetHomogeneous(true)
	WallpaperList.SetColumnSpacing(12)
	WallpaperList.SetRowSpacing(12)
	WallpaperList.SetMaxChildrenPerLine(8)
	WallpaperList.SetHAlign(gtk.AlignCenter)
	WallpaperList.SetVAlign(gtk.AlignStart)
	WallpaperList.SetMarginTop(10)
	WallpaperList.SetMarginBottom(10)
	WallpaperList.SetMarginStart(10)
	WallpaperList.SetMarginEnd(10)

	FlowBoxEscKey := gtk.NewEventControllerKey()
	WallpaperList.AddController(FlowBoxEscKey)
	FlowBoxEscKey.Connect("key-pressed", func(controller *gtk.EventControllerKey, keyval uint, keycode uint, state gdk.ModifierType) {
		if keyval == gdk.KEY_Escape {
			WallpaperList.UnselectAll()
			SelectedWallpaperItemId = ""
			WallpaperPropertiesBox.RemoveAll()
			WallpaperPropertiesBox.SetVisible(false)
		}
	})

	WallpaperPropertiesBox = gtk.NewFlowBox()
	WallpaperPropertiesBox.SetSizeRequest(400, 128)
	WallpaperPropertiesBox.SetHExpand(false)
	WallpaperPropertiesBox.SetVExpand(false)
	WallpaperPropertiesBox.SetSelectionMode(gtk.SelectionNone)
	WallpaperPropertiesBox.SetHomogeneous(false)
	WallpaperPropertiesBox.SetColumnSpacing(12)
	WallpaperPropertiesBox.SetRowSpacing(12)
	// PropertiesBox.SetMaxChildrenPerLine(2)
	WallpaperPropertiesBox.SetHAlign(gtk.AlignCenter)
	WallpaperPropertiesBox.SetVAlign(gtk.AlignStart)
	WallpaperPropertiesBox.SetVisible(false)
	WallpaperPropertiesBox.SetMarginTop(4)
	WallpaperPropertiesBox.SetMarginBottom(4)

	ScrolledWindow = gtk.NewScrolledWindow()
	ScrolledWindow.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)
	ScrolledWindow.SetMinContentHeight(600)
	ScrolledWindow.SetMinContentWidth(800)
	ScrolledWindow.SetHExpand(true)
	ScrolledWindow.SetVExpand(true)
	ScrolledWindow.SetHAlign(gtk.AlignFill)
	ScrolledWindow.SetChild(WallpaperList)

	if err := reloadWallpaperData(); err != nil {
		log.Printf("Error reloading wallpaper data: %v", err)
		showFrontError(err.Error())
	} else {
		refreshWallpaperDisplay()
	}

	vBox := gtk.NewBox(gtk.OrientationVertical, 0)
	vBox.Append(topControlBar)
	vBox.Append(StatusText)
	vBox.Append(ScrolledWindow)
	vBox.Append(WallpaperPropertiesBox)
	vBox.Append(bottomControlBar)

	MainWindow.SetChild(vBox)
	MainWindow.SetDefaultSize(800, 600)
	MainWindow.SetVisible(true)
}

// Helper function to provide custom CSS to the entire application.
func setupIconStyling() {
	cssProvider := gtk.NewCSSProvider()
	css := `
		.favorite-icon {
			color: #d1b100ff;
		}
		
		.error {
			color: #ab0000ff;
		}
		`

	cssProvider.LoadFromString(css)
	gtk.StyleContextAddProviderForDisplay(
		gdk.DisplayGetDefault(),
		cssProvider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)
}

// Helper function to replace the WallpaperList with an error message.
func showFrontError(message string) {
	WallpaperList.RemoveAll()
	WallpaperList.SetVisible(false)
	WallpaperPropertiesBox.SetVisible(false)

	errorLabel := gtk.NewLabel(message)
	errorLabel.SetHExpand(true)
	errorLabel.SetVExpand(true)
	errorLabel.SetHAlign(gtk.AlignCenter)
	errorLabel.SetVAlign(gtk.AlignCenter)

	ScrolledWindow.SetChild(errorLabel)
	updateGUIStatusText("")
}

// Helper function to hide the error message and restore the WallpaperList.
func hideFrontError() {
	// Remove the error message and set the WallpaperList as the child of ScrolledWindow
	ScrolledWindow.SetChild(WallpaperList)
	WallpaperList.SetVisible(true)
	updateGUIStatusText("Double-click a wallpaper to apply it.")

	if SelectedWallpaperItemId != "" {
		reselectItem(false)
	}
}

// Updates the GUI status text (top label in the main view) with the given message.
func updateGUIStatusText(message string) {
	if StatusText != nil {
		glib.IdleAdd(func() {
			StatusText.SetText(message)
		})
	}
}

// Reselects the previously selected wallpaper item in the WallpaperList.
//
// If `unselect` is true, it will force unselect all items if it is not found, or if SelectedWallpaperItemId is empty.
func reselectItem(unselect bool) {
	log.Printf("Reselecting item with ID: %s", SelectedWallpaperItemId)
	if SelectedWallpaperItemId != "" {
		child := WallpaperList.FirstChild()
		found := false
		for child != nil {
			if flowBoxChild, ok := child.(*gtk.FlowBoxChild); ok {
				if flowBoxChild.Child().(*gtk.Overlay).Child().(*gtk.Image).Name() == SelectedWallpaperItemId {
					WallpaperList.SelectChild(flowBoxChild)
					child = nil
					found = true
				}
				child = flowBoxChild.NextSibling()
			} else {
				log.Printf("Unexpected child type: %T", child)
				child = nil
			}
		}

		if !found && unselect {
			WallpaperList.UnselectAll()
			SelectedWallpaperItemId = ""
			WallpaperPropertiesBox.RemoveAll()
			WallpaperPropertiesBox.SetVisible(false)
		}
	} else {
		if unselect {
			WallpaperList.UnselectAll()
			SelectedWallpaperItemId = ""
			WallpaperPropertiesBox.RemoveAll()
			WallpaperPropertiesBox.SetVisible(false)
		}
	}
}

// Displays the details of a selected wallpaper item in the WallpaperPropertiesBox.
func showDetails(wallpaperItem *WallpaperItem) {
	WallpaperPropertiesBox.RemoveAll()

	WallpaperPropertiesBox.SetSizeRequest(-1, 128)
	WallpaperPropertiesBox.SetHExpand(true)
	WallpaperPropertiesBox.SetVExpand(false)
	WallpaperPropertiesBox.SetMarginTop(4)
	WallpaperPropertiesBox.SetMarginBottom(4)

	thumbnail := gtk.NewImage()
	thumbnail.SetHAlign(gtk.AlignCenter)
	thumbnail.SetVAlign(gtk.AlignCenter)
	thumbnail.SetSizeRequest(128, 128) // Fixed thumbnail size
	thumbnail.SetHExpand(false)
	thumbnail.SetVExpand(false)
	loadImageAsync(wallpaperItem.CachedPath, thumbnail, 128)

	labelsBox := gtk.NewBox(gtk.OrientationVertical, 0)
	labelsBox.SetHAlign(gtk.AlignStart)
	labelsBox.SetVAlign(gtk.AlignStart)
	labelsBox.SetHExpand(true)
	labelsBox.SetVExpand(true)

	if wallpaperItem.projectJson.Title != "" {
		titleLabel := gtk.NewLabel(wallpaperItem.projectJson.Title)
		titleLabel.SetMarkup("<span weight=\"bold\" size=\"large\">" + escapeMarkup(titleLabel.Text()) + "</span>")
		titleLabel.SetHAlign(gtk.AlignStart)
		titleLabel.SetVAlign(gtk.AlignStart)
		titleLabel.SetMarginBottom(4)
		titleLabel.SetMarginTop(4)
		titleLabel.SetSelectable(true)
		labelsBox.Append(titleLabel)
	}
	if len(wallpaperItem.projectJson.Tags) > 0 {
		tagsLabel := gtk.NewLabel(strings.Join(wallpaperItem.projectJson.Tags, ", "))
		tagsLabel.SetSelectable(true)
		tagsLabel.SetMarkup("<span size=\"small\"><i>" + escapeMarkup(tagsLabel.Text()) + "</i></span>")
		tagsLabel.SetHAlign(gtk.AlignStart)
		tagsLabel.SetVAlign(gtk.AlignStart)
		tagsLabel.SetMarginBottom(4)
		tagsLabel.SetMarginTop(4)
		labelsBox.Append(tagsLabel)
	}
	if wallpaperItem.projectJson.Description != "" {
		descriptionScrollable := gtk.NewScrolledWindow()
		descriptionScrollable.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
		descriptionScrollable.SetMaxContentHeight(60) // Limit height to trigger vertical scrolling
		descriptionLabel := gtk.NewLabel(wallpaperItem.projectJson.Description)
		descriptionLabel.SetHAlign(gtk.AlignStart)
		descriptionLabel.SetVAlign(gtk.AlignStart)
		descriptionLabel.SetMarginBottom(4)
		descriptionLabel.SetMarginTop(4)
		descriptionLabel.SetWrap(true)
		descriptionLabel.SetWrapMode(2) // PANGO_WRAP_WORD
		descriptionLabel.SetSelectable(true)
		descriptionScrollable.SetChild(descriptionLabel)
		labelsBox.Append(descriptionScrollable)
		descriptionScrollable.SetHExpand(true)
		descriptionScrollable.SetVExpand(true)
	}

	WallpaperPropertiesBox.Append(thumbnail)
	WallpaperPropertiesBox.Append(labelsBox)
	WallpaperPropertiesBox.SetVisible(true)
	log.Printf("Showing details for wallpaper: %s", wallpaperItem.WallpaperID)
}

// Saves a 128x128 preview image of the first path given, to the location of the second path.
// Used to speed up the load times of the WallpaperItems
func cacheImage(imagePath string, cachedThumbnailPath string, pixelSize int) {
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

	// Resize the image to the desired thumbnail size (pixelSize by pixelSize pixels)
	// this is to have a uniform look for all images
	thumbnail := imaging.Fit(img, pixelSize, pixelSize, imaging.Lanczos)

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

// Runs a goroutine to ensure a cached image, and sets the gtk.Image source to it.
//
// First checks the cache directory to see if it's already been cached. If it has then it just loads that one.
// If it does not find any it, it will create one via cacheImage() to the provided pixelSize by pixelSize.
func loadImageAsync(imagePath string, targetImage *gtk.Image, pixelSize int) {
	go func() {
		// check for cached thumbnail first
		// TODO: add support for gifs
		tempFilePath := path.Join(CacheDir, path.Base(path.Dir(imagePath)))
		cachedThumbnailPath := path.Join(tempFilePath, "thumbnail.png") // ~/.cache/linux-wallpaperengine-helper/<wallpaper_id>/thumbnail.png

		if _, err := os.Stat(cachedThumbnailPath); os.IsNotExist(err) {
			log.Printf("Cached thumbnail not found for %s, creating it...", imagePath)
			cacheImage(imagePath, cachedThumbnailPath, pixelSize)
		} else {
			log.Printf("Using cached thumbnail for %s", imagePath)
		}

		// convert to pixbuf
		pixbuf, err := gdkpixbuf.NewPixbufFromFile(cachedThumbnailPath)
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

// Helper function to sort the WallpaperItems by Modification Time
func sortByModTime(descending bool) {
	sort.SliceStable(WallpaperItems, func(i, j int) bool {
		iModTime := WallpaperItems[i].ModTime
		jModTime := WallpaperItems[j].ModTime
		if iModTime.IsZero() || jModTime.IsZero() {
			log.Printf("Error getting last modified info for wallpaper %s or %s", WallpaperItems[i].WallpaperID, WallpaperItems[j].WallpaperID)
			return false // keep original order if there's an error
		}
		if descending {
			return iModTime.After(jModTime)
		} else {
			return iModTime.Before(jModTime)
		}
	})
}

// Helper function to sort the WallpaperItems by the title in it's project.json
//
// Falls back to WallpaperID if Title is not provided.
func sortByProjectTitle(descending bool) {
	sort.SliceStable(WallpaperItems, func(i, j int) bool {
		iName := WallpaperItems[i].projectJson.Title
		jName := WallpaperItems[j].projectJson.Title
		if iName == "" || jName == "" {
			log.Printf("Error getting name info for wallpaper %s or %s", WallpaperItems[i].WallpaperID, WallpaperItems[j].WallpaperID)
			return false // keep original order if there's an error
		}
		if descending {
			return iName > jName
		} else {
			return iName < jName
		}
	})
}

// Helper function to sort all the items, respecting the config, favorites, and broken.
//
// First sorts all the Wallpapers by the SortBy selection from the Config.
// Then it sorts them by putting all the Favorites first
// Finally, it sorts them by putting all the Broken ones last.
func sortWallpaperItems() {
	// sort by date modified, newest first
	switch Config.SavedUIState.SortBy {
	case "date_desc":
		sortByModTime(true)
	case "date_asc":
		sortByModTime(false)
	case "name_desc":
		sortByProjectTitle(true)
	case "name_asc":
		sortByProjectTitle(false)
	default:
		log.Printf("Unknown sort criteria: %s, defaulting to date_desc", Config.SavedUIState.SortBy)
		sortByModTime(true)
	}

	// put favorites first
	sort.SliceStable(WallpaperItems, func(i, j int) bool {
		if WallpaperItems[i].IsFavorite && !WallpaperItems[j].IsFavorite {
			return true
		} else {
			return false
		}
	})

	// put broken wallpapers at the end
	// this is important to do at the end, since a wallpaper can be a favorite and broken
	// we want to make sure all the broken ones are at the bottom
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

// Forces a full refresh of the WallpaperItems.
//
// This reads the WallpaperEngineDir (contents directory) to repopulate the WallpaperItems.
//
// First it reads the directory and its subdirectories (depth of 1).
// Each subdirectory is considered a "wallpaper" and the name of the dir is its WallpaperID.
//
// Next it reads the project.json in the directory.
// It parses the JSON for the wallpaper's Title, Description, and Tags.
// If it fails reading the JSON, or it isn't present, the Title, Description, and Tags are all set to an empty string, "No description available", and empty string array respectively.
//
// Then it populates the rest of the WallpaperItem.
// It adds the ID, cache location for the preview image, checks if its a favorite/broken, and adds the Modification Time.
//
// Finally, it adds the WallpaperItem to the global WallpaperItems slice.
func reloadWallpaperData() error {
	WallpaperItems = []WallpaperItem{}

	wallpaperDir, err := ensureDir(Config.Constants.WallpaperEngineDir)
	if err != nil {
		return fmt.Errorf("failed to ensure wallpaper directory: %v", err)
	}

	wallpaperFolders, err := os.ReadDir(wallpaperDir)
	if err != nil {
		return fmt.Errorf("failed to read wallpaper directory: %v", err)
	}

	if len(wallpaperFolders) == 0 {
		return errors.New("no wallpapers found in the wallpaper directory")
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
			projectJson:   projectJson,
			WallpaperID:   wallpaperFolder.Name(),
			WallpaperPath: wallpaperPath,
			CachedPath:    cachedImagePath,
			IsFavorite:    slices.Contains(Config.SavedUIState.Favorites, wallpaperFolder.Name()),
			IsBroken:      slices.Contains(Config.SavedUIState.Broken, wallpaperFolder.Name()),
			ModTime:       modTime,
		})
	}

	return nil
}

// Refreshes only the wallpaper display.
// This reads the WallpaperItems currently set and updates the WallpaperList according to those.
//
// If there are no WallpaperItems, it will display an error on the main view saying so.
// Else, it makes sure that there are no error message displayed and add the WallpaperItems to the global WallpaperList FlexBox.
func refreshWallpaperDisplay() {
	if len(WallpaperItems) < 1 {
		// assume that an error occurred during wallpaper loading
		// let us try to reload the wallpaper data
		if err := reloadWallpaperData(); err != nil {
			log.Printf("Error reloading wallpaper data: %v", err)
			showFrontError(err.Error())
			return
		}
	}

	if _, ok := ScrolledWindow.Child().(*gtk.FlowBox); !ok {
		// we are assuming that an error message is being shown
		// this may happen if there were no wallpapers found in the wallpaper directory
		// but now since we got here, there should be
		// therefore, we should reset the error message
		hideFrontError()
	}

	WallpaperList.RemoveAll()

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
			imageWidget.SetTooltipText(wallpaperItem.projectJson.Title)
			statusIcons := gtk.NewFlowBox()
			statusIcons.SetSelectionMode(gtk.SelectionNone)
			statusIcons.SetHAlign(gtk.AlignEnd)
			statusIcons.SetVAlign(gtk.AlignStart)

			if wallpaperItem.IsFavorite {
				// if the wallpaper is a favorite, add a heart icon to the top right of the image
				favoriteIcon := gtk.NewImageFromIconName("starred-symbolic")
				favoriteIcon.SetPixelSize(24)
				favoriteIcon.AddCSSClass("favorite-icon")
				statusIcons.Append(favoriteIcon)
			}
			if wallpaperItem.IsBroken {
				// if the wallpaper is marked as broken, add a warning icon to the top right of the image
				warningIcon := gtk.NewImageFromIconName("dialog-warning-symbolic")
				warningIcon.SetPixelSize(24)
				warningIcon.AddCSSClass("error")
				statusIcons.Append(warningIcon)
			}

			if statusIcons.FirstChild() != nil {
				iconOverlay.AddOverlay(statusIcons)
			}
			iconOverlay.SetChild(imageWidget)

			attachGestures(iconOverlay, &wallpaperItem, wallpaperItem.IsFavorite, wallpaperItem.IsBroken)

			WallpaperList.Append(iconOverlay)
			loadImageAsync(wallpaperItem.CachedPath, imageWidget, 128)
		}
	}

	// filterWallpapers will already hide broken wallpapers if Config.SavedUIState.HideBroken is true
	// so we just filter by search query if it is set
	filterWallpapersBySearch(SearchQuery)

	reselectItem(true)
}

// Helper function to perform a search through the wallpaper's title, description, and tags.
func filterWallpapersBySearch(query string) {
	filterWallpapers(func(item WallpaperItem) bool {
		if query == "" {
			return true // show all wallpapers if query is empty
		}
		titleMatch := strings.Contains(strings.ToLower(item.projectJson.Title), strings.ToLower(query))
		descriptionMatch := strings.Contains(strings.ToLower(item.projectJson.Description), strings.ToLower(query))
		tagsMatch := false
		for _, tag := range item.projectJson.Tags {
			if strings.Contains(strings.ToLower(tag), strings.ToLower(query)) {
				tagsMatch = true
				break
			}
		}
		return titleMatch || descriptionMatch || tagsMatch
	})
}

// Runs the predicate function on every WallpaperItem to filter out those that do not match.
//
// If the provided function returns true, it will set that Item visible in the list.
// Else, it hides that item in the list.
//
// This also respects the Config.SavedUIState.HideBroken configuration, and will always hide broken ones regardless if true.
// Therefore, this can also act as a "hide all broken wallpapers" by providing a function that only returns true.
func filterWallpapers(predicate func(WallpaperItem) bool) {
	filtered := []WallpaperItem{}
	for _, item := range WallpaperItems {
		if predicate(item) {
			if Config.SavedUIState.HideBroken && item.IsBroken {
				continue
			} else {
				filtered = append(filtered, item)
			}
		}
	}

	child := WallpaperList.FirstChild()
	for child != nil {
		if flowBoxChild, ok := child.(*gtk.FlowBoxChild); ok {
			if slices.ContainsFunc(filtered, func(item WallpaperItem) bool {
				return item.WallpaperID == flowBoxChild.Child().(*gtk.Overlay).Child().(*gtk.Image).Name()
			}) {
				flowBoxChild.SetVisible(true)
			} else {
				flowBoxChild.SetVisible(false)
			}

			child = flowBoxChild.NextSibling()
			continue
		}
		log.Printf("Unexpected child type: %T", child)
		child = nil // break the loop if we encounter an unexpected type
		break
	}
}

// Attaches the left and right click gestures on the wallpapers shown.
//
// Left click = Shows details for the wallpaper in the PropertiesBox.
// Double left click = Applies the wallpaper.
// Right click = Shows options for the wallpaper, such as toggling favorite/broken, and more
func attachGestures(imageWidget *gtk.Overlay, wallpaperItem *WallpaperItem, isFavorite bool, isBroken bool) {
	actionGroup := gio.NewSimpleActionGroup()

	// apply action
	applyAction := gio.NewSimpleAction("apply", nil)
	applyAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		log.Println("Applying wallpaper:", wallpaperItem.WallpaperID)
		wallpaperDir := Config.Constants.WallpaperEngineDir
		fullWallpaperPath := path.Join(wallpaperDir, wallpaperItem.WallpaperID)
		go applyWallpaper(fullWallpaperPath, float64(Config.SavedUIState.Volume))
	})
	actionGroup.AddAction(&applyAction.Action)

	// toggle_favorite action
	toggleFavoriteAction := gio.NewSimpleAction("toggle_favorite", nil)
	toggleFavoriteAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		SelectedWallpaperItemId = wallpaperItem.WallpaperID
		if isFavorite {
			log.Printf("Removing %s from favorites", wallpaperItem.WallpaperID)
			wallpaperItem.IsFavorite = false
			Config.SavedUIState.Favorites = slices.Delete(Config.SavedUIState.Favorites, slices.Index(Config.SavedUIState.Favorites, wallpaperItem.WallpaperID), slices.Index(Config.SavedUIState.Favorites, wallpaperItem.WallpaperID)+1)
		} else {
			log.Printf("Favoriting %s", wallpaperItem.WallpaperID)
			wallpaperItem.IsFavorite = true
			Config.SavedUIState.Favorites = append(Config.SavedUIState.Favorites, wallpaperItem.WallpaperID)
		}

		for i := range WallpaperItems {
			if WallpaperItems[i].WallpaperID == wallpaperItem.WallpaperID {
				WallpaperItems[i].IsFavorite = wallpaperItem.IsFavorite
				break
			}
		}

		refreshWallpaperDisplay()
	})
	actionGroup.AddAction(&toggleFavoriteAction.Action)

	// toggle_broken action
	toggleBrokenAction := gio.NewSimpleAction("toggle_broken", nil)
	toggleBrokenAction.Connect("activate", func(_ *gio.SimpleAction, _ any) {
		SelectedWallpaperItemId = wallpaperItem.WallpaperID
		if isBroken {
			log.Printf("Marking %s as not broken", wallpaperItem.WallpaperID)
			wallpaperItem.IsBroken = false
			Config.SavedUIState.Broken = slices.Delete(Config.SavedUIState.Broken, slices.Index(Config.SavedUIState.Broken, wallpaperItem.WallpaperID), slices.Index(Config.SavedUIState.Broken, wallpaperItem.WallpaperID)+1)
		} else {
			log.Printf("Marking %s as broken", wallpaperItem.WallpaperID)
			wallpaperItem.IsBroken = true
			Config.SavedUIState.Broken = append(Config.SavedUIState.Broken, wallpaperItem.WallpaperID)
		}

		for i := range WallpaperItems {
			if WallpaperItems[i].WallpaperID == wallpaperItem.WallpaperID {
				WallpaperItems[i].IsBroken = wallpaperItem.IsBroken
				break
			}
		}

		refreshWallpaperDisplay()
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

	rightClickGesture := gtk.NewGestureClick()
	rightClickGesture.SetButton(3)
	imageWidget.AddController(rightClickGesture)
	rightClickGesture.ConnectReleased(func(nPress int, x, y float64) {
		if nPress == 1 { // ensure single click
			// context menu for the image
			contextMenuModel := gio.NewMenu()

			contextMenuModel.Append("Apply Wallpaper", wallpaperItem.WallpaperID+".apply")
			if isFavorite {
				contextMenuModel.Append("Remove from Favorites", wallpaperItem.WallpaperID+".toggle_favorite")
			} else {
				contextMenuModel.Append("Add to Favorites", wallpaperItem.WallpaperID+".toggle_favorite")
			}
			if isBroken {
				contextMenuModel.Append("Unmark as Broken", wallpaperItem.WallpaperID+".toggle_broken")
			} else {
				contextMenuModel.Append("Mark as Broken", wallpaperItem.WallpaperID+".toggle_broken")
			}
			contextMenuModel.Append("Open Wallpaper Directory", wallpaperItem.WallpaperID+".open_directory")
			contextMenuModel.Append("Copy Command to Clipboard", wallpaperItem.WallpaperID+".copy_command")

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

	leftClickGesture := gtk.NewGestureClick()
	leftClickGesture.SetButton(1)
	leftClickGesture.ConnectReleased(func(nPress int, x, y float64) {
		SelectedWallpaperItemId = wallpaperItem.WallpaperID
		showDetails(wallpaperItem)
		if nPress == 2 {
			log.Println("Double-click detected, applying wallpaper:", wallpaperItem.WallpaperID)
			wallpaperDir := Config.Constants.WallpaperEngineDir
			fullWallpaperPath := path.Join(wallpaperDir, wallpaperItem.WallpaperID)
			go applyWallpaper(fullWallpaperPath, float64(Config.SavedUIState.Volume))
		}
	})
	imageWidget.AddController(leftClickGesture)

	log.Println("Context menu attached to image widget for wallpaper:", wallpaperItem.WallpaperID)
}
