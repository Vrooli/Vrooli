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

func TestRunGuessedProgramHelpUsesLibraryHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"run", "program-runtime.prepare-operation", "--help"}); err != nil {
		t.Fatalf("app.Run(run <program> --help) error: %v", err)
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

func TestProgramRuntimeCommandHintRewritesProgramInvocations(t *testing.T) {
	cases := map[string]string{
		"prepare-operation --input name=device-control.do-task":                            "program-runtime library run program-runtime.prepare-operation --input name=device-control.do-task",
		"run program-runtime.prepare-operation --input name=agent-manager.friction-digest": "program-runtime library run program-runtime.prepare-operation --input name=agent-manager.friction-digest",
		"device-control.do-task --input task=volume":                                       "program-runtime library run device-control.do-task --input task=volume",
	}
	for invocation, want := range cases {
		hint := programRuntimeCommandHint(strings.Fields(invocation))
		if !strings.Contains(hint, "Run this invocation as:\n  "+want) {
			t.Errorf("%s: hint %q lacks %q", invocation, hint, want)
		}
	}
	for _, invocation := range []string{"validate scenario device-control"} {
		hint := programRuntimeCommandHint(strings.Fields(invocation))
		if strings.Contains(hint, "Run this invocation as") || !strings.Contains(hint, "library run program-runtime.prepare-operation") {
			t.Errorf("%s: expected the generic guide, got %q", invocation, hint)
		}
	}
	if hint := programRuntimeCommandHint([]string{"run", "--help"}); !strings.Contains(hint, "Run this invocation as:\n  program-runtime library run --help") {
		t.Errorf("run --help: expected the library runner help command, got %q", hint)
	}
	if hint := programRuntimeCommandHint([]string{"prepare-operation", "--help"}); !strings.Contains(hint, "Run this invocation as:\n  program-runtime library run program-runtime.prepare-operation --help") {
		t.Errorf("prepare-operation --help: expected the program help command, got %q", hint)
	}
	if got := shellQuote("name=a b"); got != "'name=a b'" {
		t.Errorf("shellQuote = %s", got)
	}
}

func TestLibraryRunArgumentsRoutesRecognizedShorthand(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "bare program name with inputs",
			args: []string{"prepare-operation", "--input", "name=device-control.do-task", "--json"},
			want: []string{"library", "run", "program-runtime.prepare-operation", "--input", "name=device-control.do-task", "--json"},
		},
		{
			name: "run alias with qualified name",
			args: []string{"run", "program-runtime.prepare-operation", "--input", "name=agent-manager.friction-digest"},
			want: []string{"library", "run", "program-runtime.prepare-operation", "--input", "name=agent-manager.friction-digest"},
		},
		{
			name: "run alias with bare name and no inputs",
			args: []string{"run", "portfolio-audit"},
			want: []string{"library", "run", "program-runtime.portfolio-audit"},
		},
		{
			name: "exec alias with qualified name",
			args: []string{"exec", "device-control.do-task", "--input", "task=volume"},
			want: []string{"library", "run", "device-control.do-task", "--input", "task=volume"},
		},
		{
			name: "research program",
			args: []string{"research/topic/program", "--input", "key=value"},
			want: []string{"library", "run", "research/topic/program", "--input", "key=value"},
		},
		{
			name: "unqualified library program name with inputs",
			args: []string{"do-task", "--input", "task=volume"},
			want: []string{"library", "run", "program-runtime.do-task", "--input", "task=volume"},
		},
		{
			name: "canonical library run with inputs is left alone",
			args: []string{"library", "run", "research/run-freshness", "--input", "quiet_hours=48"},
			want: nil,
		},
		{
			name: "help never dispatches a program",
			args: []string{"run", "program-runtime.prepare-operation", "--input", "name=agent-manager.friction-digest", "--help"},
			want: []string{"library", "run", "--help"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := libraryRunArguments(tc.args)
			if len(got) != len(tc.want) {
				t.Fatalf("libraryRunArguments(%q) = %q, want %q", tc.args, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("libraryRunArguments(%q) = %q, want %q", tc.args, got, tc.want)
				}
			}
		})
	}

	for _, args := range [][]string{{"prepare-operation"}, {"prepare-operation", "--help"}, {"validate", "scenario", "program-runtime"}} {
		if got := libraryRunArguments(args); got != nil {
			t.Errorf("libraryRunArguments(%q) = %q, want no rewrite", args, got)
		}
	}
}
