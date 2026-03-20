package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
)

// [REQ:CFAPI-001] API token management tests
// [REQ:CFAPI-002] Read current tunnel configuration tests
// [REQ:CFAPI-005] Tunnel status query tests

func TestCFClient_RequiresToken(t *testing.T) {
	t.Setenv("CF_API_TOKEN", "")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	_, err := NewCFClient()
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
}

func TestCFClient_RequiresAccountID(t *testing.T) {
	t.Setenv("CF_API_TOKEN", "tok123")
	t.Setenv("CF_ACCOUNT_ID", "")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	_, err := NewCFClient()
	if err == nil {
		t.Fatal("expected error when account ID is missing")
	}
}

func TestCFClient_RequiresTunnelID(t *testing.T) {
	t.Setenv("CF_API_TOKEN", "tok123")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "")

	_, err := NewCFClient()
	if err == nil {
		t.Fatal("expected error when tunnel ID is missing")
	}
}

func TestCFClient_ReadConfig(t *testing.T) {
	// Mock CF API server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing or wrong auth header: %s", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"config": map[string]any{
					"ingress": []map[string]string{
						{"hostname": "app.example.com", "service": "http://localhost:8080"},
						{"service": "http_status:404"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	client, err := NewCFClient(WithCFBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewCFClient: %v", err)
	}

	cfg, err := client.ReadConfig(context.Background())
	if err != nil {
		t.Fatalf("ReadConfig: %v", err)
	}

	if len(cfg.Config.Ingress) != 2 {
		t.Errorf("expected 2 ingress rules, got %d", len(cfg.Config.Ingress))
	}
}

func TestCFClient_GetTunnelStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id":     "tun456",
				"name":   "my-tunnel",
				"status": "healthy",
				"connections": []map[string]any{
					{"colo_name": "DFW", "is_alive": true},
				},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	client, _ := NewCFClient(WithCFBaseURL(srv.URL))
	status, err := client.GetTunnelStatus(context.Background())
	if err != nil {
		t.Fatalf("GetTunnelStatus: %v", err)
	}
	if status.Name != "my-tunnel" {
		t.Errorf("name: want my-tunnel, got %s", status.Name)
	}
	if status.Status != "healthy" {
		t.Errorf("status: want healthy, got %s", status.Status)
	}
	if len(status.Connections) != 1 {
		t.Errorf("connections: want 1, got %d", len(status.Connections))
	}
}

func TestRoutesToCFRules(t *testing.T) {
	routes := []domain.Route{
		{Subdomain: "app", LocalPort: 8080, Enabled: true, PublicURL: "https://app.example.com"},
		{Subdomain: "disabled", LocalPort: 9090, Enabled: false},
	}

	rules := RoutesToCFRules(routes)
	if len(rules) != 2 { // 1 enabled + catch-all
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Hostname != "app.example.com" {
		t.Errorf("rule[0] hostname: got %s", rules[0].Hostname)
	}
	if rules[1].Service != "http_status:404" {
		t.Errorf("catch-all: got %s", rules[1].Service)
	}
}

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
