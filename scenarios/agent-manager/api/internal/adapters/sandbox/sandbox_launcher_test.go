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
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSandboxLauncherPolicyFiles(t *testing.T) {
	for _, supported := range []bool{false, true} {
		t.Run(fmt.Sprintf("supported=%v", supported), func(t *testing.T) {
			mock := newSandboxTestServer(707)
			mock.hostMergedDir = "/sandbox/merged"
			mock.policyFilesSupported = supported
			server := mock.startServer(t)
			defer server.Close()
			launcher := NewSandboxLauncher(NewWorkspaceSandboxProvider(server.URL), uuid.New())
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			file := runner.PolicyFile{Source: "/owner/policy", Target: "/etc/consumer/policy", SHA256: strings.Repeat("a", 64)}
			proc, err := launcher.Launch(ctx, runner.LaunchRequest{Command: "agent", WorkingDir: mock.hostMergedDir, PolicyFiles: []runner.PolicyFile{file}})
			if !supported {
				if err == nil || proc != nil || mock.startProcessSeen.Load() {
					t.Fatalf("unsupported provider must refuse before process creation: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			go io.Copy(io.Discard, proc.Stdout())
			go io.Copy(io.Discard, proc.Stderr())
			defer proc.Kill()
			mock.mu.Lock()
			data, err := json.Marshal(mock.startProcessBody["policyFiles"])
			mock.mu.Unlock()
			var actual []runner.PolicyFile
			if err != nil || json.Unmarshal(data, &actual) != nil || len(actual) != 1 || actual[0] != file {
				t.Fatalf("policy must reach the provider without rewriting its identity: %s, %v", data, err)
			}
		})
	}
}

// TestSandboxLauncher_LaunchAndStreamLog drives the happy path: launch a
// process, push stdout chunks via SSE, mark exited, verify Stdout receives
// all the bytes and Wait returns nil.
func TestSandboxLauncher_LaunchAndStreamLog(t *testing.T) {
	mock := newSandboxTestServer(99)
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

	// Publish only after both SSE subscriptions are registered.
	mock.waitForStreams(t)
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
	mock := newSandboxTestServer(101)
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

	// Cleanup only after both streams have registered.
	mock.waitForStreams(t)
	mock.markExited(remoteExitInfo{ExitCode: 0})
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())
	_ = proc.Wait()
}

func TestLaunchNetworkOverrideHonorsExplicitDeny(t *testing.T) {
	got := launchNetworkOverride(string(domain.NetworkAccessNone))
	if got == nil || *got {
		t.Fatalf("network none override = %v, want explicit false", got)
	}
	if got := launchNetworkOverride(string(domain.NetworkAccessLocalhost)); got == nil || !*got {
		t.Fatalf("localhost override = %v, want explicit true", got)
	}
	if got := launchNetworkOverride(""); got != nil {
		t.Fatalf("legacy empty mode override = %v, want nil", got)
	}
}

func TestSandboxLauncher_PostsExplicitNetworkDeny(t *testing.T) {
	mock := newSandboxTestServer(103)
	server := mock.startServer(t)
	defer server.Close()

	provider := NewWorkspaceSandboxProvider(server.URL)
	launcher := NewSandboxLauncher(provider, uuid.New())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	proc, err := launcher.Launch(ctx, runner.LaunchRequest{
		Command:     "codex",
		NetworkMode: string(domain.NetworkAccessNone),
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	mock.mu.Lock()
	body := mock.startProcessBody
	mock.mu.Unlock()
	allow, ok := body["allowNetwork"].(bool)
	if !ok || allow {
		t.Fatalf("allowNetwork = %#v, want explicit false", body["allowNetwork"])
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		mock.markExited(remoteExitInfo{ExitCode: 0})
	}()
	go io.Copy(io.Discard, proc.Stdout())
	go io.Copy(io.Discard, proc.Stderr())
	_ = proc.Wait()
}

func TestRunRuntimeMounts_DeclaresSingleRunRuntimeFolder(t *testing.T) {
	mounts := runRuntimeMounts(map[string]string{
		"VROOLI_AGENT_RUNTIME_ROOT": "/state/run/runtime",
		"CODEX_HOME":                "/state/run/runtime/codex",
		"UNRELATED":                 "/state/unrelated",
	})
	if len(mounts) != 1 {
		t.Fatalf("mount count = %d, want 1 (the run runtime folder)", len(mounts))
	}
	if mounts[0].Path != "/state/run/runtime" || mounts[0].Purpose != "run-runtime" {
		t.Errorf("mount = %+v, want the single run runtime folder", mounts[0])
	}
	if len(runRuntimeMounts(map[string]string{"CODEX_HOME": "/state/run/runtime/codex"})) != 0 {
		t.Errorf("without the run runtime root no mount may be declared")
	}
}

// TestSandboxLauncher_KillReturnsThroughDelete verifies Kill issues a
// DELETE and Wait unblocks promptly.
func TestSandboxLauncher_KillReturnsThroughDelete(t *testing.T) {
	mock := newSandboxTestServer(202)
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
	mock := newSandboxTestServer(303)
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
	mock := newSandboxTestServer(404)
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
	mock := newSandboxTestServer(505)
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

	mock.waitForStreams(t)
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
	mock := newSandboxTestServer(606)
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

	mock.waitForStreams(t)
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
	mock := newSandboxTestServer(909)
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
	mock.waitForStreams(t)
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

func TestSandboxLauncher_ReconcilesExitInfoAfterStreamDisconnect(t *testing.T) {
	mock := newSandboxTestServer(910)
	mock.hostMergedDir = "/var/lib/workspace-sandbox/sb-reconcile/merged"
	mock.listProcessExit = &remoteExitInfo{ExitCode: 23, Signal: 0}
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

	mock.waitForStreams(t)
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
	var exitErr *remoteExitError
	if !errors.As(werr, &exitErr) {
		t.Fatalf("Wait error = %T (%v); want reconciled remote exit", werr, werr)
	}
	if exitErr.ExitCode() != 23 {
		t.Fatalf("reconciled exit code = %d; want 23", exitErr.ExitCode())
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

	mock := newSandboxTestServer(911)
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
