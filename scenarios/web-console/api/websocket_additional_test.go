package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHandleWorkspaceWebSocketSuccess(t *testing.T) {
	cleanup := setupTestLogger()
	t.Cleanup(cleanup)

	env := setupTestDirectory(t)
	t.Cleanup(env.Cleanup)

	cfg := setupTestConfig(env.TempDir)
	_, metrics, ws := setupTestSessionManager(t, cfg)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleWorkspaceWebSocket(w, r, ws, metrics)
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("websocket dial failed: %v", err)
	}

	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected initial workspace snapshot, got error: %v", err)
	}
	if !bytes.Contains(msg, []byte("activeTabId")) {
		t.Fatalf("workspace snapshot missing activeTabId: %s", string(msg))
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte("{}")); err != nil {
		t.Fatalf("failed to write noop message: %v", err)
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("error closing websocket: %v", err)
	}

	// Allow goroutines to exit cleanly
	time.Sleep(20 * time.Millisecond)

	if metrics.httpUpgradesFail.Load() != 0 {
		t.Fatalf("expected no upgrade failures, got %d", metrics.httpUpgradesFail.Load())
	}
}
