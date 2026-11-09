package deps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"yt-dlp-go/internal/downloader"
	"yt-dlp-go/internal/i18n"
)

type Status struct {
	Name    string
	OK      bool
	Version string
	Detail  string
}

func envWithDeno() []string {
	env := os.Environ()
	home, _ := os.UserHomeDir()
	deno := filepath.Join(home, ".deno/bin")
	path := os.Getenv("PATH")
	if !strings.Contains(path, deno) {
		env = append(env, "PATH="+deno+string(os.PathListSeparator)+path)
	}
	return env
}

func versionOf(bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Env = envWithDeno()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	line := strings.TrimSpace(string(out))
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	return line, nil
}

func findDeno() string {
	home, _ := os.UserHomeDir()
	cands := []string{filepath.Join(home, ".deno/bin/deno"), filepath.Join(home, ".local/bin/deno")}
	for _, c := range cands {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	if p, err := exec.LookPath("deno"); err == nil {
		return p
	}
	return ""
}

func findChrome() string {
	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "brave-browser", "microsoft-edge", "vivaldi"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func Check(ytdlpPath, videoDir string) []Status {
	var out []Status

	bin := ytdlpPath
	if bin == "" {
		if b, err := downloader.Binary(); err == nil {
			bin = b
		}
	}
	if bin == "" {
		out = append(out, Status{Name: "yt-dlp", OK: false, Detail: i18n.T("not found")})
	} else if v, err := versionOf(bin); err != nil {
		out = append(out, Status{Name: "yt-dlp", OK: false, Version: bin, Detail: err.Error()})
	} else {
		out = append(out, Status{Name: "yt-dlp", OK: true, Version: v, Detail: bin})
	}

	if p, err := exec.LookPath("ffmpeg"); err != nil {
		out = append(out, Status{Name: "ffmpeg", OK: false, Detail: i18n.T("not found (needed to merge/extract audio)")})
	} else if v, err := versionOf(p, "-version"); err != nil {
		out = append(out, Status{Name: "ffmpeg", OK: false, Version: p, Detail: err.Error()})
	} else {
		out = append(out, Status{Name: "ffmpeg", OK: true, Version: v, Detail: p})
	}

	if p := findDeno(); p == "" {
		out = append(out, Status{Name: "deno", OK: false, Detail: i18n.T("not found (enables high resolutions in yt-dlp)")})
	} else if v, err := versionOf(p, "--version"); err != nil {
		out = append(out, Status{Name: "deno", OK: false, Version: p, Detail: err.Error()})
	} else {
		out = append(out, Status{Name: "deno", OK: true, Version: v, Detail: p})
	}

	if p := findChrome(); p == "" {
		out = append(out, Status{Name: "Chrome", OK: false, Detail: i18n.T("not found (needed for login cookies)")})
	} else {
		out = append(out, Status{Name: "Chrome", OK: true, Version: i18n.T("installed"), Detail: p})
	}

	if videoDir != "" {
		if err := os.MkdirAll(videoDir, 0o755); err != nil {
			out = append(out, Status{Name: i18n.T("Output folder"), OK: false, Detail: err.Error()})
		} else {
			var st syscall.Statfs_t
			if err := syscall.Statfs(videoDir, &st); err == nil {
				free := float64(st.Bavail) * float64(st.Bsize) / (1024 * 1024 * 1024)
				out = append(out, Status{Name: i18n.T("Free space"), OK: free > 1, Version: fmt.Sprintf("%.1f GB", free), Detail: videoDir})
			}
			test := filepath.Join(videoDir, ".yt-dlp-go-write-test")
			if err := os.WriteFile(test, []byte("ok"), 0o644); err != nil {
				out = append(out, Status{Name: i18n.T("Folder permission"), OK: false, Detail: err.Error()})
			} else {
				_ = os.Remove(test)
				out = append(out, Status{Name: i18n.T("Folder permission"), OK: true, Detail: i18n.T("writable")})
			}
		}
	}

	return out
}
