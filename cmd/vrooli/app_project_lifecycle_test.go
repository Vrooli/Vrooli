//go:build integration
// +build integration

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/projectcli"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
)

var projectLifecyclePhaseBlocks = map[string]string{
	"clean": `    "clean": {
      "steps": [
        {
          "name": "capture-clean",
          "run": "mkdir -p build && printf 'clean\n' > build/clean.txt"
        }
      ]
    },`,
	"backup": `    "backup": {
      "steps": [
        {
          "name": "capture-backup",
          "run": "mkdir -p build && printf 'backup\n' > build/backup.txt"
        }
      ]
    },`,
	"restore": `    "restore": {
      "steps": [
        {
          "name": "capture-restore",
          "run": "mkdir -p build && printf 'restore\n' > build/restore.txt"
        }
      ]
    },`,
}

func TestSplitRunSetupUsesNativeProjectLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) { return root, nil }
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{Stale: false}, nil
	}
	capturedRoot := ""
	capturedHome := ""
	var capturedOpts projectsetup.Options
	app.RunProjectSetupFn = func(root, home string, opts projectsetup.Options, stdout, stderr io.Writer) error {
		capturedRoot = root
		capturedHome = home
		capturedOpts = opts
		return nil
	}

	code := app.Run([]string{"setup", "--environment", "minimal", "--resources", "none", "--yes", "yes", "--sudo-mode", "skip"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if capturedRoot != root || capturedHome != home {
		t.Fatalf("project setup called with root=%q home=%q", capturedRoot, capturedHome)
	}
	if capturedOpts.Environment != "minimal" || capturedOpts.Resources != "none" || capturedOpts.Yes != "yes" || capturedOpts.SudoMode != "skip" {
		t.Fatalf("setup options = %+v", capturedOpts)
	}
}

func TestSplitRunDevelopUsesNativeProjectLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := configuredApp()
	app.ResolveSourceRootFn = func() (string, error) { return root, nil }
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{Stale: false}, nil
	}
	calls := 0
	app.RunProjectDevelopFn = func(capturedRoot, capturedHome string, opts projectsetup.Options, stdout, stderr io.Writer) error {
		calls++
		if capturedRoot != root || capturedHome != home {
			t.Fatalf("unexpected project context root=%q home=%q", capturedRoot, capturedHome)
		}
		if opts != (projectsetup.Options{}) {
			t.Fatalf("develop options = %+v", opts)
		}
		return nil
	}

	code := app.Run([]string{"develop"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("develop exit code = %d", code)
	}
	if calls != 1 {
		t.Fatalf("develop calls = %d, want 1", calls)
	}
}

func TestSplitRunDevelopUsesProjectPortOverride(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	t.Setenv("VROOLI_API_PORT", "18094")
	app := newTestApp(root)
	app.RunProjectDevelopFn = func(capturedRoot, capturedHome string, opts projectsetup.Options, stdout, stderr io.Writer) error {
		if got := os.Getenv("VROOLI_API_PORT"); got != "18094" {
			t.Fatalf("VROOLI_API_PORT = %q", got)
		}
		if capturedRoot != root || capturedHome != home {
			t.Fatalf("unexpected project context root=%q home=%q", capturedRoot, capturedHome)
		}
		if opts != (projectsetup.Options{}) {
			t.Fatalf("develop options = %+v", opts)
		}
		return nil
	}

	code := app.Run([]string{"develop"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("develop exit code = %d", code)
	}
}

func TestSplitRunSetupPassesDryRunThroughProjectLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.RunProjectSetupFn = func(root, home string, opts projectsetup.Options, stdout, stderr io.Writer) error {
		if !opts.DryRun {
			t.Fatalf("setup options = %+v", opts)
		}
		return nil
	}

	code := app.Run([]string{"setup", "--dry-run"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
}

func TestSplitRunSetupReportsUnsupportedHostAtCLILevel(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.RunProjectSetupFn = func(root, home string, opts projectsetup.Options, stdout, stderr io.Writer) error {
		return errors.New("unsupported platform: vrooli setup is not supported on darwin (project-level setup/develop still depend on Linux-oriented shell steps)")
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"setup"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "not supported on darwin") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSplitRunDevelopReportsUnsupportedHostAtCLILevel(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.RunProjectDevelopFn = func(root, home string, opts projectsetup.Options, stdout, stderr io.Writer) error {
		return errors.New("unsupported platform: vrooli develop is not supported on windows (project-level setup/develop still execute bash-defined lifecycle steps)")
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"develop"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), "not supported on windows") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestSplitRunProjectBackupUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	code := app.Run([]string{"backup"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(root, "build", "backup.txt"))
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "backup" {
		t.Fatalf("backup output = %q", string(data))
	}
}

func TestSplitRunProjectBuildUsesNativeProjectLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := newTestApp(root)
	calls := 0
	app.RunProjectBuildFn = func(capturedRoot, capturedHome string, stdout, stderr io.Writer) error {
		calls++
		if capturedRoot != root || capturedHome != home {
			t.Fatalf("unexpected project context root=%q home=%q", capturedRoot, capturedHome)
		}
		return nil
	}

	code := app.Run([]string{"build"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if calls != 1 {
		t.Fatalf("build calls = %d, want 1", calls)
	}
}

func TestSplitRunProjectCleanUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	code := app.Run([]string{"clean"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(root, "build", "clean.txt"))
	if err != nil {
		t.Fatalf("read clean file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "clean" {
		t.Fatalf("clean output = %q", string(data))
	}
}

func TestSplitRunProjectRestoreUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	code := app.Run([]string{"restore"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(root, "build", "restore.txt"))
	if err != nil {
		t.Fatalf("read restore file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "restore" {
		t.Fatalf("restore output = %q", string(data))
	}
}

func TestSplitRunProjectLifecycleCommandsShowHelp(t *testing.T) {
	root := t.TempDir()
	app := newTestApp(root)

	commands := []string{"backup", "build", "clean", "restore"}
	for _, command := range commands {
		var stdout bytes.Buffer
		if code := app.Run([]string{command, "--help"}, &stdout, &bytes.Buffer{}); code != 0 {
			t.Fatalf("%s --help exit code = %d", command, code)
		}
		if !strings.Contains(stdout.String(), projectcli.ProjectPhaseHelpText(command)) {
			t.Fatalf("%s help = %q", command, stdout.String())
		}
	}
}

func TestSplitRunProjectBackupErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "backup")
}

func TestSplitRunProjectBuildErrorsWhenPhaseUndefined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.RunProjectBuildFn = func(root, home string, stdout, stderr io.Writer) error {
		return errors.New("build failed")
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"build"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("build unexpectedly succeeded")
	}
	if !strings.Contains(stderr.String(), "build failed") {
		t.Fatalf("build stderr = %q", stderr.String())
	}
}

func TestSplitRunProjectCleanErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "clean")
}

func TestSplitRunProjectRestoreErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "restore")
}

func assertProjectLifecycleCommandPhaseUndefined(t *testing.T, command string) {
	t.Helper()
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixtureWithoutPhase(t, root, command)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	var stderr bytes.Buffer
	code := app.Run([]string{command}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatalf("%s unexpectedly succeeded", command)
	}
	if !strings.Contains(stderr.String(), "is not defined") {
		t.Fatalf("%s stderr = %q", command, stderr.String())
	}
}

func writeProjectLifecycleFixtureWithoutPhase(t *testing.T, root, phase string) {
	t.Helper()
	writeProjectLifecycleFixture(t, root)

	block, ok := projectLifecyclePhaseBlocks[phase]
	if !ok {
		t.Fatalf("unknown project lifecycle phase %q", phase)
	}

	path := repocontractmeta.ProjectServiceManifestPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	original := string(data)
	updated := strings.Replace(original, block+"\n", "", 1)
	if updated == original {
		updated = strings.Replace(original, ",\n"+strings.TrimSuffix(block, ","), "", 1)
	}
	if updated == original {
		updated = strings.Replace(original, strings.TrimSuffix(block, ",")+"\n", "", 1)
	}
	if updated == original {
		t.Fatalf("phase block %q not found in %s", phase, path)
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
