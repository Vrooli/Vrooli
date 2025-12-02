package cliapp

import (
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

func TestAppRoutesCommandsAndRunsStaleCheck(t *testing.T) {
	called := false
	stale := &cliutil.StaleChecker{
		BuildFingerprint: "fp",
		BuildSourceRoot:  t.TempDir(),
		FingerprintFunc: func(root string, skip ...string) (string, error) {
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
