package replay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestReplayRendererRenderRequiresBrowserless(t *testing.T) {
	renderer := NewReplayRenderer(logrus.New(), "")
	renderer.browserlessURL = "" // ensure unset
	renderer.exportPageURL = ""

	spec := &ReplayMovieSpec{
		Execution: ExportExecutionMetadata{ExecutionID: uuid.New()},
		Frames:    []ExportFrame{{Index: 0, DurationMs: 100, HoldMs: 0}},
		Summary:   ExportSummary{TotalDurationMs: 100, FrameCount: 1},
	}

	_, err := renderer.Render(context.Background(), spec, RenderFormatMP4, "test.mp4")
	if err == nil {
		t.Fatalf("expected error when browserless configuration missing")
	}
	if err.Error() != "browserless export is required but not configured" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestCaptureFramesWithBrowserless(t *testing.T) {
	logger := logrus.New()
	renderer := NewReplayRenderer(logger, t.TempDir())

	const jpegData = "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"

	spec := &ReplayMovieSpec{
		Version:     "test",
		GeneratedAt: time.Now().UTC(),
		Execution: ExportExecutionMetadata{
			ExecutionID:   uuid.New(),
			WorkflowID:    uuid.New(),
			WorkflowName:  "Integration Test",
			Status:        "completed",
			StartedAt:     time.Now().Add(-2 * time.Minute),
			CompletedAt:   ptr(time.Now()),
			Progress:      100,
			TotalDuration: 1200,
		},
		Theme:  ExportTheme{},
		Cursor: ExportCursorSpec{Style: "halo"},
		Decor:  ExportDecor{},
		Playback: ExportPlayback{
			FPS:             25,
			DurationMs:      1200,
			FrameIntervalMs: 40,
			TotalFrames:     30,
		},
		Presentation: ExportPresentation{
			Canvas:   ExportDimensions{Width: 1280, Height: 720},
			Viewport: ExportDimensions{Width: 1280, Height: 720},
		},
		CursorMotion: ExportCursorMotion{},
		Summary: ExportSummary{
			FrameCount:      1,
			TotalDurationMs: 1200,
		},
		Frames: []ExportFrame{
			{
				Index:             0,
				StepIndex:         0,
				NodeID:            "node-1",
				StepType:          "screenshot",
				Title:             "Step 1",
				Status:            "completed",
				StartOffsetMs:     0,
				DurationMs:        1200,
				ScreenshotAssetID: "asset-1",
				Viewport:          ExportDimensions{Width: 1280, Height: 720},
			},
		},
		Assets: []ExportAsset{
			{
				ID:     "asset-1",
				Type:   "screenshot",
				Source: "data:image/jpeg;base64," + jpegData,
				Width:  1280,
				Height: 720,
			},
		},
	}

	var received struct {
		Context struct {
			Payload       string `json:"payload"`
			PayloadJSON   string `json:"payloadJson"`
			ExportURL     string `json:"exportUrl"`
			FrameInterval int    `json:"frameInterval"`
			Viewport      struct {
				Width             int     `json:"width"`
				Height            int     `json:"height"`
				DeviceScaleFactor float64 `json:"deviceScaleFactor"`
			} `json:"viewport"`
		} `json:"context"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/chrome/function") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		response := browserlessCaptureResponse{
			Success: true,
			FPS:     25,
			Width:   1280,
			Height:  720,
			Frames: []browserlessCaptureFrame{
				{
					Index:       0,
					TimestampMs: 0,
					FrameIndex:  0,
					Progress:    0,
					Data:        jpegData,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	renderer.browserlessURL = server.URL
	renderer.exportPageURL = "http://example.com/export/composer.html"
	renderer.apiBaseURL = "http://example.com"

	resp, err := renderer.captureFramesWithBrowserless(context.Background(), spec, 40)
	if err != nil {
		t.Fatalf("captureFramesWithBrowserless returned error: %v", err)
	}

	if resp == nil || len(resp.Frames) != 1 {
		t.Fatalf("expected one captured frame, got %#v", resp)
	}
	if received.Context.Payload == "" || received.Context.PayloadJSON == "" {
		t.Fatalf("expected payloads to be forwarded: %#v", received.Context)
	}
	if received.Context.ExportURL == "" {
		t.Fatalf("expected export URL to be provided")
	}
	if received.Context.FrameInterval <= 0 {
		t.Fatalf("expected frame interval to be positive")
	}
	if received.Context.Viewport.Width != 1280 || received.Context.Viewport.Height != 720 {
		t.Fatalf("expected viewport to match presentation dimensions: %#v", received.Context.Viewport)
	}
	if received.Context.Viewport.DeviceScaleFactor <= 0 {
		t.Fatalf("expected device scale factor to default to positive value, got %f", received.Context.Viewport.DeviceScaleFactor)
	}
}

func TestEstimateReplayRenderTimeoutBounds(t *testing.T) {
	smallSpec := &ReplayMovieSpec{
		Playback: ExportPlayback{
			FrameIntervalMs: 40,
			TotalFrames:     5,
		},
		Summary: ExportSummary{TotalDurationMs: 200},
		Frames:  []ExportFrame{{Index: 0, DurationMs: 200}},
	}

	duration := EstimateReplayRenderTimeout(smallSpec)
	if duration < 3*time.Minute {
		t.Fatalf("expected minimum timeout of 3 minutes, got %s", duration)
	}
	if duration > 15*time.Minute {
		t.Fatalf("expected timeout to remain within 15 minutes, got %s", duration)
	}

	hugeSpec := &ReplayMovieSpec{
		Playback: ExportPlayback{
			FrameIntervalMs: 20,
			TotalFrames:     120000,
		},
		Summary: ExportSummary{TotalDurationMs: 2400000},
		Frames:  make([]ExportFrame, 0),
	}

	bigDuration := EstimateReplayRenderTimeout(hugeSpec)
	if bigDuration > 15*time.Minute {
		t.Fatalf("expected timeout capped at 15 minutes, got %s", bigDuration)
	}
	if bigDuration < 3*time.Minute {
		t.Fatalf("expected timeout to exceed minimum for large specs, got %s", bigDuration)
	}
}

func ptr[T any](v T) *T {
	return &v
}
