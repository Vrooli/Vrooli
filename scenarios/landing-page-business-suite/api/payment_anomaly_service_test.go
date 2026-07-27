package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// resetAnomalyTestData clears payment_anomaly_log and payment_settings between
// cases. Mirrors the style of resetStripeTestData.
func resetAnomalyTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DELETE FROM payment_anomaly_log`); err != nil {
		t.Fatalf("reset payment_anomaly_log: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM payment_settings`); err != nil {
		t.Fatalf("reset payment_settings: %v", err)
	}
}

// configureAnomalyWebhook writes a full-dispatch-enabled row into
// payment_settings and refreshes the service so cfg.Load() picks it up.
func configureAnomalyWebhook(t *testing.T, svc *PaymentAnomalyService, db *sql.DB, url string, rateLimits string) {
	t.Helper()
	if rateLimits == "" {
		rateLimits = "{}"
	}
	_, err := db.Exec(`
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits, updated_at)
		VALUES (1, '', '', '', $1, TRUE, $2::jsonb, NOW())
		ON CONFLICT (id) DO UPDATE SET
			anomaly_webhook_url = EXCLUDED.anomaly_webhook_url,
			anomaly_webhook_enabled = EXCLUDED.anomaly_webhook_enabled,
			anomaly_rate_limits = EXCLUDED.anomaly_rate_limits,
			updated_at = NOW()
	`, url, rateLimits)
	if err != nil {
		t.Fatalf("configure anomaly settings: %v", err)
	}
	if err := svc.RefreshConfig(context.Background()); err != nil {
		t.Fatalf("refresh anomaly config: %v", err)
	}
}

func TestSchemaApplied(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// payment_anomaly_log must exist.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'payment_anomaly_log'`).Scan(&count); err != nil {
		t.Fatalf("schema probe: %v", err)
	}
	if count != 1 {
		t.Fatalf("payment_anomaly_log not created")
	}

	// payment_settings must have the three new columns.
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'payment_settings'`)
	if err != nil {
		t.Fatalf("columns probe: %v", err)
	}
	defer rows.Close()
	got := map[string]bool{}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		got[c] = true
	}
	for _, c := range []string{"anomaly_webhook_url", "anomaly_webhook_enabled", "anomaly_rate_limits"} {
		if !got[c] {
			t.Fatalf("payment_settings missing column %s", c)
		}
	}
}

func TestMigration_FreshDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'intro_anomaly_log')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("intro_anomaly_log should be dropped on fresh boot")
	}
}

func TestRuntimeSchema_DoesNotPerformLegacyIntroAnomalyMigration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Re-create the legacy table as if we were an older DB upgrading.
	if _, err := db.Exec(`
		CREATE TABLE intro_anomaly_log (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			anomaly_type VARCHAR(50) NOT NULL,
			details JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		)
	`); err != nil {
		t.Fatalf("recreate legacy table: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO intro_anomaly_log (email, customer_id, coupon_id, anomaly_type, details)
		VALUES
			('a@example.com', 'cus_a', 'cp_a', 'repeat_intro_attempt', '{"stripe_event_id":"evt_a"}'::jsonb),
			('b@example.com', 'cus_b', 'cp_b', 'ineligible_intro',     '{"stripe_event_id":"evt_b"}'::jsonb)
	`); err != nil {
		t.Fatalf("seed legacy rows: %v", err)
	}

	// Baseline: payment_anomaly_log already exists from setupTestDB. Purge it
	// so we can prove declarative reconciliation does not copy historical rows.
	if _, err := db.Exec(`DELETE FROM payment_anomaly_log`); err != nil {
		t.Fatal(err)
	}

	if err := applyRuntimeSchema(db); err != nil {
		t.Fatalf("re-run runtime schema: %v", err)
	}

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'intro_anomaly_log')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("runtime schema must not drop legacy data; use the operator migration before deployment")
	}

	var migratedRows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM payment_anomaly_log`).Scan(&migratedRows); err != nil {
		t.Fatal(err)
	}
	if migratedRows != 0 {
		t.Fatalf("runtime schema must not copy legacy rows, got %d", migratedRows)
	}
}

func TestMigration_ReRun(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Second invocation must be a no-op.
	if err := applyRuntimeSchema(db); err != nil {
		t.Fatalf("re-run: %v", err)
	}
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'payment_anomaly_log')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("payment_anomaly_log should still exist")
	}
}

func TestLog_InsertsRow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())

	id, err := svc.Log(context.Background(), PaymentAnomaly{
		Type:        "intro_ineligible",
		Severity:    "error",
		Email:       "Test@Example.com",
		CustomerID:  "cus_123",
		SubjectID:   "cp_x",
		SubjectKind: "intro_coupon",
		Details:     map[string]interface{}{"stripe_event_id": "evt_1"},
	})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if id == 0 {
		t.Fatalf("expected non-zero id")
	}

	var (
		kind, severity, email, subjKind, status string
		detailsStr                              string
	)
	err = db.QueryRow(`SELECT anomaly_type, severity, email, subject_kind, dispatch_status, details::text FROM payment_anomaly_log WHERE id = $1`, id).
		Scan(&kind, &severity, &email, &subjKind, &status, &detailsStr)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "intro_ineligible" || severity != "error" {
		t.Fatalf("unexpected kind/severity: %q %q", kind, severity)
	}
	if email != "test@example.com" {
		t.Fatalf("email not normalised: %q", email)
	}
	if subjKind != "intro_coupon" {
		t.Fatalf("subject_kind: %q", subjKind)
	}
	if status != anomalyDispatchSkipped {
		t.Fatalf("dispatch_status: %q (expected skipped when disabled)", status)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(detailsStr), &parsed); err != nil {
		t.Fatalf("details not jsonb: %v", err)
	}
	if parsed["stripe_event_id"] != "evt_1" {
		t.Fatalf("details content: %v", parsed)
	}
}

func TestLog_NoDispatchWhenDisabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	id, err := svc.Log(context.Background(), PaymentAnomaly{Type: "disabled_case"})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	status, err := svc.WaitForDispatch(ctxWithTimeout(t, 2*time.Second), id)
	if err != nil {
		t.Fatalf("WaitForDispatch: %v", err)
	}
	if status != anomalyDispatchSkipped {
		t.Fatalf("expected skipped, got %q", status)
	}
}

func TestLog_DispatchesWhenEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	var received atomic.Int32
	lastBody := make(chan []byte, 1)
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		b := readAll(r)
		select {
		case lastBody <- b:
		default:
		}
		if ua := r.Header.Get("User-Agent"); ua != AnomalyDispatchUserAgent {
			t.Errorf("user-agent %q", ua)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type %q", ct)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	configureAnomalyWebhook(t, svc, db, stub.URL, "")

	id, err := svc.Log(context.Background(), PaymentAnomaly{
		Type:        "enabled_case",
		Email:       "a@b.com",
		SubjectID:   "cs_1",
		SubjectKind: "checkout_session",
		Details:     map[string]interface{}{"k": "v"},
	})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	status, err := svc.WaitForDispatch(ctxWithTimeout(t, 3*time.Second), id)
	if err != nil {
		t.Fatalf("WaitForDispatch: %v", err)
	}
	if status != anomalyDispatchSent {
		t.Fatalf("expected sent, got %q", status)
	}
	if received.Load() != 1 {
		t.Fatalf("expected 1 POST, got %d", received.Load())
	}
	select {
	case body := <-lastBody:
		var p map[string]any
		if err := json.Unmarshal(body, &p); err != nil {
			t.Fatalf("body not JSON: %v / %s", err, string(body))
		}
		if p["type"] != "enabled_case" {
			t.Fatalf("type: %v", p["type"])
		}
		subj, ok := p["subject"].(map[string]any)
		if !ok || subj["kind"] != "checkout_session" || subj["id"] != "cs_1" {
			t.Fatalf("subject: %v", p["subject"])
		}
		if p["scenario"] != paymentAnomalyScenarioName {
			t.Fatalf("scenario: %v", p["scenario"])
		}
	case <-time.After(time.Second):
		t.Fatal("no body captured")
	}
}

func TestLog_RateLimited(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	// burst=2 to make the assertion small and deterministic.
	configureAnomalyWebhook(t, svc, db, stub.URL, `{"rate_type":{"burst":2,"refill_seconds":3600}}`)

	var ids []int64
	for i := 0; i < 3; i++ {
		id, err := svc.Log(context.Background(), PaymentAnomaly{Type: "rate_type"})
		if err != nil {
			t.Fatalf("Log %d: %v", i, err)
		}
		ids = append(ids, id)
	}

	sent := 0
	skippedRL := 0
	for _, id := range ids {
		status, err := svc.WaitForDispatch(ctxWithTimeout(t, 3*time.Second), id)
		if err != nil {
			t.Fatalf("WaitForDispatch: %v", err)
		}
		switch status {
		case anomalyDispatchSent:
			sent++
		case anomalyDispatchSkipped:
			var reason sql.NullString
			if err := db.QueryRow(`SELECT dispatch_error FROM payment_anomaly_log WHERE id = $1`, id).Scan(&reason); err != nil {
				t.Fatal(err)
			}
			if reason.String == "rate_limited" {
				skippedRL++
			}
		}
	}
	if sent != 2 || skippedRL != 1 {
		t.Fatalf("expected 2 sent + 1 rate_limited skip, got sent=%d rl=%d", sent, skippedRL)
	}
}

func TestWaitForDispatch_Timeout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	block := make(chan struct{})
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-block:
		case <-r.Context().Done():
		}
	}))
	// Release handler goroutines before closing the server — otherwise
	// stub.Close blocks on in-flight requests (LIFO defer order matters here).
	defer stub.Close()
	defer close(block)

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	// Short per-attempt timeout so the goroutine doesn't linger past the test.
	svc.dispatcher.perAttempt = 500 * time.Millisecond
	svc.dispatcher.backoffs = []time.Duration{10 * time.Millisecond}
	svc.dispatcher.maxAttempts = 1
	configureAnomalyWebhook(t, svc, db, stub.URL, "")

	id, err := svc.Log(context.Background(), PaymentAnomaly{Type: "slow_type"})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := svc.WaitForDispatch(ctx, id); err == nil {
		t.Fatal("expected ctx-timeout error")
	}
}

func TestLogPaymentAnomaly_NilServer(t *testing.T) {
	if _, err := LogPaymentAnomaly(context.Background(), nil, PaymentAnomaly{Type: "x"}); err == nil {
		t.Fatal("expected error for nil server")
	}
}

// ---- helpers ----

func readAll(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return buf
		}
	}
}

func ctxWithTimeout(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

// ensure we're not accidentally sharing dispatcher state across tests — keeps
// the go vet tools happy about unused imports.
var (
	_ = sync.Mutex{}
	_ = fmt.Sprintf
)
