package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/subscription"
)

// [REQ:REQ-SUB-001] Subscription CRUD - create SSE subscription
func TestSubscriptionCreate_SSE(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{
		"name": "my-sub",
		"owner_scenario": "notification-hub",
		"event_pattern": "swarm-manager.backlog.*",
		"delivery_type": "sse",
		"enabled": true
	}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	result := decodeJSON[map[string]any](t, resp)
	if result["id"] == nil {
		t.Fatal("expected id in response")
	}
}

// [REQ:REQ-SUB-001] Subscription CRUD - create webhook subscription
func TestSubscriptionCreate_Webhook(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{
		"name": "webhook-sub",
		"owner_scenario": "analytics",
		"event_pattern": "**.completed.v1",
		"delivery_type": "webhook",
		"delivery_target": "https://example.com/webhook",
		"enabled": true
	}`
	resp, err := http.Post(ts.URL+"/api/v1/subscriptions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Validation rejects invalid subscriptions
func TestSubscriptionCreate_Validation(t *testing.T) {
	_, ts := newTestServer(t)

	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"owner_scenario":"o","event_pattern":"*","delivery_type":"sse"}`},
		{"missing owner", `{"name":"n","event_pattern":"*","delivery_type":"sse"}`},
		{"missing pattern", `{"name":"n","owner_scenario":"o","delivery_type":"sse"}`},
		{"missing delivery_type", `{"name":"n","owner_scenario":"o","event_pattern":"*"}`},
		{"invalid delivery_type", `{"name":"n","owner_scenario":"o","event_pattern":"*","delivery_type":"invalid"}`},
		{"webhook no target", `{"name":"n","owner_scenario":"o","event_pattern":"*","delivery_type":"webhook"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, _ := http.Post(ts.URL+"/api/v1/subscriptions", "application/json", strings.NewReader(tt.body))
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}

// [REQ:REQ-SUB-001] Subscription CRUD - list with filters
func TestSubscriptionList(t *testing.T) {
	_, ts := newTestServer(t)

	http.Post(ts.URL+"/api/v1/subscriptions", "application/json",
		strings.NewReader(`{"name":"sub-a","owner_scenario":"owner-1","event_pattern":"*.v1","delivery_type":"sse","enabled":true}`))
	http.Post(ts.URL+"/api/v1/subscriptions", "application/json",
		strings.NewReader(`{"name":"sub-b","owner_scenario":"owner-2","event_pattern":"*.v2","delivery_type":"sse","enabled":true}`))

	// List all
	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions")
	subs := decodeJSON[[]subscription.Subscription](t, resp)
	resp.Body.Close()
	if len(subs) != 2 {
		t.Fatalf("expected 2, got %d", len(subs))
	}

	// By owner
	resp, _ = http.Get(ts.URL + "/api/v1/subscriptions?owner=owner-1")
	subs = decodeJSON[[]subscription.Subscription](t, resp)
	resp.Body.Close()
	if len(subs) != 1 {
		t.Fatalf("expected 1 for owner-1, got %d", len(subs))
	}
}

// [REQ:REQ-SUB-001] Subscription CRUD - get single
func TestSubscriptionGet(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"my-sub","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	resp, _ := http.Get(fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	sub := decodeJSON[subscription.Subscription](t, resp)
	if sub.Name != "my-sub" {
		t.Fatalf("expected name=my-sub, got %s", sub.Name)
	}
}

// [REQ:REQ-SUB-001] Subscription CRUD - update
func TestSubscriptionUpdate(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"orig","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	updateBody := `{"name":"updated","owner_scenario":"o","event_pattern":"*.v2","delivery_type":"sse","enabled":false}`
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id), strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	resp2, _ := http.Get(fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id))
	sub := decodeJSON[subscription.Subscription](t, resp2)
	resp2.Body.Close()
	if sub.Name != "updated" {
		t.Fatalf("expected name=updated, got %s", sub.Name)
	}
}

// [REQ:REQ-SUB-001] Subscription CRUD - delete
func TestSubscriptionDelete(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"del","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id), nil)
	resp, _ := http.DefaultClient.Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	resp, _ = http.Get(fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id))
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Empty subscription list returns empty array
func TestSubscriptionList_Empty(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions")
	defer resp.Body.Close()

	body := decodeJSON[json.RawMessage](t, resp)
	if string(body) != "[]" {
		t.Fatalf("expected [], got %s", body)
	}
}

// [REQ:REQ-SUB-001] Subscription get nonexistent returns 404
func TestSubscriptionGet_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions/99999")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Subscription get with invalid ID returns 400
func TestSubscriptionGet_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions/notanumber")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeInvalidParam {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidParam, errBody["code"])
	}
}

// [REQ:REQ-SUB-001] Subscription update nonexistent returns 404
func TestSubscriptionUpdate_NotFound(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"name":"n","owner_scenario":"o","event_pattern":"*","delivery_type":"sse"}`
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/subscriptions/99999", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Subscription update with invalid body returns 400
func TestSubscriptionUpdate_InvalidBody(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"orig","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id), strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Subscription update with validation errors returns 400
func TestSubscriptionUpdate_ValidationError(t *testing.T) {
	_, ts := newTestServer(t)

	id := createTestSubscription(t, ts.URL, `{"name":"orig","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)

	// Missing name
	body := `{"owner_scenario":"o","event_pattern":"*","delivery_type":"sse"}`
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/v1/subscriptions/%d", ts.URL, id), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeValidation {
		t.Fatalf("expected %s, got %s", ErrCodeValidation, errBody["code"])
	}
}

// [REQ:REQ-SUB-001] Subscription update with invalid ID returns 400
func TestSubscriptionUpdate_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	body := `{"name":"n","owner_scenario":"o","event_pattern":"*","delivery_type":"sse"}`
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/subscriptions/abc", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Subscription delete with invalid ID returns 400
func TestSubscriptionDelete_InvalidID(t *testing.T) {
	_, ts := newTestServer(t)

	req, _ := http.NewRequest("DELETE", ts.URL+"/api/v1/subscriptions/notanumber", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// [REQ:REQ-SUB-001] Subscription create with invalid JSON returns 400
func TestSubscriptionCreate_InvalidJSON(t *testing.T) {
	_, ts := newTestServer(t)

	resp, _ := http.Post(ts.URL+"/api/v1/subscriptions", "application/json", strings.NewReader("not json"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	errBody := decodeJSON[map[string]string](t, resp)
	if errBody["code"] != ErrCodeInvalidBody {
		t.Fatalf("expected %s, got %s", ErrCodeInvalidBody, errBody["code"])
	}
}

// [REQ:REQ-SUB-001] Subscription list filters by enabled status
func TestSubscriptionList_EnabledFilter(t *testing.T) {
	_, ts := newTestServer(t)

	createTestSubscription(t, ts.URL, `{"name":"sub-on","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":true}`)
	createTestSubscription(t, ts.URL, `{"name":"sub-off","owner_scenario":"o","event_pattern":"*","delivery_type":"sse","enabled":false}`)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions?enabled=true")
	subs := decodeJSON[[]subscription.Subscription](t, resp)
	resp.Body.Close()
	if len(subs) != 1 {
		t.Fatalf("expected 1 enabled subscription, got %d", len(subs))
	}
	if subs[0].Name != "sub-on" {
		t.Fatalf("expected sub-on, got %s", subs[0].Name)
	}
}

// [REQ:REQ-SUB-001] Subscription list filters by exact event pattern
func TestSubscriptionList_PatternFilter(t *testing.T) {
	_, ts := newTestServer(t)

	createTestSubscription(t, ts.URL, `{"name":"sub-a","owner_scenario":"o","event_pattern":"app.events.*","delivery_type":"sse","enabled":true}`)
	createTestSubscription(t, ts.URL, `{"name":"sub-b","owner_scenario":"o","event_pattern":"sys.health.*","delivery_type":"sse","enabled":true}`)

	resp, _ := http.Get(ts.URL + "/api/v1/subscriptions?pattern=app.events.*")
	subs := decodeJSON[[]subscription.Subscription](t, resp)
	resp.Body.Close()
	if len(subs) != 1 {
		t.Fatalf("expected 1 subscription matching exact pattern, got %d", len(subs))
	}
	if subs[0].Name != "sub-a" {
		t.Fatalf("expected sub-a, got %s", subs[0].Name)
	}
}
