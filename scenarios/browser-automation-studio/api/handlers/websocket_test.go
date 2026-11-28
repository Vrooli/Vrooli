package handlers

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
)

// ============================================================================
// HandleWebSocket Tests
// ============================================================================

func TestHandleRecordingWebSocketRejectsMissingOrForeignCookie(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	handler.wsAllowedOrigins = nil
	router := chi.NewRouter()
	router.Get("/ws/sessions/{sessionID}", handler.HandleRecordingWebSocket)
	sessionID := uuid.NewString()
	for _, tc := range []struct {
		name   string
		cookie *http.Cookie
	}{
		{name: "missing"},
		{name: "foreign", cookie: &http.Cookie{Name: wsHub.SessionStreamCookieName(sessionID), Value: "foreign-cookie"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/ws/sessions/"+sessionID, nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status=%d, want %d: %s", rr.Code, http.StatusForbidden, rr.Body.String())
			}
		})
	}
}

func TestRecordingWebSocketRouteBindsFramesToExactCookieSession(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()
	hub := wsHub.NewHub(handler.log)
	handler.wsHub = hub
	go hub.Run()

	sessionA, sessionB := uuid.NewString(), uuid.NewString()
	executionA, executionB := uuid.NewString(), uuid.NewString()
	leaseA := uuid.NewString()
	nameA, valueA, err := hub.IssueSessionStreamCookieRole(sessionA, executionA, leaseA, "observe", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	nameB, valueB, err := hub.IssueSessionStreamCookieRole(sessionB, executionB, uuid.NewString(), "observe", time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if nameB == nameA {
		t.Fatal("session cookie names must be scoped to their session path")
	}

	router := chi.NewRouter()
	router.Get("/ws/sessions/{sessionID}", handler.HandleRecordingWebSocket)
	server := httptest.NewServer(router)
	defer server.Close()
	defer hub.RevokeSessionStreamCookie(sessionA)
	defer hub.RevokeSessionStreamCookie(sessionB)

	// A cookie issued for B must not authorize A's exact-session path even if
	// the caller supplies A's cookie name. Path scoping alone is not authority.
	urlA := "ws" + strings.TrimPrefix(server.URL, "http") + wsHub.SessionStreamCookiePath(sessionA)
	foreignValueHeader := http.Header{"Cookie": []string{nameA + "=" + valueB}}
	if conn, response, dialErr := websocket.DefaultDialer.Dial(urlA, foreignValueHeader); dialErr == nil {
		_ = conn.Close()
		t.Fatal("session B cookie authorized session A's exact path")
	} else if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign cookie on exact session path response=%v err=%v", response, dialErr)
	}

	wrongURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + sessionB
	wrongHeader := http.Header{"Cookie": []string{wsHub.SessionStreamCookieName(sessionB) + "=" + valueA}}
	if conn, response, dialErr := websocket.DefaultDialer.Dial(wrongURL, wrongHeader); dialErr == nil {
		_ = conn.Close()
		t.Fatal("cookie for session A opened session B's path")
	} else if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign session path response=%v err=%v", response, dialErr)
	}

	url := "ws" + strings.TrimPrefix(server.URL, "http") + wsHub.SessionStreamCookiePath(sessionA)
	header := http.Header{"Cookie": []string{nameA + "=" + valueA}}
	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatalf("exact session cookie failed to open stream: %v", err)
	}
	defer conn.Close()
	var welcome map[string]any
	if err := conn.ReadJSON(&welcome); err != nil || welcome["type"] != "connected" {
		t.Fatalf("stream welcome = %#v err=%v", welcome, err)
	}

	// A run-B frame and a wrong-session frame for run A must be effect-free for
	// A's socket. Only the exact run/session tuple can reach the viewer.
	hub.BroadcastExecutionFrame(executionB, &wsHub.ExecutionFrame{ExecutionID: executionB, SessionID: sessionB, Data: "Zm9yZWlnbg==", MediaType: "image/jpeg", Width: 8, Height: 8})
	hub.BroadcastExecutionFrame(executionA, &wsHub.ExecutionFrame{ExecutionID: executionA, SessionID: sessionB, Data: "Zm9yZWlnbg==", MediaType: "image/jpeg", Width: 8, Height: 8})
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var unexpected map[string]any
	if err := conn.ReadJSON(&unexpected); err == nil {
		t.Fatalf("foreign frame reached cookie-scoped stream: %#v", unexpected)
	} else if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("expected no foreign frames, read err=%v", err)
	}
	// A Gorilla WebSocket read timeout permanently poisons that connection's
	// reader. Use a fresh exact-cookie connection for the positive control.
	_ = conn.Close()
	acceptedConn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		t.Fatalf("exact session cookie failed to reconnect: %v", err)
	}
	defer acceptedConn.Close()
	var reconnectWelcome map[string]any
	if err := acceptedConn.ReadJSON(&reconnectWelcome); err != nil || reconnectWelcome["type"] != "connected" {
		t.Fatalf("reconnected stream welcome = %#v err=%v", reconnectWelcome, err)
	}

	hub.BroadcastExecutionFrame(executionA, &wsHub.ExecutionFrame{ExecutionID: executionA, SessionID: sessionA, Data: "b3duZXI=", MediaType: "image/jpeg", Width: 8, Height: 8})
	_ = acceptedConn.SetReadDeadline(time.Now().Add(time.Second))
	var accepted map[string]any
	if err := acceptedConn.ReadJSON(&accepted); err != nil {
		t.Fatalf("authorized same-session frame read failed: %v", err)
	}
	if accepted["type"] != "session_frame" || accepted["session_id"] != sessionA || accepted["data"] != "b3duZXI=" {
		t.Fatalf("authorized frame = %#v", accepted)
	}
}

func TestHandleWebSocket_UpgradeFailure(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Set up upgrader with a check origin that passes
	handler.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Regular HTTP request (not a WebSocket upgrade request) should fail
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rr := httptest.NewRecorder()

	// This should log an error but not panic
	handler.HandleWebSocket(rr, req)

	// The response recorder doesn't support WebSocket upgrade,
	// so the upgrade will fail silently (logging error)
	// We just verify no panic occurred
}

func TestHandleWebSocket_WithExecutionID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	handler.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	executionID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/ws?execution_id="+executionID.String(), nil)
	rr := httptest.NewRecorder()

	// This will fail to upgrade but should parse the execution_id
	handler.HandleWebSocket(rr, req)

	// No assertions needed - we're testing that parsing works without panic
}

func TestHandleWebSocket_WithInvalidExecutionID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	handler.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	req := httptest.NewRequest(http.MethodGet, "/ws?execution_id=invalid-uuid", nil)
	rr := httptest.NewRecorder()

	// Invalid UUID should be ignored (executionID will be nil)
	handler.HandleWebSocket(rr, req)

	// No panic means success - invalid UUIDs are silently ignored
}

func TestHandleWebSocket_NoExecutionID(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	handler.upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	rr := httptest.NewRecorder()

	// No execution_id parameter should work fine
	handler.HandleWebSocket(rr, req)
}

func TestHandleWebSocket_OriginRejected(t *testing.T) {
	handler, _, _, _, _, _ := createTestHandler()

	// Set up upgrader to reject all origins
	handler.wsAllowedOrigins = []string{"http://allowed.com"}
	handler.upgrader = websocket.Upgrader{
		CheckOrigin: handler.isOriginAllowed,
	}

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.Header.Set("Origin", "http://rejected.com")
	rr := httptest.NewRecorder()

	// Should fail due to origin rejection
	handler.HandleWebSocket(rr, req)
}
