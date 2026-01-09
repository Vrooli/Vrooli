package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"deployment-manager/cli/cmdutil"
)

func TestHelpListsCommandGroups(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	t.Cleanup(func() { os.Stdout = origStdout })
	os.Stdout = w

	if err := app.Run([]string{"help"}); err != nil {
		t.Fatalf("help failed: %v", err)
	}
	_ = w.Close()

	out, _ := io.ReadAll(r)
	for _, section := range []string{"Overview", "Profiles", "Swaps", "Deployments", "Configuration"} {
		if !strings.Contains(string(out), section) {
			t.Fatalf("expected help output to include %q; got: %s", section, string(out))
		}
	}
}

func TestRunProfileRejectsUnknown(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	if err := app.runProfile([]string{"bogus"}); err == nil {
		t.Fatalf("expected error for unknown subcommand")
	}
}

func TestGlobalFormatFlagIsConsumed(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	cmdutil.SetGlobalFormat("json")
	if err := app.Run([]string{"--format", "table", "help"}); err != nil {
		t.Fatalf("help failed with global format: %v", err)
	}
	if got := cmdutil.GlobalFormat(); got != "table" {
		t.Fatalf("expected global format to be table, got %s", got)
	}
}
