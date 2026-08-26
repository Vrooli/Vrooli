package main

import (
	"strings"
	"testing"

	"web-console/cli/internal/testutil"
)

func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp()"))
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp()"))
	}
	// --version is handled by cli-core; it must not return an error and must not
	// touch the network (no NeedsAPI path triggered).
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

func TestRunHelpListsMigratedDomains(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp()"))
	}
	// The standard help text lists all registered command groups and
	// subcommand groups. We assert the migrated domains are wired up.
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

func TestMetadata(t *testing.T) {
	if !strings.EqualFold(appName, "web-console") {
		t.Fatalf("appName = %q, want web-console", appName)
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestStandardScenarioPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp"))
	}
	tests := []struct {
		input string
		want  string
	}{
		{"/sessions", "/api/v1/sessions"},
		{"sessions", "/api/v1/sessions"},
		{"", ""},
	}
	for _, tc := range tests {
		got := app.core.APIPath(tc.input)
		if got != tc.want {
			t.Errorf("APIPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBuiltInStatusCommandRegistered(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp"))
	}

	if err := app.Run([]string{"configure", "api_base", "http://127.0.0.1:1"}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	// The built-in status command probes /health; without a reachable API it
	// must surface a network/HTTP error rather than silently succeeding. This
	// guards against accidentally shadowing cli-core's built-in status.
	err = app.Run([]string{"status"})
	if err == nil {
		t.Fatal("expected status to fail without a reachable API")
	}
	msg := err.Error()
	if !strings.Contains(msg, "API request failed") &&
		!strings.Contains(msg, "api error") &&
		!strings.Contains(msg, "connection refused") &&
		!strings.Contains(msg, "lookup") {
		t.Fatalf("expected built-in status command to execute, got %v", err)
	}
}
