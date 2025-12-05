package execution

import (
	"testing"
	"time"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var timelineMarshalOpts = protojson.MarshalOptions{UseProtoNames: true}

func marshalTimeline(t *testing.T, tl *basv1.ExecutionTimeline) []byte {
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

func TestParseFullTimeline_BasicTimeline(t *testing.T) {
	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "exec-123",
		WorkflowId:  "wf-456",
		Status:      "completed",
		Progress:    100,
		Frames: []*basv1.TimelineFrame{
			{
				StepIndex:  0,
				NodeId:     "navigate-1",
				StepType:   "navigate",
				Status:     "completed",
				Success:    true,
				DurationMs: 1500,
			},
			{
				StepIndex: 1,
				NodeId:    "assert-1",
				StepType:  "assert",
				Status:    "completed",
				Success:   true,
			},
		},
		Logs: []*basv1.TimelineLog{
			{
				Id:        "log-1",
				Level:     "info",
				Message:   "Test message",
				StepName:  "navigate-1",
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
	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      "failed",
		Frames: []*basv1.TimelineFrame{
			{
				StepIndex: 0,
				NodeId:    "navigate-1",
				StepType:  "navigate",
				Status:    "completed",
				Success:   true,
			},
			{
				StepIndex: 1,
				NodeId:    "click-1",
				StepType:  "click",
				Status:    "failed",
				Success:   false,
				Error:     "element not found",
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

func TestParseFullTimeline_WithScreenshots(t *testing.T) {
	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      "completed",
		Frames: []*basv1.TimelineFrame{
			{
				StepIndex: 0,
				StepType:  "navigate",
				Status:    "completed",
				Success:   true,
				Screenshot: &basv1.TimelineScreenshot{
					ArtifactId: "ss-123",
					Url:        "/api/v1/screenshots/ss-123",
					Width:      1920,
					Height:     1080,
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
	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      "failed",
		Frames: []*basv1.TimelineFrame{
			{
				StepIndex: 0,
				NodeId:    "assert-1",
				StepType:  "assert",
				Status:    "completed",
				Success:   true,
				Assertion: &basv1.AssertionOutcome{
					Mode:     "visible",
					Selector: "[data-testid='header']",
					Success:  true,
				},
			},
			{
				StepIndex: 1,
				NodeId:    "assert-2",
				StepType:  "assert",
				Status:    "failed",
				Success:   false,
				Assertion: &basv1.AssertionOutcome{
					Mode:     "text",
					Selector: "[data-testid='title']",
					Expected: structpb.NewStringValue("Welcome"),
					Actual:   structpb.NewStringValue("Loading..."),
					Success:  false,
					Message:  "Text mismatch",
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
	if parsed.Assertions[0].AssertionType != "visible" {
		t.Errorf("expected assertion type 'visible', got %s", parsed.Assertions[0].AssertionType)
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
	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      "completed",
		Frames:      []*basv1.TimelineFrame{},
		Logs: []*basv1.TimelineLog{
			{
				Id:        "log-1",
				Level:     "error",
				Message:   "Failed to load resource",
				StepName:  "navigate-1",
				Timestamp: timestamppb.New(now),
			},
			{
				Id:        "log-2",
				Level:     "warn",
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
	timeline := &basv1.ExecutionTimeline{
		ExecutionId: "exec-123",
		Status:      "completed",
		Frames: []*basv1.TimelineFrame{
			{
				StepIndex:          0,
				StepType:           "navigate",
				Status:             "completed",
				Success:            true,
				DomSnapshotPreview: "<html>...</html>",
				DomSnapshot: &basv1.TimelineArtifact{
					Id:         "dom-123",
					StorageUrl: "/api/v1/artifacts/dom-123",
					Payload: map[string]*structpb.Value{
						"html": structpb.NewStringValue("<html><body>Full DOM</body></html>"),
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
