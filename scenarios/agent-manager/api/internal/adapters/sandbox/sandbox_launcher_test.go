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
	"agent-manager/internal/domain"

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

	// hostMergedDir is the value returned from GET /api/v1/sandboxes/{id};
	// the launcher uses it to translate host workingDir / env values.
	// Empty means "no translation expected" (translateHostPathToNamespace
	// becomes identity in that case).
	hostMergedDir string

	// homeOverlayState is what the mock GET returns. Empty defaults to
	// "present" so existing happy-path tests don't need to set it.
	homeOverlayState string

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
		case r.Method == "GET" && !strings.Contains(path, "/processes"):
			m.handleGetSandbox(w, r)
		default:
			t.Logf("mockSandbox: unhandled %s %s", r.Method, path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return httptest.NewServer(mux)
}

func (m *mockSandbox) handleGetSandbox(w http.ResponseWriter, r *http.Request) {
	// Extract sandbox id from /api/v1/sandboxes/<id>.
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/sandboxes/"), "/")
	id := parts[0]
	state := m.homeOverlayState
	if state == "" {
		// Mock default = Present so existing happy-path tests don't have
		// to set the field. The Phase F regression tests explicitly set
		// HomeOverlayAbsent / HomeOverlayUnsupported when needed.
		state = string(HomeOverlayPresent)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":               id,
		"scopePath":        "/scope",
		"projectRoot":      "/scope",
		"status":           "active",
		"mergedDir":        m.hostMergedDir,
		"homeOverlayState": state,
	})
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

// TestSandboxLauncher_NoExitInfo_ReportsFailure ensures that when both
// SSE log streams close without the server emitting `event: exit`
// (Phase B regression: bwrap chdir failures used to drop the exit
// frame), the client surfaces ErrSandboxNoExitInfo instead of treating
// the run as a clean success.
func TestSandboxLauncher_NoExitInfo_ReportsFailure(t *testing.T) {
	mock := newMockSandbox(909)
	mock.hostMergedDir = "/var/lib/workspace-sandbox/sb-no-exit/merged"
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "claude"})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	// Close subscribers WITHOUT sending an exit frame — simulates the
	// pre-Phase-B race where the SSE server never noticed exit info.
	time.Sleep(50 * time.Millisecond)
	mock.subsMu.Lock()
	for _, ch := range mock.stdoutSubs {
		close(ch)
	}
	for _, ch := range mock.stderrSubs {
		close(ch)
	}
	mock.stdoutSubs = nil
	mock.stderrSubs = nil
	mock.subsMu.Unlock()

	werr := proc.Wait()
	if werr == nil {
		t.Fatal("Wait returned nil; want ErrSandboxNoExitInfo")
	}
	// Sentinel-compatibility: callers that switch on the underlying
	// sentinel via errors.Is must keep working after the typed wrapping.
	if !errors.Is(werr, ErrSandboxNoExitInfo) {
		t.Errorf("Wait err = %v; want errors.Is(err, ErrSandboxNoExitInfo)=true", werr)
	}
	// Type assertion: the wrapper must be a *domain.SandboxError carrying
	// Operation="no_exit_info" so the orchestration categorizer surfaces
	// SANDBOX_NO_EXIT_INFO instead of falling through to ErrCodeInternal.
	var sbxErr *domain.SandboxError
	if !errors.As(werr, &sbxErr) {
		t.Fatalf("Wait err = %T; want errors.As to *domain.SandboxError", werr)
	}
	if sbxErr.Operation != "no_exit_info" {
		t.Errorf("SandboxError.Operation = %q; want %q", sbxErr.Operation, "no_exit_info")
	}
	if got := sbxErr.Code(); got != domain.ErrCodeSandboxNoExitInfo {
		t.Errorf("SandboxError.Code() = %q; want %q", got, domain.ErrCodeSandboxNoExitInfo)
	}
}

// TestSandboxLauncher_LogStreamSurvivesPast30s pins the 2026-04-28
// fix for the silent SANDBOX_NO_EXIT_INFO at the 30-second mark. The
// default httpClient.Timeout was 30s as a *total* deadline including
// body read, so any agent run that exceeded 30 seconds had its SSE
// log stream killed by the client (not by the server) — the launcher
// then read EOF without seeing event:exit and surfaced
// ErrSandboxNoExitInfo. The fix routes streams through a dedicated
// streamClient with no total Timeout (only Transport-level connect
// and header timeouts).
//
// This test establishes a stream, holds the connection for a duration
// well past the old 30s limit, then sends an exit frame, and verifies
// the launcher saw the exit cleanly.
func TestSandboxLauncher_LogStreamSurvivesPast30s(t *testing.T) {
	if testing.Short() {
		t.Skip("long-running stream test; skip in -short mode")
	}

	mock := newMockSandbox(911)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "sleep"})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	// Hold the stream open longer than the old 30s client timeout, then
	// emit a real exit. Pre-fix the connection was dead by 30s and Wait
	// returned ErrSandboxNoExitInfo regardless of what the server did
	// after that.
	time.Sleep(35 * time.Second)
	mock.markExited(remoteExitInfo{ExitCode: 0})

	werr := proc.Wait()
	if werr != nil {
		t.Fatalf("Wait err = %v; want nil (exit frame should arrive past 30s with the streamClient fix)", werr)
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

// TestTranslateHostPathToNamespace exercises the host→namespace path
// rewriter directly. These cases pin the contract that bwrap's
// `--bind <hostMerged> /workspace` enforces.
func TestTranslateHostPathToNamespace(t *testing.T) {
	const host = "/home/matt/.local/share/workspace-sandbox/abc/merged"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty input passes through", "", ""},
		{"exact match → /workspace", host, SandboxNamespacePath},
		{"subpath → /workspace/<rest>", host + "/sub/file.txt", SandboxNamespacePath + "/sub/file.txt"},
		{"unrelated absolute path passes through", "/etc/hosts", "/etc/hosts"},
		{"already namespace path passes through", SandboxNamespacePath + "/x", SandboxNamespacePath + "/x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := translateHostPathToNamespace(c.in, host)
			if got != c.want {
				t.Errorf("translateHostPathToNamespace(%q, host) = %q; want %q", c.in, got, c.want)
			}
		})
	}

	t.Run("empty hostMerged is identity", func(t *testing.T) {
		got := translateHostPathToNamespace("/some/path", "")
		if got != "/some/path" {
			t.Errorf("got %q; want identity passthrough", got)
		}
	})
}

// TestResolveWorkingDir pins the workdir contract:
//   - empty → SandboxNamespacePath
//   - host merged dir / subpath → translated
//   - already in-namespace → unchanged
//   - other absolute host path → *LaunchBlocked{workdir_outside_sandbox}
func TestResolveWorkingDir(t *testing.T) {
	const host = "/home/x/.local/share/workspace-sandbox/sb1/merged"

	t.Run("empty defaults to namespace path", func(t *testing.T) {
		got, err := resolveWorkingDir("", host)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != SandboxNamespacePath {
			t.Errorf("got %q; want %q", got, SandboxNamespacePath)
		}
	})

	t.Run("host merged path translates", func(t *testing.T) {
		got, err := resolveWorkingDir(host, host)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != SandboxNamespacePath {
			t.Errorf("got %q; want %q", got, SandboxNamespacePath)
		}
	})

	t.Run("subpath of merged translates", func(t *testing.T) {
		got, err := resolveWorkingDir(host+"/foo", host)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		want := SandboxNamespacePath + "/foo"
		if got != want {
			t.Errorf("got %q; want %q", got, want)
		}
	})

	t.Run("already namespace path passes through", func(t *testing.T) {
		got, err := resolveWorkingDir(SandboxNamespacePath, host)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if got != SandboxNamespacePath {
			t.Errorf("got %q; want %q", got, SandboxNamespacePath)
		}
	})

	t.Run("unrelated host path is blocked", func(t *testing.T) {
		_, err := resolveWorkingDir("/etc/hosts", host)
		var blocked *LaunchBlocked
		if !errors.As(err, &blocked) {
			t.Fatalf("err = %T (%v); want *LaunchBlocked", err, err)
		}
		if blocked.Code != "workdir_outside_sandbox" {
			t.Errorf("blocked.Code = %q; want workdir_outside_sandbox", blocked.Code)
		}
	})
}

// TestSandboxLauncher_LaunchTranslatesHostMergedPath verifies that when
// the run executor passes the *host* merged path as WorkingDir, the
// launcher rewrites it to /workspace before POSTing — preventing the
// "bwrap: Can't chdir to /home/.../merged: No such file or directory"
// regression.
func TestSandboxLauncher_LaunchTranslatesHostMergedPath(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-test/merged"

	mock := newMockSandbox(707)
	mock.hostMergedDir = host
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		WorkingDir: host,
		Env:        []string{"VROOLI_SANDBOX_MERGED=" + host, "PATH=/usr/bin"},
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	if got, _ := body["workingDir"].(string); got != SandboxNamespacePath {
		t.Errorf("workingDir = %q; want %q (host path must be translated)", got, SandboxNamespacePath)
	}
	envAny, _ := body["env"].(map[string]any)
	if got, _ := envAny["VROOLI_SANDBOX_MERGED"].(string); got != SandboxNamespacePath {
		t.Errorf("env VROOLI_SANDBOX_MERGED = %q; want %q", got, SandboxNamespacePath)
	}
	if got, _ := envAny["PATH"].(string); got != "/usr/bin" {
		t.Errorf("env PATH = %q; want /usr/bin (untouched)", got)
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestTranslateCommandToNamespace pins the binary-path rewrite contract.
// The mapping must stay in lockstep with the vrooli-aware bind layout in
// scenarios/workspace-sandbox/api/internal/driver/bwrap.go.
func TestTranslateCommandToNamespace(t *testing.T) {
	const home = "/home/matt"
	cases := []struct {
		name        string
		command     string
		hostHome    string
		state       HomeOverlayState
		want        string
		wantRewrite bool
		wantErr     bool
	}{
		{
			name:        "empty passes through",
			command:     "",
			hostHome:    home,
			want:        "",
			wantRewrite: false,
		},
		{
			name:        "bare basename → unchanged (PATH lookup in sandbox)",
			command:     "claude",
			hostHome:    home,
			want:        "claude",
			wantRewrite: false,
		},
		{
			name:        "relative path → unchanged",
			command:     "./bin/claude",
			hostHome:    home,
			want:        "./bin/claude",
			wantRewrite: false,
		},
		{
			name:        "$HOME/.local/bin/claude → unchanged (profile binds at host path)",
			command:     home + "/.local/bin/claude",
			hostHome:    home,
			want:        home + "/.local/bin/claude",
			wantRewrite: false,
		},
		{
			name:        "$HOME with trailing slash still matches",
			command:     home + "/.local/bin/codex",
			hostHome:    home + "/",
			want:        home + "/.local/bin/codex",
			wantRewrite: false,
		},
		{
			name:        "$HOME/.local/share/X/Y → unchanged (companion bind for symlink targets)",
			command:     home + "/.local/share/claude/versions/2.1.121",
			hostHome:    home,
			want:        home + "/.local/share/claude/versions/2.1.121",
			wantRewrite: false,
		},
		{
			name:        "/usr/bin/X → unchanged (sandbox mounts /usr at /usr)",
			command:     "/usr/bin/git",
			hostHome:    home,
			want:        "/usr/bin/git",
			wantRewrite: false,
		},
		{
			name:        "/bin/X → unchanged (sandbox mounts /bin at /bin)",
			command:     "/bin/sh",
			hostHome:    home,
			want:        "/bin/sh",
			wantRewrite: false,
		},
		{
			name:        "/usr/local/bin/X → unchanged (already namespace path)",
			command:     "/usr/local/bin/foo",
			hostHome:    home,
			want:        "/usr/local/bin/foo",
			wantRewrite: false,
		},
		{
			name:        "unknown host-absolute → basename fallback",
			command:     "/opt/homebrew/bin/claude",
			hostHome:    home,
			want:        "claude",
			wantRewrite: true,
		},
		{
			name:        "empty hostHome disables ~/.local/bin rule but basename fallback still applies",
			command:     "/some/random/path/tool",
			hostHome:    "",
			want:        "tool",
			wantRewrite: true,
		},
		{
			name:        "different user's home directory does not match",
			command:     "/home/other/.local/bin/claude",
			hostHome:    home,
			want:        "claude",
			wantRewrite: true,
		},
	}
	// Default state for cases that don't set it explicitly: Present.
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := c.state
			if state == "" {
				state = HomeOverlayPresent
			}
			layout := NamespaceLayout{HostHome: c.hostHome, HomeOverlayState: state}
			got, err := translateCommandToNamespace(c.command, layout)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got command=%q", got)
				}
				var typed *ErrCommandRequiresHomeOverlay
				if !errors.As(err, &typed) {
					t.Errorf("expected ErrCommandRequiresHomeOverlay, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got command %q; want %q", got, c.want)
			}
		})
	}
}

// TestTranslateCommandToNamespace_RefusesHomeWhenStateAbsent — the
// load-bearing seam introduced in Phase F. A command pointing under
// $HOME with state != Present must surface as ErrCommandRequiresHomeOverlay
// before the launcher POSTs to workspace-sandbox.
func TestTranslateCommandToNamespace_RefusesHomeWhenStateAbsent(t *testing.T) {
	const home = "/home/matt"
	for _, state := range []HomeOverlayState{HomeOverlayAbsent, HomeOverlayUnsupported, HomeOverlayNotRequested} {
		t.Run(string(state), func(t *testing.T) {
			layout := NamespaceLayout{HostHome: home, HomeOverlayState: state}
			_, err := translateCommandToNamespace(home+"/.local/bin/claude", layout)
			if err == nil {
				t.Fatalf("expected error for state=%s, got nil", state)
			}
			var typed *ErrCommandRequiresHomeOverlay
			if !errors.As(err, &typed) {
				t.Fatalf("expected ErrCommandRequiresHomeOverlay, got %T", err)
			}
			if typed.Code() != "SANDBOX_HOME_OVERLAY_UNAVAILABLE" {
				t.Errorf("Code()=%q; want SANDBOX_HOME_OVERLAY_UNAVAILABLE", typed.Code())
			}
		})
	}
}

// TestSandboxLauncher_LaunchPreservesEnvShimArgs verifies the realistic
// runner shape: claude_code/codex/opencode go through
// BuildEnvWrappedLaunchRequest, which sets Command="env" and stuffs the
// host-absolute binary path into Args[1] (after a TAG=value env-var
// assignment in Args[0]). The runtime profile binds $HOME/.local/bin at
// the *host path* inside the namespace (the profile's dst mapping is
// not honored by buildBwrapArgs), so the host-absolute binary path is
// already valid inside the sandbox — the launcher must NOT rewrite it.
// This guards against a regression where a too-aggressive rewrite mapped
// $HOME/.local/bin/X to /usr/local/bin/X, which doesn't exist inside the
// sandbox under the profile-based isolation path.
func TestSandboxLauncher_LaunchPreservesEnvShimArgs(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-envshim/merged"

	mock := newMockSandbox(910)
	mock.hostMergedDir = host
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	t.Setenv("HOME", "/home/testuser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command: "env",
		Args: []string{
			"CLAUDE_CODE_AGENT_TAG=run-12345",
			"/home/testuser/.local/bin/claude",
			"--print",
			"--output-format=stream-json",
		},
		WorkingDir: host,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()

	if got, _ := body["command"].(string); got != "env" {
		t.Errorf("command = %q; want %q (basename passes through)", got, "env")
	}

	argsAny, _ := body["args"].([]any)
	want := []string{
		"CLAUDE_CODE_AGENT_TAG=run-12345",  // tag arg untouched
		"/home/testuser/.local/bin/claude", // binary path PRESERVED — profile binds at host path
		"--print",                          // flag untouched
		"--output-format=stream-json",      // flag untouched
	}
	if len(argsAny) != len(want) {
		t.Fatalf("args length = %d; want %d (args=%v)", len(argsAny), len(want), argsAny)
	}
	for i, w := range want {
		if got, _ := argsAny[i].(string); got != w {
			t.Errorf("args[%d] = %q; want %q", i, got, w)
		}
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestSandboxLauncher_LaunchBasenameFallback verifies the basename
// fallback path: a host-absolute binary path that doesn't match any
// known sandbox mount (e.g. /opt/homebrew/bin/X on macOS) gets stripped
// to its basename so the sandbox PATH lookup can find it.
func TestSandboxLauncher_LaunchBasenameFallback(t *testing.T) {
	const host = "/var/lib/workspace-sandbox/sb-cmd/merged"

	mock := newMockSandbox(909)
	mock.hostMergedDir = host
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	// $HOME doesn't matter here — the test path is /opt/* which has no
	// known sandbox mapping, so it falls through to the basename rule.
	t.Setenv("HOME", "/home/testuser")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "/opt/homebrew/bin/claude",
		Args:       []string{"--version"},
		WorkingDir: host,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())

	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	if got, _ := body["command"].(string); got != "claude" {
		t.Errorf("command = %q; want %q (unknown host-absolute path must basename-fallback)", got, "claude")
	}
	// Args must be left untouched — they're typically data, not paths.
	if argsAny, _ := body["args"].([]any); len(argsAny) != 1 || argsAny[0] != "--version" {
		t.Errorf("args = %v; want [--version] (must not be rewritten)", argsAny)
	}

	go func() {
		time.Sleep(40 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	_ = proc.Wait()
}

// TestSandboxLauncher_LaunchRejectsUntranslatableHostPath verifies that a
// WorkingDir that is neither under the sandbox nor under
// SandboxNamespacePath is rejected as a contract violation, not silently
// passed through.
func TestSandboxLauncher_LaunchRejectsUntranslatableHostPath(t *testing.T) {
	mock := newMockSandbox(808)
	mock.hostMergedDir = "/var/lib/workspace-sandbox/sb-test/merged"
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:    "claude",
		WorkingDir: "/etc",
	})
	if err == nil {
		t.Fatal("Launch returned nil; want *LaunchBlocked")
	}
	var blocked *LaunchBlocked
	if !errors.As(err, &blocked) {
		t.Fatalf("err = %T (%v); want *LaunchBlocked", err, err)
	}
	if blocked.Code != "workdir_outside_sandbox" {
		t.Errorf("blocked.Code = %q; want workdir_outside_sandbox", blocked.Code)
	}
	if mock.startProcessSeen.Load() {
		t.Error("startProcess should NOT have been called for an untranslatable workdir")
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
