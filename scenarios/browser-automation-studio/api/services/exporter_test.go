package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestBuildReplayMovieSpecGeneratesSpec(t *testing.T) {
	executionID := uuid.New()
	workflowID := uuid.New()

	exec := &database.Execution{
		ID:         executionID,
		WorkflowID: workflowID,
		Status:     "completed",
		StartedAt:  time.Now().Add(-2 * time.Minute),
		CompletedAt: func() *time.Time {
			ts := time.Now().Add(-time.Minute)
			return &ts
		}(),
		Progress: 100,
	}

	workflow := &database.Workflow{
		ID:   workflowID,
		Name: "Demo Journey",
	}

	timeline := &ExecutionTimeline{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Status:      "completed",
		Progress:    100,
		StartedAt:   exec.StartedAt,
		CompletedAt: exec.CompletedAt,
		Frames: []TimelineFrame{
			{
				StepIndex:       0,
				NodeID:          "node-1",
				StepType:        "navigate",
				Status:          "completed",
				DurationMs:      900,
				TotalDurationMs: 1400,
				FinalURL:        "https://example.com",
				Screenshot: &TimelineScreenshot{
					ArtifactID: "shot-1",
					URL:        "https://cdn.example.com/shot-1.png",
					Width:      1280,
					Height:     720,
				},
				CursorTrail:      []runtime.Point{{X: 640, Y: 360}, {X: 700, Y: 420}},
				ClickPosition:    &runtime.Point{X: 700, Y: 420},
				HighlightRegions: []runtime.HighlightRegion{{Selector: "#hero"}},
			},
			{
				StepIndex:  1,
				NodeID:     "node-2",
				StepType:   "screenshot",
				Status:     "completed",
				DurationMs: 600,
				ZoomFactor: 1.4,
				Screenshot: &TimelineScreenshot{
					ArtifactID: "shot-2",
					URL:        "https://cdn.example.com/shot-2.png",
					Width:      1440,
					Height:     900,
				},
				FocusedElement: &runtime.ElementFocus{
					Selector:    "#cta",
					BoundingBox: &runtime.BoundingBox{X: 120, Y: 200, Width: 240, Height: 80},
				},
				DomSnapshotPreview: "<div>Preview</div>",
				DomSnapshot: &TimelineArtifact{
					ID:      "dom-2",
					Type:    "dom_snapshot",
					Payload: map[string]any{"html": "<html><body>Full DOM</body></html>"},
				},
				RetryAttempt:       2,
				RetryMaxAttempts:   3,
				RetryConfigured:    2,
				RetryDelayMs:       250,
				RetryBackoffFactor: 1.5,
				RetryHistory:       []RetryHistoryEntry{{Attempt: 1, Success: false, Error: "timeout"}, {Attempt: 2, Success: true}},
			},
		},
	}

	pkg, err := BuildReplayMovieSpec(exec, workflow, timeline)
	if err != nil {
		t.Fatalf("BuildReplayMovieSpec returned error: %v", err)
	}

	if pkg == nil {
		t.Fatalf("expected export package, got nil")
	}

	if pkg.Version == "" {
		t.Errorf("expected schema version to be set")
	}

	if pkg.Execution.WorkflowName != workflow.Name {
		t.Errorf("expected workflow name %q, got %q", workflow.Name, pkg.Execution.WorkflowName)
	}

	if pkg.Summary.FrameCount != len(timeline.Frames) {
		t.Errorf("expected frame count %d, got %d", len(timeline.Frames), pkg.Summary.FrameCount)
	}

	if len(pkg.Frames) != len(timeline.Frames) {
		t.Fatalf("expected %d frames, got %d", len(timeline.Frames), len(pkg.Frames))
	}

	first := pkg.Frames[0]
	if first.StartOffsetMs != 0 {
		t.Errorf("expected first frame start offset 0, got %d", first.StartOffsetMs)
	}
	if first.ScreenshotAssetID != "shot-1" {
		t.Errorf("expected screenshot asset ID 'shot-1', got %q", first.ScreenshotAssetID)
	}
	if len(first.NormalizedCursorTrail) != 2 {
		t.Errorf("expected 2 normalized cursor points, got %d", len(first.NormalizedCursorTrail))
	}

	second := pkg.Frames[1]
	if second.StartOffsetMs <= first.StartOffsetMs {
		t.Errorf("expected second frame to start after first, got %d", second.StartOffsetMs)
	}
	if second.Resilience.Attempt != 2 || second.Resilience.MaxAttempts != 3 {
		t.Errorf("unexpected resiliency metadata: %+v", second.Resilience)
	}
	if second.ZoomFactor <= 1.0 {
		t.Errorf("expected zoom factor to be preserved, got %f", second.ZoomFactor)
	}
	if second.NormalizedFocusBounds == nil {
		t.Errorf("expected normalized focus bounds to be populated")
	}
	if second.DomSnapshotPreview != "<div>Preview</div>" {
		t.Errorf("expected DOM snapshot preview to be preserved, got %q", second.DomSnapshotPreview)
	}
	if second.DomSnapshotHTML != "<html><body>Full DOM</body></html>" {
		t.Errorf("expected DOM snapshot HTML to be propagated")
	}

	if len(pkg.Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(pkg.Assets))
	}
	if pkg.Assets[0].ID != "shot-1" {
		t.Errorf("expected first asset id 'shot-1', got %q", pkg.Assets[0].ID)
	}

	if pkg.Theme.BrowserChrome.Title != workflow.Name {
		t.Errorf("expected chrome title %q, got %q", workflow.Name, pkg.Theme.BrowserChrome.Title)
	}

	if pkg.Decor.ChromeTheme == "" {
		t.Errorf("expected decor chrome theme to be populated")
	}
	if pkg.Decor.BackgroundTheme == "" {
		t.Errorf("expected decor background theme to be populated")
	}
	if pkg.Decor.CursorTheme == "" {
		t.Errorf("expected decor cursor theme to be populated")
	}
	if pkg.Decor.CursorScale <= 0 {
		t.Errorf("expected decor cursor scale to be positive, got %f", pkg.Decor.CursorScale)
	}
}
