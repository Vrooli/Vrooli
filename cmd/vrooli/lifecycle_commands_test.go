package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/cli/topcli"
)

func TestLifecycleCommandIsHiddenFromMainHelpButSupportsDirectHelp(t *testing.T) {
	var help bytes.Buffer
	topcli.RenderMainHelp(&help, topcli.CommandSpecs())
	if strings.Contains(help.String(), "\nlifecycle") || strings.Contains(help.String(), "Internal lifecycle command plumbing") {
		t.Fatalf("main help should hide lifecycle command: %q", help.String())
	}

	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) {
		t.Fatal("root resolution should be skipped for lifecycle help")
		return "", nil
	}
	app.CheckStalenessFn = nil

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := app.Run([]string{"lifecycle", "--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), projectcli.LifecycleProtectHelpText) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestLifecycleProtectRejectsUnmanagedExecution(t *testing.T) {
	root := t.TempDir()
	app := newTestApp(root)
	t.Setenv("VROOLI_LIFECYCLE_MANAGED", "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"lifecycle", "protect", "--", "node", "-e", "process.exit(0)"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "This UI must be run through the Vrooli lifecycle system.") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLifecycleProtectExecutesManagedCommandAndPreservesExitCode(t *testing.T) {
	root := t.TempDir()
	app := newTestApp(root)
	t.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	successScript := writeFakeExecutable(t, root, "scripts/ok.sh", "#!/usr/bin/env bash\nprintf '%s' \"$VROOLI_LIFECYCLE_MANAGED\"\n")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := app.Run([]string{"lifecycle", "protect", "--", successScript}, &stdout, &stderr); code != 0 {
		t.Fatalf("success exit code = %d", code)
	}
	if got := strings.TrimSpace(stdout.String()); got != "true" {
		t.Fatalf("stdout = %q", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}

	exitScript := writeFakeExecutable(t, root, "scripts/exit-7.sh", "#!/usr/bin/env bash\nexit 7\n")
	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"lifecycle", "protect", "--", filepath.Clean(exitScript)}, &stdout, &stderr); code != 7 {
		t.Fatalf("exit code = %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLifecycleProtectRequiresProtectedCommandAfterDoubleDash(t *testing.T) {
	root := t.TempDir()
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := app.Run([]string{"lifecycle", "protect"}, &stdout, &stderr); code != 1 {
		t.Fatalf("help exit code = %d", code)
	}
	rendered := stdout.String() + stderr.String()
	if !strings.Contains(rendered, projectcli.LifecycleProtectHelpText) {
		t.Fatalf("rendered = %q", rendered)
	}

	stdout.Reset()
	stderr.Reset()
	if code := app.Run([]string{"lifecycle", "protect", "node", "-e", "process.exit(0)"}, &stdout, &stderr); code != 1 {
		t.Fatalf("usage exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "lifecycle protect requires '--' before the protected command") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLifecycleCommandIsNotSuggestedForUnknownCommands(t *testing.T) {
	suggestions := newTestApp(t.TempDir()).Registry().SuggestTopLevel("lifecycl")
	for _, suggestion := range suggestions {
		if suggestion == "lifecycle" {
			t.Fatalf("hidden lifecycle command should not be suggested: %v", suggestions)
		}
	}
}
