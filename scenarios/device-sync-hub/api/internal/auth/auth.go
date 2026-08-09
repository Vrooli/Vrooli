// Package auth is the integration boundary over the scenario-authenticator
// scenario. Device Sync Hub does not own user identity, password storage, JWT
// signing, OAuth, or 2FA — all of that stays in scenario-authenticator. This
// package owns exactly two things:
//
//  1. Client — a Validator that verifies an owner's bearer token LOCALLY using
//     scenario-authenticator's RS256 public key (published as a JWKS), and
//     revokes a device's authenticator session on un-pairing.
//  2. Middleware (middleware.go) — request-scoped extraction of the owner
//     Identity the devices/transfer handlers read to know "who is asking".
//
// Token verification is OFFLINE. The hub resolves the authenticator's API URL
// at runtime *by name* via api-core/discovery (no AUTH_SERVICE_URL env var, no
// hardcoded port), fetches the authenticator's public key once (JWKS), caches
// it, and verifies every owner JWT's signature + expiry locally. The
// authenticator is contacted only to (a) fetch the signing key the first time
// (and again if a signature stops matching — key rotation) and (b) revoke a
// session. It is NOT called per request, so a momentarily-unavailable
// authenticator never breaks an already-issued owner session.
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
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

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
	resolver     URLResolver
	doer         httpc.Doer
	authScenario string

	mu        sync.Mutex
	keys      []*rsa.PublicKey
	fetchedAt time.Time

	now        func() time.Time
	minRefetch time.Duration
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
	refetch := cfg.MinRefetch
	if refetch <= 0 {
		refetch = 5 * time.Second
	}
	return &Client{
		resolver:     cfg.Resolver,
		doer:         doer,
		authScenario: scenario,
		now:          now,
		minRefetch:   refetch,
	}
}

// Compile-time guarantee.
var _ Validator = (*Client)(nil)

func (c *Client) Validate(ctx context.Context, bearerToken string) (Identity, error) {
	token := strings.TrimSpace(bearerToken)
	if token == "" {
		return Identity{}, ErrUnauthenticated
	}

	signingInput, sig, claims, err := parseRS256(token)
	if err != nil {
		return Identity{}, err // malformed or non-RS256 → ErrUnauthenticated
	}

	keys, err := c.ensureKeys(ctx, false)
	if err != nil {
		return Identity{}, err // could not obtain the signing key → ErrAuthUnavailable
	}
	if verifyAny(keys, signingInput, sig) {
		return claims.toIdentity(c.now())
	}

	// The signature matched no cached key — the authenticator may have rotated
	// its key. Refetch once (rate-limited) and retry before declaring the token
	// invalid, so rotation does not spuriously reject live sessions.
	keys, err = c.ensureKeys(ctx, true)
	if err != nil {
		return Identity{}, err
	}
	if verifyAny(keys, signingInput, sig) {
		return claims.toIdentity(c.now())
	}
	return Identity{}, ErrUnauthenticated
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

// ensureKeys returns the cached JWKS keys, fetching them if absent. When
// forceRefetch is set (a signature miss), it refetches unless it refetched
// within minRefetch — in which case it returns the current set so a forged
// token is rejected without hammering the authenticator.
func (c *Client) ensureKeys(ctx context.Context, forceRefetch bool) ([]*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.keys) > 0 {
		if !forceRefetch {
			return c.keys, nil
		}
		if c.now().Sub(c.fetchedAt) < c.minRefetch {
			return c.keys, nil
		}
	}

	keys, err := c.fetchJWKS(ctx)
	if err != nil {
		if len(c.keys) > 0 {
			// A refresh failed but we still hold a key set; serve it (the retry
			// verify will decide the token's fate).
			return c.keys, nil
		}
		return nil, err
	}
	c.keys = keys
	c.fetchedAt = c.now()
	return c.keys, nil
}

// fetchJWKS resolves the authenticator URL and downloads its JWKS, returning the
// usable RSA verification keys.
func (c *Client) fetchJWKS(ctx context.Context) ([]*rsa.PublicKey, error) {
	base, err := c.baseURL(ctx)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/.well-known/jwks.json", nil) //nolint:gosec // base is scenario-authenticator's URL resolved via api-core/discovery (trusted infra config from the lifecycle), not user input — not an SSRF sink
	if err != nil {
		return nil, fmt.Errorf("%w: build jwks request: %v", ErrAuthUnavailable, err)
	}
	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("%w: jwks returned %d", ErrAuthUnavailable, resp.StatusCode)
	}

	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&set); err != nil {
		return nil, fmt.Errorf("%w: decode jwks: %v", ErrAuthUnavailable, err)
	}

	var keys []*rsa.PublicKey
	for _, k := range set.Keys {
		if k.Kty != "RSA" || (k.Alg != "" && k.Alg != "RS256") {
			continue
		}
		pk, err := jwkToRSA(k.N, k.E)
		if err != nil {
			continue
		}
		keys = append(keys, pk)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: jwks contained no usable RSA key", ErrAuthUnavailable)
	}
	return keys, nil
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

// ownerClaims is the subset of scenario-authenticator's JWT claims the hub
// consumes (models.Claims + the standard registered claims).
type ownerClaims struct {
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles"`
	Scopes []string `json:"scope"`
	Exp    int64    `json:"exp"`
	Nbf    int64    `json:"nbf"`
	Iss    string   `json:"iss"`
	Aud    audience `json:"aud"`
}

// audience handles the JWT `aud` claim, which per RFC 7519 may be a single
// string OR an array of strings. Both shapes unmarshal into a []string.
type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		*a = audience{single}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err == nil {
		*a = audience(many)
		return nil
	}
	return fmt.Errorf("aud: not a string or array of strings")
}

func (a audience) contains(want string) bool {
	for _, v := range a {
		if v == want {
			return true
		}
	}
	return false
}

// toIdentity validates the time-based, issuer, and audience claims and projects
// them onto an Identity. Called only after the signature has verified.
func (c ownerClaims) toIdentity(now time.Time) (Identity, error) {
	if c.UserID == "" {
		return Identity{}, ErrUnauthenticated
	}
	if c.Iss != "" && c.Iss != AuthScenarioSlug {
		return Identity{}, ErrUnauthenticated
	}
	// Tenant isolation: the token must be aud-scoped to the realm the hub trusts.
	// A token minted for a different realm/aud is rejected even if otherwise valid.
	if !c.Aud.contains(AuthExpectedAudience) {
		return Identity{}, ErrUnauthenticated
	}
	if c.Exp > 0 && now.After(time.Unix(c.Exp, 0)) {
		return Identity{}, ErrUnauthenticated
	}
	if c.Nbf > 0 && now.Before(time.Unix(c.Nbf, 0)) {
		return Identity{}, ErrUnauthenticated
	}
	var exp time.Time
	if c.Exp > 0 {
		exp = time.Unix(c.Exp, 0).UTC()
	}
	return Identity{OwnerID: c.UserID, Email: c.Email, Roles: c.Roles, Scopes: nonNilStrings(c.Scopes), ExpiresAt: exp}, nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

// parseRS256 splits a compact JWS, rejects any algorithm other than RS256, and
// returns the signing input (header.payload), the raw signature, and the parsed
// (still-unverified) claims. Claims must not be trusted until verifyAny passes.
func parseRS256(token string) (signingInput string, sig []byte, claims ownerClaims, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}
	var hdr struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerRaw, &hdr); err != nil {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}
	// Lock to exactly RS256. This rejects "none" and any HS*/EC* algorithm,
	// eliminating algorithm-confusion attacks — we only ever verify against an
	// RSA public key.
	if hdr.Alg != "RS256" {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}

	sig, err = base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}
	claimsRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}
	if err := json.Unmarshal(claimsRaw, &claims); err != nil {
		return "", nil, ownerClaims{}, ErrUnauthenticated
	}
	return parts[0] + "." + parts[1], sig, claims, nil
}

// verifyAny reports whether the signature verifies against any of the keys
// using RSASSA-PKCS1-v1_5 over SHA-256 (RS256).
func verifyAny(keys []*rsa.PublicKey, signingInput string, sig []byte) bool {
	hashed := sha256.Sum256([]byte(signingInput))
	for _, k := range keys {
		if k == nil {
			continue
		}
		if rsa.VerifyPKCS1v15(k, crypto.SHA256, hashed[:], sig) == nil {
			return true
		}
	}
	return false
}

// jwkToRSA reconstructs an RSA public key from a JWK's base64url modulus (n) and
// exponent (e).
func jwkToRSA(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	if len(nBytes) == 0 || len(eBytes) == 0 {
		return nil, errors.New("empty modulus or exponent")
	}
	e := new(big.Int).SetBytes(eBytes)
	if !e.IsInt64() || e.Int64() <= 0 {
		return nil, errors.New("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: int(e.Int64())}, nil
}
