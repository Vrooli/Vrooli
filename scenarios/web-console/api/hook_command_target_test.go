package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// String-matching a hook command proves only that the settings file says the
// right thing. These tests pin the check that proves it can actually run —
// the one whose absence let message capture fail on every turn for a day and a
// half while the status endpoint reported the hook as registered.

func TestSplitShellTokensHonorsQuotedPaths(t *testing.T) {
	got := splitShellTokens("bash '/opt/a b/hook.sh' --url 'http://localhost:1/x' --token 'tk'")
	want := []string{"bash", "/opt/a b/hook.sh", "--url", "http://localhost:1/x", "--token", "tk"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %#v, want %#v", got, want)
	}
}

func TestHookCommandTargetMissingDetectsDeletedScript(t *testing.T) {
	// This is the exact shape that broke: an interpreter that resolves fine on
	// PATH, running a script that no longer exists.
	missing := filepath.Join(t.TempDir(), "claude-stop-hook.sh")
	command := "bash '" + missing + "' --url 'http://localhost:1/x' --token 'tk'"

	got, reason := claudeHookCommandTargetMissing(command)
	if !got {
		t.Fatal("a command whose script has been deleted must be reported as unrunnable")
	}
	if reason == "" {
		t.Error("the reason must name what is missing")
	}
}

func TestHookCommandTargetMissingAcceptsAnExistingScript(t *testing.T) {
	script := filepath.Join(t.TempDir(), "hook.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if got, reason := claudeHookCommandTargetMissing("bash '" + script + "'"); got {
		t.Fatalf("an existing script must not be reported missing: %s", reason)
	}
}

func TestHookCommandTargetMissingDetectsExecutableNotOnPath(t *testing.T) {
	got, reason := claudeHookCommandTargetMissing("definitely-not-a-real-binary-xyz hooks dispatch --event 'Stop'")
	if !got {
		t.Fatal("a command whose executable is not installed must be reported as unrunnable")
	}
	if reason == "" {
		t.Error("the reason must name the missing executable")
	}
}

func TestHookCommandTargetMissingRejectsAnEmptyCommand(t *testing.T) {
	if got, _ := claudeHookCommandTargetMissing("   "); !got {
		t.Fatal("an empty command cannot run")
	}
}

func TestExecutableTargetMissingRejectsANonExecutableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-executable")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	got, reason := executableTargetMissing(path)
	if !got {
		t.Fatal("a file without an executable bit cannot be run as a hook")
	}
	if reason == "" {
		t.Error("the reason must explain what is wrong")
	}
}
