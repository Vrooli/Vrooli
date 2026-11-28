package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/operatordecision"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/attempt"
)

func TestProposalDecisionRailMountsPOSTOnlySignedWebhook(t *testing.T) {
	const secret = "test-events-webhook-secret"
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", secret)
	t.Setenv("API_PORT", "") // Avoid launching the background subscription reconciler.
	resolved, err := resolveEventsWebhookSecret()
	if err != nil || resolved != secret {
		t.Fatalf("resolveEventsWebhookSecret() = %q, %v", resolved, err)
	}
	server := &Server{
		router:           mux.NewRouter(),
		agentSessionSvc:  &agentsessions.Service{},
		attemptDecisions: attempt.NewRouter(),
	}
	server.wireProposalDecisionRail()

	body := `{"event_id":"e1","event_type":"unrelated.event","source_scenario":"notification-hub","payload":{}}`
	unsigned := httptest.NewRequest(http.MethodPost, decisionEventWebhookPath, strings.NewReader(body))
	unsignedResponse := httptest.NewRecorder()
	server.router.ServeHTTP(unsignedResponse, unsigned)
	if unsignedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned webhook status = %d, want 401", unsignedResponse.Code)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	signed := httptest.NewRequest(http.MethodPost, decisionEventWebhookPath, strings.NewReader(body))
	signed.Header.Set(operatordecision.SignatureHeader, hex.EncodeToString(mac.Sum(nil)))
	signedResponse := httptest.NewRecorder()
	server.router.ServeHTTP(signedResponse, signed)
	if signedResponse.Code != http.StatusOK || !strings.Contains(signedResponse.Body.String(), `"ignored":true`) {
		t.Fatalf("signed unknown event response = %d %s", signedResponse.Code, signedResponse.Body.String())
	}

	get := httptest.NewRequest(http.MethodGet, decisionEventWebhookPath, nil)
	var match mux.RouteMatch
	if server.router.Match(get, &match) || match.MatchErr != mux.ErrMethodMismatch {
		t.Fatalf("GET route match = %v, match error = %v; want method mismatch", server.router.Match(get, &match), match.MatchErr)
	}
}
