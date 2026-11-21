package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
