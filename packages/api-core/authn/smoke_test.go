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
)

// TestCrossScenarioAuthSmoke is the cross-scenario auth smoke gate: it exercises
// the full issue -> verify -> decide chain across the scenario-authenticator
// issuance surface and one relying scenario, using the REAL shared provider and
// the REAL standing-grant decision — no fakes on the hot path. It is the
// blast-radius regression net for any change under packages/api-core/**: if a
// shared auth change breaks token issuance, cross-scenario verification, or the
// one authority decision, this single test fails.
//
// Run it as a single command:
//
//	cd packages/api-core && go test ./authn/ -run TestCrossScenarioAuthSmoke -v
//
// It is self-contained (an in-process JWKS server stands in for the
// authenticator's /.well-known/jwks.json) so it needs no running services and
// can gate a PR or a pre-merge hook.
func TestCrossScenarioAuthSmoke(t *testing.T) {
	// --- scenario-authenticator: issuance ---------------------------------
	// The authenticator signs tokens with its RS256 key and publishes the
	// public half at a JWKS endpoint. We model exactly that.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_700_000_000, 0).UTC()
	jwks := httptest.NewServer(jwksHandler(key))
	defer jwks.Close()

	const issuer = "scenario-authenticator"
	const relyingScenario = "agent-manager"
	relyingAudience := "scenario-authenticator:" + relyingScenario

	mint := func(claims map[string]any) string {
		base := map[string]any{"iss": issuer, "aud": relyingAudience, "exp": now.Add(time.Hour).Unix()}
		for k, v := range claims {
			base[k] = v
		}
		return signJWT(t, key, base)
	}

	// --- relying scenario (agent-manager): verify -------------------------
	// The relying scenario builds its provider chain for its own audience and
	// verifies presented tokens through the shared JWTVerifier — the same path
	// production uses via NewScenarioAuthenticatorProvider.
	verifier := NewJWTVerifier(JWTConfig{
		Source: identity.SourceScenarioAuthenticator, Issuer: issuer,
		Audience: relyingAudience, JWKSURL: jwks.URL, Now: func() time.Time { return now },
	})

	// 1. A human token issued by the authenticator verifies at the relying
	//    scenario as a verified human with its issued scopes.
	humanToken := mint(map[string]any{"user_id": "operator", "actor_kind": "human", "scope": []string{"agent-manager:read", "agent-manager:write"}})
	human, err := verifier.Verify(context.Background(), humanToken)
	if err != nil || !human.IsHuman() || human.Subject != "operator" {
		t.Fatalf("issue->verify broke for a human token: %#v %v", human, err)
	}

	// 2. A service-principal token issued by the authenticator verifies as a
	//    service (never a human) at the relying scenario.
	serviceToken := mint(map[string]any{"user_id": "service:prompt-manager:heartbeat", "actor_kind": "service", "scope": []string{"agent-manager:write"}})
	service, err := verifier.Verify(context.Background(), serviceToken)
	if err != nil || service.Kind != identity.ActorService || service.IsHuman() {
		t.Fatalf("issue->verify broke for a service token: %#v %v", service, err)
	}

	// --- relying scenario: decide -----------------------------------------
	// The verified service presents identity only; the relying scenario sources
	// the operator-approved standing grant and evaluates it through the shared
	// RequestOperationAuthority ingress. This is the one authority decision.
	// The standing-grant evaluator (standingAuthority) uses the real wall clock,
	// so the grant must expire in the real future, not in the fixed issuance
	// clock's past.
	grant := &identity.OperationGrant{
		ID: "standing", Approver: "operator", Policy: "run-creation",
		Operations: []string{"run.create"}, Targets: []string{relyingScenario + ":*"},
		ExpiresAt: time.Now().Add(time.Hour),
	}
	auth := standingAuthority{grant: grant, limits: StandingGrantLimits{MaxConcurrentRuns: 3, MaxRunBudgetUSD: 100}, usage: StandingGrantUsage{ConcurrentRuns: 1, CommittedBudgetUSD: 10, RequestBudgetUSD: 5}}
	ctx := WithRequestOperationAuthority(identity.WithPrincipal(context.Background(), service), auth)

	// Decide: a covered costly write is admitted for the verified service.
	admitted, err := RequireOperation(ctx, "agent-manager:write", "run.create", relyingScenario+":run/new")
	if err != nil || admitted.Subject != "service:prompt-manager:heartbeat" {
		t.Fatalf("issue->verify->decide broke on the admit path: %#v %v", admitted, err)
	}

	// Decide: an operation the grant does not cover is refused — the decision is
	// not a rubber stamp.
	if _, err := RequireOperation(ctx, "agent-manager:write", "agent.manage", relyingScenario+":run/new"); err == nil {
		t.Fatal("issue->verify->decide admitted an uncovered operation")
	}

	// --- negative blast-radius checks -------------------------------------
	// An anonymous request (no bearer) presenting the relying scenario audience
	// is refused at verify.
	if _, err := verifier.VerifyRequest(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil)); err == nil {
		t.Fatal("anonymous request verified")
	}
	// A token for the wrong audience (another app) is refused — a shared-package
	// audience regression would surface here.
	wrongAud := signJWT(t, key, map[string]any{"iss": issuer, "aud": "scenario-authenticator:some-other-app", "exp": now.Add(time.Hour).Unix(), "sub": "operator"})
	if _, err := verifier.Verify(context.Background(), wrongAud); err == nil {
		t.Fatal("a token for another scenario's audience verified at the relying scenario")
	}
}
