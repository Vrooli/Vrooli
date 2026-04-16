package cliapp

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestParseGlobalFlags(t *testing.T) {
	api := ""
	global := GlobalOptions{ColorEnabled: true}
	args := []string{"--api-base", "http://example.com", "--no-color", "run"}

	remaining, err := ParseGlobalFlags(args, &global, &api)
	if err != nil {
		t.Fatalf("ParseGlobalFlags: %v", err)
	}
	if api != "http://example.com" || global.APIBaseOverride != "http://example.com" {
		t.Fatalf("expected api override to propagate, got %q", api)
	}
	if global.ColorEnabled {
		t.Fatalf("expected color disabled")
	}
	if len(remaining) != 1 || remaining[0] != "run" {
		t.Fatalf("unexpected remaining args: %v", remaining)
	}
}

func TestParseGlobalFlagsMissingValue(t *testing.T) {
	global := GlobalOptions{}
	_, err := ParseGlobalFlags([]string{"--api-base"}, &global, nil)
	if err == nil {
		t.Fatalf("expected missing value error")
	}
}

func TestParseGlobalFlagsStopsAtCommandBoundary(t *testing.T) {
	api := ""
	global := GlobalOptions{ColorEnabled: true}
	args := []string{"run", "--api-base", "http://command-local"}

	remaining, err := ParseGlobalFlags(args, &global, &api)
	if err != nil {
		t.Fatalf("ParseGlobalFlags: %v", err)
	}
	if api != "" || global.APIBaseOverride != "" {
		t.Fatalf("expected command-local --api-base to remain untouched, got %q", api)
	}
	if len(remaining) != 3 || remaining[0] != "run" || remaining[1] != "--api-base" || remaining[2] != "http://command-local" {
		t.Fatalf("unexpected remaining args: %v", remaining)
	}
}

func TestAppRoutesCommandsAndRunsStaleCheck(t *testing.T) {
	called := false
	stale := &cliutil.StaleChecker{
		BuildFingerprint: "fp",
		BuildSourceRoot:  t.TempDir(),
		FingerprintFunc: func(spec cliutil.FreshnessSpec) (string, error) {
			called = true
			return "fp", nil
		},
		LookPathFunc: func(file string) (string, error) {
			return "/usr/bin/go", nil
		},
	}

	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", NeedsAPI: true, Run: func(args []string) error { return nil }},
		},
	}

	app := NewApp(AppOptions{
		Name:         "demo",
		Version:      "0.0.1",
		Commands:     []CommandGroup{group},
		ColorEnabled: DefaultColorEnabled(),
		OnColor:      func(enabled bool) {},
		StaleChecker: stale,
	})

	if err := app.Run([]string{"run"}); err != nil {
		t.Fatalf("app run: %v", err)
	}
	if !called {
		t.Fatalf("expected stale checker to run for NeedsAPI command")
	}
}

func TestAppUnknownCommand(t *testing.T) {
	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error { return nil }},
		},
	}
	app := NewApp(AppOptions{
		Name:         "demo",
		Version:      "0.0.1",
		Commands:     []CommandGroup{group},
		ColorEnabled: DefaultColorEnabled(),
	})

	err := app.Run([]string{"missing"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "Unknown command") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestAppPreflightRuns(t *testing.T) {
	preflightCalled := false
	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error { return nil }},
		},
	}
	app := NewApp(AppOptions{
		Name:     "demo",
		Commands: []CommandGroup{group},
		Preflight: func(cmd Command, global GlobalOptions) error {
			preflightCalled = true
			return nil
		},
	})

	if err := app.Run([]string{"run"}); err != nil {
		t.Fatalf("expected run to succeed: %v", err)
	}
	if !preflightCalled {
		t.Fatalf("expected preflight to be invoked")
	}
}

func TestAppPreflightErrorStopsRun(t *testing.T) {
	group := CommandGroup{
		Title: "Demo",
		Commands: []Command{
			{Name: "run", Description: "Run demo", Run: func(args []string) error {
				t.Fatalf("command should not execute when preflight fails")
				return nil
			}},
		},
	}
	app := NewApp(AppOptions{
		Name:     "demo",
		Commands: []CommandGroup{group},
		Preflight: func(cmd Command, global GlobalOptions) error {
			return errors.New("blocked")
		},
	})

	if err := app.Run([]string{"run"}); err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected preflight error, got %v", err)
	}
}

func TestAppPreflightReceivesFullSubcommandName(t *testing.T) {
	var gotName string
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{
			{
				Name: "campaigns",
				Subcommands: []Command{
					{Name: "list", NeedsAPI: true, Run: func(args []string) error { return nil }},
				},
			},
		},
		Preflight: func(cmd Command, global GlobalOptions) error {
			gotName = cmd.Name
			return nil
		},
	})

	if err := app.Run([]string{"campaigns", "list"}); err != nil {
		t.Fatalf("expected run to succeed: %v", err)
	}
	if gotName != "campaigns list" {
		t.Fatalf("preflight command name = %q, want %q", gotName, "campaigns list")
	}
}

func TestAppCommandHelpSkipsStaleCheckAndPreflight(t *testing.T) {
	var output bytes.Buffer
	staleCalled := false
	preflightCalled := false

	app := NewApp(AppOptions{
		Name: "demo",
		Commands: []CommandGroup{{
			Title: "Demo",
			Commands: []Command{{
				Name:        "status",
				Description: "Show health",
				Usage:       "demo status [--json]",
				HelpText:    "Use --json to print the raw payload.",
				NeedsAPI:    true,
				Run: func(args []string) error {
					t.Fatal("command should not run for help")
					return nil
				},
			}},
		}},
		StaleChecker: &cliutil.StaleChecker{
			BuildFingerprint: "fp",
			BuildSourceRoot:  t.TempDir(),
			FingerprintFunc: func(spec cliutil.FreshnessSpec) (string, error) {
				staleCalled = true
				return "fp", nil
			},
		},
		Preflight: func(cmd Command, global GlobalOptions) error {
			preflightCalled = true
			return nil
		},
	})

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	done := make(chan struct{})
	go func() {
		_, _ = output.ReadFrom(r)
		close(done)
	}()

	runErr := app.Run([]string{"status", "--help"})
	_ = w.Close()
	os.Stdout = originalStdout
	<-done
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	if staleCalled {
		t.Fatalf("stale checker should not run for command help")
	}
	if preflightCalled {
		t.Fatalf("preflight should not run for command help")
	}
	text := output.String()
	if !strings.Contains(text, "demo status [--json]") {
		t.Fatalf("expected command usage in help output, got %q", text)
	}
}

func TestAppSubcommandHelpSkipsStaleCheckAndPreflight(t *testing.T) {
	staleCalled := false
	preflightCalled := false
	app := NewApp(AppOptions{
		Name: "demo",
		SubcommandGroups: []SubcommandGroup{{
			Name:        "chat",
			Description: "Chat operations",
			NeedsAPI:    true,
			Subcommands: []Command{{
				Name:        "list",
				Description: "List chats",
				Usage:       "demo chat list [--json]",
				Run: func(args []string) error {
					t.Fatal("subcommand should not run for help")
					return nil
				},
			}},
		}},
		StaleChecker: &cliutil.StaleChecker{
			BuildFingerprint: "fp",
			BuildSourceRoot:  t.TempDir(),
			FingerprintFunc: func(spec cliutil.FreshnessSpec) (string, error) {
				staleCalled = true
				return "fp", nil
			},
		},
		Preflight: func(cmd Command, global GlobalOptions) error {
			preflightCalled = true
			return nil
		},
	})

	if err := app.Run([]string{"chat", "list", "--help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if staleCalled {
		t.Fatalf("stale checker should not run for subcommand help")
	}
	if preflightCalled {
		t.Fatalf("preflight should not run for subcommand help")
	}
}
