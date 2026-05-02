// Package fixtures provides domain-object factories for tests.
//
// Each NewX returns a sane default instance plus functional options for
// the unusual fields. Default values are picked so the most common test
// path is `fixtures.NewX()` with no opts.
//
// Naming convention: factories use the functional-options pattern for
// stable defaults + selective override. Mirrors workspace-sandbox's
// fixtures package; future scenarios add NewSandbox-style factories
// alongside as their domain grows.
//
// # Wire-shape parity
//
// Fixture structs mirror the upstream wire shape (e.g. api-core/health
// `Response`) field-for-field with matching JSON tags so tests can
// `json.Unmarshal` the canonical handler response straight into the
// fixture and assert on typed fields instead of `map[string]any`. Drift
// here would silently break decoding; keep tags in lockstep with the
// upstream type.
package fixtures

import "time"

// DependencyStatus mirrors api-core/health.DependencyStatus. Field names
// and JSON tags match exactly so a wire response decodes into this
// struct unmodified. Pointer types preserve omitempty semantics for
// optional numeric fields.
type DependencyStatus struct {
	Connected bool     `json:"connected"`
	Latency   *float64 `json:"latency_ms,omitempty"`
	Error     any      `json:"error,omitempty"`
	Database  string   `json:"database,omitempty"`
}

// HealthResponse mirrors api-core/health.Response. Tests decode the
// /health endpoint body into this struct via
// `assertx.MustDecodeJSON[fixtures.HealthResponse]` and assert on
// typed fields. The fixture also doubles as a builder for tests that
// construct payloads inbound (uncommon today; useful when scenarios
// add downstream consumers that read /health).
type HealthResponse struct {
	Status        string                      `json:"status"`
	Service       string                      `json:"service"`
	Timestamp     string                      `json:"timestamp"`
	Readiness     bool                        `json:"readiness"`
	Version       string                      `json:"version,omitempty"`
	UptimeSeconds float64                     `json:"uptime_seconds,omitempty"`
	Dependencies  map[string]DependencyStatus `json:"dependencies,omitempty"`
	Metrics       map[string]any              `json:"metrics,omitempty"`
}

// HealthOpt mutates a HealthResponse during construction.
type HealthOpt func(*HealthResponse)

// NewHealthResponse returns a healthy response with stable test
// defaults. Override individual fields via opts. Defaults are picked
// so the most common test path is `NewHealthResponse()` with no opts.
func NewHealthResponse(opts ...HealthOpt) HealthResponse {
	r := HealthResponse{
		Status:    "healthy",
		Service:   "react-vite-test",
		Version:   "1.0.0",
		Readiness: true,
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

// WithHealthStatus overrides the default "healthy" status.
func WithHealthStatus(s string) HealthOpt {
	return func(r *HealthResponse) { r.Status = s }
}

// WithHealthService overrides the default service name.
func WithHealthService(s string) HealthOpt {
	return func(r *HealthResponse) { r.Service = s }
}

// WithHealthVersion overrides the default version string.
func WithHealthVersion(v string) HealthOpt {
	return func(r *HealthResponse) { r.Version = v }
}

// WithHealthReadiness overrides the default true readiness flag. Tests
// asserting the unhealthy branch pair this with `WithHealthStatus("unhealthy")`.
func WithHealthReadiness(b bool) HealthOpt {
	return func(r *HealthResponse) { r.Readiness = b }
}

// WithHealthTimestamp overrides the default 2026-01-01 timestamp. The
// caller passes a `time.Time` for ergonomics; the fixture stores the
// RFC3339 wire form so JSON round-trips byte-identically.
func WithHealthTimestamp(t time.Time) HealthOpt {
	return func(r *HealthResponse) { r.Timestamp = t.Format(time.RFC3339) }
}

// WithHealthDependency adds (or overrides) a named dependency entry on
// the response. The Dependencies map is lazily initialised so callers
// passing a single dependency don't have to construct the map first.
func WithHealthDependency(name string, ds DependencyStatus) HealthOpt {
	return func(r *HealthResponse) {
		if r.Dependencies == nil {
			r.Dependencies = make(map[string]DependencyStatus)
		}
		r.Dependencies[name] = ds
	}
}
