package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:HEALTH-001] Systemd check edge cases

func TestTunnelHealth_SystemdCommandError(t *testing.T) {
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("command not found")
		}),
		WithMetricsURL("http://127.0.0.1:1"),
	)

	status := checker.Check(context.Background())
	if status.Systemd == "active" {
		t.Error("systemd should not be active when command fails")
	}
}

// [REQ:HEALTH-002] Ready endpoint edge cases

func TestTunnelHealth_MetricsURLUnreachable(t *testing.T) {
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL("http://127.0.0.1:1"), // unreachable
	)

	status := checker.Check(context.Background())
	if status.Ready == "ok" {
		t.Error("ready should not be ok when metrics URL is unreachable")
	}
}

func TestTunnelHealth_MetricsServer503(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL(ts.URL),
	)

	status := checker.Check(context.Background())
	if status.Ready == "ok" {
		t.Error("ready should not be ok when metrics returns 503")
	}
}

// [REQ:HEALTH-006] Composite health score edge cases

func TestTunnelHealth_ScorePartiallyHealthy(t *testing.T) {
	// Systemd active but metrics unreachable → partial health
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}),
		WithMetricsURL("http://127.0.0.1:1"),
	)

	status := checker.Check(context.Background())
	if status.Score >= 100 {
		t.Errorf("score = %d, should be less than 100 with unreachable metrics", status.Score)
	}
	if status.Score <= 0 {
		t.Errorf("score = %d, should be positive with active systemd", status.Score)
	}
}

// [REQ:OBS-004] Detailed health handler with tunnel info

func TestDetailedHealth_IncludesTunnelStatus(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)
	probeSvc := NewProbeService(db, routeSvc)

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

	handler := handleDetailedHealth(checker, routeSvc, probeSvc)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("overall status = %q, want healthy", resp.Status)
	}
	if resp.Tunnel.Score != 100 {
		t.Errorf("tunnel score = %d, want 100", resp.Tunnel.Score)
	}
	if resp.Tunnel.Ready != "ok" {
		t.Errorf("tunnel ready = %q, want ok", resp.Tunnel.Ready)
	}
}

func TestDetailedHealth_UnhealthyTunnel(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)
	probeSvc := NewProbeService(db, routeSvc)

	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("inactive\n"), nil
		}),
		WithMetricsURL("http://127.0.0.1:1"),
	)

	handler := handleDetailedHealth(checker, routeSvc, probeSvc)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("overall status = %q, want unhealthy", resp.Status)
	}
}

// [REQ:HEALTH-001] handleTunnelHealth handler test

func TestHandleTunnelHealth(t *testing.T) {
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

	handler := handleTunnelHealth(checker)
	req := httptest.NewRequest("GET", "/api/v1/tunnel/health", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var status TunnelStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if status.Status != "healthy" {
		t.Errorf("status = %q, want healthy", status.Status)
	}
}
