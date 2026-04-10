// Package clock provides time abstraction for testable code.
package clock

import (
	"sync"
	"time"
)

// MockClock provides a controllable clock for testing.
// All methods are safe for concurrent use.
type MockClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewMock creates a new MockClock set to the given time.
// If t is zero, it defaults to a fixed time: 2024-01-01 00:00:00 UTC.
func NewMock(t time.Time) *MockClock {
	if t.IsZero() {
		t = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return &MockClock{now: t}
}

// Now returns the mock's current time.
func (c *MockClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Set updates the clock to the given time.
func (c *MockClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

// Advance moves the clock forward by the given duration.
func (c *MockClock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	return c.now
}

// AdvanceTo moves the clock to the given time.
// If t is before the current time, it is set anyway.
func (c *MockClock) AdvanceTo(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}
