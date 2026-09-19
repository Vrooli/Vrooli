// Package probes is the domain-scoped home for liveness probing and
// failure classification. It probes each exposed route's local port
// (internal) and public URL (external), records probe history, and
// diagnoses WHERE a failure is so recovery can act precisely.
//
// Layering mirrors the canonical Vrooli pattern (see internal/routes for
// the worked sibling reference):
//
//	HTTP → handler → Service (lists routes, probes, persists) → Repository
//	                     ↑                                          ↑
//	                     FakeService (handler tests)                FakeRepository (service tests)
//	                                                                Real sqlite (repository tests)
//
// types.go owns the domain entities and the string-typed enums the
// handlers translate to/from proto at the transport edge. The proto wire
// types live one floor up (packages/proto/...) and never import this
// package; the handler is the only translation point (api-steer §7).
package probes

import "time"

// ProbeKind distinguishes an internal probe (the route's local port) from
// an external probe (its public tunnel URL). String-typed like
// routes.Tier so storage and the domain agree on one representation.
type ProbeKind string

const (
	ProbeKindInternal ProbeKind = "internal"
	ProbeKindExternal ProbeKind = "external"
)

// ProbeStatus is the outcome of a single probe.
type ProbeStatus string

const (
	ProbeStatusUp      ProbeStatus = "up"
	ProbeStatusDown    ProbeStatus = "down"
	ProbeStatusTimeout ProbeStatus = "timeout"
	ProbeStatusError   ProbeStatus = "error"
)

// FailureClass is the diagnosis of why a route is (un)reachable, derived
// from the combination of its latest internal and external probes.
type FailureClass string

const (
	FailureClassHealthy          FailureClass = "healthy"
	FailureClassTunnelDown       FailureClass = "tunnel_down"
	FailureClassScenarioDown     FailureClass = "scenario_down"
	FailureClassCloudflareOutage FailureClass = "cloudflare_outage"
	FailureClassDNSFailure       FailureClass = "dns_failure"
	FailureClassConfigDrift      FailureClass = "config_drift"
)

// ProbeResult is one recorded probe outcome. Distinct from the proto wire
// type at packages/proto/gen/go/.../v1/probes.ProbeResult — handlers
// translate at the boundary so the domain never imports proto.
type ProbeResult struct {
	ID         string
	Subdomain  string
	Kind       ProbeKind
	Status     ProbeStatus
	LatencyMS  int
	StatusCode int
	ErrorMsg   string
	CreatedAt  time.Time
}

// RouteClassification is the diagnosed reachability of one route, combining
// its latest internal and external probe statuses.
type RouteClassification struct {
	Subdomain      string
	Classification FailureClass
	Internal       ProbeStatus
	External       ProbeStatus
	Assessment     string
}
