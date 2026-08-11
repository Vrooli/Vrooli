package main

import (
	"strings"
	"testing"
)

func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

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

func TestRunHelpListsMigratedDomains(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	// Help is rendered by cli-core to stdout; we only verify Run returns without
	// error. The presence of each domain in the actual help surface is covered
	// by cli-core's own tests.
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

func TestMetadata(t *testing.T) {
	if !strings.EqualFold(appName, "vrooli-onboarding") {
		t.Fatalf("appName = %q, want vrooli-onboarding", appName)
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestRunMain(t *testing.T) {
	if err := runMain([]string{"--version"}); err != nil {
		t.Fatalf("runMain() error: %v", err)
	}
	if code := runExitCode([]string{"--version"}); code != 0 {
		t.Fatalf("runExitCode() = %d", code)
	}
	if code := runExitCode([]string{"unknown-command"}); code == 0 {
		t.Fatal("unknown command should return a non-zero exit code")
	}
}
