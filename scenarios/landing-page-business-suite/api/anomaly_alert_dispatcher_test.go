package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func insertAnomalyRow(t *testing.T, svc *PaymentAnomalyService) int64 {
	t.Helper()
	var id int64
	err := svc.db.QueryRowContext(context.Background(), `
		INSERT INTO payment_anomaly_log (anomaly_type, severity, dispatch_status, created_at)
		VALUES ('t_test', 'warn', 'pending', NOW())
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("seed anomaly row: %v", err)
	}
	return id
}

func TestDispatch_RetriesOn5xx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	var calls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer stub.Close()

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	svc.dispatcher.backoffs = []time.Duration{5 * time.Millisecond, 5 * time.Millisecond}
	rowID := insertAnomalyRow(t, svc)

	svc.dispatcher.Dispatch(context.Background(), anomalyDispatchPayload{
		ID:         rowID,
		Type:       "t_test",
		Severity:   "warn",
		WebhookURL: stub.URL,
		CreatedAt:  time.Now(),
	})

	if calls.Load() != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls.Load())
	}
	var status string
	if err := db.QueryRow(`SELECT dispatch_status FROM payment_anomaly_log WHERE id = $1`, rowID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != anomalyDispatchSent {
		t.Fatalf("status: %q", status)
	}
}

func TestDispatch_NoRetryOn4xx(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	var calls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer stub.Close()

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	svc.dispatcher.backoffs = []time.Duration{5 * time.Millisecond}
	rowID := insertAnomalyRow(t, svc)

	svc.dispatcher.Dispatch(context.Background(), anomalyDispatchPayload{
		ID: rowID, Type: "t_test", Severity: "warn", WebhookURL: stub.URL, CreatedAt: time.Now(),
	})

	if calls.Load() != 1 {
		t.Fatalf("expected 1 attempt, got %d", calls.Load())
	}
	var status, errStr string
	var attempts int
	if err := db.QueryRow(`SELECT dispatch_status, dispatch_attempts, COALESCE(dispatch_error, '') FROM payment_anomaly_log WHERE id = $1`, rowID).
		Scan(&status, &attempts, &errStr); err != nil {
		t.Fatal(err)
	}
	if status != anomalyDispatchFailed {
		t.Fatalf("status: %q", status)
	}
	if attempts != 1 {
		t.Fatalf("attempts: %d", attempts)
	}
	if errStr == "" {
		t.Fatal("expected dispatch_error recorded")
	}
}

func TestDispatch_RetryExhausted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	var calls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer stub.Close()

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	svc.dispatcher.backoffs = []time.Duration{5 * time.Millisecond, 5 * time.Millisecond}
	rowID := insertAnomalyRow(t, svc)

	svc.dispatcher.Dispatch(context.Background(), anomalyDispatchPayload{
		ID: rowID, Type: "t_test", Severity: "warn", WebhookURL: stub.URL, CreatedAt: time.Now(),
	})

	if calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls.Load())
	}
	var status string
	var attempts int
	if err := db.QueryRow(`SELECT dispatch_status, dispatch_attempts FROM payment_anomaly_log WHERE id = $1`, rowID).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != anomalyDispatchFailed || attempts != 3 {
		t.Fatalf("status=%q attempts=%d", status, attempts)
	}
}

func TestDispatch_EmptyURLNoop(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	resetAnomalyTestData(t, db)

	svc := NewPaymentAnomalyService(context.Background(), db, context.Background())
	rowID := insertAnomalyRow(t, svc)
	svc.dispatcher.Dispatch(context.Background(), anomalyDispatchPayload{ID: rowID, Type: "t_test"})
	var status string
	if err := db.QueryRow(`SELECT dispatch_status FROM payment_anomaly_log WHERE id = $1`, rowID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != anomalyDispatchPending {
		t.Fatalf("empty URL should no-op; got %q", status)
	}
}
