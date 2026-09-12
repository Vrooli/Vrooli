// Package audit is the domain-scoped home for the port-compliance auditor —
// a computed/reporting domain that owns NO persistent table. It compares each
// manifested route's scenario service.json against the manifest's expected
// local_port and surfaces drift before it silently breaks ingress.
//
// Layering mirrors the canonical Vrooli pattern (see internal/routes for the
// worked reference), minus persistence:
//
//	HTTP → handler → Service (computes findings) → RoutesReader (reads manifest)
//	                     ↑                              ↑
//	                     FakeService (handler tests)    fakeRoutesReader (service tests)
//
// There is no Repository, sqlite.go, schema.sql, or schema.go here: the audit
// domain stores nothing. Its only inputs are the routes manifest (via the
// RoutesReader seam) and the scenarios filesystem tree (via an injected root).
//
// types.go owns the domain entity and its string-typed status enum. The proto
// wire types live one floor up (packages/proto/...) and never import this
// package; the handler is the only translation point (api-steer §7).
package audit

// AuditStatus classifies a single route's port-compliance finding. String-typed
// so the domain stays independent of the proto enum; the handler translates at
// the transport edge.
type AuditStatus string

const (
	// StatusCompliant means the scenario's UI port matches the manifest.
	StatusCompliant AuditStatus = "compliant"
	// StatusMismatch means service.json declares a different UI port than the
	// manifest's expected local_port.
	StatusMismatch AuditStatus = "mismatch"
	// StatusMissingScenario means no service.json could be read for the route's
	// scenario.
	StatusMissingScenario AuditStatus = "missing_scenario"
	// StatusMissingPort means service.json declares no fixed UI port (ranged or
	// dynamic).
	StatusMissingPort AuditStatus = "missing_port"
)

// PortAuditResult is one route's compliance finding. Distinct from the proto
// wire type at packages/proto/gen/go/.../v1/audit.PortAuditResult — handlers
// translate at the boundary so the domain never imports proto.
type PortAuditResult struct {
	Subdomain    string
	Scenario     string
	ExpectedPort int
	// ActualPort is the UI port found in service.json; 0 when none was found.
	ActualPort int
	Status     AuditStatus
	Detail     string
}
