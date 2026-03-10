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
		{
			name:     "domains path",
			input:    "/domains",
			wantPath: "/api/v1/domains",
		},
		{
			name:     "stats timeline path",
			input:    "/stats/timeline",
			wantPath: "/api/v1/stats/timeline",
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
// [REQ:LD-FUNC-001] CLI commands for all API endpoints
func TestRegisterCommands(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	groups := app.registerCommands()
	if len(groups) == 0 {
		t.Error("registerCommands() returned no command groups")
	}

	// Verify we have all expected command groups
	groupNames := make(map[string]bool)
	commandNames := make(map[string]bool)
	for _, g := range groups {
		groupNames[g.Title] = true
		for _, cmd := range g.Commands {
			commandNames[cmd.Name] = true
		}
	}

	// Check required groups
	requiredGroups := []string{"Health", "Events", "Domains", "Statistics", "Configuration"}
	for _, name := range requiredGroups {
		if !groupNames[name] {
			t.Errorf("missing %s command group", name)
		}
	}

	// Check required commands for API parity
	requiredCommands := []string{
		"status",
		"event create",
		"event list",
		"event get",
		"domain register",
		"domain list",
		"domain get",
		"domain update",
		"domain health",
		"stats timeline",
		"stats summary",
	}
	for _, name := range requiredCommands {
		if !commandNames[name] {
			t.Errorf("missing command: %s", name)
		}
	}
}

// TestCommandsNeedAPI verifies API-requiring commands have NeedsAPI set
func TestCommandsNeedAPI(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	groups := app.registerCommands()

	// Commands that should require API access
	apiCommands := map[string]bool{
		"status":          true,
		"event create":    true,
		"event list":      true,
		"event get":       true,
		"domain register": true,
		"domain list":     true,
		"domain get":      true,
		"domain update":   true,
		"domain health":   true,
		"stats timeline":  true,
		"stats summary":   true,
	}

	for _, g := range groups {
		for _, cmd := range g.Commands {
			if expected, exists := apiCommands[cmd.Name]; exists {
				if cmd.NeedsAPI != expected {
					t.Errorf("command %q: NeedsAPI = %v, want %v", cmd.Name, cmd.NeedsAPI, expected)
				}
			}
		}
	}
}

// TestCommandDescriptions verifies all commands have descriptions
func TestCommandDescriptions(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	groups := app.registerCommands()

	for _, g := range groups {
		for _, cmd := range g.Commands {
			if cmd.Description == "" {
				t.Errorf("command %q has empty description", cmd.Name)
			}
		}
	}
}
