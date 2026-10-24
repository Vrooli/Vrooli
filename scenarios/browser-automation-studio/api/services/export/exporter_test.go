package export

import (
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

func TestBuildReplaySpecFromTimelineGeneratesSpec(t *testing.T) {
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
						{X: 640, Y: 360},
						{X: 700, Y: 420},
					},
					ClickPosition: &autocontracts.Point{X: 700, Y: 420},
					HighlightRegions: []*autocontracts.HighlightRegion{
						{Selector: "#hero"},
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
					RetryAttempt:       2,
					RetryMaxAttempts:   3,
					RetryConfigured:    2,
					RetryDelayMs:       250,
					RetryBackoffFactor: 1.5,
					RetryHistory:       []RetryHistoryEntry{{Attempt: 1, Success: false, Error: "timeout"}, {Attempt: 2, Success: true}},
				},
			},
		}

		pkg, err := BuildReplaySpecFromTimeline(exec, workflow, timeline)
		if err != nil {
			t.Fatalf("BuildReplaySpecFromTimeline returned error: %v", err)
		}

		if pkg == nil {
			t.Fatalf("expected export package, got nil")
		}

		if pkg.GetVersion() == "" {
			t.Errorf("expected schema version to be set")
		}

		if pkg.GetExecution().GetWorkflowName() != workflow.Name {
			t.Errorf("expected workflow name %q, got %q", workflow.Name, pkg.GetExecution().GetWorkflowName())
		}

		if int(pkg.GetSummary().GetFrameCount()) != len(timeline.Frames) {
			t.Errorf("expected frame count %d, got %d", len(timeline.Frames), pkg.GetSummary().GetFrameCount())
		}

		if len(pkg.GetFrames()) != len(timeline.Frames) {
			t.Fatalf("expected %d frames, got %d", len(timeline.Frames), len(pkg.GetFrames()))
		}

		first := pkg.GetFrames()[0]
		if first.GetStartOffsetMs() != 0 {
			t.Errorf("expected first frame start offset 0, got %d", first.GetStartOffsetMs())
		}
		if first.GetScreenshotAssetId() != "shot-1" {
			t.Errorf("expected screenshot asset ID 'shot-1', got %q", first.GetScreenshotAssetId())
		}
		if len(first.GetNormalizedCursorTrail()) != 2 {
			t.Errorf("expected 2 normalized cursor points, got %d", len(first.GetNormalizedCursorTrail()))
		}

		second := pkg.GetFrames()[1]
		if second.GetStartOffsetMs() <= first.GetStartOffsetMs() {
			t.Errorf("expected second frame to start after first, got %d", second.GetStartOffsetMs())
		}
		if second.GetResilience().GetAttempt() != 2 || second.GetResilience().GetMaxAttempts() != 3 {
			t.Errorf("unexpected resiliency metadata: %+v", second.GetResilience())
		}
		if second.GetZoomFactor() <= 1.0 {
			t.Errorf("expected zoom factor to be preserved, got %f", second.GetZoomFactor())
		}
		if second.GetNormalizedFocusBounds() == nil {
			t.Errorf("expected normalized focus bounds to be populated")
		}
		if len(pkg.GetAssets()) != 2 {
			t.Errorf("expected 2 assets, got %d", len(pkg.GetAssets()))
		}
		if pkg.GetAssets()[0].GetId() != "shot-1" {
			t.Errorf("expected first asset id 'shot-1', got %q", pkg.GetAssets()[0].GetId())
		}

		if pkg.GetTheme().GetBrowserChrome().GetTitle() != workflow.Name {
			t.Errorf("expected chrome title %q, got %q", workflow.Name, pkg.GetTheme().GetBrowserChrome().GetTitle())
		}

		if pkg.GetDecor().GetChromeTheme() == "" {
			t.Errorf("expected decor chrome theme to be populated")
		}
		if pkg.GetDecor().GetBackgroundTheme() == "" {
			t.Errorf("expected decor background theme to be populated")
		}
		if pkg.GetDecor().GetCursorTheme() == "" {
			t.Errorf("expected decor cursor theme to be populated")
		}
		if pkg.GetDecor().GetCursorScale() <= 0 {
			t.Errorf("expected decor cursor scale to be positive, got %f", pkg.GetDecor().GetCursorScale())
		}

	})
}

func TestReplaySpecKeepsCssViewportSeparateFromScreenshotRaster(t *testing.T) {
	execution := &database.ExecutionIndex{ID: uuid.New(), WorkflowID: uuid.New(), Status: database.ExecutionStatusCompleted}
	workflow := &database.WorkflowIndex{ID: execution.WorkflowID, Name: "geometry"}
	point := &autocontracts.Point{X: 720, Y: 450}
	timeline := &ExecutionTimeline{
		ExecutionID: execution.ID, WorkflowID: workflow.ID,
		ViewportWidth: 1440, ViewportHeight: 900, DeviceScaleFactor: 2,
		Frames: []TimelineFrame{{StepIndex: 0, Success: true, Screenshot: &TimelineScreenshot{ArtifactID: "shot", Width: 2880, Height: 1800}, CursorPosition: point}},
	}
	spec, err := BuildReplaySpecFromTimeline(execution, workflow, timeline)
	if err != nil {
		t.Fatalf("BuildReplaySpecFromTimeline returned error: %v", err)
	}
	if got := spec.Presentation.Canvas.Width; got != 2880 {
		t.Fatalf("canvas width = %d, want raster width 2880", got)
	}
	if got := spec.Presentation.Viewport.Width; got != 1440 {
		t.Fatalf("viewport width = %d, want CSS width 1440", got)
	}
	if got := spec.Presentation.DeviceScaleFactor; got != 2 {
		t.Fatalf("device scale factor = %v, want 2", got)
	}
	if got := spec.Frames[0].Viewport.Width; got != 1440 {
		t.Fatalf("frame viewport width = %d, want CSS width 1440", got)
	}
	if got := spec.Frames[0].NormalizedCursorTrail[0].X; got < 0.499 || got > 0.501 {
		t.Fatalf("normalized cursor x = %v, want 0.5", got)
	}
}

func TestTimelineFromReplayPackageUsesTypedTimelineEntries(t *testing.T) {
	executionID, workflowID := uuid.New(), uuid.New()
	exec := &database.ExecutionIndex{ID: executionID, WorkflowID: workflowID, Status: database.ExecutionStatusCompleted, StartedAt: time.Now()}
	step := int32(4)
	nodeID := "package-node"
	timeline, err := timelineFromReplayPackage(exec, &basevidence.ReplayPackage{Timeline: []*bastimeline.TimelineEntry{{StepIndex: &step, NodeId: &nodeID}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Frames) != 1 || timeline.Frames[0].StepIndex != 4 || timeline.Frames[0].NodeID != nodeID {
		t.Fatalf("typed replay timeline was not projected: %#v", timeline.Frames)
	}
}

func TestBuildReplaySpecFromTimelineValidatesInput(t *testing.T) {
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

		_, err := BuildReplaySpecFromTimeline(exec, workflow, timeline)
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

		_, err := BuildReplaySpecFromTimeline(exec, workflow, timeline)
		if err == nil {
			t.Fatalf("expected error due to execution/timeline mismatch")
		}
	})
}

func TestBuildReplaySpecFromTimelineHandlesScreenshotAssets(t *testing.T) {
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

		pkg, err := BuildReplaySpecFromTimeline(exec, workflow, timeline)
		if err != nil {
			t.Fatalf("BuildReplaySpecFromTimeline returned error: %v", err)
		}
		if len(pkg.GetAssets()) != 1 {
			t.Fatalf("expected screenshot assets to be deduplicated, got %d entries", len(pkg.GetAssets()))
		}
		if pkg.GetSummary().GetScreenshotCount() != 1 {
			t.Errorf("expected screenshot summary count 1, got %d", pkg.GetSummary().GetScreenshotCount())
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

		pkg, err := BuildReplaySpecFromTimeline(exec, workflow, timeline)
		if err != nil {
			t.Fatalf("BuildReplaySpecFromTimeline returned error: %v", err)
		}
		if len(pkg.GetAssets()) != 1 {
			t.Fatalf("expected one generated asset entry, got %d", len(pkg.GetAssets()))
		}
		if pkg.GetAssets()[0].GetId() == "" {
			t.Fatalf("expected generated screenshot asset id when none provided")
		}
		if pkg.GetAssets()[0].GetSource() != "https://cdn.example.com/fallback.png" {
			t.Fatalf("expected asset source to match screenshot URL, got %s", pkg.GetAssets()[0].GetSource())
		}
	})
}
