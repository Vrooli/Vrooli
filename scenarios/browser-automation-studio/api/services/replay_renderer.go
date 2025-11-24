package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/browserless"
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
}

// RenderFormat enumerates supported render formats.
type RenderFormat string

const (
	// RenderFormatMP4 renders replay as MP4 video.
	RenderFormatMP4 RenderFormat = "mp4"
	// RenderFormatGIF renders replay as animated GIF.
	RenderFormatGIF RenderFormat = "gif"

	browserlessFunctionPath         = "/chrome/function"
	defaultCaptureInterval          = 40
	defaultCaptureTailMs            = 320
	defaultPresentationWidth        = 1280
	defaultPresentationHeight       = 720
	maxBrowserlessCaptureFrames     = 720
	browserlessTimeoutBufferMillis  = 120000
	perFrameBrowserlessBudgetMillis = 220
)

// NewReplayRenderer constructs a replay renderer with sane defaults.
func NewReplayRenderer(log *logrus.Logger, recordingsRoot string) *ReplayRenderer {
	ffmpegPath := detectFFmpegBinary()
	client := &http.Client{Timeout: 16 * time.Minute}
	browserlessURL, _ := browserless.ResolveURL(log, false)
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
		browserlessURL:    browserlessURL,
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
	if !r.shouldUseBrowserless() {
		return nil, errors.New("browserless export is required but not configured")
	}
	if err := r.validateExportPageURL(); err != nil {
		return nil, err
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

func (r *ReplayRenderer) shouldUseBrowserless() bool {
	return strings.TrimSpace(r.browserlessURL) != "" && strings.TrimSpace(r.exportPageURL) != ""
}

func (r *ReplayRenderer) validateExportPageURL() error {
	trimmed := strings.TrimSpace(r.exportPageURL)
	if trimmed == "" {
		return errors.New("export page url not configured; set BAS_EXPORT_PAGE_URL or BAS_UI_BASE_URL")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid export page url %q: %w", trimmed, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("export page url must use http or https scheme (got %q)", parsed.Scheme)
	}
	if parsed.Host == "" {
		return fmt.Errorf("export page url is missing host: %q", trimmed)
	}
	return nil
}

func (r *ReplayRenderer) renderWithBrowserless(ctx context.Context, spec *ReplayMovieSpec, format RenderFormat, filename string, captureInterval int) (*RenderedMedia, error) {
	capture, err := r.captureFramesWithBrowserless(ctx, spec, captureInterval)
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

// EstimateReplayRenderTimeout returns a conservative timeout budget for rendering a
// replay movie spec. The calculation is aligned with the Browserless capture
// budgeting inside renderWithBrowserless, ensuring handlers can provision a
// long-lived context without hard-coding large constants.
func EstimateReplayRenderTimeout(spec *ReplayMovieSpec) time.Duration {
	timeoutMs := browserlessTimeoutBufferMillis
	if spec != nil {
		captureInterval := spec.Playback.FrameIntervalMs
		if captureInterval <= 0 {
			captureInterval = defaultCaptureInterval
		}

		totalDuration := spec.Summary.TotalDurationMs
		if totalDuration <= 0 {
			totalDuration = spec.Playback.DurationMs
		}
		if totalDuration <= 0 && len(spec.Frames) > 0 {
			sum := 0
			for _, frame := range spec.Frames {
				duration := frame.DurationMs
				if duration <= 0 {
					duration = frame.HoldMs + frame.Enter.DurationMs + frame.Exit.DurationMs
				}
				if duration <= 0 {
					duration = captureInterval
				}
				sum += duration
			}
			totalDuration = sum
		}

		frameBudget := spec.Playback.TotalFrames
		if frameBudget <= 0 && captureInterval > 0 && totalDuration > 0 {
			frameBudget = int(math.Ceil(float64(totalDuration) / float64(captureInterval)))
		}
		if frameBudget <= 0 {
			frameBudget = len(spec.Frames)
		}
		if frameBudget < 1 {
			frameBudget = 1
		}

		timeoutMs += frameBudget * perFrameBrowserlessBudgetMillis
		// Provide additional breathing room for ffmpeg assembly and other overheads.
		timeoutMs += 60000
	}

	duration := time.Duration(timeoutMs) * time.Millisecond
	minimum := 3 * time.Minute
	maximum := 15 * time.Minute
	if duration < minimum {
		duration = minimum
	}
	if duration > maximum {
		duration = maximum
	}
	return duration
}

type browserlessCaptureFrame struct {
	Index       int     `json:"index"`
	TimestampMs int     `json:"timestampMs"`
	FrameIndex  int     `json:"frameIndex"`
	Progress    float64 `json:"progress"`
	Data        string  `json:"data"`
}

type browserlessCaptureResponse struct {
	Success bool                      `json:"success"`
	Error   string                    `json:"error"`
	Frames  []browserlessCaptureFrame `json:"frames"`
	FPS     int                       `json:"fps"`
	Width   int                       `json:"width"`
	Height  int                       `json:"height"`
}

func (r *ReplayRenderer) captureFramesWithBrowserless(ctx context.Context, spec *ReplayMovieSpec, captureInterval int) (*browserlessCaptureResponse, error) {
	if strings.TrimSpace(r.browserlessURL) == "" {
		return nil, errors.New("browserless url not configured")
	}
	if strings.TrimSpace(r.exportPageURL) == "" {
		return nil, errors.New("export page url not configured")
	}

	if captureInterval <= 0 {
		captureInterval = r.captureIntervalMs
	}

	payloadBytes, err := json.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal movie spec: %w", err)
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	jsonPayload := string(payloadBytes)

	canvasWidth := spec.Presentation.Canvas.Width
	canvasHeight := spec.Presentation.Canvas.Height
	if canvasWidth <= 0 {
		canvasWidth = spec.Presentation.Viewport.Width
	}
	if canvasHeight <= 0 {
		canvasHeight = spec.Presentation.Viewport.Height
	}
	if canvasWidth <= 0 {
		canvasWidth = defaultPresentationWidth
	}
	if canvasHeight <= 0 {
		canvasHeight = defaultPresentationHeight
	}
	deviceScale := spec.Presentation.DeviceScaleFactor
	if deviceScale <= 0 {
		deviceScale = 1
	}

	tailDuration := defaultCaptureTailMs
	if captureInterval > tailDuration {
		tailDuration = captureInterval
	}

	executionID := ""
	if spec.Execution.ExecutionID != uuid.Nil {
		executionID = spec.Execution.ExecutionID.String()
	}
	specID := executionID

	requestBody := map[string]any{
		"code": browserlessCaptureScript,
		"context": map[string]any{
			"payload":           encodedPayload,
			"payloadJson":       jsonPayload,
			"executionId":       executionID,
			"specId":            specID,
			"exportUrl":         r.exportPageURL,
			"frameInterval":     captureInterval,
			"tailDuration":      tailDuration,
			"jpegQuality":       82,
			"navigationTimeout": 90000,
			"apiBase":           r.apiBaseURL,
			"viewport": map[string]any{
				"width":             canvasWidth,
				"height":            canvasHeight,
				"deviceScaleFactor": deviceScale,
			},
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encode browserless request: %w", err)
	}

	endpoint := strings.TrimRight(r.browserlessURL, "/") + browserlessFunctionPath
	totalDuration := spec.Summary.TotalDurationMs
	frameBudget := 0
	if captureInterval > 0 && totalDuration > 0 {
		frameBudget = int(math.Ceil(float64(totalDuration) / float64(captureInterval)))
	}
	timeoutMillis := browserlessTimeoutBufferMillis
	if frameBudget > 0 {
		timeoutMillis += frameBudget * perFrameBrowserlessBudgetMillis
	}
	if timeoutMillis < browserlessTimeoutBufferMillis {
		timeoutMillis = browserlessTimeoutBufferMillis
	}
	captureCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMillis)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(captureCtx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create browserless request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("browserless capture request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read browserless response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("browserless returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var capture browserlessCaptureResponse
	if err := json.Unmarshal(responseBody, &capture); err != nil {
		return nil, fmt.Errorf("failed to decode browserless response: %w", err)
	}

	if !capture.Success && strings.TrimSpace(capture.Error) != "" {
		return nil, fmt.Errorf("browserless capture failed: %s", capture.Error)
	}

	return &capture, nil
}

func (r *ReplayRenderer) assembleVideoFromSequence(ctx context.Context, pattern string, fps int, outputPath string) error {
	if fps <= 0 {
		fps = 25
	}
	args := []string{
		"-y",
		"-framerate", strconv.Itoa(fps),
		"-start_number", "0",
		"-i", pattern,
		"-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2,format=yuv420p",
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level", "4.1",
		"-crf", "21",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg sequence assembly failed: %w (%s)", err, stderr.String())
	}
	return nil
}

func (r *ReplayRenderer) convertToGIF(ctx context.Context, inputPath, outputPath string, targetWidth int, fps int) error {
	if fps <= 0 {
		fps = 12
	}
	if targetWidth <= 0 {
		targetWidth = defaultPresentationWidth
	}
	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos", fps, targetWidth),
		outputPath,
	}
	cmd := exec.CommandContext(ctx, r.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg gif conversion failed: %w (%s)", err, stderr.String())
	}
	return nil
}

func sanitizeFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "browser-automation-replay"
	}
	trimmed = strings.ReplaceAll(trimmed, string(os.PathSeparator), "-")
	trimmed = strings.ReplaceAll(trimmed, "\x00", "")
	return trimmed
}

func defaultFilename(spec *ReplayMovieSpec, extension string) string {
	stem := "browser-automation-replay"
	if spec != nil {
		execID := spec.Execution.ExecutionID
		if execID != uuid.Nil {
			stem = fmt.Sprintf("browser-automation-replay-%s", execID.String()[:8])
		}
	}
	return fmt.Sprintf("%s.%s", stem, extension)
}

const browserlessCaptureScript = `export default async ({ page, context }) => {
  const config = context || {};
  const payloadBase64 = typeof config.payload === 'string' ? config.payload : '';
  const payloadJson = typeof config.payloadJson === 'string' ? config.payloadJson : '';
  const executionId = typeof config.executionId === 'string' ? config.executionId : '';
  const specId = typeof config.specId === 'string' ? config.specId : '';

  const decodeBase64 = (value) => {
    if (!value) {
      return '';
    }
    try {
      return Buffer.from(value, 'base64url').toString('utf8');
    } catch (error) {
      try {
        return Buffer.from(value, 'base64').toString('utf8');
      } catch (innerError) {
        console.warn('Failed to decode base64 payload', innerError);
        return '';
      }
    }
  };

  const serializedPayload = payloadJson && payloadJson.trim() !== ''
    ? payloadJson
    : decodeBase64(payloadBase64);

  if (!serializedPayload && !executionId && !specId) {
    throw new Error('Replay payload missing');
  }
  if (!config.exportUrl) {
    throw new Error('Replay export URL missing');
  }

  const frameInterval = Number(config.frameInterval || 40);
  const tailDuration = Number(config.tailDuration || 0);
  const jpegQuality = Number(config.jpegQuality || 82);
  const navigationTimeout = Number(config.navigationTimeout || 60000);
  const viewport = config.viewport || {};

  await page.setViewport({
    width: viewport.width || 1920,
    height: viewport.height || 1080,
    deviceScaleFactor: viewport.deviceScaleFactor || 1,
  });

  await page.evaluateOnNewDocument((bootstrap) => {
    try {
      window.__BAS_EXPORT_BOOTSTRAP__ = bootstrap;
    } catch (error) {
      console.warn('Failed to prime export bootstrap payload', error);
    }
    if (bootstrap && bootstrap.apiBase) {
      try {
        window.__BAS_EXPORT_API_BASE__ = String(bootstrap.apiBase);
      } catch (error) {
        console.warn('Failed to prime export API base', error);
      }
    }
  }, {
    payloadJson: serializedPayload,
    payloadBase64,
    apiBase: config.apiBase,
    executionId: executionId || null,
    specId: specId || executionId || null,
  });

  const url = new URL(config.exportUrl);
  url.searchParams.set('mode', 'capture');
  if (config.apiBase) {
    url.searchParams.set('apiBase', config.apiBase);
  }
  if (executionId) {
    url.searchParams.set('executionId', executionId);
  }
  if (specId) {
    url.searchParams.set('specId', specId);
  }

  await page.goto(url.toString(), { waitUntil: 'networkidle0', timeout: navigationTimeout });

  await page.evaluate((bootstrap) => {
    try {
      window.__BAS_EXPORT_BOOTSTRAP__ = bootstrap;
      if (bootstrap && bootstrap.apiBase) {
        window.__BAS_EXPORT_API_BASE__ = String(bootstrap.apiBase);
      }
    } catch (error) {
      console.warn('Failed to propagate export bootstrap payload', error);
    }
  }, {
    payloadJson: serializedPayload,
    payloadBase64,
    apiBase: config.apiBase,
    executionId: executionId || null,
    specId: specId || executionId || null,
  });

  await page.waitForFunction(() => window.basExport && window.basExport.ready === true, { timeout: 30000 });

  const metadata = await page.evaluate(() => window.basExport.getMetadata());
  const rect = await page.evaluate(() => window.basExport.getViewportRect());

  const canvasWidth = Number(
    metadata && typeof metadata.canvasWidth === 'number'
      ? metadata.canvasWidth
      : rect && rect.width
  );
  const canvasHeight = Number(
    metadata && typeof metadata.canvasHeight === 'number'
      ? metadata.canvasHeight
      : rect && rect.height
  );

  const clip = {
    x: Math.max(0, Math.floor((rect && rect.x) || 0)),
    y: Math.max(0, Math.floor((rect && rect.y) || 0)),
    width: Math.max(1, Math.floor(Number.isFinite(canvasWidth) ? canvasWidth : (rect && rect.width) || 0)),
    height: Math.max(1, Math.floor(Number.isFinite(canvasHeight) ? canvasHeight : (rect && rect.height) || 0)),
  };

  if (clip.width <= 0 || clip.height <= 0) {
    throw new Error('Invalid replay viewport dimensions');
  }

  const totalDuration = Number(metadata && metadata.totalDurationMs ? metadata.totalDurationMs : 0) + Math.max(0, tailDuration);
  const interval = Math.max(12, Math.floor(frameInterval));
  const frames = [];

  for (let timestamp = 0; timestamp <= totalDuration; timestamp += interval) {
    await page.evaluate((ms) => window.basExport.seekTo(ms), timestamp);
    const state = await page.evaluate(() => window.basExport.getCurrentState());
    const data = await page.screenshot({
      encoding: 'base64',
      type: 'jpeg',
      quality: jpegQuality,
      optimizeForSpeed: true,
      clip,
    });
    frames.push({
      index: frames.length,
      timestampMs: Math.round(timestamp),
      frameIndex: state && typeof state.frameIndex === 'number' ? state.frameIndex : 0,
      progress: state && typeof state.progress === 'number' ? state.progress : 0,
      data,
    });
  }

  const fps = Math.max(1, Math.round(1000 / interval));

  return {
    success: true,
    frames,
    fps,
    width: clip.width,
    height: clip.height,
  };
};`
