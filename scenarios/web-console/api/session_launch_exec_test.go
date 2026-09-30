package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"

	"web-console/internal/ptyfake"
)

// stdinCapture wires a FakePTY whose stdin writes are forwarded over a channel
// so a test can observe exactly what the server pastes into a new pane. A
// concurrent reader is mandatory: FakePTY.WriteInput writes to an io.Pipe,
// which blocks until the bytes are read, so without draining stdin the Create
// call (which pastes synchronously) would hang.
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

// TestCreateSession_ExecuteLaunchCommand asserts the server pastes the launch
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
		t.Fatalf("launch command not pasted into PTY; saw %q", got)
	}
	if strings.Count(got, "codex --yolo") != 1 {
		t.Errorf("launch command should be pasted exactly once, saw %q", got)
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
