package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/opsalert"
)

func TestDispatchEmailDeliveryAlertIsIdempotentAndSecretFree(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	alertType := "email_provider_quota_high"
	if _, err := server.db.Exec(`DELETE FROM auth_alert_log WHERE alert_type = $1`, alertType); err != nil {
		t.Fatalf("clean alert log: %v", err)
	}

	requests := make(chan []byte, 1)
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests <- body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer webhook.Close()

	details := map[string]any{"provider": "mailgun", "used": 80, "ceiling": 100}
	for i := 0; i < 2; i++ {
		if err := server.dispatchEmailDeliveryAlert(context.Background(), opsalert.New(), webhook.URL, alertType, details); err != nil {
			t.Fatalf("dispatch %d: %v", i, err)
		}
	}

	var rows int
	if err := server.db.QueryRow(`SELECT COUNT(*) FROM auth_alert_log WHERE alert_type = $1`, alertType).Scan(&rows); err != nil {
		t.Fatalf("count alerts: %v", err)
	}
	if rows != 1 {
		t.Errorf("alert rows = %d, want 1 per window", rows)
	}
	var status string
	if err := server.db.QueryRow(`SELECT dispatch_status FROM auth_alert_log WHERE alert_type = $1`, alertType).Scan(&status); err != nil {
		t.Fatalf("read alert status: %v", err)
	}
	if status != "sent" {
		t.Errorf("dispatch_status = %q, want sent", status)
	}
	select {
	case body := <-requests:
		payload := string(body)
		if strings.Contains(payload, "password") || strings.Contains(payload, "api_key") || strings.Contains(payload, "token") {
			t.Errorf("alert payload appears to contain secret material: %s", payload)
		}
	default:
		t.Fatal("webhook did not receive the alert")
	}
}
