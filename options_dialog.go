package main

import "github.com/diamondburned/gotk4/pkg/gtk/v4"

func showOptionsDialog() {
	dialog := gtk.NewWindow()
	dialog.SetTitle("Options")
	dialog.SetDefaultSize(400, 300)

	content := gtk.NewBox(gtk.OrientationVertical, 0)
	content.SetMarginTop(10)
	content.SetMarginBottom(10)
	content.SetMarginStart(10)
	content.SetMarginEnd(10)

	postProcessingLabel := gtk.NewLabel("Post-processing Options")
	content.Append(postProcessingLabel)

	postProcessingEnabled := gtk.NewCheckButtonWithLabel("Enable Post-Processing")
	postProcessingEnabled.SetActive(Config.PostProcessing.Enabled)
	postProcessingEnabled.Connect("toggled", func() {
		Config.PostProcessing.Enabled = postProcessingEnabled.Active()
	});
	content.Append(postProcessingEnabled)

	screenshotFileBar := gtk.NewSearchBar()
	screenshotFileBox := gtk.NewBox(gtk.OrientationVertical, 0)
	screenshotFileEntry := gtk.NewSearchEntry()
	screenshotFileEntry.SetPlaceholderText("Enter path to screenshot file")
	screenshotFileEntry.Connect("search-changed", func(entry *gtk.SearchEntry) {
		Config.PostProcessing.ScreenshotFile = entry.Text()
	})
	screenshotFileEntry.SetText(Config.PostProcessing.ScreenshotFile)
	screenshotFileLabel := gtk.NewLabel("Screenshot File:")
	screenshotFileBox.Append(screenshotFileLabel)
	screenshotFileBox.Append(screenshotFileEntry)
	screenshotFileBar.SetChild(screenshotFileBox)
	screenshotFileBar.ConnectEntry(screenshotFileEntry)
	screenshotFileBar.SetSearchMode(true)
	content.Append(screenshotFileBar)

	postCommandBar := gtk.NewSearchBar()
	postCommandBox := gtk.NewBox(gtk.OrientationVertical, 0)
	postCommandEntry := gtk.NewSearchEntry()
	postCommandEntry.SetPlaceholderText("Enter post command to run")
	postCommandEntry.Connect("search-changed", func(entry *gtk.SearchEntry) {
		Config.PostProcessing.PostCommand = entry.Text()
	})
	postCommandEntry.SetText(Config.PostProcessing.PostCommand)
	postCommandLabel := gtk.NewLabel("Post Command:")
	postCommandBox.Append(postCommandLabel)
	postCommandBox.Append(postCommandEntry)
	postCommandBar.SetChild(postCommandBox)
	postCommandBar.ConnectEntry(postCommandEntry)
	postCommandBar.SetSearchMode(true)
	content.Append(postCommandBar)

	setSWWWEnabled := gtk.NewCheckButtonWithLabel("Set swww to screenshot file")
	setSWWWEnabled.SetActive(Config.PostProcessing.SetSWWW)
	setSWWWEnabled.Connect("toggled", func() {
		Config.PostProcessing.SetSWWW = setSWWWEnabled.Active()
	});
	content.Append(setSWWWEnabled)

	UILabel := gtk.NewLabel("UI Options")
	content.Append(UILabel)

	hideBroken := gtk.NewCheckButtonWithLabel("Hide Broken Wallpapers")
	hideBroken.SetActive(Config.SavedUIState.HideBroken)
	hideBroken.Connect("toggled", func() {
		Config.SavedUIState.HideBroken = hideBroken.Active()
	});
	content.Append(hideBroken)

	dialog.SetChild(content)
	dialog.SetTransientFor(&Window.Window)
	dialog.SetModal(true)
	dialog.SetDestroyWithParent(true)
	dialog.SetVisible(true)
}
