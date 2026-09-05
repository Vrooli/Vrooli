package manifest

import (
	"context"
	"fmt"
	"time"
)

// Tier classifies why a route exists. CORE routes come from the
// api-core/coreset closure and are always exposed; LEASED routes back an
// on-demand lease and auto-expire.
type Tier string

const (
	TierCore   Tier = "core"
	TierLeased Tier = "leased"
)

// RouteSource distinguishes scenario-backed routes (the local UI port of a
// known scenario) from external routes that point at an arbitrary
// service_target (e.g. http://127.0.0.1:9000). It is orthogonal to Tier:
// tiering is an exposure-policy axis, source is a provenance axis. Empty
// defaults to SourceScenario for backwards compatibility with rows written
// before the column existed.
type RouteSource string

const (
	SourceScenario RouteSource = "scenario"
	SourceExternal RouteSource = "external"
)

const (
	DefaultDomain     = "itsagitime.com"
	DefaultHealthPath = "/health"
)

// PublicExposure is a route's per-route override for the /public Access-bypass
// convention (see docs/concepts/PUBLIC_ASSETS.md). It is orthogonal to Tier
// and Source. Tri-state: Inherit defers to the global switch
// (config.public_exposure_enabled), Enabled forces the bypass on for this
// host, Disabled forces it off. Empty/legacy rows are treated as Inherit.
type PublicExposure string

const (
	PublicExposureInherit  PublicExposure = "inherit"
	PublicExposureEnabled  PublicExposure = "enabled"
	PublicExposureDisabled PublicExposure = "disabled"
)

// NormalizePublicExposure maps an empty or unrecognized value to Inherit (the
// safe default: defer to the global switch).
func NormalizePublicExposure(v PublicExposure) PublicExposure {
	switch v {
	case PublicExposureEnabled, PublicExposureDisabled:
		return v
	default:
		return PublicExposureInherit
	}
}

// Route is the shared manifest record read by monitoring, audit, exposure,
// and ingress reconciliation domains. The routes domain owns persistence and
// validation for this shape.
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
	// Source distinguishes scenario-backed routes from external ones. Empty
	// is treated as SourceScenario (the default and only kind before external
	// routes existed).
	Source RouteSource
	// ServiceTarget is the explicit local service URL an external route
	// forwards to (e.g. http://127.0.0.1:9000). Empty for scenario routes,
	// which derive http://localhost:<local_port>.
	ServiceTarget string
	// PublicExposure is the per-route override for the /public Access-bypass
	// convention. Empty is treated as Inherit (defer to the global switch).
	PublicExposure PublicExposure
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (r Route) PublicURL() string {
	return fmt.Sprintf("https://%s.%s", r.Subdomain, r.Domain)
}

type CreateInput struct {
	Subdomain      string
	Scenario       string
	Domain         string
	LocalPort      int
	Tier           Tier
	LeaseID        string
	HealthPath     string
	Enabled        *bool
	Source         RouteSource
	ServiceTarget  string
	PublicExposure PublicExposure
}

type UpdateInput struct {
	Subdomain      string
	Scenario       string
	Domain         string
	LocalPort      int
	Tier           Tier
	HealthPath     string
	Enabled        *bool
	Source         RouteSource
	ServiceTarget  string
	PublicExposure PublicExposure
}

type Reader interface {
	List(ctx context.Context, tier Tier) ([]Route, error)
}

type Store interface {
	Reader
	Create(ctx context.Context, in CreateInput) (Route, error)
	Update(ctx context.Context, id string, in UpdateInput) (Route, error)
	Delete(ctx context.Context, id string) (bool, error)
}
