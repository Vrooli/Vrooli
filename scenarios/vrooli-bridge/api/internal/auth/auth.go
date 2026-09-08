// Package auth is the integration boundary over the scenario-authenticator
// scenario. Vrooli Bridge does not own user identity, password storage, JWT
// signing, OAuth, or 2FA — all of that stays in scenario-authenticator. This
// package owns exactly one thing: Client, a Validator that verifies an owner's
// bearer token LOCALLY against scenario-authenticator's RS256 public key
// (published as a JWKS). It is the "Owner → control plane" authorization
// boundary in SECURITY.md (only the owner can register/revoke nodes, dispatch
// jobs, or provision).
//
// Token verification is OFFLINE. The control plane resolves the authenticator's
// API URL at runtime *by name* via api-core/discovery (no env var, no hardcoded
// port), fetches the authenticator's public key once (JWKS), caches it, and
// verifies every owner JWT's signature + expiry locally. The authenticator is
// contacted only to fetch the signing key the first time (and again on a
// signature miss — key rotation), never per request, so a momentarily
// unavailable authenticator never breaks an already-issued owner session.
//
// Failure mode is fail-closed: an invalid/expired token, an `alg` other than
// RS256 (this rejects "none" and HS* algorithm-confusion outright), or an
// unobtainable signing key all yield no Identity, so owner-gated RPCs reject.
//
// Trimmed from device-sync-hub's auth package: bridge has no authenticator
// *session* to revoke (node revocation destroys node credentials, the pairing
// domain's concern — not an owner auth session), so this package carries only
// the owner-token Validate path.
package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"vrooli-bridge/internal/httpc"
	localenrollment "vrooli-bridge/internal/operatorsession"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	sharedsession "github.com/vrooli/api-core/operatorsession"
	"github.com/vrooli/api-core/trustposture"
)

// AuthScenarioSlug is the scenario whose API issues and signs owner JWTs, and
// the JWT issuer ("iss") the control plane requires.
const AuthScenarioSlug = "scenario-authenticator"

// AuthExpectedAudience is the realm-qualified audience required on owner tokens.
// FROZEN cross-scenario contract with scenario-authenticator's default realm.
const AuthExpectedAudience = "scenario-authenticator:default"

// Identity is the owner identity resolved from a validated authenticator token.
type Identity struct {
	OwnerID    string
	Email      string
	Roles      []string
	Scopes     []string
	ExpiresAt  time.Time
	AuthMethod AuthMethod
}

// AuthMethod is a typed distinction for audit and policy consumers.
type AuthMethod string

const (
	AuthMethodNormal     AuthMethod = "normal"
	AuthMethodBreakGlass AuthMethod = "break_glass"
	AuthMethodEnrolled   AuthMethod = "enrolled"
)

// LocalSessionStore is the Bridge-owned enrollment authority. It returns
// public records only; private keys and locally minted credentials stay on the
// operator client.
type LocalSessionStore interface {
	Lookup(context.Context, string) (localenrollment.Record, error)
}

// Validator is the seam the middleware depends on. Production wires *Client;
// handler/middleware tests substitute a fake so they never touch the network.
type Validator interface {
	// Validate resolves the bearer token to an owner Identity. It returns
	// ErrUnauthenticated for an absent/invalid/expired token (or a non-RS256
	// alg) and ErrAuthUnavailable when the signing key could not be obtained.
	Validate(ctx context.Context, bearerToken string) (Identity, error)
}

// ErrUnauthenticated means the token was absent, malformed, expired, not signed
// by the authenticator, or used an unsupported algorithm.
var ErrUnauthenticated = errors.New("unauthenticated")

// ErrAuthUnavailable means the authenticator's signing key could not be
// obtained. Distinct from ErrUnauthenticated so "couldn't check right now" is
// never confused with "definitely not allowed".
var ErrAuthUnavailable = errors.New("authenticator unavailable")

// URLResolver resolves a scenario's API base URL by slug. *discovery.Resolver
// satisfies it; tests substitute a static resolver or a failing one.
type URLResolver interface {
	ResolveScenarioURLDefault(ctx context.Context, scenarioSlug string) (string, error)
}

// Client is the production Validator: it verifies owner JWTs locally against
// scenario-authenticator's published RS256 key, fetched lazily and cached.
type Client struct {
	resolver       URLResolver
	doer           httpc.Doer
	authScenario   string
	normalVerifier *authn.JWTVerifier

	now                func() time.Time
	breakGlassPublic   ed25519.PublicKey
	breakGlassAudience string
	breakGlassTarget   string
	localSessions      LocalSessionStore
}

// Config configures the production Client.
type Config struct {
	Resolver            URLResolver
	Doer                httpc.Doer
	AuthScenario        string
	Audience            string
	Now                 func() time.Time
	MinRefetch          time.Duration
	JWKSGrace           time.Duration
	BreakGlassPublicKey []byte
	BreakGlassAudience  string
	BreakGlassTarget    string
	LocalSessions       LocalSessionStore
}

// NewClient constructs the production Validator, filling defaults. A nil Doer
// defaults to an http.Client with a conservative timeout so a hung
// authenticator can never pin a request open (fail-closed includes failing
// fast). A nil Resolver makes the client fail closed (it can never obtain the
// signing key).
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
	grace := cfg.JWKSGrace
	if grace <= 0 {
		grace = 24 * time.Hour
	}
	audience := strings.TrimSpace(cfg.BreakGlassAudience)
	if audience == "" {
		audience = AuthExpectedAudience
	}
	normalAudience := strings.TrimSpace(cfg.Audience)
	if normalAudience == "" {
		normalAudience = strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_AUDIENCE"))
	}
	if normalAudience == "" {
		normalAudience = AuthExpectedAudience
	}
	target := strings.TrimSpace(cfg.BreakGlassTarget)
	if target == "" {
		target, _ = os.Hostname()
		target = strings.TrimSpace(target)
	}
	var public ed25519.PublicKey
	if len(cfg.BreakGlassPublicKey) == ed25519.PublicKeySize {
		public = append(ed25519.PublicKey(nil), cfg.BreakGlassPublicKey...)
	}
	client := &Client{
		resolver:           cfg.Resolver,
		doer:               doer,
		authScenario:       scenario,
		now:                now,
		breakGlassPublic:   public,
		breakGlassAudience: audience,
		breakGlassTarget:   target,
		localSessions:      cfg.LocalSessions,
	}
	client.normalVerifier = authn.NewJWTVerifier(authn.JWTConfig{
		Source:     identity.SourceScenarioAuthenticator,
		Issuer:     AuthScenarioSlug,
		Audience:   normalAudience,
		Kind:       identity.ActorHuman,
		Doer:       doer,
		Now:        now,
		CacheTTL:   grace,
		StaleGrace: grace,
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

// ValidateLocal verifies a session minted from a previously enrolled client
// key. It never contacts the authenticator; enrollment is the only online
// authority step. Revocation is checked on every request, so local sessions
// cannot outlive a revoked enrollment.
func (c *Client) ValidateLocal(ctx context.Context, token string) (Identity, error) {
	if c.localSessions == nil {
		return Identity{}, ErrUnauthenticated
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 || parts[0] != "OS1" {
		return Identity{}, ErrUnauthenticated
	}
	claimsPayload, err := decodeLocalReference(parts[1])
	if err != nil {
		return Identity{}, ErrUnauthenticated
	}
	record, err := c.localSessions.Lookup(ctx, claimsPayload.EnrollmentReference)
	if err != nil || record.Revoked || record.OperatorID != claimsPayload.OperatorID {
		return Identity{}, ErrUnauthenticated
	}
	claims, err := sharedsession.Verify(record.PublicKey, token, c.now())
	if err != nil || claims.EnrollmentReference != record.Reference || !sharedsession.ContainsAll(record.Scopes, claims.Scopes) {
		return Identity{}, ErrUnauthenticated
	}
	return Identity{OwnerID: claims.OperatorID, Scopes: append([]string(nil), claims.Scopes...), ExpiresAt: time.Unix(claims.ExpiresAt, 0).UTC(), AuthMethod: AuthMethodEnrolled}, nil
}

func decodeLocalReference(encoded string) (sharedsession.LocalSession, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return sharedsession.LocalSession{}, err
	}
	var claims sharedsession.LocalSession
	if err := json.Unmarshal(data, &claims); err != nil {
		return sharedsession.LocalSession{}, err
	}
	return claims, nil
}

// Compile-time guarantee.
var _ Validator = (*Client)(nil)

func (c *Client) Validate(ctx context.Context, bearerToken string) (Identity, error) {
	token := strings.TrimSpace(bearerToken)
	if token == "" || c == nil || c.normalVerifier == nil {
		return Identity{}, ErrUnauthenticated
	}
	principal, err := c.normalVerifier.Verify(ctx, token)
	if err != nil {
		if failure, ok := identity.FailureFromError(err); ok && failure.Class == identity.FailureUnavailable {
			return Identity{}, ErrAuthUnavailable
		}
		return Identity{}, ErrUnauthenticated
	}
	return Identity{OwnerID: principal.Subject, Email: principal.Email, Roles: append([]string(nil), principal.Roles...), Scopes: append([]string(nil), principal.Scopes...), ExpiresAt: principal.ExpiresAt, AuthMethod: AuthMethodNormal}, nil
}

// ValidateBreakGlass verifies the pre-provisioned credential entirely
// offline. It is a separate authorization scheme, never a fallback from a
// failed bearer-token verification.
func (c *Client) ValidateBreakGlass(_ context.Context, token string) (Identity, error) {
	if len(c.breakGlassPublic) != ed25519.PublicKeySize {
		return Identity{}, ErrUnauthenticated
	}
	if c.breakGlassTarget == "" {
		return Identity{}, ErrUnauthenticated
	}
	claims, err := trustposture.Verify(c.breakGlassPublic, strings.TrimSpace(token), c.breakGlassAudience, c.breakGlassTarget, c.now())
	if err != nil {
		return Identity{}, ErrUnauthenticated
	}
	return Identity{
		OwnerID:    claims.Subject,
		Scopes:     append([]string(nil), claims.Scopes...),
		ExpiresAt:  time.Unix(claims.ExpiresAt, 0).UTC(),
		AuthMethod: AuthMethodBreakGlass,
	}, nil
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
