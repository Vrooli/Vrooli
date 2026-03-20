package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tunnel-manager/adapter"
	"tunnel-manager/domain"
)

// DefaultMetricsURL is the default cloudflared Prometheus metrics endpoint.
const DefaultMetricsURL = "http://127.0.0.1:20241"

// TunnelHealthChecker monitors cloudflared tunnel health.
type TunnelHealthChecker struct {
	metricsURL string
	cmdRunner  adapter.CmdRunner
	httpClient *http.Client
}

type TunnelHealthOption func(*TunnelHealthChecker)

func WithMetricsURL(url string) TunnelHealthOption {
	return func(thc *TunnelHealthChecker) { thc.metricsURL = url }
}

func WithCmdRunner(fn adapter.CmdRunner) TunnelHealthOption {
	return func(thc *TunnelHealthChecker) { thc.cmdRunner = fn }
}

func NewTunnelHealthChecker(opts ...TunnelHealthOption) *TunnelHealthChecker {
	thc := &TunnelHealthChecker{
		metricsURL: DefaultMetricsURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
	thc.cmdRunner = adapter.DefaultCmdRunner
	for _, opt := range opts {
		opt(thc)
	}
	return thc
}

// Check runs all tunnel health checks and returns a composite status.
func (thc *TunnelHealthChecker) Check(ctx context.Context) domain.TunnelStatus {
	now := time.Now().UTC().Format(time.RFC3339)
	score := 100

	// 1. Check systemd service status
	systemdStatus := thc.checkSystemd(ctx)
	if systemdStatus != "active" {
		score -= 50
	}

	// 2. Check /ready endpoint
	readyStatus, readyLatency := thc.checkReady(ctx)
	if readyStatus != "ok" {
		score -= 30
	}

	// Determine composite status
	status := "healthy"
	var message string
	if score <= 20 {
		status = "unhealthy"
		message = "tunnel is down or critically degraded"
	} else if score <= 70 {
		status = "degraded"
		message = "tunnel is experiencing issues"
	}

	return domain.TunnelStatus{
		Status:       status,
		Systemd:      systemdStatus,
		Ready:        readyStatus,
		ReadyLatency: readyLatency,
		Score:        score,
		Message:      message,
		CheckedAt:    now,
	}
}

func (thc *TunnelHealthChecker) checkSystemd(ctx context.Context) string {
	out, err := thc.cmdRunner(ctx, "systemctl", "is-active", "cloudflared")
	if err != nil {
		return "inactive"
	}
	result := strings.TrimSpace(string(out))
	if result == "active" {
		return "active"
	}
	return result
}

func (thc *TunnelHealthChecker) checkReady(ctx context.Context) (string, int) {
	url := thc.metricsURL + "/ready"
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "error", 0
	}
	resp, err := thc.httpClient.Do(req)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		return "unreachable", latency
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "ok", latency
	}
	return fmt.Sprintf("http_%d", resp.StatusCode), latency
}

// PollReady polls the tunnel's /ready endpoint until it returns "ok" or
// the timeout elapses. Returns nil on success, error on timeout.
func PollReady(ctx context.Context, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		checker := NewTunnelHealthChecker()
		status := checker.Check(ctx)
		if status.Ready == "ok" {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("cloudflared did not become ready within %v after restart", timeout)
}
