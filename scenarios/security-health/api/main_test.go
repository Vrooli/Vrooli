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
