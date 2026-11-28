package localexchange

import (
	"testing"
	"time"
)

func TestRateLimiterIsPerPrincipalAndWindowed(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRateLimiter(2, time.Minute)
	if !r.Allow("unix:1", now) || !r.Allow("unix:1", now.Add(time.Second)) || r.Allow("unix:1", now.Add(2*time.Second)) {
		t.Fatal("principal should receive exactly two calls in the window")
	}
	if !r.Allow("unix:2", now) {
		t.Fatal("a second principal must have its own budget")
	}
	if !r.Allow("unix:1", now.Add(time.Minute)) {
		t.Fatal("budget should reset at the window boundary")
	}
}
