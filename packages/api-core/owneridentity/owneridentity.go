// Package owneridentity verifies the owner credentials issued by
// scenario-authenticator. It is intentionally transport-independent so
// control-plane scenarios can resolve the same subject and explicit scope
// ceiling without copying authenticator key-handling code.
package owneridentity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

const (
	DefaultAuthScenario = "scenario-authenticator"
	ExpectedIssuer      = "scenario-authenticator"
	ExpectedAudience    = "scenario-authenticator:default"
)

var (
	ErrUnauthenticated = errors.New("owner identity is unauthenticated")
	ErrUnavailable     = errors.New("owner identity provider unavailable")
)

// Identity is the verified owner account used to seed an agent run.
type Identity struct {
	Subject   string
	Email     string
	Scopes    []string
	ExpiresAt time.Time
}

// URLResolver resolves a scenario API URL by its lifecycle-managed name.
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Validator is the narrow seam used by agent-manager. Implementations must
// reject invalid tokens and provider outages; neither condition grants a
// default identity.
type Validator interface {
	Validate(context.Context, string) (Identity, error)
}

type Config struct {
	Resolver     URLResolver
	HTTPClient   *http.Client
	AuthScenario string
	JWKSURL      string
	Now          func() time.Time
	JWKSGrace    time.Duration
}

type Client struct {
	verifier *authn.JWTVerifier
}

func NewClient(cfg Config) *Client {
	scenario := strings.TrimSpace(cfg.AuthScenario)
	if scenario == "" {
		scenario = DefaultAuthScenario
	}
	grace := cfg.JWKSGrace
	if grace <= 0 {
		grace = 24 * time.Hour
	}
	resolveJWKS := func(ctx context.Context) (string, error) {
		if strings.TrimSpace(cfg.JWKSURL) != "" {
			return strings.TrimSpace(cfg.JWKSURL), nil
		}
		if cfg.Resolver == nil {
			return "", fmt.Errorf("%w: no scenario resolver", ErrUnavailable)
		}
		base, err := cfg.Resolver.ResolveScenarioURLDefault(ctx, scenario)
		if err != nil {
			return "", fmt.Errorf("%w: resolve authenticator: %v", ErrUnavailable, err)
		}
		return strings.TrimRight(base, "/") + "/.well-known/jwks.json", nil
	}
	return &Client{verifier: authn.NewJWTVerifier(authn.JWTConfig{
		Source:      identity.SourceScenarioAuthenticator,
		Issuer:      ExpectedIssuer,
		Audience:    ExpectedAudience,
		Kind:        identity.ActorHuman,
		ResolveJWKS: resolveJWKS,
		Client:      cfg.HTTPClient,
		Now:         cfg.Now,
		CacheTTL:    grace,
		StaleGrace:  grace,
	})}
}

var _ Validator = (*Client)(nil)

func (c *Client) Validate(ctx context.Context, raw string) (Identity, error) {
	if c == nil || c.verifier == nil || strings.TrimSpace(raw) == "" {
		return Identity{}, ErrUnauthenticated
	}
	principal, err := c.verifier.Verify(ctx, raw)
	if err != nil {
		if failure, ok := identity.FailureFromError(err); ok && failure.Class == identity.FailureUnavailable {
			return Identity{}, ErrUnavailable
		}
		return Identity{}, ErrUnauthenticated
	}
	return Identity{Subject: principal.Subject, Email: principal.Email, Scopes: append([]string(nil), principal.Scopes...), ExpiresAt: principal.ExpiresAt}, nil
}

// Reachable performs a read-only health check against the authenticator's
// published JWKS. It deliberately bypasses the grace cache: callers use this
// to report current dependency reachability, while Validate may continue to
// use a recently fetched key set during a short provider outage.
func (c *Client) Reachable(ctx context.Context) error {
	if c == nil || c.verifier == nil {
		return fmt.Errorf("%w: nil client", ErrUnavailable)
	}
	if err := c.verifier.Reachable(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}
