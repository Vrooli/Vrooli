package retention

import (
	"sort"
	"sync"
	"time"
)

// CycleStatus is the latest retention cycle's per-entry outcome, including
// budget alarms, owner reclaim receipts, and escalations. It exists because
// Enforce's results were discarded: a breached non-regenerable budget was
// visible only as a one-shot event.
type CycleStatus struct {
	ObservedAt time.Time `json:"observed_at"`
	Error      string    `json:"error,omitempty"`
	OverBudget []Result  `json:"over_budget"`
	Escalated  []Result  `json:"escalated"`
	Entries    []Result  `json:"entries"`
}

var latestCycle struct {
	sync.Mutex
	status CycleStatus
	set    bool
}

// RecordCycle stores one cycle's results for the status surface.
func RecordCycle(at time.Time, results map[string]Result, cycleErr error) {
	status := CycleStatus{ObservedAt: at.UTC(), OverBudget: []Result{}, Escalated: []Result{}, Entries: []Result{}}
	if cycleErr != nil {
		status.Error = cycleErr.Error()
	}
	for _, owner := range results {
		entries := owner.EntryResults
		if len(entries) == 0 {
			entries = []Result{owner}
		}
		for _, entry := range entries {
			entry.EntryResults = nil
			status.Entries = append(status.Entries, entry)
			if entry.OverBytes > 0 {
				status.OverBudget = append(status.OverBudget, entry)
			}
			if entry.Escalated {
				status.Escalated = append(status.Escalated, entry)
			}
		}
	}
	for _, list := range [][]Result{status.Entries, status.OverBudget, status.Escalated} {
		sort.Slice(list, func(i, j int) bool {
			if list[i].Owner != list[j].Owner {
				return list[i].Owner < list[j].Owner
			}
			return list[i].Entry < list[j].Entry
		})
	}
	latestCycle.Lock()
	latestCycle.status, latestCycle.set = status, true
	latestCycle.Unlock()
}

// LatestCycle returns the most recent recorded cycle, if any has run.
func LatestCycle() (CycleStatus, bool) {
	latestCycle.Lock()
	defer latestCycle.Unlock()
	return latestCycle.status, latestCycle.set
}
