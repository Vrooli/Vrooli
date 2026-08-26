package main

import (
	"runtime"
	"testing"
)

func TestBackendRegistry_StandardAvailableWithPlatformPTY(t *testing.T) {
	available, reason := probeStandard()
	if !available || reason != "" {
		t.Fatalf("%s standard backend = (%v, %q), want available", runtime.GOOS, available, reason)
	}
}
