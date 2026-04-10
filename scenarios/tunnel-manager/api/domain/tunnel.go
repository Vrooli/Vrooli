package domain

import "time"

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

// TunnelMetrics holds parsed Prometheus metrics from cloudflared.
type TunnelMetrics struct {
	HAConnections int     `json:"ha_connections"`
	RequestErrors float64 `json:"request_errors"`
	ActiveStreams int     `json:"active_streams"`
	SmoothedRTT   float64 `json:"smoothed_rtt_ms"`
	ScrapedAt     string  `json:"scraped_at"`
}

// MetricsRecord represents a stored metrics snapshot. [REQ:OBS-001]
type MetricsRecord struct {
	ID            int       `json:"id"`
	HAConnections int       `json:"ha_connections"`
	RequestErrors float64   `json:"request_errors"`
	ActiveStreams int       `json:"active_streams"`
	SmoothedRTT   float64   `json:"smoothed_rtt_ms"`
	ScrapedAt     time.Time `json:"scraped_at"`
}
