package diagnostics

import "sync"

// LastRunStore is the seam between the orchestrator and run history.
// The default in-memory implementation keeps a fixed-size ring; a
// future SQLite-backed implementation can drop in by satisfying this
// interface without touching the orchestrator.
type LastRunStore interface {
	Record(run Run)
	Latest() Run
	Recent() []Run
}

// inMemoryStore keeps the last N runs in insertion order with the
// newest first. Mutex-guarded for safe concurrent Record/Latest from
// the orchestrator and the GetLastRun handler.
type inMemoryStore struct {
	mu    sync.Mutex
	cap   int
	items []Run // index 0 is newest
}

// NewLastRunStore returns an in-memory store that retains the last cap
// runs. A cap <= 0 is treated as 1 (keep only the most recent).
func NewLastRunStore(cap int) LastRunStore {
	if cap <= 0 {
		cap = 1
	}
	return &inMemoryStore{cap: cap}
}

func (s *inMemoryStore) Record(run Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append([]Run{run}, s.items...)
	if len(s.items) > s.cap {
		s.items = s.items[:s.cap]
	}
}

func (s *inMemoryStore) Latest() Run {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return Run{}
	}
	return s.items[0]
}

func (s *inMemoryStore) Recent() []Run {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Run, len(s.items))
	copy(out, s.items)
	return out
}
