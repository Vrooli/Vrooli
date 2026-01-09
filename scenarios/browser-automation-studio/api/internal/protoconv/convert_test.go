package protoconv

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/export"
	"google.golang.org/protobuf/encoding/protojson"
)

func containsJSONField(data []byte, field string) bool {
	return bytes.Contains(data, []byte(`"`+field+`"`))
}

func TestExecutionToProto(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(2 * time.Minute)

	exec := &database.ExecutionIndex{
		ID:         uuid.New(),
		WorkflowID: uuid.New(),
		Status:     database.ExecutionStatusRunning,
		StartedAt:  now,
		CompletedAt: func() *time.Time {
			return &completed
		}(),
		CreatedAt: now.Add(-time.Minute),
		UpdatedAt: now,
		ErrorMessage: "boom",
	}

	pb, err := ExecutionToProto(exec)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pb.GetExecutionId() != exec.ID.String() {
		t.Fatalf("expected id %s, got %s", exec.ID, pb.GetExecutionId())
	}
	if pb.GetWorkflowId() != exec.WorkflowID.String() {
		t.Fatalf("expected workflow_id %s, got %s", exec.WorkflowID, pb.GetWorkflowId())
	}
	if pb.Error == nil || pb.GetError() != "boom" {
		t.Fatalf("expected error \"boom\", got %v", pb.Error)
	}

	// Ensure JSON uses proto field names (snake_case).
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pb)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !containsJSONField(data, "workflow_id") {
		t.Fatalf("expected workflow_id field in marshalled JSON: %s", string(data))
	}
}

func TestTimelineToProto(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(time.Minute)
	size := int64(128)

	timeline := &export.ExecutionTimeline{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Status:      database.ExecutionStatusCompleted,
		Progress:    100,
		StartedAt:   now,
		CompletedAt: &completed,
		Frames: []export.TimelineFrame{
			{
				StepIndex:            0,
				NodeID:               "node-1",
				StepType:             "navigate",
				Status:               "completed",
				Success:              true,
				DurationMs:           1200,
				TotalDurationMs:      1500,
				Progress:             100,
				StartedAt:            &now,
				CompletedAt:          &completed,
				FinalURL:             "https://example.com",
				ConsoleLogCount:      1,
				NetworkEventCount:    0,
				ExtractedDataPreview: map[string]any{"preview": "ok"},
				HighlightRegions: []*autocontracts.HighlightRegion{
					&autocontracts.HighlightRegion{Selector: "#main", Padding: 4},
				},
				MaskRegions: []*autocontracts.MaskRegion{
					&autocontracts.MaskRegion{Selector: "#mask", Opacity: 0.5},
				},
				FocusedElement:     &autocontracts.ElementFocus{Selector: "#main"},
				ElementBoundingBox: &autocontracts.BoundingBox{X: 1, Y: 2, Width: 3, Height: 4},
				ClickPosition:      &autocontracts.Point{X: 5, Y: 6},
				CursorTrail: []*autocontracts.Point{
					&autocontracts.Point{X: 1, Y: 1},
				},
				ZoomFactor: 1.2,
				Screenshot: &export.TimelineScreenshot{
					ArtifactID:    "shot-1",
					URL:           "https://cdn.example.com/shot-1.png",
					ThumbnailURL:  "https://cdn.example.com/shot-1-thumb.png",
					Width:         800,
					Height:        600,
					SizeBytes:     &size,
					ContentType:   "image/png",
				},
			},
		},
	}

	pb, err := TimelineToProto(timeline)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pb.GetExecutionId() != timeline.ExecutionID.String() {
		t.Fatalf("execution_id mismatch")
	}
	if len(pb.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(pb.Entries))
	}
	entry := pb.Entries[0]
	if entry.GetContext().GetSuccess() != true {
		t.Fatalf("expected context.success=true")
	}
	if entry.GetTelemetry().GetScreenshot().GetUrl() != "https://cdn.example.com/shot-1.png" {
		t.Fatalf("expected screenshot url propagated")
	}
}
