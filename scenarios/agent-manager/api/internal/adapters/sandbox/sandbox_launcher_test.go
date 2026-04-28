// Tests for the SandboxLauncher — the workspace-sandbox-backed runner.Launcher
// used in protected mode. These tests stand up an httptest server that
// implements just enough of the workspace-sandbox API (PUT files, POST
// processes, GET logs, GET processes, DELETE process) to exercise the
// launcher contract without needing a live workspace-sandbox process.
//
// See execute/protected-sandbox-agent-launch.

package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"

	"github.com/google/uuid"
)

// mockSandbox is a tiny in-memory workspace-sandbox simulator for the
// launcher tests. It records the requests it received and replies with
// shapes that match the real workspace-sandbox handlers.
type mockSandbox struct {
	mu sync.Mutex

	// Files written via PUT /files/content?path=...
	files map[string][]byte

	// Process state. We model a single process at a time.
	procPID     int
	procRunning atomic.Bool
	procLog     []byte
	procStarted atomic.Bool

	// Recorded requests, for assertions.
	startProcessBody map[string]any
	startProcessSeen atomic.Bool
	killSeen         atomic.Bool

	// Knobs for forced behaviors:
	startProcessStatus int    // override the StartProcess HTTP status (0 = 201)
	startProcessBody2  string // override the StartProcess body (when status != 201)
}

func newMockSandbox(initialPID int) *mockSandbox {
	m := &mockSandbox{
		files:   make(map[string][]byte),
		procPID: initialPID,
	}
	return m
}

// startServer returns an httptest server wired to m.
func (m *mockSandbox) startServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/sandboxes/", func(w http.ResponseWriter, r *http.Request) {
		// Routes:
		//   PUT  /api/v1/sandboxes/{id}/files/content?path=...
		//   POST /api/v1/sandboxes/{id}/processes
		//   GET  /api/v1/sandboxes/{id}/processes
		//   GET  /api/v1/sandboxes/{id}/processes/{pid}/logs?offset=...
		//   DELETE /api/v1/sandboxes/{id}/processes/{pid}
		path := r.URL.Path
		switch {
		case r.Method == "PUT" && strings.HasSuffix(path, "/files/content"):
			m.handleWriteFile(w, r)
		case r.Method == "POST" && strings.HasSuffix(path, "/processes"):
			m.handleStartProcess(w, r)
		case r.Method == "GET" && strings.HasSuffix(path, "/processes"):
			m.handleListProcesses(w, r)
		case r.Method == "GET" && strings.Contains(path, "/processes/") && strings.HasSuffix(path, "/logs"):
			m.handleGetLogs(w, r)
		case r.Method == "DELETE" && strings.Contains(path, "/processes/"):
			m.handleKillProcess(w, r)
		default:
			t.Logf("mockSandbox: unhandled %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	return httptest.NewServer(mux)
}

func (m *mockSandbox) handleWriteFile(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var parsed struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	_ = json.Unmarshal(body, &parsed)
	m.mu.Lock()
	m.files[relPath] = []byte(parsed.Content)
	m.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"path": relPath, "size": len(parsed.Content), "created": true})
}

func (m *mockSandbox) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)
	m.mu.Lock()
	m.startProcessBody = parsed
	status := m.startProcessStatus
	customBody := m.startProcessBody2
	m.mu.Unlock()
	m.startProcessSeen.Store(true)

	if status != 0 && status != http.StatusCreated && status != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(customBody))
		return
	}

	m.procRunning.Store(true)
	m.procStarted.Store(true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pid":       m.procPID,
		"sandboxId": uuid.New(),
		"command":   parsed["command"],
	})
}

func (m *mockSandbox) handleListProcesses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if !m.procStarted.Load() {
		_ = json.NewEncoder(w).Encode(map[string]any{"processes": []any{}})
		return
	}
	if !m.procRunning.Load() {
		// Process disappeared — treat as exited.
		_ = json.NewEncoder(w).Encode(map[string]any{"processes": []any{}})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"processes": []map[string]any{{"pid": m.procPID, "status": "running"}},
	})
}

func (m *mockSandbox) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	offset := int64(0)
	if v := r.URL.Query().Get("offset"); v != "" {
		fmt.Sscanf(v, "%d", &offset)
	}
	m.mu.Lock()
	logCopy := append([]byte(nil), m.procLog...)
	m.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if int64(len(logCopy)) <= offset {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pid":       m.procPID,
			"sizeBytes": int64(len(logCopy)),
			"isActive":  m.procRunning.Load(),
			"content":   "",
		})
		return
	}
	tail := logCopy[offset:]
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pid":       m.procPID,
		"sizeBytes": int64(len(logCopy)),
		"isActive":  m.procRunning.Load(),
		"content":   string(tail),
	})
}

func (m *mockSandbox) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	m.killSeen.Store(true)
	m.procRunning.Store(false)
	w.WriteHeader(http.StatusNoContent)
}

// appendLog atomically appends to the process log (called by tests to
// simulate the agent emitting stdout).
func (m *mockSandbox) appendLog(s string) {
	m.mu.Lock()
	m.procLog = append(m.procLog, []byte(s)...)
	m.mu.Unlock()
}

// markExited simulates the process exiting naturally.
func (m *mockSandbox) markExited() {
	m.procRunning.Store(false)
}

// =============================================================================
// Tests
// =============================================================================

// TestSandboxLauncher_LaunchAndStreamLog drives the happy path: launch a
// process, append log content, mark the process exited, verify Stdout
// receives all the bytes, and Wait returns nil.
func TestSandboxLauncher_LaunchAndStreamLog(t *testing.T) {
	mock := newMockSandbox(99)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())
	launcher.PollInterval = 20 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		Args:       []string{"--print"},
		Env:        []string{"HOME=/workspace"},
		WorkingDir: "/workspace",
		Stdin:      strings.NewReader("hello-prompt"),
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Background: stream log content over time and then mark the process exited.
	go func() {
		time.Sleep(50 * time.Millisecond)
		mock.appendLog(`{"event":"start"}` + "\n")
		time.Sleep(50 * time.Millisecond)
		mock.appendLog(`{"event":"chunk","data":"hello"}` + "\n")
		time.Sleep(50 * time.Millisecond)
		mock.markExited()
	}()

	// Reading Stdout should block until log content arrives, then unblock.
	out, err := io.ReadAll(proc.Stdout())
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(out), "hello") {
		t.Errorf("stdout = %q; want substring %q", string(out), "hello")
	}

	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}

	if !mock.startProcessSeen.Load() {
		t.Error("workspace-sandbox StartProcess was not invoked")
	}
}

// TestSandboxLauncher_StagesStdinAsFile verifies LaunchRequest.Stdin gets
// written as a file in the sandbox AND the process command-line includes
// a redirect from that file (so the agent process sees stdin via redirect).
func TestSandboxLauncher_StagesStdinAsFile(t *testing.T) {
	mock := newMockSandbox(101)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())
	launcher.PollInterval = 20 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const promptContent = "this-is-the-prompt-content"
	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command: "claude",
		Args:    []string{"--print"},
		Stdin:   strings.NewReader(promptContent),
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Verify a prompt file was written.
	mock.mu.Lock()
	var foundPath string
	for path, body := range mock.files {
		if strings.Contains(path, ".am-prompts") && string(body) == promptContent {
			foundPath = path
			break
		}
	}
	startBody := mock.startProcessBody
	mock.mu.Unlock()

	if foundPath == "" {
		t.Fatalf("expected stdin to be staged as file under .am-prompts; saw files=%v", mock.files)
	}

	// Verify the StartProcess request used a bash wrapper that redirects
	// stdin from the staged file.
	cmd, _ := startBody["command"].(string)
	if cmd != "bash" {
		t.Errorf("StartProcess command = %q; want bash (wrapper for stdin redirect)", cmd)
	}
	args, _ := startBody["args"].([]any)
	if len(args) < 2 {
		t.Fatalf("StartProcess args = %v; want at least [\"-c\", <shell-line>]", args)
	}
	if first, _ := args[0].(string); first != "-c" {
		t.Errorf("StartProcess args[0] = %q; want -c", first)
	}
	shellLine, _ := args[1].(string)
	if !strings.Contains(shellLine, foundPath) {
		t.Errorf("shell wrapper does not reference prompt file %q; got: %s", foundPath, shellLine)
	}
	if !strings.Contains(shellLine, "exec ") {
		t.Errorf("shell wrapper should `exec` the target so signals reach it; got: %s", shellLine)
	}

	// Cleanup: kill so Wait returns.
	proc.Kill()
	_ = proc.Wait()
}

// TestSandboxLauncher_KillReturnsThroughDelete verifies Kill issues a DELETE
// to the sandbox and Wait unblocks promptly.
func TestSandboxLauncher_KillReturnsThroughDelete(t *testing.T) {
	mock := newMockSandbox(202)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())
	launcher.PollInterval = 20 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "sleep", Args: []string{"30"}})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())

	time.Sleep(80 * time.Millisecond)
	proc.Kill()

	done := make(chan struct{})
	go func() { _ = proc.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return within 2s of Kill")
	}
	if !mock.killSeen.Load() {
		t.Error("workspace-sandbox kill endpoint was not invoked")
	}
}

// TestSandboxLauncher_ContextCancelKills verifies ctx cancellation triggers
// a kill on the remote process.
func TestSandboxLauncher_ContextCancelKills(t *testing.T) {
	mock := newMockSandbox(303)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())
	launcher.PollInterval = 20 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "sleep", Args: []string{"30"}})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())

	time.Sleep(80 * time.Millisecond)
	cancel()

	done := make(chan struct{})
	go func() { _ = proc.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return within 2s of ctx.Cancel")
	}
}

// TestSandboxLauncher_StartProcess403ReturnsLaunchBlocked verifies that a
// structured 403 (e.g., git allowlist denial) is surfaced as a typed
// *LaunchBlocked error rather than a generic launch failure.
func TestSandboxLauncher_StartProcess403ReturnsLaunchBlocked(t *testing.T) {
	mock := newMockSandbox(404)
	mock.startProcessStatus = http.StatusForbidden
	mock.startProcessBody2 = `{"error":"git_verb_blocked","verb":"push","message":"git verb 'push' is not in the allowlist"}`
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "git", Args: []string{"push"}})
	if err == nil {
		t.Fatal("Launch with 403 returned nil; want *LaunchBlocked")
	}
	var blocked *LaunchBlocked
	if !errors.As(err, &blocked) {
		t.Fatalf("err = %T (%v); want *LaunchBlocked", err, err)
	}
	if blocked.Code != "git_verb_blocked" {
		t.Errorf("blocked.Code = %q; want git_verb_blocked", blocked.Code)
	}
	if blocked.Verb != "push" {
		t.Errorf("blocked.Verb = %q; want push", blocked.Verb)
	}
}

// TestShellQuote_HandlesEmbeddedSingleQuotes is a focused test on the
// single-quote escaping helper used to build the bash wrapper.
func TestShellQuote_HandlesEmbeddedSingleQuotes(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "''"},
		{"hello", "'hello'"},
		{"it's", `'it'\''s'`},
		{"a 'b' c", `'a '\''b'\'' c'`},
	}
	for _, tc := range cases {
		got := shellQuote(tc.in)
		if got != tc.want {
			t.Errorf("shellQuote(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// TestEnvSliceToMap verifies the os.Environ() → map[string]string conversion.
func TestEnvSliceToMap(t *testing.T) {
	in := []string{"A=1", "B=hello world", "C=", "noEqualsHere", "D=multi=equals=ok"}
	got := envSliceToMap(in)
	want := map[string]string{
		"A": "1",
		"B": "hello world",
		"C": "",
		"D": "multi=equals=ok",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d; want %d (got %v)", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("got[%q] = %q; want %q", k, got[k], v)
		}
	}
	if _, exists := got["noEqualsHere"]; exists {
		t.Error("env entry without '=' should be skipped")
	}
}
