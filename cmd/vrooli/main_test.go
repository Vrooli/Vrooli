package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestRunRoutesScenarioCommandToBashHandler(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	var captured commandSpec
	execCommandFn = func(spec commandSpec) error {
		captured = spec
		return nil
	}
	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }
	rebuildAndReexecFn = func(args []string) error {
		t.Fatalf("unexpected rebuild")
		return nil
	}

	code := run([]string{"--json", "scenario", "list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if captured.name != "bash" {
		t.Fatalf("command name = %q", captured.name)
	}
	wantArgs := []string{"/repo/cli/commands/scenario/scenario-commands.sh", "list", "--json"}
	if strings.Join(captured.args, "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("command args = %v, want %v", captured.args, wantArgs)
	}
	if captured.dir != "/repo" {
		t.Fatalf("command dir = %q", captured.dir)
	}
}

func TestRunForceBashBypassesStaleCheck(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	t.Setenv(forceBashEnvVar, "1")
	var captured commandSpec
	execCommandFn = func(spec commandSpec) error {
		captured = spec
		return nil
	}
	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool {
		t.Fatalf("stale check should be skipped when forcing Bash")
		return false
	}

	code := run([]string{"scenario", "list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	wantArgs := []string{"/repo/cli/vrooli", "scenario", "list"}
	if strings.Join(captured.args, "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("legacy bash args = %v, want %v", captured.args, wantArgs)
	}
}

func TestRunNoStaleCheckBypassesFreshnessProbe(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	var captured commandSpec
	execCommandFn = func(spec commandSpec) error {
		captured = spec
		return nil
	}
	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool {
		t.Fatalf("stale check should be skipped when --no-stale-check is set")
		return false
	}

	code := run([]string{"--no-stale-check", "scenario", "list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	wantArgs := []string{"/repo/cli/commands/scenario/scenario-commands.sh", "list"}
	if strings.Join(captured.args, "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("command args = %v, want %v", captured.args, wantArgs)
	}
}

func TestRunTriggersRebuildBeforeDispatch(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return true }

	var rebuiltArgs []string
	rebuildAndReexecFn = func(args []string) error {
		rebuiltArgs = append([]string(nil), args...)
		return nil
	}
	execCommandFn = func(spec commandSpec) error {
		t.Fatalf("dispatcher should not run when stale rebuild succeeds")
		return nil
	}

	code := run([]string{"scenario", "list"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.Join(rebuiltArgs, "|") != "scenario|list" {
		t.Fatalf("rebuilt args = %v", rebuiltArgs)
	}
}

func TestRunInfoCommandUsesManifestAndListMode(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":["docs/context.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"info", "--list"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if strings.TrimSpace(stdout.String()) != filepath.Join(root, "docs", "context.md") {
		t.Fatalf("info list output = %q", stdout.String())
	}
}

func TestRunReturnsExitCodeFromSubprocess(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }
	execCommandFn = func(spec commandSpec) error {
		return exitCodeError{code: 23}
	}

	code := run([]string{"status"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 23 {
		t.Fatalf("run exit code = %d, want 23", code)
	}
}

func TestRunAutohealErrorsWhenBinaryMissing(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }
	lookPathFn = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	var stderr bytes.Buffer
	code := run([]string{"orphans"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "vrooli setup") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func overrideCLIHooks(t *testing.T) func() {
	t.Helper()
	originalResolveSourceRootFn := resolveSourceRootFn
	originalIsStaleFn := isStaleFn
	originalRebuildAndReexecFn := rebuildAndReexecFn
	originalLookPathFn := lookPathFn
	originalExecCommandFn := execCommandFn
	return func() {
		resolveSourceRootFn = originalResolveSourceRootFn
		isStaleFn = originalIsStaleFn
		rebuildAndReexecFn = originalRebuildAndReexecFn
		lookPathFn = originalLookPathFn
		execCommandFn = originalExecCommandFn
	}
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
