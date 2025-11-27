package services

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultCaptureInterval    = 40
	defaultCaptureTailMs      = 320
	defaultPresentationWidth  = 1280
	defaultPresentationHeight = 720
	maxCaptureFrames          = 720
)

func newReplayRendererConfig(log *logrus.Logger, recordingsRoot string) *ReplayRenderer {
	ffmpegPath := detectFFmpegBinary()
	client := &http.Client{Timeout: 16 * time.Minute}
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
	captureInterval := defaultCaptureInterval
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
