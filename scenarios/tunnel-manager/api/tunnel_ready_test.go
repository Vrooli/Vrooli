package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:HEALTH-002] Ready endpoint check
func TestTunnelHealthReadyEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	status := checker.Check(context.Background())
	if status.Ready != "ok" {
		t.Errorf("ready = %q, want ok", status.Ready)
	}
	if status.ReadyLatency < 0 {
		t.Error("expected non-negative ready latency")
	}
}
