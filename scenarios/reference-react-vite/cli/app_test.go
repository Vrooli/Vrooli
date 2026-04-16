// Package main contains CLI tests for reference-react-vite.
package main

import (
	"strings"
	"testing"
)

func TestAppCreation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	if app == nil {
		t.Fatal("expected app to be non-nil")
	}
	if app.core == nil {
		t.Fatal("expected app.core to be non-nil")
	}
}

func TestScenarioPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty_path_returns_empty", input: "", expected: ""},
		{name: "path_with_leading_slash", input: "/health", expected: "/api/v1/health"},
		{name: "path_without_leading_slash", input: "health", expected: "/api/v1/health"},
		{name: "whitespace_only_returns_empty", input: "   ", expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := app.core.APIPath(tc.input)
			if result != tc.expected {
				t.Errorf("APIPath(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestBuiltInAndDomainCommandsAreRegistered(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if err := app.Run([]string{"configure", "api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	for _, args := range [][]string{
		{"task", "help"},
		{"project", "help"},
		{"note", "help"},
	} {
		if err := app.Run(args); err != nil {
			t.Fatalf("expected %v help to succeed: %v", args, err)
		}
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
	if appName != "reference-react-vite" {
		t.Errorf("expected appName 'reference-react-vite', got '%s'", appName)
	}
	if appVersion == "" {
		t.Error("expected appVersion to be non-empty")
	}
}
