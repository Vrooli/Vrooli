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
