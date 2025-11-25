// Package services provides rendering capabilities for browser automation replay artifacts.
//
// The ReplayRenderer orchestrates video and GIF generation from replay movie specifications.
// It delegates to specialized modules for capture (Browserless/Playwright), media processing (FFmpeg),
// and utility operations (filename generation, timeout estimation).
//
// Module Organization:
//   - replay_renderer.go (this file): Core types, orchestration, and main Render method
//   - replay_renderer_browserless.go: Browserless capture client and JavaScript execution script
//   - replay_renderer_ffmpeg.go: FFmpeg video assembly and GIF conversion
//   - replay_renderer_utils.go: Filename sanitization and timeout estimation
//   - replay_renderer_config.go: Configuration and constructor
//   - replay_renderer_playwright_client.go: Playwright capture client (desktop engine)
package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// RenderedMedia represents a generated media artifact ready for download.
type RenderedMedia struct {
	Path        string
	Filename    string
	ContentType string

	cleanup func()
}

// Cleanup removes any temporary artifacts associated with the rendered media.
func (m *RenderedMedia) Cleanup() {
	if m == nil || m.cleanup == nil {
		return
	}
	m.cleanup()
}

// ReplayRenderer renders replay movie specs to video or gif assets using a browserless capture pipeline.
type ReplayRenderer struct {
	log               *logrus.Logger
	recordingsRoot    string
	ffmpegPath        string
	httpClient        *http.Client
	browserlessURL    string
	exportPageURL     string
	captureIntervalMs int
	apiBaseURL        string
	captureClient     replayCaptureClient
}

// RenderFormat enumerates supported render formats.
type RenderFormat string

const (
	// RenderFormatMP4 renders replay as MP4 video.
	RenderFormatMP4 RenderFormat = "mp4"
	// RenderFormatGIF renders replay as animated GIF.
	RenderFormatGIF RenderFormat = "gif"
)

// NewReplayRenderer constructs a replay renderer with sane defaults.
func NewReplayRenderer(log *logrus.Logger, recordingsRoot string) *ReplayRenderer {
	return newReplayRendererConfig(log, recordingsRoot)
}

// replayCaptureClient abstracts the capture transport so non-Browserless
// engines can be plugged in without rewriting the renderer pipeline.
type replayCaptureClient interface {
	Capture(ctx context.Context, spec *ReplayMovieSpec, captureInterval int) (*browserlessCaptureResponse, error)
}

// Render creates a media artifact for the provided replay movie spec.
func (r *ReplayRenderer) Render(ctx context.Context, spec *ReplayMovieSpec, format RenderFormat, filename string) (*RenderedMedia, error) {
	if spec == nil {
		return nil, errors.New("nil replay movie spec")
	}
	if len(spec.Frames) == 0 {
		return nil, errors.New("movie spec missing frames")
	}
	if format != RenderFormatMP4 && format != RenderFormatGIF {
		return nil, fmt.Errorf("unsupported render format %q", format)
	}

	if r.captureClient == nil {
		switch {
		case r.shouldUseBrowserless():
			r.captureClient = &browserlessCaptureClient{renderer: r}
		case os.Getenv("PLAYWRIGHT_DRIVER_URL") != "":
			r.captureClient = newPlaywrightCaptureClient(r.exportPageURL)
		default:
			r.captureClient = &browserlessCaptureClient{renderer: r} // preserve legacy behavior
		}
	}

	if _, ok := r.captureClient.(*browserlessCaptureClient); ok {
		if !r.shouldUseBrowserless() {
			return nil, errors.New("browserless export is required but not configured")
		}
		if err := r.validateExportPageURL(); err != nil {
			return nil, err
		}
	}
	if _, ok := r.captureClient.(*playwrightCaptureClient); ok {
		if strings.TrimSpace(r.exportPageURL) == "" {
			return nil, errors.New("export page url not configured; set BAS_EXPORT_PAGE_URL or BAS_UI_BASE_URL")
		}
	}

	captureInterval := r.captureIntervalMs
	if spec.Playback.FrameIntervalMs > 0 {
		captureInterval = spec.Playback.FrameIntervalMs
	}
	if spec.Summary.TotalDurationMs > 0 {
		targetInterval := int(math.Ceil(float64(spec.Summary.TotalDurationMs) / float64(maxBrowserlessCaptureFrames)))
		if targetInterval > captureInterval {
			captureInterval = targetInterval
		}
		if captureInterval < 20 {
			captureInterval = 20
		}
	}

	return r.renderWithBrowserless(ctx, spec, format, filename, captureInterval)
}

// WithCaptureClient allows callers to override the capture transport (e.g.,
// Playwright desktop engine) while reusing the renderer pipeline.
func (r *ReplayRenderer) WithCaptureClient(client replayCaptureClient) *ReplayRenderer {
	if client == nil {
		return r
	}
	r.captureClient = client
	return r
}

// renderWithBrowserless orchestrates the full pipeline: capture, decode, assemble, and optionally convert to GIF.
func (r *ReplayRenderer) renderWithBrowserless(ctx context.Context, spec *ReplayMovieSpec, format RenderFormat, filename string, captureInterval int) (*RenderedMedia, error) {
	if r.captureClient == nil {
		r.captureClient = &browserlessCaptureClient{renderer: r}
	}

	capture, err := r.captureClient.Capture(ctx, spec, captureInterval)
	if err != nil {
		return nil, err
	}
	if capture == nil || len(capture.Frames) == 0 {
		return nil, errors.New("browserless capture returned no frames")
	}

	tempRoot, err := os.MkdirTemp("", "bas-browserless-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	cleanup := func() {
		_ = os.RemoveAll(tempRoot)
	}

	framesDir := filepath.Join(tempRoot, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		cleanup()
		return nil, fmt.Errorf("failed to create frames directory: %w", err)
	}

	written := 0
	for idx, frame := range capture.Frames {
		data := strings.TrimSpace(frame.Data)
		if data == "" {
			continue
		}
		decoded, decodeErr := base64.StdEncoding.DecodeString(data)
		if decodeErr != nil {
			cleanup()
			return nil, fmt.Errorf("failed to decode captured frame %d: %w", idx, decodeErr)
		}
		filePath := filepath.Join(framesDir, fmt.Sprintf("frame-%05d.jpg", idx))
		if err := os.WriteFile(filePath, decoded, 0o644); err != nil {
			cleanup()
			return nil, fmt.Errorf("failed to persist captured frame %d: %w", idx, err)
		}
		written++
	}

	if written == 0 {
		cleanup()
		return nil, errors.New("browserless capture produced zero usable frames")
	}

	fps := capture.FPS
	if fps <= 0 {
		fps = int(math.Round(1000.0 / float64(defaultCaptureInterval)))
		if fps <= 0 {
			fps = 25
		}
	}

	sequencePattern := filepath.Join(framesDir, "frame-%05d.jpg")
	baseVideoPath := filepath.Join(tempRoot, "replay.mp4")
	if err := r.assembleVideoFromSequence(ctx, sequencePattern, fps, baseVideoPath); err != nil {
		cleanup()
		return nil, err
	}

	finalPath := baseVideoPath
	contentType := "video/mp4"
	if format == RenderFormatGIF {
		gifPath := filepath.Join(tempRoot, "replay.gif")
		gifWidth := capture.Width
		if gifWidth <= 0 {
			gifWidth = capture.Height
		}
		if gifWidth <= 0 {
			gifWidth = defaultPresentationWidth
		}
		gifFPS := capture.FPS
		if err := r.convertToGIF(ctx, baseVideoPath, gifPath, gifWidth, gifFPS); err != nil {
			cleanup()
			return nil, err
		}
		finalPath = gifPath
		contentType = "image/gif"
	}

	if filename == "" {
		filename = defaultFilename(spec, string(format))
	}
	filename = sanitizeFilename(filename)

	return &RenderedMedia{
		Path:        finalPath,
		Filename:    filename,
		ContentType: contentType,
		cleanup:     cleanup,
	}, nil
}
