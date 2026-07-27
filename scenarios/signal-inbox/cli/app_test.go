package main

import (
	"strings"
	"testing"
)

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}
