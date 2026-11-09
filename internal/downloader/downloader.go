package downloader

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"yt-dlp-go/internal/i18n"
)

var percentRe = regexp.MustCompile(`\[download\]\s+([0-9]+(?:\.[0-9]+)?)%`)

type Event struct {
	Line    string
	Percent float64
	Overall float64
	Total   int
	Index   int
	Err     error
	Done    bool
}

type JobConfig struct {
	URLs []string

	VideoDir     string
	SubtitleDir  string
	ThumbnailDir string
	MetadataDir  string
	TempDir      string
	ArchiveFile  string

	Format      string
	MergeOutput string
	ExtraArgs   []string

	CookiesBrowser string

	WriteSubs  bool
	AutoSubs   bool
	SubLangs   string
	EmbedSubs  bool
	WriteThumb bool
	EmbedThumb bool

	EmbedMetadata bool
	EmbedChapters bool
	WriteInfoJSON bool

	FilenameTemplate  string
	OrganizeByChannel bool
	WindowsFilenames  bool

	PlaylistMode  string
	PlaylistItems string

	LimitRate           string
	Proxy               string
	Retries             int
	FragmentRetries     int
	ConcurrentFragments int

	Archive     bool
	NoOverwrite bool
	Continue    bool

	Concurrency int
}

type Manager struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
}

func New() *Manager { return &Manager{} }

func (m *Manager) Cancel() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
}

func Binary() (string, error) {
	if b := os.Getenv("YTDLP_BIN"); b != "" {
		return b, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local/bin/yt-dlp"),
		filepath.Join(home, ".config/yt-dlg/yt-dlp.real"),
		filepath.Join(home, ".config/yt-dlg/yt-dlp"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c, nil
		}
	}
	return exec.LookPath("yt-dlp")
}

func runEnv() []string {
	env := os.Environ()
	home, _ := os.UserHomeDir()
	deno := filepath.Join(home, ".deno/bin")
	path := os.Getenv("PATH")
	if !strings.Contains(path, deno) {
		env = append(env, "PATH="+deno+string(os.PathListSeparator)+path)
	}
	return env
}

func buildArgs(o JobConfig) []string {
	args := []string{"--newline", "--ignore-config"}

	tmpl := o.FilenameTemplate
	if tmpl == "" {
		tmpl = "%(title)s.%(ext)s"
	}
	if o.OrganizeByChannel {
		tmpl = filepath.Join("%(uploader)s", tmpl)
	}
	args = append(args, "-P", "home:"+o.VideoDir)
	args = append(args, "-o", tmpl)

	if o.TempDir != "" {
		args = append(args, "-P", "temp:"+o.TempDir)
	}
	if o.Format != "" {
		args = append(args, "-f", o.Format)
	}
	if o.MergeOutput != "" {
		args = append(args, "--merge-output-format", o.MergeOutput)
	}
	if o.Continue {
		args = append(args, "--continue")
	}
	if o.Retries > 0 {
		args = append(args, "--retries", strconv.Itoa(o.Retries))
	}
	if o.FragmentRetries > 0 {
		args = append(args, "--fragment-retries", strconv.Itoa(o.FragmentRetries))
	}
	if o.ConcurrentFragments > 0 {
		args = append(args, "--concurrent-fragments", strconv.Itoa(o.ConcurrentFragments))
	}
	if o.LimitRate != "" {
		args = append(args, "--limit-rate", o.LimitRate)
	}
	if o.Proxy != "" {
		args = append(args, "--proxy", o.Proxy)
	}
	if o.NoOverwrite {
		args = append(args, "--no-overwrites")
	}
	if o.WindowsFilenames {
		args = append(args, "--windows-filenames")
	}
	if o.Archive && o.ArchiveFile != "" {
		args = append(args, "--download-archive", o.ArchiveFile)
	}

	if o.WriteSubs {
		args = append(args, "--write-subs")
	}
	if o.AutoSubs {
		args = append(args, "--write-auto-subs")
	}
	if o.WriteSubs || o.AutoSubs {
		langs := o.SubLangs
		if langs == "" {
			langs = "pt.*,pt,en.*,en"
		}
		args = append(args, "--sub-langs", langs)
		if o.EmbedSubs {
			args = append(args, "--embed-subs")
		}
		if o.SubtitleDir != "" {
			args = append(args, "-P", "subtitle:"+o.SubtitleDir)
		}
		args = append(args, "-o", "subtitle:"+tmpl)
	}
	if o.WriteThumb {
		args = append(args, "--write-thumbnail")
		if o.EmbedThumb {
			args = append(args, "--embed-thumbnail")
		}
		if o.ThumbnailDir != "" {
			args = append(args, "-P", "thumbnail:"+o.ThumbnailDir)
		}
		args = append(args, "-o", "thumbnail:"+tmpl)
	}
	if o.EmbedMetadata {
		args = append(args, "--embed-metadata")
	}
	if o.EmbedChapters {
		args = append(args, "--embed-chapters")
	}
	if o.WriteInfoJSON {
		args = append(args, "--write-info-json")
		if o.MetadataDir != "" {
			args = append(args, "-P", "infojson:"+o.MetadataDir)
			args = append(args, "-o", "infojson:"+tmpl)
		}
	}

	switch o.PlaylistMode {
	case "yes":
		args = append(args, "--yes-playlist")
	case "items":
		args = append(args, "--yes-playlist")
		if o.PlaylistItems != "" {
			args = append(args, "--playlist-items", o.PlaylistItems)
		}
	default:
		args = append(args, "--no-playlist")
	}

	if o.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", o.CookiesBrowser)
	}

	args = append(args, o.ExtraArgs...)
	return args
}

func (m *Manager) Run(ctx context.Context, o JobConfig, emit func(Event)) error {
	bin, err := Binary()
	if err != nil {
		return fmt.Errorf(i18n.T("yt-dlp not found: %w"), err)
	}
	if len(o.URLs) == 0 {
		return errors.New(i18n.T("no URLs provided"))
	}
	if o.VideoDir == "" {
		return errors.New(i18n.T("output folder not defined"))
	}
	for _, dir := range []string{o.VideoDir, o.SubtitleDir, o.ThumbnailDir, o.MetadataDir} {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf(i18n.T("could not create folder %s: %w"), dir, err)
		}
	}

	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return errors.New(i18n.T("a download is already running"))
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.running = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.running = false
		m.cancel = nil
		m.mu.Unlock()
		cancel()
	}()

	concurrency := o.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 10 {
		concurrency = 10
	}

	var emitMu sync.Mutex
	safeEmit := func(ev Event) {
		emitMu.Lock()
		emit(ev)
		emitMu.Unlock()
	}

	items := expandItems(runCtx, bin, o, safeEmit)
	total := len(items)
	if total == 0 {
		return errors.New(i18n.T("no videos found"))
	}
	safeEmit(Event{Line: fmt.Sprintf(i18n.T("Found %d video(s)"), total)})

	var pmu sync.Mutex
	percents := make([]float64, total)
	overall := func(idx int, pct float64) float64 {
		pmu.Lock()
		defer pmu.Unlock()
		percents[idx] = pct
		sum := 0.0
		for _, v := range percents {
			sum += v
		}
		return sum / float64(total)
	}

	jobs := make(chan int)
	var wg sync.WaitGroup
	var fmu sync.Mutex
	failed := 0
	var firstErr error
	markFailed := func(e error) {
		fmu.Lock()
		failed++
		if firstErr == nil {
			firstErr = e
		}
		fmu.Unlock()
	}

	worker := func() {
		defer wg.Done()
		for idx := range jobs {
			if runCtx.Err() != nil {
				return
			}
			it := items[idx]
			cfg := o
			if it.Expanded {
				cfg.PlaylistMode = "no"
			}
			args := append(buildArgs(cfg), it.URL)
			cmd := exec.Command(bin, args...)
			cmd.Env = runEnv()
			cmd.SysProcAttr = sysProcAttr()

			pr, pw, err := os.Pipe()
			if err != nil {
				markFailed(err)
				safeEmit(Event{Index: idx, Total: total, Err: err, Done: true, Overall: overall(idx, 100)})
				continue
			}
			cmd.Stdout = pw
			cmd.Stderr = pw

			if err := cmd.Start(); err != nil {
				_ = pw.Close()
				_ = pr.Close()
				markFailed(err)
				safeEmit(Event{Index: idx, Total: total, Err: err, Done: true, Overall: overall(idx, 100)})
				continue
			}
			_ = pw.Close()

			done := make(chan struct{})
			pid := cmd.Process.Pid
			go func() {
				select {
				case <-runCtx.Done():
					terminateProcess(pid)
				case <-done:
				}
			}()

			scanner := bufio.NewScanner(pr)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				if mm := percentRe.FindStringSubmatch(line); mm != nil {
					var pct float64
					fmt.Sscanf(mm[1], "%f", &pct)
					safeEmit(Event{Index: idx, Total: total, Percent: pct, Overall: overall(idx, pct)})
					continue
				}
				safeEmit(Event{Index: idx, Total: total, Line: line})
			}
			_ = pr.Close()

			waitErr := cmd.Wait()
			close(done)
			if waitErr != nil {
				markFailed(waitErr)
			}
			safeEmit(Event{Index: idx, Total: total, Err: waitErr, Done: true, Overall: overall(idx, 100)})
		}
	}

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go worker()
	}
	for i := range items {
		select {
		case jobs <- i:
		case <-runCtx.Done():
		}
		if runCtx.Err() != nil {
			break
		}
	}
	close(jobs)
	wg.Wait()

	if runCtx.Err() != nil {
		return runCtx.Err()
	}
	if failed > 0 {
		return fmt.Errorf(i18n.T("%d of %d downloads failed: %w"), failed, total, firstErr)
	}
	return nil
}

type workItem struct {
	URL      string
	Expanded bool
}

func expandItems(ctx context.Context, bin string, o JobConfig, emit func(Event)) []workItem {
	if o.PlaylistMode == "no" {
		items := make([]workItem, 0, len(o.URLs))
		for _, u := range o.URLs {
			items = append(items, workItem{URL: u})
		}
		return items
	}

	var items []workItem
	for _, u := range o.URLs {
		if ctx.Err() != nil {
			break
		}
		emit(Event{Line: fmt.Sprintf(i18n.T("Expanding playlist %s"), u)})
		urls, err := listPlaylistURLs(ctx, bin, o, u)
		if err != nil || len(urls) == 0 {
			items = append(items, workItem{URL: u})
			continue
		}
		for _, iu := range urls {
			items = append(items, workItem{URL: iu, Expanded: true})
		}
	}
	return items
}

func listPlaylistURLs(ctx context.Context, bin string, o JobConfig, url string) ([]string, error) {
	args := []string{"--flat-playlist", "--print", "%(url)s", "--ignore-config", "--no-warnings"}
	if o.PlaylistMode == "items" && o.PlaylistItems != "" {
		args = append(args, "--playlist-items", o.PlaylistItems)
	}
	if o.CookiesBrowser != "" {
		args = append(args, "--cookies-from-browser", o.CookiesBrowser)
	}
	args = append(args, url)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = runEnv()
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var urls []string
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			urls = append(urls, ln)
		}
	}
	return urls, nil
}
