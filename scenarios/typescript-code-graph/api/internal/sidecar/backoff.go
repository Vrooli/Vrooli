package sidecar

import "time"

// Restart backoff schedule per plan §7 Phase 3.
//
//	100ms, 200ms, 400ms, 800ms, 1.6s, then 5s cap.
//
// Budget: 5 restarts inside any rolling 60-second window. When the
// budget is exhausted the supervisor moves to STATUS_PERMANENTLY_UNHEALTHY
// and stops respawning.
const (
	restartBudgetCount  = 5
	restartBudgetWindow = 60 * time.Second
	backoffCap          = 5 * time.Second
)

// backoffSchedule returns the i-th wait duration (0-indexed). i may
// exceed the explicit table; the cap applies thereafter.
func backoffSchedule(i int) time.Duration {
	table := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
	}
	if i < 0 {
		i = 0
	}
	if i < len(table) {
		return table[i]
	}
	return backoffCap
}

// restartLedger tracks restart timestamps in a sliding window. It is
// NOT safe for concurrent use; callers must hold the supervisor's
// mutex.
type restartLedger struct {
	times []time.Time
}

// record adds a restart timestamp and discards entries older than the
// rolling window relative to now.
func (l *restartLedger) record(now time.Time) {
	cutoff := now.Add(-restartBudgetWindow)
	// Drop expired entries from the front.
	trimmed := l.times[:0]
	for _, t := range l.times {
		if !t.Before(cutoff) {
			trimmed = append(trimmed, t)
		}
	}
	l.times = append(trimmed, now)
}

// exhausted reports whether the budget has been used up.
func (l *restartLedger) exhausted() bool {
	return len(l.times) >= restartBudgetCount
}

// count is exposed for tests.
func (l *restartLedger) count() int { return len(l.times) }
