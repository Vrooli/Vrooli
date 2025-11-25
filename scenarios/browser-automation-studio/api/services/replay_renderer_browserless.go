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
	"strings"
	"time"

	"github.com/google/uuid"
)

// browserlessCaptureFrame represents a single captured frame from Browserless.
type browserlessCaptureFrame struct {
	Index       int     `json:"index"`
	TimestampMs int     `json:"timestampMs"`
	FrameIndex  int     `json:"frameIndex"`
	Progress    float64 `json:"progress"`
	Data        string  `json:"data"`
}

// browserlessCaptureResponse contains the result of a Browserless capture operation.
type browserlessCaptureResponse struct {
	Success bool                      `json:"success"`
	Error   string                    `json:"error"`
	Frames  []browserlessCaptureFrame `json:"frames"`
	FPS     int                       `json:"fps"`
	Width   int                       `json:"width"`
	Height  int                       `json:"height"`
}

// browserlessCaptureClient implements replayCaptureClient using Browserless.
type browserlessCaptureClient struct {
	renderer *ReplayRenderer
}

// Capture executes a Browserless-based frame capture for the given movie spec.
func (c *browserlessCaptureClient) Capture(ctx context.Context, spec *ReplayMovieSpec, captureInterval int) (*browserlessCaptureResponse, error) {
	if c == nil || c.renderer == nil {
		return nil, fmt.Errorf("capture client not configured")
	}
	return c.renderer.captureFramesWithBrowserless(ctx, spec, captureInterval)
}

// captureFramesWithBrowserless orchestrates a Browserless /function call to capture replay frames.
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

// validateExportPageURL ensures the export page URL is valid and usable.
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

// shouldUseBrowserless checks if Browserless configuration is present.
func (r *ReplayRenderer) shouldUseBrowserless() bool {
	return strings.TrimSpace(r.browserlessURL) != "" && strings.TrimSpace(r.exportPageURL) != ""
}

// browserlessCaptureScript is the JavaScript function executed by Browserless to capture replay frames.
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
