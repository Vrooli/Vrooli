package main

import (
	"testing"
)

// [REQ:CLI-HEALTH-DETAILED-001] Health detailed command is registered
func TestHealthDetailedCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	found := false
	for _, g := range groups {
		for _, cmd := range g.Commands {
			if cmd.Name == "health detailed" {
				found = true
				if !cmd.NeedsAPI {
					t.Error("health detailed command should require API")
				}
			}
		}
	}
	if !found {
		t.Error("health detailed command not registered")
	}
}

// [REQ:CLI-HEALTH-DETAILED-002] Health detailed API path
func TestHealthDetailedAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.core.APIPath("/health/detailed")
	want := "/api/v1/health/detailed"
	if got != want {
		t.Errorf("apiPath(/health/detailed) = %q, want %q", got, want)
	}
}

// [REQ:CLI-HEALTH-DETAILED-003] Probes history command is registered
func TestProbesHistoryCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	found := false
	for _, g := range groups {
		for _, cmd := range g.Commands {
			if cmd.Name == "probes history" {
				found = true
				if !cmd.NeedsAPI {
					t.Error("probes history command should require API")
				}
			}
		}
	}
	if !found {
		t.Error("probes history command not registered")
	}
}

// [REQ:CLI-HEALTH-DETAILED-004] Probes history API path
func TestProbesHistoryAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.core.APIPath("/probes/history?limit=100")
	want := "/api/v1/probes/history?limit=100"
	if got != want {
		t.Errorf("apiPath(/probes/history?limit=100) = %q, want %q", got, want)
	}
}
