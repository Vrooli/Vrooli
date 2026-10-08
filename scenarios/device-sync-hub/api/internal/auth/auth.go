// Package auth is the integration boundary over the scenario-authenticator
// scenario. Device Sync Hub does not own user identity, password storage, JWT
// signing, OAuth, or 2FA — all of that stays in scenario-authenticator. This
// package owns exactly two things:
//
//  1. Client — a Validator that verifies an owner's bearer token LOCALLY using
//     api-core/authn's provider-neutral RS256/JWKS verifier, and revokes a
//     device's authenticator session on un-pairing.
//  2. Middleware (middleware.go) — request-scoped extraction of the owner
//     Identity the devices/transfer handlers read to know "who is asking".
//
// Token verification is OFFLINE. The hub resolves the authenticator's API URL
// at runtime *by name* via api-core/discovery (no AUTH_SERVICE_URL env var, no
// hardcoded port), and api-core/authn fetches and caches the JWKS. The
// authenticator is contacted only to acquire signing keys, refresh them on key
// rotation, and revoke a session; it is NOT called per request.
//
// Failure mode is fail-closed: an invalid/expired token, an `alg` other than
// RS256 (this rejects "none" and HS* algorithm-confusion outright — the hub
// only ever verifies against an RSA public key), or an unobtainable signing key
// all yield no Identity, so owner-gated RPCs reject. Owner identity and device
// trust live in different layers on purpose: this package answers "who is the
// owner"; the devices domain answers "is this a trusted device".
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"

	"device-sync-hub/internal/httpc"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/sessions/sessions_v1connect"
)

// AuthScenarioSlug is the scenario whose API issues and signs owner JWTs. It is
// also the JWT issuer ("iss") the hub requires, so a token minted by a
// different service is rejected even if its signature somehow verified.
const AuthScenarioSlug = "scenario-authenticator"

// AuthExpectedAudience is the realm-qualified audience the hub requires on owner
// tokens. FROZEN cross-scenario contract with scenario-authenticator's default
// realm (realm.DefaultAudience). A token minted for a different realm/aud is
// rejected even if it is otherwise valid — this is the tenant-isolation check.
const AuthExpectedAudience = "scenario-authenticator:default"

// Identity is the owner identity resolved from a validated authenticator token.
// It is the request-scoped answer to "who is the owner making this call". The
// device layer keys every device row to OwnerID.
type Identity struct {
	OwnerID string
	Email   string
	Roles   []string
	Scopes  []string
	// ExpiresAt is the token's own expiry (the JWT "exp" claim), surfaced so a
	// brief validation cache never re-admits a token past its own expiry.
	ExpiresAt time.Time
}

// Validator is the seam the middleware depends on. Production wires *Client;
// handler/middleware tests substitute a fake so they never touch the network.
type Validator interface {
	// Validate resolves the bearer token to an owner Identity. It returns
	// ErrUnauthenticated for an absent/invalid/expired token (or a non-RS256
	// alg) and ErrAuthUnavailable when the signing key could not be obtained
	// (the fail-closed-but-distinguishable case).
	Validate(ctx context.Context, bearerToken string) (Identity, error)

	// RevokeSession asks the authenticator to drop a specific session by id
	// (device revocation). A blank session id is a no-op that returns nil.
	RevokeSession(ctx context.Context, sessionID string) error
}

// ErrUnauthenticated means the token was absent, malformed, expired, not signed
// by the authenticator, or used an unsupported algorithm. Handlers translate it
// to Connect CodeUnauthenticated / HTTP 401.
var ErrUnauthenticated = errors.New("unauthenticated")

// ErrAuthUnavailable means the authenticator's signing key could not be
// obtained (it could not be reached or returned no usable key). Distinct from
// ErrUnauthenticated so "couldn't check right now" is never confused with
// "definitely not allowed".
var ErrAuthUnavailable = errors.New("authenticator unavailable")

// URLResolver resolves a scenario's API base URL by slug. *discovery.Resolver
// satisfies it; tests substitute a static resolver (httptest) or a failing one.
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// Client is the production Validator: it verifies owner JWTs locally against
// scenario-authenticator's published RS256 key, fetched lazily and cached. Safe
// to share across requests.
type Client struct {
	resolver       URLResolver
	doer           httpc.Doer
	authScenario   string
	normalVerifier *authn.JWTVerifier
}

// Config configures the production Client.
type Config struct {
	// Resolver resolves the authenticator's API base URL by scenario slug. In
	// production this is a *discovery.Resolver; nil makes the client fail closed
	// (it can never obtain the signing key).
	Resolver URLResolver
	// Doer is the outbound HTTP seam (an *http.Client with a Timeout in prod).
	Doer httpc.Doer
	// AuthScenario overrides the authenticator scenario slug (defaults to
	// AuthScenarioSlug).
	AuthScenario string
	// Audience is the resource-specific audience accepted for owner tokens.
	// The compatibility default is used only when lifecycle configuration is
	// absent, so a manifest can isolate this resource from sibling consumers.
	Audience string
	// Now overrides the clock (tests). Defaults to time.Now.
	Now func() time.Time
	// MinRefetch is the minimum interval between JWKS refetches triggered by a
	// signature miss (rotation guard against hammering). Defaults to 5s.
	MinRefetch time.Duration
}

// NewClient constructs the production Validator, filling defaults for any
// optional dependency left nil. A nil Doer defaults to an http.Client with a
// conservative timeout so a hung authenticator can never pin a request open
// (fail-closed includes failing *fast*).
func NewClient(cfg Config) *Client {
	doer := cfg.Doer
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	scenario := strings.TrimSpace(cfg.AuthScenario)
	if scenario == "" {
		scenario = AuthScenarioSlug
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	audience := strings.TrimSpace(cfg.Audience)
	if audience == "" {
		audience = strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_AUDIENCE"))
	}
	if audience == "" {
		audience = AuthExpectedAudience
	}
	client := &Client{
		resolver:     cfg.Resolver,
		doer:         doer,
		authScenario: scenario,
	}
	client.normalVerifier = authn.NewJWTVerifier(authn.JWTConfig{
		Source:     identity.SourceScenarioAuthenticator,
		Issuer:     scenario,
		Audience:   audience,
		Kind:       identity.ActorHuman,
		Doer:       doer,
		Now:        now,
		StaleGrace: 24 * time.Hour,
		ResolveJWKS: func(ctx context.Context) (string, error) {
			base, err := client.baseURL(ctx)
			if err != nil {
				return "", err
			}
			return base + "/.well-known/jwks.json", nil
		},
	})
	return client
}

// Compile-time guarantee.
var _ Validator = (*Client)(nil)

func (c *Client) Validate(ctx context.Context, bearerToken string) (Identity, error) {
	if c == nil || c.normalVerifier == nil || strings.TrimSpace(bearerToken) == "" {
		return Identity{}, ErrUnauthenticated
	}

	principal, err := c.normalVerifier.Verify(ctx, bearerToken)
	if err != nil {
		failure, ok := identity.FailureFromError(err)
		if ok && failure.Class == identity.FailureUnavailable {
			return Identity{}, ErrAuthUnavailable
		}
		return Identity{}, ErrUnauthenticated
	}
	return Identity{OwnerID: principal.Subject, Email: principal.Email, Roles: principal.Roles, Scopes: nonNilStrings(principal.Scopes), ExpiresAt: principal.ExpiresAt}, nil
}

func (c *Client) RevokeSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	base, err := c.baseURL(ctx)
	if err != nil {
		return err
	}
	// Migrated from the retired REST DELETE /api/v1/sessions/{id} to the typed
	// Connect SessionsService.RevokeSession. The httpc.Doer satisfies
	// connect.HTTPClient directly. RevokeSession is idempotent server-side; a
	// NotFound is treated as already-gone (preserves the old 404→success path).
	client := sessionsconnect.NewSessionsServiceClient(c.doer, base)
	if _, err := client.RevokeSession(ctx, connect.NewRequest(&sessionsv1.RevokeSessionRequest{SessionId: sessionID})); err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	return nil
}

func (c *Client) baseURL(ctx context.Context) (string, error) {
	if c.resolver == nil {
		return "", fmt.Errorf("%w: no authenticator resolver configured", ErrAuthUnavailable)
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, c.authScenario)
	if err != nil {
		return "", fmt.Errorf("%w: resolve %s url: %v", ErrAuthUnavailable, c.authScenario, err)
	}
	return strings.TrimRight(base, "/"), nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}
