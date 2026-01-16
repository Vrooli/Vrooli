// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// GoroutineChecker helps detect goroutine leaks in tests.
// Create at the start of a test and call Check() at the end.
type GoroutineChecker struct {
	t              *testing.T
	initialCount   int
	initialStacks  string
	allowedGrowth  int
	graceTime      time.Duration
}

// NewGoroutineChecker creates a new goroutine leak detector.
// The default allowed growth is 0 (no new goroutines should remain).
func NewGoroutineChecker(t *testing.T) *GoroutineChecker {
	return &GoroutineChecker{
		t:             t,
		initialCount:  runtime.NumGoroutine(),
		initialStacks: captureStacks(),
		allowedGrowth: 0,
		graceTime:     100 * time.Millisecond,
	}
}

// AllowGrowth sets the number of additional goroutines that are acceptable.
// Use this for tests that intentionally start background workers.
func (g *GoroutineChecker) AllowGrowth(n int) *GoroutineChecker {
	g.allowedGrowth = n
	return g
}

// GraceTime sets the time to wait for goroutines to complete before checking.
// Default is 100ms.
func (g *GoroutineChecker) GraceTime(d time.Duration) *GoroutineChecker {
	g.graceTime = d
	return g
}

// Check verifies that no unexpected goroutines were leaked.
// Should be called at the end of the test (defer g.Check()).
func (g *GoroutineChecker) Check() {
	g.t.Helper()

	// Wait a brief period for goroutines to finish
	time.Sleep(g.graceTime)

	// Give goroutines a bit more time to clean up if needed
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > g.initialCount+g.allowedGrowth && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}

	finalCount := runtime.NumGoroutine()
	growth := finalCount - g.initialCount

	if growth > g.allowedGrowth {
		g.t.Errorf("Goroutine leak detected: started with %d, ended with %d (growth: %d, allowed: %d)\n\nInitial stacks:\n%s\n\nFinal stacks:\n%s",
			g.initialCount, finalCount, growth, g.allowedGrowth,
			g.initialStacks, captureStacks())
	}
}

// captureStacks returns a string representation of all goroutine stacks.
func captureStacks() string {
	buf := make([]byte, 1024*1024)
	n := runtime.Stack(buf, true)
	return string(buf[:n])
}

// CountGoroutines returns the current goroutine count.
func CountGoroutines() int {
	return runtime.NumGoroutine()
}

// WaitForGoroutineCount waits for the goroutine count to reach the target.
// Returns true if the target was reached within the timeout.
func WaitForGoroutineCount(target int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= target {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// FilterStacks filters goroutine stacks to show only those containing the given substring.
// Useful for debugging to find specific goroutines.
func FilterStacks(substring string) string {
	stacks := captureStacks()
	var filtered []string
	for _, stack := range strings.Split(stacks, "\n\n") {
		if strings.Contains(stack, substring) {
			filtered = append(filtered, stack)
		}
	}
	return strings.Join(filtered, "\n\n")
}
