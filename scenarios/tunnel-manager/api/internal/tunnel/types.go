// Package tunnel is the domain-scoped home for tunnel-wide health and the
// scraped Prometheus metrics time-series — a truthful, tunnel-wide signal
// distinct from the per-route liveness probes the routes domain owns.
//
// Layering mirrors the canonical Vrooli pattern (see internal/notes for the
// worked template reference):
//
//	HTTP → handler → Service (composes health, parses, persists) → Repository
//	                     ↑                                            ↑
//	                     FakeService (handler tests)                  FakeRepository (service tests)
//	                                                                  Real sqlite (repository tests)
//
// types.go owns the domain entities and the typed sentinel handlers translate
// at the transport edge. The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is the only
// translation point (api-steer §7).
package tunnel

import "time"

// Status classifies the composite tunnel health.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// TunnelStatus is the composite health snapshot derived from the cloudflared
// systemd unit state and its /ready probe. It is computed on demand and never
// persisted — the metrics time-series is the durable record.
type TunnelStatus struct {
	// Status is "healthy", "degraded", or "unhealthy".
	Status Status
	// Systemd is the cloudflared unit state (e.g. "active", "inactive").
	Systemd string
	// Ready is the /ready probe result (e.g. "ok", "unreachable", "http_503").
	Ready string
	// ReadyLatencyMS is the /ready probe latency in milliseconds.
	ReadyLatencyMS int
	// Score is the composite 0-100 health score.
	Score int
	// Message is a human-readable detail; empty when healthy.
	Message string
	// CheckedAt is when the snapshot was taken.
	CheckedAt time.Time
}

// MetricsSample is one scraped cloudflared Prometheus sample. Persisted to the
// metrics table so the UI can render a time-series and recovery can detect
// degraded mode (HA connections dropping, RTT spikes).
type MetricsSample struct {
	ID            string
	HAConnections int
	RequestErrors float64
	ActiveStreams int
	SmoothedRTTMS float64
	ScrapedAt     time.Time
}

// ErrNoMetrics is the typed sentinel returned when the metrics table is empty
// (Latest with no rows). Handlers treat it as a soft "no sample yet" rather
// than an error — GetStatus returns a nil latest sample.
type ErrNoMetrics struct{}

func (ErrNoMetrics) Error() string { return "no metrics samples recorded" }
