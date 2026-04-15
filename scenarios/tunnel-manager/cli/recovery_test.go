package main

import (
	"testing"
)

// [REQ:CLI-RECOVERY-001] Recovery commands are registered
func TestRecoveryCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	wantCmds := map[string]bool{
		"recovery state":         false,
		"recovery trigger":       false,
		"recovery events":        false,
		"recovery circuit-reset": false,
	}

	for _, g := range groups {
		for _, cmd := range g.Commands {
			if _, ok := wantCmds[cmd.Name]; ok {
				wantCmds[cmd.Name] = true
				if !cmd.NeedsAPI {
					t.Errorf("%s command should require API", cmd.Name)
				}
			}
		}
	}

	for name, found := range wantCmds {
		if !found {
			t.Errorf("command %q not registered", name)
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
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if !app.useJSON([]string{"--json"}) {
		t.Error("recovery --json should be detected")
	}
	if !app.useJSON([]string{"-j"}) {
		t.Error("recovery -j should be detected")
	}
}
