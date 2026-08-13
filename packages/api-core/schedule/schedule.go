// Package schedule owns the in-process time seam used by api-core packages.
// Consumers depend on this small contract instead of declaring local Clock
// interfaces, which keeps production and test time behavior consistent.
package schedule

import (
	"time"
)

// Timer is the portion of time.Timer needed by scheduling consumers.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(time.Duration) bool
}

// Ticker is the portion of time.Ticker needed by scheduling consumers.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset(time.Duration)
}

// Clock is the repository-wide in-process time seam. Implementations must
// return wall-clock timestamps from Now and must preserve monotonic clock
// readings when they return timestamps suitable for elapsed-time calculation.
type Clock interface {
	Now() time.Time
	NewTimer(time.Duration) Timer
	NewTicker(time.Duration) Ticker
	Sleep(time.Duration)
}

// System returns the production clock backed by the Go standard library.
func System() Clock { return systemClock{} }

// Since reports elapsed monotonic time when start came from System().Now().
// time.Time carries the monotonic reading alongside its wall-clock value; using
// time.Since avoids wall-clock jumps affecting deadlines and duration metrics.
func Since(start time.Time) time.Duration { return time.Since(start) }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

func (systemClock) NewTimer(d time.Duration) Timer { return realTimer{time.NewTimer(d)} }

func (systemClock) NewTicker(d time.Duration) Ticker { return realTicker{time.NewTicker(d)} }

func (systemClock) Sleep(d time.Duration) { time.Sleep(d) }

type realTimer struct{ timer *time.Timer }

func (t realTimer) C() <-chan time.Time        { return t.timer.C }
func (t realTimer) Stop() bool                 { return t.timer.Stop() }
func (t realTimer) Reset(d time.Duration) bool { return t.timer.Reset(d) }

type realTicker struct{ ticker *time.Ticker }

func (t realTicker) C() <-chan time.Time   { return t.ticker.C }
func (t realTicker) Stop()                 { t.ticker.Stop() }
func (t realTicker) Reset(d time.Duration) { t.ticker.Reset(d) }
