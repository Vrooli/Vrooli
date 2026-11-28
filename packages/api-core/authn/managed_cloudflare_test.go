package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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

func TestManagedCloudflareConcurrentBindingOutageFailsClosedWithoutSerialRetries(t *testing.T) {
	p := NewManagedCloudflareProvider("web-console")
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	p.Resolve = func(context.Context, string) (cloudflareaccess.Config, error) {
		calls++
		if calls == 1 {
			close(started)
			<-release
		}
		return cloudflareaccess.Config{}, errors.New("binding unavailable")
	}
	r := httptest.NewRequest(http.MethodPost, "/sessions", nil)
	r.Header.Set("Cf-Access-Jwt-Assertion", "offered-proof")
	const requests = 4
	results := make(chan error, requests)
	var wg sync.WaitGroup
	wg.Add(requests)
	for i := 0; i < requests; i++ {
		go func() {
			defer wg.Done()
			_, err := p.VerifyRequest(r.Context(), r)
			results <- err
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(results)
	for err := range results {
		if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureUnavailable {
			t.Fatalf("binding outage granted or changed authentication result: %v", err)
		}
	}
	if calls != 1 {
		t.Fatalf("concurrent failed binding lookups = %d, want 1", calls)
	}

	// Retry after the short negative cache expires; an outage must not become
	// a permanent authentication state.
	p.mu.Lock()
	p.retryAfter = time.Now().Add(-time.Second)
	p.mu.Unlock()
	_, err := p.VerifyRequest(r.Context(), r)
	if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureUnavailable || calls != 2 {
		t.Fatalf("binding retry did not fail closed and recheck: calls=%d err=%v", calls, err)
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
