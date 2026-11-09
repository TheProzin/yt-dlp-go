package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	VideoDir         string  `json:"video_dir"`
	SubtitleDir      string  `json:"subtitle_dir"`
	ThumbnailDir     string  `json:"thumbnail_dir"`
	MetadataDir      string  `json:"metadata_dir"`
	PresetIndex      int     `json:"preset_index"`
	CookiesBrowser   string  `json:"cookies_browser"`
	YtdlpPath        string  `json:"ytdlp_path"`
	Language         string  `json:"language"`
	WindowWidth      float32 `json:"window_width"`
	WindowHeight     float32 `json:"window_height"`
	FilenameTemplate string  `json:"filename_template"`
	Concurrency      int     `json:"concurrency"`
}

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "yt-dlp-go")
	old := filepath.Join(base, "YtDlpGo") // legacy config dir (migrated automatically)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if _, oldErr := os.Stat(old); oldErr == nil {
			_ = os.Rename(old, dir)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func Path() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

func Default() Settings {
	home, _ := os.UserHomeDir()
	video := filepath.Join(home, "Vídeos")
	if _, err := os.Stat(video); err != nil {
		video = filepath.Join(home, "Videos")
	}
	return Settings{
		VideoDir:         video,
		SubtitleDir:      "legendas",
		ThumbnailDir:     "miniaturas",
		MetadataDir:      "metadados",
		PresetIndex:      0,
		CookiesBrowser:   "chrome",
		Language:         "en",
		WindowWidth:      900,
		WindowHeight:     760,
		FilenameTemplate: "%(title)s.%(ext)s",
		Concurrency:      5,
	}
}

func Load() Settings {
	s := Default()
	p, err := Path()
	if err != nil {
		return s
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	if s.Language == "" {
		s.Language = "en"
	}
	if s.WindowWidth < 400 {
		s.WindowWidth = 900
	}
	if s.WindowHeight < 300 {
		s.WindowHeight = 760
	}
	if s.Concurrency < 1 {
		s.Concurrency = 5
	}
	if s.Concurrency > 10 {
		s.Concurrency = 10
	}
	return s
}

func Save(s Settings) error {
	p, err := Path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func ResolveDir(base, dir string) string {
	if dir == "" {
		return base
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(base, dir)
}
