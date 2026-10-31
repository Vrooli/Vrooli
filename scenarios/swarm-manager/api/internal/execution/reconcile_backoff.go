package execution

import (
	"sync"
	"time"

	"swarm-manager/internal/agentmanager"
)

const (
	// reconcileBaseInterval is the first re-check delay after an unchanged
	// observation; it equals the background tick so a fresh record is looked
	// at on consecutive ticks until it proves stable.
	reconcileBaseInterval = 2 * time.Second
	// reconcileMaxInterval caps the growth of the re-check delay.
	reconcileMaxInterval = 60 * time.Second
	// reconcileNeedsReviewInterval is the floor for needs_review records: they
	// are non-transient (an operator must act), so re-tracing them every tick
	// buys nothing.
	reconcileNeedsReviewInterval = 60 * time.Second
)

// reconcileTracker remembers, per execution record, what the last workflow
// observation looked like and when the record is next due for a trace read.
// It is in-memory only: a restart simply re-traces everything once.
type reconcileTracker struct {
	mu      sync.Mutex
	now     func() time.Time
	entries map[string]*reconcileEntry
}

type reconcileEntry struct {
	workflowID  string
	status      Status
	observation string
	interval    time.Duration
	nextCheck   time.Time
}

func newReconcileTracker() *reconcileTracker {
	return &reconcileTracker{now: time.Now, entries: make(map[string]*reconcileEntry)}
}

// due reports whether candidate should be traced on this pass. A record whose
// local status or workflow id changed since the last observation is always due.
func (t *reconcileTracker) due(candidate Record, workflowID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[candidate.ExecutionID]
	if !ok || entry.workflowID != workflowID || entry.status != candidate.Status {
		return true
	}
	return !t.now().Before(entry.nextCheck)
}

// observe records the outcome of a trace read for candidate and schedules the
// next check: unchanged observations back off geometrically up to the cap,
// while any change (or a first sighting) resets to the base interval.
func (t *reconcileTracker) observe(candidate Record, workflowID string, state agentmanager.WorkflowExecutionState, err error) {
	observation := "err"
	if err == nil {
		observation = state.Status.String() + "|" + state.UpdatedAt + "|" + state.TerminalCode
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[candidate.ExecutionID]
	if !ok {
		entry = &reconcileEntry{}
		t.entries[candidate.ExecutionID] = entry
	}
	unchanged := ok && entry.workflowID == workflowID && entry.status == candidate.Status && entry.observation == observation
	if unchanged {
		entry.interval = min(entry.interval*2, reconcileMaxInterval)
	} else {
		entry.interval = reconcileBaseInterval
	}
	if candidate.Status == StatusNeedsReview && entry.interval < reconcileNeedsReviewInterval {
		entry.interval = reconcileNeedsReviewInterval
	}
	entry.workflowID = workflowID
	entry.status = candidate.Status
	entry.observation = observation
	entry.nextCheck = t.now().Add(entry.interval)
}

// retain drops tracking for records that are no longer inspectable so the
// map stays bounded by the live candidate set.
func (t *reconcileTracker) retain(live map[string]struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id := range t.entries {
		if _, ok := live[id]; !ok {
			delete(t.entries, id)
		}
	}
}
