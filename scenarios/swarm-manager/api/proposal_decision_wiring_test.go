package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/attempt"

	"github.com/gorilla/mux"
)

// newWiringTestServer builds the minimal Server the decision-rail wiring needs: a
// router, a real agent-session service, and a registered attempt router.
func newWiringTestServer(t *testing.T) *Server {
	t.Helper()
	svc, err := agentsessions.NewService(agentsessions.ServiceConfig{
		Store:       agentsessions.NewFileStore(t.TempDir()),
		ProjectRoot: t.TempDir(),
		ProfileKey:  "swarm-manager/default",
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	router := attempt.NewRouter()
	if err := router.Register(agentsessions.AttemptSubjectProposal, agentsessions.NewAttemptDecider(svc)); err != nil {
		t.Fatalf("register attempt decider: %v", err)
	}
	return &Server{
		router:           mux.NewRouter(),
		agentSessionSvc:  svc,
		attemptDecisions: router,
	}
}

// decisionWebhookMounted reports whether a POST to the webhook path matches a
// registered route on the server's router.
func decisionWebhookMounted(s *Server) bool {
	req := httptest.NewRequest(http.MethodPost, decisionEventWebhookPath, nil)
	var match mux.RouteMatch
	return s.router.Match(req, &match) && match.MatchErr == nil
}

// With the webhook secret resolvable via the env fallback, the resolution webhook
// mounts — operator answers can round-trip. This is the regression the H2
// hardening protects: before the credential fallback, an unset env left the
// webhook unmounted and approvals silently no-oped.
func TestProposalDecisionWebhookMountsWhenSecretResolves(t *testing.T) {
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", "test-webhook-secret")
	// No API_PORT: skip the background subscription reconcile (it would try to
	// reach vrooli-events). The mount itself does not depend on it.
	t.Setenv("API_PORT", "")

	s := newWiringTestServer(t)
	if decisionWebhookMounted(s) {
		t.Fatal("webhook mounted before wireProposalDecisionRail ran")
	}
	s.wireProposalDecisionRail()
	if !decisionWebhookMounted(s) {
		t.Fatal("decision webhook did not mount even though the secret resolved")
	}
}

// resolveEventsWebhookSecret honors the env override verbatim (the agent-manager
// parity path: the same value vrooli-events signs with).
func TestResolveEventsWebhookSecretPrefersEnv(t *testing.T) {
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", "env-secret-wins")
	secret, err := resolveEventsWebhookSecret()
	if err != nil {
		t.Fatalf("resolveEventsWebhookSecret: %v", err)
	}
	if secret != "env-secret-wins" {
		t.Fatalf("secret = %q, want env override", secret)
	}
}
