package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/replay"
	"github.com/vrooli/browser-automation-studio/services/workflow"
)

type replayRendererStub struct {
	t            *testing.T
	tempFile     string
	lastCtx      context.Context
	receivedSpec *export.ReplayMovieSpec
	format       replay.RenderFormat
	filename     string
	invokedAt    time.Time
	deadline     time.Time
	hadDeadline  bool
}

func newReplayRendererStub(t *testing.T) *replayRendererStub {
	t.Helper()
	return &replayRendererStub{t: t}
}

func (s *replayRendererStub) Render(ctx context.Context, spec *export.ReplayMovieSpec, format replay.RenderFormat, filename string) (*replay.RenderedMedia, error) {
	s.t.Helper()
	file, err := os.CreateTemp("", "bas-replay-*.mp4")
	if err != nil {
		s.t.Fatalf("failed to create temp media file: %v", err)
	}
	if err := file.Close(); err != nil {
		s.t.Fatalf("failed to close temp media file: %v", err)
	}
	s.tempFile = file.Name()
	s.lastCtx = ctx
	s.receivedSpec = spec
	s.format = format
	s.filename = filename
	s.invokedAt = time.Now()
	if deadline, ok := ctx.Deadline(); ok {
		s.deadline = deadline
		s.hadDeadline = true
	}
	return &replay.RenderedMedia{
		Path:        s.tempFile,
		Filename:    filename,
		ContentType: "video/mp4",
	}, nil
}

func (s *replayRendererStub) cleanup() {
	if s.tempFile != "" {
		_ = os.Remove(s.tempFile)
	}
}

func TestPostExecutionExport_AllowsClientMovieSpec(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()

	baseSpec := &export.ReplayMovieSpec{
		Version:     "2025-11-07",
		GeneratedAt: time.Now().Add(-time.Minute),
		Execution: export.ExportExecutionMetadata{
			ExecutionID:   execID,
			WorkflowID:    wfID,
			WorkflowName:  "Demo Workflow",
			Status:        "completed",
			StartedAt:     time.Now().Add(-2 * time.Minute),
			TotalDuration: 3200,
		},
		Theme:  export.ExportTheme{},
		Cursor: export.ExportCursorSpec{},
		Decor: export.ExportDecor{
			ChromeTheme:     "aurora",
			BackgroundTheme: "aurora",
			CursorTheme:     "white",
		},
		Summary: export.ExportSummary{FrameCount: 1, TotalDurationMs: 1600},
		Frames: []export.ExportFrame{{
			Index:         0,
			StepIndex:     0,
			StepType:      "navigate",
			DurationMs:    1600,
			StartOffsetMs: 0,
		}},
	}

	incoming := &export.ReplayMovieSpec{
		Theme: export.ExportTheme{},
		Cursor: export.ExportCursorSpec{
			Style: "halo",
		},
		Decor: export.ExportDecor{
			ChromeTheme:     "sunset",
			BackgroundTheme: "sunset",
			CursorTheme:     "black",
		},
		Frames: []export.ExportFrame{{
			Index:         0,
			StepIndex:     0,
			StepType:      "navigate",
			DurationMs:    1200,
			StartOffsetMs: 0,
		}},
	}

	h := &Handler{
		workflowService: &workflowServiceMock{
			describeExecutionExportFn: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionExportPreview, error) {
				if id != execID {
					t.Fatalf("unexpected execution id %s", id)
				}
				return &workflow.ExecutionExportPreview{
					ExecutionID: execID,
					Status:      "ready",
					Message:     "ok",
					Package:     baseSpec,
				}, nil
			},
		},
		log: logrus.New(),
	}

	body := executionExportRequest{Format: "json", MovieSpec: incoming}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+execID.String()+"/export", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", execID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	resp := httptest.NewRecorder()
	h.PostExecutionExport(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var preview workflow.ExecutionExportPreview
	if err := json.Unmarshal(resp.Body.Bytes(), &preview); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if preview.Package == nil {
		t.Fatalf("expected package in response")
	}
	if preview.Package.Execution.ExecutionID != execID {
		t.Errorf("expected execution_id %s, got %s", execID, preview.Package.Execution.ExecutionID)
	}
	if preview.Package.Decor.ChromeTheme != "sunset" {
		t.Errorf("expected chrome theme sunset, got %s", preview.Package.Decor.ChromeTheme)
	}
	if preview.Package.Summary.FrameCount != 1 {
		t.Errorf("expected frame count 1, got %d", preview.Package.Summary.FrameCount)
	}
	if preview.Package.Summary.TotalDurationMs == 0 {
		t.Errorf("expected total duration to be populated")
	}
}

func TestPostExecutionExport_RejectsMismatchedSpec(t *testing.T) {
	execID := uuid.New()
	baseSpec := &export.ReplayMovieSpec{
		Execution: export.ExportExecutionMetadata{ExecutionID: execID},
		Frames:    []export.ExportFrame{{Index: 0, DurationMs: 1000}},
	}

	h := &Handler{
		workflowService: &workflowServiceMock{
			describeExecutionExportFn: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionExportPreview, error) {
				return &workflow.ExecutionExportPreview{ExecutionID: execID, Status: "ready", Package: baseSpec}, nil
			},
		},
		log: logrus.New(),
	}

	incoming := &export.ReplayMovieSpec{
		Execution: export.ExportExecutionMetadata{ExecutionID: uuid.New()},
		Frames:    []export.ExportFrame{{Index: 0, DurationMs: 1000}},
	}
	body := executionExportRequest{Format: "json", MovieSpec: incoming}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+execID.String()+"/export", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", execID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	resp := httptest.NewRecorder()
	h.PostExecutionExport(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestPostExecutionExport_ReturnsPreviewWhenNotReady(t *testing.T) {
	execID := uuid.New()
	h := &Handler{
		workflowService: &workflowServiceMock{
			describeExecutionExportFn: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionExportPreview, error) {
				return &workflow.ExecutionExportPreview{
					ExecutionID: execID,
					Status:      "pending",
					Message:     "Replay export pending – timeline frames not captured yet",
				}, nil
			},
		},
		log: logrus.New(),
	}

	body := executionExportRequest{Format: "json"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+execID.String()+"/export", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", execID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	resp := httptest.NewRecorder()
	h.PostExecutionExport(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	var preview workflow.ExecutionExportPreview
	if err := json.Unmarshal(resp.Body.Bytes(), &preview); err != nil {
		t.Fatalf("failed to decode preview: %v", err)
	}
	if preview.Status != "pending" {
		t.Fatalf("expected status pending, got %q", preview.Status)
	}
	if preview.Package != nil {
		t.Fatalf("expected no package when export is pending")
	}
}

func TestPostExecutionExport_RejectsMediaRequestWhenUnavailable(t *testing.T) {
	execID := uuid.New()
	h := &Handler{
		workflowService: &workflowServiceMock{
			describeExecutionExportFn: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionExportPreview, error) {
				return &workflow.ExecutionExportPreview{
					ExecutionID: execID,
					Status:      "unavailable",
					Message:     "Replay export unavailable – execution failed before capturing any steps",
				}, nil
			},
		},
		log: logrus.New(),
	}

	body := executionExportRequest{Format: "mp4"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+execID.String()+"/export", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", execID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	resp := httptest.NewRecorder()
	h.PostExecutionExport(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("export package unavailable")) {
		t.Fatalf("expected error response to mention unavailability, got %s", resp.Body.Bytes())
	}
}

func TestPostExecutionExport_UsesEstimatedTimeout(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	now := time.Now()
	completed := now

	baseSpec := &export.ReplayMovieSpec{
		Version:     "2025-11-07",
		GeneratedAt: now.Add(-time.Second),
		Execution: export.ExportExecutionMetadata{
			ExecutionID:   execID,
			WorkflowID:    wfID,
			WorkflowName:  "Timeout Demo",
			Status:        "completed",
			StartedAt:     now.Add(-45 * time.Second),
			CompletedAt:   &completed,
			Progress:      100,
			TotalDuration: 4000,
		},
		Theme: export.ExportTheme{
			BackgroundGradient: []string{"#0f172a", "#1d4ed8"},
			BackgroundPattern:  "aurora",
			AccentColor:        "#38BDF8",
			SurfaceColor:       "rgba(15,23,42,0.72)",
			AmbientGlow:        "rgba(56,189,248,0.22)",
			BrowserChrome: export.ExportBrowserChrome{
				Visible:     true,
				Variant:     "aurora",
				Title:       "Timeout Demo",
				ShowAddress: true,
				AccentColor: "#38BDF8",
			},
		},
		Cursor: export.ExportCursorSpec{
			Style:       "halo",
			AccentColor: "#38BDF8",
			Trail: export.ExportCursorTrail{
				Enabled: true,
				FadeMs:  650,
				Weight:  0.16,
				Opacity: 0.55,
			},
			ClickPulse: export.ExportClickPulse{
				Enabled:    true,
				Radius:     42,
				DurationMs: 420,
				Opacity:    0.65,
			},
			Scale:      1,
			InitialPos: "center",
			ClickAnim:  "pulse",
		},
		Decor: export.ExportDecor{
			ChromeTheme:          "aurora",
			BackgroundTheme:      "aurora",
			CursorTheme:          "white",
			CursorInitial:        "center",
			CursorClickAnimation: "pulse",
			CursorScale:          1,
		},
		Playback: export.ExportPlayback{
			FPS:             25,
			DurationMs:      4000,
			FrameIntervalMs: 40,
			TotalFrames:     100,
		},
		Presentation: export.ExportPresentation{
			Canvas:            export.ExportDimensions{Width: 1920, Height: 1080},
			Viewport:          export.ExportDimensions{Width: 1440, Height: 900},
			BrowserFrame:      export.ExportFrameRect{X: 0, Y: 0, Width: 1920, Height: 1080, Radius: 24},
			DeviceScaleFactor: 1,
		},
		CursorMotion: export.ExportCursorMotion{
			SpeedProfile:    "easeInOut",
			PathStyle:       "linear",
			InitialPosition: "center",
			ClickAnimation:  "pulse",
			CursorScale:     1,
		},
		Frames: []export.ExportFrame{{
			Index:             0,
			StepIndex:         0,
			StepType:          "navigate",
			DurationMs:        2000,
			StartOffsetMs:     0,
			ScreenshotAssetID: "asset-1",
		}},
		Assets: []export.ExportAsset{{
			ID:     "asset-1",
			Type:   "image/png",
			Source: "https://example.com/asset.png",
		}},
		Summary: export.ExportSummary{
			FrameCount:         1,
			ScreenshotCount:    1,
			TotalDurationMs:    2000,
			MaxFrameDurationMs: 2000,
		},
	}

	stub := newReplayRendererStub(t)
	defer stub.cleanup()

	h := &Handler{
		workflowService: &workflowServiceMock{
			describeExecutionExportFn: func(ctx context.Context, id uuid.UUID) (*workflow.ExecutionExportPreview, error) {
				return &workflow.ExecutionExportPreview{
					ExecutionID: execID,
					Status:      "ready",
					Package:     baseSpec,
				}, nil
			},
		},
		replayRenderer: stub,
		log:            logrus.New(),
	}

	body := executionExportRequest{Format: "mp4", FileName: "custom.mp4"}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions/"+execID.String()+"/export", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", execID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	resp := httptest.NewRecorder()
	h.PostExecutionExport(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if !stub.hadDeadline {
		t.Fatalf("expected render context to include deadline")
	}
	if stub.format != replay.RenderFormatMP4 {
		t.Fatalf("expected format mp4, got %s", stub.format)
	}
	if stub.filename != "custom.mp4" {
		t.Fatalf("expected filename custom.mp4, got %s", stub.filename)
	}

	expected := replay.EstimateReplayRenderTimeout(stub.receivedSpec)
	actual := stub.deadline.Sub(stub.invokedAt)
	if actual < expected-500*time.Millisecond || actual > expected+2*time.Second {
		t.Fatalf("render context deadline mismatch: expected ~%s, got %s", expected, actual)
	}
}
