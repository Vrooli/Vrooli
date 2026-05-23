// Package fixtures provides domain-object factories for tests.
//
// Each NewX returns a sane default instance plus functional options for
// the unusual fields. Default values are picked so the most common test
// path is `fixtures.NewX()` with no opts.
//
// # Wire shape lives in proto, not here
//
// The HealthResponse type is the GENERATED proto message at
// `packages/proto/gen/go/typescript-code-graph/v1/health/health.pb.go`. The
// fixture re-exports it so callers don't have to type the long generated
// import path, and provides functional-options builders for the fields
// scenarios most commonly override in tests.
//
// Adding a new option: add another `WithHealth*` helper here. The
// underlying type is the proto, so JSON tags, presence semantics, and
// the wire shape itself live in `packages/proto/schemas/typescript-code-graph/v1/health/health.proto`
// — never duplicate them in this file.
package fixtures

import (
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/health"
)

// HealthResponse is the canonical wire-shape type for /health responses.
// Re-exported so test code reads `fixtures.HealthResponse` instead of
// `healthv1.Response` — same underlying type either way.
type HealthResponse = healthv1.Response

// DependencyStatus mirrors the per-dependency shape returned in the
// `dependencies` map on a HealthResponse. Re-exported for the same
// reason as HealthResponse: short, intent-revealing alias at the call
// site.
type DependencyStatus = healthv1.DependencyStatus

// HealthOpt mutates a HealthResponse during construction. Composable
// across calls to NewHealthResponse so each test arranges only the
// fields it cares about.
type HealthOpt func(*healthv1.Response)

// NewHealthResponse returns a healthy response with stable test
// defaults. Override individual fields via opts. Defaults are picked
// so the most common test path is `NewHealthResponse()` with no opts.
func NewHealthResponse(opts ...HealthOpt) *healthv1.Response {
	r := &healthv1.Response{
		Status:    "healthy",
		Service:   "react-vite-test",
		Version:   "1.0.0",
		Readiness: true,
		Timestamp: "2026-01-01T00:00:00Z",
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithHealthStatus overrides the default "healthy" status.
func WithHealthStatus(s string) HealthOpt {
	return func(r *healthv1.Response) { r.Status = s }
}

// WithHealthService overrides the default service name.
func WithHealthService(s string) HealthOpt {
	return func(r *healthv1.Response) { r.Service = s }
}

// WithHealthVersion overrides the default version string.
func WithHealthVersion(v string) HealthOpt {
	return func(r *healthv1.Response) { r.Version = v }
}

// WithHealthReadiness overrides the default true readiness flag. Tests
// asserting the unhealthy branch pair this with `WithHealthStatus("unhealthy")`.
func WithHealthReadiness(b bool) HealthOpt {
	return func(r *healthv1.Response) { r.Readiness = b }
}

// WithHealthTimestamp overrides the default 2026-01-01 timestamp.
// Callers pass an RFC3339-formatted string to match the wire shape; the
// proto stores Timestamp as a string for round-trip parity with
// api-core/health.Response.
func WithHealthTimestamp(rfc3339 string) HealthOpt {
	return func(r *healthv1.Response) { r.Timestamp = rfc3339 }
}

// WithHealthDependency adds (or overrides) a named dependency entry on
// the response. The Dependencies map is lazily initialised so callers
// passing a single dependency don't have to construct the map first.
func WithHealthDependency(name string, ds *healthv1.DependencyStatus) HealthOpt {
	return func(r *healthv1.Response) {
		if r.Dependencies == nil {
			r.Dependencies = make(map[string]*healthv1.DependencyStatus)
		}
		r.Dependencies[name] = ds
	}
}
