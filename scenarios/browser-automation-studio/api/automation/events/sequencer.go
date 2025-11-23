package events

import (
	"sync"

	"github.com/google/uuid"
)

// Sequencer generates monotonic sequence numbers per execution. It is safe for
// concurrent use by the executor and the EventSink.
type Sequencer interface {
	Next(executionID uuid.UUID) uint64
}

// PerExecutionSequencer tracks independent counters per execution ID.
type PerExecutionSequencer struct {
	mu       sync.Mutex
	counters map[uuid.UUID]uint64
}

// NewPerExecutionSequencer returns a zeroed sequencer safe for concurrent use.
func NewPerExecutionSequencer() *PerExecutionSequencer {
	return &PerExecutionSequencer{
		counters: make(map[uuid.UUID]uint64),
	}
}

// Next increments and returns the next sequence number for the execution. The
// first call for an execution returns 1 to reserve 0 for sentinel usage.
func (s *PerExecutionSequencer) Next(executionID uuid.UUID) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.counters[executionID] + 1
	s.counters[executionID] = next
	return next
}
