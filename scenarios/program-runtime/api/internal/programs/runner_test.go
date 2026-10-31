package programs

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

func TestVerifiedPersistentTaskErrorIdentifiesUnavailableAuthority(t *testing.T) {
	var calls atomic.Int32
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{"state":"prepared"}`))
	}))
	defer owner.Close()
	verify := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "synthetic-proof", Claims: &cliutil.VerifiedClaims{RunID: "owner"}}, nil
	})
	var verified context.Context
	provenance.Middleware(verify)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { verified = r.Context() })).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.Header.Set(cliutil.HeaderAgentIdentityToken, "synthetic-proof")
		return r
	}())
	kernel, err := filepath.Abs("../../../kernel/host/engine.py")
	if err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunnerWithBindings(kernel, nil, owner.URL+"/internal/program-runtime/bindings/execute")
	defer runner.Close()
	_, err = runner.ExecuteWithMetadataAndLimitsAndCaller(verified, "persistent", "first", "PROVENANCE_AGENT", `tasks.get(attempt_id="attempt")`, Caller{}, false, ExecutionLimits{})
	if err == nil || !strings.Contains(err.Error(), "TaskAuthorityUnavailable") || !strings.Contains(err.Error(), "persistent Runtime sessions") {
		t.Fatalf("expected actionable authority diagnostic, got %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("unavailable verified authority must not call owner anonymously")
	}
	_, err = runner.ExecuteWithMetadataAndLimitsAndCaller(context.Background(), "persistent", "second", "PROVENANCE_OPERATOR", `tasks.get(attempt_id="attempt")`, Caller{}, false, ExecutionLimits{})
	if err != nil || calls.Load() != 1 {
		t.Fatalf("diagnostic leaked into next submission: calls=%d err=%v", calls.Load(), err)
	}
}

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

func TestSubprocessRunnerRetainsBoundedBootFailureDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken-kernel.py")
	if err := os.WriteFile(path, []byte("import sys\nsys.stderr.write('x'*10000 + 'specific boot failure')\nsys.exit(17)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	defer runner.Close()
	_, err := runner.Execute(context.Background(), "sess_boot_failure", "print(1)", false)
	if err == nil || !strings.Contains(err.Error(), "specific boot failure") || !strings.Contains(err.Error(), "exit status 17") || len(err.Error()) > 4300 {
		t.Fatalf("missing or unbounded kernel diagnostics: %v", err)
	}
	if _, alive := runner.MemoryBytes("sess_boot_failure"); alive {
		t.Fatal("failed kernel retained")
	}
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

func TestSubprocessRunnerCarriesTestModePerExecution(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-mode-kernel.py")
	source := `import json, sys
for line in sys.stdin:
    request = json.loads(line)
    print(json.dumps({"ok": True, "stdout": str(bool(request.get("test_mode"))).lower(), "context_bytes": 5}), flush=True)
`
	if err := os.WriteFile(path, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	defer runner.Close()
	for _, tc := range []struct {
		ctx  context.Context
		want string
	}{{database.WithTestMode(context.Background()), "true"}, {context.Background(), "false"}} {
		result, err := runner.Execute(tc.ctx, "sess_test_mode", "inspect", false)
		if err != nil || result.Stdout != tc.want {
			t.Fatalf("stdout=%q want=%q err=%v", result.Stdout, tc.want, err)
		}
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

func TestSubprocessRunnerQueuedSubmissionRespectsDeadline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queued-kernel.py")
	source := `import json, sys, time
for line in sys.stdin:
    request = json.loads(line)
    if request.get("source") == "hold":
        print(json.dumps({"type": "progress", "ok": True}), flush=True)
        time.sleep(0.5)
    print(json.dumps({"ok": True, "stdout": "done"}), flush=True)
`
	if err := os.WriteFile(path, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	defer runner.Close()
	started := make(chan struct{})
	first := make(chan error, 1)
	go func() {
		_, err := runner.ExecuteWithMetadataAndLimitsAndProgress(context.Background(), "queued", "first", "test", "hold", false, ExecutionLimits{}, func(Result) { close(started) })
		first <- err
	}()
	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	begin := time.Now()
	_, err := runner.Execute(ctx, "queued", "second", false)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(begin) > 250*time.Millisecond {
		t.Errorf("queued execution returned %v after %s; want deadline before active submission finishes", err, time.Since(begin))
	}
	if err := <-first; err != nil {
		t.Fatalf("cancelling the queued submission interrupted the active one: %v", err)
	}
}

func TestSubprocessRunnerCarriesPerSubmissionHelperContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "helper-kernel.py")
	source := `import json, sys
for line in sys.stdin:
    request = json.loads(line)
    print(json.dumps({"ok": True, "stdout": json.dumps(request.get("declared_helper_contract"))}), flush=True)
`
	if err := os.WriteFile(path, []byte(source), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	defer runner.Close()
	limits := ExecutionLimits{DeclaredContract: &HelperContract{Version: "2", Outputs: json.RawMessage(`{"errors":{"classes":["invalid_response"]}}`)}}
	result, err := runner.ExecuteWithMetadataAndLimits(context.Background(), "helper", "declared", "test", "source", false, limits)
	if err != nil || !strings.Contains(result.Stdout, "invalid_response") || !strings.Contains(result.Stdout, `"version": "2"`) {
		t.Fatalf("declared contract did not reach kernel: result=%+v err=%v", result, err)
	}
	result, err = runner.Execute(context.Background(), "helper", "source", false)
	if err != nil || result.Stdout != "null" {
		t.Fatalf("ad-hoc submission inherited declared contract: result=%+v err=%v", result, err)
	}
}

func TestSubprocessRunnerUsesAllowlistedEnvironmentAndPinnedScratchDir(t *testing.T) { // [REQ:PRT-P1-004]
	dataDir := t.TempDir()
	t.Setenv("SCENARIO_DATA_DIR", dataDir)
	t.Setenv("VROOLI_EVENTS_API_BASE", "https://secret.invalid")
	t.Setenv("PROGRAM_RUNTIME_SECRET", "do-not-leak")
	learningState := t.TempDir()
	t.Setenv("PROGRAM_RUNTIME_LEARNING_STATE_DIR", learningState)
	hostPythonPath := filepath.Join(t.TempDir(), "host-python-path")
	t.Setenv("PYTHONPATH", hostPythonPath)
	path := filepath.Join(t.TempDir(), "probe.py")
	if err := os.WriteFile(path, []byte(`import json, os, sys
for line in sys.stdin:
    source = json.loads(line).get("source", "")
    if source == "env": value = str(os.environ.get("VROOLI_EVENTS_API_BASE"))
    elif source == "secret": value = str(os.environ.get("PROGRAM_RUNTIME_SECRET"))
    elif source == "learning_state": value = str(os.environ.get("PROGRAM_RUNTIME_LEARNING_STATE_DIR"))
    elif source == "python": value = sys.executable
    elif source == "sys_path": value = os.pathsep.join(sys.path)
    else: value = os.getcwd()
    print(json.dumps({"ok": True, "stdout": value, "context_bytes": len(value)}), flush=True)
`), 0o700); err != nil {
		t.Fatal(err)
	}
	runner := NewSubprocessRunner(path)
	defer runner.Close()
	for source, want := range map[string]string{"env": "None", "secret": "None", "learning_state": learningState} {
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
