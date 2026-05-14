package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	metricsH "web-console/handlers/metrics"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"

	"web-console/internal/events"
	"web-console/internal/metrics"
)

// [REQ:P1-004b] Operational Metrics Collection - metrics tests

func TestMetrics_Snapshot_Defaults(t *testing.T) {
	m := metrics.New()
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
	m := metrics.New()

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
	m := metrics.New()
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

// MetricsService.Get returns the current snapshot through the Connect
// handler. Exercises the same transport-neutral path the real CLI uses.
func TestMetricsService_Get(t *testing.T) {
	srv := newFakeTestServer()

	// Simulate some operations
	srv.metrics.SessionsCreated.Add(2)
	srv.metrics.ActiveSessions.Add(1)
	srv.metrics.ConnectionsTotal.Add(3)

	h := metricsH.NewConnectHandler(metricsH.Deps{Service: newMetricsAdapter(srv)})
	resp, err := h.Get(context.Background(), connect.NewRequest(&metricsv1.GetRequest{}))
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	got := resp.Msg
	if got.GetSessions().GetCreated() != 2 {
		t.Errorf("expected 2 created, got %d", got.GetSessions().GetCreated())
	}
	if got.GetSessions().GetActive() != 1 {
		t.Errorf("expected 1 active, got %d", got.GetSessions().GetActive())
	}
	if got.GetConnections().GetTotal() != 3 {
		t.Errorf("expected 3 total connections, got %d", got.GetConnections().GetTotal())
	}
	if got.GetUptime() == "" {
		t.Error("uptime should not be empty")
	}
}

// [REQ:P1-004b] Verify metrics overhead is negligible (<1ms per operation)
func TestMetrics_PerformanceOverhead(t *testing.T) {
	m := metrics.New()

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

	sess, err := callCreate(t, srv, 80, 24, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	evts := srv.events.Recent(10)
	found := false
	for _, evt := range evts {
		if evt.Type == events.SessionCreated {
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

	snap := srv.metrics.Snapshot()
	if snap.Sessions.Created != 1 {
		t.Errorf("expected 1 session created metric, got %d", snap.Sessions.Created)
	}
	if snap.Sessions.Active != 1 {
		t.Errorf("expected 1 active session metric, got %d", snap.Sessions.Active)
	}

	_ = srv.sessions.Delete(sess.GetId())
}

func TestHandleDeleteSession_EmitsEvent(t *testing.T) {
	srv := newTestServer()

	sess, err := srv.sessions.Create("", 80, 24, "", nil)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	srv.metrics.SessionsCreated.Add(1)
	srv.metrics.ActiveSessions.Add(1)

	if err := callDelete(t, srv, sess.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	evts := srv.events.Recent(10)
	found := false
	for _, evt := range evts {
		if evt.Type == events.SessionDeleted && evt.SessionID == sess.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session.deleted event to be emitted")
	}

	snap := srv.metrics.Snapshot()
	if snap.Sessions.Deleted != 1 {
		t.Errorf("expected 1 deleted metric, got %d", snap.Sessions.Deleted)
	}
}
