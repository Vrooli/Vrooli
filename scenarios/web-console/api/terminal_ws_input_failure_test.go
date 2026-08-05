package main

// terminal_ws_input_failure_test.go: what a backend write failure does
// to the connection.
//
// The distinction is load-bearing. A dead PTY means the session is gone
// and the socket should close. A backend that rejected one payload does
// not — and closing anyway made such failures self-sustaining: the
// client re-enqueues in-flight payloads on close (useStdinAck's
// handleClose), reconnects, flushes the same payload, and gets closed
// again. One oversized paste became an endless reconnect loop.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"web-console/internal/pty"
	"web-console/internal/ptyfake"

	"github.com/gorilla/websocket"
)

// dialSessionWS opens the terminal WebSocket for sessionID and reads
// frames until session_ready, so the caller can send stdin immediately.
func dialSessionWS(t *testing.T, ts *httptest.Server, sessionID string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessionID+"/ws"), nil)
	if err != nil {
		t.Fatalf("dial terminal websocket: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("waiting for session_ready: %v", err)
		}
		if msg.Type == MsgTypeSessionReady {
			return conn
		}
	}
}

// readUntilType reads frames until one of wantType arrives, ignoring the
// output and keepalive traffic that shares the socket.
func readUntilType(t *testing.T, conn *websocket.Conn, wantType string) TerminalMessage {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("waiting for %s frame: %v", wantType, err)
		}
		if msg.Type == wantType {
			return msg
		}
	}
}

// serverWithFailingPTY builds a test server whose sessions all use a
// fake PTY that fails writes with writeErr, and returns the server plus
// a live session ID.
func serverWithFailingPTY(t *testing.T, writeErr error) (*httptest.Server, string) {
	t.Helper()

	fake := ptyfake.NewFakePTYWithOutput()
	srv := newFakeTestServerWithFactory(func(pty.LaunchSpec) (pty.PTY, error) {
		return fake, nil
	})
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)

	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	fake.SetWriteInputErr(writeErr)
	return ts, sess.ID
}

// TestDispatchInput_PayloadRejection_KeepsConnectionOpen asserts that a
// backend write failure which is not a dead PTY is reported on the ack
// and nowhere else — the socket stays usable for the next keystroke.
func TestDispatchInput_PayloadRejection_KeepsConnectionOpen(t *testing.T) {
	ts, sessionID := serverWithFailingPTY(t, errors.New("tmux send-keys failed: command too long"))
	conn := dialSessionWS(t, ts, sessionID)

	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "some payload", Seq: 1}); err != nil {
		t.Fatalf("send stdin: %v", err)
	}

	ack := readUntilType(t, conn, MsgTypeStdinAck)
	if ack.Ok {
		t.Error("stdin_ack reported success for a payload the backend rejected")
	}
	if ack.Reason != StdinAckReasonTmuxWriteFailed {
		t.Errorf("stdin_ack.reason = %q, want %q", ack.Reason, StdinAckReasonTmuxWriteFailed)
	}
	if ack.Seq != 1 {
		t.Errorf("stdin_ack.seq = %d, want 1", ack.Seq)
	}

	// The real assertion: the connection survived. A ping round-trip
	// proves the server is still dispatching on this socket rather than
	// having torn it down behind the ack.
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypePing}); err != nil {
		t.Fatalf("connection was closed after a rejected payload: %v", err)
	}
	if got := readUntilType(t, conn, MsgTypePong); got.Type != MsgTypePong {
		t.Fatalf("no pong after a rejected payload; got %q", got.Type)
	}
}

// TestDispatchInput_DeadPTY_ClosesConnection is the other half: when the
// PTY is genuinely gone the session cannot recover, so the server must
// still close rather than leave the client acking into a void.
func TestDispatchInput_DeadPTY_ClosesConnection(t *testing.T) {
	ts, sessionID := serverWithFailingPTY(t, errPTYClosed)
	conn := dialSessionWS(t, ts, sessionID)

	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "some payload", Seq: 1}); err != nil {
		t.Fatalf("send stdin: %v", err)
	}

	ack := readUntilType(t, conn, MsgTypeStdinAck)
	if ack.Ok {
		t.Error("stdin_ack reported success against a closed PTY")
	}
	if ack.Reason != StdinAckReasonPTYClosed {
		t.Errorf("stdin_ack.reason = %q, want %q", ack.Reason, StdinAckReasonPTYClosed)
	}

	// Drain until the server hangs up. Reading past the close yields an
	// error, which is the outcome under test.
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		var msg json.RawMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return // closed, as expected
		}
	}
}
