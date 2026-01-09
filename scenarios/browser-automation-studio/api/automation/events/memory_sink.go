package events

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// MemorySink is an in-memory EventSink that enforces basic backpressure limits.
// It is intended for tests and early wiring; production sinks should handle
// durability and client fan-out.
type MemorySink struct {
	limits       contracts.EventBufferLimits
	mu           sync.Mutex
	events       []contracts.EventEnvelope
	dropCounters map[uuid.UUID]contracts.DropCounters
}

// NewMemorySink returns a MemorySink using the supplied limits (or defaults if
// limits are zero/invalid).
func NewMemorySink(limits contracts.EventBufferLimits) *MemorySink {
	if limits.PerExecution == 0 || limits.PerAttempt == 0 || limits.Validate() != nil {
		limits = contracts.DefaultEventBufferLimits
	}
	return &MemorySink{
		limits:       limits,
		dropCounters: make(map[uuid.UUID]contracts.DropCounters),
	}
}

// Publish applies a simple bounded-queue policy: telemetry/heartbeat events may
// be dropped when limits are exceeded, but completion/failure events are
// preserved. Drop counters are tracked per execution.
func (m *MemorySink) Publish(_ context.Context, event contracts.EventEnvelope) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	execID := event.ExecutionID
	var droppedSeq uint64

	// Enforce per-execution limit.
	if m.exceedsExecutionLimit(execID) && isDroppable(event.Kind) {
		if dropped, ok := m.dropOldest(execID, nil, nil); ok {
			counter := m.dropCounters[execID]
			counter.Dropped++
			if counter.OldestDropped == 0 && dropped > 0 {
				counter.OldestDropped = dropped
			}
			m.dropCounters[execID] = counter
			droppedSeq = dropped
		}
	}

	// Enforce per-attempt limit when step/attempt available.
	if event.StepIndex != nil && event.Attempt != nil && isDroppable(event.Kind) && m.exceedsAttemptLimit(execID, *event.StepIndex, *event.Attempt) {
		if dropped, ok := m.dropOldest(execID, event.StepIndex, event.Attempt); ok {
			counter := m.dropCounters[execID]
			counter.Dropped++
			if counter.OldestDropped == 0 && dropped > 0 {
				counter.OldestDropped = dropped
			}
			m.dropCounters[execID] = counter
			droppedSeq = dropped
		}
	}

	if counter := m.dropCounters[execID]; counter.Dropped > 0 {
		event.Drops = counter
		if event.Drops.OldestDropped == 0 && droppedSeq > 0 {
			event.Drops.OldestDropped = droppedSeq
		}
	}

	m.events = append(m.events, event)
	return nil
}

// Limits returns the configured buffer limits.
func (m *MemorySink) Limits() contracts.EventBufferLimits {
	return m.limits
}

// Events returns a copy of the stored events.
func (m *MemorySink) Events() []contracts.EventEnvelope {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]contracts.EventEnvelope, len(m.events))
	copy(out, m.events)
	return out
}

// DropCounters returns the per-execution drop statistics.
func (m *MemorySink) DropCounters() map[uuid.UUID]contracts.DropCounters {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[uuid.UUID]contracts.DropCounters, len(m.dropCounters))
	for k, v := range m.dropCounters {
		out[k] = v
	}
	return out
}

// CloseExecution clears buffered events and drop counters for a finished
// execution. Included to mirror WSHubSink lifecycle cleanup semantics.
func (m *MemorySink) CloseExecution(executionID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	filtered := m.events[:0]
	for _, ev := range m.events {
		if ev.ExecutionID != executionID {
			filtered = append(filtered, ev)
		}
	}
	m.events = filtered
	delete(m.dropCounters, executionID)
}

func (m *MemorySink) exceedsExecutionLimit(executionID uuid.UUID) bool {
	count := 0
	for _, ev := range m.events {
		if ev.ExecutionID == executionID {
			count++
		}
	}
	return count >= m.limits.PerExecution
}

func (m *MemorySink) exceedsAttemptLimit(executionID uuid.UUID, stepIndex, attempt int) bool {
	count := 0
	for _, ev := range m.events {
		if ev.ExecutionID == executionID && ev.StepIndex != nil && ev.Attempt != nil && *ev.StepIndex == stepIndex && *ev.Attempt == attempt {
			count++
		}
	}
	return count >= m.limits.PerAttempt
}

func (m *MemorySink) dropOldest(executionID uuid.UUID, stepIndex, attempt *int) (uint64, bool) {
	for i, ev := range m.events {
		if ev.ExecutionID != executionID || !isDroppable(ev.Kind) {
			continue
		}
		if stepIndex != nil && attempt != nil {
			if ev.StepIndex == nil || ev.Attempt == nil || *ev.StepIndex != *stepIndex || *ev.Attempt != *attempt {
				continue
			}
		}
		m.events = append(m.events[:i], m.events[i+1:]...)
		return ev.Sequence, true
	}
	return 0, false
}
