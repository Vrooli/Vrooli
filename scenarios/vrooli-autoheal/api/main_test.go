package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/internal/recovery"
	apiHandlers "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/handlers"
)

func TestUnavailableCodingAgentLedgerDoesNotRemoveMechanicalRecoveryRequester(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	path, err := recovery.DefaultCodingAgentRecordPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid ledger fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	broker, initErr := newCodingAgentRecoveryBroker()
	if initErr == nil || broker != nil {
		t.Fatal("corrupt fixture unexpectedly initialized a broker")
	}
	requester := codingAgentRecoveryRequester(broker, initErr)
	if requester == nil {
		t.Fatal("startup removed the explicit escalation failure path")
	}
	if id, err := requester(context.Background(), "web-console", "fixture"); err == nil || id.RequestID != "" || !strings.Contains(err.Error(), "coding-agent recovery unavailable") {
		t.Fatalf("unavailable escalation fabricated admission: admission=%+v err=%v", id, err)
	}
}

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
	mountDecisionResolutionWebhook(context.Background(), router, h)

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
	mountDecisionResolutionWebhook(context.Background(), router, h)

	if !webhookRouteMounted(router) {
		t.Fatalf("expected the decision resolution webhook to be mounted when the secret is set via env")
	}
}

func TestDecisionSubscriptionResolvesCurrentEventsAddress(t *testing.T) {
	var firstCalls, secondCalls atomic.Int32
	makeServer := func(calls *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/subscriptions":
				if got := r.URL.Query().Get("owner"); got != "vrooli-autoheal" {
					t.Errorf("subscription owner = %q", got)
				}
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("[]"))
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/subscriptions":
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode subscription: %v", err)
				}
				if body["event_pattern"] != "notification_hub.ask.resolved.v1" {
					t.Errorf("event pattern = %v", body["event_pattern"])
				}
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":1}`))
			default:
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				http.NotFound(w, r)
			}
		}))
	}
	first, second := makeServer(&firstCalls), makeServer(&secondCalls)
	defer first.Close()
	defer second.Close()
	addresses := []string{first.URL, second.URL}
	var resolves atomic.Int32
	resolve := func(context.Context) (string, error) { return addresses[int(resolves.Add(1))-1], nil }
	for range 2 {
		if err := reconcileDecisionEventSubscriptionWithResolver(context.Background(), "http://127.0.0.1:1234/hook", resolve); err != nil {
			t.Fatalf("reconcileDecisionEventSubscriptionWithResolver() error = %v", err)
		}
	}
	if resolves.Load() != 2 || firstCalls.Load() != 2 || secondCalls.Load() != 2 {
		t.Fatalf("resolutions/first requests/second requests = %d/%d/%d, want 2/2/2", resolves.Load(), firstCalls.Load(), secondCalls.Load())
	}
}

func TestDecisionSubscriptionRetriesUnavailableEventsPeerWithinBound(t *testing.T) {
	var calls, waits, peerCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peerCalls.Add(1)
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()
	resolve := func(context.Context) (string, error) {
		if calls.Add(1) < 3 {
			return "", errors.New("vrooli-events is not ready")
		}
		return server.URL, nil
	}
	wait := func(ctx context.Context, _ time.Duration) error {
		waits.Add(1)
		return ctx.Err()
	}
	if err := reconcileDecisionEventSubscriptionWithRetry(context.Background(), "http://127.0.0.1:1234/hook", resolve, wait); err != nil {
		t.Fatalf("reconcileDecisionEventSubscriptionWithRetry() error = %v", err)
	}
	if calls.Load() != 3 || waits.Load() != 2 || peerCalls.Load() != 2 {
		t.Fatalf("resolve/wait/peer calls = %d/%d/%d, want 3/2/2", calls.Load(), waits.Load(), peerCalls.Load())
	}
}
