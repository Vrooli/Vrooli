package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var timelineMarshalOpts = protojson.MarshalOptions{UseProtoNames: true}

func strPtr(s string) *string { return &s }
func int32Ptr(v int32) *int32 { return &v }
func boolPtr(v bool) *bool    { return &v }

func marshalTimeline(t *testing.T, tl *bastimeline.ExecutionTimeline) []byte {
	t.Helper()
	data, err := timelineMarshalOpts.Marshal(tl)
	if err != nil {
		t.Fatalf("failed to marshal timeline: %v", err)
	}
	return data
}

func TestParseFullTimeline_Empty(t *testing.T) {
	parsed, err := ParseFullTimeline(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected non-nil result for empty input")
	}
	if len(parsed.Frames) != 0 {
		t.Errorf("expected 0 frames, got %d", len(parsed.Frames))
	}
}

func TestParseFullTimeline_InvalidJSON(t *testing.T) {
	_, err := ParseFullTimeline([]byte("not valid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if _, ok := err.(*TimelineParseError); !ok {
		t.Errorf("expected TimelineParseError, got %T", err)
	}
}

func TestParseFullTimeline_UnknownFieldFails(t *testing.T) {
	data := []byte(`{"execution_id":"exec-123","unknown_field":true}`)

	_, err := ParseFullTimeline(data)
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
	if _, ok := err.(*TimelineParseError); !ok {
		t.Fatalf("expected TimelineParseError, got %T", err)
	}
}

func TestParseFullTimeline_BasicTimeline(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "exec-123",
		WorkflowId:  "wf-456",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Progress:    100,
		Entries: []*bastimeline.TimelineEntry{
			{
				StepIndex: int32Ptr(0),
				NodeId:    strPtr("navigate-1"),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
				Context: &basbase.EventContext{
					Success: boolPtr(true),
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_COMPLETED,
				},
			},
			{
				StepIndex: int32Ptr(1),
				NodeId:    strPtr("assert-1"),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_ASSERT},
				Context: &basbase.EventContext{
					Success: boolPtr(true),
					Assertion: &basbase.AssertionResult{
						Mode:     basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS,
						Selector: "#title",
						Success:  true,
					},
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_COMPLETED,
				},
			},
		},
		Logs: []*bastimeline.TimelineLog{
			{
				Id:        "log-1",
				Level:     basbase.LogLevel_LOG_LEVEL_INFO,
				Message:   "Test message",
				StepName:  strPtr("navigate-1"),
				Timestamp: timestamppb.Now(),
			},
		},
	}

	data := marshalTimeline(t, timeline)
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.ExecutionID != "exec-123" {
		t.Errorf("expected execution_id exec-123, got %s", parsed.ExecutionID)
	}
	if parsed.WorkflowID != "wf-456" {
		t.Errorf("expected workflow_id wf-456, got %s", parsed.WorkflowID)
	}
	if parsed.Status != "completed" {
		t.Errorf("expected status completed, got %s", parsed.Status)
	}
	if len(parsed.Frames) != 2 {
		t.Errorf("expected 2 frames, got %d", len(parsed.Frames))
	}
	if len(parsed.Logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(parsed.Logs))
	}
	if parsed.Summary.TotalSteps != 2 {
		t.Errorf("expected 2 total steps, got %d", parsed.Summary.TotalSteps)
	}
	if parsed.Summary.TotalAsserts != 1 {
		t.Errorf("expected 1 assertion, got %d", parsed.Summary.TotalAsserts)
	}
	if parsed.Summary.AssertsPassed != 1 {
		t.Errorf("expected 1 passed assertion, got %d", parsed.Summary.AssertsPassed)
	}
}

func TestParseFullTimeline_WithFailedFrame(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
		Entries: []*bastimeline.TimelineEntry{
			{
				StepIndex: int32Ptr(0),
				NodeId:    strPtr("navigate-1"),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
				Context: &basbase.EventContext{
					Success: boolPtr(true),
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_COMPLETED,
				},
			},
			{
				StepIndex: int32Ptr(1),
				NodeId:    strPtr("click-1"),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_CLICK},
				Context: &basbase.EventContext{
					Success: boolPtr(false),
					Error:   strPtr("element not found"),
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_FAILED,
				},
			},
		},
	}

	data := marshalTimeline(t, timeline)
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FailedFrame == nil {
		t.Fatal("expected failed frame to be set")
	}
	if parsed.FailedFrame.StepIndex != 1 {
		t.Errorf("expected failed frame step_index 1, got %d", parsed.FailedFrame.StepIndex)
	}
	if parsed.FailedFrame.Error != "element not found" {
		t.Errorf("expected error 'element not found', got %s", parsed.FailedFrame.Error)
	}
}

func TestParseFullTimeline_GoldenHappyFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "timeline_happy.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error parsing happy fixture: %v", err)
	}

	if parsed.Status != "completed" {
		t.Fatalf("expected status completed, got %s", parsed.Status)
	}
	if parsed.Summary.TotalSteps != 2 || parsed.Summary.TotalAsserts != 1 || parsed.Summary.AssertsPassed != 1 {
		t.Fatalf("unexpected summary: %+v", parsed.Summary)
	}
	if parsed.FinalDOMPreview == "" {
		t.Fatalf("expected dom snapshot preview to be present")
	}
}

func TestParseFullTimeline_GoldenFailureFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "timeline_failure.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error parsing failure fixture: %v", err)
	}

	if parsed.Status != "failed" {
		t.Fatalf("expected status failed, got %s", parsed.Status)
	}
	if parsed.FailedFrame == nil || parsed.FailedFrame.NodeID != "assert-1" {
		t.Fatalf("expected failed frame for assert-1, got %+v", parsed.FailedFrame)
	}
	if parsed.Summary.TotalAsserts == 0 || parsed.Summary.AssertsPassed != 0 {
		t.Fatalf("expected failed assertions to be reflected in summary, got %+v", parsed.Summary)
	}
}

func TestParseFullTimeline_WithScreenshots(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Entries: []*bastimeline.TimelineEntry{
			{
				StepIndex: int32Ptr(0),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
				Context: &basbase.EventContext{
					Success: boolPtr(true),
				},
				Telemetry: &basdomain.ActionTelemetry{
					Screenshot: &basdomain.TimelineScreenshot{
						ArtifactId: "ss-123",
						Url:        "/api/v1/screenshots/ss-123",
						Width:      1920,
						Height:     1080,
					},
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_COMPLETED,
				},
			},
		},
	}

	data := marshalTimeline(t, timeline)
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(parsed.Frames))
	}
	if parsed.Frames[0].Screenshot == nil {
		t.Fatal("expected screenshot to be set")
	}
	if parsed.Frames[0].Screenshot.URL != "/api/v1/screenshots/ss-123" {
		t.Errorf("expected screenshot URL, got %s", parsed.Frames[0].Screenshot.URL)
	}
	if parsed.Frames[0].Screenshot.Width != 1920 {
		t.Errorf("expected width 1920, got %d", parsed.Frames[0].Screenshot.Width)
	}
}

func TestParseFullTimeline_WithAssertions(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
		Entries: []*bastimeline.TimelineEntry{
			{
				StepIndex: int32Ptr(0),
				NodeId:    strPtr("assert-1"),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_ASSERT},
				Context: &basbase.EventContext{
					Success: boolPtr(true),
					Assertion: &basbase.AssertionResult{
						Mode:     basbase.AssertionMode_ASSERTION_MODE_VISIBLE,
						Selector: "[data-testid='header']",
						Success:  true,
					},
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_COMPLETED,
				},
			},
			{
				StepIndex: int32Ptr(1),
				NodeId:    strPtr("assert-2"),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_ASSERT},
				Context: &basbase.EventContext{
					Success: boolPtr(false),
					Assertion: &basbase.AssertionResult{
						Mode:     basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS,
						Selector: "[data-testid='title']",
						Expected: &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "Welcome"}},
						Actual:   &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: "Loading..."}},
						Success:  false,
						Message:  strPtr("Text mismatch"),
					},
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_FAILED,
				},
			},
		},
	}

	data := marshalTimeline(t, timeline)
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.Assertions) != 2 {
		t.Fatalf("expected 2 assertions, got %d", len(parsed.Assertions))
	}

	// First assertion (passed)
	if !parsed.Assertions[0].Passed {
		t.Error("expected first assertion to pass")
	}
	if parsed.Assertions[0].AssertionType != "ASSERTION_MODE_VISIBLE" {
		t.Errorf("expected assertion type 'ASSERTION_MODE_VISIBLE', got %s", parsed.Assertions[0].AssertionType)
	}

	// Second assertion (failed)
	if parsed.Assertions[1].Passed {
		t.Error("expected second assertion to fail")
	}
	if parsed.Assertions[1].Expected != "Welcome" {
		t.Errorf("expected 'Welcome', got %s", parsed.Assertions[1].Expected)
	}
	if parsed.Assertions[1].Actual != "Loading..." {
		t.Errorf("expected 'Loading...', got %s", parsed.Assertions[1].Actual)
	}
}

func TestParseFullTimeline_WithLogs(t *testing.T) {
	now := time.Now().UTC()
	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Entries:     []*bastimeline.TimelineEntry{},
		Logs: []*bastimeline.TimelineLog{
			{
				Id:        "log-1",
				Level:     basbase.LogLevel_LOG_LEVEL_ERROR,
				Message:   "Failed to load resource",
				StepName:  strPtr("navigate-1"),
				Timestamp: timestamppb.New(now),
			},
			{
				Id:        "log-2",
				Level:     basbase.LogLevel_LOG_LEVEL_WARN,
				Message:   "Deprecated API usage",
				Timestamp: timestamppb.New(now),
			},
		},
	}

	data := marshalTimeline(t, timeline)
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.Logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(parsed.Logs))
	}
	if parsed.Logs[0].Level != "error" {
		t.Errorf("expected level 'error', got %s", parsed.Logs[0].Level)
	}
	if parsed.Logs[0].Message != "Failed to load resource" {
		t.Errorf("expected error message, got %s", parsed.Logs[0].Message)
	}
	if parsed.Logs[0].StepName != "navigate-1" {
		t.Errorf("expected step_name 'navigate-1', got %s", parsed.Logs[0].StepName)
	}
}

func TestParseFullTimeline_WithDOMSnapshot(t *testing.T) {
	timeline := &bastimeline.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Entries: []*bastimeline.TimelineEntry{
			{
				StepIndex: int32Ptr(0),
				Action:    &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE},
				Context: &basbase.EventContext{
					Success: boolPtr(true),
				},
				Aggregates: &bastimeline.TimelineEntryAggregates{
					Status: basbase.StepStatus_STEP_STATUS_COMPLETED,
					Artifacts: []*bastimeline.TimelineArtifact{
						{
							Id:         "dom-123",
							Type:       basbase.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT,
							StorageUrl: "/api/v1/artifacts/dom-123",
							Payload: map[string]*commonv1.JsonValue{
								"html":                 {Kind: &commonv1.JsonValue_StringValue{StringValue: "<html><body>Full DOM</body></html>"}},
								"dom_snapshot_preview": {Kind: &commonv1.JsonValue_StringValue{StringValue: "<html>...</html>"}},
							},
						},
					},
				},
			},
		},
	}

	data := marshalTimeline(t, timeline)
	parsed, err := ParseFullTimeline(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.FinalDOMPreview != "<html>...</html>" {
		t.Errorf("expected DOM preview, got %s", parsed.FinalDOMPreview)
	}
	if parsed.FinalDOM != "<html><body>Full DOM</body></html>" {
		t.Errorf("expected full DOM, got %s", parsed.FinalDOM)
	}
}
