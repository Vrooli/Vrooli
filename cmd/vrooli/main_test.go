package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
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

func TestRunRoutesNonMigratedScenarioCommandToBashHandler(t *testing.T) {
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

	code := run([]string{"--json", "scenario", "start", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if captured.name != "bash" {
		t.Fatalf("command name = %q", captured.name)
	}
	wantArgs := []string{"/repo/cli/commands/scenario/scenario-commands.sh", "start", "alpha", "--json"}
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

	code := run([]string{"--no-stale-check", "scenario", "start", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	wantArgs := []string{"/repo/cli/commands/scenario/scenario-commands.sh", "start", "alpha"}
	if strings.Join(captured.args, "|") != strings.Join(wantArgs, "|") {
		t.Fatalf("command args = %v, want %v", captured.args, wantArgs)
	}
}

func TestRunScenarioListJSONOutput(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "_artifacts"), 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }
	execCommandFn = func(spec commandSpec) error {
		t.Fatalf("scenario list should not shell to bash")
		return nil
	}

	var stdout bytes.Buffer
	code := run([]string{"scenario", "list", "--json"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario", "info", "alpha", "--json"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	t.Setenv("VROOLI_API_PORT", "1")
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario", "status", "alpha", "--json"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }
	execCommandFn = func(spec commandSpec) error {
		t.Fatalf("scenario list should not shell to bash")
		return nil
	}

	var stdout bytes.Buffer
	code := run([]string{"scenario", "list", "--include-ports"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario", "status", "alpha"}, &stdout, &bytes.Buffer{})
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

func TestRunScenarioHelpListsMigratedCommands(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	output := stdout.String()
	if !strings.Contains(output, "info <name> [--json]") {
		t.Fatalf("missing info help line: %s", output)
	}
	if !strings.Contains(output, "status [name] [--json]") {
		t.Fatalf("missing status help line: %s", output)
	}
}

func TestRunScenarioInfoRequiresName(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }

	var stderr bytes.Buffer
	code := run([]string{"scenario", "info"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "requires a scenario name") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunScenarioListRejectsUnknownFlag(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }

	var stderr bytes.Buffer
	code := run([]string{"scenario", "list", "--bogus"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown option for scenario list") {
		t.Fatalf("stderr = %q", stderr.String())
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
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"--json", "--version"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	tests := []struct {
		name     string
		args     []string
		wantName string
		wantArgs []string
	}{
		{
			name:     "setup uses manage script",
			args:     []string{"setup"},
			wantName: "bash",
			wantArgs: []string{"/repo/scripts/manage.sh", "setup"},
		},
		{
			name:     "clean uses clean commands script",
			args:     []string{"clean"},
			wantName: "bash",
			wantArgs: []string{"/repo/cli/commands/clean-commands.sh"},
		},
		{
			name:     "resource uses resource script",
			args:     []string{"resource", "status"},
			wantName: "bash",
			wantArgs: []string{"/repo/cli/commands/resource-commands.sh", "status"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var captured commandSpec
			execCommandFn = func(spec commandSpec) error {
				captured = spec
				return nil
			}
			resolveSourceRootFn = func() (string, error) { return "/repo", nil }
			isStaleFn = func() bool { return false }

			code := run(tc.args, &bytes.Buffer{}, &bytes.Buffer{})
			if code != 0 {
				t.Fatalf("run exit code = %d", code)
			}
			if captured.name != tc.wantName {
				t.Fatalf("command name = %q, want %q", captured.name, tc.wantName)
			}
			if strings.Join(captured.args, "|") != strings.Join(tc.wantArgs, "|") {
				t.Fatalf("command args = %v, want %v", captured.args, tc.wantArgs)
			}
		})
	}
}

func TestRunCleanupLocksUsesAutohealBinary(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	var captured commandSpec
	execCommandFn = func(spec commandSpec) error {
		captured = spec
		return nil
	}
	lookPathFn = func(file string) (string, error) {
		if file != "vrooli-autoheal" {
			t.Fatalf("lookPath requested %q", file)
		}
		return "/usr/local/bin/vrooli-autoheal", nil
	}
	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }

	code := run([]string{"--json", "cleanup", "locks"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if captured.name != "/usr/local/bin/vrooli-autoheal" {
		t.Fatalf("command name = %q", captured.name)
	}
	if strings.Join(captured.args, "|") != "locks|clean|--json" {
		t.Fatalf("command args = %v", captured.args)
	}
}

func TestRunInfoListJSONOutput(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":["docs/context.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"--json", "info", "--list"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":["docs/context.md","docs/missing.md"]}`)
	writeTestFile(t, root, "docs/context.md", "hello world\n")
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--json", "info"}, &stdout, &stderr)
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestScenarioService(t, root, "beta", "Beta scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario", "status", "--json"}, &stdout, &bytes.Buffer{})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestScenarioService(t, root, "beta", "Beta scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario", "status"}, &stdout, &bytes.Buffer{})
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
	if !strings.Contains(installStep, "cli/install.sh") {
		t.Fatalf("install-cli step does not invoke cli/install.sh: %q", installStep)
	}

	installScript, err := os.ReadFile(filepath.Join(repoRoot, "cli", "install.sh"))
	if err != nil {
		t.Fatalf("read cli/install.sh: %v", err)
	}
	installContents := string(installScript)
	if !strings.Contains(installContents, "make install") {
		t.Fatalf("cli/install.sh no longer invokes make install")
	}
	if !strings.Contains(installContents, "${HOME}/.vrooli/bin") {
		t.Fatalf("cli/install.sh no longer targets ~/.vrooli/bin")
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

func TestRunCleanupCommandRoutesTargets(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/vrooli-autoheal", nil
	}

	t.Run("orphans", func(t *testing.T) {
		var captured commandSpec
		execCommandFn = func(spec commandSpec) error {
			captured = spec
			return nil
		}

		err := runCleanupCommand("/repo", parsedArgs{
			args: []string{"orphans"},
			globals: globalOptions{
				json:    true,
				verbose: true,
			},
		}, &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("runCleanupCommand: %v", err)
		}
		want := []string{"orphans", "kill", "--json", "--verbose"}
		if strings.Join(captured.args, "|") != strings.Join(want, "|") {
			t.Fatalf("args = %v, want %v", captured.args, want)
		}
	})

	t.Run("locks", func(t *testing.T) {
		var captured commandSpec
		execCommandFn = func(spec commandSpec) error {
			captured = spec
			return nil
		}

		err := runCleanupCommand("/repo", parsedArgs{
			args: []string{"locks"},
			globals: globalOptions{
				noColor: true,
			},
		}, &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil {
			t.Fatalf("runCleanupCommand: %v", err)
		}
		want := []string{"locks", "clean", "--no-color"}
		if strings.Join(captured.args, "|") != strings.Join(want, "|") {
			t.Fatalf("args = %v, want %v", captured.args, want)
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

	var stderr bytes.Buffer
	err := runCleanupCommand("/repo", parsedArgs{args: []string{"bogus"}}, &bytes.Buffer{}, &stderr)
	if exitCode(err) != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode(err))
	}
	if !strings.Contains(stderr.String(), "Unknown cleanup target") {
		t.Fatalf("stderr = %q", stderr.String())
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

	files, err := collectInfoSources(root)
	if err != nil {
		t.Fatalf("collectInfoSources env: %v", err)
	}
	if got, want := strings.Join(files, ","), "docs/context.md,/tmp/extra.md"; got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}

	t.Setenv("VROOLI_INFO_FILES", "")
	writeTestFile(t, root, ".vrooli/info-manifest.json", `{"files":`)

	files, err = collectInfoSources(root)
	if err != nil {
		t.Fatalf("collectInfoSources fallback: %v", err)
	}
	if got, want := strings.Join(files, ","), strings.Join(infoDefaultFiles, ","); got != want {
		t.Fatalf("files = %q, want %q", got, want)
	}
}

func TestRunUnknownCommandSuggestsNearestMatch(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "/repo", nil }
	isStaleFn = func() bool { return false }

	var stderr bytes.Buffer
	code := run([]string{"statuz"}, &bytes.Buffer{}, &stderr)
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
	if !strings.Contains(help.String(), "scenario list") {
		t.Fatalf("help output = %q", help.String())
	}
}

func TestRunReturnsErrorWhenRootCannotBeResolved(t *testing.T) {
	restore := overrideCLIHooks(t)
	defer restore()

	resolveSourceRootFn = func() (string, error) { return "", errors.New("boom") }

	var stderr bytes.Buffer
	code := run([]string{"version"}, &bytes.Buffer{}, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "resolve Vrooli root") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCommandEnvPreservesExistingSourceRootAndNoColor(t *testing.T) {
	t.Setenv("HOME", "/tmp/home")
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("LANG", "C")
	t.Setenv("VROOLI_SOURCE_ROOT", "/custom/source")

	env := commandEnv("/repo", globalOptions{noColor: true})
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
	restore := overrideCLIHooks(t)
	defer restore()

	root := t.TempDir()
	home := t.TempDir()
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")
	writeTestScenarioService(t, root, "beta", "Beta scenario")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))

	t.Setenv("HOME", home)
	resolveSourceRootFn = func() (string, error) { return root, nil }
	isStaleFn = func() bool { return false }

	var stdout bytes.Buffer
	code := run([]string{"scenario", "status", "--json"}, &stdout, &bytes.Buffer{})
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
