package main

import (
	"context"
	"testing"

	"landing-page-business-suite-api/internal/providerprobe"
)

// A persistently broken credential must alert once per window, not once per
// probe: an hourly loop would otherwise page the operator every hour forever.
func TestDispatchProviderAlertIsIdempotentWithinAWindow(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	result := providerprobe.Result{
		Provider: "smtp", Capability: "customer sign-in email",
		Status: providerprobe.StatusFail, Detail: "535 Authentication failed",
	}
	alertType := "provider_credential_unusable:smtp"
	if _, err := server.db.Exec(`DELETE FROM auth_alert_log WHERE alert_type = $1`, alertType); err != nil {
		t.Fatalf("clean alert log: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := server.dispatchProviderAlert(context.Background(), nil, result); err != nil {
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
}

// With no webhook configured the failure is still recorded, and recorded as
// skipped rather than sent: the operator's report must not claim a delivery
// that never happened.
func TestDispatchProviderAlertRecordsSkippedWithoutAWebhook(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()
	alertType := "provider_credential_unusable:stripe"
	if _, err := server.db.Exec(`DELETE FROM auth_alert_log WHERE alert_type = $1`, alertType); err != nil {
		t.Fatalf("clean alert log: %v", err)
	}

	err := server.dispatchProviderAlert(context.Background(), nil, providerprobe.Result{
		Provider: "stripe", Capability: "card payments",
		Status: providerprobe.StatusFail, Detail: "Stripe rejected the secret key (HTTP 401)",
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	var status string
	if err := server.db.QueryRow(`SELECT dispatch_status FROM auth_alert_log WHERE alert_type = $1`, alertType).Scan(&status); err != nil {
		t.Fatalf("read alert: %v", err)
	}
	if status != "skipped" {
		t.Errorf("dispatch_status = %q, want skipped", status)
	}
}
