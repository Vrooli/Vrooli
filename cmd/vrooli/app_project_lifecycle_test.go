package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var projectLifecyclePhaseBlocks = map[string]string{
	"build": `    "build": {
      "steps": [
        {
          "name": "capture-build",
          "run": "mkdir -p build && printf 'build\n' > build/build.txt"
        }
      ]
    },`,
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
	"deploy": `    "deploy": {
      "steps": [
        {
          "name": "capture-deploy",
          "run": "mkdir -p build && printf 'deploy\n' > build/deploy.txt"
        }
      ]
    },`,
}

func TestSplitRunSetupUsesNativeProjectLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return root, nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("setup should not route to bash: %+v", spec)
		return nil
	}
	capturedRoot := ""
	capturedHome := ""
	var capturedArgs []string
	app.runProjectSetup = func(root, home string, args []string, stdout, stderr io.Writer) error {
		capturedRoot = root
		capturedHome = home
		capturedArgs = append([]string(nil), args...)
		return nil
	}

	code := app.Run([]string{"setup", "--environment", "minimal", "--resources", "none", "--yes", "yes", "--sudo-mode", "skip"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if capturedRoot != root || capturedHome != home {
		t.Fatalf("project setup called with root=%q home=%q", capturedRoot, capturedHome)
	}
	if got := strings.Join(capturedArgs, "|"); got != "--environment|minimal|--resources|none|--yes|yes|--sudo-mode|skip" {
		t.Fatalf("setup args = %q", got)
	}
}

func TestSplitRunDevelopUsesNativeProjectLifecycle(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	t.Setenv("HOME", home)
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return root, nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("develop should not route to bash: %+v", spec)
		return nil
	}
	calls := 0
	app.runProjectDevelop = func(capturedRoot, capturedHome string, args []string, stdout, stderr io.Writer) error {
		calls++
		if capturedRoot != root || capturedHome != home {
			t.Fatalf("unexpected project context root=%q home=%q", capturedRoot, capturedHome)
		}
		if len(args) != 0 {
			t.Fatalf("develop args = %v", args)
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
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("develop should not route to bash: %+v", spec)
		return nil
	}
	app.runProjectDevelop = func(capturedRoot, capturedHome string, args []string, stdout, stderr io.Writer) error {
		if got := os.Getenv("VROOLI_API_PORT"); got != "18094" {
			t.Fatalf("VROOLI_API_PORT = %q", got)
		}
		if capturedRoot != root || capturedHome != home {
			t.Fatalf("unexpected project context root=%q home=%q", capturedRoot, capturedHome)
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
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("setup should not route to bash: %+v", spec)
		return nil
	}
	app.runProjectSetup = func(root, home string, args []string, stdout, stderr io.Writer) error {
		if got := strings.Join(args, "|"); got != "--dry-run" {
			t.Fatalf("setup args = %q", got)
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
	app.runProjectSetup = func(root, home string, args []string, stdout, stderr io.Writer) error {
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
	app.runProjectDevelop = func(root, home string, args []string, stdout, stderr io.Writer) error {
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
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("backup should not route to bash: %+v", spec)
		return nil
	}

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

func TestSplitRunProjectBuildUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("build should not route to bash: %+v", spec)
		return nil
	}

	code := app.Run([]string{"build"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(root, "build", "build.txt"))
	if err != nil {
		t.Fatalf("read build file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "build" {
		t.Fatalf("build output = %q", string(data))
	}
}

func TestSplitRunProjectCleanUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("clean should not route to bash: %+v", spec)
		return nil
	}

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

func TestSplitRunProjectDeployUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("deploy should not route to bash: %+v", spec)
		return nil
	}

	code := app.Run([]string{"deploy"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(root, "build", "deploy.txt"))
	if err != nil {
		t.Fatalf("read deploy file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "deploy" {
		t.Fatalf("deploy output = %q", string(data))
	}
}

func TestSplitRunProjectRestoreUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("restore should not route to bash: %+v", spec)
		return nil
	}

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

	commands := []string{"backup", "build", "clean", "deploy", "restore"}
	for _, command := range commands {
		var stdout bytes.Buffer
		if code := app.Run([]string{command, "--help"}, &stdout, &bytes.Buffer{}); code != 0 {
			t.Fatalf("%s --help exit code = %d", command, code)
		}
		if !strings.Contains(stdout.String(), "Usage: vrooli "+command) {
			t.Fatalf("%s help = %q", command, stdout.String())
		}
	}
}

func TestSplitRunProjectBackupErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "backup")
}

func TestSplitRunProjectBuildErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "build")
}

func TestSplitRunProjectCleanErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "clean")
}

func TestSplitRunProjectDeployErrorsWhenPhaseUndefined(t *testing.T) {
	assertProjectLifecycleCommandPhaseUndefined(t, "deploy")
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
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("%s should not route to bash: %+v", command, spec)
		return nil
	}

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

	path := filepath.Join(root, ".vrooli", "service.json")
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
