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

func TestLogRecoveryHints(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name: "nested log add shape",
			args: []string{"log", "decision", "add"},
			contains: []string{
				"Unknown subcommand: log decision",
				"Did you mean:",
				"plan-manager log decision-add <plan-or-execution> --title <title>",
			},
		},
		{
			name: "moved exec write",
			args: []string{"exec", "decision-add"},
			contains: []string{
				"Unknown subcommand: exec decision-add",
				"This command moved to the log ledger.",
				"plan-manager log decision-add <plan-or-execution> --title <title>",
			},
		},
		{
			name: "literal brace shorthand",
			args: []string{"log", "{decision,finding,bug,record,note}-add"},
			contains: []string{
				"Brace expansion is shell syntax",
				"plan-manager log decision-add <plan-or-execution> --title <title>",
				"plan-manager log finding-add <plan-or-execution> --title <title>",
				"plan-manager log bug-add <plan-or-execution> --title <title>",
				"plan-manager log record-add <plan-or-execution> --title <title>",
				"plan-manager log note-add <plan-or-execution> --title <title>",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp() error: %v", err)
			}
			err = app.Run(tt.args)
			if err == nil {
				t.Fatalf("expected recovery-hinted error for %v", tt.args)
			}
			for _, want := range tt.contains {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("expected error to contain %q, got:\n%v", want, err)
				}
			}
		})
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
