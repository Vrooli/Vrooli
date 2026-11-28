package scenariohandlers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/process"
)

func TestScenarioLogHelperReaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alpha.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	tail, err := ReadLastLogLines(path, 2)
	if err != nil {
		t.Fatalf("ReadLastLogLines() error = %v", err)
	}
	if string(tail) != "two\nthree\n" {
		t.Fatalf("tail = %q", string(tail))
	}

	delta, nextOffset, err := ReadScenarioLogDelta(path, int64(len("one\n")))
	if err != nil {
		t.Fatalf("ReadScenarioLogDelta() error = %v", err)
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

	delta, _, err = ReadScenarioLogDelta(path, nextOffset)
	if err != nil {
		t.Fatalf("ReadScenarioLogDelta() appended error = %v", err)
	}
	if string(delta) != "four\n" {
		t.Fatalf("appended delta = %q", string(delta))
	}
}

func TestShowScenarioLifecycleLogHonorsTailOption(t *testing.T) {
	home := t.TempDir()
	path, err := process.ScenarioLifecycleLogPath(home, "alpha")
	if err != nil {
		t.Fatalf("ScenarioLifecycleLogPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir lifecycle log dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("write lifecycle log: %v", err)
	}

	var stdout bytes.Buffer
	if err := showScenarioLifecycleLog(t.TempDir(), home, "alpha", LogOptions{Tail: 1}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("showScenarioLifecycleLog: %v", err)
	}
	got := stdout.String()
	if strings.Contains(got, "two\n") || !strings.Contains(got, "three\n") {
		t.Fatalf("tail output = %q", got)
	}
}

// Disk-pressure recovery reclaims the regenerable logs root; the lifecycle
// log must then come from the run retained under the state root rather than
// "no lifecycle log found".
func TestShowScenarioLifecycleLogFallsBackToRetainedRun(t *testing.T) {
	home := t.TempDir()
	dir, err := lifecycle.LifecycleRunLogsDir(home, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for file, content := range map[string]string{
		"20261027-062405.774246632-111.restart.develop.log": "older run\n",
		"20261028-212514.976424189-222.restart.setup.log":   "setup of latest run\n",
		"20261028-212533.754640648-222.restart.develop.log": "develop of latest run\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout bytes.Buffer
	if err := showScenarioLifecycleLog(t.TempDir(), home, "alpha", LogOptions{}, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("showScenarioLifecycleLog: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "setup of latest run") || !strings.Contains(got, "develop of latest run") || strings.Contains(got, "older run") {
		t.Fatalf("retained fallback output = %q", got)
	}
	if err := showScenarioLifecycleLog(t.TempDir(), t.TempDir(), "alpha", LogOptions{}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "no lifecycle log found") {
		t.Fatalf("missing log and no retained run: err = %v", err)
	}
}

func TestShowScenarioRuntimeAndStepLogsHonorTailOption(t *testing.T) {
	home := t.TempDir()
	logsDir, err := process.ScenarioLogsDir(home, "alpha")
	if err != nil {
		t.Fatalf("ScenarioLogsDir: %v", err)
	}
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		t.Fatalf("mkdir runtime logs dir: %v", err)
	}
	path := filepath.Join(logsDir, "vrooli.develop.alpha.start-api.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("write runtime log: %v", err)
	}

	var runtimeOut bytes.Buffer
	if err := showScenarioRuntimeLogs(home, "alpha", LogOptions{Tail: 1}, &runtimeOut); err != nil {
		t.Fatalf("showScenarioRuntimeLogs: %v", err)
	}
	if got := runtimeOut.String(); strings.Contains(got, "two\n") || !strings.Contains(got, "three\n") {
		t.Fatalf("runtime tail output = %q", got)
	}

	var stepOut bytes.Buffer
	if err := showScenarioStepLog(home, "alpha", LogOptions{StepName: "start-api", Tail: 1}, &stepOut); err != nil {
		t.Fatalf("showScenarioStepLog: %v", err)
	}
	if got := stepOut.String(); strings.Contains(got, "two\n") || !strings.Contains(got, "three\n") {
		t.Fatalf("step tail output = %q", got)
	}
}
