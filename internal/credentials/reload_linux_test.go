//go:build linux

package credentials

import (
	"testing"
	"time"
)

// systemd renders durations in its own spelling, and a wrong parse silently
// reverts the restart budget to the fallback — which is how the first live run
// reported a successful repair as a failure.
func TestParseSystemdDuration(t *testing.T) {
	cases := []struct {
		value string
		want  time.Duration
		ok    bool
	}{
		{"1min 30s", 90 * time.Second, true},
		{"90s", 90 * time.Second, true},
		{"2min", 2 * time.Minute, true},
		{"5min 10s", 5*time.Minute + 10*time.Second, true},
		{"infinity", 0, false},
		{"", 0, false},
		{"not-a-duration", 0, false},
	}
	for _, testCase := range cases {
		got, ok := parseSystemdDuration(testCase.value)
		if ok != testCase.ok || got != testCase.want {
			t.Fatalf("parseSystemdDuration(%q) = (%s, %t), want (%s, %t)", testCase.value, got, ok, testCase.want, testCase.ok)
		}
	}
}

// The budget must exceed the unit's stop timeout, or the command gives up
// before systemd has finished escalating to SIGKILL.
func TestReloadBudgetExceedsUnitStopTimeout(t *testing.T) {
	stop, ok := parseSystemdDuration("1min 30s")
	if !ok {
		t.Fatal("fixture failed to parse")
	}
	if stop+reloadMargin <= stop {
		t.Fatal("reloadMargin must be positive")
	}
	if reloadFallback <= 90*time.Second {
		t.Fatalf("reloadFallback = %s, want more than systemd's 90s default stop timeout", reloadFallback)
	}
}
