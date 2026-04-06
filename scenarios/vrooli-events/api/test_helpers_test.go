package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/config"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/policy"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/testutil"
)

// newTestServer creates a Server backed by a real in-memory SQLite store and broker.
// Use this for integration-style handler tests that need the full pipeline.
func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	eventStore := testutil.NewTestStore(t)

	eventBroker := broker.NewBroker(eventStore, broker.BrokerConfig{})
	t.Cleanup(func() { eventBroker.Close() })

	polStore, err := policy.NewSQLiteStore(eventStore.DB())
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}

	subStore, err := subscription.NewSQLiteStore(eventStore.DB())
	if err != nil {
		t.Fatalf("new subscription store: %v", err)
	}

	srv := &Server{
		store:             eventStore,
		broker:            eventBroker,
		policyStore:       polStore,
		policyEval:        policy.NewEvaluator(polStore),
		subStore:          subStore,
		policyBroadcaster: policy.NewPolicyBroadcaster(),
		webhookDeliverer:  subscription.NewWebhookDeliverer(),
		config: config.Config{
			MaxBodyBytes: config.DefaultMaxBodyBytes,
			SSERetryMs:   config.DefaultSSERetryMs,
			ReplayLimit:  config.DefaultReplayLimit,
		},
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)

	return srv, ts
}

// newMockedServer creates a Server backed by MockStore and MockBroker.
// Use this for unit-style handler tests where you want to verify handler
// logic in isolation without SQLite or real goroutines.
func newMockedServer(t *testing.T, ms *testutil.MockStore, mb *testutil.MockBroker) *httptest.Server {
	t.Helper()

	// For mocked servers, create real in-memory stores since policy/subscription
	// tests use the integration server instead.
	polStore := testutil.NewTestPolicyStore(t)
	subSt := testutil.NewTestSubscriptionStore(t)

	srv := &Server{
		store:             ms,
		broker:            mb,
		policyStore:       polStore,
		policyEval:        policy.NewEvaluator(polStore),
		subStore:          subSt,
		policyBroadcaster: policy.NewPolicyBroadcaster(),
		webhookDeliverer:  subscription.NewWebhookDeliverer(),
		config: config.Config{
			MaxBodyBytes: config.DefaultMaxBodyBytes,
			SSERetryMs:   config.DefaultSSERetryMs,
			ReplayLimit:  config.DefaultReplayLimit,
		},
	}
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	return ts
}

// itoa converts an int64 to its decimal string representation.
func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}

// decodeJSON reads and JSON-decodes the response body into a value of type T.
// Reduces the repeated json.NewDecoder(resp.Body).Decode(&v) boilerplate across tests.
func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

// createTestPolicy creates a policy rule via the API and returns its ID.
// Reduces boilerplate in tests that need a pre-existing rule for get/update/delete operations.
func createTestPolicy(t *testing.T, tsURL, body string) int64 {
	t.Helper()
	resp, err := http.Post(tsURL+"/api/v1/policies", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create policy: expected 201, got %d", resp.StatusCode)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return int64(result["id"].(float64))
}

// postOverride sends a circuit breaker override request and returns the response.
func postOverride(t *testing.T, tsURL string, policyID int64, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(fmt.Sprintf("%s/api/v1/policies/%d/override", tsURL, policyID), "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("override request: %v", err)
	}
	return resp
}

// doJSONRequest sends an HTTP request with a JSON body and returns the response.
// Consolidates the repeated NewRequest + Content-Type header + Do pattern
// used across PUT, DELETE, and other non-GET/POST test requests.
func doJSONRequest(t *testing.T, method, url, body string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

// createTestSubscription creates a subscription via the API and returns its ID.
func createTestSubscription(t *testing.T, tsURL, body string) int64 {
	t.Helper()
	resp, err := http.Post(tsURL+"/api/v1/subscriptions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create subscription: expected 201, got %d", resp.StatusCode)
	}
	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return int64(result["id"].(float64))
}
