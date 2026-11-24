package commerce

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type incidentHTTPClient struct{ calls atomic.Int64 }

func (c *incidentHTTPClient) Do(req *http.Request) (*http.Response, error) {
	call := c.calls.Add(1)
	status := http.StatusOK
	if call == 1 {
		status = http.StatusServiceUnavailable
	}
	return &http.Response{StatusCode: status, Body: http.NoBody, Header: make(http.Header)}, nil
}

func TestPaymentReconciliationIncidentLifecycleAndAlertOutboxRestart(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("lpbs_incidents_test"), postgres.WithUsername("lpbs"), postgres.WithPassword("test-only"))
	if err != nil {
		t.Fatalf("start isolated PostgreSQL test fixture: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	connection, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("postgres", connection)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	deadline := time.Now().Add(10 * time.Second)
	for db.PingContext(ctx) != nil {
		if time.Now().After(deadline) {
			t.Fatal("isolated PostgreSQL fixture did not accept connections")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := db.Exec(FinancialSchema()); err != nil {
		t.Fatalf("apply isolated schema: %v", err)
	}
	dispatcher := NewAnomalyAlertDispatcher(db, DispatcherRuntime{ScenarioName: "landing-page-business-suite"})
	client := &incidentHTTPClient{}
	dispatcher.UseHTTPClient(client)
	dispatcher.UseBackoff([]time.Duration{0})
	dispatcher.UseMaxAttempts(1)
	service, err := NewPaymentReconciliationIncidentService(db, dispatcher, func() string { return "https://alerts.invalid/hook" }, DispatcherRuntime{ScenarioName: "landing-page-business-suite"})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.Open(ctx, "stripe-event-reconciliation", "error", map[string]interface{}{"code": "provider_timeout", "credential": "synthetic-canary-never-forward"})
	if err != nil || !opened {
		t.Fatalf("first open = %t, %v", opened, err)
	}
	opened, err = service.Open(ctx, "stripe-event-reconciliation", "error", map[string]interface{}{"code": "provider_timeout"})
	if err != nil || opened {
		t.Fatalf("duplicate open = %t, %v", opened, err)
	}
	var safeDetails string
	if err := db.QueryRow(`SELECT details::text FROM payment_reconciliation_incidents WHERE incident_key=$1`, "stripe-event-reconciliation").Scan(&safeDetails); err != nil || strings.Contains(safeDetails, "synthetic-canary") {
		t.Fatalf("stored incident details=%q err=%v; caller-supplied extra field leaked", safeDetails, err)
	}
	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatal(err)
	}
	resolved, err := service.Resolve(ctx, "stripe-event-reconciliation", map[string]interface{}{"code": "provider_recovered"})
	if err != nil || !resolved {
		t.Fatalf("resolve = %t, %v", resolved, err)
	}
	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if calls := client.calls.Load(); calls != 1 {
		t.Fatalf("resolved alert overtook undelivered open alert: webhook calls=%d", calls)
	}
	if _, err := db.Exec(`UPDATE payment_reconciliation_incident_events SET dispatch_after=CURRENT_TIMESTAMP WHERE incident_key=$1 AND transition='opened' AND generation=1`, "stripe-event-reconciliation"); err != nil {
		t.Fatal(err)
	}
	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if duplicate, err := service.Resolve(ctx, "stripe-event-reconciliation", nil); err != nil || duplicate {
		t.Fatalf("duplicate resolve = %t, %v", duplicate, err)
	}
	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if reopened, err := service.Open(ctx, "stripe-event-reconciliation", "error", map[string]interface{}{"code": "provider_timeout_again"}); err != nil || !reopened {
		t.Fatalf("reopen = %t, %v", reopened, err)
	}
	if err := service.DispatchPending(ctx, 10); err != nil {
		t.Fatal(err)
	}
	var generation int64
	var status string
	if err := db.QueryRow(`SELECT generation,status FROM payment_reconciliation_incidents WHERE incident_key=$1`, "stripe-event-reconciliation").Scan(&generation, &status); err != nil {
		t.Fatal(err)
	}
	if generation != 2 || status != "open" {
		t.Fatalf("incident generation/status = %d/%q; want 2/open", generation, status)
	}
	var events, sent int
	if err := db.QueryRow(`SELECT COUNT(*),COUNT(*) FILTER (WHERE dispatch_status='sent') FROM payment_reconciliation_incident_events WHERE incident_key=$1`, "stripe-event-reconciliation").Scan(&events, &sent); err != nil {
		t.Fatal(err)
	}
	if events != 3 || sent != 3 || client.calls.Load() != 4 {
		t.Fatalf("incident events=%d sent=%d webhook calls=%d; want 3 delivered transitions and one retry", events, sent, client.calls.Load())
	}
	var attempts int
	if err := db.QueryRow(`SELECT dispatch_attempts FROM payment_reconciliation_incident_events WHERE incident_key=$1 AND generation=1 AND transition='opened'`, "stripe-event-reconciliation").Scan(&attempts); err != nil || attempts != 2 {
		t.Fatalf("opened alert attempts=%d err=%v; want one retry", attempts, err)
	}
}
