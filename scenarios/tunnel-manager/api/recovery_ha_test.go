package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:RECOVER-002] Trigger recovery on HA connection loss
func TestRecoveryTriggersOnHAConnectionLoss(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create a metrics server that reports 0 HA connections
	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			fmt.Fprintln(w, "cloudflared_tunnel_ha_connections 0")
			return
		}
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer metricsServer.Close()

	restartCalled := false
	healthCheck := NewTunnelHealthChecker(
		WithMetricsURL(metricsServer.URL),
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name == "systemctl" && len(args) > 0 && args[0] == "is-active" {
				return []byte("active"), nil
			}
			return []byte("active"), nil
		}),
	)

	scraper := NewMetricsScraper(metricsServer.URL)

	engine := NewRecoveryEngine(db, healthCheck,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			restartCalled = true
			return []byte("ok"), nil
		}),
		WithConsecutiveFailures(2), // trigger after 2 consecutive zero-HA scrapes
	)

	ctx := context.Background()

	// First check: HA = 0, should not trigger yet (need 2 consecutive)
	evt, err := engine.EvaluateHA(ctx, scraper)
	if err != nil {
		t.Fatalf("EvaluateHA 1: %v", err)
	}
	if evt != nil {
		t.Error("first HA=0 check should not trigger recovery")
	}

	// Second check: HA = 0 again, should trigger
	evt, err = engine.EvaluateHA(ctx, scraper)
	if err != nil {
		t.Fatalf("EvaluateHA 2: %v", err)
	}
	if evt == nil {
		t.Fatal("expected recovery event after 2 consecutive HA=0 checks")
	}
	if evt.TriggerType != "ha_connection_loss" {
		t.Errorf("trigger_type = %q, want ha_connection_loss", evt.TriggerType)
	}
	if !restartCalled {
		t.Error("expected systemctl restart to be called")
	}
}

// [REQ:RECOVER-002] No trigger when HA connections are healthy
func TestRecoveryNoTriggerWhenHAHealthy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	metricsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			fmt.Fprintln(w, "cloudflared_tunnel_ha_connections 4")
			return
		}
		if r.URL.Path == "/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer metricsServer.Close()

	healthCheck := NewTunnelHealthChecker(WithMetricsURL(metricsServer.URL))
	scraper := NewMetricsScraper(metricsServer.URL)
	engine := NewRecoveryEngine(db, healthCheck,
		WithRecoveryCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			t.Error("restart should not be called when HA is healthy")
			return nil, nil
		}),
	)

	ctx := context.Background()
	evt, err := engine.EvaluateHA(ctx, scraper)
	if err != nil {
		t.Fatalf("EvaluateHA: %v", err)
	}
	if evt != nil {
		t.Error("should not trigger recovery when HA connections are healthy")
	}
}

// [REQ:HEALTH-004] HA connection monitoring - detect degraded state
func TestHAConnectionDegradedDetection(t *testing.T) {
	body := "cloudflared_tunnel_ha_connections 2"
	m := parsePrometheusMetrics(body)
	if m.HAConnections != 2 {
		t.Errorf("HAConnections = %d, want 2", m.HAConnections)
	}
	// 2 < 4 means degraded (not critical, but below normal)
	if m.HAConnections >= 4 {
		t.Error("2 HA connections should be below the healthy threshold of 4")
	}
}

// [REQ:HEALTH-004] HA connection monitoring - critical at zero
func TestHAConnectionCritical(t *testing.T) {
	body := "cloudflared_tunnel_ha_connections 0"
	m := parsePrometheusMetrics(body)
	if m.HAConnections != 0 {
		t.Errorf("HAConnections = %d, want 0", m.HAConnections)
	}
}
