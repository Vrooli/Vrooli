package main

import (
	"testing"

	"tunnel-manager/cli/internal/flags"

	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:CLI-ROUTES-001] Routes command is registered and callable
func TestRoutesCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.subcommandGroups()
	found := false
	for _, group := range groups {
		if group.Name != "route" {
			continue
		}
		found = hasSubcommand(group, "list")
		if !group.NeedsAPI {
			t.Error("route group should require API")
		}
	}
	if !found {
		t.Error("route list command not registered")
	}
}

// [REQ:CLI-ROUTES-001] Routes command API path
func TestRoutesAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.core.APIPath("/routes")
	want := "/api/v1/routes"
	if got != want {
		t.Errorf("apiPath(/routes) = %q, want %q", got, want)
	}
}

// [REQ:CLI-ROUTES-002] Routes command supports JSON output
func TestRoutesJSONFlag(t *testing.T) {
	if !flags.HasJSONOutput([]string{"--json"}) {
		t.Error("route list --json should be detected")
	}
	if flags.HasJSONOutput([]string{}) {
		t.Error("route list without --json should not detect JSON mode")
	}
}

func hasSubcommand(group cliapp.SubcommandGroup, name string) bool {
	for _, cmd := range group.Subcommands {
		if cmd.Name == name {
			return true
		}
	}
	return false
}
