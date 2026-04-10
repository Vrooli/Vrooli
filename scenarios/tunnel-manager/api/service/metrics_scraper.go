package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tunnel-manager/domain"
)

// MetricsScraper fetches and parses cloudflared Prometheus metrics.
type MetricsScraper struct {
	metricsURL string
	httpClient *http.Client
}

func NewMetricsScraper(metricsURL string) *MetricsScraper {
	if metricsURL == "" {
		metricsURL = DefaultMetricsURL
	}
	return &MetricsScraper{
		metricsURL: metricsURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// Scrape fetches /metrics and parses key gauges/counters.
func (ms *MetricsScraper) Scrape(ctx context.Context) (*domain.TunnelMetrics, error) {
	url := ms.metricsURL + "/metrics"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("metrics request: %w", err)
	}

	resp, err := ms.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("metrics fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics endpoint returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read metrics body: %w", err)
	}

	return ParsePrometheusMetrics(string(body)), nil
}

// ParsePrometheusMetrics extracts key metrics from Prometheus text format.
func ParsePrometheusMetrics(body string) *domain.TunnelMetrics {
	m := &domain.TunnelMetrics{
		ScrapedAt: time.Now().UTC().Format(time.RFC3339),
	}

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
			m.SmoothedRTT = parseFloatMetric(line)
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
