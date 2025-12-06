package protoconv

import (
	"bytes"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExecutionToProto(t *testing.T) {
	now := time.Now().UTC()
	completed := now.Add(2 * time.Minute)
	heartbeat := now.Add(time.Minute)

	exec := &database.Execution{
		ID:              uuid.New(),
		WorkflowID:      uuid.New(),
		WorkflowVersion: 3,
		Status:          "running",
		TriggerType:     "manual",
		TriggerMetadata: database.JSONMap{"source": "api"},
		Parameters:      database.JSONMap{"foo": "bar", "count": 2},
		StartedAt:       now,
		CompletedAt:     &completed,
		LastHeartbeat:   &heartbeat,
		Error: database.NullableString{
			NullString: sql.NullString{String: "boom", Valid: true},
		},
		Result:      database.JSONMap{"ok": true},
		Progress:    42,
		CurrentStep: "navigate",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now,
	}

	pb, err := ExecutionToProto(exec)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pb.GetId() != exec.ID.String() {
		t.Fatalf("expected id %s, got %s", exec.ID, pb.GetId())
	}
	if pb.GetWorkflowVersion() != 3 {
		t.Fatalf("expected workflow_version 3, got %d", pb.GetWorkflowVersion())
	}
	if pb.Error == nil || pb.GetError() != "boom" {
		t.Fatalf("expected error \"boom\", got %v", pb.Error)
	}
	if pb.TriggerMetadata["source"].GetStringValue() != "api" {
		t.Fatalf("expected trigger metadata source=api, got %v", pb.TriggerMetadata["source"])
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
	stepIndex := 0

	timeline := &export.ExecutionTimeline{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Status:      "completed",
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
				HighlightRegions:     []autocontracts.HighlightRegion{{Selector: "#main", Padding: 4}},
				MaskRegions:          []autocontracts.MaskRegion{{Selector: "#mask", Opacity: 0.5}},
				FocusedElement:       &autocontracts.ElementFocus{Selector: "#main"},
				ElementBoundingBox:   &autocontracts.BoundingBox{X: 1, Y: 2, Width: 3, Height: 4},
				ClickPosition:        &autocontracts.Point{X: 5, Y: 6},
				CursorTrail:          []autocontracts.Point{{X: 1, Y: 1}},
				ZoomFactor:           1.2,
				Screenshot: &typeconv.TimelineScreenshot{
					ArtifactID: "shot-1",
					URL:        "https://example.com/shot",
					SizeBytes:  &size,
				},
				Artifacts: []typeconv.TimelineArtifact{{
					ID:        "art-1",
					Type:      "timeline_frame",
					StepIndex: &stepIndex,
					Payload:   map[string]any{"foo": "bar"},
				}},
				Assertion: &autocontracts.AssertionOutcome{
					Mode:     "equals",
					Selector: "#main",
					Success:  true,
					Expected: "a",
					Actual:   "a",
				},
				RetryAttempt:       1,
				RetryMaxAttempts:   3,
				RetryConfigured:    1,
				RetryDelayMs:       50,
				RetryBackoffFactor: 1.5,
				RetryHistory: []typeconv.RetryHistoryEntry{{
					Attempt:        1,
					Success:        true,
					DurationMs:     50,
					CallDurationMs: 45,
				}},
				DomSnapshotPreview: "preview",
				DomSnapshot: &typeconv.TimelineArtifact{
					ID: "dom-1",
				},
			},
		},
		Logs: []export.TimelineLog{{
			ID:        "log-1",
			Level:     "info",
			Message:   "ok",
			Timestamp: now,
		}},
	}

	pb, err := TimelineToProto(timeline)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(pb.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(pb.Frames))
	}
	frame := pb.Frames[0]
	if frame.GetScreenshot().GetArtifactId() != "shot-1" {
		t.Fatalf("unexpected screenshot artifact id: %s", frame.GetScreenshot().GetArtifactId())
	}
	if frame.Assertion == nil || frame.Assertion.GetMode() != "equals" {
		t.Fatalf("assertion not converted correctly: %+v", frame.Assertion)
	}
	if frame.ExtractedDataPreview == nil || frame.ExtractedDataPreview.GetStructValue().Fields["preview"].GetStringValue() != "ok" {
		t.Fatalf("expected extracted_data_preview to be preserved, got %+v", frame.ExtractedDataPreview)
	}
	if len(frame.Artifacts) != 1 || frame.Artifacts[0].Payload["foo"].GetStringValue() != "bar" {
		t.Fatalf("artifact payload not converted correctly: %+v", frame.Artifacts)
	}
}

func TestScreenshotsToProto(t *testing.T) {
	execID := uuid.New()
	shots := []*database.Screenshot{
		{
			ID:           uuid.New(),
			ExecutionID:  execID,
			StepName:     "navigate",
			Timestamp:    time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
			StorageURL:   "https://example.com/full.png",
			ThumbnailURL: "",
			Width:        800,
			Height:       600,
			SizeBytes:    1024,
			Metadata: database.JSONMap{
				"step_index":    1,
				"thumbnail_url": "https://example.com/thumb.png",
			},
		},
	}

	pb, err := ScreenshotsToProto(shots)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(pb.Screenshots) != 1 {
		t.Fatalf("expected 1 screenshot, got %d", len(pb.Screenshots))
	}
	shot := pb.Screenshots[0]
	if shot.GetExecutionId() != execID.String() {
		t.Fatalf("unexpected execution_id: %s", shot.GetExecutionId())
	}
	if shot.GetStepIndex() != 1 {
		t.Fatalf("expected step_index 1, got %d", shot.GetStepIndex())
	}
	if shot.GetThumbnailUrl() != "https://example.com/thumb.png" {
		t.Fatalf("expected thumbnail_url override, got %s", shot.GetThumbnailUrl())
	}

	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pb)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !containsJSONField(data, "storage_url") || !containsJSONField(data, "thumbnail_url") {
		t.Fatalf("expected protojson to use proto names: %s", string(data))
	}
}

func containsJSONField(data []byte, field string) bool {
	return bytes.Contains(data, []byte(`"`+field+`"`))
}
