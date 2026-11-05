package authn

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
)

// This file pins the shared authorization invariants documented in
// docs/concepts/AUTHORIZATION-INVARIANTS.md. Each test is one invariant; several
// invariants are already pinned by neighbouring suites and are referenced from
// the doc rather than duplicated here.

// INV-1 — An anonymous request is refused for any protected operation.
func TestInvariantAnonymousRefused(t *testing.T) {
	// No principal in context: RequireOperation must refuse (no human, no agent
	// grant, no service authority).
	if _, err := RequireOperation(context.Background(), "agent-manager:write", "run.create", "agent-manager:run/new"); err == nil {
		t.Fatal("anonymous request admitted a costly operation")
	}
	// A capability check with no principal must also refuse.
	if _, err := RequireCapability(context.Background(), "agent-manager:write"); err == nil {
		t.Fatal("anonymous request passed a capability check")
	}
	// The middleware labels an anonymous request as signed-out, never verified.
	Middleware(Config{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if _, ok := identity.PrincipalFromContext(r.Context()); ok {
			t.Fatal("anonymous request produced a verified principal")
		}
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

// INV-2 — A forged / bad-signature token is refused.
func TestInvariantForgedTokenRefused(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	attacker, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	server := httptest.NewServer(jwksHandler(key))
	defer server.Close()
	verifier := NewJWTVerifier(JWTConfig{Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator", Audience: "scenario:demo", JWKSURL: server.URL, Now: func() time.Time { return now }})
	// Signed by a key the JWKS does not publish: signature cannot validate.
	forged := signJWT(t, attacker, map[string]any{"sub": "owner-1", "iss": "scenario-authenticator", "aud": "scenario:demo", "exp": now.Add(time.Hour).Unix()})
	if _, err := verifier.Verify(context.Background(), forged); err == nil {
		t.Fatal("forged-signature token was accepted")
	}
	// Tampering the payload after signing also fails closed.
	valid := signJWT(t, key, map[string]any{"sub": "owner-1", "iss": "scenario-authenticator", "aud": "scenario:demo", "exp": now.Add(time.Hour).Unix()})
	tampered := valid[:len(valid)-4] + "AAAA"
	if _, err := verifier.Verify(context.Background(), tampered); err == nil {
		t.Fatal("tampered token was accepted")
	}
}

// INV-3 — An expired token OR an expired grant is refused.
func TestInvariantExpiredRefused(t *testing.T) {
	// Expired grant on an otherwise-verified agent principal.
	expiredGrant := &identity.OperationGrant{ID: "g", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(-time.Second)}
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: expiredGrant}
	ctx := identity.WithPrincipal(context.Background(), p)
	if _, err := RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:run/new"); err == nil {
		t.Fatal("expired grant admitted a costly operation")
	}
	// An expired human principal (token past exp) is also refused.
	expiredHuman := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"agent-manager:write"}, ExpiresAt: time.Now().Add(-time.Second)}
	hctx := identity.WithPrincipal(context.Background(), expiredHuman)
	if _, err := RequireOperation(hctx, "agent-manager:write", "run.create", "agent-manager:run/new"); err == nil {
		t.Fatal("expired human principal admitted a costly operation")
	}
}

// INV-4 — Over-limit (concurrency AND budget) is refused distinctly from "no
// authority". The ask-don't-block path is reserved for "no grant", never for an
// exhausted bound.
func TestInvariantOverLimitRefused(t *testing.T) {
	now := time.Now()
	grant := &identity.OperationGrant{ID: "g", Approver: "operator", Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: now.Add(time.Hour)}
	// Concurrency bound exhausted.
	if err := AuthorizeStandingGrant(grant, StandingGrantLimits{MaxConcurrentRuns: 2}, StandingGrantUsage{ConcurrentRuns: 2}, "run.create", "agent-manager:run/new", now); err != ErrStandingGrantLimit {
		t.Fatalf("concurrency over-limit not refused with ErrStandingGrantLimit: %v", err)
	}
	// Budget bound exhausted — proves G1 (budget) enforcement is wired, not just
	// concurrency.
	if err := AuthorizeStandingGrant(grant, StandingGrantLimits{MaxRunBudgetUSD: 50}, StandingGrantUsage{CommittedBudgetUSD: 45, RequestBudgetUSD: 10}, "run.create", "agent-manager:run/new", now); err != ErrStandingGrantLimit {
		t.Fatalf("budget over-limit not refused with ErrStandingGrantLimit: %v", err)
	}
	// Within both bounds admits.
	if err := AuthorizeStandingGrant(grant, StandingGrantLimits{MaxConcurrentRuns: 5, MaxRunBudgetUSD: 100}, StandingGrantUsage{ConcurrentRuns: 1, CommittedBudgetUSD: 40, RequestBudgetUSD: 10}, "run.create", "agent-manager:run/new", now); err != nil {
		t.Fatalf("within-limit request refused: %v", err)
	}
}

// INV-5 — A service principal cannot impersonate a human: a verified service
// never satisfies RequireHuman / RequireIndependentHuman, and the standing-grant
// ingress seam refuses a human routed through it.
func TestInvariantServiceCannotImpersonateHuman(t *testing.T) {
	service := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	ctx := identity.WithPrincipal(context.Background(), service)
	if _, err := RequireHuman(ctx); err == nil {
		t.Fatal("service principal satisfied RequireHuman")
	}
	if _, err := RequireIndependentHuman(ctx); err == nil {
		t.Fatal("service principal satisfied RequireIndependentHuman")
	}
	// A verified service cannot carry a human-only bearer either: the middleware
	// never promotes a service to human.
	h := Middleware(Config{Providers: []Provider{fakeProvider{source: identity.SourceScenarioAuthenticator, fn: func() (identity.Principal, error) {
		return identity.Principal{Kind: identity.ActorService, Subject: "svc", Verified: true, Source: identity.SourceScenarioAuthenticator}, nil
	}}}})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if _, err := RequireHuman(r.Context()); err == nil {
			t.Fatal("service was promoted to human by middleware")
		}
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

// INV-8 — One decision path per operation: middleware and the handler agree. A
// forwarded human credential cannot broaden an agent's grant at the middleware,
// and RequireOperation reaches the same refusal the middleware set up. (The
// full forwarded-owner matrix is pinned by
// operation_test.go::TestForwardedOwnerProofCannotExpandAnInteractiveGrant;
// this guards the narrower "two gates agree" property.)
func TestInvariantOneDecisionPathPerOperation(t *testing.T) {
	grant := &identity.OperationGrant{ID: "g", Approver: "owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:run/allowed"}, ExpiresAt: time.Now().Add(time.Hour)}
	p := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: grant}
	ctx := identity.WithPrincipal(context.Background(), p)
	// Allowed target admits.
	if _, err := RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:run/allowed"); err != nil {
		t.Fatalf("grant-covered operation refused: %v", err)
	}
	// A different target is refused by the same single evaluator — no second gate
	// silently widens it.
	if _, err := RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:run/other"); err == nil {
		t.Fatal("a second decision path widened the grant target")
	}
	// A different operation under the same scope is refused too.
	if _, err := RequireOperation(ctx, "agent-manager:write", "agent.manage", "agent-manager:run/allowed"); err == nil {
		t.Fatal("a second decision path widened the grant operation")
	}
}

// INV-10 — A third-party proof (Cf-Access assertion) is never trusted as a
// human unless it carries the verified human user claim. An invalid/absent
// Cf-Access assertion must not produce a human principal.
func TestInvariantThirdPartyProofNotTrustedAsHuman(t *testing.T) {
	// The managed provider with no presented assertion returns a missing failure,
	// never a human.
	p := NewManagedCloudflareProvider("agent-manager")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	principal, err := p.VerifyRequest(r.Context(), r)
	if err == nil || principal.IsHuman() {
		t.Fatalf("absent Cf-Access assertion produced a human: %#v %v", principal, err)
	}
	// An empty assertion header is invalid, not a human.
	r.Header.Set("Cf-Access-Jwt-Assertion", "   ")
	principal, err = p.VerifyRequest(r.Context(), r)
	if err == nil || principal.IsHuman() {
		t.Fatalf("empty Cf-Access assertion produced a human: %#v %v", principal, err)
	}
	// Owner bindings never infer capabilities from Cloudflare reachability: an
	// empty binding set grants no scopes even for a verified Cf-Access human.
	cfg := Config{Providers: []Provider{fakeProvider{source: identity.SourceCloudflareAccess, fn: func() (identity.Principal, error) {
		return identity.Principal{Kind: identity.ActorHuman, Subject: "cf-sub", Verified: true, Source: identity.SourceCloudflareAccess}, nil
	}}}}
	got, err := cfg.Authenticate(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Scopes) != 0 {
		t.Fatalf("Cf-Access reachability granted scopes without an owner binding: %+v", got.Scopes)
	}
}

// INV-11 (new) — Audience-mismatch rejection. A token minted for
// scenario-authenticator:default is refused where a resource audience is
// required, so a token for one app cannot be replayed at another.
func TestInvariantAudienceMismatchRejected(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	server := httptest.NewServer(jwksHandler(key))
	defer server.Close()
	// Verifier for a specific resource audience, with no accepted migration set.
	verifier := NewJWTVerifier(JWTConfig{
		Source: identity.SourceScenarioAuthenticator, Issuer: "scenario-authenticator",
		Audience: "scenario-authenticator:nutrition-planner", JWKSURL: server.URL, Now: func() time.Time { return now },
	})
	// Token minted for the default audience — a different application.
	token := signJWT(t, key, map[string]any{
		"sub": "owner-1", "iss": "scenario-authenticator",
		"aud": "scenario-authenticator:default", "exp": now.Add(time.Hour).Unix(),
	})
	_, err = verifier.Verify(context.Background(), token)
	failure, ok := identity.FailureFromError(err)
	if !ok || failure.Class != identity.FailureInvalid {
		t.Fatalf("mismatched audience not rejected as invalid: %v (%#v)", err, failure)
	}
	// The matching-audience token still verifies, proving the rejection is the
	// audience, not a broken verifier.
	match := signJWT(t, key, map[string]any{
		"sub": "owner-1", "iss": "scenario-authenticator",
		"aud": "scenario-authenticator:nutrition-planner", "exp": now.Add(time.Hour).Unix(),
	})
	if principal, err := verifier.Verify(context.Background(), match); err != nil || principal.Subject != "owner-1" {
		t.Fatalf("matching audience did not verify: %#v %v", principal, err)
	}
}

// INV-12 (new) — Scope-catalog wildcard matrix. Exact, scenario-wildcard,
// effect-wildcard, and full-wildcard held scopes accept/refuse required scopes
// correctly; malformed or whitespace-padded scopes never grant access. This
// pins scopecatalog.Resolve, the coarse capability evaluator used by
// RequireCapability and the agent-grant attenuation in Middleware.
func TestInvariantScopeCatalogWildcardMatrix(t *testing.T) {
	cases := []struct {
		name     string
		held     []string
		required string
		want     bool
	}{
		{"exact match", []string{"agent-manager:write"}, "agent-manager:write", true},
		{"exact miss effect", []string{"agent-manager:read"}, "agent-manager:write", false},
		{"exact miss scenario", []string{"git-control-tower:write"}, "agent-manager:write", false},
		{"scenario wildcard covers effect", []string{"agent-manager:*"}, "agent-manager:write", true},
		{"scenario wildcard scoped to scenario", []string{"agent-manager:*"}, "git-control-tower:write", false},
		{"effect wildcard covers scenario", []string{"*:write"}, "agent-manager:write", true},
		{"effect wildcard scoped to effect", []string{"*:write"}, "agent-manager:read", false},
		{"full wildcard covers all", []string{"*"}, "agent-manager:destructive", true},
		{"empty held grants nothing", []string{}, "agent-manager:read", false},
		{"malformed held ignored", []string{"agent-manager"}, "agent-manager:read", false},
		{"whitespace-padded held ignored", []string{" agent-manager:read "}, "agent-manager:read", false},
		{"whitespace-padded required refused", []string{"agent-manager:read"}, " agent-manager:read ", false},
		{"malformed required refused", []string{"*"}, "agent-manager", false},
		{"empty required refused", []string{"*"}, "", false},
		{"attenuation: narrower held does not cover wider required", []string{"agent-manager:read"}, "agent-manager:destructive", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := scopecatalog.Resolve(tc.held, tc.required); got != tc.want {
				t.Fatalf("Resolve(%v, %q) = %t, want %t", tc.held, tc.required, got, tc.want)
			}
		})
	}

	// The same matrix must hold through RequireCapability, the context-level
	// evaluator, so the catalog grammar and the request gate cannot diverge.
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Verified: true, Scopes: []string{"agent-manager:*"}})
	if _, err := RequireCapability(ctx, "agent-manager:destructive"); err != nil {
		t.Fatalf("scenario wildcard did not satisfy RequireCapability: %v", err)
	}
	if _, err := RequireCapability(ctx, "git-control-tower:read"); err == nil {
		t.Fatal("scenario wildcard leaked to another scenario via RequireCapability")
	}
}
