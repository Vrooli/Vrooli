package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func withIsPIDRunningFn(t *testing.T, fn func(int) bool) {
	t.Helper()
	previous := isPIDRunningFn
	isPIDRunningFn = fn
	t.Cleanup(func() {
		isPIDRunningFn = previous
	})
}

func withReadProcessEnvironmentFn(t *testing.T, fn func(int) (map[string]string, error)) {
	t.Helper()
	previous := readProcessEnvironmentFn
	readProcessEnvironmentFn = fn
	t.Cleanup(func() {
		readProcessEnvironmentFn = previous
	})
}

func TestHomeDirPrefersEnv(t *testing.T) {
	t.Setenv("HOME", "/tmp/vrooli-home")

	home, err := HomeDir()
	if err != nil {
		t.Fatalf("HomeDir: %v", err)
	}
	if home != "/tmp/vrooli-home" {
		t.Fatalf("home = %q, want /tmp/vrooli-home", home)
	}
}

func TestReadAndSummarizeScenarioRecords(t *testing.T) {
	home := t.TempDir()
	if err := WriteScenarioRecord(home, "alpha", "start-api", Record{
		PID:        os.Getpid(),
		PGID:       os.Getpid(),
		Phase:      "develop",
		Scenario:   "alpha",
		Step:       "start-api",
		Command:    "sleep 10",
		WorkingDir: "/repo/scenarios/alpha",
		LogFile:    "/tmp/alpha.log",
		Port:       18080,
		StartedAt:  time.Now().Add(-2 * time.Minute),
		Status:     "running",
	}); err != nil {
		t.Fatalf("WriteScenarioRecord alpha/start-api: %v", err)
	}
	if err := WriteScenarioRecord(home, "alpha", "start-ui", Record{
		PID:        999999,
		PGID:       999999,
		Phase:      "develop",
		Scenario:   "alpha",
		Step:       "start-ui",
		Command:    "sleep 10",
		WorkingDir: "/repo/scenarios/alpha",
		LogFile:    "/tmp/alpha.log",
		Port:       38080,
		StartedAt:  time.Now().Add(-1 * time.Minute),
		Status:     "running",
	}); err != nil {
		t.Fatalf("WriteScenarioRecord alpha/start-ui: %v", err)
	}
	withIsPIDRunningFn(t, func(pid int) bool {
		return pid == os.Getpid()
	})

	records, err := ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("record count = %d, want 2", len(records))
	}

	live := LiveRecords(records)
	if len(live) != 1 {
		t.Fatalf("live count = %d, want 1", len(live))
	}
	if live[0].Step != "start-api" {
		t.Fatalf("live step = %q", live[0].Step)
	}

	runtime := SummarizeScenario("alpha", records)
	if runtime.ProcessCount != 1 {
		t.Fatalf("process count = %d, want 1", runtime.ProcessCount)
	}
	if runtime.StartedAt == nil {
		t.Fatalf("expected started_at")
	}
	if !strings.HasSuffix(runtime.Runtime, "m") && !strings.HasSuffix(runtime.Runtime, "h") {
		t.Fatalf("runtime = %q", runtime.Runtime)
	}
}

func TestHomeDirFallsBackToUserHomeWhenHOMEUnset(t *testing.T) {
	t.Setenv("HOME", "")

	got, err := HomeDir()
	want, err := os.UserHomeDir()
	if err != nil {
		if got != "" {
			t.Fatalf("HomeDir returned value %q alongside error %v", got, err)
		}
		if _, homeErr := HomeDir(); homeErr == nil {
			t.Fatalf("expected HomeDir to mirror os.UserHomeDir failure")
		}
		return
	}
	if got != want {
		t.Fatalf("HomeDir = %q, want %q", got, want)
	}
}

func TestDiscoverRunningScenariosFiltersStopped(t *testing.T) {
	home := t.TempDir()
	if err := WriteScenarioRecord(home, "alpha", "start-api", Record{
		PID:        os.Getpid(),
		PGID:       os.Getpid(),
		Phase:      "develop",
		Scenario:   "alpha",
		Step:       "start-api",
		Command:    "sleep 10",
		WorkingDir: "/repo/scenarios/alpha",
		LogFile:    "/tmp/alpha.log",
		Port:       18080,
		StartedAt:  time.Now().Add(-2 * time.Minute),
		Status:     "running",
	}); err != nil {
		t.Fatalf("WriteScenarioRecord alpha/start-api: %v", err)
	}
	if err := WriteScenarioRecord(home, "beta", "start-api", Record{
		PID:        999999,
		PGID:       999999,
		Phase:      "develop",
		Scenario:   "beta",
		Step:       "start-api",
		Command:    "sleep 10",
		WorkingDir: "/repo/scenarios/beta",
		LogFile:    "/tmp/beta.log",
		Port:       18081,
		StartedAt:  time.Now().Add(-1 * time.Minute),
		Status:     "running",
	}); err != nil {
		t.Fatalf("WriteScenarioRecord beta/start-api: %v", err)
	}
	withIsPIDRunningFn(t, func(pid int) bool {
		return pid == os.Getpid()
	})

	runtimes, err := DiscoverRunningScenarios(home, func(name string) bool { return name != "skip-me" })
	if err != nil {
		t.Fatalf("DiscoverRunningScenarios: %v", err)
	}
	if len(runtimes) != 1 {
		t.Fatalf("runtime count = %d, want 1", len(runtimes))
	}
	if runtimes[0].Name != "alpha" {
		t.Fatalf("runtime name = %q", runtimes[0].Name)
	}
}

func TestReadScenarioRecordsBackfillsStepAndScenario(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "custom-step.json")
	testkitgo.WriteJSON(t, path, map[string]any{
		"pid":        os.Getpid(),
		"started_at": time.Now().UTC().Format(time.RFC3339),
	})

	records, err := ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].Scenario != "alpha" {
		t.Fatalf("scenario = %q", records[0].Scenario)
	}
	if records[0].Step != "custom-step" {
		t.Fatalf("step = %q", records[0].Step)
	}
}

func TestReadScenarioRecordsRejectsInvalidJSON(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "broken.json")
	testkitgo.WriteMalformedJSON(t, path, "{broken", 0o644)

	if _, err := ReadScenarioRecords(home, "alpha"); err == nil {
		t.Fatalf("expected invalid process metadata to fail")
	}
}

func TestDiscoverRunningScenariosReturnsNilWhenProcessRootMissing(t *testing.T) {
	runtimes, err := DiscoverRunningScenarios(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("DiscoverRunningScenarios: %v", err)
	}
	if len(runtimes) != 0 {
		t.Fatalf("runtime count = %d, want 0", len(runtimes))
	}
}

func TestDiscoverRunningScenariosPropagatesReadErrors(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "broken.json")
	testkitgo.WriteMalformedJSON(t, path, "{broken", 0o644)

	if _, err := DiscoverRunningScenarios(home, nil); err == nil {
		t.Fatalf("expected invalid process metadata to fail discovery")
	}
}

func TestLiveRecordsSortsByStepThenPID(t *testing.T) {
	withIsPIDRunningFn(t, func(pid int) bool {
		return pid > 0
	})

	records := LiveRecords([]Record{
		{PID: 42, Step: "start-ui"},
		{PID: 7, Step: "start-api"},
	})
	if len(records) != 2 {
		t.Fatalf("live record count = %d, want 2", len(records))
	}
	if records[0].Step != "start-api" || records[1].Step != "start-ui" {
		t.Fatalf("live records not sorted by step: %#v", records)
	}
}

func TestReadEnvironmentPortsUsesInjectedEnvironmentReader(t *testing.T) {
	withReadProcessEnvironmentFn(t, func(pid int) (map[string]string, error) {
		switch pid {
		case 11:
			return map[string]string{
				"API_PORT": "18080",
				"UI_PORT":  "38080",
			}, nil
		case 99:
			return map[string]string{
				"API_PORT": "18081",
			}, nil
		default:
			return nil, os.ErrNotExist
		}
	})

	ports := ReadEnvironmentPorts([]Record{{PID: 11}, {PID: 99}}, []string{"API_PORT", "UI_PORT", "WS_PORT"})

	if ports["API_PORT"] != 18080 {
		t.Fatalf("API_PORT = %d, want 18080", ports["API_PORT"])
	}
	if ports["UI_PORT"] != 38080 {
		t.Fatalf("UI_PORT = %d, want 38080", ports["UI_PORT"])
	}
	if _, exists := ports["WS_PORT"]; exists {
		t.Fatalf("unexpected WS_PORT entry: %#v", ports)
	}
}

func TestReadEnvironmentPortsIgnoresInvalidValuesAndEmptyKeys(t *testing.T) {
	withReadProcessEnvironmentFn(t, func(pid int) (map[string]string, error) {
		return map[string]string{
			"API_PORT": "not-a-number",
			"UI_PORT":  "38080",
		}, nil
	})

	ports := ReadEnvironmentPorts([]Record{{PID: 11}}, []string{"", "API_PORT", "UI_PORT"})

	if _, exists := ports["API_PORT"]; exists {
		t.Fatalf("expected invalid numeric value to be ignored, got %#v", ports)
	}
	if ports["UI_PORT"] != 38080 {
		t.Fatalf("UI_PORT = %d, want 38080", ports["UI_PORT"])
	}
}

func TestReadEnvironmentPortsFromLiveProcessSmoke(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process environment inspection uses /proc on linux")
	}

	cmd := exec.Command("sleep", "30")
	cmd.Env = append(os.Environ(), "API_PORT=18080", "UI_PORT=38080")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	var ports map[string]int
	for attempt := 0; attempt < 20; attempt++ {
		ports = ReadEnvironmentPorts([]Record{{PID: cmd.Process.Pid}}, []string{"API_PORT", "UI_PORT", "WS_PORT"})
		if ports["API_PORT"] == 18080 && ports["UI_PORT"] == 38080 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if ports["API_PORT"] != 18080 {
		t.Fatalf("API_PORT = %d, want 18080", ports["API_PORT"])
	}
	if ports["UI_PORT"] != 38080 {
		t.Fatalf("UI_PORT = %d, want 38080", ports["UI_PORT"])
	}
	if _, exists := ports["WS_PORT"]; exists {
		t.Fatalf("unexpected WS_PORT entry: %#v", ports)
	}
}

func TestIsPIDRunningRejectsInvalidPID(t *testing.T) {
	if IsPIDRunning(-1) {
		t.Fatalf("expected invalid pid to be treated as not running")
	}
}

func TestHumanDurationHandlesNegativeAndDayScale(t *testing.T) {
	if got := humanDuration(-5 * time.Minute); got != "0m" {
		t.Fatalf("humanDuration negative = %q, want 0m", got)
	}
	if got := humanDuration(49 * time.Hour); got != "2.0d" {
		t.Fatalf("humanDuration day scale = %q, want 2.0d", got)
	}
}

func TestSummarizeScenarioWithoutTimestampsKeepsRuntimeUnknown(t *testing.T) {
	runtimeState := SummarizeScenario("alpha", []Record{{PID: os.Getpid(), Step: "start-api"}})
	if runtimeState.ProcessCount != 1 {
		t.Fatalf("process count = %d, want 1", runtimeState.ProcessCount)
	}
	if runtimeState.StartedAt != nil {
		t.Fatalf("expected nil started_at for zero timestamps, got %v", runtimeState.StartedAt)
	}
	if runtimeState.Runtime != "N/A" {
		t.Fatalf("runtime = %q, want N/A", runtimeState.Runtime)
	}
}

func TestWriteAndRemoveScenarioRecordMaintainsProcessMetadataContract(t *testing.T) {
	home := t.TempDir()
	record := Record{
		PID:       os.Getpid(),
		Phase:     "develop",
		Command:   "sleep 10",
		LogFile:   "/tmp/alpha.log",
		StartedAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := WriteScenarioRecord(home, "alpha", "start-api", record); err != nil {
		t.Fatalf("WriteScenarioRecord: %v", err)
	}

	processDir := ScenarioProcessDir(home, "alpha")
	if processDir != filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha") {
		t.Fatalf("ScenarioProcessDir = %q", processDir)
	}
	if ScenarioLogsDir(home, "alpha") != filepath.Join(home, ".vrooli", "logs", "scenarios", "alpha") {
		t.Fatalf("ScenarioLogsDir mismatch")
	}
	if ScenarioLifecycleLogPath(home, "alpha") != filepath.Join(home, ".vrooli", "logs", "alpha.log") {
		t.Fatalf("ScenarioLifecycleLogPath mismatch")
	}
	if ScenarioStateDir(home) != filepath.Join(home, ".vrooli", "state", "scenarios") {
		t.Fatalf("ScenarioStateDir mismatch")
	}
	if ScenarioDegradedPath(home, "alpha") != filepath.Join(home, ".vrooli", "processes", "scenarios", "alpha", "degraded.json") {
		t.Fatalf("ScenarioDegradedPath mismatch")
	}

	recordPath := filepath.Join(processDir, "start-api.json")
	pidPath := filepath.Join(processDir, "start-api.pid")
	if _, err := os.Stat(recordPath); err != nil {
		t.Fatalf("expected record file: %v", err)
	}
	if _, err := os.Stat(pidPath); err != nil {
		t.Fatalf("expected pid file: %v", err)
	}

	records, err := ReadScenarioRecords(home, "alpha")
	if err != nil {
		t.Fatalf("ReadScenarioRecords: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want 1", len(records))
	}
	if records[0].Step != "start-api" || records[0].Scenario != "alpha" {
		t.Fatalf("record metadata = %#v", records[0])
	}

	if err := RemoveScenarioRecord(home, "alpha", "start-api"); err != nil {
		t.Fatalf("RemoveScenarioRecord: %v", err)
	}
	if err := RemoveScenarioRecord(home, "alpha", "start-api"); err != nil {
		t.Fatalf("RemoveScenarioRecord should be idempotent: %v", err)
	}
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("expected record file removal, stat err=%v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("expected pid file removal, stat err=%v", err)
	}
}
