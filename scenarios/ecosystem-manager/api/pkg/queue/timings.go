package queue

import (
	"sync"
	"time"
)

var (
	timingScaleMu sync.RWMutex
	timingScale   = 1.0
)

// scaleDuration applies a global timing scale (primarily for tests).
func scaleDuration(d time.Duration) time.Duration {
	timingScaleMu.RLock()
	scale := timingScale
	timingScaleMu.RUnlock()

	if scale == 1.0 {
		return d
	}

	return time.Duration(float64(d) * scale)
}

// SetTimingScaleForTests adjusts timing-sensitive sleeps/tickers.
// Returns a reset func that should be deferred to restore defaults.
func SetTimingScaleForTests(scale float64) func() {
	timingScaleMu.Lock()
	prev := timingScale
	if scale <= 0 {
		scale = 1.0
	}
	timingScale = scale
	timingScaleMu.Unlock()

	return func() {
		timingScaleMu.Lock()
		timingScale = prev
		timingScaleMu.Unlock()
	}
}
