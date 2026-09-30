// Package clock contains the shared time seam for control-plane components.
package clock

import (
	"sync"
	"time"
)

// Clock supplies the current time to code that needs deterministic tests.
type Clock interface {
	Now() time.Time
}

// Real is the production clock. It normalizes observations to UTC at the
// boundary so callers do not accidentally persist local wall-clock zones.
type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

// Fake is a concurrency-safe manually controlled clock for tests.
type Fake struct {
	mu  sync.Mutex
	now time.Time
}

func NewFake(start time.Time) *Fake { return &Fake{now: start.UTC()} }

func (c *Fake) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Fake) Set(value time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = value.UTC()
}

func (c *Fake) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}
