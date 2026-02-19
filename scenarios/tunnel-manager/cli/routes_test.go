package main

import (
	"testing"
)

// [REQ:CLI-ROUTES-001] Routes command is registered and callable
func TestRoutesCommandRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	// Verify the routes command is registered in the command groups
	groups := app.registerCommands()
	found := false
	for _, g := range groups {
		for _, cmd := range g.Commands {
			if cmd.Name == "routes" {
				found = true
				if !cmd.NeedsAPI {
					t.Error("routes command should require API")
				}
			}
		}
	}
	if !found {
		t.Error("routes command not registered")
	}
}

// [REQ:CLI-ROUTES-001] Routes command API path
func TestRoutesAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	got := app.apiPath("/routes")
	want := "/api/v1/routes"
	if got != want {
		t.Errorf("apiPath(/routes) = %q, want %q", got, want)
	}
}

// [REQ:CLI-ROUTES-002] Routes command supports JSON output
func TestRoutesJSONFlag(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if !app.useJSON([]string{"--json"}) {
		t.Error("routes --json should be detected")
	}
	if app.useJSON([]string{}) {
		t.Error("routes without --json should not detect JSON mode")
	}
}
