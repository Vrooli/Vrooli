package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/pty"
	"web-console/internal/ptyfake"
)

// stdinCapture wires a FakePTY whose stdin writes are forwarded over a channel
// so a test can observe exactly what the server writes into a new pane. A
// concurrent reader is mandatory: FakePTY.WriteInput writes to an io.Pipe,
// which blocks until the bytes are read, so without draining stdin the Create
// call (which writes synchronously) would hang.
func stdinCapture(t *testing.T) (*ptyfake.FakePTY, <-chan []byte) {
	t.Helper()
	stdoutR, stdoutW := io.Pipe()
	stdinR, stdinW := io.Pipe()
	fake := &ptyfake.FakePTY{StdoutReader: stdoutR, StdinWriter: stdinW}
	inputCh := make(chan []byte, 16)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdinR.Read(buf)
			if n > 0 {
				b := make([]byte, n)
				copy(b, buf[:n])
				inputCh <- b
			}
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		_ = stdoutW.Close()
		_ = fake.Close()
	})
	return fake, inputCh
}

// A pasted newline is text in Bash's bracketed-paste mode. Observe a file
// written by the shell; an echoed command in the pane does not prove execution.
func TestCreateSession_ExecuteLaunchCommandRunsOnceInBash(t *testing.T) {
	requireTmux(t)
	dir := t.TempDir()
	marker := filepath.Join(dir, "executed")
	rc := filepath.Join(dir, "bashrc")
	if err := os.WriteFile(rc, []byte("bind 'set enable-bracketed-paste on'\nPS1='WC_LAUNCH_READY> '\n"), 0600); err != nil {
		t.Fatal(err)
	}
	srv := newFakeTestServerWithFactory(func(spec pty.LaunchSpec) (pty.PTY, error) {
		spec.Shell = fmt.Sprintf("/bin/bash --noprofile --rcfile %q -i", rc)
		p, err := tmuxPTYFactory(spec)
		if err == nil && !waitForPaneContent(t, tmuxSessionPrefix+spec.SessionID, "WC_LAUNCH_READY>", 3*time.Second) {
			_ = p.Kill()
			_ = p.Close()
			return nil, fmt.Errorf("bash prompt did not become ready")
		}
		return p, err
	})
	handler := newSessionsConnectHandlerForServer(srv)
	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Cols: 80, Rows: 24, LaunchCommand: fmt.Sprintf("printf x >> %q", marker),
		ExecuteLaunchCommand: true,
	})
	req.Header().Set("X-Idempotency-Key", "launch-once")
	resp, err := handler.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	sess := resp.Msg.GetSession()
	t.Cleanup(func() { _ = srv.sessions.Delete(context.Background(), sess.GetId()) })
	deadline := time.Now().Add(3 * time.Second)
	for {
		contents, err := os.ReadFile(marker)
		if err == nil {
			if string(contents) != "x" {
				t.Fatalf("command did not execute exactly once: %q", contents)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("launch did not execute without extra terminal input: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	repeated, err := handler.Create(context.Background(), req)
	if err != nil || repeated.Msg.GetSession().GetId() != sess.GetId() {
		t.Fatalf("repeated create did not return the same session: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	contents, err := os.ReadFile(marker)
	if err != nil || string(contents) != "x" {
		t.Fatalf("command executed more than once: %q, %v", contents, err)
	}
}

// waitForPTYInput accumulates stdin chunks until they contain want or the
// timeout elapses, returning everything seen so the failure message is useful.
func waitForPTYInput(t *testing.T, ch <-chan []byte, want string, timeout time.Duration) (string, bool) {
	t.Helper()
	var got strings.Builder
	deadline := time.After(timeout)
	for {
		select {
		case b := <-ch:
			got.Write(b)
			if strings.Contains(got.String(), want) {
				return got.String(), true
			}
		case <-deadline:
			return got.String(), false
		}
	}
}

func createWithLaunch(t *testing.T, srv *Server, launch string, execute bool) (*sessionsv1.Session, error) {
	t.Helper()
	req := connect.NewRequest(&sessionsv1.CreateRequest{
		Cols:                 80,
		Rows:                 24,
		LaunchCommand:        launch,
		ExecuteLaunchCommand: execute,
	})
	resp, err := newSessionsConnectHandlerForServer(srv).Create(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetSession(), nil
}

// TestCreateSession_ExecuteLaunchCommand asserts the server types the launch
// command (with a trailing newline so it runs) into the fresh PTY exactly once
// when execute_launch_command is set.
func TestCreateSession_ExecuteLaunchCommand(t *testing.T) {
	fake, inputCh := stdinCapture(t)
	srv := newFakeTestServer()
	srv.sessions = newSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := createWithLaunch(t, srv, "codex --yolo", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(context.Background(), sess.GetId()) })

	got, ok := waitForPTYInput(t, inputCh, "codex --yolo\n", time.Second)
	if !ok {
		t.Fatalf("launch command not written to PTY; saw %q", got)
	}
	if strings.Count(got, "codex --yolo") != 1 {
		t.Errorf("launch command should be written exactly once, saw %q", got)
	}
}

// TestCreateSession_NoLaunchExecutionWhenUnset asserts that a create without
// the execute flag writes nothing to the PTY, even when a launch_command is
// carried for provenance/recovery metadata.
func TestCreateSession_NoLaunchExecutionWhenUnset(t *testing.T) {
	fake, inputCh := stdinCapture(t)
	srv := newFakeTestServer()
	srv.sessions = newSessionManagerWithFactory(ptyfake.Factory(fake))

	sess, err := createWithLaunch(t, srv, "codex --yolo", false)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { _ = srv.sessions.Delete(context.Background(), sess.GetId()) })

	select {
	case b := <-inputCh:
		t.Fatalf("expected no PTY input on create without execute flag, got %q", string(b))
	case <-time.After(200 * time.Millisecond):
	}
}
