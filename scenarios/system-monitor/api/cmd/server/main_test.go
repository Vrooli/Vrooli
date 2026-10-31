package main

import (
	"errors"
	"testing"

	"github.com/vrooli/api-core/preflight"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/config"
)

func TestRunStopsWhenPreflightHandledTheProcess(t *testing.T) {
	served := false
	code := run(
		func(preflight.Config) bool { return true },
		func() *config.Config { return &config.Config{} },
		func(*config.Config) error { served = true; return nil },
	)
	if code != 0 || served {
		t.Fatalf("run() = %d, served = %v; want 0 without starting the server", code, served)
	}
}

func TestRunServesLoadedConfigForSystemMonitor(t *testing.T) {
	loaded := &config.Config{}
	var gotScenario string
	var gotConfig *config.Config
	code := run(
		func(cfg preflight.Config) bool { gotScenario = cfg.ScenarioName; return false },
		func() *config.Config { return loaded },
		func(cfg *config.Config) error { gotConfig = cfg; return nil },
	)
	if code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
	if gotScenario != "system-monitor" {
		t.Fatalf("preflight scenario = %q, want system-monitor", gotScenario)
	}
	if gotConfig != loaded {
		t.Fatal("server did not receive the loaded config")
	}
}

func TestRunReportsServerFailureAsNonZeroExit(t *testing.T) {
	code := run(
		func(preflight.Config) bool { return false },
		func() *config.Config { return &config.Config{} },
		func(*config.Config) error { return errors.New("listen failed") },
	)
	if code != 1 {
		t.Fatalf("run() = %d, want 1 on server failure", code)
	}
}
