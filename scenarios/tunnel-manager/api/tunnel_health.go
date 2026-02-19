package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// TunnelStatus holds the overall tunnel health assessment.
type TunnelStatus struct {
	Status       string `json:"status"` // "healthy", "degraded", "unhealthy"
	Systemd      string `json:"systemd"`
	Ready        string `json:"ready"`
	ReadyLatency int    `json:"ready_latency_ms,omitempty"`
	Score        int    `json:"score"` // 0-100 composite health score
	Message      string `json:"message,omitempty"`
	CheckedAt    string `json:"checked_at"`
}

// TunnelHealthChecker monitors cloudflared tunnel health.
type TunnelHealthChecker struct {
	metricsURL string // cloudflared metrics endpoint (default: http://127.0.0.1:20241)
	cmdRunner  func(ctx context.Context, name string, args ...string) ([]byte, error)
	httpClient *http.Client
}

type TunnelHealthOption func(*TunnelHealthChecker)

func WithMetricsURL(url string) TunnelHealthOption {
	return func(thc *TunnelHealthChecker) { thc.metricsURL = url }
}

func WithCmdRunner(fn func(ctx context.Context, name string, args ...string) ([]byte, error)) TunnelHealthOption {
	return func(thc *TunnelHealthChecker) { thc.cmdRunner = fn }
}

func NewTunnelHealthChecker(opts ...TunnelHealthOption) *TunnelHealthChecker {
	thc := &TunnelHealthChecker{
		metricsURL: "http://127.0.0.1:20241",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
	thc.cmdRunner = defaultCmdRunner
	for _, opt := range opts {
		opt(thc)
	}
	return thc
}

func defaultCmdRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Check runs all tunnel health checks and returns a composite status.
func (thc *TunnelHealthChecker) Check(ctx context.Context) TunnelStatus {
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

	return TunnelStatus{
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

// --- HTTP Handler ---

func handleTunnelHealth(checker *TunnelHealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		status := checker.Check(ctx)
		writeJSON(w, http.StatusOK, status)
	}
}
