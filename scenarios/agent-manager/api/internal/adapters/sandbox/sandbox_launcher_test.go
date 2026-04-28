// Tests for the SandboxLauncher — the workspace-sandbox-backed runner.Launcher
// used in protected mode. These tests stand up an httptest server that
// implements just enough of the workspace-sandbox API (POST processes,
// SSE-based GET /logs/stream, POST /stdin, DELETE /processes/{pid}) to
// exercise the launcher contract without needing a live workspace-sandbox
// process.
//
// See execute/protected-sandbox-agent-launch and the four ws-sb-* follow-on
// items.

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

// mockSandbox is an httptest-backed simulator of the workspace-sandbox
// /processes endpoints. It models a single process at a time. Tests can
// drive the lifecycle by calling appendStdout / appendStderr / markExited
// to feed bytes into the SSE channels and wind down the streams cleanly.
type mockSandbox struct {
	mu sync.Mutex

	procPID     int
	procRunning atomic.Bool
	procStarted atomic.Bool

	// Recorded request state for assertions.
	startProcessBody  map[string]any
	startProcessSeen  atomic.Bool
	killSeen          atomic.Bool
	stdinBody         []byte
	stdinClose        atomic.Bool
	stdinSeen         atomic.Bool
	startProcessCode  int
	startProcessReply string

	// Per-stream subscriber registry. Each /logs/stream connection adds a
	// channel that receives chunks; markExited closes them after sending
	// the exit frame.
	stdoutSubs []chan sseChunk
	stderrSubs []chan sseChunk
	subsMu     sync.Mutex

	// Buffered exit info; sent on exitFrame as JSON.
	exitInfo *remoteExitInfo
}

type sseChunk struct {
	event string
	data  []byte
}

func newMockSandbox(initialPID int) *mockSandbox {
	return &mockSandbox{procPID: initialPID}
}

// startServer wires the routes and returns the running test server.
func (m *mockSandbox) startServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sandboxes/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case r.Method == "POST" && strings.HasSuffix(path, "/processes"):
			m.handleStartProcess(w, r)
		case r.Method == "POST" && strings.Contains(path, "/processes/") && strings.HasSuffix(path, "/stdin"):
			m.handleStdin(w, r)
		case r.Method == "GET" && strings.Contains(path, "/processes/") && strings.HasSuffix(path, "/logs/stream"):
			m.handleStreamLogs(w, r)
		case r.Method == "DELETE" && strings.Contains(path, "/processes/"):
			m.handleKillProcess(w, r)
		default:
			t.Logf("mockSandbox: unhandled %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return httptest.NewServer(mux)
}

func (m *mockSandbox) handleStartProcess(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	var parsed map[string]any
	_ = json.Unmarshal(body, &parsed)

	m.mu.Lock()
	m.startProcessBody = parsed
	override := m.startProcessCode
	overrideBody := m.startProcessReply
	m.mu.Unlock()
	m.startProcessSeen.Store(true)

	if override != 0 && override != http.StatusCreated && override != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(override)
		_, _ = w.Write([]byte(overrideBody))
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
		"withStdin": parsed["withStdin"],
	})
}

func (m *mockSandbox) handleStdin(w http.ResponseWriter, r *http.Request) {
	m.stdinSeen.Store(true)
	body, _ := io.ReadAll(r.Body)
	m.mu.Lock()
	m.stdinBody = append(m.stdinBody, body...)
	m.mu.Unlock()
	if r.URL.Query().Get("close") == "true" {
		m.stdinClose.Store(true)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"pid":          m.procPID,
		"bytesWritten": len(body),
		"closed":       r.URL.Query().Get("close") == "true",
	})
}

func (m *mockSandbox) handleStreamLogs(w http.ResponseWriter, r *http.Request) {
	stream := r.URL.Query().Get("stream")
	if stream != "stdout" && stream != "stderr" {
		http.Error(w, "missing stream", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "no flusher", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	ch := make(chan sseChunk, 32)
	m.subsMu.Lock()
	if stream == "stdout" {
		m.stdoutSubs = append(m.stdoutSubs, ch)
	} else {
		m.stderrSubs = append(m.stderrSubs, ch)
	}
	m.subsMu.Unlock()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case chunk, open := <-ch:
			if !open {
				_, _ = fmt.Fprintf(w, "event: end\ndata: stream closed\n\n")
				flusher.Flush()
				return
			}
			if chunk.event == "exit" {
				_, _ = fmt.Fprintf(w, "event: exit\ndata: %s\n\n", string(chunk.data))
			} else {
				_, _ = fmt.Fprintf(w, "data: %s\n\n", string(chunk.data))
			}
			flusher.Flush()
		}
	}
}

func (m *mockSandbox) handleKillProcess(w http.ResponseWriter, r *http.Request) {
	m.killSeen.Store(true)
	m.procRunning.Store(false)
	// Closing all subscribers terminates their SSE goroutines.
	m.subsMu.Lock()
	for _, ch := range m.stdoutSubs {
		close(ch)
	}
	for _, ch := range m.stderrSubs {
		close(ch)
	}
	m.stdoutSubs = nil
	m.stderrSubs = nil
	m.subsMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// appendStdout pushes a chunk to all stdout subscribers.
func (m *mockSandbox) appendStdout(b []byte) {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()
	for _, ch := range m.stdoutSubs {
		ch <- sseChunk{data: append([]byte(nil), b...)}
	}
}

// appendStderr pushes a chunk to all stderr subscribers.
func (m *mockSandbox) appendStderr(b []byte) {
	m.subsMu.Lock()
	defer m.subsMu.Unlock()
	for _, ch := range m.stderrSubs {
		ch <- sseChunk{data: append([]byte(nil), b...)}
	}
}

// markExited sends the exit frame on both streams and closes them.
func (m *mockSandbox) markExited(info remoteExitInfo) {
	m.procRunning.Store(false)
	m.exitInfo = &info
	payload, _ := json.Marshal(info)
	m.subsMu.Lock()
	for _, ch := range m.stdoutSubs {
		ch <- sseChunk{event: "exit", data: payload}
		close(ch)
	}
	for _, ch := range m.stderrSubs {
		ch <- sseChunk{event: "exit", data: payload}
		close(ch)
	}
	m.stdoutSubs = nil
	m.stderrSubs = nil
	m.subsMu.Unlock()
}

// =============================================================================
// Tests
// =============================================================================

// TestSandboxLauncher_LaunchAndStreamLog drives the happy path: launch a
// process, push stdout chunks via SSE, mark exited, verify Stdout receives
// all the bytes and Wait returns nil.
func TestSandboxLauncher_LaunchAndStreamLog(t *testing.T) {
	mock := newMockSandbox(99)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		Args:       []string{"--print"},
		Env:        []string{"HOME=/workspace"},
		WorkingDir: "/workspace",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	// Start collecting stdout.
	got := make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(proc.Stdout())
		got <- string(buf)
	}()

	// Wait briefly for the SSE subscription to register on the server side.
	time.Sleep(50 * time.Millisecond)
	mock.appendStdout([]byte(`{"event":"start"}` + "\n"))
	mock.appendStdout([]byte(`{"event":"chunk","data":"hello"}` + "\n"))
	mock.markExited(remoteExitInfo{ExitCode: 0})

	if err := proc.Wait(); err != nil {
		t.Errorf("Wait: %v", err)
	}
	select {
	case out := <-got:
		if !strings.Contains(out, "hello") {
			t.Errorf("stdout = %q; want substring %q", out, "hello")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive stdout in time")
	}

	if !mock.startProcessSeen.Load() {
		t.Error("StartProcess was not invoked")
	}
}

// TestSandboxLauncher_StdinPostedNotStaged verifies LaunchRequest.Stdin
// reaches the /processes/{pid}/stdin endpoint with close=true (not the
// old .am-prompts file-staging path).
func TestSandboxLauncher_StdinPostedNotStaged(t *testing.T) {
	mock := newMockSandbox(101)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

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

	if !mock.stdinSeen.Load() {
		t.Fatal("expected stdin POST endpoint to be hit")
	}
	if !mock.stdinClose.Load() {
		t.Errorf("expected ?close=true on stdin POST")
	}
	mock.mu.Lock()
	gotStdin := string(mock.stdinBody)
	startBody := mock.startProcessBody
	mock.mu.Unlock()
	if gotStdin != promptContent {
		t.Errorf("stdin body = %q; want %q", gotStdin, promptContent)
	}

	// StartProcess must say the underlying command (not bash wrapper) and
	// signal withStdin=true to the server.
	if cmd, _ := startBody["command"].(string); cmd != "claude" {
		t.Errorf("StartProcess command = %q; want %q (no bash wrapper)", cmd, "claude")
	}
	if w, _ := startBody["withStdin"].(bool); !w {
		t.Errorf("StartProcess withStdin = %v; want true", w)
	}
	args, _ := startBody["args"].([]any)
	if len(args) != 1 || args[0] != "--print" {
		t.Errorf("StartProcess args = %v; want [--print]", args)
	}

	// Cleanup so Wait returns.
	go func() {
		time.Sleep(50 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())
	_ = proc.Wait()
}

// TestSandboxLauncher_KillReturnsThroughDelete verifies Kill issues a
// DELETE and Wait unblocks promptly.
func TestSandboxLauncher_KillReturnsThroughDelete(t *testing.T) {
	mock := newMockSandbox(202)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "sleep", Args: []string{"30"}})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

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
		t.Error("kill endpoint was not invoked")
	}
}

// TestSandboxLauncher_ContextCancelKills verifies ctx cancellation
// triggers the SSE streams to close so Wait returns.
func TestSandboxLauncher_ContextCancelKills(t *testing.T) {
	mock := newMockSandbox(303)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithCancel(context.Background())
	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "sleep", Args: []string{"30"}})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

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
// *LaunchBlocked error.
func TestSandboxLauncher_StartProcess403ReturnsLaunchBlocked(t *testing.T) {
	mock := newMockSandbox(404)
	mock.startProcessCode = http.StatusForbidden
	mock.startProcessReply = `{"error":"git_verb_blocked","verb":"push","message":"git verb 'push' is not in the allowlist"}`
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

// TestSandboxLauncher_StderrStreamPopulates ensures the real /processes
// stderr stream is wired through to proc.Stderr() (not a closed pipe).
func TestSandboxLauncher_StderrStreamPopulates(t *testing.T) {
	mock := newMockSandbox(505)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "echo"})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}

	stdoutCh := make(chan string, 1)
	stderrCh := make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(proc.Stdout())
		stdoutCh <- string(buf)
	}()
	go func() {
		buf, _ := io.ReadAll(proc.Stderr())
		stderrCh <- string(buf)
	}()

	time.Sleep(50 * time.Millisecond)
	mock.appendStdout([]byte("on stdout"))
	mock.appendStderr([]byte("on stderr"))
	mock.markExited(remoteExitInfo{ExitCode: 0})

	_ = proc.Wait()
	select {
	case s := <-stdoutCh:
		if !strings.Contains(s, "on stdout") {
			t.Errorf("stdout = %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stdout never closed")
	}
	select {
	case s := <-stderrCh:
		if !strings.Contains(s, "on stderr") {
			t.Errorf("stderr = %q", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stderr never closed")
	}
}

// TestSandboxLauncher_ExitInfoSurfaces verifies that the structured exit
// frame results in a *remoteExitError carrying exit code, signal, and
// OOMKilled flag.
func TestSandboxLauncher_ExitInfoSurfaces(t *testing.T) {
	mock := newMockSandbox(606)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "false"})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	time.Sleep(50 * time.Millisecond)
	mock.markExited(remoteExitInfo{ExitCode: 137, Signal: 9, OOMKilled: true})

	werr := proc.Wait()
	if werr == nil {
		t.Fatal("Wait returned nil; want non-zero exit error")
	}
	var ree *remoteExitError
	if !errors.As(werr, &ree) {
		t.Fatalf("err = %T (%v); want *remoteExitError", werr, werr)
	}
	if ree.ExitCode() != 137 {
		t.Errorf("ExitCode = %d; want 137", ree.ExitCode())
	}
	if ree.signal != 9 {
		t.Errorf("signal = %d; want 9", ree.signal)
	}
	if !ree.oomKilled {
		t.Error("oomKilled should be true")
	}
}

// TestSSEParser_BasicEvents pins the SSE field grammar.
func TestSSEParser_BasicEvents(t *testing.T) {
	input := "data: hello\n\n" +
		"data: line1\ndata: line2\n\n" +
		"event: exit\ndata: {\"exitCode\":0}\n\n" +
		"event: end\ndata: bye\n\n"
	parser := newSSEParser(strings.NewReader(input))

	events := []sseEvent{}
	for {
		ev, ok := parser.next()
		if !ok {
			break
		}
		events = append(events, ev)
	}
	if len(events) != 4 {
		t.Fatalf("got %d events; want 4: %+v", len(events), events)
	}
	if string(events[0].data) != "hello" {
		t.Errorf("event[0].data = %q; want hello", string(events[0].data))
	}
	if string(events[1].data) != "line1\nline2" {
		t.Errorf("event[1].data = %q; want line1\\nline2", string(events[1].data))
	}
	if events[2].eventType != "exit" || string(events[2].data) != `{"exitCode":0}` {
		t.Errorf("event[2] = %+v", events[2])
	}
	if events[3].eventType != "end" {
		t.Errorf("event[3].eventType = %q; want end", events[3].eventType)
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
