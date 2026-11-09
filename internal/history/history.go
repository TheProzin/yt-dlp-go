package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"yt-dlp-go/internal/i18n"
	"yt-dlp-go/internal/settings"
)

const schemaVersion = 1
const maxItems = 200

type Item struct {
	ID         string    `json:"id"`
	URLs       []string  `json:"urls"`
	Preset     string    `json:"preset"`
	OutputDir  string    `json:"output_dir"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
	Count      int       `json:"count"`
}

type fileFormat struct {
	SchemaVersion int    `json:"schema_version"`
	Items         []Item `json:"items"`
}

type Store struct {
	mu    sync.Mutex
	path  string
	lock  *os.File
	items []Item
}

func Open() (*Store, error) {
	dir, err := settings.ConfigDir()
	if err != nil {
		return nil, err
	}
	lf, err := os.OpenFile(filepath.Join(dir, ".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lf.Close()
		return nil, errors.New(i18n.T("another yt-dlp-go instance is already running"))
	}
	s := &Store{path: filepath.Join(dir, "history.json"), lock: lf}
	s.load()
	return s, nil
}

func (s *Store) load() {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var f fileFormat
	if err := json.Unmarshal(b, &f); err != nil {
		return
	}
	s.items = f.Items
	changed := false
	for i := range s.items {
		if s.items[i].Status == "running" {
			s.items[i].Status = "interrompido"
			s.items[i].FinishedAt = time.Now()
			changed = true
		}
	}
	if changed {
		_ = s.saveLocked()
	}
}

func (s *Store) saveLocked() error {
	f := fileFormat{SchemaVersion: schemaVersion, Items: s.items}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) Add(it Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append([]Item{it}, s.items...)
	if len(s.items) > maxItems {
		s.items = s.items[:maxItems]
	}
	_ = s.saveLocked()
}

func (s *Store) Update(id string, fn func(*Item)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			fn(&s.items[i])
			_ = s.saveLocked()
			return
		}
	}
}

func (s *Store) Items() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out
}

func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lock != nil {
		_ = syscall.Flock(int(s.lock.Fd()), syscall.LOCK_UN)
		_ = s.lock.Close()
		s.lock = nil
	}
}
