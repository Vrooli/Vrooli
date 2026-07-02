// Package httpc declares the outbound HTTP seam every scenario uses
// when calling external services.
//
// Production wires *http.Client directly — it satisfies Doer through
// the compile-time assertion below. Tests substitute mocks.FakeDoer to
// pin request shape and stub responses without touching the network.
//
// # Why ship the seam unwired in production
//
// The template has no production consumer of Doer (the notes
// endpoints are internal-only). Defining the seam *before* the first
// outbound call means the first scenario to need one — a webhook
// dispatcher, an upstream-API client, an OAuth handshake — copies the
// reference test's substitution pattern instead of inventing one. The
// cost of an unused interface is zero; the cost of a parallel
// reinvention is hours.
//
// Add the field to server.Deps when wiring the first consumer; the
// idiomatic shape is `Doer httpc.Doer` with `&http.Client{Timeout:
// 10 * time.Second}` constructed in main.go.
package httpc

import (
	"net/http"
	"time"
)

// DefaultDoer constructs the canonical production Doer with a long
// timeout suited for AI/vendor adapters. Adapter constructors call this
// instead of literal &http.Client{...} so the seam-registry "no raw
// http.Client outside httpc" gate can be enforced without forcing
// every callsite to thread Doer from main.go on day one.
func DefaultDoer() Doer {
	return &http.Client{Timeout: 120 * time.Second}
}

// LivenessDoer constructs a short-timeout Doer for UI/status health probes.
// Keep expensive readiness/quality checks on DefaultDoer; liveness should
// answer whether a dependency is reachable without waiting on model work.
func LivenessDoer() Doer {
	return &http.Client{Timeout: 3 * time.Second}
}

// seam: Doer is the outbound-HTTP seam (SEAMS.md row "httpc.Doer").
// Production wires *http.Client; tests wire mocks.FakeDoer.
//
// Doer is the canonical outbound HTTP interface. The single-method
// surface is intentional — handlers depend on what they need (Do)
// rather than the full *http.Client surface (Get / Post / CloseIdle…),
// which keeps the FakeDoer surface small and the seam easy to substitute.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Compile-time guarantee that *http.Client satisfies Doer. Production
// callers pass `&http.Client{...}` directly — no wrapper required.
var _ Doer = (*http.Client)(nil)
