package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
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
