package main

import (
	"testing"

	"tunnel-manager/cli/internal/flags"
)

// [REQ:CLI-AUDIT-001] Audit command is registered and callable
func TestAuditCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.subcommandGroups()
	found := false
	for _, group := range groups {
		if group.Name != "audit" {
			continue
		}
		found = hasSubcommand(group, "ports")
		if !group.NeedsAPI {
			t.Error("audit group should require API")
		}
	}
	if !found {
		t.Error("audit ports command not registered")
	}
}

// [REQ:CLI-AUDIT-001] Audit command API path
func TestAuditAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.core.APIPath("/audit/ports")
	want := "/api/v1/audit/ports"
	if got != want {
		t.Errorf("apiPath(/audit/ports) = %q, want %q", got, want)
	}
}

// [REQ:CLI-AUDIT-002] Audit command supports JSON output
func TestAuditJSONFlag(t *testing.T) {
	if !flags.HasJSONOutput([]string{"--json"}) {
		t.Error("audit --json should be detected")
	}
}

// [REQ:CLI-AUDIT-001] The migrated command surface is registered
func TestAllCommandGroupsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if len(app.commandGroups()) != 1 {
		t.Fatalf("expected 1 flat command group, got %d", len(app.commandGroups()))
	}

	wantSubgroups := map[string]bool{
		"health":   false,
		"route":    false,
		"probe":    false,
		"audit":    false,
		"metrics":  false,
		"recovery": false,
	}
	for _, group := range app.subcommandGroups() {
		if _, ok := wantSubgroups[group.Name]; ok {
			wantSubgroups[group.Name] = true
		}
	}

	for name, found := range wantSubgroups {
		if !found {
			t.Errorf("subcommand group %q not registered", name)
		}
	}
}
