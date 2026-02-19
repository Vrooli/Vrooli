package main

import (
	"context"
	"testing"
)

// [REQ:HEALTH-001] Systemd service status check
func TestTunnelHealthSystemdActive(t *testing.T) {
	checker := NewTunnelHealthChecker(
		WithCmdRunner(func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name == "systemctl" && len(args) > 0 && args[0] == "is-active" {
				return []byte("active\n"), nil
			}
			return nil, nil
		}),
		WithMetricsURL("http://127.0.0.1:1"), // unreachable on purpose
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
