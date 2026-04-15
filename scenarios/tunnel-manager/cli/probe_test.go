package main

import (
	"testing"

	"tunnel-manager/cli/internal/flags"
)

// [REQ:CLI-PROBE-001] Probe command is registered and callable
func TestProbeCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.subcommandGroups()
	found := false
	for _, group := range groups {
		if group.Name != "probe" {
			continue
		}
		found = hasSubcommand(group, "run")
		if !group.NeedsAPI {
			t.Error("probe group should require API")
		}
	}
	if !found {
		t.Error("probe run command not registered")
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
	if !flags.HasJSONOutput([]string{"--json"}) {
		t.Error("probe --json should be detected")
	}
	if !flags.HasJSONOutput([]string{"-j"}) {
		t.Error("probe -j should be detected")
	}
}
