// Package httpc declares the outbound HTTP seam every scenario uses
// when calling external services.
//
// Production wires the timeout-backed client from NewDefaultClient. Tests
// substitute mocks.FakeDoer to pin request shape and stub responses without
// touching the network.
//
// # Why ship the seam unwired in production
//
// The current backup manager does not need outbound HTTP in production.
// Defining the seam *before* the first
// outbound call means the first scenario to need one — a webhook
// dispatcher, an upstream-API client, an OAuth handshake — copies the
// reference test's substitution pattern instead of inventing one. The
// cost of an unused interface is zero; the cost of a parallel
// reinvention is hours.
//
// Add the field to server.Deps when wiring the first consumer; the
// idiomatic shape is `Doer httpc.Doer` with `httpc.NewDefaultClient()`
// constructed in main.go.
package httpc

import (
	"net/http"
	"time"
)

// DefaultTimeout is the outbound request timeout used by production callers
// unless a domain has a stricter integration-specific budget.
const DefaultTimeout = 10 * time.Second

// Doer is the canonical outbound HTTP interface. The single-method
// surface is intentional — handlers depend on what they need (Do)
// rather than the full *http.Client surface (Get / Post / CloseIdle…),
// which keeps the FakeDoer surface small and the seam easy to substitute.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// NewDefaultClient returns the production Doer for outbound HTTP calls. Keep
// all production construction through this helper so a no-timeout client cannot
// slip past review or standards scanning.
func NewDefaultClient() Doer {
	return &http.Client{Timeout: DefaultTimeout}
}
