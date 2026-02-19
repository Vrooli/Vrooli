package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:CFAPI-003] Push route manifest to tunnel config tests
// [REQ:CFAPI-004] Hot-reload verification tests

func TestCFClient_PushConfig(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	client, _ := NewCFClient(WithCFBaseURL(srv.URL))

	rules := []CFIngressRule{
		{Hostname: "app.example.com", Service: "http://localhost:8080"},
	}

	err := client.PushConfig(context.Background(), rules)
	if err != nil {
		t.Fatalf("PushConfig: %v", err)
	}

	// Verify catch-all was added
	cfg, ok := received["config"].(map[string]any)
	if !ok {
		t.Fatal("missing config in payload")
	}
	ingress, ok := cfg["ingress"].([]any)
	if !ok {
		t.Fatal("missing ingress in config")
	}
	if len(ingress) != 2 {
		t.Errorf("expected 2 rules (1 + catch-all), got %d", len(ingress))
	}
}

func TestCFClient_PushConfigPreservesCatchAll(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	client, _ := NewCFClient(WithCFBaseURL(srv.URL))

	// Rules already have a catch-all
	rules := []CFIngressRule{
		{Hostname: "app.example.com", Service: "http://localhost:8080"},
		{Service: "http_status:404"},
	}

	if err := client.PushConfig(context.Background(), rules); err != nil {
		t.Fatalf("PushConfig: %v", err)
	}

	cfg := received["config"].(map[string]any)
	ingress := cfg["ingress"].([]any)
	// Should NOT add duplicate catch-all
	if len(ingress) != 2 {
		t.Errorf("expected 2 rules (no duplicate catch-all), got %d", len(ingress))
	}
}

func TestCFClient_PushConfigAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"message":"forbidden"}]}`))
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "bad-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	client, _ := NewCFClient(WithCFBaseURL(srv.URL))
	err := client.PushConfig(context.Background(), []CFIngressRule{})
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
