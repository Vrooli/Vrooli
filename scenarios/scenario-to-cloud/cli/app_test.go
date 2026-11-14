package main

import (
	"strings"
	"testing"
)

// TestNewAppWiresEveryCommandDomain proves NewApp builds the standard scenario
// app with every domain group registered, so a missing domain registration
// fails here rather than at first use.
func TestNewAppWiresEveryCommandDomain(t *testing.T) {
	app := newTestApp(t)
	if app.core == nil || app.core.CLI == nil {
		t.Fatalf("NewApp must wire the cli-core scenario app")
	}
	output := captureStdout(t, func() {
		if err := app.Run([]string{"help"}); err != nil {
			t.Fatalf("help failed: %v", err)
		}
	})
	for _, domain := range []string{
		"manifest", "bundle", "deployment", "redeploy", "preflight", "vps",
		"inspect", "process", "edge", "secrets", "scenario", "task",
	} {
		if !strings.Contains(output, domain) {
			t.Errorf("help output does not list the %q domain", domain)
		}
	}
	if !strings.Contains(output, appName) {
		t.Errorf("help output does not name the app %q", appName)
	}
}

// TestAppRunRejectsEmptyAndUnknownInvocationsConsistently proves the wiring
// returns a typed error for an unknown command and does not panic on empty args.
func TestAppRunRejectsEmptyAndUnknownInvocationsConsistently(t *testing.T) {
	app := newTestApp(t)
	captureStdout(t, func() {
		if err := app.Run(nil); err != nil {
			t.Fatalf("empty args should print usage, got error: %v", err)
		}
	})
	captureStdout(t, func() {
		err := app.Run([]string{"no-such-domain-9f3a"})
		if err == nil {
			t.Fatalf("unknown command must return an error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "unknown") {
			t.Fatalf("error should say the command is unknown, got: %v", err)
		}
	})
}
