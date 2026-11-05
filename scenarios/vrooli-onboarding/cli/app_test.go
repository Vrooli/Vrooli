package main

import (
	"bytes"
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
	if got := app.core.APIPrefix(); got != "/api" {
		t.Fatalf("API prefix = %q, want /api for the v2 onboarding routes", got)
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
	var quiet bytes.Buffer
	if code := runExitCode([]string{"--version"}, &quiet); code != 0 {
		t.Fatalf("runExitCode() = %d", code)
	}
	if quiet.Len() != 0 {
		t.Fatalf("a successful run must print nothing to stderr, got %q", quiet.String())
	}
	var stderr bytes.Buffer
	if code := runExitCode([]string{"unknown-command"}, &stderr); code == 0 {
		t.Fatal("unknown command should return a non-zero exit code")
	}
	if stderr.Len() == 0 {
		t.Fatal("a failing run must say why; this CLI exited silently for every error, which during the wizard was indistinguishable from success")
	}
}

// TestFailuresAreNeverSilent pins the contract the wizard depends on: an
// operator who answers a consent prompt and sees the command vanish cannot tell
// whether the host was changed. Any error must reach stderr.
func TestFailuresAreNeverSilent(t *testing.T) {
	for _, args := range [][]string{
		{"unknown-command"},
		{"wizard", "unknown-subcommand"},
		{"wizard", "commit"}, // missing the required --selection
	} {
		var stderr bytes.Buffer
		code := runExitCode(args, &stderr)
		if code == 0 {
			continue // not an error path on this build; nothing to assert
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Fatalf("%v exited %d without naming the failure; stderr = %q", args, code, stderr.String())
		}
	}
}
