package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/cloudflareaccess"
	"github.com/vrooli/api-core/identity"
)

func TestManagedCloudflareResolvesOnlyPresentedProof(t *testing.T) {
	calls := 0
	p := NewManagedCloudflareProvider("agent-manager")
	p.Resolve = func(context.Context, string) (cloudflareaccess.Config, error) {
		calls++
		return cloudflareaccess.Config{}, errors.New("binding unavailable")
	}
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	_, err := p.VerifyRequest(r.Context(), r)
	if f, ok := identity.FailureFromError(err); !ok || f.Class != identity.FailureMissing || calls != 0 {
		t.Fatalf("absent proof acquired a dependency: calls=%d err=%v", calls, err)
	}
	r.Header.Set("Cf-Access-Jwt-Assertion", "offered-proof")
	_, err = p.VerifyRequest(r.Context(), r)
	if f, ok := identity.FailureFromError(err); !ok || f.Class != identity.FailureUnavailable || calls != 1 {
		t.Fatalf("binding outage did not fail closed: calls=%d err=%v", calls, err)
	}
}

func TestManagedCloudflareBindingReachabilitySelfCheck(t *testing.T) {
	// A deliberately broken/unreachable binding must fail loudly at the self-check
	// rather than silently blocking a human login later.
	broken := NewManagedCloudflareProvider("agent-manager")
	broken.Resolve = func(context.Context, string) (cloudflareaccess.Config, error) {
		return cloudflareaccess.Config{}, errors.New("tunnel-manager unreachable")
	}
	if err := broken.CheckBindingReachable(context.Background()); !errors.Is(err, ErrBindingUnreachable) {
		t.Fatalf("broken binding did not fail the self-check: %v", err)
	}

	// A resolvable binding that nonetheless cannot build a verifier (empty audience)
	// is also unreachable: the check must not report a usable binding.
	empty := NewManagedCloudflareProvider("agent-manager")
	empty.Resolve = func(context.Context, string) (cloudflareaccess.Config, error) {
		return cloudflareaccess.Config{}, nil
	}
	if err := empty.CheckBindingReachable(context.Background()); !errors.Is(err, ErrBindingUnreachable) {
		t.Fatalf("unusable binding passed the self-check: %v", err)
	}

	// A good binding passes and primes the cache so the first real request does
	// not pay the resolution.
	good := NewManagedCloudflareProvider("agent-manager")
	resolved := 0
	good.Resolve = func(context.Context, string) (cloudflareaccess.Config, error) {
		resolved++
		return cloudflareaccess.Config{
			TeamDomain:   "https://team.example.test",
			Audience:     "agent-manager-audience",
			RequireUser:  true,
			ClockSkew:    cloudflareaccess.DefaultClockSkew,
			JWKSCacheTTL: cloudflareaccess.DefaultJWKSCacheTTL,
			TokenLimit:   cloudflareaccess.DefaultTokenLimit,
			Client:       &http.Client{},
		}, nil
	}
	if err := good.CheckBindingReachable(context.Background()); err != nil {
		t.Fatalf("good binding failed the self-check: %v", err)
	}
	if resolved != 1 {
		t.Fatalf("self-check resolved the binding %d times, want 1", resolved)
	}
	// The cache is primed: a presented proof must not re-resolve the binding
	// (verification itself will fail on the fake assertion, but resolution is
	// what the self-check was meant to amortize).
	r := httptest.NewRequest(http.MethodGet, "/mint", nil)
	r.Header.Set("Cf-Access-Jwt-Assertion", "fake-assertion")
	_, _ = good.VerifyRequest(r.Context(), r)
	if resolved != 1 {
		t.Fatalf("primed cache was not reused: resolved=%d", resolved)
	}
}

func TestManagedCloudflarePreservesExplicitProviderAndOwnerPolicy(t *testing.T) {
	explicit := NewManagedCloudflareProvider("explicit")
	c := Config{Providers: []Provider{explicit}}.WithManagedCloudflare("agent-manager")
	if len(c.Providers) != 1 || c.Providers[0] != explicit || len(c.OwnerBindings) != 0 {
		t.Fatal("managed provider replaced explicit configuration or inferred owner authority")
	}
}
