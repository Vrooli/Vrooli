package main

import (
	"flag"
	"testing"
)

// [REQ:P0-001] Test requireArg validates positional arguments
func TestRequireArg(t *testing.T) {
	t.Run("returns error when no args", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		err := requireArg(fs, "test <id>")
		if err == nil {
			t.Fatal("expected error")
		}
		if got := err.Error(); got != "usage: test <id>" {
			t.Errorf("got %q, want %q", got, "usage: test <id>")
		}
	})

	t.Run("succeeds with args", func(t *testing.T) {
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		if err := fs.Parse([]string{"my-id"}); err != nil {
			t.Fatal(err)
		}
		if err := requireArg(fs, "test <id>"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// [REQ:P0-001] Test unmarshalBody wraps errors consistently
func TestUnmarshalBody(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		var result struct {
			Name string `json:"name"`
		}
		if err := unmarshalBody([]byte(`{"name":"test"}`), &result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "test" {
			t.Errorf("got %q, want %q", result.Name, "test")
		}
	})

	t.Run("invalid JSON wraps error", func(t *testing.T) {
		var result struct{}
		err := unmarshalBody([]byte(`{bad json`), &result)
		if err == nil {
			t.Fatal("expected error")
		}
		if got := err.Error(); len(got) < 16 || got[:16] != "parse response: " {
			t.Errorf("expected error starting with 'parse response: ', got %q", got)
		}
	})
}

// [REQ:P0-001] Test newFlagSet creates flag set with ContinueOnError
func TestNewFlagSet(t *testing.T) {
	fs := newFlagSet("my-command")
	if fs == nil {
		t.Fatal("expected non-nil FlagSet")
	}
	if fs.Name() != "my-command" {
		t.Errorf("expected name %q, got %q", "my-command", fs.Name())
	}
	// ContinueOnError means Parse returns error instead of calling os.Exit
	err := fs.Parse([]string{"--nonexistent"})
	if err == nil {
		t.Error("expected error for unknown flag with ContinueOnError")
	}
}

// [REQ:P0-001] Test cmdFlags creates flag set with --json and parses args
func TestCmdFlags(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	t.Run("parses json flag", func(t *testing.T) {
		fs, jsonOut, err := app.cmdFlags("test", []string{"--json", "arg1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !*jsonOut {
			t.Error("expected --json to be true")
		}
		if fs.NArg() != 1 || fs.Arg(0) != "arg1" {
			t.Errorf("expected 1 positional arg 'arg1', got %d args", fs.NArg())
		}
	})

	t.Run("defaults json to false", func(t *testing.T) {
		_, jsonOut, err := app.cmdFlags("test", []string{"arg1"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *jsonOut {
			t.Error("expected --json to default to false")
		}
	})

	t.Run("returns error on bad flag", func(t *testing.T) {
		_, _, err := app.cmdFlags("test", []string{"--nonexistent"})
		if err == nil {
			t.Error("expected error for unknown flag")
		}
	})
}
