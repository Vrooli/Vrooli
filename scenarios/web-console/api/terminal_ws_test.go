package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/pty"
	"web-console/internal/ptyfake"
	"web-console/internal/wireproto"
	"web-console/terminal"

	intai "web-console/internal/ai"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	intsessions "web-console/internal/sessions"
	intworkspace "web-console/internal/workspace"
)

// Test fixtures use short local names to keep protocol assertions readable.
// Production code references wireproto directly; these are not part of the
// package's runtime API.
const (
	MsgTypeStdin                  = wireproto.MsgTypeStdin
	MsgTypeStdout                 = wireproto.MsgTypeStdout
	MsgTypeResize                 = wireproto.MsgTypeResize
	MsgTypeResizeInfo             = wireproto.MsgTypeResizeInfo
	MsgTypeSizeInfo               = wireproto.MsgTypeSizeInfo
	MsgTypeTakeLease              = wireproto.MsgTypeTakeLease
	MsgTypeExit                   = wireproto.MsgTypeExit
	MsgTypeError                  = wireproto.MsgTypeError
	MsgTypePing                   = wireproto.MsgTypePing
	MsgTypePong                   = wireproto.MsgTypePong
	MsgTypeSyncWarning            = wireproto.MsgTypeSyncWarning
	MsgTypeHistoryEnd             = wireproto.MsgTypeHistoryEnd
	MsgTypeConversationAck        = wireproto.MsgTypeConversationAck
	MsgTypeSessionReady           = wireproto.MsgTypeSessionReady
	MsgTypeStdinAck               = wireproto.MsgTypeStdinAck
	MsgTypeControl                = wireproto.MsgTypeControl
	MsgTypeHello                  = wireproto.MsgTypeHello
	MsgTypeResync                 = wireproto.MsgTypeResync
	MsgTypeSnapshotNotice         = wireproto.MsgTypeSnapshotNotice
	MsgTypeEchoState              = wireproto.MsgTypeEchoState
	MsgTypeMouseMode              = wireproto.MsgTypeMouseMode
	MsgTypeScroll                 = wireproto.MsgTypeScroll
	MsgTypePresence               = wireproto.MsgTypePresence
	MsgTypeDeviceState            = wireproto.MsgTypeDeviceState
	StdinIntentTyping             = wireproto.StdinIntentTyping
	StdinIntentBulkText           = wireproto.StdinIntentBulkText
	StdinIntentNamedKey           = wireproto.StdinIntentNamedKey
	StdinAckReasonTmuxWriteFailed = wireproto.StdinAckReasonTmuxFailed
	StdinAckReasonPTYClosed       = wireproto.StdinAckReasonPTYClosed
	StdinAckReasonOffsetGap       = wireproto.StdinAckReasonOffsetGap
	StdinAckReasonUnreconcilable  = wireproto.StdinAckReasonUnreconcilable
	StdinAckReasonQueueFull       = wireproto.StdinAckReasonQueueFull
	ProtocolVersion               = wireproto.ProtocolVersion
)

// wsURL converts an httptest.Server URL to a WebSocket URL.
func wsURL(s *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(s.URL, "http") + path
}

func TestBoundSnapshotKeepsNewestCompleteLines(t *testing.T) {
	got, dropped, truncated := boundSnapshot([]byte("old-1\nold-2\nold-3\nold-4\nnew-1\nnew-2\n"), len(terminal.SnapshotPrologue)+12)
	if !truncated {
		t.Fatal("expected snapshot to be truncated")
	}
	if dropped != 4 {
		t.Fatalf("dropped lines = %d, want 4", dropped)
	}
	if string(got) != terminal.SnapshotPrologue+"new-1\nnew-2\n" {
		t.Fatalf("bounded snapshot = %q", got)
	}
	if strings.Contains(string(got), "old-") {
		t.Fatalf("bounded snapshot retained old content: %q", got)
	}
}

func TestBoundSnapshotDoesNotEmitPartialLine(t *testing.T) {
	got, dropped, truncated := boundSnapshot([]byte("a very long line without a terminator"), 8)
	if !truncated || dropped != 0 {
		t.Fatalf("truncation = %v dropped = %d", truncated, dropped)
	}
	if string(got) != terminal.SnapshotPrologue {
		t.Fatalf("partial line was emitted: %q", got)
	}
}

func TestBoundSnapshotRespectsTotalByteCap(t *testing.T) {
	got, _, truncated := boundSnapshot([]byte("old line\nolder line\nancient line\nnew line\n"), len(terminal.SnapshotPrologue)+9)
	if !truncated {
		t.Fatal("expected snapshot to be truncated")
	}
	if len(got) > len(terminal.SnapshotPrologue)+9 {
		t.Fatalf("bounded snapshot length = %d, want <= cap: %q", len(got), got)
	}
	if !strings.HasSuffix(string(got), "new line\n") {
		t.Fatalf("bounded snapshot lost the newest complete line: %q", got)
	}
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

// createTestSession creates a session via the in-memory manager and
// returns its ID. The terminal WS still lives on the legacy REST path
// while the Connect-RPC SessionsService owns everything else, so we
// bypass the wire and call the manager directly.
func createTestSession(t *testing.T, ts *httptest.Server, srv *Server) string {
	t.Helper()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return sess.ID
}

// The API serves terminal sockets through Server.Handler in production, not
// directly through the mux. Keep this path covered: response-writer middleware
// must preserve http.Hijacker or gorilla/websocket cannot complete an upgrade.
func TestTerminalWS_FullHandlerStackSupportsWebSocketUpgrade(t *testing.T) {
	srv := newFakeTestServer()
	srv.setupRoutes()
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL(ts, "/api/v1/sessions/"+sess.ID+"/ws"), nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("dial terminal websocket through full handler: status=%d: %v", resp.StatusCode, err)
		}
		t.Fatalf("dial terminal websocket through full handler: %v", err)
	}
	defer conn.Close()
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
	deadFake := ptyfake.NewFakePTYWithOutput()
	sm := newSessionManagerWithFactory(ptyfake.Factory(deadFake))
	deadSrv := &Server{
		router:    mux.NewRouter(),
		sessions:  sm,
		events:    events.NewLogger(100),
		metrics:   metrics.New(),
		aiChain:   intai.NewChain(),
		shortcuts: NewShortcutProfileStore(),
		aiConfig:  intai.NewMemConfigStore(),
		workspace: intworkspace.NewMemStore(),
	}
	deadSess, _ := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
	sessID := deadSess.ID

	// Close output pipe to simulate process exit; auto-removal cleans up the map
	deadFake.OutW.Close()
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

// skipHistoryEnd reads and discards the initial server-to-client handshake
// messages — history_end, pty_state (initial alt-buffer view), and
// session_ready — that every fresh connection produces. Early stdout
// from the shell (prompt render) may arrive between these and is
// transparently dropped so downstream reads can focus on the messages
// under test.
func skipHistoryEnd(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	sawHistoryEnd := false
	sawSessionReady := false
	sawSizeInfo := false
	deadline := time.Now().Add(3 * time.Second)
	for !(sawHistoryEnd && sawSessionReady && sawSizeInfo) {
		_ = conn.SetReadDeadline(deadline)
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("skipHistoryEnd: read failed: %v", err)
		}
		switch msg.Type {
		case MsgTypeHistoryEnd:
			sawHistoryEnd = true
		case MsgTypeSessionReady:
			sawSessionReady = true
		case MsgTypeSizeInfo:
			sawSizeInfo = true
		case MsgTypeStdout, MsgTypeResizeInfo, MsgTypeSyncWarning, MsgTypeEchoState:
			// Non-handshake traffic that may arrive before the handshake
			// completes (snapshot frames, early shell prompt).
		default:
			t.Fatalf("skipHistoryEnd: unexpected message type=%s before handshake complete", msg.Type)
		}
	}
}

// [REQ:P0-002c] Every connected viewer is told the one authoritative grid.
func TestTerminalWS_SecondClientReceivesSizeInfo(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessionID := createTestSession(t, ts, srv)
	url := wsURL(ts, "/api/v1/sessions/"+sessionID+"/ws")
	first, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial first: %v", err)
	}
	defer first.Close()
	skipHistoryEnd(t, first)
	second, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial second: %v", err)
	}
	defer second.Close()
	skipHistoryEnd(t, second)
	if err := first.WriteJSON(TerminalMessage{Type: MsgTypeResize, Cols: 120, Rows: 40}); err != nil {
		t.Fatalf("write resize: %v", err)
	}
	for _, conn := range []*websocket.Conn{first, second} {
		deadline := time.Now().Add(2 * time.Second)
		found := false
		for !found {
			_ = conn.SetReadDeadline(deadline)
			var msg TerminalMessage
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatalf("read size_info: %v", err)
			}
			if msg.Type == MsgTypeSizeInfo {
				found = msg.Cols == 120 && msg.Rows == 40
			}
		}
	}
	if session, ok := srv.sessions.Get(sessionID); !ok {
		t.Fatal("session missing")
	} else if cols, rows := session.EffectiveSize(); cols != 120 || rows != 40 {
		t.Fatalf("session size = %dx%d", cols, rows)
	}
}

// [REQ:P0-002e] A follower receives the leader's device class at connect and
// its keyboard state as it changes, so the device frame never has to infer
// either one from the shared grid.
func TestTerminalWS_FollowerReceivesLeaderDevicePresentation(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessionID := createTestSession(t, ts, srv)
	base := wsURL(ts, "/api/v1/sessions/"+sessionID+"/ws")

	leader, _, err := websocket.DefaultDialer.Dial(base+"?deviceId=phone-1&deviceLabel=iPhone&deviceClass=phone", nil)
	if err != nil {
		t.Fatalf("dial leader: %v", err)
	}
	defer leader.Close()
	skipHistoryEnd(t, leader)

	follower, _, err := websocket.DefaultDialer.Dial(base+"?deviceId=desk-1&deviceLabel=Desktop&deviceClass=monitor", nil)
	if err != nil {
		t.Fatalf("dial follower: %v", err)
	}
	defer follower.Close()
	skipHistoryEnd(t, follower)

	// awaitPresentation reads until a message carries leader presentation.
	awaitPresentation := func(conn *websocket.Conn, want func(TerminalMessage) bool) TerminalMessage {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for {
			_ = conn.SetReadDeadline(deadline)
			var msg TerminalMessage
			if err := conn.ReadJSON(&msg); err != nil {
				t.Fatalf("read presentation: %v", err)
			}
			if (msg.Type == MsgTypeSizeInfo || msg.Type == MsgTypePresence) && want(msg) {
				return msg
			}
		}
	}

	// The leader takes the lease, so its class must reach the follower.
	if err := leader.WriteJSON(TerminalMessage{Type: MsgTypeTakeLease}); err != nil {
		t.Fatalf("write take_lease: %v", err)
	}
	got := awaitPresentation(follower, func(m TerminalMessage) bool { return m.DeviceClass != "" })
	if got.DeviceClass != "phone" || got.LeaderDevice != "iPhone" {
		t.Fatalf("follower presentation = %+v, want phone/iPhone", got)
	}
	if got.KbOpen {
		t.Fatalf("keyboard reported open before it was declared: %+v", got)
	}

	// Opening the leader's keyboard reaches the follower as state, not as a
	// grid change.
	if err := leader.WriteJSON(TerminalMessage{Type: MsgTypeDeviceState, KbOpen: true}); err != nil {
		t.Fatalf("write device_state: %v", err)
	}
	if got := awaitPresentation(follower, func(m TerminalMessage) bool { return m.KbOpen }); got.DeviceClass != "phone" {
		t.Fatalf("keyboard-open presentation = %+v", got)
	}
}

// [REQ:P0-002c] A reconnecting follower declares its size but cannot steal it.
func TestTerminalWS_ReconnectDoesNotStealSize(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessionID := createTestSession(t, ts, srv)
	url := wsURL(ts, "/api/v1/sessions/"+sessionID+"/ws")
	leader, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer leader.Close()
	skipHistoryEnd(t, leader)
	if err := leader.WriteJSON(TerminalMessage{Type: MsgTypeResize, Cols: 160, Rows: 50}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	follower, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	skipHistoryEnd(t, follower)
	if err := follower.WriteJSON(TerminalMessage{Type: MsgTypeResize, Cols: 45, Rows: 30}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	session, _ := srv.sessions.Get(sessionID)
	if cols, rows := session.EffectiveSize(); cols != 160 || rows != 50 {
		t.Fatalf("follower stole size: %dx%d", cols, rows)
	}
	follower.Close()
}

// [REQ:P0-002d] An explicit takeover applies the requesting device's
// declaration and immediately reports the new lease to that same socket.
func TestTerminalWS_TakeLeaseUpdatesRequestingClient(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessionID := createTestSession(t, ts, srv)
	url := wsURL(ts, "/api/v1/sessions/"+sessionID+"/ws")
	leader, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer leader.Close()
	skipHistoryEnd(t, leader)
	if err := leader.WriteJSON(TerminalMessage{Type: MsgTypeResize, Cols: 160, Rows: 50}); err != nil {
		t.Fatal(err)
	}
	follower, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer follower.Close()
	skipHistoryEnd(t, follower)
	// These consecutive frames are intentional: browsers send their current
	// declaration before an explicit take-over request.
	if err := follower.WriteJSON(TerminalMessage{Type: MsgTypeResize, Cols: 45, Rows: 30}); err != nil {
		t.Fatal(err)
	}
	if err := follower.WriteJSON(TerminalMessage{Type: MsgTypeTakeLease}); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		_ = follower.SetReadDeadline(deadline)
		var msg TerminalMessage
		if err := follower.ReadJSON(&msg); err != nil {
			t.Fatalf("read takeover size_info: %v", err)
		}
		if msg.Type == MsgTypeSizeInfo && msg.Cols == 45 && msg.Rows == 30 && msg.HoldsLease {
			break
		}
	}
	if session, ok := srv.sessions.Get(sessionID); !ok {
		t.Fatal("session missing")
	} else if cols, rows := session.EffectiveSize(); cols != 45 || rows != 30 {
		t.Fatalf("takeover size = %dx%d, want 45x30", cols, rows)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - successful WS upgrade and ping/pong
func TestHandleTerminalWS_PingPong(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

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
	sessID := createTestSession(t, ts, srv)

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
	sessID := createTestSession(t, ts, srv)

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
	cols, rows := sess.EffectiveSize()
	if cols != 120 || rows != 40 {
		t.Errorf("expected 120x40, got %dx%d", cols, rows)
	}
}

// [REQ:P0-002b] WebSocket I/O Streaming - invalid JSON returns error message
func TestHandleTerminalWS_InvalidJSON(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

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

// Every stdin message must receive a matching stdin_ack carrying the
// cumulative accepted byte offset when the PTY write succeeds.
func TestHandleTerminalWS_StdinAck_Success(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

	conn, _, err := (&websocket.Dialer{}).Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	skipHistoryEnd(t, conn)

	for _, offset := range []int64{0, 1, 2} {
		if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "x", Offset: offset}); err != nil {
			t.Fatalf("write stdin offset=%d: %v", offset, err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var gotAck bool
		for !gotAck {
			var resp TerminalMessage
			if err := conn.ReadJSON(&resp); err != nil {
				t.Fatalf("read ack offset=%d: %v", offset, err)
			}
			switch resp.Type {
			case MsgTypeStdinAck:
				if resp.AcceptedThrough != offset+1 {
					t.Errorf("offset=%d: ack accepted_through=%d, want %d", offset, resp.AcceptedThrough, offset+1)
				}
				if !resp.Ok {
					t.Errorf("offset=%d: expected ok=true", offset)
				}
				gotAck = true
			case MsgTypeStdout, MsgTypeResizeInfo, MsgTypeSyncWarning, MsgTypePresence:
				// Shell prompt echo or similar — keep reading until ack.
			default:
				t.Fatalf("offset=%d: unexpected message type=%s", offset, resp.Type)
			}
		}
	}
}

// TestLiveRemoteSessionThroughWebConsole is an opt-in two-machine contract
// probe. It exercises the production Connect create path and the browser-facing
// terminal protocol against a running Web Console/Bridge pair without placing
// any credentials in the test or its output.
func TestLiveRemoteSessionThroughWebConsole(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WEB_CONSOLE_LIVE_URL")), "/")
	targetID := strings.TrimSpace(os.Getenv("WEB_CONSOLE_LIVE_TARGET_ID"))
	if baseURL == "" || targetID == "" {
		t.Skip("set WEB_CONSOLE_LIVE_URL and WEB_CONSOLE_LIVE_TARGET_ID for the live two-machine probe")
	}

	createBody := fmt.Sprintf(`{"shell":"/bin/sh","cols":80,"rows":24,"backend":"remote","origin":"SESSION_ORIGIN_UI","target_id":%q}`, targetID)
	createReq, err := http.NewRequest(http.MethodPost, baseURL+"/vrooli.web_console.v1.sessions.SessionsService/Create", strings.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	createReq.Header.Set("Content-Type", "application/json")
	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		t.Fatalf("create remote session: %v", err)
	}
	defer createResp.Body.Close()
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("create remote session status = %s", createResp.Status)
	}
	createPayload, err := io.ReadAll(createResp.Body)
	if err != nil {
		t.Fatalf("read remote session response: %v", err)
	}
	if lower := strings.ToLower(string(createPayload)); strings.Contains(lower, "token") || strings.Contains(lower, "credential") || strings.Contains(lower, "reauth") || strings.Contains(lower, "secret") {
		t.Fatalf("remote session response leaked a credential field: %s", createPayload)
	}
	var created struct {
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
	}
	if err := json.Unmarshal(createPayload, &created); err != nil {
		t.Fatalf("decode remote session: %v", err)
	}
	if created.Session.ID == "" {
		t.Fatal("remote session response did not contain an id")
	}
	sessionID := created.Session.ID
	t.Cleanup(func() {
		req, reqErr := http.NewRequest(http.MethodPost, baseURL+"/vrooli.web_console.v1.sessions.SessionsService/Delete", strings.NewReader(fmt.Sprintf(`{"id":%q}`, sessionID)))
		if reqErr != nil {
			t.Errorf("build remote session cleanup request: %v", reqErr)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, doErr := http.DefaultClient.Do(req)
		if doErr != nil {
			t.Errorf("delete remote session: %v", doErr)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("delete remote session status = %s", resp.Status)
		}
	})

	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/api/v1/sessions/" + sessionID + "/ws"
	conn, _, err := (&websocket.Dialer{}).Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial Web Console terminal: %v", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(20 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeHello}); err != nil {
		t.Fatalf("send terminal hello: %v", err)
	}

	var output strings.Builder
	readySeen := false
	readMessage := func(label string) TerminalMessage {
		t.Helper()
		_, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			t.Fatalf("read %s: %v", label, readErr)
		}
		if lower := strings.ToLower(string(payload)); strings.Contains(lower, "token") || strings.Contains(lower, "credential") || strings.Contains(lower, "reauth") || strings.Contains(lower, "secret") {
			t.Fatalf("terminal payload leaked a credential field: %s", payload)
		}
		var message TerminalMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			t.Fatalf("decode %s: %v", label, err)
		}
		return message
	}
	for !readySeen {
		message := readMessage("terminal message")
		switch message.Type {
		case MsgTypeSessionReady:
			readySeen = true
		case MsgTypeStdout:
			output.WriteString(message.Data)
		case MsgTypeError:
			t.Fatalf("remote terminal error: %s", message.Data)
		}
	}

	const uname = "uname -a\n"
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: uname, Offset: 0, Intent: StdinIntentBulkText}); err != nil {
		t.Fatalf("send uname: %v", err)
	}
	for !strings.Contains(output.String(), "Darwin") {
		message := readMessage("uname response")
		switch message.Type {
		case MsgTypeStdout:
			output.WriteString(message.Data)
		case MsgTypeError:
			t.Fatalf("remote terminal error during uname: %s", message.Data)
		}
	}

	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeResize, Cols: 100, Rows: 30}); err != nil {
		t.Fatalf("send terminal resize: %v", err)
	}
	resizeSeen := false
	for !resizeSeen {
		message := readMessage("resize response")
		switch message.Type {
		case MsgTypeResizeInfo:
			resizeSeen = message.Cols == 100 && message.Rows == 30
		case MsgTypeStdout:
			output.WriteString(message.Data)
		case MsgTypeError:
			t.Fatalf("remote terminal resize error: %s", message.Data)
		}
	}

	const stty = "stty size\n"
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: stty, Offset: int64(len(uname)), Intent: StdinIntentBulkText}); err != nil {
		t.Fatalf("send stty: %v", err)
	}
	for !strings.Contains(output.String(), "30 100") {
		message := readMessage("stty response")
		switch message.Type {
		case MsgTypeStdout:
			output.WriteString(message.Data)
		case MsgTypeError:
			t.Fatalf("remote terminal error during stty: %s", message.Data)
		}
	}
	t.Logf("remote transcript assertions: uname -a contained Darwin; resize reported 100x30; stty size reported 30 100; output=%q", output.String())
}

func TestHandleTerminalWS_ConnectionRelativeOffsetStartsAtZero(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)
	sess, ok := srv.sessions.Get(sessID)
	if !ok {
		t.Fatal("session disappeared before websocket connect")
	}
	if got := sess.AdvanceAcceptedThrough(5); got != 5 {
		t.Fatalf("seed accepted_through = %d, want 5", got)
	}

	conn, _, err := (&websocket.Dialer{}).Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read session_ready: %v", err)
		}
		if msg.Type == MsgTypeSessionReady {
			if msg.AcceptedThrough != 0 {
				t.Fatalf("connection accepted_through = %d, want connection-relative zero", msg.AcceptedThrough)
			}
			return
		}
	}
}

func TestHandleTerminalWS_TwoClientsBothDeliverInput(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	srv := newFakeTestServerWithFactory(func(pty.LaunchSpec) (pty.PTY, error) { return fake, nil })
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	first := dialSessionWS(t, ts, sess.ID)
	second := dialSessionWS(t, ts, sess.ID)
	if err := first.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "one", Offset: 0}); err != nil {
		t.Fatalf("first stdin: %v", err)
	}
	if ack := readUntilType(t, first, MsgTypeStdinAck); !ack.Ok || ack.AcceptedThrough != 3 {
		t.Fatalf("first ack = %+v, want success through 3", ack)
	}
	if err := second.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "two", Offset: 0}); err != nil {
		t.Fatalf("second stdin: %v", err)
	}
	if ack := readUntilType(t, second, MsgTypeStdinAck); !ack.Ok || ack.AcceptedThrough != 3 {
		t.Fatalf("second ack = %+v, want independent success through 3", ack)
	}
	if len(fake.Inputs) != 2 || string(fake.Inputs[0]) != "one" || string(fake.Inputs[1]) != "two" {
		t.Fatalf("PTY inputs = %q, want [one two]", fake.Inputs)
	}
}

func TestHandleTerminalWS_OffsetBehindOnlyAcknowledgesExactDuplicate(t *testing.T) {
	fake := ptyfake.NewFakePTYWithOutput()
	srv := newFakeTestServerWithFactory(func(pty.LaunchSpec) (pty.PTY, error) { return fake, nil })
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)
	sess, err := srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	conn := dialSessionWS(t, ts, sess.ID)
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "x", Offset: 0}); err != nil {
		t.Fatalf("initial stdin: %v", err)
	}
	if ack := readUntilType(t, conn, MsgTypeStdinAck); !ack.Ok {
		t.Fatalf("initial ack = %+v, want success", ack)
	}
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeStdin, Data: "y", Offset: 0}); err != nil {
		t.Fatalf("duplicate stdin: %v", err)
	}
	ack := readUntilType(t, conn, MsgTypeStdinAck)
	if ack.Ok || ack.Reason != StdinAckReasonOffsetGap {
		t.Fatalf("mismatched duplicate ack = %+v, want offset_gap rejection", ack)
	}
	if len(fake.Inputs) != 1 || string(fake.Inputs[0]) != "x" {
		t.Fatalf("PTY inputs = %q, want only original payload", fake.Inputs)
	}
}

// A reconnecting client must not convince the server to skip bytes it never
// accepted. The refusal is cumulative and must not trigger a replay.
func TestHandleTerminalWS_HelloAheadOfServerIsRefused(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

	conn, _, err := (&websocket.Dialer{}).Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	skipHistoryEnd(t, conn)

	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeHello, HaveThrough: 1}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	ack := readUntilType(t, conn, MsgTypeStdinAck)
	if ack.Ok || ack.AcceptedThrough != 0 || ack.Reason != StdinAckReasonUnreconcilable {
		t.Fatalf("ahead hello ack = %+v, want rejected unreconcilable at zero", ack)
	}

	// No stdin frame was accepted, so the server must not emit a replay or
	// advance the session's reliable input prefix.
	got, ok := srv.sessions.Get(sessID)
	if !ok || got == nil || got.AcceptedThrough() != 0 {
		t.Fatalf("session accepted_through advanced after refused hello")
	}
}

// session_ready is emitted exactly once per connection before the first
// stdin_ack, matching the client's gating contract.
func TestHandleTerminalWS_SessionReady_EmittedOnce(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

	conn, _, err := (&websocket.Dialer{}).Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	sawSessionReady := 0
	sawHistoryEnd := 0
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 5; i++ {
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		switch msg.Type {
		case MsgTypeSessionReady:
			sawSessionReady++
		case MsgTypeHistoryEnd:
			sawHistoryEnd++
		}
		if sawSessionReady > 0 && sawHistoryEnd > 0 {
			break
		}
	}
	if sawSessionReady != 1 {
		t.Errorf("expected session_ready to be emitted exactly once, got %d", sawSessionReady)
	}
	if sawHistoryEnd != 1 {
		t.Errorf("expected history_end to be emitted exactly once, got %d", sawHistoryEnd)
	}
}

// [REQ:P1-004a] Metrics - WebSocket connections increment metrics
func TestHandleTerminalWS_MetricsIncrement(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

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
	sessID := createTestSession(t, ts, srv)

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
	ts, srv := setupWSServer(t)
	sessID := createTestSession(t, ts, srv)

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
func setupWSServerWithPTY(t *testing.T) (*httptest.Server, string, *ptyfake.FakePTYWithOutput) {
	t.Helper()
	fake := ptyfake.NewFakePTYWithOutput()
	sm := newSessionManagerWithFactory(ptyfake.Factory(fake))
	srv := &Server{
		router:      mux.NewRouter(),
		sessions:    sm,
		events:      events.NewLogger(100),
		metrics:     metrics.New(),
		aiChain:     intai.NewChain(),
		shortcuts:   NewShortcutProfileStore(),
		aiConfig:    intai.NewMemConfigStore(),
		idempotency: intsessions.NewIdempotencyCache(),
		workspace:   intworkspace.NewMemStore(),
	}
	srv.setupRoutes()
	ts := httptest.NewServer(srv.router)
	t.Cleanup(ts.Close)

	sess, err := sm.Create(context.Background(), "/fake/shell", 80, 24, "", nil)
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

	// No output written — connect immediately. The server still sends a
	// self-contained snapshot (full reset + blank rows + cursor) as one
	// or more stdout frames followed by history_end.
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	sawHistoryEnd := false
	for i := 0; i < 8 && !sawHistoryEnd; i++ {
		msg := readTerminalMessage(t, conn)
		switch msg.Type {
		case MsgTypeHistoryEnd:
			sawHistoryEnd = true
		case MsgTypeSessionReady, MsgTypeStdout, MsgTypeEchoState:
			// Snapshot stdout frames + handshake sibling are allowed.
		default:
			t.Errorf("unexpected message type before history_end: %q", msg.Type)
		}
	}
	if !sawHistoryEnd {
		t.Error("history_end never arrived")
	}
}

func TestHandleTerminalWS_HistoryEnd_AfterHistory(t *testing.T) {
	ts, sessID, fake := setupWSServerWithPTY(t)

	// Write output so history is built up.
	_, _ = fake.OutW.Write([]byte("line 1\r\nline 2\r\n"))
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

func TestHandleTerminalWS_ReconnectReplaysOutputDelta(t *testing.T) {
	ts, sessID, fake := setupWSServerWithPTY(t)
	defer fake.Close()
	_, _ = fake.OutW.Write([]byte("first"))
	time.Sleep(100 * time.Millisecond)

	dialer := websocket.Dialer{}
	first, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("first ws dial: %v", err)
	}
	var rendered int64
	for i := 0; i < 20; i++ {
		msg := readTerminalMessage(t, first)
		if msg.Type == MsgTypeHistoryEnd {
			rendered = msg.OutputCursor
			break
		}
	}
	if rendered == 0 {
		t.Fatal("first connection did not report an output cursor")
	}
	_ = first.Close()
	_, _ = fake.OutW.Write([]byte("second"))
	time.Sleep(100 * time.Millisecond)

	second, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("second ws dial: %v", err)
	}
	defer second.Close()
	if err := second.WriteJSON(TerminalMessage{Type: MsgTypeHello, WantResume: true, RenderedThrough: rendered}); err != nil {
		t.Fatalf("send resume hello: %v", err)
	}
	var stdout string
	for i := 0; i < 20; i++ {
		msg := readTerminalMessage(t, second)
		switch msg.Type {
		case MsgTypeStdout:
			stdout += msg.Data
		case MsgTypeResync:
			t.Fatal("covered output cursor unexpectedly requested a full resync")
		case MsgTypeHistoryEnd:
			if !strings.Contains(stdout, "second") || strings.Contains(stdout, "first") {
				t.Fatalf("resume stdout = %q, want only output after cursor", stdout)
			}
			return
		}
	}
	t.Fatal("resumed connection did not reach history_end")
}

// --- Snapshot replay WebSocket tests ---

// TestHandleTerminalWS_SnapshotPrecedesHistoryEnd verifies the snapshot
// stream and the closing history_end frame.
func TestHandleTerminalWS_SnapshotPrecedesHistoryEnd(t *testing.T) {
	ts, sessID, fake := setupWSServerWithPTY(t)
	defer fake.Close()

	_, _ = fake.OutW.Write([]byte("snapshot replay marker"))
	time.Sleep(100 * time.Millisecond)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()

	var (
		gotStdout bool
		stdoutAcc string
	)
	for i := 0; i < 30; i++ {
		msg := readTerminalMessage(t, conn)
		switch msg.Type {
		case MsgTypeStdout:
			gotStdout = true
			stdoutAcc += msg.Data
		case MsgTypeHistoryEnd:
			if !gotStdout {
				t.Error("expected at least one snapshot stdout frame before history_end")
			}
			if !strings.Contains(stdoutAcc, "snapshot replay marker") {
				t.Errorf("snapshot stream missing prior PTY output; got=%q", stdoutAcc)
			}
			return
		case MsgTypeSessionReady:
			// Allowed at any point after history_end.
		}
	}
	t.Fatal("never received history_end message")
}

// TestHandleTerminalWS_ServerPingKeepalive verifies that the server sends
// WebSocket ping frames to keep the connection alive through reverse proxies
// (e.g. Cloudflare tunnel's ~100s idle timeout).
func TestHandleTerminalWS_ServerPingKeepalive(t *testing.T) {
	// Shorten the keepalive interval so the test runs quickly. The behavior
	// being verified (server sends a ping on schedule) is identical.
	prevPingPeriod := wsPingPeriod.Load()
	wsPingPeriod.Store(int64(100 * time.Millisecond))
	t.Cleanup(func() { wsPingPeriod.Store(prevPingPeriod) })

	ts, sessID, _ := setupWSServerWithPTY(t)

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer conn.Close()
	skipHistoryEnd(t, conn)

	// Track server-initiated pings via the pong handler.
	// The gorilla/websocket library automatically responds to pings with pongs,
	// but we can register a handler to observe them.
	pingReceived := make(chan struct{}, 1)
	conn.SetPingHandler(func(appData string) error {
		select {
		case pingReceived <- struct{}{}:
		default:
		}
		// Send pong (default behavior)
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	// Read in a goroutine so the ping handler fires
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	select {
	case <-pingReceived:
		// Server-initiated ping received — keepalive is working
	case <-time.After(2 * time.Second):
		t.Fatal("no server-initiated ping received within 2s (override interval was 100ms)")
	}
}

// TestHandleTerminalWS_ReconnectToSameSession verifies that a client can
// disconnect and reconnect to the same session, receiving history on the
// second connection.
func TestHandleTerminalWS_ReconnectToSameSession(t *testing.T) {
	ts, sessID, fake := setupWSServerWithPTY(t)

	// First connection: write some output
	dialer := websocket.Dialer{}
	conn1, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial 1: %v", err)
	}
	skipHistoryEnd(t, conn1)

	// Inject PTY output
	_, _ = fake.OutW.Write([]byte("hello from session"))
	time.Sleep(100 * time.Millisecond)

	// Read the output
	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg TerminalMessage
	if err := conn1.ReadJSON(&msg); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if msg.Type != MsgTypeStdout || msg.Data != "hello from session" {
		t.Errorf("first conn: got type=%s data=%q", msg.Type, msg.Data)
	}

	// Disconnect
	conn1.Close()
	// The remote-session contract promises that a short network interruption
	// does not terminate the server-owned session. Keep the socket detached for
	// the full five-second interval before attaching again, so this test cannot
	// pass merely because the reconnect happened immediately.
	time.Sleep(5 * time.Second)

	// Second connection: should get history
	conn2, _, err := dialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessID+"/ws"), nil)
	if err != nil {
		t.Fatalf("ws dial 2: %v", err)
	}
	defer conn2.Close()

	// Read history + history_end
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	var gotHistory bool
	for {
		var m TerminalMessage
		if err := conn2.ReadJSON(&m); err != nil {
			t.Fatalf("read on reconnect: %v", err)
		}
		if m.Type == MsgTypeStdout {
			gotHistory = true
		}
		if m.Type == MsgTypeHistoryEnd {
			break
		}
	}
	if !gotHistory {
		t.Error("expected to receive history on reconnect")
	}
}

// TestTerminalWS_ScrollFrameIsAnsweredAndNeverFatal covers the wire contract
// for backend-driven scrolling. A `scroll` frame must always be answered on
// the same socket — with ok on a backend that owns history, or a typed
// unsupported when the pane keeps real client-side scrollback — and must never
// close the connection. A dropped or fatal scroll frame would strand the
// browser's scroll gate waiting for an acknowledgement that never arrives.
func TestTerminalWS_ScrollFrameIsAnsweredAndNeverFatal(t *testing.T) {
	ts, srv := setupWSServer(t)
	sessionID := createTestSession(t, ts, srv)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(ts, "/api/v1/sessions/"+sessionID+"/ws"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	skipHistoryEnd(t, conn)

	// A zero-line scroll carries no intent and must be ignored outright, so
	// the first reply we see has to belong to the request that followed it.
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeScroll, Lines: 0}); err != nil {
		t.Fatalf("write empty scroll: %v", err)
	}
	if err := conn.WriteJSON(TerminalMessage{Type: MsgTypeScroll, Lines: -3}); err != nil {
		t.Fatalf("write scroll: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		var msg TerminalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read scroll reply: %v", err)
		}
		if msg.Type != MsgTypeScroll {
			continue
		}
		switch {
		case msg.Ok:
			if msg.Lines != -3 {
				t.Fatalf("scroll reply echoed %d lines, want -3 (the empty frame must be ignored)", msg.Lines)
			}
		case msg.Data == "unsupported":
			if strings.TrimSpace(msg.Reason) == "" {
				t.Fatal("unsupported scroll reply carried no reason")
			}
		default:
			t.Fatalf("scroll reply was neither ok nor typed-unsupported: %+v", msg)
		}
		return
	}
}
