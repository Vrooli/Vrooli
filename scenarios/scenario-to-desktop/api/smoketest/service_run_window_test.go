package smoketest

import (
	"testing"
	"time"
)

func TestMeasuredDemoHoldCoversMeasurementWindows(t *testing.T) {
	holdMs, timeout := demoHoldTiming(true)
	hold := time.Duration(holdMs) * time.Millisecond
	if hold < 5*time.Minute {
		t.Fatalf("measured demo hold = %v, want at least five minutes for readiness, idle CPU, journey, and steady PSS", hold)
	}
	if timeout <= hold {
		t.Fatalf("measured executor timeout = %v, want greater than app hold %v", timeout, hold)
	}
}

func TestUnmeasuredDemoTimingKeepsExistingDefaults(t *testing.T) {
	holdMs, timeout := demoHoldTiming(false)
	if holdMs != DefaultDemoHoldMs || timeout != 90*time.Second {
		t.Fatalf("unmeasured demo timing = (%d ms, %v), want (%d ms, 90s)", holdMs, timeout, DefaultDemoHoldMs)
	}
}
