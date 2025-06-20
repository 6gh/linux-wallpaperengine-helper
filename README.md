# linux-wallpaperengine Helper

A really simple helper GUI app to apply wallpapers using [linux-wallpaperengine](https://github.com/Almamu/linux-wallpaperengine).

## Purpose

I wanted a GUI for linux-wallpaperengine as manually browsing the folders to get a new wallpaper was not it. Also changing the default on startup was tedious. Currently, there is no official GUI for linux-wallpaperengine, although [someone is making one which is hype](https://github.com/Almamu/linux-wallpaperengine/issues/320).

Until that one is completed, I made a very simple one which only applies wallpapers with options and takes a note of the current wallpaper. It also saves a screenshot and can run a configurable script to do whatever you'd like. For example, I have this apply the wallpaper, take a screenshot, and change the color scheme of my system with it.

Again, this is meant to be a simple GUI without advanced features like playlists. I just wanted something that works while I wait for the one linked above :)

## How to use

Run it with `./linux-wallpaperengine-helper`. If you want to restore on boot, you can configure your DE/WM to run `./linux-wallpaperengine-helper --restore` which tries to read the `last_set_id` from the config, set that ID, and then exits.

You can configure other stuff in `./config/linux-wallpaperengine-helper/settings.toml`.

## License

[MIT](./LICENSE)
