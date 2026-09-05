// Package httpc declares the outbound HTTP seam every scenario uses
// when calling external services.
//
// Production wires *http.Client directly — it satisfies Doer through
// the compile-time assertion below. Tests substitute mocks.FakeDoer to
// pin request shape and stub responses without touching the network.
//
// # Why the seam can ship ahead of a consumer
//
// Defining the seam *before* a domain wires its own substitutable
// transport means the first one to need it — a webhook
// dispatcher, an upstream-API client, an OAuth handshake — copies the
// reference test's substitution pattern instead of inventing one. The
// cost of an unused interface is zero; the cost of a parallel
// reinvention is hours.
//
// Add the field to server.Deps when wiring the first consumer; the
// idiomatic shape is `Doer httpc.Doer` backed by an `http.Client`
// with a `Timeout` (e.g. 10 * time.Second) constructed in main.go.
package httpc

import "net/http"

// Doer is the canonical outbound HTTP interface. The single-method
// surface is intentional — handlers depend on what they need (Do)
// rather than the full *http.Client surface (Get / Post / CloseIdle…),
// which keeps the FakeDoer surface small and the seam easy to substitute.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Compile-time guarantee that *http.Client satisfies Doer. Production
// callers pass a pointer to a plain `http.Client` (with its `Timeout`
// set) directly — no wrapper required.
var _ Doer = (*http.Client)(nil)
