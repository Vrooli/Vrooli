package routes

import (
	"context"
	"regexp"
	"strings"
)

// dnsLabel matches a single valid DNS label (RFC 1123): 1-63 chars,
// lowercase alphanumerics and hyphens, not starting or ending with a
// hyphen. Subdomains must be valid labels or Cloudflare ingress rejects
// them.
var dnsLabel = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// Service is the application-layer surface the routes handlers depend on.
// Owns validation, default substitution, and cross-handler policy. The
// handler is intentionally thin around it: decode → call service →
// translate errors.
type Service interface {
	// Create validates in and persists a route. Subdomain (valid DNS
	// label), scenario, and local_port (1-65535) are required; domain
	// defaults to DefaultDomain, health_path to DefaultHealthPath, tier
	// to TierLeased, enabled to true. Returns ErrInvalidRoute on
	// validation failure or ErrRouteConflict when the subdomain is taken.
	Create(ctx context.Context, in CreateInput) (Route, error)

	// Get is a thin pass-through; ErrRouteNotFound propagates verbatim.
	Get(ctx context.Context, id string) (Route, error)

	// List returns routes ordered by subdomain, optionally filtered by tier.
	List(ctx context.Context, tier Tier) ([]Route, error)

	// Update applies a partial update to the route with the given ID.
	Update(ctx context.Context, id string, in UpdateInput) (Route, error)

	// Delete removes a route by ID; returns false when the ID did not exist.
	Delete(ctx context.Context, id string) (bool, error)
}

type service struct {
	repo Repository
}

// NewService constructs the production Service.
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Create(ctx context.Context, in CreateInput) (Route, error) {
	subdomain := strings.TrimSpace(in.Subdomain)
	if subdomain == "" {
		return Route{}, ErrInvalidRoute{Field: "subdomain", Reason: "required"}
	}
	if !dnsLabel.MatchString(subdomain) {
		return Route{}, ErrInvalidRoute{Field: "subdomain", Reason: "must be a valid DNS label (lowercase alphanumerics and hyphens, 1-63 chars)"}
	}
	scenario := strings.TrimSpace(in.Scenario)
	if scenario == "" {
		return Route{}, ErrInvalidRoute{Field: "scenario", Reason: "required"}
	}
	if in.LocalPort < 1 || in.LocalPort > 65535 {
		return Route{}, ErrInvalidRoute{Field: "local_port", Reason: "must be between 1 and 65535"}
	}

	r := Route{
		Subdomain:  subdomain,
		Scenario:   scenario,
		Domain:     orDefault(in.Domain, DefaultDomain),
		LocalPort:  in.LocalPort,
		Tier:       in.Tier,
		LeaseID:    strings.TrimSpace(in.LeaseID),
		HealthPath: orDefault(in.HealthPath, DefaultHealthPath),
		Enabled:    true,
	}
	if r.Tier == "" {
		r.Tier = TierLeased
	}
	if in.Enabled != nil {
		r.Enabled = *in.Enabled
	}
	return s.repo.Create(ctx, r)
}

func (s *service) Get(ctx context.Context, id string) (Route, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) List(ctx context.Context, tier Tier) ([]Route, error) {
	return s.repo.List(ctx, tier)
}

func (s *service) Update(ctx context.Context, id string, in UpdateInput) (Route, error) {
	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return Route{}, err
	}

	if v := strings.TrimSpace(in.Subdomain); v != "" {
		if !dnsLabel.MatchString(v) {
			return Route{}, ErrInvalidRoute{Field: "subdomain", Reason: "must be a valid DNS label"}
		}
		existing.Subdomain = v
	}
	if v := strings.TrimSpace(in.Scenario); v != "" {
		existing.Scenario = v
	}
	if v := strings.TrimSpace(in.Domain); v != "" {
		existing.Domain = v
	}
	if in.LocalPort != 0 {
		if in.LocalPort < 1 || in.LocalPort > 65535 {
			return Route{}, ErrInvalidRoute{Field: "local_port", Reason: "must be between 1 and 65535"}
		}
		existing.LocalPort = in.LocalPort
	}
	if in.Tier != "" {
		existing.Tier = in.Tier
	}
	if v := strings.TrimSpace(in.HealthPath); v != "" {
		existing.HealthPath = v
	}
	if in.Enabled != nil {
		existing.Enabled = *in.Enabled
	}
	return s.repo.Update(ctx, existing)
}

func (s *service) Delete(ctx context.Context, id string) (bool, error) {
	return s.repo.Delete(ctx, id)
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
