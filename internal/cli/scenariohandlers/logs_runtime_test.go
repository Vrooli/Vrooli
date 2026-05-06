package scenariohandlers

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/vrooli/vrooli/internal/cli/scenariocli"
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
	path := process.ScenarioLifecycleLogPath(home, "alpha")
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

func TestShowScenarioRuntimeAndStepLogsHonorTailOption(t *testing.T) {
	home := t.TempDir()
	logsDir := process.ScenarioLogsDir(home, "alpha")
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
