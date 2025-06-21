# linux-wallpaperengine Helper

A really simple helper GUI app to apply wallpapers using [linux-wallpaperengine](https://github.com/Almamu/linux-wallpaperengine).

## Purpose

I wanted a GUI for linux-wallpaperengine as manually browsing the folders to get a new wallpaper was not it. Also changing the default on startup was tedious. Currently, there is no official GUI for linux-wallpaperengine, although [someone is making one which is hype](https://github.com/Almamu/linux-wallpaperengine/issues/320).

Until that one is completed, I made a very simple one which only applies wallpapers with options and takes a note of the current wallpaper. It also saves a screenshot and can run a configurable command to do whatever you'd like. For example, I have this apply the wallpaper, take a screenshot, and change the color scheme of my system with it.

Again, this is meant to be a simple GUI without advanced features like playlists. I just wanted something that works (that I can use comfortably) while I wait for the one linked above :)

## How to use

Run it with `./linux-wallpaperengine-helper`. If you want to restore on boot, you can configure your DE/WM to run `./linux-wallpaperengine-helper --restore` which tries to read the `last_set_id` from the config, set that ID, and then exits.

You can configure other stuff in `~/.config/linux-wallpaperengine-helper/config.toml`.

## Configuration

Some configs are configurable via the UI, but every config is editable via the config.toml file

Below are the defaults, as well as some added comments to explain each one.

```toml
# Holds variables that should not be changed manually
[Constants]
# The executable path to the linux-wallpaperengine binary.
# Leave as-is if you added the binary to path
# Else, this should be an absolute path
# * Required
linux_wallpaperengine_bin = 'linux-wallpaperengine'

# The directory to look for wallpapers in
# This should point to Wallpaper Engins's workshop folder
# Must be absolute path, and cannot contain `~` for home folder
# * Required
wallpaper_engine_dir = '<home_dir>/.steam/steam/steamapps/workshop/content/431960'

# This is the latest wallpaper you set
# --restore will set the wallpaper to this
# * Can be an empty string
last_set_id = ''

# Holds configuration that is run after setting a wallpaper
[PostProcessing]
# Enables or disables this whole section
# * Required
enabled = true

# The file linux-wallpaperengine should save to
# This goes after the `--screenshot` flag
# * Can be an empty string
screenshot_file = '<home_dir>/.config/linux-wallpaperengine-helper/screenshot.png'

# The command to run after
# If empty, it will do nothing after setting a wallpaper
# You can put %screenshot% anywhere in the string which is replaced with the screenshot_file variable above
# * Can be an empty string
# *Disabled if screenshot_file is empty
post_command = ""

# Variables the UI reads
[SavedUIState]
# String array of wallpapers marked as broken
# If hide_broken is true, these won't appear in the UI
# If hide_broken is false, these will appear in the end of the list
# * Can be an empty array
broken = []

# String array of wallpapers favorited
# These will apear in the start of the list
# * Can be an empty array
favorites = []

# Determines if broken wallpapers should appear at all or not
# If false, will put broken wallpapers at the end
# * Required
hide_broken = false

# The volume of the wallpaper
# Set after the `--volume` flag
# If it is zero, it will add the `--silent` flag instead
# * Required; Must be an integer of 0-100
volume = 100
```

## License

[MIT](./LICENSE)
