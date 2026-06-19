// Package routes is the domain-scoped home for the exposure manifest —
// the single source of truth for which scenario is exposed at which
// subdomain/port, in which tier, under which lease.
//
// Layering mirrors the canonical Vrooli pattern (see internal/notes for
// the worked template reference):
//
//	HTTP → handler → Service (validates, applies defaults) → Repository (persists)
//	                     ↑                                       ↑
//	                     FakeService (handler tests)             FakeRepository (service tests)
//	                                                             Real sqlite (repository tests)
//
// types.go owns the domain entity and the typed sentinels handlers
// translate at the transport edge. The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is the
// only translation point (api-steer §7).
package routes

import (
	"fmt"
	"time"
)

// Tier classifies why a route exists. CORE routes come from the
// api-core/coreset closure and are always exposed; LEASED routes back an
// on-demand lease and auto-expire (owned by the exposure domain).
type Tier string

const (
	TierCore   Tier = "core"
	TierLeased Tier = "leased"
)

// DefaultDomain is the apex domain a subdomain hangs off when a route is
// created without one. It is a field, never a hardcoded constant baked
// into URLs — the old scenario hardcoded ".vrooli.com"; the live tunnel
// is ".itsagitime.com".
const DefaultDomain = "itsagitime.com"

// DefaultHealthPath is the per-route liveness path used when unset.
const DefaultHealthPath = "/health"

// Route is the internal domain shape for one manifest entry. Distinct
// from the proto wire type at packages/proto/gen/go/.../v1/routes.Route —
// handlers translate at the boundary so the domain never imports proto.
type Route struct {
	ID         string
	Subdomain  string
	Scenario   string
	Domain     string
	LocalPort  int
	Tier       Tier
	LeaseID    string
	Enabled    bool
	HealthPath string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// PublicURL derives the route's reachable URL from its subdomain and
// domain. Derived (never stored) so the apex domain can change without a
// migration.
func (r Route) PublicURL() string {
	return fmt.Sprintf("https://%s.%s", r.Subdomain, r.Domain)
}

// CreateInput is the explicit input DTO Service.Create accepts. Distinct
// from Route so callers cannot pass an ID or timestamp the service has no
// way to honour — those belong to the persistence layer.
type CreateInput struct {
	Subdomain  string
	Scenario   string
	Domain     string
	LocalPort  int
	Tier       Tier
	LeaseID    string
	HealthPath string
	// Enabled is tri-state: nil defaults to true on create.
	Enabled *bool
}

// UpdateInput is a partial update; only non-zero/non-nil fields change.
type UpdateInput struct {
	Subdomain  string
	Scenario   string
	Domain     string
	LocalPort  int
	Tier       Tier
	HealthPath string
	Enabled    *bool
}

// ErrRouteNotFound is the typed sentinel returned when no row matches.
// Handlers translate via errors.As into a 404 (connect.CodeNotFound).
type ErrRouteNotFound struct {
	ID string
}

func (e ErrRouteNotFound) Error() string {
	return fmt.Sprintf("route %q not found", e.ID)
}

// ErrInvalidRoute is the typed sentinel returned when validation fails.
// Handlers translate via errors.As into a 400 (connect.CodeInvalidArgument).
type ErrInvalidRoute struct {
	Field  string
	Reason string
}

func (e ErrInvalidRoute) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ErrRouteConflict is the typed sentinel returned when a route would
// violate the one-route-per-subdomain invariant. Handlers translate via
// errors.As into a 409 (connect.CodeAlreadyExists).
type ErrRouteConflict struct {
	Subdomain string
}

func (e ErrRouteConflict) Error() string {
	return fmt.Sprintf("a route already exists for subdomain %q", e.Subdomain)
}
