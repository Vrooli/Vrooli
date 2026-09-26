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

// Doer is the canonical outbound HTTP interface. The single-method
// surface is intentional — handlers depend on what they need (Do)
// rather than the full *http.Client surface (Get / Post / CloseIdle…),
// which keeps the FakeDoer surface small and the seam easy to substitute.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// NewDefault returns the canonical outbound client with a sane request
// Timeout. The router's provider fan-out (Phase 4) constructs its client
// through this so every outbound call is bounded; tests substitute
// testutil/mocks.FakeDoer.
func NewDefault() Doer {
	return &http.Client{Timeout: 30 * time.Second}
}

// Compile-time guarantee that the standard-library client satisfies Doer.
// Production callers pass a configured client (one with a Timeout set, e.g.
// from NewDefault) directly — no wrapper required.
var _ Doer = (*http.Client)(nil)
