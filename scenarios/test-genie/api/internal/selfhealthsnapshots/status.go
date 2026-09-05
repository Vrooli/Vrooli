package selfhealthsnapshots

import (
	"sync"
	"time"
)

// SweepStatus is the cached advisory-work state exposed to serving code. It is
// deliberately data-only: reading it never triggers a rollup or database call.
type SweepStatus struct {
	StartedAt   time.Time
	CompletedAt time.Time
	Duration    time.Duration
	RunCount    int
	Deadline    time.Duration
	PoolWaits   int64
	PoolWait    time.Duration
	PoolOpen    int
	PoolInUse   int
	Outcome     string // succeeded | failed | timed_out
	Error       string
}

type StatusStore struct {
	mu    sync.RWMutex
	value SweepStatus
}

func (s *StatusStore) Record(v SweepStatus)  { s.mu.Lock(); s.value = v; s.mu.Unlock() }
func (s *StatusStore) Snapshot() SweepStatus { s.mu.RLock(); defer s.mu.RUnlock(); return s.value }
