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
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	basrecording "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recording"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
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
		TriggerMetadata: database.JSONMap{"client_id": "api"},
		Parameters: database.JSONMap{
			"start_url": "https://example.test",
			"variables": map[string]any{"foo": "bar"},
		},
		StartedAt:       now,
		CompletedAt:     &completed,
		LastHeartbeat:   &heartbeat,
		Error: database.NullableString{
			NullString: sql.NullString{String: "boom", Valid: true},
		},
		Result:      database.JSONMap{"success": true},
		Progress:    42,
		CurrentStep: "navigate",
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now,
	}

	pb, err := ExecutionToProto(exec)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if pb.GetExecutionId() != exec.ID.String() {
		t.Fatalf("expected id %s, got %s", exec.ID, pb.GetExecutionId())
	}
	if pb.GetWorkflowVersion() != 3 {
		t.Fatalf("expected workflow_version 3, got %d", pb.GetWorkflowVersion())
	}
	if pb.Error == nil || pb.GetError() != "boom" {
		t.Fatalf("expected error \"boom\", got %v", pb.Error)
	}
	if pb.TriggerMetadata == nil || pb.TriggerMetadata.GetClientId() != "api" {
		t.Fatalf("expected trigger metadata client_id=api, got %+v", pb.TriggerMetadata)
	}
	if pb.Parameters == nil || pb.Parameters.GetStartUrl() != "https://example.test" {
		t.Fatalf("expected parameters start_url=https://example.test, got %+v", pb.Parameters)
	}
	if pb.Parameters.GetVariables()["foo"] != "bar" {
		t.Fatalf("expected parameters.variables.foo=bar, got %+v", pb.Parameters.GetVariables())
	}
	if pb.Result == nil || pb.Result.Success != true {
		t.Fatalf("expected result.success=true, got %+v", pb.Result)
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
						Mode:     "text_equals",
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
	if len(pb.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(pb.Entries))
	}
	entry := pb.Entries[0]
	if entry.GetTelemetry().GetScreenshot().GetArtifactId() != "shot-1" {
		t.Fatalf("unexpected screenshot artifact id: %s", entry.GetTelemetry().GetScreenshot().GetArtifactId())
	}
	if entry.GetContext().GetAssertion() == nil || entry.GetContext().GetAssertion().GetMode() != basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS {
		t.Fatalf("assertion not converted correctly: %+v", entry.GetContext().GetAssertion())
	}
	if entry.GetAggregates().GetExtractedDataPreview() == nil || entry.GetAggregates().GetExtractedDataPreview().GetObjectValue().Fields["preview"].GetStringValue() != "ok" {
		t.Fatalf("expected extracted_data_preview to be preserved, got %+v", entry.GetAggregates().GetExtractedDataPreview())
	}
	if len(entry.GetAggregates().GetArtifacts()) != 1 || entry.GetAggregates().GetArtifacts()[0].Payload["foo"].GetStringValue() != "bar" {
		t.Fatalf("artifact payload not converted correctly: %+v", entry.GetAggregates().GetArtifacts())
	}
	if entry.GetContext().GetRetryStatus() == nil || !entry.GetContext().GetRetryStatus().GetConfigured() {
		t.Fatalf("retry_status.configured should be set to true")
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

	pb, err := ScreenshotsToProto(shots, execID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(pb.Screenshots) != 1 {
		t.Fatalf("expected 1 screenshot, got %d", len(pb.Screenshots))
	}
	if pb.GetExecutionId() != execID.String() {
		t.Fatalf("unexpected execution_id: %s", pb.GetExecutionId())
	}
	shot := pb.Screenshots[0]
	if shot.GetStepIndex() != 1 {
		t.Fatalf("expected step_index 1, got %d", shot.GetStepIndex())
	}
	if shot.GetScreenshot().GetThumbnailUrl() != "https://example.com/thumb.png" {
		t.Fatalf("expected thumbnail_url override, got %s", shot.GetScreenshot().GetThumbnailUrl())
	}

	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pb)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !containsJSONField(data, "thumbnail_url") || !containsJSONField(data, "execution_id") {
		t.Fatalf("expected protojson to use proto names: %s", string(data))
	}
}

func containsJSONField(data []byte, field string) bool {
	return bytes.Contains(data, []byte(`"`+field+`"`))
}
