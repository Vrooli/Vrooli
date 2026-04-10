package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleMetrics = `# HELP cloudflared_tunnel_ha_connections Number of HA connections
# TYPE cloudflared_tunnel_ha_connections gauge
cloudflared_tunnel_ha_connections 4
# HELP cloudflared_tunnel_request_errors_total Total number of request errors
# TYPE cloudflared_tunnel_request_errors_total counter
cloudflared_tunnel_request_errors_total 12
# HELP cloudflared_tunnel_active_streams Number of active streams
# TYPE cloudflared_tunnel_active_streams gauge
cloudflared_tunnel_active_streams 3
# HELP quic_client_smoothed_rtt Smoothed RTT in seconds
# TYPE quic_client_smoothed_rtt gauge
quic_client_smoothed_rtt 0.025
`

// [REQ:HEALTH-003] Prometheus metrics scraping
func TestMetricsScraperParsesPrometheus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(sampleMetrics))
	}))
	defer ts.Close()

	scraper := NewMetricsScraper(ts.URL)
	metrics, err := scraper.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}

	if metrics.HAConnections != 4 {
		t.Errorf("HAConnections = %d, want 4", metrics.HAConnections)
	}
	if metrics.RequestErrors != 12 {
		t.Errorf("RequestErrors = %f, want 12", metrics.RequestErrors)
	}
	if metrics.ActiveStreams != 3 {
		t.Errorf("ActiveStreams = %d, want 3", metrics.ActiveStreams)
	}
	if metrics.SmoothedRTT != 0.025 {
		t.Errorf("SmoothedRTT = %f, want 0.025", metrics.SmoothedRTT)
	}
	if metrics.ScrapedAt == "" {
		t.Error("expected non-empty ScrapedAt")
	}
}

// [REQ:HEALTH-003] Metrics scraper handles unreachable endpoint
func TestMetricsScraperUnreachable(t *testing.T) {
	scraper := NewMetricsScraper("http://127.0.0.1:1") // nothing listening
	_, err := scraper.Scrape(context.Background())
	if err == nil {
		t.Error("expected error for unreachable metrics endpoint")
	}
}

// [REQ:HEALTH-005] Error rate tracking from scraped metrics
func TestMetricsParseErrorRate(t *testing.T) {
	body := `cloudflared_tunnel_request_errors_total 42
cloudflared_tunnel_ha_connections 4
`
	m := ParsePrometheusMetrics(body)
	if m.RequestErrors != 42 {
		t.Errorf("RequestErrors = %f, want 42", m.RequestErrors)
	}
}
