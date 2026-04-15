package main

import (
	"testing"

	"tunnel-manager/cli/internal/flags"
)

// [REQ:CLI-RECOVERY-001] Recovery commands are registered
func TestRecoveryCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.subcommandGroups()
	wantCmds := map[string]bool{"state": false, "trigger": false, "events": false, "circuit-reset": false}

	for _, group := range groups {
		if group.Name != "recovery" {
			continue
		}
		for _, cmd := range group.Subcommands {
			if _, ok := wantCmds[cmd.Name]; ok {
				wantCmds[cmd.Name] = true
				if !group.NeedsAPI && !cmd.NeedsAPI {
					t.Errorf("recovery %s should require API", cmd.Name)
				}
			}
		}
	}

	for name, found := range wantCmds {
		if !found {
			t.Errorf("subcommand recovery %q not registered", name)
		}
	}
}

// [REQ:CLI-RECOVERY-002] Recovery API paths
func TestRecoveryAPIPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/recovery/state", "/api/v1/recovery/state"},
		{"/recovery/trigger", "/api/v1/recovery/trigger"},
		{"/recovery/events?limit=50", "/api/v1/recovery/events?limit=50"},
		{"/recovery/circuit/reset", "/api/v1/recovery/circuit/reset"},
	}

	for _, tc := range tests {
		got := app.core.APIPath(tc.input)
		if got != tc.want {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// [REQ:CLI-RECOVERY-003] Recovery commands support JSON output
func TestRecoveryJSONFlag(t *testing.T) {
	if !flags.HasJSONOutput([]string{"--json"}) {
		t.Error("recovery --json should be detected")
	}
	if !flags.HasJSONOutput([]string{"-j"}) {
		t.Error("recovery -j should be detected")
	}
}
