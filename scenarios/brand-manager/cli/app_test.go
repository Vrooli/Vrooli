package main

import (
	"strings"
	"testing"
)

// [REQ:BM-REQ-CLI-CRUD] Verify CLI app initialization and command registration.

// mustNewApp creates a new App or fails the test.
func mustNewApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	return app
}

func TestNewApp(t *testing.T) {
	app := mustNewApp(t)
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}
	if app.core == nil {
		t.Fatal("app.core is nil")
	}
}

func TestAPIPath_Prefix(t *testing.T) {
	app := mustNewApp(t)

	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{"brands path", "/brands", "/api/v1/brands"},
		{"health path", "/health", "/api/v1/health"},
		{"empty path", "", ""},
		{"no leading slash", "brands", "/api/v1/brands"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := app.core.APIPath(tt.input)
			if tt.contains == "" && result != "" {
				t.Errorf("expected empty, got %q", result)
			} else if tt.contains != "" && result != tt.contains {
				t.Errorf("expected %q, got %q", tt.contains, result)
			}
		})
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	app := mustNewApp(t)

	err := app.Run([]string{"nonexistent-command"})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

// [REQ:BM-REQ-CLI-CRUD] [REQ:BM-REQ-CLI-DISCOVER] [REQ:BM-REQ-CLI-APPLY]
// [REQ:BM-REQ-CLI-STATUS] [REQ:BM-REQ-CLI-GEN]
// Verify commands requiring arguments fail with a meaningful error (not "unknown command").
func TestCommands_NoArgs(t *testing.T) {
	app := mustNewApp(t)

	tests := []struct {
		name string
		cmd  string
	}{
		{"discover", "discover"},
		{"apply", "apply"},
		{"scan", "scan"},
		{"get", "get"},
		{"delete", "delete"},
		{"update", "update"},
		{"versions", "versions"},
		{"scenario-status", "scenario-status"},
		{"assign", "assign"},
		{"unassign", "unassign"},
		{"generate", "generate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run([]string{tt.cmd})
			if err == nil {
				t.Fatalf("expected error when %s called without arguments", tt.cmd)
			}
			if strings.Contains(err.Error(), "unknown command") {
				t.Errorf("command %q should be registered", tt.cmd)
			}
		})
	}
}

// [REQ:BM-REQ-CLI-CRUD] Verify list command is registered and accepts no-arg invocation.
func TestList_ReturnsError(t *testing.T) {
	app := mustNewApp(t)

	// List without API should error but not with "unknown command"
	err := app.Run([]string{"list"})
	if err == nil {
		t.Log("list succeeded (API might be available)")
	}
	if err != nil && strings.Contains(err.Error(), "unknown command") {
		t.Error("list command should be registered")
	}
}

// [REQ:BM-REQ-CLI-DISCOVER] [REQ:BM-REQ-CLI-APPLY] [REQ:BM-REQ-CLI-STATUS] [REQ:BM-REQ-CLI-GEN]
// Verify commands with arguments hit API (not "unknown command").
func TestCommands_WithArgs(t *testing.T) {
	app := mustNewApp(t)

	tests := []struct {
		name string
		args []string
	}{
		{"discover with scenario", []string{"discover", "nonexistent-scenario-xyz"}},
		{"scan with scenario", []string{"scan", "test-scenario"}},
		{"apply with args", []string{"apply", "brand-id", "scenario-name"}},
		{"generate with brand", []string{"generate", "brand-id-123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := app.Run(tt.args)
			if err == nil {
				t.Log("command succeeded (API might be running)")
				return
			}
			if strings.Contains(err.Error(), "unknown command") {
				t.Errorf("command %q should be registered", tt.args[0])
			}
		})
	}
}

// [REQ:BM-REQ-CLI-CRUD] [REQ:BM-REQ-CLI-GEN] Verify all commands are registered.
func TestCommandRegistration(t *testing.T) {
	app := mustNewApp(t)

	// Each command should produce an error (not "unknown command" which Run returns)
	commands := []string{
		"create", "list", "get", "update", "delete", "versions",
		"assign", "unassign", "scenario-status",
		"generate", "discover", "apply", "scan",
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			err := app.Run([]string{cmd})
			// Acceptable: fails because no API or missing args, but NOT "unknown command"
			if err != nil && strings.Contains(err.Error(), "unknown command") {
				t.Errorf("command %q is not registered", cmd)
			}
		})
	}
}
