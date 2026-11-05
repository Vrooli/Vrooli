package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/attempt"
	"swarm-manager/internal/backlog"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/operatordecision"
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

func TestDecisionWebhookPassesInteractiveGrantOnlyBehindHMAC(t *testing.T) {
	const secret = "test-webhook-secret"
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", secret)
	t.Setenv("API_PORT", "")
	s := newWiringTestServer(t)
	s.router.Use(backlog.InteractiveGrantMiddleware)
	s.wireProposalDecisionRail()

	for _, tc := range []struct {
		name, signature string
		wantStatus      int
		wantBody        string
	}{
		{name: "missing signature", wantStatus: http.StatusUnauthorized, wantBody: "event webhook signature is invalid or unavailable"},
		{name: "bad signature", signature: "deadbeef", wantStatus: http.StatusUnauthorized, wantBody: "event webhook signature is invalid or unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, decisionEventWebhookPath, strings.NewReader(`{}`))
			if tc.signature != "" {
				request.Header.Set(operatordecision.SignatureHeader, tc.signature)
			}
			response := httptest.NewRecorder()
			s.router.ServeHTTP(response, request)
			if response.Code != tc.wantStatus || !strings.Contains(response.Body.String(), tc.wantBody) {
				t.Fatalf("status=%d body=%q; want %d containing %q", response.Code, response.Body.String(), tc.wantStatus, tc.wantBody)
			}
		})
	}

	// Non-POST methods on the exact webhook path remain outside the narrowly
	// allowed signed delivery route and are denied by the global middleware.
	wrongMethod := httptest.NewRequest(http.MethodPatch, decisionEventWebhookPath, strings.NewReader(`{}`))
	wrongMethodResponse := httptest.NewRecorder()
	backlog.InteractiveGrantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("non-POST webhook unexpectedly passed the global middleware")
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(wrongMethodResponse, wrongMethod)
	if wrongMethodResponse.Code != http.StatusUnauthorized {
		t.Fatalf("non-POST webhook status=%d body=%q; want middleware denial", wrongMethodResponse.Code, wrongMethodResponse.Body.String())
	}

	// A correctly signed but unrelated resolution proves the exact webhook path
	// traverses the global middleware and reaches the canonical handler. It is a
	// no-op: the registered proposal effect does not claim this event type.
	body := []byte(`{"event_id":"fixture-event","event_type":"notification_hub.ask.resolved.v1","source_scenario":"notification-hub","payload":{"ask_id":"fixture-ask","state":"answered","answer":"approve","source_event_type":"unrelated.decision_requested.v1","correlation":{}}}`)
	request := httptest.NewRequest(http.MethodPost, decisionEventWebhookPath, strings.NewReader(string(body)))
	request.Header.Set(operatordecision.SignatureHeader, signDecisionWebhook(body, secret))
	response := httptest.NewRecorder()
	s.router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"ignored":true`) {
		t.Fatalf("signed benign resolution status=%d body=%q; want canonical no-op response", response.Code, response.Body.String())
	}
}

func signDecisionWebhook(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
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
