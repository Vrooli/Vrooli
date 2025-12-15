package export

import (
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
)

func TestBuildReplayMovieSpecGeneratesSpec(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] generates complete replay movie spec", func(t *testing.T) {
		executionID := uuid.New()
		workflowID := uuid.New()
		now := time.Now()

		exec := &database.ExecutionIndex{
			ID:         executionID,
			WorkflowID: workflowID,
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  now.Add(-2 * time.Minute),
			CreatedAt:  now.Add(-2 * time.Minute),
			UpdatedAt:  now.Add(-time.Minute),
			CompletedAt: func() *time.Time {
				ts := now.Add(-time.Minute)
				return &ts
			}(),
		}

		workflow := &database.WorkflowIndex{
			ID:         workflowID,
			Name:       "Demo Journey",
			FolderPath: "/",
			Version:    1,
		}

		timeline := &ExecutionTimeline{
			ExecutionID: executionID,
			WorkflowID:  workflowID,
			Status:      database.ExecutionStatusCompleted,
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
					CursorTrail: []*autocontracts.Point{
						&autocontracts.Point{X: 640, Y: 360},
						&autocontracts.Point{X: 700, Y: 420},
					},
					ClickPosition: &autocontracts.Point{X: 700, Y: 420},
					HighlightRegions: []*autocontracts.HighlightRegion{
						&autocontracts.HighlightRegion{Selector: "#hero"},
					},
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
					FocusedElement: &autocontracts.ElementFocus{
						Selector:    "#cta",
						BoundingBox: &autocontracts.BoundingBox{X: 120, Y: 200, Width: 240, Height: 80},
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
	})
}

func TestBuildReplayMovieSpecValidatesInput(t *testing.T) {
	t.Run("errors when timeline missing frames", func(t *testing.T) {
		now := time.Now()
		exec := &database.ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  now.Add(-time.Minute),
			CreatedAt:  now.Add(-time.Minute),
			UpdatedAt:  now,
			CompletedAt: func() *time.Time {
				ts := now
				return &ts
			}(),
		}
		workflow := &database.WorkflowIndex{
			ID:         exec.WorkflowID,
			Name:       "Missing Frames Workflow",
			FolderPath: "/",
			Version:    1,
		}
		timeline := &ExecutionTimeline{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Status:      database.ExecutionStatusCompleted,
			Frames:      nil,
		}

		_, err := BuildReplayMovieSpec(exec, workflow, timeline)
		if err == nil {
			t.Fatalf("expected error when timeline has no frames")
		}
	})

	t.Run("errors when execution and timeline mismatch", func(t *testing.T) {
		now := time.Now()
		exec := &database.ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  now.Add(-time.Minute),
			CreatedAt:  now.Add(-time.Minute),
			UpdatedAt:  now,
		}
		workflow := &database.WorkflowIndex{
			ID:         exec.WorkflowID,
			Name:       "Mismatch Workflow",
			FolderPath: "/",
			Version:    1,
		}
		timeline := &ExecutionTimeline{
			ExecutionID: uuid.New(),
			WorkflowID:  uuid.New(),
			Status:      database.ExecutionStatusCompleted,
			Frames:      []TimelineFrame{{StepIndex: 0, NodeID: "node-1", StepType: "navigate", Status: "completed"}},
		}

		_, err := BuildReplayMovieSpec(exec, workflow, timeline)
		if err == nil {
			t.Fatalf("expected error due to execution/timeline mismatch")
		}
	})
}

func TestBuildReplayMovieSpecHandlesScreenshotAssets(t *testing.T) {
	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] deduplicates screenshot assets", func(t *testing.T) {
		now := time.Now()
		exec := &database.ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  now.Add(-time.Minute),
			CreatedAt:  now.Add(-time.Minute),
			UpdatedAt:  now,
		}
		workflow := &database.WorkflowIndex{ID: exec.WorkflowID, Name: "Asset Workflow", FolderPath: "/", Version: 1}

		frame := TimelineFrame{
			StepIndex: 0,
			NodeID:    "node-1",
			Status:    "completed",
			Screenshot: &TimelineScreenshot{
				ArtifactID: "shot-shared",
				URL:        "https://cdn.example.com/shared.png",
				Width:      800,
				Height:     600,
			},
		}
		timeline := &ExecutionTimeline{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Status:      "completed",
			Frames:      []TimelineFrame{frame, frame},
		}

		pkg, err := BuildReplayMovieSpec(exec, workflow, timeline)
		if err != nil {
			t.Fatalf("BuildReplayMovieSpec returned error: %v", err)
		}
		if len(pkg.Assets) != 1 {
			t.Fatalf("expected screenshot assets to be deduplicated, got %d entries", len(pkg.Assets))
		}
		if pkg.Summary.ScreenshotCount != 1 {
			t.Errorf("expected screenshot summary count 1, got %d", pkg.Summary.ScreenshotCount)
		}
	})

	t.Run("[REQ:BAS-REPLAY-EXPORT-BUNDLE] generates asset IDs when missing", func(t *testing.T) {
		now := time.Now()
		exec := &database.ExecutionIndex{
			ID:         uuid.New(),
			WorkflowID: uuid.New(),
			Status:     database.ExecutionStatusCompleted,
			StartedAt:  now.Add(-time.Minute),
			CreatedAt:  now.Add(-time.Minute),
			UpdatedAt:  now,
		}
		workflow := &database.WorkflowIndex{ID: exec.WorkflowID, Name: "Fallback Asset Workflow", FolderPath: "/", Version: 1}

		timeline := &ExecutionTimeline{
			ExecutionID: exec.ID,
			WorkflowID:  exec.WorkflowID,
			Status:      database.ExecutionStatusCompleted,
			Frames: []TimelineFrame{
				{
					StepIndex: 0,
					NodeID:    "node-1",
					Status:    "completed",
					Screenshot: &TimelineScreenshot{
						ArtifactID: "",
						URL:        "https://cdn.example.com/fallback.png",
					},
				},
			},
		}

		pkg, err := BuildReplayMovieSpec(exec, workflow, timeline)
		if err != nil {
			t.Fatalf("BuildReplayMovieSpec returned error: %v", err)
		}
		if len(pkg.Assets) != 1 {
			t.Fatalf("expected one generated asset entry, got %d", len(pkg.Assets))
		}
		if pkg.Assets[0].ID == "" {
			t.Fatalf("expected generated screenshot asset id when none provided")
		}
		if pkg.Assets[0].Source != "https://cdn.example.com/fallback.png" {
			t.Fatalf("expected asset source to match screenshot URL, got %s", pkg.Assets[0].Source)
		}
	})
}
