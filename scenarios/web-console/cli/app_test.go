package main

import (
	"strings"
	"testing"
)

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}
	if app == nil {
		t.Fatal("expected non-nil app")
	}
	if app.core == nil {
		t.Fatal("expected non-nil core")
	}
}

func TestStandardScenarioPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/health", "/api/v1/health"},
		{"health", "/api/v1/health"},
		{"", ""},
	}

	for _, tc := range tests {
		got := app.core.APIPath(tc.input)
		if got != tc.want {
			t.Errorf("APIPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBuiltInCommandsAreRegistered(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
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
