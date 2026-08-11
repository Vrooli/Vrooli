package programs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
    print(json.dumps({"ok": True, "stdout": "2\n", "context_bytes": 1, "output_limit_bytes": 4096, "invocations": []}), flush=True)
`
	if err := os.WriteFile(path, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSubprocessRunnerReturnsTypedDeadlineAndRespawns(t *testing.T) { // [REQ:PRT-P1-005]
	runner := NewSubprocessRunner(writeTestKernel(t))
	runner.SubmissionDeadline = 75 * time.Millisecond
	defer runner.Close()

	started := time.Now()
	_, err := runner.Execute(context.Background(), "sess_deadline", "while True: pass", false)
	if time.Since(started) > time.Second {
		t.Fatalf("deadline did not bound execution: %s", time.Since(started))
	}
	var deadlineErr *DeadlineExceededError
	if !errors.As(err, &deadlineErr) {
		t.Fatalf("error=%v, want DeadlineExceededError", err)
	}
	if !strings.Contains(err.Error(), "session variables were lost") {
		t.Fatalf("deadline detail=%q", err)
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
	runner.SubmissionDeadline = time.Second
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
	t.Setenv("VROOLI_EVENTS_API_BASE", "https://secret.invalid")
	t.Setenv("PROGRAM_RUNTIME_SECRET", "do-not-leak")
	path := filepath.Join(t.TempDir(), "probe.py")
	if err := os.WriteFile(path, []byte(`import json, os, sys
for line in sys.stdin:
    source = json.loads(line).get("source", "")
    if source == "env": value = str(os.environ.get("VROOLI_EVENTS_API_BASE"))
    elif source == "secret": value = str(os.environ.get("PROGRAM_RUNTIME_SECRET"))
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
	result, err := runner.Execute(context.Background(), "sess_hardening", "cwd", false)
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
