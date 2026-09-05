package collectors

import (
	"sync"
	"time"
)

// forkRateReading is one platform's answer for host process-creation rate.
// A cumulative counter is the honest primitive: the rate is a derivative this
// process computes between cycles, so a restart yields "no rate yet" rather
// than a fabricated spike.
type forkRateReading struct {
	// total is the cumulative count of processes created since host boot.
	total uint64
	// supported is false when the platform exposes no such counter.
	supported bool
	// reason explains an unsupported reading.
	reason string
	// provenance names the backend that produced the reading.
	provenance string
}

func forkRateUnsupported(reason string) forkRateReading {
	return forkRateReading{supported: false, reason: reason, provenance: "platform backend"}
}

// forkRateTracker converts a monotonic cumulative counter into a per-second
// rate. It is the diagnostic that names a fork storm outright: during the
// 2026-08-21 incident the host sustained ~2,481 forks/sec while every collected
// metric showed only the symptoms (load, CPU, stalls) and none the cause.
type forkRateTracker struct {
	tracker counterRateTracker
}

// counterRateTracker converts any named monotonic counter into a rate. Keeping
// the previous sample per name is important because vmstat exposes many
// counters in one read and they must not share an interval or baseline.
type counterRateTracker struct {
	mu   sync.Mutex
	last map[string]counterRateSample
}

type counterRateSample struct {
	total uint64
	at    time.Time
}

func newCounterRateTracker() *counterRateTracker {
	return &counterRateTracker{last: make(map[string]counterRateSample)}
}

func (t *counterRateTracker) observe(name string, total uint64, now time.Time) (rate float64, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.last == nil {
		t.last = make(map[string]counterRateSample)
	}

	previous, primed := t.last[name]
	t.last[name] = counterRateSample{total: total, at: now}
	if !primed || total < previous.total {
		return 0, false
	}
	elapsed := now.Sub(previous.at).Seconds()
	if elapsed <= 0 {
		return 0, false
	}
	return float64(total-previous.total) / elapsed, true
}

func counterRateValues(tracker *counterRateTracker, name string, total uint64, now time.Time) map[string]interface{} {
	values := map[string]interface{}{
		name + "_total":        total,
		name + "_rate_status":  "not_yet_sampled",
		name + "_rate_primed":  false,
		name + "_rate_pending": true,
	}
	if rate, ok := tracker.observe(name, total, now); ok {
		values[name+"_per_second"] = rate
		values[name+"_rate_status"] = "measured"
		values[name+"_rate_primed"] = true
		values[name+"_rate_pending"] = false
	}
	return values
}

// observe records a cumulative sample and returns the rate per second since the
// previous sample. ok is false for the first sample (no interval yet) and when
// the counter goes backwards, which means the host rebooted — reporting a huge
// negative-turned-positive delta there would be worse than reporting nothing.
func (t *forkRateTracker) observe(total uint64, now time.Time) (rate float64, ok bool) {
	return t.tracker.observe("forks", total, now)
}

// forkRateValues renders the tracker's view for a metric payload. Callers merge
// the result into their collector's Values map.
func forkRateValues(tracker *forkRateTracker, reading forkRateReading, now time.Time) map[string]interface{} {
	if !reading.supported {
		return map[string]interface{}{
			"fork_rate_status": "unsupported",
			"fork_rate_reason": reading.reason,
		}
	}
	values := map[string]interface{}{
		"forks_total":       reading.total,
		"fork_rate_source":  reading.provenance,
		"fork_rate_status":  "measured",
		"forks_per_second":  float64(0),
		"fork_rate_primed":  false,
		"fork_rate_pending": true,
	}
	if rate, ok := tracker.observe(reading.total, now); ok {
		values["forks_per_second"] = rate
		values["fork_rate_primed"] = true
		values["fork_rate_pending"] = false
	}
	return values
}
