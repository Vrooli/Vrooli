package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// wsURL converts an httptest.Server URL to a WebSocket URL.
func wsURL(s *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(s.URL, "http") + path
}

// setupWSServer creates a test server with routes registered for WebSocket testing.
func setupWSServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	srv := newFakeTestServer()
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)
	return ts, srv
}

// createTestSession creates a session via the API and returns its ID.
func createTestSession(t *testing.T, ts *httptest.Server) string {
	t.Helper()
	resp, err := http.Post(ts.URL+"/api/v1/sessions", "application/json",
		strings.NewReader(`{"cols":80,"rows":24}`))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var sr SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return sr.ID
}

// [REQ:P0-002b] WebSocket I/O Streaming - session not found returns 404
func TestHandleTerminalWS_SessionNotFound(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/v1/sessions/nonexistent/ws", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "nonexistent"})
	rec := httptest.NewRecorder()

	srv.handleTerminalWS(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing session, got %d", rec.Code)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - exited session auto-removed returns 404
func TestHandleTerminalWS_ExitedSession(t *testing.T) {
	deadFake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(deadFake))
	deadSrv := &Server{
		router:    mux.NewRouter(),
		sessions:  sm,
		events:    NewEventLogger(100),
		metrics:   NewMetrics(),
		aiChain:   NewAIProviderChain(),
		shortcuts: NewShortcutProfileStore(),
		aiConfig:  NewAIProviderConfigStore(),
	}
	deadSess, _ := sm.Create("/fake/shell", 80, 24)
	sessID := deadSess.ID

	// Close output pipe to simulate process exit; auto-removal cleans up the map
	deadFake.outW.Close()
	time.Sleep(150 * time.Millisecond)

	req := httptest.NewRequest("GET", "/api/v1/sessions/"+sessID+"/ws", nil)
	req = mux.SetURLVars(req, map[string]string{"id": sessID})
	rec := httptest.NewRecorder()

	deadSrv.handleTerminalWS(rec, req)

	// After auto-removal, session is no longer in the map → 404
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for auto-removed dead session, got %d", rec.Code)
	}
}

// skipHistoryEnd reads and discards the initial history_end message that
// the server sends on every fresh connection. Tests that care about
// subsequent messages should call this first.
func skipHistoryEnd(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg TerminalMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("skipHistoryEnd: read failed: %v", err)
	}
	if msg.Type != MsgTypeHistoryEnd {
		t.Fatalf("skipHistoryEnd: expected history_end, got %s", msg.Type)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - successful WS upgrade and ping/pong
func TestHandleTerminalWS_PingPong(t *testing.T) {
	ts, _ := setupWSServer(t)
	sessID := createTestSession(t, ts)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	skipHistoryEnd(t, conn)

	// Send ping
	ping := TerminalMessage{Type: MsgTypePing}
	if err := conn.WriteJSON(ping); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	// Expect pong
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var pong TerminalMessage
	if err := conn.ReadJSON(&pong); err != nil {
		t.Fatalf("read pong: %v", err)
	}
	if pong.Type != MsgTypePong {
		t.Errorf("expected pong message, got type=%s", pong.Type)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - stdin writes succeed without error
func TestHandleTerminalWS_Stdin(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	beforeReceived := srv.metrics.WSMessagesReceived.Load()

	// Send stdin data - should not cause an error
	msg := TerminalMessage{Type: MsgTypeStdin, Data: "test input"}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("write stdin: %v", err)
	}

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	afterReceived := srv.metrics.WSMessagesReceived.Load()
	if afterReceived <= beforeReceived {
		t.Errorf("WSMessagesReceived should increment: before=%d after=%d", beforeReceived, afterReceived)
	}
}

// [REQ:P0-002c] Terminal Resize Handling - resize via WebSocket
func TestHandleTerminalWS_Resize(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Send resize
	msg := TerminalMessage{Type: MsgTypeResize, Cols: 120, Rows: 40}
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("write resize: %v", err)
	}

	// Give the server time to process
	time.Sleep(100 * time.Millisecond)

	// Verify resize was applied
	sess, ok := srv.sessions.Get(sessID)
	if !ok {
		t.Fatal("session not found after resize")
	}
	if sess.Cols != 120 || sess.Rows != 40 {
		t.Errorf("expected 120x40, got %dx%d", sess.Cols, sess.Rows)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - invalid JSON returns error message
func TestHandleTerminalWS_InvalidJSON(t *testing.T) {
	ts, _ := setupWSServer(t)
	sessID := createTestSession(t, ts)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	skipHistoryEnd(t, conn)

	// Send invalid JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatalf("write invalid: %v", err)
	}

	// Should get error message back
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var resp TerminalMessage
	if err := conn.ReadJSON(&resp); err != nil {
		t.Fatalf("read error: %v", err)
	}
	if resp.Type != MsgTypeError {
		t.Errorf("expected error message, got type=%s", resp.Type)
	}
}

// [REQ:P1-004a] Metrics - WebSocket connections increment metrics
func TestHandleTerminalWS_MetricsIncrement(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts)

	before := srv.metrics.ConnectionsTotal.Load()

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}

	// Give server time to process connection
	time.Sleep(50 * time.Millisecond)

	after := srv.metrics.ConnectionsTotal.Load()
	if after <= before {
		t.Errorf("ConnectionsTotal should increment: before=%d after=%d", before, after)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)
}

// [REQ:P0-002b] WebSocket message type constants are defined
func TestTerminalMessageTypes(t *testing.T) {
	types := map[string]string{
		"stdin":  MsgTypeStdin,
		"stdout": MsgTypeStdout,
		"resize": MsgTypeResize,
		"exit":   MsgTypeExit,
		"error":  MsgTypeError,
		"ping":   MsgTypePing,
		"pong":   MsgTypePong,
	}
	for expected, actual := range types {
		if actual != expected {
			t.Errorf("MsgType constant mismatch: expected %q, got %q", expected, actual)
		}
	}
}

// [REQ:P0-002b] TerminalMessage JSON serialization
func TestTerminalMessage_JSON(t *testing.T) {
	msg := TerminalMessage{Type: MsgTypeStdout, Data: "hello"}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TerminalMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Type != MsgTypeStdout || decoded.Data != "hello" {
		t.Errorf("round-trip mismatch: %+v", decoded)
	}
}

// [REQ:P0-002c] TerminalMessage resize fields
func TestTerminalMessage_ResizeFields(t *testing.T) {
	msg := TerminalMessage{Type: MsgTypeResize, Cols: 120, Rows: 40}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded TerminalMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Cols != 120 || decoded.Rows != 40 {
		t.Errorf("resize fields mismatch: cols=%d rows=%d", decoded.Cols, decoded.Rows)
	}
}

// --- Goroutine lifecycle tests ---

// TestHandleTerminalWS_ForwarderExitsOnClientClose verifies that the output
// forwarder goroutine exits when the WebSocket client disconnects, preventing
// goroutine leaks. The session should remain alive (PTY not killed).
func TestHandleTerminalWS_ForwarderExitsOnClientClose(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}

	// Give server time to start forwarder goroutine.
	time.Sleep(50 * time.Millisecond)

	// Close the WebSocket from the client side.
	conn.Close()
	time.Sleep(500 * time.Millisecond)

	// Session should still be alive (PTY not killed by WS disconnect).
	sess, ok := srv.sessions.Get(sessID)
	if !ok {
		t.Fatal("session should still exist after WS disconnect")
	}
	if sess.IsDead() {
		t.Error("session should not be dead after WS disconnect")
	}
}

// TestHandleTerminalWS_RepeatedConnectDisconnect verifies that repeatedly
// connecting and disconnecting WebSocket clients does not leak goroutines.
func TestHandleTerminalWS_RepeatedConnectDisconnect(t *testing.T) {
	ts, _ := setupWSServer(t)
	sessID := createTestSession(t, ts)

	// Let any startup goroutines settle.
	time.Sleep(100 * time.Millisecond)
	runtime.GC()
	baseline := runtime.NumGoroutine()

	for i := 0; i < 5; i++ {
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
		if err != nil {
			t.Fatalf("ws dial #%d: %v", i, err)
		}
		time.Sleep(30 * time.Millisecond)
		conn.Close()
		time.Sleep(100 * time.Millisecond)
	}

	// Allow goroutines to settle.
	time.Sleep(500 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()

	// Allow small jitter (±3) for GC/runtime goroutines.
	delta := after - baseline
	if delta > 3 {
		t.Errorf("goroutine leak: baseline=%d, after=%d, delta=%d (expected ≤3)", baseline, after, delta)
	}
}

// --- history_end message tests ---

// setupWSServerWithPTY creates a test server whose single session uses a
// controllable fake PTY. Returns the server, session ID, and the fake PTY
// (so the test can inject output to build history).
func setupWSServerWithPTY(t *testing.T) (*httptest.Server, string, *fakePTYWithOutput) {
	t.Helper()
	fake := newFakePTYWithOutput()
	sm := NewSessionManagerWithFactory(fakePTYFactory(fake))
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    sm,
		events:      NewEventLogger(100),
		metrics:     NewMetrics(),
		aiChain:     NewAIProviderChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    NewAIProviderConfigStore(),
		idempotency: newIdempotencyCache(),
	}
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)

	sess, err := sm.Create("/fake/shell", 80, 24)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return ts, sess.ID, fake
}

// readTerminalMessage reads a single JSON message with a deadline.
func readTerminalMessage(t *testing.T, conn *websocket.Conn) TerminalMessage {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var msg TerminalMessage
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("readTerminalMessage: %v", err)
	}
	return msg
}

func TestHandleTerminalWS_HistoryEnd_NoHistory(t *testing.T) {
	ts, sessID, fake := setupWSServerWithPTY(t)
	defer func() {
		fake.Close()
	}()

	// No output written — connect immediately.
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// The very first server message should be history_end (no history to send).
	msg := readTerminalMessage(t, conn)
	if msg.Type != MsgTypeHistoryEnd {
		t.Errorf("expected first message type=%q, got %q (data=%q)", MsgTypeHistoryEnd, msg.Type, msg.Data)
	}
}

func TestHandleTerminalWS_HistoryEnd_AfterHistory(t *testing.T) {
	ts, sessID, fake := setupWSServerWithPTY(t)

	// Write output so history is built up.
	_, _ = fake.outW.Write([]byte("line 1\r\nline 2\r\n"))
	time.Sleep(100 * time.Millisecond)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	// Read messages until we see history_end.
	var stdoutCount int
	for i := 0; i < 20; i++ {
		msg := readTerminalMessage(t, conn)
		switch msg.Type {
		case MsgTypeStdout:
			stdoutCount++
		case MsgTypeHistoryEnd:
			if stdoutCount == 0 {
				t.Error("expected at least one stdout chunk before history_end")
			}
			// Success — history_end arrived after stdout chunks.
			fake.Close()
			return
		default:
			// Ignore other message types (e.g., resize_info).
		}
	}
	t.Fatal("never received history_end message")
}
