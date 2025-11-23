package events

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestSequencerAndEnvelopeMonotonicity(t *testing.T) {
	seq := NewPerExecutionSequencer()
	execID := uuid.New()
	sink := NewMemorySink(contracts.DefaultEventBufferLimits)

	for i := 0; i < 3; i++ {
		ev := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    execID,
			WorkflowID:     uuid.New(),
			StepIndex:      intPtr(i),
			Attempt:        intPtr(1),
			Sequence:       seq.Next(execID),
			Timestamp:      time.Now().UTC(),
			Payload: contracts.StepTelemetry{
				SchemaVersion:  contracts.TelemetrySchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				ExecutionID:    execID,
				StepIndex:      i,
				Attempt:        1,
				Kind:           contracts.TelemetryKindHeartbeat,
				Timestamp:      time.Now().UTC(),
			},
		}
		_ = sink.Publish(context.Background(), ev)
	}

	events := sink.Events()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	for i := 1; i < len(events); i++ {
		if events[i].Sequence <= events[i-1].Sequence {
			t.Fatalf("expected monotonic sequence, got %d after %d", events[i].Sequence, events[i-1].Sequence)
		}
	}
}

func intPtr(v int) *int { return &v }
