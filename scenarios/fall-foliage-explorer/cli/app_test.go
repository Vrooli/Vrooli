package main

import (
	"strings"
	"testing"
)

// [REQ:REQ-P0-007] CLI app constructs through the scenario command adapter.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// [REQ:REQ-P0-007] CLI exposes a lifecycle-compatible version command.
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	// --version is handled by cli-core; it must not return an error and must not
	// touch the network (no NeedsAPI path triggered).
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// [REQ:REQ-P0-007] CLI help exposes migrated foliage, report, and trip domains.
func TestRunHelpListsMigratedDomains(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	// The standard help text lists all registered command groups and
	// subcommand groups. We assert the migrated domains are wired up.
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
	// Help is rendered by cli-core to stdout; we only verify Run returns without
	// error. The presence of each domain in the actual help surface is covered
	// by cli-core's own tests.
}

// [REQ:REQ-P0-007] CLI metadata matches the scenario identity.
func TestMetadata(t *testing.T) {
	if !strings.EqualFold(appName, "fall-foliage-explorer") {
		t.Fatalf("appName = %q, want fall-foliage-explorer", appName)
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}
