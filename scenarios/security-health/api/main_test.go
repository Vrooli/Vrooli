package main

import (
	"log"
	"testing"
	"time"
)

func TestLoadReconcileInterval(t *testing.T) {
	logger := log.New(log.Writer(), "", 0)
	cases := []struct {
		name string
		env  string
		set  bool
		want time.Duration
	}{
		{name: "unset uses default", set: false, want: defaultReconcileInterval},
		{name: "valid override", set: true, env: "10m", want: 10 * time.Minute},
		{name: "invalid falls back", set: true, env: "nonsense", want: defaultReconcileInterval},
		{name: "non-positive falls back", set: true, env: "0s", want: defaultReconcileInterval},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(EnvReconcileInterval, tc.env)
			} else {
				t.Setenv(EnvReconcileInterval, "")
			}
			if got := loadReconcileInterval(logger); got != tc.want {
				t.Fatalf("loadReconcileInterval() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestReconcileJitterBounded(t *testing.T) {
	interval := 5 * time.Minute
	max := interval / 4
	for i := 0; i < 1000; i++ {
		j := reconcileJitter(interval)
		if j < 0 || j >= max {
			t.Fatalf("jitter %s out of [0, %s)", j, max)
		}
	}
	// Degenerate interval yields no jitter rather than panicking.
	if j := reconcileJitter(0); j != 0 {
		t.Fatalf("reconcileJitter(0) = %s, want 0", j)
	}
}

func TestNextReconcileIntervalBacksOffAndResets(t *testing.T) {
	base, ceiling := 5*time.Minute, time.Hour
	interval, passes := nextReconcileInterval(base, ceiling, base, 0, false)
	if interval != 10*time.Minute || passes != 1 {
		t.Fatalf("first unchanged pass = %s/%d", interval, passes)
	}
	for i := 0; i < 10; i++ {
		interval, passes = nextReconcileInterval(base, ceiling, interval, passes, false)
	}
	if interval != ceiling {
		t.Fatalf("ceiling = %s, want %s", interval, ceiling)
	}
	interval, passes = nextReconcileInterval(base, ceiling, interval, passes, true)
	if interval != base || passes != 0 {
		t.Fatalf("changed pass = %s/%d, want %s/0", interval, passes, base)
	}
}
