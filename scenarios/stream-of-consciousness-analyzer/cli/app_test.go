package main

import (
	"strings"
	"testing"
)

func mustNewApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	return app
}

func TestNewApp(t *testing.T) {
	app := mustNewApp(t)
	if app == nil || app.core == nil {
		t.Fatal("expected initialized app core")
	}
}

func TestAPIPath(t *testing.T) {
	app := mustNewApp(t)

	cases := []struct {
		input    string
		expected string
	}{
		{"/schemes", "/api/v1/schemes"},
		{"thoughts", "/api/v1/thoughts"},
		{"/providers", "/api/v1/providers"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := app.core.APIPath(tc.input); got != tc.expected {
			t.Errorf("APIPath(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRunRegistersCommands(t *testing.T) {
	app := mustNewApp(t)

	commands := [][]string{
		{"scheme", "list"},
		{"scheme", "get"},
		{"scheme", "create"},
		{"scheme", "update"},
		{"scheme", "delete"},
		{"scheme", "export"},
		{"thought", "list"},
		{"thought", "get"},
		{"thought", "create"},
		{"thought", "update"},
		{"thought", "delete"},
		{"edge", "list"},
		{"edge", "create"},
		{"edge", "delete"},
		{"info", "list"},
		{"info", "create"},
		{"info", "update"},
		{"info", "delete"},
		{"provider", "list"},
		{"suggestion", "generate"},
		{"status"},
		{"configure"},
	}

	for _, args := range commands {
		t.Run(strings.Join(args, "-"), func(t *testing.T) {
			err := app.Run(args)
			if err != nil && strings.Contains(err.Error(), "unknown command") {
				t.Fatalf("expected command %q to be registered", strings.Join(args, " "))
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	app := mustNewApp(t)

	tests := []struct {
		name string
		args []string
	}{
		{"scheme get", []string{"scheme", "get"}},
		{"scheme update", []string{"scheme", "update", "scheme-1"}},
		{"thought create", []string{"thought", "create"}},
		{"thought update", []string{"thought", "update", "thought-1"}},
		{"edge create", []string{"edge", "create", "thought-1"}},
		{"info create", []string{"info", "create"}},
		{"info update", []string{"info", "update", "info-1", "--scheme", "scheme-1"}},
		{"suggestion generate", []string{"suggestion", "generate"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := app.Run(tc.args); err == nil {
				t.Fatalf("expected validation error for %q", strings.Join(tc.args, " "))
			}
		})
	}
}

func TestAppConstants(t *testing.T) {
	if appName != "stream-of-consciousness-analyzer" {
		t.Fatalf("unexpected appName %q", appName)
	}
	if appVersion == "" {
		t.Fatal("appVersion must be set")
	}
}
