package replay

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
)

// Package-level accessors for config values used across the replay package.
// These provide the values from the control surface for use in rendering and encoding.
var (
	defaultCaptureInterval    = config.Load().Replay.CaptureIntervalMs
	defaultCaptureTailMs      = config.Load().Replay.CaptureTailMs
	defaultPresentationWidth  = config.Load().Replay.PresentationWidth
	defaultPresentationHeight = config.Load().Replay.PresentationHeight
	maxCaptureFrames          = config.Load().Replay.MaxCaptureFrames
)

func newReplayRendererConfig(log *logrus.Logger, recordingsRoot string) *ReplayRenderer {
	cfg := config.Load()
	ffmpegPath := detectFFmpegBinary()
	client := &http.Client{Timeout: cfg.Replay.RenderTimeout}
	exportPageURL := strings.TrimSpace(os.Getenv("BAS_EXPORT_PAGE_URL"))
	if exportPageURL == "" {
		exportPageURL = strings.TrimSpace(os.Getenv("BAS_UI_EXPORT_URL"))
	}
	if exportPageURL == "" {
		baseURL := strings.TrimSpace(os.Getenv("BAS_UI_BASE_URL"))
		if baseURL != "" {
			trimmed := strings.TrimRight(baseURL, "/")
			exportPageURL = trimmed + "/export/composer.html"
		}
	}
	if exportPageURL == "" {
		scheme := strings.TrimSpace(os.Getenv("UI_SCHEME"))
		if scheme == "" {
			scheme = "http"
		}
		host := strings.TrimSpace(os.Getenv("UI_HOST"))
		if host == "" {
			host = "127.0.0.1"
		}
		port := strings.TrimSpace(os.Getenv("UI_PORT"))
		path := strings.TrimSpace(os.Getenv("BAS_EXPORT_PAGE_PATH"))
		if path == "" {
			path = "/export/composer.html"
		} else if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		if port != "" {
			exportPageURL = fmt.Sprintf("%s://%s:%s%s", scheme, host, port, path)
		} else {
			exportPageURL = fmt.Sprintf("%s://%s%s", scheme, host, path)
		}
	}
	// Allow env var override for backward compatibility
	captureInterval := cfg.Replay.CaptureIntervalMs
	if intervalEnv := strings.TrimSpace(os.Getenv("BAS_EXPORT_FRAME_INTERVAL_MS")); intervalEnv != "" {
		if parsed, err := strconv.Atoi(intervalEnv); err == nil && parsed > 0 {
			captureInterval = parsed
		}
	}
	apiScheme := strings.TrimSpace(os.Getenv("API_SCHEME"))
	if apiScheme == "" {
		apiScheme = "http"
	}
	apiHost := strings.TrimSpace(os.Getenv("API_HOST"))
	if apiHost == "" {
		apiHost = "127.0.0.1"
	}
	apiPort := strings.TrimSpace(os.Getenv("API_PORT"))
	apiBaseURL := fmt.Sprintf("%s://%s", apiScheme, apiHost)
	if apiPort != "" {
		apiBaseURL = fmt.Sprintf("%s://%s:%s", apiScheme, apiHost, apiPort)
	}
	return &ReplayRenderer{
		log:               log,
		recordingsRoot:    recordingsRoot,
		ffmpegPath:        ffmpegPath,
		httpClient:        client,
		exportPageURL:     exportPageURL,
		captureIntervalMs: captureInterval,
		apiBaseURL:        apiBaseURL,
	}
}

func detectFFmpegBinary() string {
	if custom := strings.TrimSpace(os.Getenv("FFMPEG_BIN")); custom != "" {
		return custom
	}
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return "ffmpeg"
	}
	defaultPath := filepath.Join(os.Getenv("HOME"), ".ffmpeg", "bin", "ffmpeg")
	return defaultPath
}

// DetectFFmpegBinary exposes the ffmpeg binary resolution used by replay rendering.
func DetectFFmpegBinary() string {
	return detectFFmpegBinary()
}
