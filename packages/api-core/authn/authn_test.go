package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/identity"
)

func TestFromEnvironmentBuildsSharedCloudflareProvider(t *testing.T) {
	values := map[string]string{
		"VROOLI_AUTH_PROVIDERS":                "scenario_authenticator, cloudflare_access",
		"VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN": "https://team.example.test",
		"VROOLI_CLOUDFLARE_ACCESS_AUDIENCE":    "gct-audience",
		"VROOLI_CLOUDFLARE_ACCESS_JWKS_URL":    "https://team.example.test/certs",
		"VROOLI_AUTH_RECOVERY_URL":             "https://team.example.test/cdn-cgi/access/login",
	}
	cfg, err := FromEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatalf("FromEnvironment() error = %v", err)
	}
	if len(cfg.Providers) != 2 || cfg.Providers[0].Source() != identity.SourceScenarioAuthenticator || cfg.Providers[1].Source() != identity.SourceCloudflareAccess {
		t.Fatalf("providers=%v", configuredSources(cfg.Providers))
	}
	if cfg.RecoveryURL != values["VROOLI_AUTH_RECOVERY_URL"] {
		t.Fatalf("recovery URL=%q", cfg.RecoveryURL)
	}
}

type fakeProvider struct {
	source identity.AuthSource
	fn     func() (identity.Principal, error)
}

func (p fakeProvider) Source() identity.AuthSource { return p.source }
func (p fakeProvider) VerifyRequest(_ context.Context, _ *http.Request) (identity.Principal, error) {
	return p.fn()
}

func TestAuthenticateRejectsConflictingVerifiedSubjects(t *testing.T) {
	c := Config{Providers: []Provider{
		fakeProvider{source: identity.SourceScenarioAuthenticator, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorHuman, Subject: "one", Verified: true, Source: identity.SourceScenarioAuthenticator}, nil
		}},
		fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorHuman, Subject: "two", Verified: true, Source: identity.SourceCloudflareAccess}, nil
		}},
	}}
	_, err := c.Authenticate(t.Context(), httptest.NewRequest(http.MethodGet, "/", nil))
	if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureConflict {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateRequiresExplicitIdentityMapping(t *testing.T) {
	providers := []Provider{
		fakeProvider{source: identity.SourceScenarioAuthenticator, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorHuman, Subject: "same-user", Verified: true, Source: identity.SourceScenarioAuthenticator}, nil
		}},
		fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorHuman, Subject: "same-user", Verified: true, Source: identity.SourceCloudflareAccess}, nil
		}},
	}
	_, err := (Config{Providers: providers}).Authenticate(t.Context(), httptest.NewRequest(http.MethodGet, "/", nil))
	if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureConflict {
		t.Fatalf("without mapping err=%v", err)
	}
	c := Config{Providers: providers, IdentityMappings: []IdentityMapping{{
		LeftSource: identity.SourceScenarioAuthenticator, LeftSubject: "same-user",
		RightSource: identity.SourceCloudflareAccess, RightSubject: "same-user",
	}}}
	principal, err := c.Authenticate(t.Context(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if !principal.IsHuman() || len(principal.Sources) != 2 || principal.Sources[0] != identity.SourceScenarioAuthenticator || principal.Sources[1] != identity.SourceCloudflareAccess {
		t.Fatalf("principal=%#v", principal)
	}
}

func TestRequireKindsAndCapabilities(t *testing.T) {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Verified: true, Scopes: []string{"demo:write"}}
	ctx := identity.WithPrincipal(t.Context(), principal)
	if got, err := RequireHuman(ctx); err != nil || got.Subject != "human-1" {
		t.Fatalf("RequireHuman() = %#v, %v", got, err)
	}
	if _, err := RequireAgent(ctx); !errors.Is(err, ErrAgentRequired) {
		t.Fatalf("RequireAgent() error = %v", err)
	}
	if _, err := RequireService(ctx); !errors.Is(err, ErrServiceRequired) {
		t.Fatalf("RequireService() error = %v", err)
	}
	if _, err := RequireCapability(ctx, "demo:write"); err != nil {
		t.Fatalf("RequireCapability() error = %v", err)
	}
	if _, err := RequireCapability(ctx, "demo:destructive"); !errors.Is(err, ErrCapability) {
		t.Fatalf("missing capability error = %v", err)
	}
}

func TestAuthenticateRejectsPresentedInvalidBeforeHumanResult(t *testing.T) {
	c := Config{Providers: []Provider{
		fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
			return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, identity.SourceCloudflareAccess)
		}},
		fakeProvider{source: identity.SourceScenarioAuthenticator, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorHuman, Subject: "one", Verified: true, Source: identity.SourceScenarioAuthenticator}, nil
		}},
	}}
	_, err := c.Authenticate(t.Context(), httptest.NewRequest(http.MethodGet, "/", nil))
	if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureInvalid {
		t.Fatalf("err=%v", err)
	}
}

func TestAuthenticateClassifiesProviderFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		class identity.FailureClass
	}{
		{name: "missing", class: identity.FailureMissing},
		{name: "invalid", class: identity.FailureInvalid},
		{name: "expired", class: identity.FailureExpired},
		{name: "unavailable", class: identity.FailureUnavailable},
		{name: "service", class: identity.FailureService},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := fakeProvider{source: identity.SourceScenarioAuthenticator, fn: func() (identity.Principal, error) {
				return identity.Principal{}, identity.NewFailure(tc.class, identity.SourceScenarioAuthenticator)
			}}
			_, err := (Config{Providers: []Provider{provider}}).Authenticate(t.Context(), httptest.NewRequest(http.MethodGet, "/", nil))
			failure, ok := identity.FailureFromError(err)
			if !ok || failure.Class != tc.class {
				t.Fatalf("error=%v failure=%#v ok=%t", err, failure, ok)
			}
		})
	}
}

func TestMiddlewareIsPassiveAndExposesRecoveryState(t *testing.T) {
	called := false
	h := Middleware(Config{RecoveryURL: "https://team.example.test/cdn-cgi/access/login", Providers: []Provider{
		fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
			return identity.Principal{}, identity.NewFailure(identity.FailureExpired, identity.SourceCloudflareAccess)
		}},
	}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		status, ok := identity.StatusFromContext(r.Context())
		if !ok || status.State != identity.StateExpired || status.RecoveryURL == "" {
			t.Fatalf("status=%#v ok=%t", status, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if !called || res.Code != http.StatusNoContent {
		t.Fatalf("called=%t status=%d", called, res.Code)
	}
}

func TestMiddlewareLabelsVerifiedServiceAsService(t *testing.T) {
	h := Middleware(Config{Providers: []Provider{
		fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorService, Subject: "service-1", Verified: true, Source: identity.SourceCloudflareAccess}, nil
		}},
	}})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status, ok := identity.StatusFromContext(r.Context())
		if !ok || status.State != identity.StateService || !status.Authenticated {
			t.Fatalf("status=%#v ok=%t", status, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("status=%d", res.Code)
	}
}
