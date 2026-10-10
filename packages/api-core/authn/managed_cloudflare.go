package authn

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/cloudflareaccess"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"
)

// ManagedCloudflareProvider obtains non-secret application bindings from their
// existing Tunnel Manager owner. Resolution is lazy and cached briefly, so a
// local read-only scenario does not acquire a startup/network dependency just
// to verify an occasional externally authenticated operator request.
type ManagedCloudflareProvider struct {
	Resource string
	Resolve  func(context.Context, string) (cloudflareaccess.Config, error)
	mu       sync.Mutex
	verifier *cloudflareaccess.Verifier
	until    time.Time
}

func NewManagedCloudflareProvider(resource string) *ManagedCloudflareProvider {
	return &ManagedCloudflareProvider{Resource: resource, Resolve: resolveManagedCloudflareBinding}
}

func (*ManagedCloudflareProvider) Source() identity.AuthSource {
	return identity.SourceCloudflareAccess
}

func (p *ManagedCloudflareProvider) VerifyRequest(ctx context.Context, r *http.Request) (identity.Principal, error) {
	if r == nil || len(r.Header.Values("Cf-Access-Jwt-Assertion")) == 0 {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.Source())
	}
	if strings.TrimSpace(r.Header.Get("Cf-Access-Jwt-Assertion")) == "" {
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, p.Source())
	}
	p.mu.Lock()
	verifier := p.verifier
	if verifier == nil || !time.Now().Before(p.until) {
		resolve := p.Resolve
		if resolve == nil {
			resolve = resolveManagedCloudflareBinding
		}
		cfg, err := resolve(ctx, p.Resource)
		if err != nil {
			p.mu.Unlock()
			return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, p.Source())
		}
		verifier, err = cloudflareaccess.NewVerifier(cfg)
		if err != nil {
			p.mu.Unlock()
			return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, p.Source())
		}
		p.verifier, p.until = verifier, time.Now().Add(5*time.Minute)
	}
	p.mu.Unlock()
	return verifier.VerifyRequest(ctx, r)
}

func resolveManagedCloudflareBinding(ctx context.Context, resource string) (cloudflareaccess.Config, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "tunnel-manager")
	if err != nil {
		return cloudflareaccess.Config{}, err
	}
	client := configconnect.NewConfigServiceClient(&http.Client{Timeout: 5 * time.Second}, base)
	response, err := client.GetAuthenticationBinding(ctx, connect.NewRequest(&configv1.GetAuthenticationBindingRequest{Scenario: resource}))
	if err != nil {
		return cloudflareaccess.Config{}, err
	}
	if response.Msg == nil || response.Msg.Binding == nil {
		return cloudflareaccess.Config{}, ErrNoProvider
	}
	binding := response.Msg.Binding
	return cloudflareaccess.Config{TeamDomain: binding.TeamDomain, Audience: binding.Audience, RequireUser: true, ClockSkew: cloudflareaccess.DefaultClockSkew, JWKSCacheTTL: cloudflareaccess.DefaultJWKSCacheTTL, TokenLimit: cloudflareaccess.DefaultTokenLimit, Client: &http.Client{Timeout: 5 * time.Second}}, nil
}

// WithManagedCloudflare retains an explicitly configured verifier when one
// exists. It never assigns capabilities or canonical owner mappings.
func (c Config) WithManagedCloudflare(resource string) Config {
	for _, provider := range c.Providers {
		if provider != nil && provider.Source() == identity.SourceCloudflareAccess {
			return c
		}
	}
	c.Providers = append(append([]Provider(nil), c.Providers...), NewManagedCloudflareProvider(resource))
	return c
}
