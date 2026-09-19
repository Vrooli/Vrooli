package livesearch

import (
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// Default budget governor settings: how many live SearXNG calls are allowed per
// rolling window. Live web search is an EXTERNAL, rate-limited dependency, so
// the governor caps spend and degrades gracefully rather than hammering it.
const (
	DefaultGovernorCapacity = 60
	DefaultGovernorWindow   = time.Minute
)

// Governor is a fixed-window token-bucket rate limiter for live SearXNG calls.
// It refills capacity tokens at the start of each window. On exhaustion the
// service returns a degraded response WITHOUT calling SearXNG. The clock is
// injected so window rollover is deterministic in tests.
type Governor struct {
	capacity int
	window   time.Duration
	clock    schedule.Clock

	mu          sync.Mutex
	tokens      int
	windowStart time.Time
}

// NewGovernor constructs a token-bucket governor. A non-positive capacity falls
// back to DefaultGovernorCapacity; a non-positive window to DefaultGovernorWindow;
// a nil clock to schedule.System. The bucket starts full.
func NewGovernor(capacity int, window time.Duration, clk schedule.Clock) *Governor {
	if capacity <= 0 {
		capacity = DefaultGovernorCapacity
	}
	if window <= 0 {
		window = DefaultGovernorWindow
	}
	if clk == nil {
		clk = schedule.System()
	}
	return &Governor{
		capacity:    capacity,
		window:      window,
		clock:       clk,
		tokens:      capacity,
		windowStart: clk.Now(),
	}
}

// Allow consumes one token, refilling the bucket first if the current window
// has elapsed. It returns false when no tokens remain in the active window —
// the caller must then degrade without calling the upstream.
func (g *Governor) Allow() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.clock.Now()
	if now.Sub(g.windowStart) >= g.window {
		g.tokens = g.capacity
		g.windowStart = now
	}
	if g.tokens <= 0 {
		return false
	}
	g.tokens--
	return true
}
