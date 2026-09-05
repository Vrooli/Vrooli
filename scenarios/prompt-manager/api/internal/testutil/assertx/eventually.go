package assertx

import (
	"testing"
	"time"
)

// Eventually polls fn at the given interval until it returns nil or timeout
// elapses. On timeout the test fails with the last error fn returned.
//
// Useful for waiting on goroutine-driven state (e.g., a sweeper tick) without
// scattering time.Sleep calls through tests.
func Eventually(t testing.TB, timeout, interval time.Duration, fn func() error) {
	t.Helper()
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(interval)
	}
	if lastErr != nil {
		t.Fatalf("Eventually: condition not met within %s: %v", timeout, lastErr)
	} else {
		t.Fatalf("Eventually: condition not met within %s", timeout)
	}
}
