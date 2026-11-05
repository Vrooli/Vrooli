package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	apiHandlers "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/handlers"
)

// webhookRouteMounted reports whether a POST to the decision-resolution webhook
// path matches a route on the given router. The mount is the observable effect:
// when a signing secret resolves, the route exists; when none does, it does not.
func webhookRouteMounted(router *mux.Router) bool {
	req := httptest.NewRequest(http.MethodPost, apiHandlers.DecisionEventWebhookPath, nil)
	var match mux.RouteMatch
	return router.Match(req, &match)
}

// TestWebhookMountsWhenSecretResolvesViaFallback proves the hardening: with no
// VROOLI_EVENTS_WEBHOOK_SECRET env var set, the webhook still mounts because the
// secret resolves through the credential-authority fallback (the same value
// vrooli-events signs with). This is the regression the env-only gate had: an
// unset env var left operator approvals silently no-op.
func TestWebhookMountsWhenSecretResolvesViaFallback(t *testing.T) {
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", "")
	secret, err := resolveEventsWebhookSecret()
	if err != nil || secret == "" {
		t.Skipf("credential authority unavailable in this environment (err=%v); fallback path not exercisable here", err)
	}

	router := mux.NewRouter()
	// The mount function never dereferences the handler during registration, so a
	// nil handler is sufficient to observe the route being added.
	var h *apiHandlers.Handlers
	mountDecisionResolutionWebhook(router, h)

	if !webhookRouteMounted(router) {
		t.Fatalf("expected the decision resolution webhook to be mounted when the secret resolves via the fallback")
	}
}

// TestWebhookMountsWhenSecretFromEnv proves the env override still works and
// mounts the webhook, independent of the credential authority.
func TestWebhookMountsWhenSecretFromEnv(t *testing.T) {
	t.Setenv("VROOLI_EVENTS_WEBHOOK_SECRET", "test-secret-value")

	secret, err := resolveEventsWebhookSecret()
	if err != nil {
		t.Fatalf("resolveEventsWebhookSecret() error = %v", err)
	}
	if secret != "test-secret-value" {
		t.Fatalf("env override not honoured: secret = %q", secret)
	}

	router := mux.NewRouter()
	var h *apiHandlers.Handlers
	mountDecisionResolutionWebhook(router, h)

	if !webhookRouteMounted(router) {
		t.Fatalf("expected the decision resolution webhook to be mounted when the secret is set via env")
	}
}
