package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"yt-dlp-go/internal/deps"
	"yt-dlp-go/internal/downloader"
	"yt-dlp-go/internal/history"
	"yt-dlp-go/internal/i18n"
	"yt-dlp-go/internal/settings"
)

//go:embed assets/icon.png
var iconPNG []byte

type preset struct {
	Label      string
	FormatBest string
	FormatH264 string
	Merge      string
	AudioOnly  bool
}

var presets = []preset{
	{"Best quality", "bv*+ba/b", "bv*[vcodec^=avc1]+ba[ext=m4a]/b[vcodec^=avc1]/b", "mp4", false},
	{"1080p", "bv*[height<=1080]+ba/b[height<=1080]", "bv*[height<=1080][vcodec^=avc1]+ba[ext=m4a]/b[height<=1080][vcodec^=avc1]/b[height<=1080]", "mp4", false},
	{"720p", "bv*[height<=720]+ba/b[height<=720]", "bv*[height<=720][vcodec^=avc1]+ba[ext=m4a]/b[height<=720][vcodec^=avc1]/b[height<=720]", "mp4", false},
	{"480p", "bv*[height<=480]+ba/b[height<=480]", "bv*[height<=480][vcodec^=avc1]+ba[ext=m4a]/b[height<=480][vcodec^=avc1]/b[height<=480]", "mp4", false},
	{"Audio only (MP3)", "ba/b", "ba/b", "", true},
}

var (
	cookieValues = []string{"", "chrome", "firefox", "brave", "edge", "chromium", "opera", "vivaldi"}

	playlistValues = []string{"no", "yes", "items"}
	playlistKeys   = []string{"Single video", "Full playlist", "Range"}

	languageValues = []string{"en", "pt-BR"}
	languageKeys   = []string{"English", "Portuguese (Brazil)"}
)

func presetLabels() []string {
	l := make([]string, len(presets))
	for i, p := range presets {
		l[i] = i18n.T(p.Label)
	}
	return l
}

func cookieLabels() []string {
	l := make([]string, len(cookieValues))
	for i := range cookieValues {
		if i == 0 {
			l[i] = i18n.T("None")
		} else {
			l[i] = cookieValues[i]
		}
	}
	return l
}

func indexOf(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return 0
}

func translatedKeys(keys []string) []string {
	l := make([]string, len(keys))
	for i, k := range keys {
		l[i] = i18n.T(k)
	}
	return l
}

type logView struct {
	mu     sync.Mutex
	lines  []string
	dirty  bool
	last   time.Time
	label  *widget.Label
	scroll *container.Scroll
}

func newLogView() *logView {
	l := &logView{label: widget.NewLabel("")}
	l.label.Wrapping = fyne.TextWrapWord
	l.scroll = container.NewVScroll(l.label)
	l.scroll.SetMinSize(fyne.NewSize(820, 200))
	return l
}

func (l *logView) Add(s string) {
	l.mu.Lock()
	l.lines = append(l.lines, s)
	if len(l.lines) > 600 {
		l.lines = l.lines[len(l.lines)-600:]
	}
	l.dirty = true
	l.mu.Unlock()
}

func (l *logView) Clear() {
	l.mu.Lock()
	l.lines = nil
	l.dirty = true
	l.last = time.Time{}
	l.mu.Unlock()
}

func (l *logView) Flush() {
	l.mu.Lock()
	if !l.dirty || time.Since(l.last) < 120*time.Millisecond {
		l.mu.Unlock()
		return
	}
	text := strings.Join(l.lines, "\n")
	l.dirty = false
	l.last = time.Now()
	l.mu.Unlock()
	l.label.SetText(text)
	l.scroll.ScrollToBottom()
}

type appUI struct {
	fyneApp  fyne.App
	win      fyne.Window
	cfg      settings.Settings
	store    *history.Store
	storeErr error
	mgr      *downloader.Manager

	urlEntry     *widget.Entry
	dirEntry     *widget.Entry
	quality      *widget.Select
	preferH264   *widget.Check
	cookiesSel   *widget.Select
	customFormat string
	customLabel  *widget.Label

	subsCheck     *widget.Check
	autoSubs      *widget.Check
	subLangs      *widget.Entry
	embedSubs     *widget.Check
	thumbCheck    *widget.Check
	embedThumb    *widget.Check
	metaCheck     *widget.Check
	chapCheck     *widget.Check
	infoJSON      *widget.Check
	tmplEntry     *widget.Entry
	organize      *widget.Check
	winNames      *widget.Check
	playlistSel   *widget.Select
	playlistItems *widget.Entry
	limitRate     *widget.Entry
	proxy         *widget.Entry
	retries       *widget.Entry
	fragments     *widget.Entry
	archive       *widget.Check
	noOverwrite   *widget.Check

	progress    *widget.ProgressBar
	status      *widget.Label
	log         *logView
	downloadBtn *widget.Button
	cancelBtn   *widget.Button

	historyList     *widget.List
	historyItems    []history.Item
	selectedHistory int

	diagLabel *widget.Label

	tabs *container.AppTabs
}

func main() {
	cfg := settings.Load()
	i18n.Set(cfg.Language)
	configureFyneLocale()
	registerTranslations()

	a := app.NewWithID("br.com.yt-dlp-go")
	a.SetIcon(fyne.NewStaticResource("icon.png", iconPNG))

	u := &appUI{
		fyneApp: a,
		cfg:     cfg,
		mgr:     downloader.New(),
	}
	u.store, u.storeErr = history.Open()

	u.win = a.NewWindow(i18n.T("yt-dlp-go - yt-dlp manager"))
	u.win.Resize(fyne.NewSize(cfg.WindowWidth, cfg.WindowHeight))
	u.build()

	u.win.SetOnClosed(func() {
		size := u.win.Canvas().Size()
		u.cfg.WindowWidth = size.Width
		u.cfg.WindowHeight = size.Height
		_ = settings.Save(u.cfg)
		if u.store != nil {
			u.store.Close()
		}
	})

	if u.storeErr != nil {
		dialog.ShowError(u.storeErr, u.win)
	}

	u.win.ShowAndRun()
}

func configureFyneLocale() {
	if i18n.IsPortuguese() {
		os.Setenv("LANGUAGE", "pt_BR:pt")
		os.Setenv("LC_ALL", "pt_BR.UTF-8")
		os.Setenv("LANG", "pt_BR.UTF-8")
	} else {
		os.Setenv("LANGUAGE", "en")
		os.Setenv("LC_ALL", "en_US.UTF-8")
		os.Setenv("LANG", "en_US.UTF-8")
	}
	// Re-evaluate Fyne's locale using the environment set above.
	_ = lang.AddTranslationsForLocale([]byte("{}"), lang.SystemLocale())
}

func registerTranslations() {
	loc := lang.SystemLocale()
	placeholder := "(Select one)"
	if strings.HasPrefix(strings.ToLower(string(loc)), "pt") {
		placeholder = "Selecione um item"
	}
	data := []byte(fmt.Sprintf(`{"(Select one)": %q}`, placeholder))
	_ = lang.AddTranslationsForLocale(data, loc)
}

func (u *appUI) build() {
	u.log = newLogView()

	u.tabs = container.NewAppTabs(
		container.NewTabItem(i18n.T("Download"), u.buildDownloadTab()),
		container.NewTabItem(i18n.T("History"), u.buildHistoryTab()),
		container.NewTabItem(i18n.T("Settings"), u.buildSettingsTab()),
		container.NewTabItem(i18n.T("Diagnostics"), u.buildDiagTab()),
	)
	u.win.SetContent(u.tabs)
	u.refreshHistory()
	bin, dir := u.cfg.YtdlpPath, u.cfg.VideoDir
	go u.runDiagnostics(bin, dir)
}

func label(text string, obj fyne.CanvasObject) fyne.CanvasObject {
	l := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewVBox(l, obj)
}

func (u *appUI) buildDownloadTab() fyne.CanvasObject {
	u.urlEntry = widget.NewMultiLineEntry()
	u.urlEntry.SetPlaceHolder(i18n.T("Paste one or more URLs (one per line)..."))
	u.urlEntry.SetMinRowsVisible(3)

	u.dirEntry = widget.NewEntry()
	u.dirEntry.SetText(u.cfg.VideoDir)

	chooseBtn := widget.NewButton(i18n.T("Choose..."), func() {
		d := dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil || lu == nil {
				return
			}
			u.dirEntry.SetText(lu.Path())
		}, u.win)
		d.Show()
	})
	openBtn := widget.NewButton(i18n.T("Open"), func() {
		dir := strings.TrimSpace(u.dirEntry.Text)
		if err := openPath(dir); err != nil {
			dialog.ShowError(fmt.Errorf(i18n.T("Could not open %q: %v"), dir, err), u.win)
		}
	})
	dirRow := container.NewBorder(nil, nil, nil, container.NewHBox(openBtn, chooseBtn), u.dirEntry)

	u.quality = widget.NewSelect(presetLabels(), func(string) {
		u.customFormat = ""
		if u.customLabel != nil {
			u.customLabel.SetText(i18n.T("Format: automatic (preset)"))
		}
	})
	idx := u.cfg.PresetIndex
	if idx < 0 || idx >= len(presets) {
		idx = 0
	}
	u.quality.SetSelectedIndex(idx)

	u.preferH264 = widget.NewCheck(i18n.T("Prefer H.264 (compatible)"), nil)
	u.preferH264.SetChecked(true)

	u.cookiesSel = widget.NewSelect(cookieLabels(), nil)
	u.cookiesSel.SetSelectedIndex(indexOf(cookieValues, u.cfg.CookiesBrowser))

	u.customFormat = ""
	u.customLabel = widget.NewLabel(i18n.T("Format: automatic (preset)"))
	formatsBtn := widget.NewButton(i18n.T("Formats..."), func() { u.openFormatsDialog() })
	qualityRow := container.NewBorder(nil, nil, nil, formatsBtn, u.quality)

	u.subsCheck = widget.NewCheck(i18n.T("Download subtitles"), nil)
	u.autoSubs = widget.NewCheck(i18n.T("Include auto-generated subtitles"), nil)
	u.subLangs = widget.NewEntry()
	u.subLangs.SetText("pt.*,pt,en.*,en")
	u.embedSubs = widget.NewCheck(i18n.T("Embed subtitles in video"), nil)

	u.thumbCheck = widget.NewCheck(i18n.T("Download thumbnail"), nil)
	u.embedThumb = widget.NewCheck(i18n.T("Embed thumbnail"), nil)

	u.metaCheck = widget.NewCheck(i18n.T("Embed metadata"), nil)
	u.chapCheck = widget.NewCheck(i18n.T("Embed chapters"), nil)
	u.infoJSON = widget.NewCheck(i18n.T("Save metadata file (.info.json)"), nil)

	u.tmplEntry = widget.NewEntry()
	u.tmplEntry.SetText(u.cfg.FilenameTemplate)
	u.organize = widget.NewCheck(i18n.T("Organize into channel subfolder"), nil)
	u.winNames = widget.NewCheck(i18n.T("Windows-compatible names"), nil)

	u.playlistSel = widget.NewSelect(translatedKeys(playlistKeys), nil)
	u.playlistSel.SetSelectedIndex(0)
	u.playlistItems = widget.NewEntry()
	u.playlistItems.SetPlaceHolder(i18n.T("Example: 1-5,8,10-"))

	u.limitRate = widget.NewEntry()
	u.limitRate.SetPlaceHolder(i18n.T("Example: 5M (empty = no limit)"))
	u.proxy = widget.NewEntry()
	u.proxy.SetPlaceHolder(i18n.T("Example: socks5://127.0.0.1:9050"))
	u.retries = widget.NewEntry()
	u.retries.SetText("10")
	u.fragments = widget.NewEntry()
	u.fragments.SetText("4")

	u.archive = widget.NewCheck(i18n.T("Record downloaded (avoid repeats)"), nil)
	u.noOverwrite = widget.NewCheck(i18n.T("Do not overwrite existing files"), nil)

	adv := widget.NewAccordion(
		widget.NewAccordionItem(i18n.T("Subtitles"), container.NewVBox(u.subsCheck, u.autoSubs, label(i18n.T("Languages"), u.subLangs), u.embedSubs)),
		widget.NewAccordionItem(i18n.T("Thumbnail"), container.NewVBox(u.thumbCheck, u.embedThumb)),
		widget.NewAccordionItem(i18n.T("Metadata"), container.NewVBox(u.metaCheck, u.chapCheck, u.infoJSON)),
		widget.NewAccordionItem(i18n.T("Name and organization"), container.NewVBox(label(i18n.T("Template"), u.tmplEntry), u.organize, u.winNames)),
		widget.NewAccordionItem(i18n.T("Playlist"), container.NewVBox(u.playlistSel, label(i18n.T("Range"), u.playlistItems))),
		widget.NewAccordionItem(i18n.T("Network"), container.NewVBox(label(i18n.T("Limit"), u.limitRate), label(i18n.T("Proxy"), u.proxy), label(i18n.T("Retries"), u.retries), label(i18n.T("Fragments per download"), u.fragments))),
		widget.NewAccordionItem(i18n.T("Duplicates"), container.NewVBox(u.archive, u.noOverwrite)),
	)

	u.progress = widget.NewProgressBar()
	u.progress.SetValue(0)
	u.status = widget.NewLabel(i18n.T("No download in progress."))

	u.downloadBtn = widget.NewButton(i18n.T("Download"), func() { u.startDownload() })
	u.downloadBtn.Importance = widget.HighImportance
	u.cancelBtn = widget.NewButton(i18n.T("Cancel"), func() {
		u.mgr.Cancel()
		u.status.SetText(i18n.T("Cancelling..."))
	})
	u.cancelBtn.Disable()

	content := container.NewVBox(
		label(i18n.T("URLs (one per line)"), u.urlEntry),
		label(i18n.T("Output folder"), dirRow),
		container.NewBorder(nil, nil, widget.NewLabel(i18n.T("Quality:")), nil, qualityRow),
		container.NewHBox(u.preferH264, widget.NewLabel(i18n.T("Cookies:")), u.cookiesSel),
		u.customLabel,
		adv,
		container.NewHBox(u.downloadBtn, u.cancelBtn),
		u.progress,
		u.status,
		label(i18n.T("Log"), u.log.scroll),
	)
	return container.NewPadded(container.NewVScroll(content))
}

func (u *appUI) buildHistoryTab() fyne.CanvasObject {
	u.selectedHistory = -1
	u.historyList = widget.NewList(
		func() int { return len(u.historyItems) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id < 0 || id >= len(u.historyItems) {
				return
			}
			it := u.historyItems[id]
			o.(*widget.Label).SetText(fmt.Sprintf(i18n.T("%s  [%s]  %s  (%d items)\n%s"),
				it.StartedAt.Format("2006-01-02 15:04"), i18n.T(it.Status), it.Preset, it.Count, it.OutputDir))
		},
	)
	u.historyList.OnSelected = func(id widget.ListItemID) { u.selectedHistory = id }

	openBtn := widget.NewButton(i18n.T("Open folder"), func() {
		idx := u.selectedHistory
		if idx < 0 && len(u.historyItems) > 0 {
			idx = 0
		}
		if idx < 0 || idx >= len(u.historyItems) {
			dialog.ShowInformation(i18n.T("Open folder"), i18n.T("No items in history."), u.win)
			return
		}
		dir := u.historyItems[idx].OutputDir
		if dir == "" {
			dir = u.cfg.VideoDir
		}
		if err := openPath(dir); err != nil {
			dialog.ShowError(fmt.Errorf(i18n.T("Could not open %q: %v"), dir, err), u.win)
		}
	})
	repeatBtn := widget.NewButton(i18n.T("Repeat"), func() {
		if u.selectedHistory >= 0 && u.selectedHistory < len(u.historyItems) {
			it := u.historyItems[u.selectedHistory]
			u.urlEntry.SetText(strings.Join(it.URLs, "\n"))
			u.tabs.SelectIndex(0)
		}
	})
	refreshBtn := widget.NewButton(i18n.T("Refresh"), func() { u.refreshHistory() })

	header := widget.NewLabel(i18n.T("History of downloads"))
	if u.storeErr != nil {
		header.SetText(fmt.Sprintf(i18n.T("History unavailable: %s"), u.storeErr.Error()))
	}
	return container.NewBorder(
		container.NewVBox(header, container.NewHBox(openBtn, repeatBtn, refreshBtn)),
		nil, nil, nil,
		u.historyList,
	)
}

func (u *appUI) refreshHistory() {
	if u.store == nil || u.historyList == nil {
		return
	}
	u.historyItems = u.store.Items()
	u.historyList.Refresh()
}

func (u *appUI) buildSettingsTab() fyne.CanvasObject {
	videoEntry := widget.NewEntry()
	videoEntry.SetText(u.cfg.VideoDir)
	subEntry := widget.NewEntry()
	subEntry.SetText(u.cfg.SubtitleDir)
	thumbEntry := widget.NewEntry()
	thumbEntry.SetText(u.cfg.ThumbnailDir)
	metaEntry := widget.NewEntry()
	metaEntry.SetText(u.cfg.MetadataDir)
	tmplEntry := widget.NewEntry()
	tmplEntry.SetText(u.cfg.FilenameTemplate)

	presetSel := widget.NewSelect(presetLabels(), nil)
	presetSel.SetSelectedIndex(u.cfg.PresetIndex)

	cookieSel := widget.NewSelect(cookieLabels(), nil)
	cookieSel.SetSelectedIndex(indexOf(cookieValues, u.cfg.CookiesBrowser))

	concOptions := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	concSel := widget.NewSelect(concOptions, nil)
	concSel.SetSelected(strconv.Itoa(u.cfg.Concurrency))

	ytdlpEntry := widget.NewEntry()
	ytdlpEntry.SetText(u.cfg.YtdlpPath)
	ytdlpEntry.SetPlaceHolder(i18n.T("empty = auto-detect"))

	languageSel := widget.NewSelect(translatedKeys(languageKeys), nil)
	languageSel.SetSelectedIndex(indexOf(languageValues, u.cfg.Language))

	detectBtn := widget.NewButton(i18n.T("Detect"), func() {
		if b, err := downloader.Binary(); err == nil {
			ytdlpEntry.SetText(b)
		} else {
			dialog.ShowError(err, u.win)
		}
	})

	updateBtn := widget.NewButton(i18n.T("Update yt-dlp"), func() {
		bin := ytdlpEntry.Text
		if bin == "" {
			if b, err := downloader.Binary(); err == nil {
				bin = b
			}
		}
		if bin == "" {
			dialog.ShowInformation("yt-dlp", i18n.T("Binary not found."), u.win)
			return
		}
		go func() {
			out, err := exec.Command(bin, "-U").CombinedOutput()
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(fmt.Errorf("%v\n%s", err, string(out)), u.win)
				} else {
					dialog.ShowInformation("yt-dlp", string(out), u.win)
				}
			})
		}()
	})

	saveBtn := widget.NewButton(i18n.T("Save settings"), func() {
		oldLang := u.cfg.Language
		u.cfg.VideoDir = strings.TrimSpace(videoEntry.Text)
		u.cfg.SubtitleDir = strings.TrimSpace(subEntry.Text)
		u.cfg.ThumbnailDir = strings.TrimSpace(thumbEntry.Text)
		u.cfg.MetadataDir = strings.TrimSpace(metaEntry.Text)
		u.cfg.FilenameTemplate = strings.TrimSpace(tmplEntry.Text)
		if presetSel.SelectedIndex() >= 0 {
			u.cfg.PresetIndex = presetSel.SelectedIndex()
		}
		u.cfg.CookiesBrowser = cookieValues[cookieSel.SelectedIndex()]
		u.cfg.Concurrency = atoiDefault(concSel.Selected, 5)
		u.cfg.YtdlpPath = strings.TrimSpace(ytdlpEntry.Text)
		u.cfg.Language = languageValues[languageSel.SelectedIndex()]
		if err := settings.Save(u.cfg); err != nil {
			dialog.ShowError(err, u.win)
			return
		}
		u.dirEntry.SetText(u.cfg.VideoDir)
		if u.cfg.PresetIndex >= 0 && u.cfg.PresetIndex < len(presets) {
			u.quality.SetSelectedIndex(u.cfg.PresetIndex)
		}
		u.cookiesSel.SetSelectedIndex(indexOf(cookieValues, u.cfg.CookiesBrowser))
		u.status.SetText(i18n.T("Settings saved."))
		if oldLang != u.cfg.Language {
			dialog.ShowInformation(i18n.T("Settings"), i18n.T("Restart the app to apply the language."), u.win)
		}
	})

	form := widget.NewForm(
		widget.NewFormItem(i18n.T("Videos folder"), videoEntry),
		widget.NewFormItem(i18n.T("Subtitles subfolder"), subEntry),
		widget.NewFormItem(i18n.T("Thumbnails subfolder"), thumbEntry),
		widget.NewFormItem(i18n.T("Metadata subfolder"), metaEntry),
		widget.NewFormItem(i18n.T("Filename template"), tmplEntry),
		widget.NewFormItem(i18n.T("Default preset"), presetSel),
		widget.NewFormItem(i18n.T("Cookies browser"), cookieSel),
		widget.NewFormItem(i18n.T("Parallel downloads"), concSel),
		widget.NewFormItem(i18n.T("yt-dlp path"), container.NewBorder(nil, nil, nil, container.NewHBox(detectBtn, updateBtn), ytdlpEntry)),
		widget.NewFormItem(i18n.T("Language"), languageSel),
	)
	return container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("Settings"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form,
		saveBtn,
	))
}

func (u *appUI) buildDiagTab() fyne.CanvasObject {
	u.diagLabel = widget.NewLabel(i18n.T("Checking..."))
	u.diagLabel.Wrapping = fyne.TextWrapWord
	checkBtn := widget.NewButton(i18n.T("Check now"), func() {
		u.diagLabel.SetText(i18n.T("Checking..."))
		bin, dir := u.cfg.YtdlpPath, u.cfg.VideoDir
		go u.runDiagnostics(bin, dir)
	})
	return container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle(i18n.T("Diagnostics"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		checkBtn,
		u.diagLabel,
	))
}

func (u *appUI) runDiagnostics(bin, dir string) {
	list := deps.Check(bin, dir)
	var b strings.Builder
	for _, s := range list {
		mark := "OK   "
		if !s.OK {
			mark = i18n.T("FAIL")
		}
		b.WriteString(fmt.Sprintf("[%s] %s", mark, s.Name))
		if s.Version != "" {
			b.WriteString(" - " + s.Version)
		}
		if s.Detail != "" {
			b.WriteString(" (" + s.Detail + ")")
		}
		b.WriteString("\n")
	}
	text := b.String()
	fyne.Do(func() {
		if u.diagLabel != nil {
			u.diagLabel.SetText(text)
		}
	})
}

func openPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New(i18n.T("empty path"))
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf(i18n.T("folder does not exist: %w"), err)
	}
	if !fi.IsDir() {
		path = filepath.Dir(path)
	}
	openers := [][]string{
		{"xdg-open", path},
		{"exo-open", path},
		{"gio", "open", path},
		{"thunar", path},
		{"dolphin", path},
		{"nautilus", path},
		{"pcmanfm", path},
		{"caja", path},
		{"nemo", path},
	}
	var lastErr error
	for _, c := range openers {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		if err := cmd.Start(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}
	if lastErr == nil {
		lastErr = errors.New(i18n.T("no folder opener found"))
	}
	return lastErr
}

func parseURLs(text string) []string {
	var urls []string
	for _, ln := range strings.Split(text, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			urls = append(urls, ln)
		}
	}
	return urls
}

func humanSize(n int64) string {
	if n <= 0 {
		return "-"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

func (u *appUI) setBusy(busy bool) {
	if busy {
		u.downloadBtn.Disable()
		u.cancelBtn.Enable()
	} else {
		u.downloadBtn.Enable()
		u.cancelBtn.Disable()
	}
}

func (u *appUI) notify(title, msg string) {
	u.fyneApp.SendNotification(fyne.NewNotification(title, msg))
}

func (u *appUI) startDownload() {
	urls := parseURLs(u.urlEntry.Text)
	if len(urls) == 0 {
		dialog.ShowInformation(i18n.T("Warning"), i18n.T("Enter at least one URL."), u.win)
		return
	}
	videoDir := strings.TrimSpace(u.dirEntry.Text)
	if videoDir == "" {
		dialog.ShowInformation(i18n.T("Warning"), i18n.T("Choose the output folder."), u.win)
		return
	}
	u.cfg.VideoDir = videoDir

	idx := u.quality.SelectedIndex()
	if idx < 0 || idx >= len(presets) {
		idx = 0
	}
	p := presets[idx]

	format := p.FormatBest
	merge := p.Merge
	var extra []string
	if u.customFormat != "" {
		format = u.customFormat
		merge = "mp4"
	} else if u.preferH264.Checked {
		format = p.FormatH264
	}
	if p.AudioOnly && u.customFormat == "" {
		extra = []string{"--extract-audio", "--audio-format", "mp3", "--audio-quality", "0"}
		merge = ""
	}

	writeSubs := u.subsCheck.Checked || u.embedSubs.Checked
	writeThumb := u.thumbCheck.Checked || u.embedThumb.Checked

	opts := downloader.JobConfig{
		URLs:                urls,
		VideoDir:            videoDir,
		SubtitleDir:         settings.ResolveDir(videoDir, u.cfg.SubtitleDir),
		ThumbnailDir:        settings.ResolveDir(videoDir, u.cfg.ThumbnailDir),
		MetadataDir:         settings.ResolveDir(videoDir, u.cfg.MetadataDir),
		TempDir:             filepath.Join(os.TempDir(), "yt-dlp-go"),
		Format:              format,
		MergeOutput:         merge,
		ExtraArgs:           extra,
		WriteSubs:           writeSubs,
		AutoSubs:            u.autoSubs.Checked && writeSubs,
		SubLangs:            strings.TrimSpace(u.subLangs.Text),
		EmbedSubs:           u.embedSubs.Checked,
		WriteThumb:          writeThumb,
		EmbedThumb:          u.embedThumb.Checked,
		EmbedMetadata:       u.metaCheck.Checked,
		EmbedChapters:       u.chapCheck.Checked,
		WriteInfoJSON:       u.infoJSON.Checked,
		FilenameTemplate:    strings.TrimSpace(u.tmplEntry.Text),
		OrganizeByChannel:   u.organize.Checked,
		WindowsFilenames:    u.winNames.Checked,
		LimitRate:           strings.TrimSpace(u.limitRate.Text),
		Proxy:               strings.TrimSpace(u.proxy.Text),
		Retries:             atoiDefault(u.retries.Text, 10),
		FragmentRetries:     atoiDefault(u.retries.Text, 10),
		ConcurrentFragments: atoiDefault(u.fragments.Text, 4),
		Archive:             u.archive.Checked,
		NoOverwrite:         u.noOverwrite.Checked,
		Continue:            true,
		Concurrency:         u.cfg.Concurrency,
	}
	if opts.Retries <= 0 {
		opts.Retries = 10
		opts.FragmentRetries = 10
	}
	if opts.FilenameTemplate == "" {
		opts.FilenameTemplate = "%(title)s.%(ext)s"
	}
	if i := u.playlistSel.SelectedIndex(); i >= 0 && i < len(playlistValues) {
		opts.PlaylistMode = playlistValues[i]
		if opts.PlaylistMode == "items" {
			opts.PlaylistItems = strings.TrimSpace(u.playlistItems.Text)
		}
	}
	if i := u.cookiesSel.SelectedIndex(); i > 0 && i < len(cookieValues) {
		opts.CookiesBrowser = cookieValues[i]
	}
	if opts.Archive {
		if dir, err := settings.ConfigDir(); err == nil {
			opts.ArchiveFile = filepath.Join(dir, "download-archive.txt")
		} else {
			opts.ArchiveFile = filepath.Join(videoDir, ".download-archive.txt")
		}
	}

	id := fmt.Sprintf("%d", time.Now().UnixNano())
	if u.store != nil {
		u.store.Add(history.Item{
			ID:        id,
			URLs:      urls,
			Preset:    u.quality.Selected,
			OutputDir: videoDir,
			StartedAt: time.Now(),
			Status:    "running",
			Count:     len(urls),
		})
	}

	u.log.Clear()
	u.log.Add(fmt.Sprintf(i18n.T("Starting %d download(s) (%d in parallel) in %s"), len(urls), opts.Concurrency, videoDir))
	u.progress.SetValue(0)
	u.status.SetText(i18n.T("Running yt-dlp..."))
	u.setBusy(true)

	go func() {
		err := u.mgr.Run(context.Background(), opts, func(ev downloader.Event) {
			if ev.Line != "" {
				u.log.Add(ev.Line)
				fyne.Do(u.log.Flush)
			}
			if ev.Overall > 0 {
				ov := ev.Overall
				fyne.Do(func() { u.progress.SetValue(ov / 100.0) })
			}
			if ev.Done {
				n := ev.Index + 1
				if ev.Err != nil {
					u.log.Add(fmt.Sprintf(i18n.T("[%d/%d] failed: %v"), n, ev.Total, ev.Err))
				} else {
					u.log.Add(fmt.Sprintf(i18n.T("[%d/%d] completed"), n, ev.Total))
				}
				fyne.Do(u.log.Flush)
			}
		})

		status := "ok"
		errMsg := ""
		if err != nil {
			if errors.Is(err, context.Canceled) {
				status = "canceled"
			} else {
				status = "error"
				errMsg = err.Error()
			}
		}
		if u.store != nil {
			u.store.Update(id, func(it *history.Item) {
				it.Status = status
				it.FinishedAt = time.Now()
				it.Error = errMsg
			})
		}

		fyne.Do(func() {
			u.setBusy(false)
			u.progress.SetValue(0)
			u.refreshHistory()
			switch status {
			case "ok":
				u.status.SetText(i18n.T("Finished."))
				u.log.Add(i18n.T("END."))
				u.notify("yt-dlp-go", i18n.T("Downloads finished."))
			case "canceled":
				u.status.SetText(i18n.T("Cancelled."))
				u.log.Add(i18n.T("CANCELLED."))
			default:
				u.status.SetText(fmt.Sprintf(i18n.T("Finished with error: %s"), errMsg))
				u.log.Add(fmt.Sprintf(i18n.T("ERROR: %s"), errMsg))
				u.notify("yt-dlp-go", fmt.Sprintf(i18n.T("Download failed: %s"), errMsg))
			}
			u.log.Flush()
		})
	}()
}

func (u *appUI) openFormatsDialog() {
	urls := parseURLs(u.urlEntry.Text)
	if len(urls) == 0 {
		dialog.ShowInformation(i18n.T("Warning"), i18n.T("Enter a URL to list formats."), u.win)
		return
	}
	url := urls[0]
	bin := u.cfg.YtdlpPath
	if bin == "" {
		if b, err := downloader.Binary(); err == nil {
			bin = b
		}
	}
	cb := ""
	if i := u.cookiesSel.SelectedIndex(); i > 0 && i < len(cookieValues) {
		cb = cookieValues[i]
	}
	u.status.SetText(i18n.T("Fetching formats..."))

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		info, err := downloader.ListFormats(ctx, bin, url, cb, "")
		fyne.Do(func() {
			if err != nil {
				u.status.SetText(i18n.T("Failed to fetch formats."))
				dialog.ShowError(err, u.win)
				return
			}
			u.status.SetText(i18n.T("Formats loaded."))
			u.showFormats(info)
		})
	}()
}

func (u *appUI) showFormats(info *downloader.MediaInfo) {
	bestAudio := info.BestAudio()
	labels := []string{i18n.T("Automatic (preset)")}
	values := []string{""}
	for _, f := range info.Formats {
		if !f.HasVideo() {
			continue
		}
		val := f.ID
		if !f.HasAudio() && bestAudio != nil {
			val = f.ID + "+" + bestAudio.ID
		}
		lbl := fmt.Sprintf("%s | %s | %s | %s", f.Resolution, f.Ext, f.VCodec, humanSize(f.Filesize))
		if f.Note != "" {
			lbl += " | " + f.Note
		}
		labels = append(labels, lbl)
		values = append(values, val)
	}
	for _, f := range info.Formats {
		if f.HasVideo() || !f.HasAudio() {
			continue
		}
		lbl := fmt.Sprintf("%s | %s | %s | %.0fkbps | %s", i18n.T("Audio"), f.Ext, f.ACodec, f.TBR, humanSize(f.Filesize))
		labels = append(labels, lbl)
		values = append(values, f.ID)
	}

	sel := widget.NewSelect(labels, nil)
	sel.SetSelectedIndex(0)
	title := info.Title
	if title == "" {
		title = i18n.T("video")
	}
	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("%s (%.0f min)", title, info.Duration/60)),
		sel,
	)
	d := dialog.NewCustomConfirm(i18n.T("Choose format"), i18n.T("Use"), i18n.T("Cancel"), content, func(ok bool) {
		if !ok {
			return
		}
		i := sel.SelectedIndex()
		if i <= 0 {
			u.customFormat = ""
			u.customLabel.SetText(i18n.T("Format: automatic (preset)"))
			return
		}
		u.customFormat = values[i]
		u.customLabel.SetText(fmt.Sprintf(i18n.T("Format: %s"), labels[i]))
	}, u.win)
	d.Resize(fyne.NewSize(720, 220))
	d.Show()
}
