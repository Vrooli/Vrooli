package events

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestSequencerAndEnvelopeMonotonicity(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] produces monotonically increasing sequences", func(t *testing.T) {
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
		require.Len(t, events, 3)
		for i := 1; i < len(events); i++ {
			assert.Greater(t, events[i].Sequence, events[i-1].Sequence,
				"sequence should be monotonically increasing")
		}
	})
}

func TestPerExecutionSequencer(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] first sequence starts at 1", func(t *testing.T) {
		seq := NewPerExecutionSequencer()
		execID := uuid.New()

		first := seq.Next(execID)

		assert.Equal(t, uint64(1), first, "first sequence should be 1, not 0")
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] sequences increment correctly", func(t *testing.T) {
		seq := NewPerExecutionSequencer()
		execID := uuid.New()

		for expected := uint64(1); expected <= 10; expected++ {
			actual := seq.Next(execID)
			assert.Equal(t, expected, actual)
		}
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] isolates sequences per execution", func(t *testing.T) {
		seq := NewPerExecutionSequencer()
		exec1 := uuid.New()
		exec2 := uuid.New()

		// Advance exec1 several times
		for i := 0; i < 5; i++ {
			seq.Next(exec1)
		}

		// exec2 should start fresh at 1
		first := seq.Next(exec2)
		assert.Equal(t, uint64(1), first, "new execution should start at 1")

		// exec1 should continue from 6
		next := seq.Next(exec1)
		assert.Equal(t, uint64(6), next, "exec1 should continue where it left off")
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles concurrent access safely", func(t *testing.T) {
		seq := NewPerExecutionSequencer()
		execID := uuid.New()

		const goroutines = 10
		const callsPerGoroutine = 100

		var wg sync.WaitGroup
		results := make(chan uint64, goroutines*callsPerGoroutine)

		wg.Add(goroutines)
		for g := 0; g < goroutines; g++ {
			go func() {
				defer wg.Done()
				for i := 0; i < callsPerGoroutine; i++ {
					results <- seq.Next(execID)
				}
			}()
		}

		wg.Wait()
		close(results)

		// Collect all sequences
		seen := make(map[uint64]bool)
		for seqNum := range results {
			assert.False(t, seen[seqNum], "sequence %d should not be duplicated", seqNum)
			seen[seqNum] = true
		}

		// Verify we got all expected sequences
		assert.Len(t, seen, goroutines*callsPerGoroutine)
		for expected := uint64(1); expected <= goroutines*callsPerGoroutine; expected++ {
			assert.True(t, seen[expected], "expected sequence %d to be present", expected)
		}
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles multiple executions concurrently", func(t *testing.T) {
		seq := NewPerExecutionSequencer()
		exec1 := uuid.New()
		exec2 := uuid.New()

		const callsPerExec = 50
		var wg sync.WaitGroup

		results1 := make(chan uint64, callsPerExec)
		results2 := make(chan uint64, callsPerExec)

		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < callsPerExec; i++ {
				results1 <- seq.Next(exec1)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < callsPerExec; i++ {
				results2 <- seq.Next(exec2)
			}
		}()

		wg.Wait()
		close(results1)
		close(results2)

		// Both should have independent sequence sets
		seen1 := make(map[uint64]bool)
		seen2 := make(map[uint64]bool)

		for s := range results1 {
			seen1[s] = true
		}
		for s := range results2 {
			seen2[s] = true
		}

		// Both should have 1 through callsPerExec
		for expected := uint64(1); expected <= callsPerExec; expected++ {
			assert.True(t, seen1[expected], "exec1 missing sequence %d", expected)
			assert.True(t, seen2[expected], "exec2 missing sequence %d", expected)
		}
	})
}

func TestMemorySinkWithEventEnvelope(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] stores events correctly", func(t *testing.T) {
		sink := NewMemorySink(contracts.DefaultEventBufferLimits)
		execID := uuid.New()
		workflowID := uuid.New()

		ev := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepCompleted,
			ExecutionID:    execID,
			WorkflowID:     workflowID,
			StepIndex:      intPtr(0),
			Attempt:        intPtr(1),
			Sequence:       1,
			Timestamp:      time.Now().UTC(),
			Payload:        map[string]any{"result": "success"},
		}

		err := sink.Publish(context.Background(), ev)

		require.NoError(t, err)
		events := sink.Events()
		require.Len(t, events, 1)
		assert.Equal(t, execID, events[0].ExecutionID)
		assert.Equal(t, workflowID, events[0].WorkflowID)
		assert.Equal(t, contracts.EventKindStepCompleted, events[0].Kind)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] preserves event order", func(t *testing.T) {
		sink := NewMemorySink(contracts.DefaultEventBufferLimits)
		execID := uuid.New()
		workflowID := uuid.New()

		for i := 1; i <= 5; i++ {
			ev := contracts.EventEnvelope{
				SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				Kind:           contracts.EventKindStepTelemetry,
				ExecutionID:    execID,
				WorkflowID:     workflowID,
				StepIndex:      intPtr(i - 1),
				Attempt:        intPtr(1),
				Sequence:       uint64(i),
				Timestamp:      time.Now().UTC(),
			}
			err := sink.Publish(context.Background(), ev)
			require.NoError(t, err)
		}

		events := sink.Events()
		require.Len(t, events, 5)
		for i, ev := range events {
			assert.Equal(t, uint64(i+1), ev.Sequence, "events should maintain order")
		}
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] handles multiple event kinds", func(t *testing.T) {
		sink := NewMemorySink(contracts.DefaultEventBufferLimits)
		execID := uuid.New()
		workflowID := uuid.New()

		kinds := []contracts.EventKind{
			contracts.EventKindStepTelemetry,
			contracts.EventKindStepCompleted,
			contracts.EventKindExecutionCompleted,
		}

		for i, kind := range kinds {
			ev := contracts.EventEnvelope{
				SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
				PayloadVersion: contracts.PayloadVersion,
				Kind:           kind,
				ExecutionID:    execID,
				WorkflowID:     workflowID,
				StepIndex:      intPtr(i),
				Attempt:        intPtr(1),
				Sequence:       uint64(i + 1),
				Timestamp:      time.Now().UTC(),
			}
			err := sink.Publish(context.Background(), ev)
			require.NoError(t, err)
		}

		events := sink.Events()
		require.Len(t, events, 3)
		for i, ev := range events {
			assert.Equal(t, kinds[i], ev.Kind)
		}
	})
}

func TestEventEnvelopeValidation(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] validates schema version constant", func(t *testing.T) {
		assert.NotEmpty(t, contracts.EventEnvelopeSchemaVersion)
		assert.NotEmpty(t, contracts.PayloadVersion)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] event kinds are distinct", func(t *testing.T) {
		kinds := []contracts.EventKind{
			contracts.EventKindStepTelemetry,
			contracts.EventKindStepCompleted,
			contracts.EventKindExecutionCompleted,
		}

		seen := make(map[contracts.EventKind]bool)
		for _, kind := range kinds {
			assert.False(t, seen[kind], "event kind %s should be unique", kind)
			seen[kind] = true
		}
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] telemetry kinds are distinct", func(t *testing.T) {
		kinds := []contracts.TelemetryKind{
			contracts.TelemetryKindHeartbeat,
			contracts.TelemetryKindScreenshot,
			contracts.TelemetryKindDOMSnapshot,
		}

		seen := make(map[contracts.TelemetryKind]bool)
		for _, kind := range kinds {
			assert.False(t, seen[kind], "telemetry kind %s should be unique", kind)
			seen[kind] = true
		}
	})
}

func TestDropCounters(t *testing.T) {
	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] drop counters initialized to zero", func(t *testing.T) {
		drops := contracts.DropCounters{}

		assert.Equal(t, uint64(0), drops.Dropped)
		assert.Equal(t, uint64(0), drops.OldestDropped)
	})

	t.Run("[REQ:BAS-EXEC-TELEMETRY-STREAM] drop counters propagate through envelope", func(t *testing.T) {
		env := contracts.EventEnvelope{
			SchemaVersion:  contracts.EventEnvelopeSchemaVersion,
			PayloadVersion: contracts.PayloadVersion,
			Kind:           contracts.EventKindStepTelemetry,
			ExecutionID:    uuid.New(),
			WorkflowID:     uuid.New(),
			Sequence:       1,
			Timestamp:      time.Now().UTC(),
			Drops: contracts.DropCounters{
				Dropped:       5,
				OldestDropped: 10,
			},
		}

		assert.Equal(t, uint64(5), env.Drops.Dropped)
		assert.Equal(t, uint64(10), env.Drops.OldestDropped)
	})
}

func intPtr(v int) *int { return &v }
