package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
)

// [REQ:OBS-004] Detailed health handler with tunnel info

func TestDetailedHealth_IncludesTunnelStatus(t *testing.T) {
	checker := &mockHandlerTunnelChecker{
		checkFn: func(ctx context.Context) domain.TunnelStatus {
			return domain.TunnelStatus{Status: "healthy", Ready: "ok", Systemd: "active", Score: 100}
		},
	}
	lister := &mockHandlerRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}
	runner := &mockHandlerProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{}, nil
		},
	}

	h := HandleDetailedHealth(checker, lister, runner)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp domain.DetailedHealth
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
	checker := &mockHandlerTunnelChecker{
		checkFn: func(ctx context.Context) domain.TunnelStatus {
			return domain.TunnelStatus{Status: "unhealthy", Ready: "unreachable", Systemd: "inactive", Score: 0}
		},
	}
	lister := &mockHandlerRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}
	runner := &mockHandlerProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{}, nil
		},
	}

	h := HandleDetailedHealth(checker, lister, runner)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp domain.DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("overall status = %q, want unhealthy", resp.Status)
	}
}

// [REQ:HEALTH-001] handleTunnelHealth handler test

func TestHandleTunnelHealth(t *testing.T) {
	checker := &mockHandlerTunnelChecker{
		checkFn: func(ctx context.Context) domain.TunnelStatus {
			return domain.TunnelStatus{Status: "healthy", Ready: "ok", Systemd: "active", Score: 100}
		},
	}

	h := HandleTunnelHealth(checker)
	req := httptest.NewRequest("GET", "/api/v1/tunnel/health", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var status domain.TunnelStatus
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if status.Status != "healthy" {
		t.Errorf("status = %q, want healthy", status.Status)
	}
}

// [REQ:OBS-004] Health API endpoint tests

func TestDetailedHealth_EmptyRoutes(t *testing.T) {
	checker := &mockHandlerTunnelChecker{
		checkFn: func(ctx context.Context) domain.TunnelStatus {
			return domain.TunnelStatus{Status: "healthy", Ready: "ok", Systemd: "active", Score: 100}
		},
	}
	lister := &mockHandlerRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}
	runner := &mockHandlerProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{}, nil
		},
	}

	h := HandleDetailedHealth(checker, lister, runner)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp domain.DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	if len(resp.Routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(resp.Routes))
	}
}

func TestDetailedHealth_WithRoutes(t *testing.T) {
	checker := &mockHandlerTunnelChecker{
		checkFn: func(ctx context.Context) domain.TunnelStatus {
			return domain.TunnelStatus{Status: "healthy", Ready: "ok", Systemd: "active", Score: 100}
		},
	}
	lister := &mockHandlerRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{ID: 1, Subdomain: "app-a", ScenarioName: "scenario-a", LocalPort: 8080, Enabled: true},
				{ID: 2, Subdomain: "app-b", ScenarioName: "scenario-b", LocalPort: 9090, Enabled: true},
			}, nil
		},
	}
	runner := &mockHandlerProbeRunner{
		runAllFn: func(ctx context.Context) ([]domain.ProbeResult, error) {
			return []domain.ProbeResult{}, nil
		},
	}

	h := HandleDetailedHealth(checker, lister, runner)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp domain.DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(resp.Routes))
	}
}
