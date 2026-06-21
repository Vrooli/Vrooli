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

const (
	DefaultDomain     = "itsagitime.com"
	DefaultHealthPath = "/health"
)

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
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (r Route) PublicURL() string {
	return fmt.Sprintf("https://%s.%s", r.Subdomain, r.Domain)
}

type CreateInput struct {
	Subdomain  string
	Scenario   string
	Domain     string
	LocalPort  int
	Tier       Tier
	LeaseID    string
	HealthPath string
	Enabled    *bool
}

type UpdateInput struct {
	Subdomain  string
	Scenario   string
	Domain     string
	LocalPort  int
	Tier       Tier
	HealthPath string
	Enabled    *bool
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
