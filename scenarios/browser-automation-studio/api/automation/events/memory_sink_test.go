package events

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestMemorySinkDropsTelemetryAndAnnotatesCounters(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 3, PerAttempt: 2}
	sink := NewMemorySink(limits)
	execID := uuid.New()

	makeEvent := func(kind contracts.EventKind, seq uint64) contracts.EventEnvelope {
		return contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           kind,
			ExecutionID:    execID,
			Timestamp:      time.Now(),
			Sequence:       seq,
		}
	}

	// Fill the buffer: two telemetry + one completion (non-droppable).
	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepTelemetry, 1))
	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepTelemetry, 2))
	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepCompleted, 3))

	// This telemetry should cause the oldest telemetry (seq=1) to be dropped.
	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepTelemetry, 4))

	events := sink.Events()
	if len(events) != 3 {
		t.Fatalf("expected 3 events after drop, got %d", len(events))
	}

	// Expect sequences 2,3,4 to remain.
	remaining := map[uint64]bool{}
	for _, ev := range events {
		remaining[ev.Sequence] = true
	}
	for _, seq := range []uint64{2, 3, 4} {
		if !remaining[seq] {
			t.Fatalf("expected sequence %d to remain, got %+v", seq, remaining)
		}
	}

	// Drop counters should be recorded and attached to the newest event.
	counters := sink.DropCounters()[execID]
	if counters.Dropped != 1 || counters.OldestDropped != 1 {
		t.Fatalf("unexpected drop counters: %+v", counters)
	}

	last := events[len(events)-1]
	if last.Drops.Dropped != 1 || last.Drops.OldestDropped != 1 {
		t.Fatalf("expected drop counters on last event, got %+v", last.Drops)
	}
}

func TestMemorySinkPerAttemptLimits(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 10, PerAttempt: 2}
	sink := NewMemorySink(limits)
	execID := uuid.New()

	makeEvent := func(kind contracts.EventKind, seq uint64, step, attempt int) contracts.EventEnvelope {
		return contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           kind,
			ExecutionID:    execID,
			StepIndex:      &step,
			Attempt:        &attempt,
			Timestamp:      time.Now(),
			Sequence:       seq,
		}
	}

	step := 1
	attempt := 2

	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepTelemetry, 1, step, attempt))
	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepTelemetry, 2, step, attempt))
	// This should trigger per-attempt drop, removing seq=1.
	_ = sink.Publish(context.Background(), makeEvent(contracts.EventKindStepTelemetry, 3, step, attempt))

	evs := sink.Events()
	if len(evs) != 2 {
		t.Fatalf("expected 2 events after per-attempt drop, got %d", len(evs))
	}
	if evs[0].Sequence != 2 || evs[1].Sequence != 3 {
		t.Fatalf("expected sequences 2 and 3 after drop, got %+v", []uint64{evs[0].Sequence, evs[1].Sequence})
	}
	if evs[1].Drops.Dropped != 1 || evs[1].Drops.OldestDropped != 1 {
		t.Fatalf("expected drop counters on last event, got %+v", evs[1].Drops)
	}
}

func TestMemorySinkDoesNotDropTerminalEvents(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 1, PerAttempt: 1}
	sink := NewMemorySink(limits)
	execID := uuid.New()

	step := 0
	attempt := 1
	completed := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepCompleted,
		ExecutionID:    execID,
		StepIndex:      &step,
		Attempt:        &attempt,
		Sequence:       1,
		Timestamp:      time.Now(),
	}
	telemetry := contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepTelemetry,
		ExecutionID:    execID,
		StepIndex:      &step,
		Attempt:        &attempt,
		Sequence:       2,
		Timestamp:      time.Now(),
	}

	_ = sink.Publish(context.Background(), completed)
	_ = sink.Publish(context.Background(), telemetry)

	events := sink.Events()
	if len(events) != 2 {
		t.Fatalf("expected both terminal and telemetry events to be retained, got %d", len(events))
	}
	if events[0].Kind != contracts.EventKindStepCompleted || events[1].Kind != contracts.EventKindStepTelemetry {
		t.Fatalf("unexpected event kinds: %+v", events)
	}
	if counters := sink.DropCounters()[execID]; counters.Dropped != 0 {
		t.Fatalf("expected no drops when only terminal event preceded, got %+v", counters)
	}
}

func TestMemorySinkMultipleExecutionsIsolation(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 5, PerAttempt: 3}
	sink := NewMemorySink(limits)
	execA := uuid.New()
	execB := uuid.New()

	makeEvent := func(execID uuid.UUID, seq uint64) contracts.EventEnvelope {
		return contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    execID,
			Timestamp:      time.Now(),
			Sequence:       seq,
		}
	}

	// Publish events for both executions
	for i := uint64(1); i <= 3; i++ {
		_ = sink.Publish(context.Background(), makeEvent(execA, i))
		_ = sink.Publish(context.Background(), makeEvent(execB, i+10))
	}

	events := sink.Events()
	if len(events) != 6 {
		t.Fatalf("expected 6 events total, got %d", len(events))
	}

	// Verify events from both executions are present
	execACounts, execBCounts := 0, 0
	for _, ev := range events {
		if ev.ExecutionID == execA {
			execACounts++
		} else if ev.ExecutionID == execB {
			execBCounts++
		}
	}
	if execACounts != 3 || execBCounts != 3 {
		t.Fatalf("expected 3 events each, got execA=%d, execB=%d", execACounts, execBCounts)
	}
}

func TestMemorySinkCloseExecution(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 10, PerAttempt: 5}
	sink := NewMemorySink(limits)
	execA := uuid.New()
	execB := uuid.New()

	makeEvent := func(execID uuid.UUID, seq uint64) contracts.EventEnvelope {
		return contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    execID,
			Timestamp:      time.Now(),
			Sequence:       seq,
		}
	}

	_ = sink.Publish(context.Background(), makeEvent(execA, 1))
	_ = sink.Publish(context.Background(), makeEvent(execA, 2))
	_ = sink.Publish(context.Background(), makeEvent(execB, 1))

	// Close execution for execA only
	sink.CloseExecution(execA)

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event after closing execA, got %d", len(events))
	}
	if events[0].ExecutionID != execB {
		t.Fatalf("expected remaining event to be from execB, got %s", events[0].ExecutionID)
	}

	// Drop counters should also be cleared
	counters := sink.DropCounters()
	if _, exists := counters[execA]; exists {
		t.Fatalf("expected drop counters for execA to be cleared")
	}
}

func TestMemorySinkEventOrdering(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 100, PerAttempt: 50}
	sink := NewMemorySink(limits)
	execID := uuid.New()

	// Publish events out of order
	for _, seq := range []uint64{3, 1, 5, 2, 4} {
		_ = sink.Publish(context.Background(), contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    execID,
			Timestamp:      time.Now(),
			Sequence:       seq,
		})
	}

	events := sink.Events()
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}

	// Events should be in insertion order (not sorted by sequence)
	expectedOrder := []uint64{3, 1, 5, 2, 4}
	for i, ev := range events {
		if ev.Sequence != expectedOrder[i] {
			t.Errorf("event %d: expected sequence %d, got %d", i, expectedOrder[i], ev.Sequence)
		}
	}
}

func TestMemorySinkEmptyBuffer(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 10, PerAttempt: 5}
	sink := NewMemorySink(limits)

	events := sink.Events()
	if len(events) != 0 {
		t.Fatalf("expected empty events slice, got %d", len(events))
	}

	counters := sink.DropCounters()
	if len(counters) != 0 {
		t.Fatalf("expected empty counters map, got %+v", counters)
	}
}

func TestMemorySinkContextCancellation(t *testing.T) {
	limits := contracts.EventBufferLimits{PerExecution: 10, PerAttempt: 5}
	sink := NewMemorySink(limits)
	execID := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Publish with cancelled context - should still work as memory sink doesn't use context
	err := sink.Publish(ctx, contracts.EventEnvelope{
		SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		Kind:           contracts.EventKindStepTelemetry,
		ExecutionID:    execID,
		Timestamp:      time.Now(),
		Sequence:       1,
	})
	if err != nil {
		t.Fatalf("expected no error with cancelled context, got %v", err)
	}

	events := sink.Events()
	if len(events) != 1 {
		t.Fatalf("expected 1 event despite cancelled context, got %d", len(events))
	}
}

func TestMemorySinkDefaultLimitsUsed(t *testing.T) {
	// Test with default limits
	sink := NewMemorySink(contracts.DefaultEventBufferLimits)
	execID := uuid.New()

	// Should be able to publish many events without issues
	for i := uint64(1); i <= 100; i++ {
		_ = sink.Publish(context.Background(), contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    execID,
			Timestamp:      time.Now(),
			Sequence:       i,
		})
	}

	events := sink.Events()
	// With default limits, some events may be dropped
	if len(events) > 100 {
		t.Fatalf("expected at most 100 events, got %d", len(events))
	}
}
