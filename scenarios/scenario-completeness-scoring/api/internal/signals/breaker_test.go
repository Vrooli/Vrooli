package signals

import (
	"testing"
	"time"
)

// fakeClock drives breaker time without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }
func newFakeClock() *fakeClock               { return &fakeClock{t: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)} }

func TestBreakerOpensAfterThresholdFailures(t *testing.T) {
	clock := newFakeClock()
	b := newBreaker(clock.now)

	for i := 0; i < failureThreshold-1; i++ {
		b.recordFailure()
		if !b.allow() {
			t.Fatalf("breaker open after %d failures; want still closed", i+1)
		}
	}
	b.recordFailure()
	if b.allow() {
		t.Fatalf("breaker still allows after %d failures; want open", failureThreshold)
	}
}

func TestBreakerHalfOpenAfterRetryInterval(t *testing.T) {
	tests := []struct {
		name        string
		probeFails  bool
		wantAllowed bool
	}{
		{name: "probe success closes", probeFails: false, wantAllowed: true},
		{name: "probe failure reopens", probeFails: true, wantAllowed: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := newFakeClock()
			b := newBreaker(clock.now)
			for i := 0; i < failureThreshold; i++ {
				b.recordFailure()
			}
			if b.allow() {
				t.Fatal("breaker should be open before retry interval")
			}

			clock.advance(retryInterval)
			if !b.allow() {
				t.Fatal("breaker should allow a half-open probe after retry interval")
			}

			if tt.probeFails {
				b.recordFailure()
			} else {
				b.recordSuccess()
			}
			if got := b.allow(); got != tt.wantAllowed {
				t.Fatalf("allow() after probe = %v, want %v", got, tt.wantAllowed)
			}
		})
	}
}

func TestBreakerSuccessResetsFailureCount(t *testing.T) {
	b := newBreaker(newFakeClock().now)

	b.recordFailure()
	b.recordFailure()
	b.recordSuccess()
	b.recordFailure()
	b.recordFailure()
	if !b.allow() {
		t.Fatal("interleaved success should reset the consecutive-failure count")
	}
}

func TestBreakerSetTracksPerCollector(t *testing.T) {
	set := newBreakerSet(newFakeClock().now)

	a := set.get("requirements")
	for i := 0; i < failureThreshold; i++ {
		a.recordFailure()
	}
	if set.get("requirements").allow() {
		t.Fatal("requirements breaker should be open")
	}
	if !set.get("ui").allow() {
		t.Fatal("ui breaker must be independent of the requirements breaker")
	}
	if set.get("requirements") != a {
		t.Fatal("get must return the same breaker per id")
	}
}
