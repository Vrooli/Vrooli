package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=5 | LAST: 2026-04-11

func TestParseArgsRecognizesLeadingGlobalFlags(t *testing.T) {
	parsed, err := parseArgs([]string{"--json", "--verbose", "--no-color", "scenario", "list"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if parsed.command != "scenario" {
		t.Fatalf("command = %q", parsed.command)
	}
	if strings.Join(parsed.args, ",") != "list" {
		t.Fatalf("args = %v", parsed.args)
	}
	if !parsed.globals.json || !parsed.globals.verbose || !parsed.globals.noColor {
		t.Fatalf("globals = %+v", parsed.globals)
	}
}

func TestConsumeInlineGlobalFlagsPromotesCommandScopedGlobals(t *testing.T) {
	globals, args := consumeInlineGlobalFlags(globalOptions{}, []string{"--scenarios", "--json", "--verbose", "--no-color"})
	if !globals.json || !globals.verbose || !globals.noColor {
		t.Fatalf("globals = %+v", globals)
	}
	if got := strings.Join(args, ","); got != "--scenarios" {
		t.Fatalf("args = %q", got)
	}
}

func TestRunScenarioTestUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioTestPhaseFixture(t, root, "alpha")
	writeScenarioPortRegistryFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario test should not route to bash: %+v", spec)
		return nil
	}
	app.rebuildAndReexec = func(args []string) error {
		t.Fatalf("unexpected rebuild")
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "test", "alpha", "unit"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios", "alpha", "coverage", "selector.txt"))
	if err != nil {
		t.Fatalf("read selector file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "unit" {
		t.Fatalf("selector = %q", string(data))
	}
}

func TestRunSetupUsesNativeProjectLifecycle(t *testing.T) {
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

func TestRunDevelopUsesNativeProjectLifecycle(t *testing.T) {
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

func TestRunDevelopUsesProjectPortOverride(t *testing.T) {
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

func TestRunSetupPassesDryRunThroughProjectLifecycle(t *testing.T) {
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

func TestRunSetupReportsUnsupportedHostAtCLILevel(t *testing.T) {
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

func TestRunDevelopReportsUnsupportedHostAtCLILevel(t *testing.T) {
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

func TestRunProjectBackupUsesNativePhaseRunner(t *testing.T) {
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

func TestRunProjectBuildUsesNativePhaseRunner(t *testing.T) {
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

func TestRunProjectCleanUsesNativePhaseRunner(t *testing.T) {
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

func TestRunProjectDeployUsesNativePhaseRunner(t *testing.T) {
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

func TestRunProjectRestoreUsesNativePhaseRunner(t *testing.T) {
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

func TestRunProjectLifecycleCommandsShowHelp(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeProjectLifecycleFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("project lifecycle help should not route to bash: %+v", spec)
		return nil
	}

	for _, command := range []string{"build", "clean", "deploy", "backup", "restore"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			code := app.Run([]string{command, "--help"}, &stdout, &bytes.Buffer{})
			if code != 0 {
				t.Fatalf("run exit code = %d", code)
			}
			if got := stdout.String(); !strings.Contains(got, "Usage: vrooli "+command) {
				t.Fatalf("stdout = %q", got)
			}
			if _, err := os.Stat(filepath.Join(root, "build", command+".txt")); !os.IsNotExist(err) {
				t.Fatalf("expected no lifecycle output for help, stat err=%v", err)
			}
		})
	}
}

func TestRunProjectBackupErrorsWhenPhaseUndefined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioPortRegistryFixture(t, root)
	writeTestFile(t, root, ".vrooli/service.json", `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
	}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("backup should not route to bash: %+v", spec)
		return nil
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"backup"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), `project lifecycle phase "backup" is not defined`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunProjectBuildErrorsWhenPhaseUndefined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioPortRegistryFixture(t, root)
	writeTestFile(t, root, ".vrooli/service.json", `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
	}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("build should not route to bash: %+v", spec)
		return nil
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"build"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), `project lifecycle phase "build" is not defined`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunProjectCleanErrorsWhenPhaseUndefined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioPortRegistryFixture(t, root)
	writeTestFile(t, root, ".vrooli/service.json", `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
	}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("clean should not route to bash: %+v", spec)
		return nil
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"clean"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), `project lifecycle phase "clean" is not defined`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunProjectDeployErrorsWhenPhaseUndefined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioPortRegistryFixture(t, root)
	writeTestFile(t, root, ".vrooli/service.json", `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
	}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("deploy should not route to bash: %+v", spec)
		return nil
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"deploy"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), `project lifecycle phase "deploy" is not defined`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunProjectRestoreErrorsWhenPhaseUndefined(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioPortRegistryFixture(t, root)
	writeTestFile(t, root, ".vrooli/service.json", `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
	}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("restore should not route to bash: %+v", spec)
		return nil
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"restore"}, &bytes.Buffer{}, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if !strings.Contains(stderr.String(), `project lifecycle phase "restore" is not defined`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunNoStaleCheckBypassesFreshnessProbe(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioSetupOnlyFixture(t, root, "alpha")
	writeScenarioPortRegistryFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.isStale = func() bool {
		t.Fatalf("stale check should be skipped when --no-stale-check is set")
		return false
	}
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario setup should not route to bash: %+v", spec)
		return nil
	}

	code := app.Run([]string{"--no-stale-check", "scenario", "setup", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "scenarios", "alpha", "build", "setup.txt")); err != nil {
		t.Fatalf("expected native setup output: %v", err)
	}
}

func TestRunScenarioSetupReportsUndefinedPhase(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioWithoutSetupFixture(t, root, "alpha")
	writeScenarioPortRegistryFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario setup should not route to bash: %+v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "setup", "alpha", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"status": "undefined"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunScenarioListJSONOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioPortRegistryFixture(t, root)
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "_artifacts"), 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario list should not shell to bash")
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "list", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.Contains(stdout.String(), "_artifacts") {
		t.Fatalf("list output should exclude directories without service.json: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"name": "alpha"`) {
		t.Fatalf("list output missing alpha: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"running": 1`) {
		t.Fatalf("list summary missing running count: %s", stdout.String())
	}
}

func TestRunScenarioInfoJSONOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "info", "alpha", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"name": "alpha"`) {
		t.Fatalf("info output missing name: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"sandbox_redirected": false`) {
		t.Fatalf("info output missing sandbox flag: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"status": "running"`) {
		t.Fatalf("info output missing runtime status: %s", stdout.String())
	}
}

func TestRunScenarioStatusJSONDoesNotRequireAPI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	t.Setenv("VROOLI_API_PORT", "1")
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "status", "alpha", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"status": "running"`) {
		t.Fatalf("status output missing running state: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ports": {`) {
		t.Fatalf("status output missing ports object: %s", stdout.String())
	}
}

func TestRunScenarioListHumanOutputIncludesPorts(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario list should not shell to bash")
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "list", "--include-ports"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Available scenarios") {
		t.Fatalf("missing list header: %s", output)
	}
	if !strings.Contains(output, "alpha - Alpha scenario") {
		t.Fatalf("missing scenario line: %s", output)
	}
	if !strings.Contains(output, "API_PORT=18080") {
		t.Fatalf("missing live port output: %s", output)
	}
}

func TestRunScenarioStatusHumanOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "status", "alpha"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Scenario: alpha") {
		t.Fatalf("missing scenario header: %s", output)
	}
	if !strings.Contains(output, "Status: running") {
		t.Fatalf("missing status line: %s", output)
	}
	if !strings.Contains(output, "Processes:") {
		t.Fatalf("missing processes section: %s", output)
	}
}

func TestRunScenarioStartStopRestartLifecycleCommands(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleScenarioService(t, root, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("week 3 lifecycle commands should not shell to bash: %#v", spec)
		return nil
	}

	t.Cleanup(func() {
		var stdout bytes.Buffer
		_ = app.Run([]string{"scenario", "stop", "alpha"}, &stdout, &bytes.Buffer{})
	})

	var startOut bytes.Buffer
	var startErr bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha", "--json"}, &startOut, &startErr)
	if code != 0 {
		t.Fatalf("start exit code = %d, output=%s stderr=%s", code, startOut.String(), startErr.String())
	}
	if !strings.Contains(startOut.String(), `"status": "started"`) {
		t.Fatalf("start output missing started status: %s", startOut.String())
	}
	if !strings.Contains(startOut.String(), `"health": "healthy"`) {
		t.Fatalf("start output missing healthy status: %s", startOut.String())
	}

	startRecords, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after start: %v", err)
	}
	startLive := process.LiveRecords(startRecords)
	if len(startLive) != 1 {
		t.Fatalf("live records after start = %#v", startLive)
	}
	firstPID := startLive[0].PID
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", fmt.Sprintf(".port_%d.lock", startLive[0].Port))
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("expected port lock after start: %v", err)
	}

	var restartOut bytes.Buffer
	var restartErr bytes.Buffer
	code = app.Run([]string{"scenario", "restart", "alpha", "--json"}, &restartOut, &restartErr)
	if code != 0 {
		t.Fatalf("restart exit code = %d, output=%s stderr=%s", code, restartOut.String(), restartErr.String())
	}
	if !strings.Contains(restartOut.String(), `"status": "restarted"`) {
		t.Fatalf("restart output missing restarted status: %s", restartOut.String())
	}

	restartRecords, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after restart: %v", err)
	}
	restartLive := process.LiveRecords(restartRecords)
	if len(restartLive) != 1 {
		t.Fatalf("live records after restart = %#v", restartLive)
	}
	if restartLive[0].PID == firstPID {
		t.Fatalf("expected restart to replace PID, still %d", firstPID)
	}

	var stopOut bytes.Buffer
	var stopErr bytes.Buffer
	code = app.Run([]string{"scenario", "stop", "alpha", "--json"}, &stopOut, &stopErr)
	if code != 0 {
		t.Fatalf("stop exit code = %d, output=%s stderr=%s", code, stopOut.String(), stopErr.String())
	}
	if !strings.Contains(stopOut.String(), `"status": "stopped"`) {
		t.Fatalf("stop output missing stopped status: %s", stopOut.String())
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected port lock to be removed, stat err=%v", err)
	}
	finalRecords, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after stop: %v", err)
	}
	if len(process.LiveRecords(finalRecords)) != 0 {
		t.Fatalf("expected no live records after stop: %#v", finalRecords)
	}
}

func TestRunScenarioStartSupportsCustomPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	customPath := t.TempDir()
	writeLifecycleScenarioServiceAtPath(t, root, customPath, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("custom-path lifecycle start should not shell to bash: %#v", spec)
		return nil
	}

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha", "--path", customPath, "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("start exit code = %d, output=%s", code, stdout.String())
	}

	records, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	live := process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("live records = %#v", live)
	}
	if live[0].WorkingDir != customPath {
		t.Fatalf("working_dir = %q, want %q", live[0].WorkingDir, customPath)
	}

	var stopOut bytes.Buffer
	code = app.Run([]string{"scenario", "stop", "alpha", "--json"}, &stopOut, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("stop exit code = %d, output=%s", code, stopOut.String())
	}
}

func TestRunScenarioStartCleanStaleRemovesDeadLock(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	port := reserveFreePort(t)
	writeFixedPortLifecycleScenarioService(t, root, "alpha", port)

	stateDir := filepath.Join(home, ".vrooli", "state", "scenarios")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", stateDir, err)
	}
	lockPath := filepath.Join(stateDir, fmt.Sprintf(".port_%d.lock", port))
	if err := os.WriteFile(lockPath, []byte("ghost:999999:1\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", lockPath, err)
	}

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha", "--clean-stale", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("start exit code = %d, output=%s", code, stdout.String())
	}

	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read %s: %v", lockPath, err)
	}
	if !strings.HasPrefix(string(lockData), "alpha:") {
		t.Fatalf("lock contents = %q", string(lockData))
	}

	var stopOut bytes.Buffer
	code = app.Run([]string{"scenario", "stop", "alpha", "--json"}, &stopOut, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("stop exit code = %d, output=%s", code, stopOut.String())
	}
}

func TestRunScenarioStartBestEffortCapturesFailedDependencies(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeBestEffortLifecycleScenarioService(t, root, "alpha", "missing-dep")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha", "--best-effort", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("start exit code = %d, output=%s", code, stdout.String())
	}

	var payload struct {
		Success   bool `json:"success"`
		Scenarios []struct {
			Name               string   `json:"name"`
			Status             string   `json:"status"`
			FailedDependencies []string `json:"failed_dependencies"`
		} `json:"scenarios"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if !payload.Success || len(payload.Scenarios) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Scenarios[0].Status != "started" {
		t.Fatalf("status = %q", payload.Scenarios[0].Status)
	}
	if len(payload.Scenarios[0].FailedDependencies) != 1 || payload.Scenarios[0].FailedDependencies[0] != "missing-dep" {
		t.Fatalf("failed dependencies = %v", payload.Scenarios[0].FailedDependencies)
	}

	if _, err := os.Stat(filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "degraded.json")); err != nil {
		t.Fatalf("expected degraded.json after best-effort start: %v", err)
	}

	var stopOut bytes.Buffer
	code = app.Run([]string{"scenario", "stop", "alpha", "--json"}, &stopOut, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("stop exit code = %d, output=%s", code, stopOut.String())
	}
}

func TestRunScenarioStartReportsAlreadyRunning(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleScenarioService(t, root, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("already-running lifecycle start should not shell to bash: %#v", spec)
		return nil
	}

	t.Cleanup(func() {
		var stdout bytes.Buffer
		_ = app.Run([]string{"scenario", "stop", "alpha"}, &stdout, &bytes.Buffer{})
	})

	var firstOut bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha", "--json"}, &firstOut, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("first start exit code = %d, output=%s", code, firstOut.String())
	}

	records, err := process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after first start: %v", err)
	}
	live := process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("live records after first start = %#v", live)
	}
	firstPID := live[0].PID

	var secondOut bytes.Buffer
	code = app.Run([]string{"scenario", "start", "alpha", "--json"}, &secondOut, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("second start exit code = %d, output=%s", code, secondOut.String())
	}
	if !strings.Contains(secondOut.String(), `"status": "already_running"`) {
		t.Fatalf("expected already_running status, output=%s", secondOut.String())
	}

	records, err = process.ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords after second start: %v", err)
	}
	live = process.LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("live records after second start = %#v", live)
	}
	if live[0].PID != firstPID {
		t.Fatalf("expected already-running start to preserve pid %d, got %d", firstPID, live[0].PID)
	}
}

func TestRunScenarioHelpListsMigratedCommands(t *testing.T) {
	app := newTestApp("/repo")

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "info") || !strings.Contains(output, "Show scenario metadata and runtime summary") {
		t.Fatalf("missing info help line: %s", output)
	}
	if !strings.Contains(output, "status") || !strings.Contains(output, "Show scenario runtime status") {
		t.Fatalf("missing status help line: %s", output)
	}
}

func TestRunScenarioInfoRequiresName(t *testing.T) {
	app := newTestApp("/repo")

	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "info"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "requires a scenario name") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunScenarioListRejectsUnknownFlag(t *testing.T) {
	app := newTestApp("/repo")

	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "list", "--bogus"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown option for scenario list") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunTriggersRebuildBeforeDispatch(t *testing.T) {
	app := newTestApp("/repo")
	app.isStale = func() bool { return true }

	var rebuiltArgs []string
	app.rebuildAndReexec = func(args []string) error {
		rebuiltArgs = append([]string(nil), args...)
		return nil
	}
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("dispatcher should not run when stale rebuild succeeds")
		return nil
	}

	code := app.Run([]string{"scenario", "list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.Join(rebuiltArgs, "|") != "scenario|list" {
		t.Fatalf("rebuilt args = %v", rebuiltArgs)
	}
}

func TestRunReportsStaleCheckFailure(t *testing.T) {
	app := newTestApp("/repo")
	app.checkStaleness = func() (buildinfo.StaleCheck, error) {
		return buildinfo.StaleCheck{}, errors.New("fingerprint targets drifted")
	}

	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "list"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Runtime error: stale binary check failed: fingerprint targets drifted") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Use --no-stale-check for local experiments") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunInfoCommandUsesManifestAndListMode(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":["docs/context.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"info", "--list"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.TrimSpace(stdout.String()) != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("info list output = %q", stdout.String())
	}
}

func TestRunInfoCommandRejectsUnknownOption(t *testing.T) {
	err := runInfoCommand("/repo", globalOptions{}, []string{"--bogus"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown option for info") {
		t.Fatalf("runInfoCommand error = %v", err)
	}
}

func TestRunInfoCommandErrorsWhenNoSourcesConfigured(t *testing.T) {
	originalDefaults := infoDefaultFiles
	infoDefaultFiles = nil
	t.Cleanup(func() {
		infoDefaultFiles = originalDefaults
	})

	err := runInfoCommand(t.TempDir(), globalOptions{}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no context sources defined") {
		t.Fatalf("runInfoCommand error = %v", err)
	}
}

func TestRunVersionJSONOutput(t *testing.T) {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return "/repo", nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "--version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}

	var payload map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal version json: %v", err)
	}
	if payload["root"] != "/repo" {
		t.Fatalf("root = %q", payload["root"])
	}
	if payload["cli_version"] != cliVersion {
		t.Fatalf("cli_version = %q", payload["cli_version"])
	}
}

func TestRunDispatchesTopLevelCommandsToExpectedHandlers(t *testing.T) {
	t.Run("resource status uses native controller", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeResourceStatusFixture(t, root, "fixture-resource", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)

		t.Setenv("HOME", home)
		app := newTestApp(root)
		app.execCommand = func(spec commandSpec) error {
			t.Fatalf("resource status should not route through CLI bash shim: %+v", spec)
			return nil
		}

		var stdout bytes.Buffer
		code := app.Run([]string{"resource", "status", "fixture-resource"}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("run exit code = %d", code)
		}
		output := stdout.String()
		if !strings.Contains(output, "fixture-resource") || !strings.Contains(output, "healthy") {
			t.Fatalf("stdout = %q", output)
		}
	})

	t.Run("status uses native project controller", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeProjectLifecycleFixture(t, root)
		writeTestScenarioService(t, root, "alpha", "Alpha scenario")
		writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

		t.Setenv("HOME", home)
		app := newTestApp(root)
		app.execCommand = func(spec commandSpec) error {
			t.Fatalf("status should not route through CLI bash shim: %+v", spec)
			return nil
		}

		var stdout bytes.Buffer
		code := app.Run([]string{"--json", "status", "--scenarios"}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("run exit code = %d", code)
		}
		if !strings.Contains(stdout.String(), `"scenarios_total": 1`) {
			t.Fatalf("stdout = %q", stdout.String())
		}
		if !strings.Contains(stdout.String(), `"name": "alpha"`) {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})

	t.Run("status accepts trailing global json flag", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeProjectLifecycleFixture(t, root)
		writeTestScenarioService(t, root, "alpha", "Alpha scenario")
		writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

		t.Setenv("HOME", home)
		app := newTestApp(root)
		app.execCommand = func(spec commandSpec) error {
			t.Fatalf("status should not route through CLI bash shim: %+v", spec)
			return nil
		}

		var stdout bytes.Buffer
		code := app.Run([]string{"status", "--scenarios", "--json"}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("run exit code = %d", code)
		}
		if !strings.Contains(stdout.String(), `"name": "alpha"`) {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})

	t.Run("doctor accepts trailing global json flag", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeProjectLifecycleFixture(t, root)

		t.Setenv("HOME", home)
		app := newTestApp(root)
		app.execCommand = func(spec commandSpec) error {
			t.Fatalf("doctor should not route through CLI bash shim: %+v", spec)
			return nil
		}

		var stdout bytes.Buffer
		code := app.Run([]string{"doctor", "--json"}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("run exit code = %d", code)
		}
		if !strings.Contains(stdout.String(), `"checks"`) {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})

	t.Run("stop accepts trailing global json flag", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeProjectLifecycleFixture(t, root)

		t.Setenv("HOME", home)
		app := newTestApp(root)
		app.execCommand = func(spec commandSpec) error {
			t.Fatalf("stop should not route through CLI bash shim: %+v", spec)
			return nil
		}

		var stdout bytes.Buffer
		code := app.Run([]string{"stop", "--json"}, &stdout, &bytes.Buffer{})
		if code != 0 {
			t.Fatalf("run exit code = %d", code)
		}
		if !strings.Contains(stdout.String(), `"success": true`) {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})
}

func TestRunCleanupLocksUsesNativeMaintenance(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "cleanup", "locks"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock removal, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), `"success": true`) || !strings.Contains(stdout.String(), `"21234"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunInfoListJSONOutput(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":["docs/context.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"--json", "info", "--list"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}

	var payload struct {
		Root  string   `json:"root"`
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal info json: %v", err)
	}
	if payload.Root != root {
		t.Fatalf("root = %q", payload.Root)
	}
	if len(payload.Files) != 1 || payload.Files[0] != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("files = %v", payload.Files)
	}
}

func TestRunInfoCommandSkipsMissingSourcesInJSONMode(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":["docs/context.md","docs/missing.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"--json", "info"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"path": "`+filepath.Join(root, "docs", "context.md")+`"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "missing.md") {
		t.Fatalf("stdout should skip missing files: %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Skipping missing context file: docs/missing.md") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunScenarioStatusAllJSONSummaryOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestScenarioService(t, root, "beta", "Beta scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "status", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"total_scenarios": 2`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"running": 1`) || !strings.Contains(stdout.String(), `"stopped": 1`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunScenarioStatusAllHumanOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestScenarioService(t, root, "beta", "Beta scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "status"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Name") || !strings.Contains(output, "alpha") || !strings.Contains(output, "beta") {
		t.Fatalf("status table output = %q", output)
	}
}

func TestBuildListPortsFallsBackToLiveEnvironment(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process environment inspection uses /proc on linux")
	}

	cmd := exec.Command("sleep", "30")
	cmd.Env = append(os.Environ(), "API_PORT=18080")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {EnvVar: "API_PORT"},
		},
	}

	var (
		listPorts []scenarioListPortOutput
		ports     map[string]int
	)
	for attempt := 0; attempt < 20; attempt++ {
		listPorts, ports = buildListPorts(manifest, []process.Record{{PID: cmd.Process.Pid, Step: "start-api"}})
		if ports["API_PORT"] == 18080 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(listPorts) != 0 {
		t.Fatalf("listPorts = %#v, want no explicit record-derived ports", listPorts)
	}
	if ports["API_PORT"] != 18080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestCanonicalSetupInstallContract(t *testing.T) {
	repoRoot := repoRootFromCaller(t)

	serviceData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "service.json"))
	if err != nil {
		t.Fatalf("read service.json: %v", err)
	}

	var service struct {
		Lifecycle struct {
			Setup struct {
				Steps []struct {
					Name string `json:"name"`
					Run  string `json:"run"`
				} `json:"steps"`
			} `json:"setup"`
		} `json:"lifecycle"`
	}
	if err := json.Unmarshal(serviceData, &service); err != nil {
		t.Fatalf("unmarshal service.json: %v", err)
	}

	var installStep string
	for _, step := range service.Lifecycle.Setup.Steps {
		if step.Name == "install-cli" {
			installStep = step.Run
			break
		}
	}
	if installStep == "" {
		t.Fatalf("expected lifecycle.setup.steps to include install-cli")
	}
	if !strings.Contains(installStep, "make install") {
		t.Fatalf("install-cli step does not invoke make install: %q", installStep)
	}

	makefileData, err := os.ReadFile(filepath.Join(repoRoot, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	makefileContents := string(makefileData)
	if !strings.Contains(makefileContents, "install: build") {
		t.Fatalf("Makefile no longer defines the install target contract")
	}
	if !strings.Contains(makefileContents, "INSTALL_DIR = $(HOME)/.vrooli/bin") {
		t.Fatalf("Makefile no longer targets ~/.vrooli/bin")
	}
}

func TestRunLocksCommandListsNativeState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"locks", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"stale": true`) || !strings.Contains(stdout.String(), `"port": 21234`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunLocksCommandHumanOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"locks"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "Port") || !strings.Contains(output, "21234") || !strings.Contains(output, "stale") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestRunDiagnosePortReturnsJSON(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"--json", "diagnose-port", "21234", "alpha"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"port": 21234`) || !strings.Contains(stdout.String(), `"scenario": "alpha"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunDiagnosePortHumanOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
	lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
	writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"diagnose-port", "21234", "alpha"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "Port 21234") || !strings.Contains(output, "Scenario: alpha") || !strings.Contains(output, "Recommended actions:") {
		t.Fatalf("stdout = %q", output)
	}
}

func TestInferPortEnvVarUsesStepPrefixesAndManifestNames(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
			"frontend": {
				EnvVar: "UI_PORT",
			},
		},
	}

	tests := []struct {
		step string
		want string
	}{
		{step: "start-api", want: "API_PORT"},
		{step: "launch-frontend", want: "UI_PORT"},
		{step: "serve-ui", want: "UI_PORT"},
		{step: "unknown", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.step, func(t *testing.T) {
			if got := inferPortEnvVar(manifest, tc.step); got != tc.want {
				t.Fatalf("inferPortEnvVar(%q) = %q, want %q", tc.step, got, tc.want)
			}
		})
	}
}

func TestBuildListPortsSortsAndMapsRecords(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
			"ui": {
				EnvVar: "UI_PORT",
			},
		},
	}

	listPorts, ports := buildListPorts(manifest, []process.Record{
		{Step: "start-ui", Port: 38080},
		{Step: "start-api", Port: 18080},
	})

	if len(listPorts) != 2 {
		t.Fatalf("list port count = %d, want 2", len(listPorts))
	}
	if listPorts[0].Key != "API_PORT" || listPorts[1].Key != "UI_PORT" {
		t.Fatalf("list port order = %#v", listPorts)
	}
	if ports["API_PORT"] != 18080 || ports["UI_PORT"] != 38080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestBuildListPortsKeepsFirstExplicitRecordPerPort(t *testing.T) {
	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
		},
	}

	listPorts, ports := buildListPorts(manifest, []process.Record{
		{Step: "start-api", Port: 18080},
		{Step: "run-api", Port: 19090},
	})

	if len(listPorts) != 1 {
		t.Fatalf("list port count = %d, want 1", len(listPorts))
	}
	if listPorts[0].Port != 18080 {
		t.Fatalf("explicit list port = %d, want 18080", listPorts[0].Port)
	}
	if ports["API_PORT"] != 18080 {
		t.Fatalf("summary port = %d, want first explicit port 18080", ports["API_PORT"])
	}
}

func TestParseOptionalScenarioNameAndJSONValidation(t *testing.T) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSON("status", false, []string{"alpha", "--json"})
	if err != nil {
		t.Fatalf("parseOptionalScenarioNameAndJSON: %v", err)
	}
	if name != "alpha" || !jsonFlag {
		t.Fatalf("name/json = %q/%v", name, jsonFlag)
	}

	if _, _, err := parseOptionalScenarioNameAndJSON("status", false, []string{"alpha", "beta"}); err == nil {
		t.Fatalf("expected duplicate scenario names to fail")
	}
	if _, _, err := parseOptionalScenarioNameAndJSON("status", false, []string{"--bogus"}); err == nil {
		t.Fatalf("expected unknown option to fail")
	}
	if _, _, err := parseScenarioNameAndJSON("info", false, nil); err == nil {
		t.Fatalf("expected missing scenario name to fail")
	}
}

func TestParseScenarioStartArgsAndSingleStartValidation(t *testing.T) {
	names, opts, jsonFlag, openAfter, err := parseScenarioStartArgs(false, []string{
		"alpha", "beta", "--json", "--open", "--best-effort", "--clean-stale", "--path", "/tmp/custom",
	})
	if err != nil {
		t.Fatalf("parseScenarioStartArgs: %v", err)
	}
	if got := strings.Join(names, ","); got != "alpha,beta" {
		t.Fatalf("names = %q", got)
	}
	if !jsonFlag || !openAfter || !opts.BestEffort || !opts.CleanStale || opts.CustomPath != "/tmp/custom" {
		t.Fatalf("opts/json/open = %+v/%v/%v", opts, jsonFlag, openAfter)
	}

	if _, _, _, _, err := parseScenarioStartArgs(false, []string{"--path"}); err == nil {
		t.Fatalf("expected missing --path value to fail")
	}
	if _, _, _, _, err := parseScenarioStartArgs(false, []string{"--bogus"}); err == nil {
		t.Fatalf("expected unknown option to fail")
	}
	if _, _, _, _, err := parseScenarioSingleStartArgs("restart", false, nil); err == nil {
		t.Fatalf("expected missing restart target to fail")
	}
	if _, _, _, _, err := parseScenarioSingleStartArgs("restart", false, []string{"alpha", "beta"}); err == nil {
		t.Fatalf("expected duplicate restart targets to fail")
	}
}

func TestRunScenarioStartOpenUsesNativeURLLauncher(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleScenarioService(t, root, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	app.execCommand = func(spec commandSpec) error { return nil }
	var opened scenarioSubprocessSpec
	app.lookPath = func(file string) (string, error) {
		if file == "xdg-open" {
			return "/usr/bin/xdg-open", nil
		}
		return "", exec.ErrNotFound
	}
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		opened = spec
		return nil
	}

	t.Cleanup(func() {
		var stdout bytes.Buffer
		_ = app.Run([]string{"scenario", "stop", "alpha"}, &stdout, &bytes.Buffer{})
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha", "--open"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("start --open exit code = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if opened.name != "/usr/bin/xdg-open" || len(opened.args) != 1 || !strings.HasPrefix(opened.args[0], "http://localhost:") {
		t.Fatalf("opened = %+v", opened)
	}
}

func TestRunScenarioRestartOpenUsesNativeURLLauncher(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleScenarioService(t, root, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	app.execCommand = func(spec commandSpec) error { return nil }
	var opened scenarioSubprocessSpec
	app.lookPath = func(file string) (string, error) {
		if file == "xdg-open" {
			return "/usr/bin/xdg-open", nil
		}
		return "", exec.ErrNotFound
	}
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		opened = spec
		return nil
	}

	t.Cleanup(func() {
		var stdout bytes.Buffer
		_ = app.Run([]string{"scenario", "stop", "alpha"}, &stdout, &bytes.Buffer{})
	})

	if code := app.Run([]string{"scenario", "start", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("initial start exit code = %d", code)
	}

	opened = scenarioSubprocessSpec{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "restart", "alpha", "--open"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("restart --open exit code = %d, stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if opened.name != "/usr/bin/xdg-open" || len(opened.args) != 1 || !strings.HasPrefix(opened.args[0], "http://localhost:") {
		t.Fatalf("opened = %+v", opened)
	}
}

func TestRunScenarioPortAndOpenCommandsUseNativeState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioServiceWithPorts(t, root, "alpha")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-time.Minute))
	writeScenarioProcessRecord(t, home, "alpha", "start-ui", os.Getpid(), 38080, time.Now().Add(-time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario port/open should not route to bash: %+v", spec)
		return nil
	}

	var portStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "port", "alpha", "UI_PORT"}, &portStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario port exit code = %d", code)
	}
	if strings.TrimSpace(portStdout.String()) != "38080" {
		t.Fatalf("port output = %q", portStdout.String())
	}

	var openStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "open", "alpha", "--print-url"}, &openStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario open exit code = %d", code)
	}
	if strings.TrimSpace(openStdout.String()) != "http://localhost:38080" {
		t.Fatalf("open output = %q", openStdout.String())
	}
}

func TestRunScenarioLogsCleanRemovesOrphans(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")

	logsDir := filepath.Join(home, ".vrooli", "logs", "scenarios", "alpha")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("mkdir logs: %v", err)
	}
	writeTestFile(t, home, ".vrooli/logs/scenarios/alpha/vrooli.develop.alpha.start-api.log", "expected\n")
	writeTestFile(t, home, ".vrooli/logs/scenarios/alpha/vrooli.develop.alpha.orphan-worker.log", "orphan\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	if code := app.Run([]string{"scenario", "logs", "alpha", "--clean"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario logs --clean exit code = %d", code)
	}
	if _, err := os.Stat(filepath.Join(logsDir, "vrooli.develop.alpha.start-api.log")); err != nil {
		t.Fatalf("expected log missing after clean: %v", err)
	}
	if _, err := os.Stat(filepath.Join(logsDir, "vrooli.develop.alpha.orphan-worker.log")); !os.IsNotExist(err) {
		t.Fatalf("expected orphan log to be removed, err=%v", err)
	}
}

func TestRunScenarioTemplateGenerateScaffoldsFiles(t *testing.T) {
	root := t.TempDir()
	templateBase := filepath.Join(root, "templates")
	writeScenarioTemplateFixture(t, templateBase, "demo")

	t.Setenv(config.TemplateBaseDirEnvVar, templateBase)
	app := newTestApp(root)
	app.execCommand = func(spec commandSpec) error {
		t.Fatalf("scenario generate should not route to bash: %+v", spec)
		return nil
	}

	var listStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "template", "list"}, &listStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario template list exit code = %d", code)
	}
	if !strings.Contains(listStdout.String(), "demo") {
		t.Fatalf("template list output = %q", listStdout.String())
	}

	var stdout bytes.Buffer
	code := app.Run([]string{
		"scenario", "generate", "demo",
		"--id", "alpha",
		"--display-name", "Alpha App",
		"--description", "Generated alpha",
		"--author", "Test Runner",
	}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("scenario generate exit code = %d", code)
	}
	readmePath := filepath.Join(root, "scenarios", "alpha", "README.md")
	data, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read generated README: %v", err)
	}
	if !strings.Contains(string(data), "Alpha App") || !strings.Contains(string(data), "Generated alpha") {
		t.Fatalf("generated README = %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(root, "scenarios", "alpha", "template.json")); !os.IsNotExist(err) {
		t.Fatalf("template.json should not be copied, err=%v", err)
	}
}

func TestRunScenarioRequirementsSnapshotReadsLatestFile(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "scenarios/alpha/coverage/requirements-sync/latest.json", `{"synced_at":"2026-04-10T12:00:00Z","tests_run":["vrooli scenario test alpha"]}`)

	app := newTestApp(root)

	var stdout bytes.Buffer
	if code := app.Run([]string{"scenario", "requirements", "snapshot", "alpha"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario requirements snapshot exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "Requirements snapshot (alpha)") || !strings.Contains(stdout.String(), "vrooli scenario test alpha") {
		t.Fatalf("snapshot output = %q", stdout.String())
	}
}

func TestRunScenarioHealFromSandboxRelaunchesAffectedScenarios(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioPortRegistryFixture(t, root)
	writeScenarioProcessRecordWithWorkingDir(t, home, "alpha", "start-api", 999999, 18080, time.Now().Add(-time.Minute), filepath.Join("/merged", "scenarios", "alpha"))

	t.Setenv("HOME", home)
	app := newTestApp(root)

	relaunchLog := filepath.Join(root, "relaunch.log")
	app.scenarioExecutable = func() (string, error) {
		return writeFakeExecutable(t, root, "bin/fake-vrooli", fmt.Sprintf("#!/usr/bin/env bash\nprintf '%%s\\n' \"$@\" >> %q\n", relaunchLog)), nil
	}

	if code := app.Run([]string{"scenario", "heal-from-sandbox", "--merged-path", "/merged"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario heal-from-sandbox exit code = %d", code)
	}
	waitForTestFile(t, relaunchLog)
	data, err := os.ReadFile(relaunchLog)
	if err != nil {
		t.Fatalf("read relaunch log: %v", err)
	}
	relaunched := strings.Fields(string(data))
	if len(relaunched) != 3 || strings.Join(relaunched, " ") != "scenario start alpha" {
		t.Fatalf("relaunched = %#v", relaunched)
	}
	if _, err := os.Stat(filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "start-api.json")); !os.IsNotExist(err) {
		t.Fatalf("expected process record to be removed, err=%v", err)
	}
}

func TestRunScenarioStartAllAndStopAllUseNativeLifecycle(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	alphaPort := reserveFreePort(t)
	betaPort := reserveFreePort(t)
	writeFixedPortLifecycleScenarioService(t, root, "alpha", alphaPort)
	writeFixedPortLifecycleScenarioService(t, root, "beta", betaPort)

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var startStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "start-all", "--json"}, &startStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario start-all exit code = %d", code)
	}
	var startPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Started []struct {
				Name string `json:"name"`
			} `json:"started"`
			Failed []struct {
				Name  string `json:"name"`
				Error string `json:"error"`
			} `json:"failed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(startStdout.Bytes(), &startPayload); err != nil {
		t.Fatalf("parse start-all payload: %v\noutput=%s", err, startStdout.String())
	}
	if !startPayload.Success {
		t.Fatalf("start-all reported failure: %s", startStdout.String())
	}
	if len(startPayload.Data.Failed) != 0 {
		t.Fatalf("expected no failed scenarios during start-all, got %#v\noutput=%s", startPayload.Data.Failed, startStdout.String())
	}
	if len(startPayload.Data.Started) != 2 {
		t.Fatalf("expected 2 started scenarios, got %#v\noutput=%s", startPayload.Data.Started, startStdout.String())
	}

	for _, name := range []string{"alpha", "beta"} {
		waitForTestFile(t, filepath.Join(home, ".vrooli", "processes", "scenarios", name, "start-api.json"))
	}

	var stopStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "stop-all", "--json"}, &stopStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario stop-all exit code = %d", code)
	}
	var stopPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Stopped []string `json:"stopped"`
			Failed  []struct {
				Name  string `json:"name"`
				Error string `json:"error"`
			} `json:"failed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stopStdout.Bytes(), &stopPayload); err != nil {
		t.Fatalf("parse stop-all payload: %v\noutput=%s", err, stopStdout.String())
	}
	if !stopPayload.Success {
		t.Fatalf("stop-all reported failure: %s", stopStdout.String())
	}
	if len(stopPayload.Data.Failed) != 0 {
		t.Fatalf("expected no failed scenarios during stop-all, got %#v\noutput=%s", stopPayload.Data.Failed, stopStdout.String())
	}
	if len(stopPayload.Data.Stopped) != 2 {
		t.Fatalf("expected 2 stopped scenarios, got %#v\noutput=%s", stopPayload.Data.Stopped, stopStdout.String())
	}

	for _, name := range []string{"alpha", "beta"} {
		if _, err := os.Stat(filepath.Join(home, ".vrooli", "processes", "scenarios", name, "start-api.json")); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be stopped, err=%v", name, err)
		}
	}
}

func TestRunScenarioUISmokeUsesTranslatedSubprocess(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	fakeCLI := writeFakeExecutable(t, home, ".vrooli/bin/test-genie", "#!/usr/bin/env bash\nexit 0\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var captured scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		captured = spec
		return nil
	}

	if code := app.Run([]string{"scenario", "ui-smoke", "alpha", "--json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario ui-smoke exit code = %d", code)
	}
	if captured.name != fakeCLI {
		t.Fatalf("subprocess name = %q", captured.name)
	}
	if strings.Join(captured.args, "|") != "ui-smoke|alpha|--json" {
		t.Fatalf("subprocess args = %v", captured.args)
	}
}

func TestRunScenarioRequirementsReportUsesTranslatedSubprocess(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	fakeCLI := writeFakeExecutable(t, home, ".vrooli/bin/test-genie", "#!/usr/bin/env bash\nexit 0\n")
	writeTestFile(t, root, "scenarios/alpha/requirements/index.json", `{"ok":true}`)

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var captured scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		captured = spec
		return nil
	}

	if code := app.Run([]string{"scenario", "requirements", "report", "alpha", "--json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario requirements report exit code = %d", code)
	}
	if captured.name != fakeCLI {
		t.Fatalf("subprocess name = %q", captured.name)
	}
	args := strings.Join(captured.args, "|")
	if !strings.Contains(args, "requirements|report") || !strings.Contains(args, "--dir|"+filepath.Join(root, "scenarios", "alpha")) || !strings.Contains(args, "--json") {
		t.Fatalf("subprocess args = %v", captured.args)
	}
}

func TestRunScenarioCompletenessUsesTranslatedSubprocess(t *testing.T) {
	root := t.TempDir()
	cliPath := writeFakeExecutable(t, root, "scenarios/scenario-completeness-scoring/cli/scenario-completeness-scoring", "#!/usr/bin/env bash\nexit 0\n")

	app := newTestApp(root)
	app.lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	var captured scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		captured = spec
		return nil
	}

	if code := app.Run([]string{"scenario", "completeness", "alpha", "--format", "json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario completeness exit code = %d", code)
	}
	if captured.name != cliPath {
		t.Fatalf("subprocess name = %q", captured.name)
	}
	if strings.Join(captured.args, "|") != "alpha|--format|json" {
		t.Fatalf("subprocess args = %v", captured.args)
	}
}

func TestRunScenarioLogsHelpAndViews(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioServiceWithPorts(t, root, "alpha")
	writeTestScenarioService(t, root, "beta", "Beta scenario")

	writeTestFile(t, home, ".vrooli/logs/alpha.log", "setup\nready\n")
	writeTestFile(t, home, ".vrooli/logs/scenarios/alpha/vrooli.develop.alpha.start-api.log", "api line 1\napi line 2\n")
	writeTestFile(t, home, ".vrooli/logs/scenarios/alpha/vrooli.develop.alpha.start-api.log.bak", "previous api line\n")
	writeTestFile(t, home, ".vrooli/logs/scenarios/alpha/vrooli.develop.alpha.orphan-worker.log", "orphan line\n")
	writeTestFile(t, home, ".vrooli/logs/scenarios/beta/vrooli.develop.beta.start-api.log", "beta line\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var help bytes.Buffer
	if code := app.Run([]string{"scenario", "logs", "--help"}, &help, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario logs --help exit code = %d", code)
	}
	if !strings.Contains(help.String(), "Available scenarios with logs:") || !strings.Contains(help.String(), "alpha") || !strings.Contains(help.String(), "beta") {
		t.Fatalf("help output = %q", help.String())
	}

	var runtimeStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "logs", "alpha", "--runtime", "--follow"}, &runtimeStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario logs --runtime --follow exit code = %d", code)
	}
	runtimeOutput := runtimeStdout.String()
	if !strings.Contains(runtimeOutput, "Non-interactive environment detected") || !strings.Contains(runtimeOutput, "vrooli.develop.alpha.start-api.log") {
		t.Fatalf("runtime output = %q", runtimeOutput)
	}

	var stepStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "logs", "alpha", "--step", "start-api", "--previous"}, &stepStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario logs --step --previous exit code = %d", code)
	}
	if !strings.Contains(stepStdout.String(), "previous api line") {
		t.Fatalf("step output = %q", stepStdout.String())
	}

	var lifecycleStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "logs", "alpha"}, &lifecycleStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario logs exit code = %d", code)
	}
	lifecycleOutput := lifecycleStdout.String()
	if !strings.Contains(lifecycleOutput, "Recent lifecycle execution for scenario 'alpha'") ||
		!strings.Contains(lifecycleOutput, "Background step logs:") ||
		!strings.Contains(lifecycleOutput, "start-api (develop)") ||
		!strings.Contains(lifecycleOutput, "start-ui (develop) [missing]") ||
		!strings.Contains(lifecycleOutput, "Orphaned background logs:") ||
		!strings.Contains(lifecycleOutput, "orphan-worker (develop)") {
		t.Fatalf("lifecycle output = %q", lifecycleOutput)
	}
}

func TestScenarioLogHelperReaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alpha.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	tail, err := readLastLogLines(path, 2)
	if err != nil {
		t.Fatalf("readLastLogLines: %v", err)
	}
	if string(tail) != "two\nthree\n" {
		t.Fatalf("tail = %q", string(tail))
	}

	delta, nextOffset, err := readScenarioLogDelta(path, int64(len("one\n")))
	if err != nil {
		t.Fatalf("readScenarioLogDelta initial: %v", err)
	}
	if string(delta) != "two\nthree\n" {
		t.Fatalf("delta = %q", string(delta))
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := file.WriteString("four\n"); err != nil {
		_ = file.Close()
		t.Fatalf("append log: %v", err)
	}
	_ = file.Close()

	delta, _, err = readScenarioLogDelta(path, nextOffset)
	if err != nil {
		t.Fatalf("readScenarioLogDelta appended: %v", err)
	}
	if string(delta) != "four\n" {
		t.Fatalf("appended delta = %q", string(delta))
	}
}

func TestRunScenarioTemplateShowAndHooks(t *testing.T) {
	root := t.TempDir()
	templateBase := filepath.Join(root, "templates")
	writeScenarioTemplateFixture(t, templateBase, "demo")
	writeTestFile(t, filepath.Join(templateBase, "demo"), "template.json", `{
  "name": "demo",
  "displayName": "Demo Template",
  "description": "Template test fixture",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id", "description": "Scenario id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name", "description": "Scenario name"},
    "SCENARIO_DESCRIPTION": {"flag": "description", "description": "Scenario description"}
  },
  "optionalVars": {
    "AUTHOR": {"flag": "author", "description": "Author", "default": "Generator Agent"},
    "DATE": {"flag": "date", "description": "Date", "default": "{{CURRENT_DATE}}"}
  },
  "docs": {
    "playbook": "https://example.com/template"
  },
  "postHooks": [
    {"description": "Echo hook", "cmd": "echo hook-ran", "cwd": "."}
  ]
}`)

	t.Setenv(config.TemplateBaseDirEnvVar, templateBase)
	app := newTestApp(root)

	var showStdout bytes.Buffer
	if code := app.Run([]string{"scenario", "template", "show", "demo"}, &showStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario template show exit code = %d", code)
	}
	showOutput := showStdout.String()
	if !strings.Contains(showOutput, "Post Hooks:") ||
		!strings.Contains(showOutput, "Echo hook") ||
		!strings.Contains(showOutput, "Docs:") ||
		!strings.Contains(showOutput, "playbook: https://example.com/template") ||
		!strings.Contains(showOutput, "Files:") ||
		!strings.Contains(showOutput, "README.md") ||
		!strings.Contains(showOutput, "vrooli scenario generate demo") {
		t.Fatalf("show output = %q", showOutput)
	}

	var captured scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		captured = spec
		return nil
	}

	var generateStdout bytes.Buffer
	if code := app.Run([]string{
		"scenario", "generate", "demo",
		"--id", "alpha",
		"--display-name", "Alpha App",
		"--description", "Generated alpha",
		"--run-hooks",
	}, &generateStdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario generate --run-hooks exit code = %d", code)
	}
	if captured.name != "bash" {
		t.Fatalf("hook subprocess = %+v", captured)
	}
	if strings.Join(captured.args, "|") != "-lc|echo hook-ran" {
		t.Fatalf("hook args = %v", captured.args)
	}
	if captured.dir != filepath.Join(root, "scenarios", "alpha") {
		t.Fatalf("hook dir = %q", captured.dir)
	}
	if !strings.Contains(generateStdout.String(), "[Hook 1] Echo hook") {
		t.Fatalf("generate output = %q", generateStdout.String())
	}
}

func TestScenarioTemplateParsersAndFormatting(t *testing.T) {
	manifest := scenarioTemplateManifest{
		RequiredVars: map[string]scenarioTemplateVar{
			"SCENARIO_ID":           {Flag: "id"},
			"SCENARIO_DISPLAY_NAME": {Flag: "display-name"},
			"SCENARIO_DESCRIPTION":  {Flag: "description"},
		},
		OptionalVars: map[string]scenarioTemplateVar{
			"AUTHOR": {Flag: "author"},
		},
	}

	var stderr bytes.Buffer
	opts, err := parseScenarioGenerateArgs([]string{
		"--id", "alpha",
		"--display-name=Alpha App",
		"--description", "Generated alpha",
		"--var", "CUSTOM=1",
		"--unknown", "mystery",
	}, manifest, &stderr)
	if err != nil {
		t.Fatalf("parseScenarioGenerateArgs: %v", err)
	}
	if opts.Values["SCENARIO_ID"] != "alpha" || opts.Values["SCENARIO_DISPLAY_NAME"] != "Alpha App" || opts.Values["SCENARIO_DESCRIPTION"] != "Generated alpha" || opts.Values["CUSTOM"] != "1" {
		t.Fatalf("values = %#v", opts.Values)
	}
	if !strings.Contains(stderr.String(), "unknown flag --unknown") {
		t.Fatalf("stderr = %q", stderr.String())
	}

	if _, _, _, err := parseScenarioTemplateFlag("--display-name", []string{"--display-name"}, 0); err == nil {
		t.Fatalf("expected parseScenarioTemplateFlag to reject missing value")
	}
	if _, _, err := parseScenarioTemplateKeyValue("broken"); err == nil {
		t.Fatalf("expected parseScenarioTemplateKeyValue to reject invalid pair")
	}
	if looksLikeTextFile([]byte{0}) {
		t.Fatalf("looksLikeTextFile should reject binary content")
	}

	requiredFlags := formatScenarioTemplateRequiredFlags(manifest)
	if !strings.Contains(requiredFlags, "--id <scenario_id>") || !strings.Contains(requiredFlags, "--display-name <scenario_display_name>") {
		t.Fatalf("required flags = %q", requiredFlags)
	}
}

func TestScenarioHelperCLIResolution(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	app := newTestApp(root)
	overrideCLI := writeFakeExecutable(t, root, "override/test-genie", "#!/usr/bin/env bash\nexit 0\n")
	t.Setenv("VROOLI_TEST_GENIE_CLI", overrideCLI)

	path, err := app.locateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("locateTestGenieCLI override: %v", err)
	}
	if path != overrideCLI {
		t.Fatalf("override path = %q", path)
	}

	t.Setenv("VROOLI_TEST_GENIE_CLI", "")
	homeCLI := writeFakeExecutable(t, home, ".vrooli/bin/test-genie", "#!/usr/bin/env bash\nexit 0\n")
	app.lookPath = func(file string) (string, error) { return "", exec.ErrNotFound }
	path, err = app.locateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("locateTestGenieCLI home: %v", err)
	}
	if path != homeCLI {
		t.Fatalf("home path = %q", path)
	}
	if err := os.Remove(homeCLI); err != nil {
		t.Fatalf("remove home CLI: %v", err)
	}

	pathCLI := writeFakeExecutable(t, root, "bin/test-genie", "#!/usr/bin/env bash\nexit 0\n")
	app.lookPath = func(file string) (string, error) { return pathCLI, nil }
	path, err = app.locateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("locateTestGenieCLI PATH: %v", err)
	}
	if path != pathCLI {
		t.Fatalf("PATH CLI = %q", path)
	}

	repoCLI := writeFakeExecutable(t, root, "scenarios/test-genie/cli/test-genie", "#!/usr/bin/env bash\nexit 0\n")
	app.lookPath = func(file string) (string, error) { return "", exec.ErrNotFound }
	path, err = app.locateTestGenieCLI(root, home)
	if err != nil {
		t.Fatalf("locateTestGenieCLI repo: %v", err)
	}
	if path != repoCLI {
		t.Fatalf("repo CLI = %q", path)
	}

	completenessCLI := writeFakeExecutable(t, root, "scenarios/scenario-completeness-scoring/cli/scenario-completeness-scoring", "#!/usr/bin/env bash\nexit 0\n")
	path, err = app.locateScenarioCompletenessCLI(root)
	if err != nil {
		t.Fatalf("locateScenarioCompletenessCLI: %v", err)
	}
	if path != completenessCLI {
		t.Fatalf("completeness CLI = %q", path)
	}
}

func TestScenarioHelperProcessUtilities(t *testing.T) {
	if writerSupportsStreaming(&bytes.Buffer{}) {
		t.Fatalf("bytes.Buffer should not be treated as a streaming writer")
	}

	app := newTestApp(t.TempDir())
	url := "http://localhost:1234"
	var opened scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		opened = spec
		return nil
	}
	app.lookPath = func(file string) (string, error) {
		switch runtime.GOOS {
		case "linux":
			if file == "xdg-open" {
				return "/usr/bin/xdg-open", nil
			}
		default:
			return "", exec.ErrNotFound
		}
		return "", exec.ErrNotFound
	}
	if err := app.openScenarioURL(url); err != nil {
		t.Fatalf("openScenarioURL: %v", err)
	}
	switch runtime.GOOS {
	case "linux":
		if opened.name != "/usr/bin/xdg-open" || strings.Join(opened.args, "|") != url {
			t.Fatalf("open spec = %+v", opened)
		}
	case "darwin":
		if opened.name != "open" || strings.Join(opened.args, "|") != url {
			t.Fatalf("open spec = %+v", opened)
		}
	case "windows":
		if opened.name != "cmd" || strings.Join(opened.args, "|") != "/c|start||"+url {
			t.Fatalf("open spec = %+v", opened)
		}
	}

	if runtime.GOOS == "windows" {
		t.Skip("bash-based helper smoke tests run on unix-like shells")
	}

	var stdout bytes.Buffer
	if err := runScenarioSubprocess(scenarioSubprocessSpec{
		name:   "bash",
		args:   []string{"-lc", "printf helper-ok"},
		stdout: &stdout,
		stderr: &bytes.Buffer{},
	}); err != nil {
		t.Fatalf("runScenarioSubprocess: %v", err)
	}
	if stdout.String() != "helper-ok" {
		t.Fatalf("subprocess stdout = %q", stdout.String())
	}
}

func TestLaunchDetachedScenarioPropagatesExpectedArgsAndEnv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("detached-process validation uses a bash fixture")
	}

	root := t.TempDir()
	app := newTestApp(root)
	argsPath := filepath.Join(root, "args.txt")
	envPath := filepath.Join(root, "env.txt")
	executable := writeFakeExecutable(t, root, "bin/fake-vrooli", fmt.Sprintf("#!/usr/bin/env bash\nprintf '%%s\\n' \"$@\" > %q\nenv | sort > %q\n", argsPath, envPath))
	app.scenarioExecutable = func() (string, error) { return executable, nil }

	t.Setenv("VROOLI_SANDBOX_ID", "sandbox-123")
	t.Setenv("VROOLI_SANDBOX_MERGED", "/merged")
	t.Setenv("VROOLI_SANDBOX_SCOPE", "scenarios/alpha")
	t.Setenv("SANDBOX_MERGED_DIR", "/merged")
	t.Setenv("VROOLI_SOURCE_ROOT", "/source-root")

	if err := app.launchDetachedScenario(root, globalOptions{json: true, verbose: true, noColor: true}, "start", "alpha"); err != nil {
		t.Fatalf("launchDetachedScenario: %v", err)
	}

	waitForTestFile(t, argsPath)
	waitForTestFile(t, envPath)

	argsData, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	args := strings.Fields(string(argsData))
	if got, want := strings.Join(args, "|"), "scenario|start|alpha|--json|--verbose|--no-color"; got != want {
		t.Fatalf("args = %q, want %q", got, want)
	}

	envData, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	env := string(envData)
	for _, forbidden := range []string{
		"VROOLI_SANDBOX_ID=",
		"VROOLI_SANDBOX_MERGED=",
		"VROOLI_SANDBOX_SCOPE=",
		"SANDBOX_MERGED_DIR=",
	} {
		if strings.Contains(env, forbidden) {
			t.Fatalf("env should not contain %q: %s", forbidden, env)
		}
	}
	if !strings.Contains(env, "VROOLI_ROOT="+root) || !strings.Contains(env, "VROOLI_SOURCE_ROOT=/source-root") || !strings.Contains(env, "NO_COLOR=1") {
		t.Fatalf("env = %s", env)
	}
}

func TestRunScenarioRequirementsHelpAndInitTranslation(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "alpha")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}
	fakeCLI := writeFakeExecutable(t, home, ".vrooli/bin/test-genie", "#!/usr/bin/env bash\nexit 0\n")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var help bytes.Buffer
	if code := app.Run([]string{"scenario", "requirements", "--help"}, &help, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario requirements --help exit code = %d", code)
	}
	if !strings.Contains(help.String(), "snapshot <name>") || !strings.Contains(help.String(), "manual-log <name> <req>") {
		t.Fatalf("help output = %q", help.String())
	}

	var captured scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		captured = spec
		return nil
	}

	if code := app.Run([]string{"--json", "scenario", "requirements", "init", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario requirements init exit code = %d", code)
	}
	if captured.name != fakeCLI {
		t.Fatalf("subprocess name = %q", captured.name)
	}
	args := strings.Join(captured.args, "|")
	if !strings.Contains(args, "requirements|init") || !strings.Contains(args, "--dir|"+scenarioDir) || !strings.Contains(args, "--scenario|alpha") {
		t.Fatalf("subprocess args = %v", captured.args)
	}
	if strings.Contains(args, "--json") {
		t.Fatalf("init translation should not add --json: %v", captured.args)
	}
	if captured.dir != scenarioDir {
		t.Fatalf("subprocess dir = %q", captured.dir)
	}
}

func TestRunScenarioRunAliasesStartValidation(t *testing.T) {
	err := runScenarioRunCommand("/repo", globalOptions{}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "scenario start requires at least one scenario name") {
		t.Fatalf("runScenarioRunCommand error = %v", err)
	}
}

func TestScenarioTemplateHelpAndManualHooksOutput(t *testing.T) {
	var templateHelp bytes.Buffer
	showScenarioTemplateHelp(&templateHelp)
	if !strings.Contains(templateHelp.String(), "vrooli scenario template show <template>") {
		t.Fatalf("template help = %q", templateHelp.String())
	}

	var generateHelp bytes.Buffer
	showScenarioGenerateHelp(&generateHelp)
	if !strings.Contains(generateHelp.String(), "--run-hooks") {
		t.Fatalf("generate help = %q", generateHelp.String())
	}

	var hooks bytes.Buffer
	writeScenarioTemplateHooks(&hooks, scenarioTemplateManifest{
		PostHooks: []scenarioTemplateHook{{Description: "Install deps", Cmd: "pnpm install"}},
	})
	if !strings.Contains(hooks.String(), "Install deps") {
		t.Fatalf("hook output = %q", hooks.String())
	}
}

func TestRunCleanupCommandRoutesTargets(t *testing.T) {
	t.Run("orphans", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)

		t.Setenv("HOME", home)
		err := runCleanupCommand(root, parsedArgs{
			args: []string{"orphans", "help"},
			globals: globalOptions{
				json:    true,
				verbose: true,
			},
		}, &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("runCleanupCommand: %v", err)
		}
	})

	t.Run("locks", func(t *testing.T) {
		root := t.TempDir()
		home := t.TempDir()
		writeTestFile(t, root, "scripts/resources/port_registry.json", `{"resource_ports":{},"reserved_ranges":{}}`)
		lockPath := filepath.Join(home, ".vrooli", "state", "scenarios", ".port_21234.lock")
		writeTestFile(t, filepath.Dir(lockPath), filepath.Base(lockPath), "ghost:999999:1\n")

		t.Setenv("HOME", home)
		err := runCleanupCommand(root, parsedArgs{
			args: []string{"locks"},
			globals: globalOptions{
				noColor: true,
			},
		}, &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("runCleanupCommand: %v", err)
		}
		if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
			t.Fatalf("expected stale lock removal, stat err=%v", err)
		}
	})
}

func TestRunCleanupCommandHelpAndUnknownTarget(t *testing.T) {
	var stdout bytes.Buffer
	if err := runCleanupCommand("/repo", parsedArgs{}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("runCleanupCommand help: %v", err)
	}
	if !strings.Contains(stdout.String(), "vrooli cleanup") {
		t.Fatalf("missing cleanup help output: %s", stdout.String())
	}

	err := runCleanupCommand("/repo", parsedArgs{args: []string{"bogus"}}, &bytes.Buffer{}, &bytes.Buffer{})
	if exitCode(err) != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode(err))
	}
	if !strings.Contains(err.Error(), "unknown cleanup target: bogus") {
		t.Fatalf("err = %v", err)
	}
}

func TestRunInfoCommandHelpAndJSONMissingFiles(t *testing.T) {
	root := t.TempDir()
	absoluteFile := filepath.Join(t.TempDir(), "extra.md")
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	writeTestFile(t, filepath.Dir(absoluteFile), filepath.Base(absoluteFile), "extra context\n")

	var help bytes.Buffer
	if err := runInfoCommand(root, globalOptions{}, []string{"--help"}, &help, &bytes.Buffer{}); err != nil {
		t.Fatalf("runInfoCommand help: %v", err)
	}
	if !strings.Contains(help.String(), "Usage: vrooli info [--list]") {
		t.Fatalf("missing help output: %s", help.String())
	}

	t.Setenv("VROOLI_INFO_FILES", "docs/context.md:"+absoluteFile+":docs/missing.md")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := runInfoCommand(root, globalOptions{json: true}, nil, &stdout, &stderr); err != nil {
		t.Fatalf("runInfoCommand json: %v", err)
	}

	var payload infoOutput
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal info payload: %v", err)
	}
	if len(payload.Files) != 2 {
		t.Fatalf("file count = %d, want 2", len(payload.Files))
	}
	if payload.Files[1].Path != absoluteFile {
		t.Fatalf("expected absolute info path to be preserved, got %q", payload.Files[1].Path)
	}
	if !strings.Contains(stderr.String(), "Skipping missing context file: docs/missing.md") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCollectInfoSourcesPrefersEnvAndFallsBackOnInvalidManifest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("VROOLI_INFO_FILES", "docs/context.md:/tmp/extra.md")

	files, warnings, err := collectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("collectInfoSources env: %v", err)
	}
	if got, want := strings.Join(files, ","), "docs/context.md,/tmp/extra.md"; got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v, want none", warnings)
	}

	t.Setenv("VROOLI_INFO_FILES", "")
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":`)

	files, warnings, err = collectInfoSourcesDetailed(root)
	if err != nil {
		t.Fatalf("collectInfoSources fallback: %v", err)
	}
	if got, want := strings.Join(files, ","), strings.Join(infoDefaultFiles, ","); got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Invalid info manifest") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestRunUnknownCommandSuggestsNearestMatch(t *testing.T) {
	app := newTestApp("/repo")

	var stderr bytes.Buffer
	code := app.Run([]string{"statuz"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Unknown command: statuz") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "status") {
		t.Fatalf("expected suggestion list in stderr, got %q", stderr.String())
	}
}

func TestShowVersionAndHelpOutput(t *testing.T) {
	var version bytes.Buffer
	if err := showVersion(&version, "/repo", globalOptions{}); err != nil {
		t.Fatalf("showVersion: %v", err)
	}
	if !strings.Contains(version.String(), "Vrooli CLI v"+cliVersion) {
		t.Fatalf("version output = %q", version.String())
	}

	var help bytes.Buffer
	showMainHelp(&help)
	if !strings.Contains(help.String(), "scenario") || !strings.Contains(help.String(), "Manage scenarios from their source locations") {
		t.Fatalf("help output = %q", help.String())
	}
}

func TestRunVersionDoesNotRequireRootResolution(t *testing.T) {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return "", errors.New("boom") }
	app.checkStaleness = nil

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Vrooli CLI v"+cliVersion) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunMainHelpAndUnknownCommandDoNotRequireRootResolution(t *testing.T) {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) {
		t.Fatal("root resolution should be skipped")
		return "", nil
	}
	app.checkStaleness = nil

	var help bytes.Buffer
	if code := app.Run([]string{"--help"}, &help, &bytes.Buffer{}); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(help.String(), "Vrooli CLI - AI Platform Management Tool") {
		t.Fatalf("help output = %q", help.String())
	}

	var stderr bytes.Buffer
	if code := app.Run([]string{"statuz"}, &bytes.Buffer{}, &stderr); code != 1 {
		t.Fatalf("unknown-command exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "Unknown command: statuz") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCommandEnvPreservesExistingSourceRootAndNoColor(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("LANG", "C")
	t.Setenv("VROOLI_SOURCE_ROOT", "/custom/source")

	env := configuredApp().commandEnv("/repo", globalOptions{noColor: true})
	got := strings.Join(env, "\n")
	if !strings.Contains(got, "VROOLI_ROOT=/repo") {
		t.Fatalf("env missing VROOLI_ROOT: %v", env)
	}
	if !strings.Contains(got, "VROOLI_SOURCE_ROOT=/custom/source") {
		t.Fatalf("env missing preserved source root: %v", env)
	}
	if !strings.Contains(got, "NO_COLOR=1") {
		t.Fatalf("env missing NO_COLOR: %v", env)
	}
}

func TestResolveInfoPathAndPassthroughFlags(t *testing.T) {
	absolute := resolveInfoPath("/repo", "/tmp/context.md")
	if absolute != "/tmp/context.md" {
		t.Fatalf("resolveInfoPath absolute = %q", absolute)
	}

	flags := passthroughFlags(globalOptions{json: true, verbose: true, noColor: true}, []string{"--json", "scenario"})
	if got, want := strings.Join(flags, ","), "--verbose,--no-color"; got != want {
		t.Fatalf("flags = %q, want %q", got, want)
	}
	if containsArg([]string{"alpha", "beta"}, "--json") {
		t.Fatalf("containsArg should not match absent flag")
	}
}

func TestRunExternalCommandAndExitCodeError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based smoke test uses sh")
	}

	if err := runExternalCommand(commandSpec{name: "sh", args: []string{"-c", "exit 0"}}); err != nil {
		t.Fatalf("runExternalCommand success: %v", err)
	}
	if err := runExternalCommand(commandSpec{name: "sh", args: []string{"-c", "exit 7"}}); err == nil {
		t.Fatalf("expected failing command to return an error")
	}

	if got := (exitCodeError{code: 7, message: "boom"}).Error(); got != "boom" {
		t.Fatalf("exitCodeError message = %q", got)
	}
	if got := (exitCodeError{code: 7}).Error(); got != "exit code 7" {
		t.Fatalf("exitCodeError default = %q", got)
	}
}

func TestRunScenarioStatusAllJSONOutput(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestScenarioService(t, root, "beta", "Beta scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	code := app.Run([]string{"scenario", "status", "--json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), `"total_scenarios": 2`) || !strings.Contains(stdout.String(), `"running": 1`) {
		t.Fatalf("status output = %s", stdout.String())
	}
}

func TestBuildScenarioStatusItemAndHumanWriters(t *testing.T) {
	startedAt := time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)
	fixedPort := 5432
	item := scenario.Scenario{
		Slug:       "alpha",
		Path:       "/repo/scenarios/alpha",
		Redirected: true,
		Manifest: scenario.ServiceManifest{
			Service: scenario.ServiceMetadata{
				Name:        "alpha",
				DisplayName: "Alpha",
				Description: "Alpha scenario",
				Version:     "0.1.0",
				Type:        "tool",
				Category:    "ops",
				Tags:        []string{"internal", "go"},
			},
			Ports: map[string]scenario.Port{
				"api": {
					EnvVar: "API_PORT",
					Range:  "15000-19999",
				},
				"db": {
					Port: &fixedPort,
				},
			},
			Lifecycle: scenario.Lifecycle{
				Version: "2.0.0",
			},
		},
	}
	runtimeState := process.ScenarioRuntime{
		ProcessCount: 1,
		Runtime:      "2m",
		StartedAt:    &startedAt,
		Records: []process.Record{
			{Step: "start-api", PID: 1234, Port: 18080, StartedAt: startedAt},
		},
	}

	status := buildScenarioStatusItem(item, runtimeState)
	if status.Status != "running" || status.Health != "running" {
		t.Fatalf("status item = %+v", status)
	}

	var infoOut bytes.Buffer
	writeScenarioInfoHuman(&infoOut, buildScenarioInfoData(item), buildScenarioRuntimeData(item.Manifest, runtimeState))
	if !strings.Contains(infoOut.String(), "Configured ports:") ||
		!strings.Contains(infoOut.String(), "API_PORT (api)") ||
		!strings.Contains(infoOut.String(), "DB_PORT (db) fixed=5432") ||
		!strings.Contains(infoOut.String(), "Version: 0.1.0") ||
		!strings.Contains(infoOut.String(), "Type: tool") ||
		!strings.Contains(infoOut.String(), "Category: ops") ||
		!strings.Contains(infoOut.String(), "Tags: internal, go") ||
		!strings.Contains(infoOut.String(), "Lifecycle version: 2.0.0") ||
		!strings.Contains(infoOut.String(), "Sandbox: using redirected scenario path") {
		t.Fatalf("scenario info output = %s", infoOut.String())
	}

	var tableOut bytes.Buffer
	writeScenarioStatusTable(&tableOut, []scenarioStatusItemOutput{status})
	if !strings.Contains(tableOut.String(), "Name") || !strings.Contains(tableOut.String(), "alpha") {
		t.Fatalf("scenario table output = %s", tableOut.String())
	}

	var statusOut bytes.Buffer
	writeScenarioStatusHuman(&statusOut, scenarioStatusSingleOutput{
		Scenario: status,
		Info:     buildScenarioInfoData(item),
		Runtime:  buildScenarioRuntimeData(item.Manifest, runtimeState),
	})
	if !strings.Contains(statusOut.String(), "Health: running") || !strings.Contains(statusOut.String(), "Processes:") {
		t.Fatalf("scenario status output = %s", statusOut.String())
	}
}

func TestLoadScenarioDetailMissingScenario(t *testing.T) {
	_, _, _, err := loadScenarioDetail(t.TempDir(), "missing")
	if err == nil || !strings.Contains(err.Error(), `scenario "missing" not found`) {
		t.Fatalf("loadScenarioDetail error = %v", err)
	}
}

func TestLoadScenarioStateFiltersUnknownProcessDirectories(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))
	writeScenarioProcessRecord(t, home, "ghost", "start-api", os.Getpid(), 28080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)

	scenarios, runtimes, err := loadScenarioState(root)
	if err != nil {
		t.Fatalf("loadScenarioState: %v", err)
	}
	if len(scenarios) != 1 || scenarios[0].Slug != "alpha" {
		t.Fatalf("scenarios = %#v", scenarios)
	}
	if _, ok := runtimes["alpha"]; !ok {
		t.Fatalf("expected alpha runtime, got %#v", runtimes)
	}
	if _, ok := runtimes["ghost"]; ok {
		t.Fatalf("unexpected stale runtime for undiscovered scenario: %#v", runtimes)
	}
}

func TestLoadScenarioDetailRejectsBrokenProcessMetadata(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestFile(t, home, ".vrooli/processes/scenarios/alpha/broken.json", "{broken")

	t.Setenv("HOME", home)

	if _, _, _, err := loadScenarioDetail(root, "alpha"); err == nil {
		t.Fatalf("expected invalid process metadata to fail scenario detail loading")
	}
}

func TestBuildListPortsFallsBackToEnvironment(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process environment inspection uses /proc on linux")
	}

	cmd := exec.Command("sleep", "30")
	cmd.Env = append(os.Environ(), "API_PORT=18080", "WS_PORT=28080")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	manifest := scenario.ServiceManifest{
		Ports: map[string]scenario.Port{
			"api": {
				EnvVar: "API_PORT",
			},
			"websocket": {
				EnvVar: "WS_PORT",
			},
		},
	}

	var listPorts []scenarioListPortOutput
	var ports map[string]int
	for attempt := 0; attempt < 20; attempt++ {
		listPorts, ports = buildListPorts(manifest, []process.Record{{
			PID:  cmd.Process.Pid,
			Step: "start-api",
			Port: 18080,
		}})
		if ports["WS_PORT"] == 28080 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(listPorts) != 1 || listPorts[0].Key != "API_PORT" {
		t.Fatalf("list ports = %#v", listPorts)
	}
	if ports["API_PORT"] != 18080 || ports["WS_PORT"] != 28080 {
		t.Fatalf("ports = %#v", ports)
	}
}

func TestCopyHelpersReturnIndependentSlices(t *testing.T) {
	originalStrings := []string{"alpha"}
	originalRecords := []process.Record{{PID: 1234}}
	copiedStrings := copyStrings(originalStrings)
	copiedRecords := copyProcessRecords(originalRecords)
	if len(copyStrings(nil)) != 0 || len(copyProcessRecords(nil)) != 0 {
		t.Fatalf("expected nil inputs to return empty slices")
	}

	copiedStrings[0] = "beta"
	copiedRecords[0].PID = 99
	if originalStrings[0] != "alpha" || originalRecords[0].PID != 1234 {
		t.Fatalf("expected copies to avoid mutating originals")
	}
}

func newTestApp(root string) *App {
	app := configuredApp()
	app.resolveSourceRoot = func() (string, error) { return root, nil }
	app.isStale = func() bool { return false }
	app.checkStaleness = nil
	return app
}

func writeTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeFakeExecutable(t *testing.T, root, rel, contents string) string {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func waitForTestFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func reserveFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func repoRootFromCaller(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func writeTestScenarioService(t *testing.T, root, name, description string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "` + strings.Title(strings.ReplaceAll(name, "-", " ")) + `",
    "description": "` + description + `",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "develop": {
      "description": "Run the scenario",
      "steps": [
        {
          "name": "start-api",
          "run": "sleep 10",
          "background": true
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeProjectLifecycleFixture(t *testing.T, root string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)
	path := filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha",
    "description": "Project-level lifecycle fixture",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "VROOLI_API_PORT",
      "port": 8092
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "data",
            "path": "data"
          }
        ]
      },
      "steps": [
        {
          "name": "capture-setup",
          "run": "mkdir -p data build && printf 'setup\n' >> build/setup-count.txt && printf '%s|%s|%s|%s|%s|%s|%s|%s|%s\n' \"${ENVIRONMENT:-}\" \"${RESOURCES:-}\" \"${YES:-}\" \"${SUDO_MODE:-}\" \"${TARGET:-}\" \"${LOCATION:-}\" \"${DRY_RUN:-false}\" \"${APP_ROOT:-}\" \"${SERVICE_JSON_PATH:-}\" > build/setup-env.txt && printf 'ready\n' > data/bootstrap.txt"
        },
        {
          "name": "add-data",
          "run": "printf 'data\n' >> data/bootstrap.txt"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "capture-develop",
          "run": "mkdir -p build && printf 'develop\n' >> build/develop-count.txt && printf '%s\n' \"${VROOLI_API_PORT:-}\" > build/develop-port.txt"
        }
      ]
    },
    "build": {
      "steps": [
        {
          "name": "capture-build",
          "run": "mkdir -p build && printf 'build\n' > build/build.txt"
        }
      ]
    },
    "clean": {
      "steps": [
        {
          "name": "capture-clean",
          "run": "mkdir -p build && printf 'clean\n' > build/clean.txt"
        }
      ]
    },
    "deploy": {
      "steps": [
        {
          "name": "capture-deploy",
          "run": "mkdir -p build && printf 'deploy\n' > build/deploy.txt"
        }
      ]
    },
    "backup": {
      "steps": [
        {
          "name": "capture-backup",
          "run": "mkdir -p build && printf 'backup\n' > build/backup.txt"
        }
      ]
    },
    "restore": {
      "steps": [
        {
          "name": "capture-restore",
          "run": "mkdir -p build && printf 'restore\n' > build/restore.txt"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	portRegistryPath := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.WriteFile(portRegistryPath, []byte("#!/usr/bin/env bash\nRESOURCE_PORTS=( [\"vrooli-api\"]=\"8092\" )\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryPath, err)
	}
	portRegistryJSONPath := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.WriteFile(portRegistryJSONPath, []byte("{\n  \"resource_ports\": {\n    \"vrooli-api\": 8092\n  },\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryJSONPath, err)
	}
}

func writeResourceStatusFixture(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	writeTestFile(t, root, ".vrooli/service.json", `{
  "service": {
    "name": "project-alpha",
    "displayName": "Project Alpha"
  },
  "dependencies": {
    "resources": {
      "`+name+`": {
        "enabled": true
      }
    }
  }
}`)
	writeTestFile(t, root, filepath.Join("resources", name, "resource.json"), `{
  "name": "`+name+`",
  "display_name": "`+name+`",
  "description": "Fixture resource",
  "template": "legacy-adapter",
  "driver": "legacy-adapter",
  "legacy_adapter": {
    "owner": "CLI tests",
    "decision_deadline": "2026-12-31",
    "final_disposition": "migrate",
    "legacy_cli_path": "resources/`+name+`/cli.sh"
  },
  "portability_tier": "partial",
  "platforms": {
    "linux": "supported",
    "macos": "partial",
    "windows": "unsupported"
  }
}`)
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\n"
	writeFakeExecutable(t, root, filepath.Join("resources", name, "cli.sh"), script)
}

func writeScenarioSetupOnlyFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Setup ` + strings.Title(name) + `",
    "description": "Setup validation scenario",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0",
    "setup": {
      "steps": [
        {
          "name": "write-file",
          "run": "mkdir -p build && printf 'ok\n' > build/setup.txt"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioWithoutSetupFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "No Setup ` + strings.Title(name) + `",
    "description": "Scenario without setup phase",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0"
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioTestPhaseFixture(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	scriptPath := filepath.Join(root, "scenarios", name, "run-test.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nset -e\nmkdir -p coverage\nprintf '%s\\n' \"$1\" > coverage/selector.txt\n"), 0o755); err != nil {
		t.Fatalf("write %s: %v", scriptPath, err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Test ` + strings.Title(name) + `",
    "description": "Test validation scenario",
    "version": "0.1.0"
  },
  "lifecycle": {
    "version": "2.0.0",
    "test": {
      "steps": [
        {
          "name": "run-tests",
          "run": "./run-test.sh"
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioServiceWithPorts(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Ports ` + strings.Title(name) + `",
    "description": "Port validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    },
    "ui": {
      "env_var": "UI_PORT",
      "range": "35000-39999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "sleep 10",
          "background": true
        },
        {
          "name": "start-ui",
          "run": "sleep 10",
          "background": true
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioTemplateFixture(t *testing.T, templateBase, name string) {
	t.Helper()
	manifestPath := filepath.Join(templateBase, name, "template.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(manifestPath), err)
	}
	manifest := `{
  "name": "` + name + `",
  "displayName": "Demo Template",
  "description": "Template test fixture",
  "requiredVars": {
    "SCENARIO_ID": {"flag": "id", "description": "Scenario id"},
    "SCENARIO_DISPLAY_NAME": {"flag": "display-name", "description": "Scenario name"},
    "SCENARIO_DESCRIPTION": {"flag": "description", "description": "Scenario description"}
  },
  "optionalVars": {
    "AUTHOR": {"flag": "author", "description": "Author", "default": "Generator Agent"},
    "DATE": {"flag": "date", "description": "Date", "default": "{{CURRENT_DATE}}"}
  }
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write %s: %v", manifestPath, err)
	}
	writeTestFile(t, filepath.Join(templateBase, name), "README.md", "# {{SCENARIO_DISPLAY_NAME}}\n\n{{SCENARIO_DESCRIPTION}}\n")
	writeTestFile(t, filepath.Join(templateBase, name), ".vrooli/service.json", `{"service":{"name":"{{SCENARIO_ID}}","displayName":"{{SCENARIO_DISPLAY_NAME}}","description":"{{SCENARIO_DESCRIPTION}}"}}`)
	writeTestFile(t, filepath.Join(templateBase, name), "requirements/index.json", `{"owner":"{{AUTHOR}}","date":"{{DATE}}"}`)
}

func writeLifecycleScenarioService(t *testing.T, root, name string) {
	t.Helper()
	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	writeScenarioPortRegistryFixture(t, root)
	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Lifecycle ` + strings.Title(name) + `",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      "checks": [
        {
          "name": "api",
          "type": "http",
          "target": "http://127.0.0.1:${API_PORT}/health",
          "critical": true,
          "timeout": 1000
        }
      ],
      "startup_grace_period": 1000,
      "timeout": 30000,
      "interval": 250
    },
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": [
              "api/mock-api"
            ]
          }
        ]
      },
      "steps": [
        {
          "name": "build-api",
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./mock-api",
          "background": true,
          "condition": {
            "file_exists": "api/mock-api"
          }
        }
      ]
    }
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeLifecycleScenarioServiceAtPath(t *testing.T, root, scenarioPath, name string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)

	servicePath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(servicePath), err)
	}

	data := `{
  "version": "1.0.0",
  "service": {
    "name": "` + name + `",
    "displayName": "Lifecycle ` + strings.Title(name) + `",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      "checks": [
        {
          "name": "api",
          "type": "http",
          "target": "http://127.0.0.1:${API_PORT}/health",
          "critical": true,
          "timeout": 1000
        }
      ],
      "startup_grace_period": 1000,
      "timeout": 30000,
      "interval": 250
    },
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": [
              "api/mock-api"
            ]
          }
        ]
      },
      "steps": [
        {
          "name": "build-api",
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./mock-api",
          "background": true,
          "condition": {
            "file_exists": "api/mock-api"
          }
        }
      ]
    }
  }
}`
	if err := os.WriteFile(servicePath, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", servicePath, err)
	}
}

func writeFixedPortLifecycleScenarioService(t *testing.T, root, name string, port int) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)

	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	data := fmt.Sprintf(`{
  "version": "1.0.0",
  "service": {
    "name": %q,
    "displayName": "Lifecycle %s",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "port": %d
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      "checks": [
        {
          "name": "api",
          "type": "http",
          "target": "http://127.0.0.1:${API_PORT}/health",
          "critical": true,
          "timeout": 1000
        }
      ],
      "startup_grace_period": 1000,
      "timeout": 30000,
      "interval": 250
    },
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": [
              "api/mock-api"
            ]
          }
        ]
      },
      "steps": [
        {
          "name": "build-api",
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./mock-api",
          "background": true,
          "condition": {
            "file_exists": "api/mock-api"
          }
        }
      ]
    }
  }
}`, name, strings.Title(name), port)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeBestEffortLifecycleScenarioService(t *testing.T, root, name, dependency string) {
	t.Helper()
	writeScenarioPortRegistryFixture(t, root)

	path := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}

	data := fmt.Sprintf(`{
  "version": "1.0.0",
  "service": {
    "name": %q,
    "displayName": "Lifecycle %s",
    "description": "Lifecycle validation scenario",
    "version": "0.1.0"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "version": "2.0.0",
    "health": {
      "checks": [
        {
          "name": "api",
          "type": "http",
          "target": "http://127.0.0.1:${API_PORT}/health",
          "critical": true,
          "timeout": 1000
        }
      ],
      "startup_grace_period": 1000,
      "timeout": 30000,
      "interval": 250
    },
    "setup": {
      "condition": {
        "checks": [
          {
            "type": "binaries",
            "targets": [
              "api/mock-api"
            ]
          }
        ]
      },
      "steps": [
        {
          "name": "build-api",
          "run": "mkdir -p api public && printf '#!/usr/bin/env bash\npython3 -m http.server \"$API_PORT\" --bind 127.0.0.1 --directory ../public\n' > api/mock-api && chmod +x api/mock-api && printf 'ok\n' > public/health"
        }
      ]
    },
    "develop": {
      "steps": [
        {
          "name": "start-api",
          "run": "cd api && ./mock-api",
          "background": true,
          "condition": {
            "file_exists": "api/mock-api"
          }
        }
      ]
    }
  },
  "dependencies": {
    "scenarios": {
      %q: {
        "type": "scenario",
        "required": true
      }
    }
  }
}`, name, strings.Title(name), dependency)
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioPortRegistryFixture(t *testing.T, root string) {
	t.Helper()
	portRegistryPath := filepath.Join(root, "scripts", "resources", "port_registry.sh")
	if err := os.MkdirAll(filepath.Dir(portRegistryPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(portRegistryPath), err)
	}
	if err := os.WriteFile(portRegistryPath, []byte("#!/usr/bin/env bash\ndeclare -g -A RESOURCE_PORTS=()\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryPath, err)
	}
	portRegistryJSONPath := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.WriteFile(portRegistryJSONPath, []byte("{\n  \"resource_ports\": {},\n  \"reserved_ranges\": {}\n}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", portRegistryJSONPath, err)
	}
}

func writeScenarioProcessRecord(t *testing.T, home, name, step string, pid, port int, startedAt time.Time) {
	t.Helper()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", name, step+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": ` + fmt.Sprintf("%d", pid) + `,
  "pgid": ` + fmt.Sprintf("%d", pid) + `,
  "process_id": "vrooli.develop.` + name + `.` + step + `",
  "phase": "develop",
  "scenario": "` + name + `",
  "step": "` + step + `",
  "command": "sleep 10",
  "working_dir": "` + filepath.Join("/repo/scenarios", name) + `",
  "log_file": "/tmp/` + name + `.log",
  "port": ` + fmt.Sprintf("%d", port) + `,
  "started_at": "` + startedAt.UTC().Format(time.RFC3339) + `",
  "status": "running"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeScenarioProcessRecordWithWorkingDir(t *testing.T, home, name, step string, pid, port int, startedAt time.Time, workingDir string) {
	t.Helper()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", name, step+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": ` + fmt.Sprintf("%d", pid) + `,
  "pgid": ` + fmt.Sprintf("%d", pid) + `,
  "process_id": "vrooli.develop.` + name + `.` + step + `",
  "phase": "develop",
  "scenario": "` + name + `",
  "step": "` + step + `",
  "command": "sleep 10",
  "working_dir": "` + workingDir + `",
  "log_file": "/tmp/` + name + `.log",
  "port": ` + fmt.Sprintf("%d", port) + `,
  "started_at": "` + startedAt.UTC().Format(time.RFC3339) + `",
  "status": "running"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
