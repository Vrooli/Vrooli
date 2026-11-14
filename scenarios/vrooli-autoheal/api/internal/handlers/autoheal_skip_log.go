package handlers

import (
	"strings"
	"sync"
)

// A skip is a steady state, not an event. Writing one row per tick per skipped
// check turned the action history into a tape of the same sentence: resource-
// reranker alone logged "auto-heal not enabled for this check" roughly 9,600
// times in ten hours, and the one row that mattered — the first one — was
// indistinguishable from the 9,599 that followed.
//
// skipLogGate keeps the first row and every row that says something new,
// and drops the repeats.

type skipLogGate struct {
	mu     sync.Mutex
	lastBy map[string]string
}

func newSkipLogGate() *skipLogGate {
	return &skipLogGate{lastBy: map[string]string{}}
}

// ShouldLog reports whether this skip reason differs from the last one recorded
// for the check, and records it either way. An empty reason is never logged.
func (g *skipLogGate) ShouldLog(checkID, reason string) bool {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return false
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if previous, seen := g.lastBy[checkID]; seen && previous == reason {
		return false
	}
	g.lastBy[checkID] = reason
	return true
}

// Clear forgets a check's last skip reason, so the next skip is recorded again.
// It is called when a check heals or succeeds: the next time it starts skipping
// is a new state change, not a continuation of the old one.
func (g *skipLogGate) Clear(checkID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.lastBy, checkID)
}
