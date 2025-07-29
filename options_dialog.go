package main

import (
	"context"
	"log"
	"time"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var Dialog *gtk.Window = nil
var reloadRequired bool = false
var refreshRequired bool = false
var filterRequired bool = false

func showOptionsDialog() {
	Dialog = gtk.NewWindow()
	Dialog.SetTitle("Options")
	Dialog.SetDefaultSize(600, 400)
	Dialog.SetHExpand(true)
	Dialog.SetVExpand(true)

	Dialog.Connect("close-request", func() bool {
		validateConfig()
		if reloadRequired || refreshRequired {
			if reloadRequired {
				reloadWallpaperData()
			}
			if refreshRequired {
				refreshWallpaperDisplay()
			}
		} else if filterRequired {
			filterWallpapersBySearch(SearchQuery)
		}
		return false
	})

	notebook := gtk.NewNotebook()

	notebook.AppendPage(createUIPage(), gtk.NewLabel("User Interface"))
	notebook.AppendPage(createConstantsPage(), gtk.NewLabel("Constants"))
	notebook.AppendPage(createPostProcessingPage(), gtk.NewLabel("Post Processing"))

	Dialog.SetChild(notebook)
	Dialog.SetTransientFor(&Window.Window)
	Dialog.SetModal(true)
	Dialog.SetDestroyWithParent(true)
	Dialog.SetVisible(true)
}

func createUIPage() *gtk.Box {
	uiPage := gtk.NewBox(gtk.OrientationVertical, 0)
	uiPage.SetMarginTop(10)
	uiPage.SetMarginBottom(10)
	uiPage.SetMarginStart(10)
	uiPage.SetMarginEnd(10)
	uiPage.SetSpacing(10)
	uiPage.SetHExpand(true)
	uiPage.SetVExpand(true)
	uiPage.SetHAlign(gtk.AlignFill)

	toggleablesLabel := gtk.NewLabel("Toggleables")
	toggleablesLabel.SetMarkup("<b>" + toggleablesLabel.Text() + "</b>")
	toggleablesLabel.SetHExpand(true)
	toggleablesLabel.SetHAlign(gtk.AlignStart)
	toggleablesLabel.SetMarginTop(10)
	toggleablesLabel.SetMarginBottom(10)
	uiPage.Append(toggleablesLabel)

	hideBrokenToggle := gtk.NewCheckButtonWithLabel("Hide Broken Wallpapers")
	hideBrokenToggle.SetHAlign(gtk.AlignStart)
	hideBrokenToggle.SetActive(Config.SavedUIState.HideBroken)
	hideBrokenToggle.Connect("toggled", func() {
		Config.SavedUIState.HideBroken = hideBrokenToggle.Active()
		filterRequired = true
	})
	uiPage.Append(hideBrokenToggle)

	actionsLabel := gtk.NewLabel("Quick Actions")
	actionsLabel.SetMarkup("<b>" + actionsLabel.Text() + "</b>")
	actionsLabel.SetHExpand(true)
	actionsLabel.SetHAlign(gtk.AlignStart)
	actionsLabel.SetMarginTop(10)
	actionsLabel.SetMarginBottom(10)
	uiPage.Append(actionsLabel)

	restoreButton := gtk.NewButtonWithLabel("Restore Last Set")
	restoreButton.SetHExpand(false)
	restoreButton.SetVExpand(false)
	restoreButton.SetHAlign(gtk.AlignStart)
	restoreButton.Connect("clicked", func() {
		log.Println("Restoring last set wallpaper...")
		go restoreWallpaper()
	})
	uiPage.Append(restoreButton)

	resetBrokenButton := gtk.NewButtonWithLabel("Reset Broken Wallpapers (Irreversible!)")
	resetBrokenButton.SetHExpand(false)
	resetBrokenButton.SetVExpand(false)
	resetBrokenButton.SetHAlign(gtk.AlignStart)
	resetBrokenButton.Connect("clicked", func() {
		// i have to use NewMessageDialog even though its deprecated
		// the reason is cause NewAlertDialog does not exist
		// so I'm not sure how i would create gtk.AlertDialog effectively
		//
		// Not the only one with this issue, as i found the following issue
		// see https://github.com/diamondburned/gotk4/issues/165
		dialog := gtk.NewMessageDialog(Dialog, gtk.DialogModal, gtk.MessageWarning, gtk.ButtonsYesNo)
		dialog.SetTitle("Confirm Reset")
		dialogMessage := gtk.NewLabel("Are you sure you want to reset all marks for broken wallpapers? This is irreversable!")
		if dialogBox, ok := dialog.MessageArea().(*gtk.Box); ok {
			dialogBox.Append(dialogMessage)
		} else {
			log.Println("Failed to set message area for dialog")
			dialog.SetTitle("Are you sure you want to reset all marks for broken wallpapers? This is irreversable!")
		}

		dialog.Connect("response", func(response gtk.ResponseType) {
			if response == gtk.ResponseYes {
				log.Println("Resetting broken wallpapers...")
				Config.SavedUIState.Broken = []string{}
				reloadRequired = true
				refreshRequired = true
			} else {
				log.Println("Reset broken wallpapers cancelled")
			}
			dialog.Destroy()
		})

		dialog.SetVisible(true)
	})
	uiPage.Append(resetBrokenButton)

	resetFavoritesButton := gtk.NewButtonWithLabel("Reset Favorites (Irreversible!)")
	resetFavoritesButton.SetHExpand(false)
	resetFavoritesButton.SetVExpand(false)
	resetFavoritesButton.SetHAlign(gtk.AlignStart)
	resetFavoritesButton.Connect("clicked", func() {
		// i have to use NewMessageDialog even though its deprecated
		// the reason is cause NewAlertDialog does not exist
		// so I'm not sure how i would create gtk.AlertDialog effectively
		//
		// Not the only one with this issue, as i found the following issue
		// see https://github.com/diamondburned/gotk4/issues/165
		dialog := gtk.NewMessageDialog(Dialog, gtk.DialogModal, gtk.MessageWarning, gtk.ButtonsYesNo)
		dialog.SetTitle("Confirm Reset")
		dialogMessage := gtk.NewLabel("Are you sure you want to reset all favorites? This is irreversable!")
		if dialogBox, ok := dialog.MessageArea().(*gtk.Box); ok {
			dialogBox.Append(dialogMessage)
		} else {
			log.Println("Failed to set message area for dialog")
			dialog.SetTitle("Are you sure you want to reset all favorites? This is irreversable!")
		}

		dialog.Connect("response", func(response gtk.ResponseType) {
			if response == gtk.ResponseYes {
				log.Println("Resetting favorites...")
				Config.SavedUIState.Favorites = []string{}
				reloadRequired = true
				refreshRequired = true
			} else {
				log.Println("Reset favorites cancelled")
			}
			dialog.Destroy()
		})

		dialog.SetVisible(true)
	})
	uiPage.Append(resetFavoritesButton)

	return uiPage
}

func createConstantsPage() *gtk.Box {
	constantsPage := gtk.NewBox(gtk.OrientationVertical, 0)
	constantsPage.SetMarginTop(10)
	constantsPage.SetMarginBottom(10)
	constantsPage.SetMarginStart(10)
	constantsPage.SetMarginEnd(10)
	constantsPage.SetSpacing(10)
	constantsPage.SetHExpand(true)
	constantsPage.SetVExpand(true)
	constantsPage.SetHAlign(gtk.AlignFill)

	toggleablesLabel := gtk.NewLabel("Toggleables")
	toggleablesLabel.SetMarkup("<b>" + toggleablesLabel.Text() + "</b>")
	toggleablesLabel.SetHExpand(true)
	toggleablesLabel.SetHAlign(gtk.AlignStart)
	toggleablesLabel.SetMarginTop(10)
	toggleablesLabel.SetMarginBottom(10)
	constantsPage.Append(toggleablesLabel)

	discardProcessLogsToggle := gtk.NewCheckButtonWithLabel("Discard Created Process Logs (stdout to /dev/null)")
	discardProcessLogsToggle.SetHAlign(gtk.AlignStart)
	discardProcessLogsToggle.SetActive(Config.Constants.DiscardProcessLogs)
	discardProcessLogsToggle.Connect("toggled", func() {
		Config.Constants.DiscardProcessLogs = discardProcessLogsToggle.Active()
	})
	constantsPage.Append(discardProcessLogsToggle)

	wallpaperEngineBinaryLabel := gtk.NewLabel("Wallpaper Engine Binary")
	wallpaperEngineBinaryLabel.SetMarkup("<b>" + wallpaperEngineBinaryLabel.Text() + "</b>")
	wallpaperEngineBinaryLabel.SetHExpand(true)
	wallpaperEngineBinaryLabel.SetHAlign(gtk.AlignStart)
	wallpaperEngineBinaryLabel.SetMarginTop(10)
	wallpaperEngineBinaryLabel.SetMarginBottom(10)
	constantsPage.Append(wallpaperEngineBinaryLabel)

	wallpaperEngineBinaryEntry := gtk.NewEntry()
	wallpaperEngineBinaryEntry.SetText(Config.Constants.LinuxWallpaperEngineBin)
	wallpaperEngineBinaryEntry.SetEditable(true)
	wallpaperEngineBinaryEntry.SetHExpand(true)
	wallpaperEngineBinaryEntry.SetHAlign(gtk.AlignFill)
	wallpaperEngineBinaryEntry.Connect("changed", func() {
		Config.Constants.LinuxWallpaperEngineBin = wallpaperEngineBinaryEntry.Text()
	})
	wallpaperEngineBinaryEntry.SetPlaceholderText("linux-wallpaperengine (must be in PATH!)")
	constantsPage.Append(wallpaperEngineBinaryEntry)

	wallpaperEngineDirLabel := gtk.NewLabel("Wallpaper Engine Content")
	wallpaperEngineDirLabel.SetMarkup("<b>" + wallpaperEngineDirLabel.Text() + "</b>")
	wallpaperEngineDirLabel.SetHExpand(true)
	wallpaperEngineDirLabel.SetHAlign(gtk.AlignStart)
	wallpaperEngineDirLabel.SetMarginTop(10)
	wallpaperEngineDirLabel.SetMarginBottom(10)
	constantsPage.Append(wallpaperEngineDirLabel)

	wallpaperEngineDirBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	wallpaperEngineDirBox.SetHExpand(true)
	wallpaperEngineDirBox.SetVExpand(false)
	constantsPage.Append(wallpaperEngineDirBox)

	wallpaperEngineDirEntry := gtk.NewEntry()
	wallpaperEngineDirEntry.SetText(Config.Constants.WallpaperEngineDir)
	wallpaperEngineDirEntry.SetEditable(false)
	wallpaperEngineDirEntry.SetHExpand(true)
	wallpaperEngineDirEntry.SetHAlign(gtk.AlignFill)

	wallpaperEngineDirButton := gtk.NewButtonFromIconName("folder-open")
	wallpaperEngineDirButton.SetHExpand(false)
	wallpaperEngineDirButton.SetVExpand(false)
	wallpaperEngineDirButton.SetHAlign(gtk.AlignStart)
	wallpaperEngineDirButton.SetSizeRequest(24, 24)
	wallpaperEngineDirButton.Connect("clicked", func() {
		filer := gio.NewFileForPath(Config.Constants.WallpaperEngineDir)

		// open a file dialog to select a new screenshot file
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("Select where to load Wallpaper Engine Wallpapers from")
		fileDialog.SetAcceptLabel("Select")
		fileDialog.SetModal(true)
		fileDialog.SetInitialFolder(filer)
		fileDialog.SelectFolder(context.TODO(), Dialog, func(result gio.AsyncResulter) {
			selectedFile, err := fileDialog.SelectFolderFinish(result)
			if err != nil {
				log.Printf("Failed to save screenshot file: %v", err)
				return
			}
			if selectedFile.Path() != "" {
				Config.Constants.WallpaperEngineDir = selectedFile.Path()
				wallpaperEngineDirEntry.SetText(Config.Constants.WallpaperEngineDir)
				reloadRequired = true
				refreshRequired = true
			}
		})
	})
	wallpaperEngineDirBox.Append(wallpaperEngineDirButton)
	wallpaperEngineDirBox.Append(wallpaperEngineDirEntry)

	wallpaperEngineAssetsLabel := gtk.NewLabel("Wallpaper Engine Assets")
	wallpaperEngineAssetsLabel.SetMarkup("<b>" + wallpaperEngineAssetsLabel.Text() + "</b>")
	wallpaperEngineAssetsLabel.SetHExpand(true)
	wallpaperEngineAssetsLabel.SetHAlign(gtk.AlignStart)
	wallpaperEngineAssetsLabel.SetMarginTop(10)
	wallpaperEngineAssetsLabel.SetMarginBottom(10)
	constantsPage.Append(wallpaperEngineAssetsLabel)

	wallpaperEngineAssetsBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	wallpaperEngineAssetsBox.SetHExpand(true)
	wallpaperEngineAssetsBox.SetVExpand(false)
	constantsPage.Append(wallpaperEngineAssetsBox)

	wallpaperEngineAssetsEntry := gtk.NewEntry()
	wallpaperEngineAssetsEntry.SetText(Config.Constants.WallpaperEngineAssets)
	wallpaperEngineAssetsEntry.SetEditable(false)
	wallpaperEngineAssetsEntry.SetHExpand(true)
	wallpaperEngineAssetsEntry.SetHAlign(gtk.AlignFill)

	wallpaperEngineAssetsButton := gtk.NewButtonFromIconName("folder-open")
	wallpaperEngineAssetsButton.SetHExpand(false)
	wallpaperEngineAssetsButton.SetVExpand(false)
	wallpaperEngineAssetsButton.SetHAlign(gtk.AlignStart)
	wallpaperEngineAssetsButton.SetSizeRequest(24, 24)
	wallpaperEngineAssetsButton.Connect("clicked", func() {
		filer := gio.NewFileForPath(Config.Constants.WallpaperEngineAssets)

		// open a file dialog to select a new screenshot file
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("Select where to load Wallpaper Engine Assets from")
		fileDialog.SetAcceptLabel("Select")
		fileDialog.SetModal(true)
		fileDialog.SetInitialFolder(filer)
		fileDialog.SelectFolder(context.TODO(), Dialog, func(result gio.AsyncResulter) {
			selectedFile, err := fileDialog.SelectFolderFinish(result)
			if err != nil {
				log.Printf("Failed to save screenshot file: %v", err)
				return
			}
			if selectedFile.Path() != "" {
				Config.Constants.WallpaperEngineAssets = selectedFile.Path()
				wallpaperEngineAssetsEntry.SetText(Config.Constants.WallpaperEngineAssets)
			}
		})
	})
	wallpaperEngineAssetsBox.Append(wallpaperEngineAssetsButton)
	wallpaperEngineAssetsBox.Append(wallpaperEngineAssetsEntry)

	return constantsPage
}

func createPostProcessingPage() *gtk.Box {
	postProcessingPage := gtk.NewBox(gtk.OrientationVertical, 0)
	postProcessingPage.SetMarginTop(10)
	postProcessingPage.SetMarginBottom(10)
	postProcessingPage.SetMarginStart(10)
	postProcessingPage.SetMarginEnd(10)
	postProcessingPage.SetSpacing(10)
	postProcessingPage.SetHExpand(true)
	postProcessingPage.SetVExpand(true)
	postProcessingPage.SetHAlign(gtk.AlignFill)

	toggleablesLabel := gtk.NewLabel("Toggleables")
	toggleablesLabel.SetMarkup("<b>" + toggleablesLabel.Text() + "</b>")
	toggleablesLabel.SetHExpand(true)
	toggleablesLabel.SetHAlign(gtk.AlignStart)
	toggleablesLabel.SetMarginTop(10)
	toggleablesLabel.SetMarginBottom(10)
	postProcessingPage.Append(toggleablesLabel)

	postProcessingEnabled := gtk.NewCheckButtonWithLabel("Enable Post-Processing")
	postProcessingEnabled.SetHAlign(gtk.AlignStart)
	postProcessingEnabled.SetActive(Config.PostProcessing.Enabled)
	postProcessingEnabled.Connect("toggled", func() {
		Config.PostProcessing.Enabled = postProcessingEnabled.Active()
	})
	postProcessingPage.Append(postProcessingEnabled)

	setSWWWEnabled := gtk.NewCheckButtonWithLabel("Set swww to screenshot file")
	setSWWWEnabled.SetHAlign(gtk.AlignStart)
	setSWWWEnabled.SetActive(Config.PostProcessing.SetSWWW)
	setSWWWEnabled.Connect("toggled", func() {
		Config.PostProcessing.SetSWWW = setSWWWEnabled.Active()
	})
	postProcessingPage.Append(setSWWWEnabled)

	artificialDelayLabel := gtk.NewLabel("Artificial Delay")
	artificialDelayLabel.SetMarkup("<b>" + artificialDelayLabel.Text() + "</b>")
	artificialDelayLabel.SetHExpand(true)
	artificialDelayLabel.SetHAlign(gtk.AlignStart)
	artificialDelayLabel.SetMarginTop(10)
	artificialDelayLabel.SetMarginBottom(10)
	postProcessingPage.Append(artificialDelayLabel)

	artificialDelayBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	postProcessingPage.Append(artificialDelayBox)

	artificialDelayWarning := gtk.NewImageFromIconName("dialog-warning-symbolic")
	artificialDelayWarning.SetHExpand(false)
	artificialDelayWarning.SetVExpand(false)
	artificialDelayWarning.SetHAlign(gtk.AlignStart)
	artificialDelayWarning.SetSizeRequest(24, 24)
	artificialDelayWarning.SetVisible(false)

	artificialDelayEntry := gtk.NewEntry()
	artificialDelayEntry.SetText((time.Duration(Config.PostProcessing.ArtificialDelay) * time.Second).String())
	artificialDelayEntry.SetEditable(true)
	artificialDelayEntry.SetHExpand(true)
	artificialDelayEntry.SetHAlign(gtk.AlignFill)
	artificialDelayEntry.SetPlaceholderText("Enter artificial delay in seconds (e.g. 2s, 1m) (Has to be at least 1s)")
	artificialDelayEntry.Connect("changed", func() {
		delay, err := time.ParseDuration(artificialDelayEntry.Text())
		if err != nil {
			artificialDelayWarning.SetTooltipText("Invalid duration format")
			artificialDelayWarning.SetVisible(true)
			return
		}

		Config.PostProcessing.ArtificialDelay = int64(delay / time.Second)
		artificialDelayWarning.SetVisible(false)
	})
	artificialDelayBox.Append(artificialDelayEntry)
	artificialDelayBox.Append(artificialDelayWarning)

	screenshotFilesLabel := gtk.NewLabel("Screenshot Files (remove all to disable)")
	screenshotFilesLabel.SetMarkup("<b>" + screenshotFilesLabel.Text() + "</b>")
	screenshotFilesLabel.SetHExpand(true)
	screenshotFilesLabel.SetHAlign(gtk.AlignStart)
	screenshotFilesLabel.SetMarginTop(10)
	screenshotFilesLabel.SetMarginBottom(10)
	postProcessingPage.Append(screenshotFilesLabel)

	screenshotFilesFlowBox := gtk.NewFlowBox()
	screenshotFilesFlowBox.SetHAlign(gtk.AlignFill)
	screenshotFilesFlowBox.SetOrientation(gtk.OrientationHorizontal)
	screenshotFilesFlowBox.SetSelectionMode(gtk.SelectionNone)
	screenshotFilesFlowBox.SetColumnSpacing(4)
	screenshotFilesFlowBox.SetRowSpacing(4)
	screenshotFilesFlowBox.SetMinChildrenPerLine(1)
	screenshotFilesFlowBox.SetMaxChildrenPerLine(1)
	screenshotFilesFlowBox.SetHomogeneous(true)
	screenshotFilesFlowBox.SetHExpand(true)
	screenshotFilesFlowBox.SetVExpand(false)
	refreshScreenshotFilesList(screenshotFilesFlowBox)
	postProcessingPage.Append(screenshotFilesFlowBox)

	postCommandLabel := gtk.NewLabel("Post Command")
	postCommandLabel.SetMarkup("<b>" + postCommandLabel.Text() + "</b>")
	postCommandLabel.SetHExpand(true)
	postCommandLabel.SetHAlign(gtk.AlignStart)
	postCommandLabel.SetMarginTop(10)
	postCommandLabel.SetMarginBottom(10)
	postProcessingPage.Append(postCommandLabel)

	postCommandEntry := gtk.NewEntry()
	postCommandEntry.SetText(Config.PostProcessing.PostCommand)
	postCommandEntry.SetEditable(true)
	postCommandEntry.SetHExpand(true)
	postCommandEntry.SetHAlign(gtk.AlignFill)
	postCommandEntry.Connect("changed", func() {
		Config.PostProcessing.PostCommand = postCommandEntry.Text()
	})
	postCommandEntry.SetPlaceholderText("Enter command to run after screenshot, leave empty to disable")
	postProcessingPage.Append(postCommandEntry)

	return postProcessingPage
}

func refreshScreenshotFilesList(screenshotFilesFlowBox *gtk.FlowBox) {
	screenshotFilesFlowBox.RemoveAll()

	for i, file := range Config.PostProcessing.ScreenshotFiles {
		hBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
		hBox.SetHExpand(true)
		hBox.SetVExpand(false)

		button := gtk.NewButtonFromIconName("document-open")
		button.SetHExpand(false)
		button.SetVExpand(false)
		button.SetHAlign(gtk.AlignStart)
		button.SetSizeRequest(24, 24)
		button.Connect("clicked", func() {
			filer := gio.NewFileForPath(file)

			// open a file dialog to select a new screenshot file
			fileDialog := gtk.NewFileDialog()
			fileDialog.SetTitle("Select where to save the screenshot file")
			fileDialog.SetAcceptLabel("Save")
			fileDialog.SetModal(true)
			fileDialog.SetInitialFile(filer)
			fileDialog.Save(context.TODO(), Dialog, func(result gio.AsyncResulter) {
				selectedFile, err := fileDialog.SaveFinish(result)
				if err != nil {
					log.Printf("Failed to save screenshot file: %v", err)
					return
				}
				if selectedFile.Path() != "" {
					Config.PostProcessing.ScreenshotFiles[i] = selectedFile.Path()
					refreshScreenshotFilesList(screenshotFilesFlowBox)
				}
			})
		})
		hBox.Append(button)

		label := gtk.NewEntry()
		label.SetText(file)
		label.SetEditable(false)
		label.SetHExpand(true)
		label.SetHAlign(gtk.AlignFill)
		hBox.Append(label)

		removeButton := gtk.NewButtonFromIconName("edit-delete")
		removeButton.SetHExpand(false)
		removeButton.SetVExpand(false)
		removeButton.SetHAlign(gtk.AlignEnd)
		removeButton.SetSizeRequest(24, 24)
		removeButton.Connect("clicked", func() {
			Config.PostProcessing.ScreenshotFiles = append(Config.PostProcessing.ScreenshotFiles[:i], Config.PostProcessing.ScreenshotFiles[i+1:]...)
			refreshScreenshotFilesList(screenshotFilesFlowBox)
		})
		hBox.Append(removeButton)

		screenshotFilesFlowBox.Append(hBox)
	}

	addButton := gtk.NewButtonFromIconName("list-add")
	addButton.SetHExpand(true)
	addButton.SetVExpand(false)
	addButton.SetHAlign(gtk.AlignFill)
	addButton.SetSizeRequest(-1, 24)
	addButton.Connect("clicked", func() {
		// open a file dialog to select a new screenshot file
		fileDialog := gtk.NewFileDialog()
		fileDialog.SetTitle("Select where to save the screenshot file")
		fileDialog.SetAcceptLabel("Save")
		fileDialog.SetModal(true)
		fileDialog.Save(context.TODO(), Dialog, func(result gio.AsyncResulter) {
			selectedFile, err := fileDialog.SaveFinish(result)
			if err != nil {
				log.Printf("Failed to save screenshot file: %v", err)
				return
			}
			if selectedFile.Path() != "" {
				Config.PostProcessing.ScreenshotFiles = append(Config.PostProcessing.ScreenshotFiles, selectedFile.Path())
				refreshScreenshotFilesList(screenshotFilesFlowBox)
			}
		})
	})

	screenshotFilesFlowBox.Append(addButton)
}
