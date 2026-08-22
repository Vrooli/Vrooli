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
	mu       sync.Mutex
	lastVal  uint64
	lastTime time.Time
	primed   bool
}

// observe records a cumulative sample and returns the rate per second since the
// previous sample. ok is false for the first sample (no interval yet) and when
// the counter goes backwards, which means the host rebooted — reporting a huge
// negative-turned-positive delta there would be worse than reporting nothing.
func (t *forkRateTracker) observe(total uint64, now time.Time) (rate float64, ok bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	prevVal, prevTime, primed := t.lastVal, t.lastTime, t.primed
	t.lastVal, t.lastTime, t.primed = total, now, true

	if !primed {
		return 0, false
	}
	if total < prevVal {
		return 0, false
	}
	elapsed := now.Sub(prevTime).Seconds()
	if elapsed <= 0 {
		return 0, false
	}
	return float64(total-prevVal) / elapsed, true
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
