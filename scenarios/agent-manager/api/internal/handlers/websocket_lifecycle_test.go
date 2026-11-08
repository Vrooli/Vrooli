package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	coredb "github.com/vrooli/api-core/database"
)

func TestWebSocketSaturationDoesNotBlockDurableProducers(t *testing.T) {
	hub := NewWebSocketHub()
	t.Cleanup(hub.Close)
	primary := &WebSocketClient{send: make(chan []byte, 1)}
	testClient := &WebSocketClient{send: make(chan []byte, 1), partition: coredb.ContextPartition(coredb.WithTestMode(t.Context()))}
	hub.clients[primary], hub.clients[testClient] = true, true
	// The stopped consumer reproduces saturation without a scheduler/timing
	// assumption. Overflow must force only the affected partition to reconcile.
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for i := 0; i <= cap(hub.broadcast); i++ {
			hub.BroadcastEvent(t.Context(), domain.NewLogEvent(uuid.New(), "info", "durable event"))
		}
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("saturated notification queue blocked the durable producer")
	}
	if _, open := <-primary.send; open {
		t.Fatal("overflow left an affected subscriber silently missing data")
	}
	select {
	case <-testClient.send:
		t.Fatal("primary overflow disconnected a foreign routing partition")
	default:
	}
}

func TestWebSocketShutdownClosesConnectionsAndLatePublishers(t *testing.T) {
	hub := NewWebSocketHub()
	stopped := make(chan struct{})
	go func() { defer close(stopped); hub.Run() }()
	server := httptest.NewServer((&Handler{}).HandleWebSocket(hub))
	t.Cleanup(server.Close)
	t.Cleanup(hub.Close)
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, _, err := conn.ReadMessage(); err != nil {
		t.Fatal(err)
	}
	hub.Close()
	hub.Close()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("hub main loop survived shutdown")
	}
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("hijacked socket survived HTTP-owner cleanup")
	}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for i := 0; i < 1000; i++ {
			hub.BroadcastEvent(context.Background(), domain.NewLogEvent(uuid.New(), "info", "late durable callback"))
		}
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("late runner callbacks blocked after hub shutdown")
	}
	response, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("closed hub admitted a new request: HTTP%d", response.StatusCode)
	}
}
