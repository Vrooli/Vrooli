package main

import (
	"testing"
)

// [REQ:REQ-P0-003] CLI basic smoke tests

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

func TestAppAPIPath(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "absolute path", input: "/health", want: "/api/v1/health"},
		{name: "relative path", input: "health", want: "/api/v1/health"},
		{name: "empty input", input: "", want: ""},
		{name: "path with subpath", input: "/resources/health", want: "/api/v1/resources/health"},
		{name: "whitespace input", input: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.apiPath(tt.input)
			if got != tt.want {
				t.Errorf("apiPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAppHasCommands(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	// Verify the CLI has registered commands
	if app.core.CLI == nil {
		t.Fatal("expected CLI to be initialized")
	}
}

func TestAppConstants(t *testing.T) {
	if appName != "vrooli-onboarding" {
		t.Errorf("appName = %q, want %q", appName, "vrooli-onboarding")
	}
	if appVersion == "" {
		t.Error("appVersion should not be empty")
	}
}
