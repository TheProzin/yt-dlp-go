# yt-dlp-go

Desktop manager (Fyne) for `yt-dlp`.

**Languages:** English (this file) · [Português (Brasil)](README.pt-BR.md)

## Legal notice and personal use

This tool is intended for **personal and private use** and should only be used
to download content that is:

- in the **public domain**;
- under a **license that permits** download and/or redistribution;
- **expressly authorized** by the author or the platform;
- or that **you own the rights to**.

The author of yt-dlp-go **does not encourage or support** copyright infringement
or violations of third-party terms of service. You are **solely responsible**
for using the tool and for the content you download. The software is provided
**"as is"**, without warranties of any kind, and the author is not liable for
any misuse, data loss, or legal consequences arising from its use.

> Summary: it is a personal tool for free/authorized content. Respect copyright
> and each site's terms of service.

## Features

- Download one or more URLs (one per line).
- Choose the output folder and where auxiliary files are saved.
- Quality: presets (Best, 1080p, 720p, 480p, audio-only MP3) plus real format
  selection via `yt-dlp -J`.
- **H.264 preference** (compatible with older players such as VLC 3.x).
- Browser cookies (Chrome, Firefox, etc.) for login / anti-bot.
- Advanced options: subtitles, thumbnail, metadata/chapters, filename and
  organization, playlists, network (rate limit, proxy, retries) and a download
  archive to avoid duplicates.
- Persistent history, dependency diagnostics and real-time log.
- **Parallel downloads**: configurable concurrency (default 5) for multiple
  URLs and playlist items.
- Cancelling kills the whole process tree (including `ffmpeg`).
- **UI language: English by default, pt-BR optional** (Settings → Language).

## Requirements

- Go 1.25+ (only to build from source; the ready packages include the binary)
- `yt-dlp` in PATH or at `~/.local/bin/yt-dlp` (or `~/.config/yt-dlg/yt-dlp.real`)
- `deno` at `~/.deno/bin` (yt-dlp uses it to solve JS challenges / unlock 1080p)
- `ffmpeg` to merge audio/video and convert to MP3
- Browser logged in to YouTube (if using cookies)

## Installation

### Ubuntu / Debian (.deb)

Download `yt-dlp-go_1.1.0_amd64.deb` and install with a double click or:

```bash
sudo apt install ./yt-dlp-go_1.1.0_amd64.deb
```

Or with `sudo dpkg -i yt-dlp-go_1.1.0_amd64.deb`.
Then open **yt-dlp-go** from the menu.

Remove with:

```bash
sudo apt remove yt-dlp-go
```

### Any Linux distro (AppImage)

The AppImage is a single file, no installation needed:

```bash
chmod +x yt-dlp-go-1.1.0-amd64.AppImage
./yt-dlp-go-1.1.0-amd64.AppImage
```

> On distros without FUSE 2, run with `--appimage-extract-and-run`.
> On Ubuntu 24.04: `sudo apt install libfuse2t64` (or use the flag above).

### Development

```bash
go run .                 # run directly
go build -o yt-dlp-go .  # build the binary
```

### Build the packages (.deb + AppImage)

```bash
./package.sh             # requires dpkg-deb and downloads appimagetool on first run
# output in dist/:
#   yt-dlp-go_1.1.0_amd64.deb
#   yt-dlp-go-1.1.0-amd64.AppImage
```

## Where files are saved

Everything starts from the **videos folder** chosen on the Download tab. By
default, auxiliary files go into automatic subfolders:

```
<Videos>/<video name>
<Videos>/subtitles/...
<Videos>/thumbnails/...
<Videos>/metadata/...
```

The `metadata/` folder is used when **Save metadata file (.info.json)** is enabled.
You can change the subfolder names in **Settings**.

## Persistence

- `~/.config/yt-dlp-go/settings.json` - preferences (including language).
- `~/.config/yt-dlp-go/history.json` - history (permission 0600).
- `~/.config/yt-dlp-go/.lock` - single-instance lock.
- `~/.config/yt-dlp-go/download-archive.txt` - download archive (if enabled).

## Notes

- The yt-dlp binary is located in this order: `$YTDLP_BIN`, `~/.local/bin/yt-dlp`,
  `~/.config/yt-dlg/yt-dlp.real`, `~/.config/yt-dlg/yt-dlp`, then `PATH`.
- The app passes `--ignore-config`, so it does not inherit `~/.config/yt-dlp/config`.
- The app requires a browser login for videos with YouTube's anti-bot check.
- Full playlists are expanded into individual videos so they can download in
  parallel; playlist-specific template fields may be unavailable.
