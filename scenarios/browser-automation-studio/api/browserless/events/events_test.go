package events

import (
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

type captureBroadcaster struct {
	updates []wsHub.ExecutionUpdate
}

func (c *captureBroadcaster) BroadcastUpdate(update wsHub.ExecutionUpdate) {
	c.updates = append(c.updates, update)
}

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

func TestEmitterEmitConstructsExecutionUpdate(t *testing.T) {
	broadcaster := &captureBroadcaster{}
	log := logrus.New()
	log.SetOutput(io.Discard)

	emitter := NewEmitter(broadcaster, log)
	if emitter == nil {
		t.Fatal("expected emitter instance")
	}

	executionID := uuid.New()
	workflowID := uuid.New()
	event := NewEvent(
		EventStepHeartbeat,
		executionID,
		workflowID,
		WithStep(1, "node-1", "wait"),
		WithProgress(25),
		WithStatus("running"),
		WithMessage("waiting"),
		WithPayload(map[string]any{"elapsed_ms": int64(500)}),
	)

	emitter.Emit(event)

	if len(broadcaster.updates) != 1 {
		t.Fatalf("expected 1 broadcast update, got %d", len(broadcaster.updates))
	}

	update := broadcaster.updates[0]
	if update.Type != "event" {
		t.Fatalf("expected event update, got %+v", update)
	}
	if update.ExecutionID != executionID || update.Status != "running" || update.Progress != 25 {
		t.Fatalf("unexpected identifiers/status: %+v", update)
	}

	payload, ok := update.Data.(Event)
	if !ok {
		t.Fatalf("expected Event payload, got %T", update.Data)
	}
	if payload.Type != EventStepHeartbeat {
		t.Fatalf("expected heartbeat event, got %s", payload.Type)
	}
	if payload.Payload["elapsed_ms"] != int64(500) {
		t.Fatalf("payload mismatch: %+v", payload.Payload)
	}
}
