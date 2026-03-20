package main

import (
	"testing"
)

// [REQ:CLI-AUDIT-001] Audit command is registered and callable
func TestAuditCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	found := false
	for _, g := range groups {
		for _, cmd := range g.Commands {
			if cmd.Name == "audit" {
				found = true
				if !cmd.NeedsAPI {
					t.Error("audit command should require API")
				}
			}
		}
	}
	if !found {
		t.Error("audit command not registered")
	}
}

// [REQ:CLI-AUDIT-001] Audit command API path
func TestAuditAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.apiPath("/audit/ports")
	want := "/api/v1/audit/ports"
	if got != want {
		t.Errorf("apiPath(/audit/ports) = %q, want %q", got, want)
	}
}

// [REQ:CLI-AUDIT-002] Audit command supports JSON output
func TestAuditJSONFlag(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if !app.useJSON([]string{"--json"}) {
		t.Error("audit --json should be detected")
	}
}

// [REQ:CLI-AUDIT-001] All five command groups are registered
func TestAllCommandGroupsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	wantGroups := map[string]bool{
		"Health":        false,
		"Routes":        false,
		"Probes":        false,
		"Audit":         false,
		"Metrics":       false,
		"Recovery":      false,
		"Configuration": false,
	}

	for _, g := range groups {
		if _, ok := wantGroups[g.Title]; ok {
			wantGroups[g.Title] = true
		}
	}

	for name, found := range wantGroups {
		if !found {
			t.Errorf("command group %q not registered", name)
		}
	}
}
