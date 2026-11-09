package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"yt-dlp-go/internal/i18n"
)

type Format struct {
	ID         string
	Ext        string
	Resolution string
	VCodec     string
	ACodec     string
	Filesize   int64
	TBR        float64
	Note       string
}

func (f Format) HasVideo() bool { return f.VCodec != "" && f.VCodec != "none" }
func (f Format) HasAudio() bool { return f.ACodec != "" && f.ACodec != "none" }

type MediaInfo struct {
	Title    string
	Duration float64
	Formats  []Format
}

type rawInfo struct {
	Title    string  `json:"title"`
	Duration float64 `json:"duration"`
	Formats  []struct {
		FormatID       string  `json:"format_id"`
		Ext            string  `json:"ext"`
		Resolution     string  `json:"resolution"`
		VCodec         string  `json:"vcodec"`
		ACodec         string  `json:"acodec"`
		Filesize       int64   `json:"filesize"`
		FilesizeApprox int64   `json:"filesize_approx"`
		TBR            float64 `json:"tbr"`
		FormatNote     string  `json:"format_note"`
	} `json:"formats"`
}

func ListFormats(ctx context.Context, bin, url, cookiesBrowser, cookiesProfile string) (*MediaInfo, error) {
	if bin == "" {
		b, err := Binary()
		if err != nil {
			return nil, err
		}
		bin = b
	}
	args := []string{"-J", "--no-playlist", "--ignore-config", "--no-warnings"}
	if cookiesBrowser != "" {
		cb := cookiesBrowser
		if cookiesProfile != "" {
			cb = cb + ":" + cookiesProfile
		}
		args = append(args, "--cookies-from-browser", cb)
	}
	args = append(args, url)

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = runEnv()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg == "" {
				msg = err.Error()
			}
			return nil, fmt.Errorf("%s", msg)
		}
		return nil, err
	}

	var ri rawInfo
	if err := json.Unmarshal(out, &ri); err != nil {
		return nil, fmt.Errorf(i18n.T("invalid json: %w"), err)
	}
	info := &MediaInfo{Title: ri.Title, Duration: ri.Duration}
	for _, f := range ri.Formats {
		size := f.Filesize
		if size == 0 {
			size = f.FilesizeApprox
		}
		info.Formats = append(info.Formats, Format{
			ID:         f.FormatID,
			Ext:        f.Ext,
			Resolution: f.Resolution,
			VCodec:     f.VCodec,
			ACodec:     f.ACodec,
			Filesize:   size,
			TBR:        f.TBR,
			Note:       f.FormatNote,
		})
	}
	return info, nil
}

func (m *MediaInfo) BestAudio() *Format {
	var best *Format
	for i := range m.Formats {
		f := &m.Formats[i]
		if !f.HasAudio() || f.HasVideo() {
			continue
		}
		if best == nil || f.TBR > best.TBR {
			best = f
		}
	}
	return best
}
