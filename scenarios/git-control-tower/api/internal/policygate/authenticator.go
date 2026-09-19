package policygate

// This file is the relying-party adapter for scenario-authenticator. GCT does
// not issue credentials and never treats a caller header as a principal.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
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
		audience = strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_AUDIENCE"))
	}
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
	cfg      AuthenticatorConfig
	verifier *authn.JWTVerifier
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
	jwtConfig := authn.JWTConfig{
		Source:     identity.SourceScenarioAuthenticator,
		Issuer:     cfg.Issuer,
		Audience:   cfg.Audience,
		Kind:       identity.ActorHuman,
		Client:     cfg.Client,
		Now:        cfg.Now,
		CacheTTL:   cfg.JWKSCacheTTL,
		StaleGrace: 24 * time.Hour,
	}
	if strings.TrimSpace(cfg.JWKSURL) != "" {
		jwtConfig.JWKSURL = strings.TrimSpace(cfg.JWKSURL)
	} else {
		jwtConfig.ResolveJWKS = func(ctx context.Context) (string, error) {
			base, err := discovery.ResolveScenarioURLDefault(ctx, DefaultAuthenticatorIssuer)
			if err != nil {
				return "", err
			}
			return strings.TrimRight(base, "/") + "/.well-known/jwks.json", nil
		}
	}
	return &JWTVerifier{cfg: cfg, verifier: authn.NewJWTVerifier(jwtConfig)}
}

func (v *JWTVerifier) Verify(ctx context.Context, raw string) (Principal, error) {
	shared, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		failure, ok := identity.FailureFromError(err)
		if ok && failure.Class == identity.FailureUnavailable {
			return Principal{}, fmt.Errorf("%w: %v", ErrAuthenticatorUnavailable, err)
		}
		return Principal{}, fmt.Errorf("%w: %v", ErrInvalidCredential, err)
	}
	realm := strings.TrimSpace(shared.Realm)
	if realm == "" {
		realm = v.cfg.Realm
	}
	principal := Principal{
		Kind:     cliutil.CallerKindHuman,
		Subject:  shared.Subject,
		Email:    shared.Email,
		Realm:    realm,
		Roles:    append([]string(nil), shared.Roles...),
		Scopes:   append([]string(nil), shared.Scopes...),
		Verified: shared.Verified,
		Issuer:   shared.Issuer,
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
		validated.Issuer = shared.Issuer
	}
	if validated.Realm == "" {
		validated.Realm = v.cfg.Realm
	}
	return validated, nil
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
