package service

import (
	"context"
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
		WithMetricsURL("http://127.0.0.1:1"),
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

// [REQ:HEALTH-006] Composite health score - healthy
func TestTunnelHealthScoreHealthy(t *testing.T) {
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
	if status.Status != "healthy" {
		t.Errorf("status = %q, want healthy", status.Status)
	}
	if status.Score != 100 {
		t.Errorf("score = %d, want 100", status.Score)
	}
}

// [REQ:HEALTH-006] Composite health score - unhealthy when systemd down
func TestTunnelHealthScoreUnhealthy(t *testing.T) {
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("inactive\n"), nil
		}),
		WithMetricsURL("http://127.0.0.1:1"),
	)

	status := checker.Check(context.Background())
	if status.Status != "unhealthy" {
		t.Errorf("status = %q, want unhealthy", status.Status)
	}
	if status.Score > 20 {
		t.Errorf("score = %d, should be <= 20", status.Score)
	}
}

// [REQ:HEALTH-001] Systemd service status check
func TestTunnelHealthSystemdActive(t *testing.T) {
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name == "systemctl" && len(args) > 0 && args[0] == "is-active" {
				return []byte("active\n"), nil
			}
			return nil, nil
		}),
		WithMetricsURL("http://127.0.0.1:1"),
	)

	status := checker.Check(context.Background())
	if status.Systemd != "active" {
		t.Errorf("systemd = %q, want active", status.Systemd)
	}
}

// [REQ:HEALTH-001] Systemd service status check - inactive
func TestTunnelHealthSystemdInactive(t *testing.T) {
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("inactive\n"), nil
		}),
		WithMetricsURL("http://127.0.0.1:1"),
	)

	status := checker.Check(context.Background())
	if status.Systemd != "inactive" {
		t.Errorf("systemd = %q, want inactive", status.Systemd)
	}
}
