package main

import (
	"strings"
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

func TestAppPaths(t *testing.T) {
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
			got := app.core.APIPath(tt.input)
			if got != tt.want {
				t.Errorf("APIPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAppHasBuiltInCommands(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() returned error: %v", err)
	}

	if app.core.CLI == nil {
		t.Fatal("expected CLI to be initialized")
	}
	if err := app.Run([]string{"configure", "api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	err = app.Run([]string{"status"})
	if err == nil {
		t.Fatal("expected status to fail without a reachable API")
	}
	if !strings.Contains(err.Error(), "API request failed") &&
		!strings.Contains(err.Error(), "api error (404)") &&
		!strings.Contains(err.Error(), "connection refused") &&
		!strings.Contains(err.Error(), "lookup") {
		t.Fatalf("expected built-in status command to execute, got %v", err)
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
