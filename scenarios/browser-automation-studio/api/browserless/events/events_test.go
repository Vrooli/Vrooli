package events

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEventAppliesOptions(t *testing.T) {
	execID := uuid.New()
	workflowID := uuid.New()
	ts := time.Now().Add(-time.Hour).UTC()
	stepIndex := 2

	event := NewEvent(
		EventStepCompleted,
		execID,
		workflowID,
		WithStep(stepIndex, "node-2", "screenshot"),
		WithStatus("success"),
		WithProgress(80),
		WithMessage("Step completed"),
		WithPayload(map[string]any{"duration_ms": 1234}),
		WithTimestamp(ts),
	)

	if event.Type != EventStepCompleted {
		t.Fatalf("unexpected type: %s", event.Type)
	}
	if event.ExecutionID != execID || event.WorkflowID != workflowID {
		t.Fatalf("unexpected identifiers: %+v", event)
	}
	if event.StepIndex == nil || *event.StepIndex != stepIndex {
		t.Fatalf("expected step index %d, got %+v", stepIndex, event.StepIndex)
	}
	if event.StepNodeID != "node-2" || event.StepType != "screenshot" {
		t.Fatalf("step metadata not applied: %+v", event)
	}
	if event.Status != "success" {
		t.Fatalf("expected status success, got %s", event.Status)
	}
	if event.Progress == nil || *event.Progress != 80 {
		t.Fatalf("expected progress 80, got %+v", event.Progress)
	}
	if event.Timestamp != ts {
		t.Fatalf("expected timestamp %s, got %s", ts, event.Timestamp)
	}
	if event.Payload["duration_ms"] != 1234 {
		t.Fatalf("payload not applied: %+v", event.Payload)
	}
}
