package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

func TestOperationGrantCannotApproveHumanDecisionsOrOtherObjects(t *testing.T) {
	g := &identity.OperationGrant{ID: "grant", Approver: "owner", Operations: []string{"swarm.bookkeeping"}, Targets: []string{"swarm-manager:backlog/execute/obsolete"}, ExpiresAt: time.Now().Add(time.Hour)}
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: []string{"swarm-manager:write"}, Grant: g}
	ctx := identity.WithPrincipal(context.Background(), p)
	if _, err := RequireOperation(ctx, "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:backlog/execute/obsolete"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][2]string{{"swarm.accept", "swarm-manager:backlog/execute/obsolete"}, {"swarm.bookkeeping", "swarm-manager:backlog/execute/other"}, {"swarm.bookkeeping", "git-control-tower:*"}} {
		if _, err := RequireOperation(ctx, "swarm-manager:write", args[0], args[1]); err == nil {
			t.Fatalf("accepted unapproved operation/target: %v", args)
		}
	}
	if _, err := RequireHuman(ctx); err == nil {
		t.Fatal("grant approver became the agent actor")
	}
	g.ExpiresAt = time.Now().Add(-time.Second)
	if _, err := RequireOperation(ctx, "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:backlog/execute/obsolete"); err == nil {
		t.Fatal("expired grant accepted")
	}
}

func TestTeamPreparationPolicyDoesNotGrantActivationOrOtherObjects(t *testing.T) {
	p, ok := InteractivePolicy("team-preparation")
	if !ok || len(p.Operations) != 1 || p.Operations[0] != "team.prepare" || len(p.Domains) != 1 || p.Domains[0] != "prompt-manager" {
		t.Fatalf("invalid bounded policy: %+v", p)
	}
	grant := &identity.OperationGrant{ID: "g", Approver: "owner", Operations: p.Operations, Targets: []string{"prompt-manager:team/draft"}, ExpiresAt: time.Now().Add(time.Hour)}
	principal := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: p.Scopes, Grant: grant}
	ctx := identity.WithPrincipal(context.Background(), principal)
	if _, err := RequireOperation(ctx, "prompt-manager:write", "team.prepare", "prompt-manager:team/draft"); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"team.activate", "prompt-manager:team/draft"}, {"team.prepare", "prompt-manager:team/other"}, {"team.prepare", "agent-manager:effort/draft"}} {
		if _, err := RequireOperation(ctx, "prompt-manager:write", pair[0], pair[1]); err == nil {
			t.Fatalf("grant escaped preparation: %v", pair)
		}
	}
}

func TestMiddlewarePreservesVerifiedAgentWithoutHumanProviders(t *testing.T) {
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run", Subject: "owner", Scopes: []string{"swarm-manager:write"}})
	request := httptest.NewRequest(http.MethodGet, "/api", nil).WithContext(ctx)
	request.Header.Set("X-Agent-Identity-Token", "verified")
	called := false
	Middleware(Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		p, ok := identity.PrincipalFromContext(r.Context())
		if !ok || p.Kind != identity.ActorAgent || p.Subject != "run" || p.OwnerSubject() != "owner" {
			t.Fatalf("agent principal lost: %+v", p)
		}
	})).ServeHTTP(httptest.NewRecorder(), request)
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestOwnerBindingRequiresExactVerifiedProviderSubject(t *testing.T) {
	provider := TokenProvider{ProviderSource: identity.SourceCloudflareAccess, Extract: func(*http.Request) string { return "proof" }, Verifier: operationVerifier{}}
	cfg := Config{Providers: []Provider{provider}, OwnerBindings: []OwnerBinding{{Source: identity.SourceCloudflareAccess, Subject: "cf-sub", CanonicalSubject: "owner", Scopes: []string{"agent-manager:write"}}}}
	p, err := cfg.Authenticate(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil || p.Subject != "owner" || len(p.Scopes) != 1 {
		t.Fatalf("binding failed: %+v %v", p, err)
	}
	cfg.OwnerBindings[0].Source = identity.SourceScenarioAuthenticator
	p, err = cfg.Authenticate(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil || p.Subject != "cf-sub" || len(p.Scopes) != 0 {
		t.Fatal("a different provider supplied owner authority")
	}
}

type operationVerifier struct{}

func (operationVerifier) Verify(context.Context, string) (identity.Principal, error) {
	return identity.Principal{Kind: identity.ActorHuman, Subject: "cf-sub", Email: "owner@example.test", Verified: true, Source: identity.SourceCloudflareAccess}, nil
}

func TestInvalidOfferedProofClearsAnEarlierPrincipal(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "previous-owner", Verified: true, Scopes: []string{"swarm-manager:write"}})
	req := httptest.NewRequest(http.MethodPost, "/api", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer offered-without-provider")
	Middleware(Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := identity.PrincipalFromContext(r.Context()); ok {
			t.Fatal("earlier principal survived invalid offered proof")
		}
	})).ServeHTTP(httptest.NewRecorder(), req)
}

func TestAgentAttributionCannotBorrowTheHumanPrincipalForADecision(t *testing.T) {
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "worker"})
	ctx = identity.WithPrincipal(ctx, identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true})
	if _, err := RequireIndependentHuman(ctx); err == nil {
		t.Fatal("mixed proof made the agent a human decision maker")
	}
}

func TestForwardedOwnerProofCannotExpandAnInteractiveGrant(t *testing.T) {
	grant := &identity.OperationGrant{ID: "grant", Approver: "owner", Operations: []string{"swarm.bookkeeping"}, Targets: []string{"swarm-manager:backlog/execute/allowed"}, ExpiresAt: time.Now().Add(time.Hour)}
	for _, subject := range []string{"owner", "other-owner"} {
		ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "worker", Subject: "owner", Scopes: []string{"swarm-manager:write"}, Grant: grant})
		provider := fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
			return identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true, Source: identity.SourceCloudflareAccess, Scopes: []string{"swarm-manager:*"}}, nil
		}}
		Middleware(Config{Providers: []Provider{provider}})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			if subject != "owner" {
				if _, ok := identity.PrincipalFromContext(r.Context()); ok {
					t.Fatal("conflicting owner survived")
				}
				return
			}
			if _, err := RequireOperation(r.Context(), "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:backlog/execute/allowed"); err != nil {
				t.Fatal(err)
			}
			if _, err := RequireOperation(r.Context(), "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:backlog/execute/other"); err == nil {
				t.Fatal("human credential expanded agent target authority")
			}
			if _, err := RequireHuman(r.Context()); err == nil {
				t.Fatal("forwarded proof changed the agent actor")
			}
		})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx))
	}
}
