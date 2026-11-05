package policy

import (
	"testing"
	"time"
)

// TestCustomDurationConstants verifies custom duration bounds are internally consistent.
func TestCustomDurationConstants(t *testing.T) {
	if customDurationMin >= customDurationMax {
		t.Errorf("min (%s) should be less than max (%s)", customDurationMin, customDurationMax)
	}
	if customDurationMin != time.Minute {
		t.Errorf("customDurationMin should be 1m, got %s", customDurationMin)
	}
	if customDurationMax != 7*24*time.Hour {
		t.Errorf("customDurationMax should be 168h, got %s", customDurationMax)
	}
}

func TestPolicyValidationResolutionAndExpiry(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy Policy
		valid  bool
		ttl    time.Duration
	}{
		{name: "never", policy: Default(), valid: true},
		{name: "preset", policy: Policy{Mode: Preset, Duration: "8h"}, valid: true, ttl: 8 * time.Hour},
		{name: "custom", policy: Policy{Mode: Custom, Duration: "2h"}, valid: true, ttl: 2 * time.Hour},
		{name: "bad mode", policy: Policy{Mode: "unknown"}},
		{name: "bad preset", policy: Policy{Mode: Preset, Duration: "2h"}},
		{name: "short custom", policy: Policy{Mode: Custom, Duration: "30s"}, ttl: 30 * time.Second},
		{name: "long custom", policy: Policy{Mode: Custom, Duration: "200h"}, ttl: 200 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.policy)
			if tc.valid != (err == nil) {
				t.Fatalf("Validate(%+v) error=%v, want valid=%v", tc.policy, err, tc.valid)
			}
			if got := ResolveTTL(tc.policy); got != tc.ttl {
				t.Fatalf("ResolveTTL(%+v)=%s, want %s", tc.policy, got, tc.ttl)
			}
		})
	}
	if !IsExpired(time.Now().Add(-2*time.Hour), Policy{Mode: Preset, Duration: "1h"}) {
		t.Fatal("an old session should be expired")
	}
	if IsExpired(time.Now().Add(-2*time.Hour), Default()) {
		t.Fatal("never-expire policy should not expire")
	}
}
