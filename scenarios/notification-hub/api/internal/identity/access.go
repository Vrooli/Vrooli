package identity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/user"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/cloudflareaccess"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"
)

// The hub trusts a Cloudflare Access login as its owner, as vrooli-onboarding
// does: the operator passes Cloudflare once and needs no Vrooli password. The
// scenario-authenticator bearer and cookie remain a secondary path (CLI and
// signed-in browsers without Access).

const (
	accessAssertionHeader = "Cf-Access-Jwt-Assertion"
	// accessBindingRetry spaces binding lookups after a failure, so a
	// tunnel-manager or Cloudflare outage costs one call per minute.
	accessBindingRetry = time.Minute
	// accessBindingTTL re-reads a resolved binding so a re-created Access
	// application's new audience is picked up without a restart.
	accessBindingTTL = time.Hour
)

type ownerContextKey struct{}

type ownerContext struct {
	identity owneridentity.Identity
	refused  bool
}

// OwnerAuthenticator verifies the request-bound owner providers once per
// request. A verified human is the owner; the recipient subject comes from
// OwnerSubject, so the signed-in owner and the recipient of inbound asks are
// the same subject. A presented credential that fails verification is
// refused. A missing credential, or a provider that cannot verify right now,
// leaves the request to the bearer and cookie path.
type OwnerAuthenticator struct {
	Providers    []authn.Provider
	OwnerSubject func(context.Context) string
}

func (a OwnerAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(a.authenticate(r)))
	})
}

func (a OwnerAuthenticator) authenticate(r *http.Request) context.Context {
	ctx := r.Context()
	if len(a.Providers) == 0 {
		return ctx
	}
	principal, err := authn.Config{Providers: a.Providers}.Authenticate(ctx, r)
	if err != nil {
		failure, ok := apiidentity.FailureFromError(err)
		if ok && (failure.Class == apiidentity.FailureMissing || failure.Class == apiidentity.FailureUnavailable) {
			return ctx
		}
		return context.WithValue(ctx, ownerContextKey{}, ownerContext{refused: true})
	}
	if !principal.IsHuman() || a.OwnerSubject == nil {
		return ctx
	}
	subject := strings.TrimSpace(a.OwnerSubject(ctx))
	if subject == "" {
		return ctx
	}
	return context.WithValue(ctx, ownerContextKey{}, ownerContext{identity: owneridentity.Identity{
		Subject: subject, Email: principal.Email, ExpiresAt: principal.ExpiresAt,
	}})
}

func ownerFromContext(ctx context.Context) (ownerContext, bool) {
	if ctx == nil {
		return ownerContext{}, false
	}
	owner, ok := ctx.Value(ownerContextKey{}).(ownerContext)
	return owner, ok
}

// OwnerProvidersFromEnvironment selects the owner providers the way
// vrooli-onboarding selects its mode: VROOLI_AUTH_MODE=personal_local is the
// explicit loopback opt-in; every other mode trusts Cloudflare Access.
func OwnerProvidersFromEnvironment(getenv func(string) string, resolver URLResolver, scenario string) []authn.Provider {
	if strings.EqualFold(strings.TrimSpace(getenv("VROOLI_AUTH_MODE")), "personal_local") {
		return []authn.Provider{LoopbackOwnerProvider{}}
	}
	return []authn.Provider{NewAccessProvider(AccessBindingFromEnvironment(getenv, resolver, scenario))}
}

// AccessBindingSource returns the Access application's verifier
// configuration (team domain and audience).
type AccessBindingSource func(context.Context) (cloudflareaccess.Config, error)

// AccessBindingFromEnvironment reads the same source as vrooli-onboarding:
// the VROOLI_CLOUDFLARE_ACCESS_* runtime binding when the environment carries
// it, else tunnel-manager's GetAuthenticationBinding for this scenario's
// route, which is what the lifecycle injects for onboarding.
func AccessBindingFromEnvironment(getenv func(string) string, resolver URLResolver, scenario string) AccessBindingSource {
	return func(ctx context.Context) (cloudflareaccess.Config, error) {
		if strings.TrimSpace(getenv("VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN")) != "" || strings.TrimSpace(getenv("VROOLI_CLOUDFLARE_ACCESS_AUDIENCE")) != "" {
			return cloudflareaccess.ConfigFromEnv(getenv)
		}
		base := strings.TrimSpace(getenv("VROOLI_TUNNEL_MANAGER_API_BASE"))
		if base == "" {
			if resolver == nil {
				return cloudflareaccess.Config{}, errors.New("tunnel-manager is not resolvable")
			}
			var err error
			if base, err = resolver.ResolveScenarioURLDefault(ctx, "tunnel-manager"); err != nil {
				return cloudflareaccess.Config{}, fmt.Errorf("discover tunnel-manager: %w", err)
			}
		}
		client := configconnect.NewConfigServiceClient(&http.Client{Timeout: 10 * time.Second}, strings.TrimRight(base, "/"))
		response, err := client.GetAuthenticationBinding(ctx, connect.NewRequest(&configv1.GetAuthenticationBindingRequest{Scenario: scenario}))
		if err != nil {
			return cloudflareaccess.Config{}, fmt.Errorf("tunnel-manager authentication binding: %w", err)
		}
		binding := response.Msg.GetBinding()
		cfg := cloudflareaccess.Config{TeamDomain: binding.GetTeamDomain(), Audience: binding.GetAudience(), RequireUser: true}
		if strings.TrimSpace(cfg.TeamDomain) == "" || strings.TrimSpace(cfg.Audience) == "" {
			return cloudflareaccess.Config{}, errors.New("tunnel-manager returned an incomplete authentication binding")
		}
		return cfg, nil
	}
}

// AccessProvider verifies Cf-Access-Jwt-Assertion with the shared api-core
// verifier. It resolves the binding on the first assertion, not at startup,
// so a tunnel-manager or Cloudflare outage never stops the hub: it only makes
// the Access path unavailable until the binding resolves.
type AccessProvider struct {
	resolve AccessBindingSource
	now     func() time.Time

	mu         sync.Mutex
	verifier   *cloudflareaccess.Verifier
	resolvedAt time.Time
	failedAt   time.Time
	lastErr    error
}

func NewAccessProvider(resolve AccessBindingSource) *AccessProvider {
	return &AccessProvider{resolve: resolve, now: time.Now}
}

func (p *AccessProvider) Source() apiidentity.AuthSource { return apiidentity.SourceCloudflareAccess }

func (p *AccessProvider) VerifyRequest(ctx context.Context, req *http.Request) (apiidentity.Principal, error) {
	if req == nil || (strings.TrimSpace(req.Header.Get(accessAssertionHeader)) == "" &&
		strings.TrimSpace(req.Header.Get("CF-Access-Client-Id")) == "" && strings.TrimSpace(req.Header.Get("CF-Access-Client-Secret")) == "") {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureMissing, p.Source())
	}
	verifier, err := p.current(ctx)
	if err != nil {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureUnavailable, p.Source())
	}
	return verifier.VerifyRequest(ctx, req)
}

func (p *AccessProvider) current(ctx context.Context) (*cloudflareaccess.Verifier, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	if p.verifier != nil && now.Sub(p.resolvedAt) < accessBindingTTL {
		return p.verifier, nil
	}
	if p.lastErr != nil && now.Sub(p.failedAt) < accessBindingRetry {
		if p.verifier != nil {
			return p.verifier, nil
		}
		return nil, p.lastErr
	}
	verifier, err := p.build(ctx)
	if err != nil {
		p.lastErr, p.failedAt = err, now
		if p.verifier != nil {
			// Keep verifying with the last good binding while a refresh fails.
			return p.verifier, nil
		}
		return nil, err
	}
	p.verifier, p.resolvedAt, p.lastErr = verifier, now, nil
	return verifier, nil
}

func (p *AccessProvider) build(ctx context.Context) (*cloudflareaccess.Verifier, error) {
	if p.resolve == nil {
		return nil, errors.New("cloudflare access binding is not configured")
	}
	cfg, err := p.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return cloudflareaccess.NewVerifier(cfg)
}

// LoopbackOwnerProvider is vrooli-onboarding's personal_local contract: in
// that explicit mode a loopback request is the current OS user. A request
// that arrived through the Cloudflare tunnel is not local, even though the
// tunnel connector reaches the origin over loopback.
type LoopbackOwnerProvider struct{}

func (LoopbackOwnerProvider) Source() apiidentity.AuthSource { return apiidentity.SourcePersonalLocal }

func (p LoopbackOwnerProvider) VerifyRequest(_ context.Context, req *http.Request) (apiidentity.Principal, error) {
	if req == nil || !isLoopback(req.RemoteAddr) ||
		strings.TrimSpace(req.Header.Get("Cf-Connecting-Ip")) != "" || strings.TrimSpace(req.Header.Get(accessAssertionHeader)) != "" {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureMissing, p.Source())
	}
	current, err := user.Current()
	if err != nil || current == nil || strings.TrimSpace(current.Uid) == "" {
		return apiidentity.Principal{}, apiidentity.NewFailure(apiidentity.FailureUnavailable, p.Source())
	}
	return apiidentity.Principal{
		Kind: apiidentity.ActorHuman, Subject: "osuser:" + strings.TrimSpace(current.Uid), Realm: "personal_local",
		Verified: true, Source: p.Source(), Sources: []apiidentity.AuthSource{p.Source()},
	}, nil
}

func isLoopback(remoteAddr string) bool {
	host := strings.TrimSpace(remoteAddr)
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

// Refused reports a request whose presented owner credential (for example an
// expired Cloudflare Access assertion) failed verification. Such a request
// must not be signed in by a weaker path such as a session refresh.
func Refused(ctx context.Context) bool {
	owner, ok := ownerFromContext(ctx)
	return ok && owner.refused
}
