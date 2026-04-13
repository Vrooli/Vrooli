//go:build integration
// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/config"
)

func TestSplitRunScenarioTemplateGenerateScaffoldsFiles(t *testing.T) {
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

func TestSplitRunScenarioRequirementsSnapshotReadsLatestFile(t *testing.T) {
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

func TestSplitRunScenarioHealFromSandboxRelaunchesAffectedScenarios(t *testing.T) {
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

func TestSplitRunScenarioStartAllAndStopAllUseNativeLifecycle(t *testing.T) {
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

func TestSplitRunScenarioUISmokeUsesTranslatedSubprocess(t *testing.T) {
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

func TestSplitRunScenarioLogsHelpAndViews(t *testing.T) {
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

func TestSplitRunScenarioRequirementsReportUsesTranslatedSubprocess(t *testing.T) {
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

func TestSplitRunScenarioCompletenessUsesTranslatedSubprocess(t *testing.T) {
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
	if got := strings.Join(captured.args, "|"); got != "alpha|--format|json" {
		t.Fatalf("subprocess args = %q", got)
	}
}

func TestSplitRunScenarioRequirementsHelpAndInitTranslation(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	fakeCLI := writeFakeExecutable(t, home, ".vrooli/bin/test-genie", "#!/usr/bin/env bash\nexit 0\n")
	writeTestScenarioService(t, root, "alpha", "Alpha scenario")

	t.Setenv("HOME", home)
	app := newTestApp(root)

	var stdout bytes.Buffer
	if code := app.Run([]string{"scenario", "requirements", "--help"}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario requirements --help exit code = %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage: vrooli scenario requirements") {
		t.Fatalf("help output = %q", stdout.String())
	}

	var captured scenarioSubprocessSpec
	app.runScenarioSubprocess = func(spec scenarioSubprocessSpec) error {
		captured = spec
		return nil
	}

	if code := app.Run([]string{"scenario", "requirements", "init", "alpha"}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Fatalf("scenario requirements init exit code = %d", code)
	}
	if captured.name != fakeCLI {
		t.Fatalf("subprocess name = %q", captured.name)
	}
	if got := strings.Join(captured.args, "|"); !strings.Contains(got, "requirements|init") || !strings.Contains(got, "--dir|"+filepath.Join(root, "scenarios", "alpha")) {
		t.Fatalf("subprocess args = %q", got)
	}
}

func TestSplitRunScenarioRunAliasesStartValidation(t *testing.T) {
	err := runScenarioRunCommandForRoot("/repo", globalOptions{}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "scenario start requires at least one scenario name") {
		t.Fatalf("unexpected error: %v", err)
	}
}
