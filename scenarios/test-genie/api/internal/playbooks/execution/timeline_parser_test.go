package execution

import (
	"encoding/json"
	"testing"
	"time"
)

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
	timeline := map[string]any{
		"execution_id": "exec-123",
		"workflow_id":  "wf-456",
		"status":       "completed",
		"progress":     100,
		"frames": []map[string]any{
			{
				"step_index": 0,
				"node_id":    "navigate-1",
				"step_type":  "navigate",
				"status":     "completed",
				"success":    true,
				"duration_ms": 1500,
			},
			{
				"step_index": 1,
				"node_id":    "assert-1",
				"step_type":  "assert",
				"status":     "completed",
				"success":    true,
			},
		},
		"logs": []map[string]any{
			{
				"id":        "log-1",
				"level":     "info",
				"message":   "Test message",
				"step_name": "navigate-1",
			},
		},
	}

	data, _ := json.Marshal(timeline)
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
	timeline := map[string]any{
		"execution_id": "exec-123",
		"status":       "failed",
		"frames": []map[string]any{
			{
				"step_index": 0,
				"node_id":    "navigate-1",
				"step_type":  "navigate",
				"status":     "completed",
				"success":    true,
			},
			{
				"step_index": 1,
				"node_id":    "click-1",
				"step_type":  "click",
				"status":     "failed",
				"success":    false,
				"error":      "element not found",
			},
		},
	}

	data, _ := json.Marshal(timeline)
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
	timeline := map[string]any{
		"execution_id": "exec-123",
		"status":       "completed",
		"frames": []map[string]any{
			{
				"step_index": 0,
				"step_type":  "navigate",
				"status":     "completed",
				"success":    true,
				"screenshot": map[string]any{
					"artifact_id": "ss-123",
					"url":         "/api/v1/screenshots/ss-123",
					"width":       1920,
					"height":      1080,
				},
			},
		},
	}

	data, _ := json.Marshal(timeline)
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
	timeline := map[string]any{
		"execution_id": "exec-123",
		"status":       "failed",
		"frames": []map[string]any{
			{
				"step_index": 0,
				"node_id":    "assert-1",
				"step_type":  "assert",
				"status":     "completed",
				"success":    true,
				"assertion": map[string]any{
					"type":     "visible",
					"selector": "[data-testid='header']",
					"passed":   true,
				},
			},
			{
				"step_index": 1,
				"node_id":    "assert-2",
				"step_type":  "assert",
				"status":     "failed",
				"success":    false,
				"assertion": map[string]any{
					"type":     "text",
					"selector": "[data-testid='title']",
					"expected": "Welcome",
					"actual":   "Loading...",
					"passed":   false,
					"message":  "Text mismatch",
				},
			},
		},
	}

	data, _ := json.Marshal(timeline)
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
	timeline := map[string]any{
		"execution_id": "exec-123",
		"status":       "completed",
		"frames":       []map[string]any{},
		"logs": []map[string]any{
			{
				"id":        "log-1",
				"level":     "error",
				"message":   "Failed to load resource",
				"step_name": "navigate-1",
				"timestamp": now.Format(time.RFC3339),
			},
			{
				"id":        "log-2",
				"level":     "warn",
				"message":   "Deprecated API usage",
				"timestamp": now.Format(time.RFC3339),
			},
		},
	}

	data, _ := json.Marshal(timeline)
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
	timeline := map[string]any{
		"execution_id": "exec-123",
		"status":       "completed",
		"frames": []map[string]any{
			{
				"step_index":           0,
				"step_type":            "navigate",
				"status":               "completed",
				"success":              true,
				"dom_snapshot_preview": "<html>...</html>",
				"dom_snapshot": map[string]any{
					"id":          "dom-123",
					"storage_url": "/api/v1/artifacts/dom-123",
					"payload": map[string]any{
						"html": "<html><body>Full DOM</body></html>",
					},
				},
			},
		},
	}

	data, _ := json.Marshal(timeline)
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
