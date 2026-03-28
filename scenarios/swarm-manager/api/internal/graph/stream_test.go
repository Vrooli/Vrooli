package graph

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/scenarios"
)

func TestWSStream(t *testing.T) {
	broker := NewBroker()
	defer broker.Stop()

	projSvc := NewProjectionService(ProjectionConfig{
		Backlog: &mockBacklogLister{items: []backlog.BacklogItem{
			{Kind: "execute", Name: "task-a", Title: "A", Status: backlog.StatusInProgress},
		}},
		Scenario: &mockScenarioLister{scens: []scenarios.Scenario{
			{Name: "my-app", Status: scenarios.StatusRunning},
		}},
		Execution: &mockExecutionLister{records: []execution.Record{
			{ExecutionID: "exec-1", BacklogKind: "execute", BacklogName: "task-a", Status: execution.StatusRunning},
		}},
	})

	streamHandler := NewStreamHandler(projSvc, broker)

	srv := httptest.NewServer(streamHandler)
	defer srv.Close()

	// Connect via WebSocket using the path that RegisterRoutes would use.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Should receive full-sync message.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Type != WSFullSync {
		t.Errorf("expected full-sync, got %s", msg.Type)
	}
	if msg.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	// Send a broadcast event and verify the client receives it.
	broker.BroadcastUpdate(WSNodeUpdate, map[string]string{"id": "scenario/my-app"})

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw2, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read incremental error: %v", err)
	}

	var msg2 WSMessage
	if err := json.Unmarshal(raw2, &msg2); err != nil {
		t.Fatalf("unmarshal incremental error: %v", err)
	}
	if msg2.Type != WSNodeUpdate {
		t.Errorf("expected node-update, got %s", msg2.Type)
	}
}

// TestWSStreamHandlerServeHTTP verifies the stream handler works as an http.Handler
// (the httptest.NewServer uses ServeHTTP which maps to HandleWebSocket).
func TestWSStreamHandlerServeHTTP(t *testing.T) {
	broker := NewBroker()
	defer broker.Stop()

	projSvc := NewProjectionService(ProjectionConfig{
		Backlog:  &mockBacklogLister{items: nil},
		Scenario: &mockScenarioLister{scens: nil},
	})

	streamHandler := NewStreamHandler(projSvc, broker)
	srv := httptest.NewServer(streamHandler)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	// Should get full-sync even with empty data.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var msg WSMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if msg.Type != WSFullSync {
		t.Errorf("expected full-sync, got %s", msg.Type)
	}
}
