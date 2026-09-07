package policygate

// This file is the relying-party adapter for scenario-authenticator. GCT does
// not issue credentials and never treats a caller header as a principal.

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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	accountsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts"
	accountsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-authenticator/v1/accounts/accounts_v1connect"
)

const (
	DefaultAuthenticatorIssuer   = "scenario-authenticator"
	DefaultAuthenticatorRealm    = "default"
	DefaultAuthenticatorAudience = "scenario-authenticator:default"
	DefaultJWKSCacheTTL          = 5 * time.Minute
	accessTokenCookieName        = "gct_access_token"
)

var (
	ErrMissingCredential        = errors.New("missing authenticator credential")
	ErrInvalidCredential        = errors.New("invalid authenticator credential")
	ErrRevokedCredential        = errors.New("revoked authenticator credential")
	ErrAuthenticatorUnavailable = errors.New("authenticator unavailable")
)

// PrincipalVerifier is intentionally narrow so the middleware can be tested
// with recording fakes and production can use the authenticator's JWKS.
type PrincipalVerifier interface {
	Verify(context.Context, string) (Principal, error)
}

type PrincipalVerifierFunc func(context.Context, string) (Principal, error)

func (f PrincipalVerifierFunc) Verify(ctx context.Context, token string) (Principal, error) {
	return f(ctx, token)
}

// SharedAuthenticatorProvider adapts GCT's existing scenario-authenticator
// verifier to api-core's provider-neutral contract. It keeps the Vrooli IdP
// supported during Cloudflare migration without moving its token semantics
// into shared infrastructure.
type SharedAuthenticatorProvider struct{ verifier PrincipalVerifier }

func NewSharedAuthenticatorProvider(verifier PrincipalVerifier) authn.Provider {
	return SharedAuthenticatorProvider{verifier: verifier}
}

func (p SharedAuthenticatorProvider) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}

func (p SharedAuthenticatorProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	raw := extractAccessToken(req)
	if raw == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
	}
	if p.verifier == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, p.Source())
	}
	principal, err := p.verifier.Verify(ctx, raw)
	if err != nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	return identity.Principal{
		Kind: identity.ActorHuman, Subject: principal.Subject, Email: principal.Email,
		Realm: principal.Realm, Roles: append([]string(nil), principal.Roles...),
		Scopes: append([]string(nil), principal.Scopes...), Verified: principal.Verified,
		Source: p.Source(), Sources: []identity.AuthSource{p.Source()}, Issuer: principal.Issuer,
	}, nil
}

// AuthenticatorConfig describes the RP-side contract. JWKSURL is optional;
// when absent, api-core discovery resolves scenario-authenticator by slug.
type AuthenticatorConfig struct {
	Issuer       string
	Realm        string
	Audience     string
	JWKSURL      string
	JWKSCacheTTL time.Duration
	Client       *http.Client
	Now          func() time.Time
	// Validate optionally performs a live authenticator session check after
	// local signature verification. It is required by deployments that need
	// immediate logout/password-change revocation rather than expiry-based
	// revocation.
	Validate func(context.Context, string) (Principal, error)
}

func DefaultAuthenticatorConfig() AuthenticatorConfig {
	issuer := strings.TrimSpace(os.Getenv("GCT_AUTH_ISSUER"))
	if issuer == "" {
		issuer = DefaultAuthenticatorIssuer
	}
	realm := strings.TrimSpace(os.Getenv("GCT_AUTH_REALM"))
	if realm == "" {
		realm = DefaultAuthenticatorRealm
	}
	audience := strings.TrimSpace(os.Getenv("GCT_AUTH_AUDIENCE"))
	if audience == "" {
		audience = DefaultAuthenticatorAudience
	}
	client := &http.Client{Timeout: 5 * time.Second}
	jwksTTL := DefaultJWKSCacheTTL
	if rawTTL := strings.TrimSpace(os.Getenv("GCT_AUTH_JWKS_TTL_SECONDS")); rawTTL != "" {
		if seconds, err := strconv.Atoi(rawTTL); err == nil && seconds > 0 {
			jwksTTL = time.Duration(seconds) * time.Second
		}
	}
	cfg := AuthenticatorConfig{
		Issuer:       issuer,
		Realm:        realm,
		Audience:     audience,
		JWKSURL:      strings.TrimSpace(os.Getenv("GCT_AUTH_JWKS_URL")),
		JWKSCacheTTL: jwksTTL,
		Client:       client,
		Now:          time.Now,
	}
	if validateURL := strings.TrimSpace(os.Getenv("GCT_AUTH_VALIDATE_URL")); validateURL != "" {
		cfg.Validate = RemoteTokenValidator{BaseURL: validateURL, Client: client}.Validate
	}
	return cfg
}

// RemoteTokenValidator calls the authenticator's typed Validate RPC. It is
// only needed where immediate session revocation is a requirement; the JWT
// signature, issuer, audience, and expiry are still checked locally first.
type RemoteTokenValidator struct {
	BaseURL string
	Client  *http.Client
}

func (v RemoteTokenValidator) Validate(ctx context.Context, token string) (Principal, error) {
	if strings.TrimSpace(v.BaseURL) == "" {
		return Principal{}, fmt.Errorf("%w: authenticator validate URL is empty", ErrAuthenticatorUnavailable)
	}
	client := v.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	accountsClient := accountsconnect.NewAccountsServiceClient(client, strings.TrimRight(v.BaseURL, "/"))
	response, err := accountsClient.Validate(ctx, connect.NewRequest(&accountsv1.ValidateRequest{AccessToken: token}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeUnauthenticated || connect.CodeOf(err) == connect.CodePermissionDenied {
			return Principal{}, fmt.Errorf("%w: authenticator rejected credential", ErrRevokedCredential)
		}
		return Principal{}, fmt.Errorf("%w: %v", ErrAuthenticatorUnavailable, err)
	}
	if response == nil || !response.Msg.GetValid() || strings.TrimSpace(response.Msg.GetUserId()) == "" {
		return Principal{}, fmt.Errorf("%w: authenticator rejected credential", ErrRevokedCredential)
	}
	return Principal{
		Kind: cliutil.CallerKindHuman, Subject: response.Msg.GetUserId(), Email: response.Msg.GetEmail(),
		Realm: response.Msg.GetRealm(), Roles: append([]string(nil), response.Msg.GetRoles()...),
		Scopes: append([]string(nil), response.Msg.GetScopes()...), Verified: true,
	}, nil
}

// JWTVerifier validates the authenticator's RS256 token locally against its
// published JWKS. It verifies issuer, audience, expiry, algorithm and key id.
// Revocation remains an authenticator-owned concern: the short-lived token is
// invalidated when it expires, while deployments that require immediate
// revocation can inject a verifier that performs the authenticator Validate RPC.
type JWTVerifier struct {
	cfg           AuthenticatorConfig
	mu            sync.RWMutex
	keys          map[string]*rsa.PublicKey
	keysFetchedAt time.Time
	fingerprint   string
}

func NewJWTVerifier(cfg AuthenticatorConfig) *JWTVerifier {
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 5 * time.Second}
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.JWKSCacheTTL <= 0 {
		cfg.JWKSCacheTTL = DefaultJWKSCacheTTL
	}
	return &JWTVerifier{cfg: cfg}
}

func (v *JWTVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	header, claims, signature, signingInput, err := parseJWT(raw)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: %v", ErrInvalidCredential, err)
	}
	if header.Alg != "RS256" {
		return Principal{}, fmt.Errorf("%w: unsupported signing algorithm", ErrInvalidCredential)
	}
	keys, err := v.publicKeys(ctx, false)
	if err != nil {
		return Principal{}, err
	}
	key := keys[header.Kid]
	if key == nil && header.Kid == "" && len(keys) == 1 {
		for _, candidate := range keys {
			key = candidate
		}
	}
	if key == nil {
		keys, err = v.publicKeys(ctx, true)
		if err != nil {
			return Principal{}, err
		}
		key = keys[header.Kid]
		if key == nil && header.Kid == "" && len(keys) == 1 {
			for _, candidate := range keys {
				key = candidate
			}
		}
		if key == nil {
			return Principal{}, fmt.Errorf("%w: signing key not found", ErrInvalidCredential)
		}
	}
	digest := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, cryptoHashSHA256, digest[:], signature); err != nil {
		return Principal{}, fmt.Errorf("%w: signature verification failed", ErrInvalidCredential)
	}

	now := v.cfg.Now().UTC()
	if claims.Issuer != v.cfg.Issuer || !contains(claims.Audience, v.cfg.Audience) || claims.ExpiresAt <= now.Unix() {
		return Principal{}, fmt.Errorf("%w: issuer, audience, or expiry mismatch", ErrInvalidCredential)
	}
	subject := strings.TrimSpace(claims.UserID)
	if subject == "" {
		subject = strings.TrimSpace(claims.Subject)
	}
	if subject == "" {
		return Principal{}, fmt.Errorf("%w: subject missing", ErrInvalidCredential)
	}
	principal := Principal{
		Kind:     cliutil.CallerKindHuman,
		Subject:  subject,
		Email:    claims.Email,
		Realm:    v.cfg.Realm,
		Roles:    append([]string(nil), claims.Roles...),
		Scopes:   append([]string(nil), claims.Scopes...),
		Verified: true,
		Issuer:   claims.Issuer,
	}
	if v.cfg.Validate == nil {
		return principal, nil
	}
	validated, validateErr := v.cfg.Validate(ctx, raw)
	if validateErr != nil {
		return Principal{}, validateErr
	}
	if !validated.Verified || strings.TrimSpace(validated.Subject) == "" {
		return Principal{}, fmt.Errorf("%w: authenticator rejected credential", ErrRevokedCredential)
	}
	if validated.Kind == cliutil.CallerKindUnknown {
		validated.Kind = cliutil.CallerKindHuman
	}
	if validated.Issuer == "" {
		validated.Issuer = claims.Issuer
	}
	if validated.Realm == "" {
		validated.Realm = v.cfg.Realm
	}
	return validated, nil
}

// cryptoHashSHA256 is kept as a constant-like value without importing a JWT
// package. rsa.VerifyPKCS1v15 accepts crypto.Hash and the standard library
// provides the complete verification primitive.
var cryptoHashSHA256 = crypto.SHA256

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}
type jwtClaims struct {
	UserID    string   `json:"user_id"`
	Subject   string   `json:"sub"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Scopes    []string `json:"scope"`
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
}

func parseJWT(raw string) (jwtHeader, jwtClaims, []byte, string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return jwtHeader{}, jwtClaims{}, nil, "", errors.New("malformed token")
	}
	decode := func(value string, target any) error {
		data, err := base64.RawURLEncoding.DecodeString(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, target)
	}
	var header jwtHeader
	var claims struct {
		UserID    string          `json:"user_id"`
		Subject   string          `json:"sub"`
		Email     string          `json:"email"`
		Roles     []string        `json:"roles"`
		Scopes    json.RawMessage `json:"scope"`
		Issuer    string          `json:"iss"`
		Audience  json.RawMessage `json:"aud"`
		ExpiresAt int64           `json:"exp"`
	}
	if err := decode(parts[0], &header); err != nil {
		return jwtHeader{}, jwtClaims{}, nil, "", err
	}
	if err := decode(parts[1], &claims); err != nil {
		return jwtHeader{}, jwtClaims{}, nil, "", err
	}
	parseStrings := func(raw json.RawMessage) []string {
		var many []string
		if json.Unmarshal(raw, &many) == nil {
			return many
		}
		var one string
		if json.Unmarshal(raw, &one) == nil && one != "" {
			return strings.Fields(one)
		}
		return nil
	}
	return header, jwtClaims{UserID: claims.UserID, Subject: claims.Subject, Email: claims.Email, Roles: claims.Roles, Scopes: parseStrings(claims.Scopes), Issuer: claims.Issuer, Audience: parseStrings(claims.Audience), ExpiresAt: claims.ExpiresAt},
		mustDecode(parts[2]), parts[0] + "." + parts[1], nil
}

func mustDecode(value string) []byte {
	data, _ := base64.RawURLEncoding.DecodeString(value)
	return data
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}
type jwk struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *JWTVerifier) publicKeys(ctx context.Context, forceRefresh bool) (map[string]*rsa.PublicKey, error) {
	now := v.cfg.Now().UTC()
	v.mu.RLock()
	if !forceRefresh && len(v.keys) > 0 && now.Before(v.keysFetchedAt.Add(v.cfg.JWKSCacheTTL)) {
		keys := v.keys
		v.mu.RUnlock()
		return keys, nil
	}
	v.mu.RUnlock()
	url := strings.TrimSpace(v.cfg.JWKSURL)
	if url == "" {
		base, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-authenticator")
		if err != nil || strings.TrimSpace(base) == "" {
			return nil, fmt.Errorf("%w: %v", ErrAuthenticatorUnavailable, err)
		}
		url = strings.TrimRight(base, "/") + "/.well-known/jwks.json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticatorUnavailable, err)
	}
	resp, err := v.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticatorUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: jwks status %d", ErrAuthenticatorUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticatorUnavailable, err)
	}
	var set jwksResponse
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("%w: invalid jwks", ErrAuthenticatorUnavailable)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, item := range set.Keys {
		if item.Kty != "RSA" || (item.Alg != "" && item.Alg != "RS256") || item.Kid == "" {
			continue
		}
		n, errN := base64.RawURLEncoding.DecodeString(item.N)
		e, errE := base64.RawURLEncoding.DecodeString(item.E)
		if errN != nil || errE != nil {
			continue
		}
		// JWK modulus/exponent are not a DER key. Build the RSA key directly.
		bigN := new(big.Int).SetBytes(n)
		exponent := new(big.Int).SetBytes(e).Int64()
		if bigN.Sign() <= 0 || exponent <= 0 {
			continue
		}
		keys[item.Kid] = &rsa.PublicKey{N: bigN, E: int(exponent)}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: no usable RSA keys", ErrAuthenticatorUnavailable)
	}
	v.mu.Lock()
	v.keys = keys
	v.keysFetchedAt = now
	v.mu.Unlock()
	return keys, nil
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type authFailureContextKey struct{}

func WithAuthFailure(ctx context.Context, err error) context.Context {
	return context.WithValue(ctx, authFailureContextKey{}, err)
}

func AuthFailureFromContext(ctx context.Context) error {
	if err, ok := ctx.Value(authFailureContextKey{}).(error); ok {
		return err
	}
	return nil
}

// Middleware attaches a verified principal. Missing or invalid credentials do
// not break read-only routes; mutation boundaries translate the recorded
// failure into an actionable 401/403 response.
func Middleware(verifier PrincipalVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if token := extractAccessToken(r); token != "" && verifier != nil {
				principal, err := verifier.Verify(ctx, token)
				if err == nil {
					next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, principal)))
					return
				}
				ctx = WithAuthFailure(ctx, err)
			} else if verifier == nil {
				ctx = WithAuthFailure(ctx, ErrAuthenticatorUnavailable)
			}
			// A verified agent identity is attribution only. api-core provenance is
			// the only accepted source; X-Vrooli-Caller is intentionally ignored.
			prov := provenance.FromContext(ctx)
			if prov.IsVerifiedAgent() {
				ctx = WithPrincipal(ctx, Principal{Kind: cliutil.CallerKindVrooliAgent, Subject: prov.RunID, Verified: true, Scopes: append([]string(nil), prov.Scopes...)})
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractAccessToken(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	for _, name := range []string{accessTokenCookieName, "access_token"} {
		if cookie, err := r.Cookie(name); err == nil && strings.TrimSpace(cookie.Value) != "" {
			return strings.TrimSpace(cookie.Value)
		}
	}
	return ""
}
