package tuning

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestDurationUsesCompiledFallbackWhenOverrideIsAbsent(t *testing.T) {
	resetDurationCacheForTest()
	t.Setenv("VROOLI_TUNING_FIXTURE_TIMEOUT", "")
	if got := Duration("FixtureTimeout", 3*time.Second); got != 3*time.Second {
		t.Fatalf("Duration = %v, want 3s", got)
	}
	if got := Duration("FixtureTimeout", 4*time.Second); got != 4*time.Second {
		t.Fatalf("Duration with a runtime fallback = %v, want 4s", got)
	}
}

func TestDurationUsesValidOverrideOncePerProcess(t *testing.T) {
	resetDurationCacheForTest()
	t.Setenv("VROOLI_TUNING_FIXTURE_TIMEOUT", "9s")
	if got := Duration("FixtureTimeout", 3*time.Second); got != 9*time.Second {
		t.Fatalf("Duration = %v, want 9s", got)
	}
	t.Setenv("VROOLI_TUNING_FIXTURE_TIMEOUT", "12s")
	if got := Duration("FixtureTimeout", 3*time.Second); got != 9*time.Second {
		t.Fatalf("cached Duration = %v, want 9s", got)
	}
}

func TestDurationWarnsAndFallsBackForMalformedOverride(t *testing.T) {
	resetDurationCacheForTest()
	var output bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })
	t.Setenv("VROOLI_TUNING_FIXTURE_TIMEOUT", "eventually")

	if got := Duration("FixtureTimeout", 3*time.Second); got != 3*time.Second {
		t.Fatalf("Duration = %v, want fallback 3s", got)
	}
	if got := Duration("FixtureTimeout", 4*time.Second); got != 4*time.Second {
		t.Fatalf("cached malformed Duration = %v, want fallback 4s", got)
	}
	if log := output.String(); !strings.Contains(log, "VROOLI_TUNING_FIXTURE_TIMEOUT") || !strings.Contains(log, "eventually") {
		t.Fatalf("warning = %q", log)
	} else if strings.Count(log, "Ignoring malformed timing override") != 1 {
		t.Fatalf("warning count = %d, want 1; log = %q", strings.Count(log, "Ignoring malformed timing override"), log)
	}
}

func TestUpperSnakePreservesInitialisms(t *testing.T) {
	for input, want := range map[string]string{
		"ScenarioHeartbeatTTL": "SCENARIO_HEARTBEAT_TTL",
		"HostGPUInventoryTTL":  "HOST_GPU_INVENTORY_TTL",
		"health-check timeout": "HEALTH_CHECK_TIMEOUT",
	} {
		if got := upperSnake(input); got != want {
			t.Errorf("upperSnake(%q) = %q, want %q", input, got, want)
		}
	}
}
