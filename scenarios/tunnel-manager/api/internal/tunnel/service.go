package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tunnel-manager/internal/clock"
	"tunnel-manager/internal/cmdrunner"
	"tunnel-manager/internal/httpc"
)

// DefaultMetricsEndpoint is the cloudflared Prometheus metrics base URL used
// when one is not supplied. The /ready and /metrics paths hang off it.
const DefaultMetricsEndpoint = "http://127.0.0.1:20241"

// Health-score penalties and thresholds. Ported verbatim from the pre-1.0
// tunnel-manager (api/service/tunnel_health.go): systemd-not-active costs 50,
// a failing /ready probe costs 30; score <= 20 is unhealthy, <= 70 degraded.
const (
	scoreMax           = 100
	penaltySystemdDown = 50
	penaltyReadyFail   = 30
	thresholdUnhealthy = 20
	thresholdDegraded  = 70
)

// Service is the application-layer surface the tunnel handlers depend on. Owns
// composite-health computation, Prometheus parsing, and persistence policy.
// The handler is intentionally thin around it: decode → call service →
// translate errors.
type Service interface {
	// GetStatus computes the current composite tunnel health (cloudflared
	// systemd unit state + /ready probe) and returns it alongside the most
	// recent persisted metrics sample. The latest sample is nil when none has
	// been scraped yet (ErrNoMetrics is swallowed, not surfaced).
	GetStatus(ctx context.Context) (TunnelStatus, *MetricsSample, error)

	// ListMetrics returns persisted samples scraped within [from, to].
	ListMetrics(ctx context.Context, from, to time.Time) ([]MetricsSample, error)

	// Scrape fetches /metrics once, parses the key gauges/counters, persists
	// the sample, and returns it.
	Scrape(ctx context.Context) (MetricsSample, error)
}

type service struct {
	repo     MetricsRepository
	runner   cmdrunner.Runner
	doer     httpc.Doer
	clock    clock.Clock
	endpoint string
}

// NewService constructs the production Service. endpoint is the cloudflared
// metrics base URL; an empty value falls back to DefaultMetricsEndpoint.
func NewService(repo MetricsRepository, runner cmdrunner.Runner, doer httpc.Doer, clk clock.Clock, endpoint string) Service {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = DefaultMetricsEndpoint
	}
	return &service{
		repo:     repo,
		runner:   runner,
		doer:     doer,
		clock:    clk,
		endpoint: strings.TrimRight(endpoint, "/"),
	}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) GetStatus(ctx context.Context) (TunnelStatus, *MetricsSample, error) {
	score := scoreMax

	systemd := s.checkSystemd(ctx)
	if systemd != "active" {
		score -= penaltySystemdDown
	}

	ready, readyLatency := s.checkReady(ctx)
	if ready != "ok" {
		score -= penaltyReadyFail
	}

	status := StatusHealthy
	var message string
	switch {
	case score <= thresholdUnhealthy:
		status = StatusUnhealthy
		message = "tunnel is down or critically degraded"
	case score <= thresholdDegraded:
		status = StatusDegraded
		message = "tunnel is experiencing issues"
	}

	snapshot := TunnelStatus{
		Status:         status,
		Systemd:        systemd,
		Ready:          ready,
		ReadyLatencyMS: readyLatency,
		Score:          score,
		Message:        message,
		CheckedAt:      s.clock.Now().UTC(),
	}

	latest, err := s.repo.Latest(ctx)
	if err != nil {
		var noMetrics ErrNoMetrics
		if errors.As(err, &noMetrics) {
			return snapshot, nil, nil
		}
		return TunnelStatus{}, nil, err
	}
	return snapshot, &latest, nil
}

func (s *service) ListMetrics(ctx context.Context, from, to time.Time) ([]MetricsSample, error) {
	return s.repo.Query(ctx, from, to)
}

func (s *service) Scrape(ctx context.Context) (MetricsSample, error) {
	body, err := s.fetch(ctx, "/metrics")
	if err != nil {
		return MetricsSample{}, fmt.Errorf("scrape metrics: %w", err)
	}
	sample := parsePrometheusMetrics(body)
	sample.ScrapedAt = s.clock.Now().UTC()
	return s.repo.Store(ctx, sample)
}

// checkSystemd queries the cloudflared unit state via `systemctl is-active`. A
// runner error (unit missing, systemctl absent) is reported as "inactive" so
// the score penalty still applies.
func (s *service) checkSystemd(ctx context.Context) string {
	out, err := s.runner(ctx, "systemctl", "is-active", "cloudflared")
	if err != nil {
		return "inactive"
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "inactive"
	}
	return result
}

// checkReady probes the cloudflared /ready endpoint, returning a status token
// ("ok", "unreachable", "http_503", …) and the probe latency in milliseconds.
func (s *service) checkReady(ctx context.Context) (string, int) {
	start := s.clock.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+"/ready", nil)
	if err != nil {
		return "error", 0
	}
	resp, err := s.doer.Do(req)
	latency := int(s.clock.Now().Sub(start).Milliseconds())
	if err != nil {
		return "unreachable", latency
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "ok", latency
	}
	return fmt.Sprintf("http_%d", resp.StatusCode), latency
}

// fetch GETs path off the metrics endpoint and returns the body, mapping a
// non-200 to an error.
func (s *service) fetch(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+path, nil)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	resp, err := s.doer.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("endpoint returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return string(body), nil
}

// parsePrometheusMetrics extracts the four cloudflared metrics we track from
// the Prometheus text exposition format. Ported from the pre-1.0
// metrics_scraper.go; ScrapedAt is set by the caller via the clock seam.
func parsePrometheusMetrics(body string) MetricsSample {
	var m MetricsSample
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "cloudflared_tunnel_ha_connections "):
			m.HAConnections = parseIntMetric(line)
		case strings.HasPrefix(line, "cloudflared_tunnel_request_errors_total ") ||
			strings.HasPrefix(line, "cloudflared_tunnel_request_errors "):
			m.RequestErrors = parseFloatMetric(line)
		case strings.HasPrefix(line, "cloudflared_tunnel_active_streams "):
			m.ActiveStreams = parseIntMetric(line)
		case strings.HasPrefix(line, "quic_client_smoothed_rtt "):
			m.SmoothedRTTMS = parseFloatMetric(line)
		}
	}
	return m
}

func parseIntMetric(line string) int {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	v, _ := strconv.Atoi(parts[len(parts)-1])
	return v
}

func parseFloatMetric(line string) float64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return 0
	}
	v, _ := strconv.ParseFloat(parts[len(parts)-1], 64)
	return v
}
