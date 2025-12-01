package events

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

type stubHub struct {
	mu      sync.Mutex
	updates []any
}

type closingHub struct {
	stubHub
	closed []uuid.UUID
}

func (s *stubHub) ServeWS(*websocket.Conn, *uuid.UUID) {}
func (s *stubHub) Run()                                {}
func (s *stubHub) GetClientCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.updates)
}

func (s *stubHub) BroadcastEnvelope(event any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, event)
}

func (s *stubHub) Updates() []any {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]any, len(s.updates))
	copy(cp, s.updates)
	return cp
}

func (s *stubHub) CloseExecution(executionID uuid.UUID) {
	_ = executionID
}

func (c *closingHub) CloseExecution(executionID uuid.UUID) {
	c.stubHub.mu.Lock()
	c.closed = append(c.closed, executionID)
	c.stubHub.mu.Unlock()
}

func TestWSHubSinkPublishesAdaptedEvent(t *testing.T) {
	hub := &stubHub{}
	sink := NewWSHubSink(hub, logrus.New(), contracts.DefaultEventBufferLimits)

	stepIndex := 3
	attempt := 1
	execID := uuid.New()
	workflowID := uuid.New()

	env := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepCompleted,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		StepIndex:      &stepIndex,
		Attempt:        &attempt,
		Sequence:       42,
		Timestamp:      time.Unix(0, 0).UTC(),
		Payload: map[string]any{
			"foo": "bar",
		},
		Drops: contracts.DropCounters{
			Dropped:       2,
			OldestDropped: 7,
		},
	}

	if err := sink.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}

	updates := waitForUpdates(hub, 1, time.Second)
	if len(updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(updates))
	}
	payload, ok := updates[0].(contracts.EventEnvelope)
	if !ok {
		t.Fatalf("expected envelope payload, got %T", updates[0])
	}
	if payload.ExecutionID != execID || payload.Kind != contracts.EventKindStepCompleted {
		t.Fatalf("unexpected payload metadata: %+v", payload)
	}
	if payload.StepIndex == nil || *payload.StepIndex != stepIndex {
		t.Fatalf("unexpected payload content: %+v", payload)
	}

	if m, ok := payload.Payload.(map[string]any); ok {
		if val, exists := m["foo"]; !exists || val != "bar" {
			t.Fatalf("expected payload data foo=bar, got %+v", payload.Payload)
		}
	} else {
		t.Fatalf("expected map payload, got %T", payload.Payload)
	}

	if payload.Drops.Dropped != 2 || payload.Drops.OldestDropped != 7 {
		t.Fatalf("expected drops counters propagated, got %+v", payload.Drops)
	}
}

func TestWSHubSinkStepEnvelopeShapeMatchesLegacyExpectations(t *testing.T) {
	hub := &stubHub{}
	sink := NewWSHubSink(hub, logrus.New(), contracts.DefaultEventBufferLimits)

	stepIndex := 0
	attempt := 1
	execID := uuid.New()
	workflowID := uuid.New()

	stepOutcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    execID,
		StepIndex:      stepIndex,
		NodeID:         "node-1",
		StepType:       "click",
		Attempt:        attempt,
		Success:        true,
	}

	env := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepCompleted,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		StepIndex:      &stepIndex,
		Attempt:        &attempt,
		Sequence:       1,
		Timestamp:      time.Unix(0, 0).UTC(),
		Payload: map[string]any{
			"outcome": stepOutcome,
		},
	}

	if err := sink.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	updates := waitForUpdates(hub, 1, time.Second)
	if len(updates) != 1 {
		t.Fatalf("expected one update, got %d", len(updates))
	}

	payload, ok := updates[0].(contracts.EventEnvelope)
	if !ok {
		t.Fatalf("expected envelope payload, got %T", updates[0])
	}

	payloadMap, ok := payload.Payload.(map[string]any)
	if !ok {
		t.Fatalf("expected map payload, got %T", payload.Payload)
	}
	out, ok := payloadMap["outcome"].(contracts.StepOutcome)
	if !ok {
		t.Fatalf("expected outcome in payload, got %T", payloadMap["outcome"])
	}
	if out.SchemaVersion != contracts.StepOutcomeSchemaVersion || out.PayloadVersion != contracts.PayloadVersion {
		t.Fatalf("expected outcome schema/payload versions to be preserved, got %s/%s", out.SchemaVersion, out.PayloadVersion)
	}
	if out.StepIndex != stepIndex || out.NodeID != "node-1" || !out.Success {
		t.Fatalf("unexpected outcome content: %+v", out)
	}
}

func TestWSHubSinkCloseExecutionStopsEnqueue(t *testing.T) {
	hub := &stubHub{}
	sink := NewWSHubSink(hub, logrus.New(), contracts.DefaultEventBufferLimits)

	stepIndex := 1
	attempt := 1
	execID := uuid.New()
	workflowID := uuid.New()

	env := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepCompleted,
		ExecutionID:    execID,
		WorkflowID:     workflowID,
		StepIndex:      &stepIndex,
		Attempt:        &attempt,
		Sequence:       1,
		Timestamp:      time.Unix(0, 0).UTC(),
	}

	if err := sink.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish returned error: %v", err)
	}
	updates := waitForUpdates(hub, 1, time.Second)
	if len(updates) != 1 {
		t.Fatalf("expected 1 update before close, got %d", len(updates))
	}

	sink.CloseExecution(execID)

	// Attempt to publish after close should be ignored for this execution.
	if err := sink.Publish(context.Background(), env); err != nil {
		t.Fatalf("publish after close returned error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if count := len(hub.Updates()); count != 1 {
		t.Fatalf("expected updates to stay at 1 after close, got %d", count)
	}
}

// blockingHub simulates a slow downstream consumer to trigger backpressure logic.
type blockingHub struct {
	mu      sync.Mutex
	updates []any
	unblock chan struct{}
}

func newBlockingHub() *blockingHub {
	return &blockingHub{unblock: make(chan struct{})}
}

func (b *blockingHub) ServeWS(*websocket.Conn, *uuid.UUID) {}
func (b *blockingHub) Run()                                {}
func (b *blockingHub) GetClientCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.updates)
}

func (b *blockingHub) Updates() []any {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]any, len(b.updates))
	copy(cp, b.updates)
	return cp
}

func (b *blockingHub) CloseExecution(executionID uuid.UUID) {
	_ = executionID
}

func (b *blockingHub) BroadcastEnvelope(event any) {
	<-b.unblock
	b.mu.Lock()
	defer b.mu.Unlock()
	b.updates = append(b.updates, event)
}

func TestWSHubSinkDropsTelemetryWhenBufferFull(t *testing.T) {
	hub := newBlockingHub()
	limits := contracts.EventBufferLimits{PerExecution: 2, PerAttempt: 2}
	sink := NewWSHubSink(hub, logrus.New(), limits)

	stepIndex, attempt := 0, 1
	execID := uuid.New()
	workflowID := uuid.New()

	makeTelemetry := func(seq uint64) contracts.EventEnvelope {
		return contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			StepIndex:      &stepIndex,
			Attempt:        &attempt,
			Sequence:       seq,
			Timestamp:      time.Unix(int64(seq), 0).UTC(),
			Payload: map[string]any{
				"note": fmt.Sprintf("seq-%d", seq),
			},
		}
	}

	// Block the hub to simulate slow consumers and force the queue to fill.
	if err := sink.Publish(context.Background(), makeTelemetry(1)); err != nil {
		t.Fatalf("publish seq1 returned error: %v", err)
	}
	if err := sink.Publish(context.Background(), makeTelemetry(2)); err != nil {
		t.Fatalf("publish seq2 returned error: %v", err)
	}
	if err := sink.Publish(context.Background(), makeTelemetry(3)); err != nil {
		t.Fatalf("publish seq3 returned error: %v", err)
	}

	// Allow two events through.
	hub.unblock <- struct{}{}
	hub.unblock <- struct{}{}

	updates := waitForUpdates(hub, 2, 2*time.Second)
	if len(updates) != 2 {
		t.Fatalf("expected 2 updates after backpressure, got %d", len(updates))
	}

	receivedSeqs := []uint64{}
	var dropsSeen contracts.DropCounters
	for _, update := range updates {
		payload, ok := update.(contracts.EventEnvelope)
		if !ok {
			t.Fatalf("expected envelope payload, got %T", update)
		}
		receivedSeqs = append(receivedSeqs, payload.Sequence)
		dropsSeen = payload.Drops
	}

	if len(receivedSeqs) != 2 {
		t.Fatalf("expected 2 sequences after drop, got %+v", receivedSeqs)
	}

	expected := map[uint64]bool{1: true, 2: true, 3: true}
	for _, seq := range receivedSeqs {
		delete(expected, seq)
	}
	if len(expected) != 1 {
		t.Fatalf("expected exactly one sequence dropped, missing %+v", expected)
	}
	for missing := range expected {
		if dropsSeen.Dropped != 1 || dropsSeen.OldestDropped != missing {
			t.Fatalf("expected drop counters to reflect missing sequence %d, got %+v", missing, dropsSeen)
		}
	}

	// Ensure drops metadata flows through WS payload wrapper.
	lastPayload, ok := updates[len(updates)-1].(contracts.EventEnvelope)
	if !ok {
		t.Fatalf("expected envelope payload, got %T", updates[len(updates)-1])
	}
	if lastPayload.Drops.Dropped == 0 {
		t.Fatalf("expected drops metadata propagated in WS payload, got %+v", lastPayload.Drops)
	}
}

func TestExecutionQueueDropsOldAttemptTelemetry(t *testing.T) {
	hub := &stubHub{}
	limits := contracts.EventBufferLimits{PerExecution: 5, PerAttempt: 1}
	queue := newExecutionQueue(uuid.New(), hub, limits, logrus.New())

	stepIndex, attempt := 0, 1
	queue.enqueue(contracts.EventEnvelope{
		SchemaVersion: contracts.EventEnvelopeSchemaVersion,
		Kind:          contracts.EventKindStepTelemetry,
		ExecutionID:   uuid.New(),
		StepIndex:     &stepIndex,
		Attempt:       &attempt,
		Sequence:      1,
	})
	queue.enqueue(contracts.EventEnvelope{
		SchemaVersion: contracts.EventEnvelopeSchemaVersion,
		Kind:          contracts.EventKindStepTelemetry,
		ExecutionID:   uuid.New(),
		StepIndex:     &stepIndex,
		Attempt:       &attempt,
		Sequence:      2,
	})

	queue.mu.Lock()
	events := append([]contracts.EventEnvelope(nil), queue.events...)
	drops := queue.dropCounter
	queue.mu.Unlock()

	if len(events) != 1 {
		t.Fatalf("expected oldest telemetry to be dropped, queue contains %d events", len(events))
	}
	if events[0].Sequence != 2 {
		t.Fatalf("expected latest telemetry retained, got sequence %d", events[0].Sequence)
	}
	if events[0].Drops.Dropped != 1 || events[0].Drops.OldestDropped != 1 {
		t.Fatalf("expected drop counters to be attached to surviving event, got %+v", events[0].Drops)
	}
	if drops.OldestDropped != 1 {
		t.Fatalf("expected queue drop counter to be updated, got %+v", drops)
	}
}

func TestExecutionQueuePreservesNonDroppableEvents(t *testing.T) {
	hub := &stubHub{}
	limits := contracts.EventBufferLimits{PerExecution: 1, PerAttempt: 1}
	queue := newExecutionQueue(uuid.New(), hub, limits, logrus.New())

	queue.enqueue(contracts.EventEnvelope{
		SchemaVersion: contracts.EventEnvelopeSchemaVersion,
		Kind:          contracts.EventKindExecutionCompleted,
		ExecutionID:   uuid.New(),
		Sequence:      1,
	})
	queue.enqueue(contracts.EventEnvelope{
		SchemaVersion: contracts.EventEnvelopeSchemaVersion,
		Kind:          contracts.EventKindStepTelemetry,
		ExecutionID:   uuid.New(),
		Sequence:      2,
	})

	queue.mu.Lock()
	defer queue.mu.Unlock()
	if len(queue.events) != 2 {
		t.Fatalf("non-droppable completion event should be preserved, queue size %d", len(queue.events))
	}
	if queue.events[0].Kind != contracts.EventKindExecutionCompleted {
		t.Fatalf("expected completion event to stay in queue, got %s", queue.events[0].Kind)
	}
	if queue.events[1].Drops.Dropped != 0 {
		t.Fatalf("telemetry appended after non-droppable should not inherit drops, got %+v", queue.events[1].Drops)
	}
}

func TestWSHubSinkSanitizesInvalidLimits(t *testing.T) {
	hub := &stubHub{}
	sink := NewWSHubSink(hub, logrus.New(), contracts.EventBufferLimits{
		PerExecution: 0,
		PerAttempt:   -10,
	})

	limits := sink.Limits()
	if limits.PerExecution != contracts.DefaultEventBufferLimits.PerExecution || limits.PerAttempt != contracts.DefaultEventBufferLimits.PerAttempt {
		t.Fatalf("expected invalid limits replaced with defaults, got %+v", limits)
	}
}

func TestWSHubSinkCloseExecutionNotifiesHub(t *testing.T) {
	hub := &closingHub{}
	sink := NewWSHubSink(hub, logrus.New(), contracts.DefaultEventBufferLimits)
	execID := uuid.New()

	sink.CloseExecution(execID)

	hub.stubHub.mu.Lock()
	defer hub.stubHub.mu.Unlock()
	if len(hub.closed) != 1 || hub.closed[0] != execID {
		t.Fatalf("expected hub.CloseExecution invoked with %s, got %+v", execID, hub.closed)
	}
}

type updatesReader interface {
	Updates() []any
}

func waitForUpdates(h updatesReader, expected int, timeout time.Duration) []any {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if updates := h.Updates(); len(updates) >= expected {
			return updates
		}
		time.Sleep(10 * time.Millisecond)
	}
	return h.Updates()
}
