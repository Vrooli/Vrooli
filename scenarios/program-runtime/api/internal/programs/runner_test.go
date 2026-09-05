package programs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeTestKernel(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kernel.py")
	source := `import json
import sys
import time

for line in sys.stdin:
    request = json.loads(line)
    if "while True" in request.get("source", ""):
        while True:
            time.sleep(0.01)
    if "progress" in request.get("source", ""):
        print(json.dumps({"type": "progress", "ok": True, "stdout": "partial\n", "context_bytes": 8, "agent_bytes": 8, "output_limit_bytes": 4096}), flush=True)
    print(json.dumps({"ok": True, "stdout": "2\n", "context_bytes": 1, "output_limit_bytes": 4096, "invocations": []}), flush=True)
`
	if err := os.WriteFile(path, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSubprocessRunnerPersistsProgressCallback(t *testing.T) {
	runner := NewSubprocessRunner(writeTestKernel(t))
	defer runner.Close()
	var updates []Result
	result, err := runner.ExecuteWithMetadataAndLimitsAndProgress(context.Background(), "sess_progress", "prog_progress", "agent", "progress", false, ExecutionLimits{Wall: time.Second}, func(update Result) {
		updates = append(updates, update)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].Stdout != "partial\n" {
		t.Fatalf("progress updates=%+v", updates)
	}
	if result.Stdout != "2\n" {
		t.Fatalf("terminal result=%+v", result)
	}
}

func TestSubprocessRunnerReturnsTypedDeadlineAndRespawns(t *testing.T) { // [REQ:PRT-P1-005]
	runner := NewSubprocessRunner(writeTestKernel(t))
	defer runner.Close()

	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Millisecond)
	defer cancel()
	_, err := runner.Execute(ctx, "sess_deadline", "while True: pass", false)
	if time.Since(started) > 5*time.Second {
		t.Fatalf("deadline did not bound execution: %s", time.Since(started))
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v, want context deadline", err)
	}
	if _, alive := runner.MemoryBytes("sess_deadline"); alive {
		t.Fatal("killed kernel remained in process map")
	}

	result, err := runner.Execute(context.Background(), "sess_deadline", "print(1 + 1)", false)
	if err != nil || result.Stdout != "2\n" {
		t.Fatalf("respawn result=%+v err=%v", result, err)
	}
}

func TestSubprocessRunnerSerializesConcurrentSessionSubmissions(t *testing.T) {
	runner := NewSubprocessRunner(writeTestKernel(t))
	defer runner.Close()
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := runner.Execute(context.Background(), "sess_serial", "print(1 + 1)", false)
			results <- err
		}()
	}
	if err := <-results; err != nil {
		t.Fatal(err)
	}
	if err := <-results; err != nil {
		t.Fatal(err)
	}
}

func TestSubprocessRunnerUsesAllowlistedEnvironmentAndPinnedScratchDir(t *testing.T) { // [REQ:PRT-P1-004]
	dataDir := t.TempDir()
	t.Setenv("SCENARIO_DATA_DIR", dataDir)
	t.Setenv("VROOLI_EVENTS_API_BASE", "https://secret.invalid")
	t.Setenv("PROGRAM_RUNTIME_SECRET", "do-not-leak")
	hostPythonPath := filepath.Join(t.TempDir(), "host-python-path")
	t.Setenv("PYTHONPATH", hostPythonPath)
	path := filepath.Join(t.TempDir(), "probe.py")
	if err := os.WriteFile(path, []byte(`import json, os, sys
for line in sys.stdin:
    source = json.loads(line).get("source", "")
    if source == "env": value = str(os.environ.get("VROOLI_EVENTS_API_BASE"))
    elif source == "secret": value = str(os.environ.get("PROGRAM_RUNTIME_SECRET"))
    elif source == "python": value = sys.executable
    elif source == "sys_path": value = os.pathsep.join(sys.path)
    else: value = os.getcwd()
    print(json.dumps({"ok": True, "stdout": value, "context_bytes": len(value)}), flush=True)
`), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	defer runner.Close()
	for source, want := range map[string]string{"env": "None", "secret": "None"} {
		result, err := runner.Execute(context.Background(), "sess_hardening", source, false)
		if err != nil || result.Stdout != want {
			t.Fatalf("source=%s stdout=%q err=%v", source, result.Stdout, err)
		}
	}
	result, err := runner.Execute(context.Background(), "sess_hardening", "python", false)
	expectedPython := filepath.Join(dataDir, "program-runtime", "python", "venv", "bin", "python")
	if runtime.GOOS == "windows" {
		expectedPython = filepath.Join(dataDir, "program-runtime", "python", "venv", "Scripts", "python.exe")
	}
	if err != nil || result.Stdout != expectedPython {
		t.Fatalf("python=%q err=%v", result.Stdout, err)
	}
	result, err = runner.Execute(context.Background(), "sess_hardening", "sys_path", false)
	if err != nil || strings.Contains(result.Stdout, hostPythonPath) {
		t.Fatalf("host PYTHONPATH leaked into isolated kernel: %q err=%v", result.Stdout, err)
	}
	result, err = runner.Execute(context.Background(), "sess_hardening", "cwd", false)
	if err != nil || !strings.Contains(result.Stdout, filepath.Join("program-runtime", "sessions", "sess_hardening")) {
		t.Fatalf("cwd=%q err=%v", result.Stdout, err)
	}
}

func TestSubprocessRunnerUsesResolvedWorkspaceRoot(t *testing.T) { // [REQ:PRT-P1-004]
	root := t.TempDir()
	path := filepath.Join(t.TempDir(), "cwd-kernel.py")
	if err := os.WriteFile(path, []byte(`import json, os, sys
for line in sys.stdin:
    json.loads(line)
    print(json.dumps({"ok": True, "stdout": os.getcwd()}), flush=True)
`), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	runner.SetSessionWorkspace("sess_workspace", root)
	defer runner.Close()
	result, err := runner.Execute(context.Background(), "sess_workspace", "print(1)", false)
	if err != nil || result.Stdout != root {
		t.Fatalf("cwd=%q err=%v want=%q", result.Stdout, err, root)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("resolved workspace was removed: %v", err)
	}
}
