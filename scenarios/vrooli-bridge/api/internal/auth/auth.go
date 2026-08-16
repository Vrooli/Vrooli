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
	"crypto"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"vrooli-bridge/internal/httpc"

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
)

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
	resolver     URLResolver
	doer         httpc.Doer
	authScenario string

	mu        sync.Mutex
	keys      []*rsa.PublicKey
	fetchedAt time.Time

	now                func() time.Time
	minRefetch         time.Duration
	maxJWKSGrace       time.Duration
	breakGlassPublic   ed25519.PublicKey
	breakGlassAudience string
	breakGlassTarget   string
}

// Config configures the production Client.
type Config struct {
	Resolver            URLResolver
	Doer                httpc.Doer
	AuthScenario        string
	Now                 func() time.Time
	MinRefetch          time.Duration
	JWKSGrace           time.Duration
	BreakGlassPublicKey []byte
	BreakGlassAudience  string
	BreakGlassTarget    string
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
	refetch := cfg.MinRefetch
	if refetch <= 0 {
		refetch = 5 * time.Second
	}
	grace := cfg.JWKSGrace
	if grace <= 0 {
		grace = 24 * time.Hour
	}
	audience := strings.TrimSpace(cfg.BreakGlassAudience)
	if audience == "" {
		audience = AuthExpectedAudience
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
	return &Client{
		resolver:           cfg.Resolver,
		doer:               doer,
		authScenario:       scenario,
		now:                now,
		minRefetch:         refetch,
		maxJWKSGrace:       grace,
		breakGlassPublic:   public,
		breakGlassAudience: audience,
		breakGlassTarget:   target,
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
		return Identity{}, err
	}

	keys, err := c.ensureKeys(ctx, false)
	if err != nil {
		return Identity{}, err
	}
	if verifyAny(keys, signingInput, sig) {
		return claims.toIdentity(c.now())
	}

	// The signature matched no cached key — the authenticator may have rotated.
	// Refetch once (rate-limited) and retry before declaring the token invalid.
	keys, err = c.ensureKeys(ctx, true)
	if err != nil {
		return Identity{}, err
	}
	if verifyAny(keys, signingInput, sig) {
		return claims.toIdentity(c.now())
	}
	return Identity{}, ErrUnauthenticated
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

func (c *Client) ensureKeys(ctx context.Context, forceRefetch bool) ([]*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.keys) > 0 {
		if !forceRefetch {
			if c.now().Sub(c.fetchedAt) <= c.maxJWKSGrace {
				return c.keys, nil
			}
		}
		if c.now().Sub(c.fetchedAt) < c.minRefetch {
			return c.keys, nil
		}
	}

	keys, err := c.fetchJWKS(ctx)
	if err != nil {
		if len(c.keys) > 0 && c.now().Sub(c.fetchedAt) <= c.maxJWKSGrace {
			return c.keys, nil
		}
		return nil, err
	}
	c.keys = keys
	c.fetchedAt = c.now()
	return c.keys, nil
}

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

// ownerClaims is the subset of scenario-authenticator's JWT claims consumed.
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
// string OR an array of strings.
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

func (c ownerClaims) toIdentity(now time.Time) (Identity, error) {
	if c.UserID == "" {
		return Identity{}, ErrUnauthenticated
	}
	if c.Iss != "" && c.Iss != AuthScenarioSlug {
		return Identity{}, ErrUnauthenticated
	}
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
	return Identity{OwnerID: c.UserID, Email: c.Email, Roles: c.Roles, Scopes: nonNilStrings(c.Scopes), ExpiresAt: exp, AuthMethod: AuthMethodNormal}, nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return append([]string(nil), values...)
}

// parseRS256 splits a compact JWS, rejects any algorithm other than RS256, and
// returns the signing input, raw signature, and parsed (still-unverified)
// claims. Claims must not be trusted until verifyAny passes.
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

// verifyAny reports whether the signature verifies against any key using
// RSASSA-PKCS1-v1_5 over SHA-256 (RS256).
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

// jwkToRSA reconstructs an RSA public key from a JWK's base64url modulus and
// exponent.
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
