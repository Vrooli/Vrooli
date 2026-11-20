package capabilities

import (
	"context"
	"sync"
	"time"
)

// LatencyRecorder keeps the most recent execution time of a provider liveness
// check. The shared capability registry owns status and message but has no
// latency field, so the scenario times the checker at the seam and the health
// handler reports it. A cache hit reports the last real check's latency rather
// than inventing a fresh number.
type LatencyRecorder struct {
	mu   sync.RWMutex
	last map[string]time.Duration
}

// NewLatencyRecorder returns an empty recorder.
func NewLatencyRecorder() *LatencyRecorder {
	return &LatencyRecorder{last: make(map[string]time.Duration)}
}

// Wrap returns a checker that records the duration of every Check call. A nil
// recorder or checker is returned unchanged so wiring stays optional.
func (r *LatencyRecorder) Wrap(providerID string, checker Checker) Checker {
	if r == nil || checker == nil {
		return checker
	}
	return &timedChecker{providerID: providerID, inner: checker, recorder: r}
}

// WrapAll wraps every checker in a new map, preserving the input map.
func (r *LatencyRecorder) WrapAll(checkers map[string]Checker) map[string]Checker {
	if r == nil || len(checkers) == 0 {
		return checkers
	}
	out := make(map[string]Checker, len(checkers))
	for id, checker := range checkers {
		out[id] = r.Wrap(id, checker)
	}
	return out
}

// Last returns the most recent measured duration for a provider.
func (r *LatencyRecorder) Last(providerID string) (time.Duration, bool) {
	if r == nil {
		return 0, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	duration, ok := r.last[providerID]
	return duration, ok
}

func (r *LatencyRecorder) record(providerID string, duration time.Duration) {
	r.mu.Lock()
	r.last[providerID] = duration
	r.mu.Unlock()
}

type timedChecker struct {
	providerID string
	inner      Checker
	recorder   *LatencyRecorder
}

func (c *timedChecker) Check(ctx context.Context) (Status, string) {
	start := time.Now()
	status, message := c.inner.Check(ctx)
	c.recorder.record(c.providerID, time.Since(start))
	return status, message
}
