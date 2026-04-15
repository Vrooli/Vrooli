package main

import (
	"testing"
)

// [REQ:CLI-PROBE-001] Probe command is registered and callable
func TestProbeCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	found := false
	for _, g := range groups {
		for _, cmd := range g.Commands {
			if cmd.Name == "probe" {
				found = true
				if !cmd.NeedsAPI {
					t.Error("probe command should require API")
				}
			}
		}
	}
	if !found {
		t.Error("probe command not registered")
	}
}

// [REQ:CLI-PROBE-001] Probe command API path uses POST /probes
func TestProbeAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.core.APIPath("/probes")
	want := "/api/v1/probes"
	if got != want {
		t.Errorf("apiPath(/probes) = %q, want %q", got, want)
	}
}

// [REQ:CLI-PROBE-002] Probe command supports JSON output
func TestProbeJSONFlag(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if !app.useJSON([]string{"--json"}) {
		t.Error("probe --json should be detected")
	}
	if !app.useJSON([]string{"-j"}) {
		t.Error("probe -j should be detected")
	}
}
