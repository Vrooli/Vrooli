package graph

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSStream(t *testing.T) {
	broker := NewBroker()
	defer broker.Stop()

	streamHandler := NewStreamHandler(broker)
	srv := httptest.NewServer(streamHandler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	broker.BroadcastUpdate(WSNodeUpdate, map[string]string{"id": "scenario/my-app"})

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read incremental error: %v", err)
	}

	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal incremental error: %v", err)
	}
	if msg.Type != WSNodeUpdate {
		t.Errorf("expected node-update, got %s", msg.Type)
	}
}

// TestWSStreamHandlerServeHTTP verifies the stream handler works as an
// http.Handler and remains idle until an explicit event is broadcast.
func TestWSStreamHandlerServeHTTP(t *testing.T) {
	broker := NewBroker()
	defer broker.Stop()

	streamHandler := NewStreamHandler(broker)
	srv := httptest.NewServer(streamHandler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected no initial websocket payload")
	} else if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("expected websocket read timeout, got %v", err)
	}
}
