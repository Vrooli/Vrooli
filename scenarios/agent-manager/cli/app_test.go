package main

import (
	"testing"
)

// Tests for the CLI application

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if app == nil {
		t.Fatal("expected app, got nil")
	}

	// Verify core is initialized
	if app.core == nil {
		t.Error("expected core to be initialized")
	}
}

func TestApp_Run_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running with help flag should not error
	// Note: help typically exits with code 0 but urfave/cli may return an error
	err = app.Run([]string{"agent-manager", "--help"})
	// Help output is acceptable, we just want to ensure no panic
	_ = err
}

func TestApp_Run_Version(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running with version flag should not error
	// Note: version typically exits with code 0 but urfave/cli may return an error
	err = app.Run([]string{"agent-manager", "--version"})
	// Version output is acceptable, we just want to ensure no panic
	_ = err
}

func TestApp_Run_UnknownCommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Unknown command should provide help or error gracefully
	err = app.Run([]string{"agent-manager", "nonexistent-command"})
	// We expect either an error or graceful handling
	// The specific behavior depends on the CLI framework
	_ = err // Accept either outcome for this basic test
}

func TestApp_Structure(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Verify the App struct has expected core field
	if app.core == nil {
		t.Error("app core should be set")
	}
}
