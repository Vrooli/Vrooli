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

func TestManagedCloudflarePreservesExplicitProviderAndOwnerPolicy(t *testing.T) {
	explicit := NewManagedCloudflareProvider("explicit")
	c := Config{Providers: []Provider{explicit}}.WithManagedCloudflare("agent-manager")
	if len(c.Providers) != 1 || c.Providers[0] != explicit || len(c.OwnerBindings) != 0 {
		t.Fatal("managed provider replaced explicit configuration or inferred owner authority")
	}
}
