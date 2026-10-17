package agentsession

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// fakeHost is a process table and agent slice for reconciliation tests.
type fakeHost struct {
	facts    map[int]ProcessFact
	envs     map[int]map[string]string
	names    map[int]string
	started  map[int]time.Time
	members  map[string][]int
	activity map[string]time.Time
	memory   map[int][2]int64
}

func (h *fakeHost) ProcessMemory(pid int) (int64, int64, error) {
	m, ok := h.memory[pid]
	if !ok {
		return 0, 0, ErrProcessGone
	}
	return m[0], m[1], nil
}

func (h *fakeHost) Environment(pid int) (map[string]string, error) {
	env, ok := h.envs[pid]
	if !ok {
		return nil, errors.New("environment unreadable")
	}
	return env, nil
}

func (h *fakeHost) Process(pid int) (ProcessFact, error) {
	fact, ok := h.facts[pid]
	if !ok {
		return ProcessFact{}, ErrProcessGone
	}
	return fact, nil
}

func (h *fakeHost) ProcessName(pid int) (string, error) {
	name, ok := h.names[pid]
	if !ok {
		return "", ErrProcessGone
	}
	return name, nil
}

func (h *fakeHost) StartedAt(pid int) (time.Time, error) {
	at, ok := h.started[pid]
	if !ok {
		return time.Time{}, ErrProcessGone
	}
	return at, nil
}

func (h *fakeHost) ScopeMembers(unit string) ([]int, error) { return h.members[unit], nil }

func (h *fakeHost) TTYActivity(tty string) (time.Time, error) {
	at, ok := h.activity[tty]
	if !ok {
		return time.Time{}, errors.New("no device")
	}
	return at, nil
}

// fakeOwnerStore records backfills and refreshes.
type fakeOwnerStore struct {
	mu        sync.Mutex
	backfills map[string]scenarioruntime.EditorLeaseOwner
	refreshes map[string]scenarioruntime.EditorLeaseOwner
}

func newFakeOwnerStore() *fakeOwnerStore {
	return &fakeOwnerStore{backfills: map[string]scenarioruntime.EditorLeaseOwner{}, refreshes: map[string]scenarioruntime.EditorLeaseOwner{}}
}

func (s *fakeOwnerStore) BackfillEditorLeaseOwner(_ context.Context, id string, owner scenarioruntime.EditorLeaseOwner) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backfills[id] = owner
	return true, nil
}

func (s *fakeOwnerStore) RefreshEditorLeaseOwnerEvidence(_ context.Context, id string, owner scenarioruntime.EditorLeaseOwner) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshes[id] = owner
	return true, nil
}

// probeCalls counts calls per probe so a test can prove a probe was or was
// not asked.
type probeCalls struct {
	mu      sync.Mutex
	run     int
	console int
	start   int
	desktop int
}

func (c *probeCalls) inc(field *int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	*field++
}
