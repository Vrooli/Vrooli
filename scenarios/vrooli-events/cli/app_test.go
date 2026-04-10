package main

import (
	"testing"
)

// [REQ:REQ-CLI-001] CLI initializes with cli-core ScenarioApp
func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp() returned nil app")
	}
	if app.core == nil {
		t.Fatal("NewApp() returned app with nil core")
	}
}

// [REQ:REQ-CLI-001] API path resolution prepends /api/v1
func TestApiPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple path", "/events", "/api/v1/events"},
		{"no leading slash", "events", "/api/v1/events"},
		{"empty", "", ""},
		{"with whitespace", "  /health  ", "/api/v1/health"},
		{"health path", "/health", "/api/v1/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.apiPath(tt.input)
			if got != tt.expected {
				t.Errorf("apiPath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// [REQ:REQ-CLI-001] Command registration includes all event commands
func TestCommandRegistration(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()

	expectedCommands := map[string]bool{
		"ingest":    false,
		"query":     false,
		"subscribe": false,
		"stats":     false,
		"status":    false,
		"configure": false,
	}

	for _, g := range groups {
		for _, cmd := range g.Commands {
			if _, exists := expectedCommands[cmd.Name]; exists {
				expectedCommands[cmd.Name] = true
			}
		}
	}

	for name, found := range expectedCommands {
		if !found {
			t.Errorf("expected command %q to be registered", name)
		}
	}
}
