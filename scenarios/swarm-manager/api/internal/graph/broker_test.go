package graph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSBroker(t *testing.T) {
	broker := NewBroker()
	defer broker.Stop()

	// Create a test WebSocket server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade error: %v", err)
			return
		}
		broker.AddClient(conn)
		defer func() {
			broker.RemoveClient(conn)
			conn.Close()
		}()
		// Keep connection open.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer srv.Close()

	// Connect client.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Wait for client registration.
	time.Sleep(50 * time.Millisecond)

	if broker.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", broker.ClientCount())
	}

	// Broadcast a message.
	broker.BroadcastUpdate("test-event", map[string]string{"key": "value"})

	// Read the message.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Type != "test-event" {
		t.Errorf("expected type test-event, got %s", msg.Type)
	}
	if msg.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	// Disconnect and verify cleanup.
	conn.Close()
	time.Sleep(100 * time.Millisecond)
}

func TestWSBrokerClientRemoval(t *testing.T) {
	broker := NewBroker()
	defer broker.Stop()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		broker.AddClient(conn)
		defer func() {
			broker.RemoveClient(conn)
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if broker.ClientCount() != 1 {
		t.Fatalf("expected 1 client, got %d", broker.ClientCount())
	}

	conn.Close()
	time.Sleep(200 * time.Millisecond)

	// After close, broadcast should clean up the dead connection.
	broker.BroadcastUpdate("cleanup-test", nil)
	time.Sleep(100 * time.Millisecond)

	if broker.ClientCount() != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", broker.ClientCount())
	}
}
