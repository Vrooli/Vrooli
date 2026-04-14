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

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestRunScenarioTestUsesNativePhaseRunner(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioTestPhaseFixture(t, root, "alpha")
	writeScenarioPortRegistryFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.RebuildAndReexecFn = func(args []string) error {
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

func TestRunNoStaleCheckBypassesFreshnessProbe(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioSetupOnlyFixture(t, root, "alpha")
	writeScenarioPortRegistryFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	app.CheckStalenessFn = func() (buildinfo.StaleCheck, error) {
		t.Fatalf("stale check should be skipped when --no-stale-check is set")
		return buildinfo.StaleCheck{}, nil
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
	port := testkitgo.ReserveFreePort(t)
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

func TestRunScenarioPortAndOpenCommandsUseNativeState(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioServiceWithPorts(t, root, "alpha")
	writeScenarioProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-time.Minute))
	writeScenarioProcessRecord(t, home, "alpha", "start-ui", os.Getpid(), 38080, time.Now().Add(-time.Minute))

	t.Setenv("HOME", home)
	app := newTestApp(root)
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

func TestRunScenarioStartOpenUsesNativeURLLauncher(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("lifecycle process management currently targets linux")
	}

	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleScenarioService(t, root, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var opened scenarioSubprocessSpec
	app.LookPathFn = func(file string) (string, error) {
		if file == "xdg-open" {
			return "/usr/bin/xdg-open", nil
		}
		return "", exec.ErrNotFound
	}
	app.RunScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
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
	if opened.Name != "/usr/bin/xdg-open" || len(opened.Args) != 1 || !strings.HasPrefix(opened.Args[0], "http://localhost:") {
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

	var opened scenarioSubprocessSpec
	app.LookPathFn = func(file string) (string, error) {
		if file == "xdg-open" {
			return "/usr/bin/xdg-open", nil
		}
		return "", exec.ErrNotFound
	}
	app.RunScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
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
	if opened.Name != "/usr/bin/xdg-open" || len(opened.Args) != 1 || !strings.HasPrefix(opened.Args[0], "http://localhost:") {
		t.Fatalf("opened = %+v", opened)
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
	app.ScenarioExecutableFn = func() (string, error) {
		return writeFakeExecutable(t, root, "bin/fake-vrooli", fmt.Sprintf("#!/usr/bin/env bash\nprintf '%%s\\n' \"$@\" >> %q\n", relaunchLog)), nil
	}

	if code := app.Run([]string{"scenario", "heal-from-sandbox", "--merged-path", "/merged"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario heal-from-sandbox exit code = %d", code)
	}
	testkitgo.WaitForFile(t, relaunchLog)
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
	alphaPort := testkitgo.ReserveFreePort(t)
	betaPort := testkitgo.ReserveFreePort(t)
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
		testkitgo.WaitForFile(t, filepath.Join(home, ".vrooli", "processes", "scenarios", name, "start-api.json"))
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
	app.RunScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
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
	if captured.Name != "bash" {
		t.Fatalf("hook subprocess = %+v", captured)
	}
	if strings.Join(captured.Args, "|") != "-lc|echo hook-ran" {
		t.Fatalf("hook args = %v", captured.Args)
	}
	if captured.Dir != filepath.Join(root, "scenarios", "alpha") {
		t.Fatalf("hook dir = %q", captured.Dir)
	}
	if !strings.Contains(generateStdout.String(), "[Hook 1] Echo hook") {
		t.Fatalf("generate output = %q", generateStdout.String())
	}
}

func TestRunScenarioRunAliasesStartValidation(t *testing.T) {
	_, ctx := newConfiguredCommandContext("/repo", globalOptions{}, &bytes.Buffer{}, &bytes.Buffer{})
	handler, ok := configuredApp().Registry().ScenarioHandler(string(scenariocli.CommandRun))
	if !ok {
		t.Fatal("missing scenario run handler")
	}
	err := handler(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "scenario start requires at least one scenario name") {
		t.Fatalf("scenario run handler error = %v", err)
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

func TestRunScenarioStartEnsuresScenarioCLI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeLifecycleScenarioService(t, root, "alpha")

	t.Setenv("HOME", home)
	app := newTestApp(root)
	var ensured []string
	app.EnsureScenarioCLIFn = func(rootArg, homeArg, name string) error {
		if rootArg != root || homeArg != home {
			t.Fatalf("ensure args = (%q, %q), want (%q, %q)", rootArg, homeArg, root, home)
		}
		ensured = append(ensured, name)
		return errors.New("ensure scenario cli")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "start", "alpha"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if got := strings.Join(ensured, "|"); got != "alpha" {
		t.Fatalf("ensured = %q", got)
	}
	if !strings.Contains(stderr.String(), "ensure scenario cli") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunScenarioTestEnsuresScenarioCLI(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioTestPhaseFixture(t, root, "alpha")
	writeScenarioPortRegistryFixture(t, root)

	t.Setenv("HOME", home)
	app := newTestApp(root)
	var ensured []string
	app.EnsureScenarioCLIFn = func(rootArg, homeArg, name string) error {
		if rootArg != root || homeArg != home {
			t.Fatalf("ensure args = (%q, %q), want (%q, %q)", rootArg, homeArg, root, home)
		}
		ensured = append(ensured, name)
		return errors.New("ensure scenario cli")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := app.Run([]string{"scenario", "test", "alpha"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run exit code = %d", code)
	}
	if got := strings.Join(ensured, "|"); got != "alpha" {
		t.Fatalf("ensured = %q", got)
	}
	if !strings.Contains(stderr.String(), "ensure scenario cli") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
