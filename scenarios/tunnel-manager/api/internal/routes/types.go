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

	"tunnel-manager/internal/manifest"
)

// Tier classifies why a route exists. CORE routes come from the
// api-core/coreset closure and are always exposed; LEASED routes back an
// on-demand lease and auto-expire (owned by the exposure domain).
type Tier = manifest.Tier

const (
	TierCore   = manifest.TierCore
	TierLeased = manifest.TierLeased
)

// RouteSource distinguishes scenario-backed routes from external ones. Aliased
// from the manifest type so the routes domain owns its validation while other
// domains share one shape.
type RouteSource = manifest.RouteSource

const (
	SourceScenario = manifest.SourceScenario
	SourceExternal = manifest.SourceExternal
)

// PublicExposure is the per-route override for the /public Access-bypass
// convention. Aliased from the manifest type so the routes domain owns its
// validation while other domains share one shape.
type PublicExposure = manifest.PublicExposure

const (
	PublicExposureInherit  = manifest.PublicExposureInherit
	PublicExposureEnabled  = manifest.PublicExposureEnabled
	PublicExposureDisabled = manifest.PublicExposureDisabled
)

// NormalizePublicExposure maps an empty or unrecognized value to Inherit.
var NormalizePublicExposure = manifest.NormalizePublicExposure

// DefaultDomain is the apex domain a subdomain hangs off when a route is
// created without one. It is a field, never a hardcoded constant baked
// into URLs — the old scenario hardcoded ".vrooli.com"; the live tunnel
// is ".itsagitime.com".
const DefaultDomain = manifest.DefaultDomain

// DefaultHealthPath is the per-route liveness path used when unset.
const DefaultHealthPath = manifest.DefaultHealthPath

// Route is the internal domain shape for one manifest entry. Distinct
// from the proto wire type at packages/proto/gen/go/.../v1/routes.Route —
// handlers translate at the boundary so the domain never imports proto.
type Route = manifest.Route

// PublicURL derives the route's reachable URL from its subdomain and
// domain. Derived (never stored) so the apex domain can change without a
// migration.
// CreateInput is the explicit input DTO Service.Create accepts. Distinct
// from Route so callers cannot pass an ID or timestamp the service has no
// way to honour — those belong to the persistence layer.
type CreateInput = manifest.CreateInput

// UpdateInput is a partial update; only non-zero/non-nil fields change.
type UpdateInput = manifest.UpdateInput

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
