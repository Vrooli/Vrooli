package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReadAndSummarizeScenarioRecords(t *testing.T) {
	home := t.TempDir()
	writeProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))
	writeProcessRecord(t, home, "alpha", "start-ui", 999999, 38080, time.Now().Add(-1*time.Minute))

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

func TestDiscoverRunningScenariosFiltersStopped(t *testing.T) {
	home := t.TempDir()
	writeProcessRecord(t, home, "alpha", "start-api", os.Getpid(), 18080, time.Now().Add(-2*time.Minute))
	writeProcessRecord(t, home, "beta", "start-api", 999999, 18081, time.Now().Add(-1*time.Minute))

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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": ` + strconv.Itoa(os.Getpid()) + `,
  "started_at": "` + time.Now().UTC().Format(time.RFC3339) + `"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

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

func TestDiscoverRunningScenariosReturnsNilWhenProcessRootMissing(t *testing.T) {
	runtimes, err := DiscoverRunningScenarios(t.TempDir(), nil)
	if err != nil {
		t.Fatalf("DiscoverRunningScenarios: %v", err)
	}
	if len(runtimes) != 0 {
		t.Fatalf("runtime count = %d, want 0", len(runtimes))
	}
}

func TestReadEnvironmentPortsFromLiveProcess(t *testing.T) {
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

func writeProcessRecord(t *testing.T, home, scenarioName, step string, pid, port int, startedAt time.Time) {
	t.Helper()
	path := filepath.Join(home, ".vrooli", "processes", "scenarios", scenarioName, step+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	data := `{
  "pid": ` + strconv.Itoa(pid) + `,
  "pgid": ` + strconv.Itoa(pid) + `,
  "phase": "develop",
  "scenario": "` + scenarioName + `",
  "step": "` + step + `",
  "command": "sleep 10",
  "working_dir": "/repo/scenarios/` + scenarioName + `",
  "log_file": "/tmp/` + scenarioName + `.log",
  "port": ` + strconv.Itoa(port) + `,
  "started_at": "` + startedAt.UTC().Format(time.RFC3339) + `",
  "status": "running"
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
