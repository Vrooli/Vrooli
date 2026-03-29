package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:P1-004b] Operational Metrics Collection - metrics tests

func TestMetrics_Snapshot_Defaults(t *testing.T) {
	m := NewMetrics()
	snap := m.Snapshot()

	if snap.Sessions.Created != 0 {
		t.Errorf("expected 0 created sessions, got %d", snap.Sessions.Created)
	}
	if snap.Sessions.Active != 0 {
		t.Errorf("expected 0 active sessions, got %d", snap.Sessions.Active)
	}
	if snap.Connections.Total != 0 {
		t.Errorf("expected 0 total connections, got %d", snap.Connections.Total)
	}
	if snap.Messages.Sent != 0 {
		t.Errorf("expected 0 messages sent, got %d", snap.Messages.Sent)
	}
	if snap.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

func TestMetrics_Snapshot_AfterOperations(t *testing.T) {
	m := NewMetrics()

	m.SessionsCreated.Add(3)
	m.SessionsDeleted.Add(1)
	m.ActiveSessions.Add(2)
	m.ConnectionsTotal.Add(5)
	m.ActiveConnections.Add(2)
	m.WSMessagesSent.Add(100)
	m.WSMessagesReceived.Add(50)
	m.ResizeCount.Add(7)

	snap := m.Snapshot()

	if snap.Sessions.Created != 3 {
		t.Errorf("expected 3 created, got %d", snap.Sessions.Created)
	}
	if snap.Sessions.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", snap.Sessions.Deleted)
	}
	if snap.Sessions.Active != 2 {
		t.Errorf("expected 2 active, got %d", snap.Sessions.Active)
	}
	if snap.Sessions.Resizes != 7 {
		t.Errorf("expected 7 resizes, got %d", snap.Sessions.Resizes)
	}
	if snap.Connections.Total != 5 {
		t.Errorf("expected 5 total connections, got %d", snap.Connections.Total)
	}
	if snap.Connections.Active != 2 {
		t.Errorf("expected 2 active connections, got %d", snap.Connections.Active)
	}
	if snap.Messages.Sent != 100 {
		t.Errorf("expected 100 sent, got %d", snap.Messages.Sent)
	}
	if snap.Messages.Received != 50 {
		t.Errorf("expected 50 received, got %d", snap.Messages.Received)
	}
}

func TestMetrics_AtomicConcurrency(t *testing.T) {
	m := NewMetrics()
	done := make(chan struct{})

	for i := 0; i < 100; i++ {
		go func() {
			m.SessionsCreated.Add(1)
			m.WSMessagesSent.Add(1)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}

	snap := m.Snapshot()
	if snap.Sessions.Created != 100 {
		t.Errorf("expected 100 created, got %d", snap.Sessions.Created)
	}
	if snap.Messages.Sent != 100 {
		t.Errorf("expected 100 sent, got %d", snap.Messages.Sent)
	}
}

func TestHandleMetrics_Endpoint(t *testing.T) {
	srv := newFakeTestServer()

	// Simulate some operations
	srv.metrics.SessionsCreated.Add(2)
	srv.metrics.ActiveSessions.Add(1)
	srv.metrics.ConnectionsTotal.Add(3)

	req := httptest.NewRequest("GET", "/api/v1/metrics", nil)
	rec := httptest.NewRecorder()

	srv.handleMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}

	var resp MetricsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Sessions.Created != 2 {
		t.Errorf("expected 2 created, got %d", resp.Sessions.Created)
	}
	if resp.Sessions.Active != 1 {
		t.Errorf("expected 1 active, got %d", resp.Sessions.Active)
	}
	if resp.Connections.Total != 3 {
		t.Errorf("expected 3 total connections, got %d", resp.Connections.Total)
	}
	if resp.Uptime == "" {
		t.Error("uptime should not be empty")
	}
}

// [REQ:P1-004b] Verify metrics overhead is negligible (<1ms per operation)
func TestMetrics_PerformanceOverhead(t *testing.T) {
	m := NewMetrics()

	start := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.SessionsCreated.Add(1)
			m.WSMessagesSent.Add(1)
			m.Snapshot()
		}
	})

	// 10000 iterations should complete very quickly; atomic ops are <100ns each
	nsPerOp := start.NsPerOp()
	if nsPerOp > 1_000_000 { // 1ms threshold
		t.Errorf("metrics overhead too high: %d ns/op (threshold: 1,000,000 ns)", nsPerOp)
	}
}

func TestHandleCreateSession_EmitsEvent(t *testing.T) {
	srv := newTestServer()

	body := strings.NewReader(`{"cols": 80, "rows": 24}`)
	req := httptest.NewRequest("POST", "/api/v1/sessions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.handleCreateSession(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify event was emitted
	events := srv.events.Recent(10)
	found := false
	for _, evt := range events {
		if evt.Type == EventSessionCreated {
			found = true
			if evt.Details["shell"] == "" {
				t.Error("expected shell detail in session.created event")
			}
			break
		}
	}
	if !found {
		t.Error("expected session.created event to be emitted")
	}

	// Verify metrics were incremented
	snap := srv.metrics.Snapshot()
	if snap.Sessions.Created != 1 {
		t.Errorf("expected 1 session created metric, got %d", snap.Sessions.Created)
	}
	if snap.Sessions.Active != 1 {
		t.Errorf("expected 1 active session metric, got %d", snap.Sessions.Active)
	}

	// Cleanup
	var resp SessionResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	_ = srv.sessions.Delete(resp.ID)
}

func TestHandleDeleteSession_EmitsEvent(t *testing.T) {
	srv := newTestServer()

	// Create a session first
	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	srv.metrics.SessionsCreated.Add(1)
	srv.metrics.ActiveSessions.Add(1)

	req := httptest.NewRequest("DELETE", "/api/v1/sessions/"+sess.ID, nil)
	req = mux.SetURLVars(req, map[string]string{"id": sess.ID})
	rec := httptest.NewRecorder()

	srv.handleDeleteSession(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Verify event
	events := srv.events.Recent(10)
	found := false
	for _, evt := range events {
		if evt.Type == EventSessionDeleted && evt.SessionID == sess.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session.deleted event to be emitted")
	}

	// Verify metrics
	snap := srv.metrics.Snapshot()
	if snap.Sessions.Deleted != 1 {
		t.Errorf("expected 1 deleted metric, got %d", snap.Sessions.Deleted)
	}
}
