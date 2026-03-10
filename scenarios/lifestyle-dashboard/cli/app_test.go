package main

import (
	"testing"
)

// TestNewApp verifies the CLI app can be constructed without errors
// [REQ:LD-FUNC-001] CLI support for scenario management
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

// TestAppConstants verifies CLI constants are properly set
func TestAppConstants(t *testing.T) {
	if appName != "lifestyle-dashboard" {
		t.Errorf("appName = %q, want %q", appName, "lifestyle-dashboard")
	}
	if appVersion == "" {
		t.Error("appVersion is empty")
	}
}

// TestAPIPath verifies the API path construction helper
func TestAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		wantPath string
	}{
		{
			name:     "empty path",
			input:    "",
			wantPath: "",
		},
		{
			name:     "health path without leading slash",
			input:    "health",
			wantPath: "/api/v1/health",
		},
		{
			name:     "health path with leading slash",
			input:    "/health",
			wantPath: "/api/v1/health",
		},
		{
			name:     "events path",
			input:    "/events",
			wantPath: "/api/v1/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.apiPath(tt.input)
			if got != tt.wantPath {
				t.Errorf("apiPath(%q) = %q, want %q", tt.input, got, tt.wantPath)
			}
		})
	}
}

// TestRegisterCommands verifies the command registration
func TestRegisterCommands(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	groups := app.registerCommands()
	if len(groups) == 0 {
		t.Error("registerCommands() returned no command groups")
	}

	// Verify we have Health and Configuration groups
	groupNames := make(map[string]bool)
	for _, g := range groups {
		groupNames[g.Title] = true
	}

	if !groupNames["Health"] {
		t.Error("missing Health command group")
	}
	if !groupNames["Configuration"] {
		t.Error("missing Configuration command group")
	}
}
