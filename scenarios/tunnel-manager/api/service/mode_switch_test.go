package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/adapter"
	"tunnel-manager/domain"
)

// [REQ:CFAPI-006] Mode switching tests

func TestModeSwitcher_SwitchToRemote(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "test-app", ScenarioName: "test-scenario", LocalPort: 8080, Enabled: true, PublicURL: "https://test-app.example.com"},
			}, nil
		},
	}

	// Mock CF API
	var pushedConfig map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			_ = json.NewDecoder(r.Body).Decode(&pushedConfig)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	cfClient, err := adapter.NewCFClient(adapter.WithCFBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewCFClient: %v", err)
	}

	switcher := NewModeSwitcher(nil, cfClient, lister)
	if err := switcher.SwitchTo(context.Background(), domain.ModeRemote); err != nil {
		t.Fatalf("SwitchTo remote: %v", err)
	}

	if pushedConfig == nil {
		t.Fatal("config was not pushed to CF API")
	}
}

func TestModeSwitcher_SwitchToRemoteRequiresClient(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return nil, nil
		},
	}

	switcher := NewModeSwitcher(nil, nil, lister)
	err := switcher.SwitchTo(context.Background(), domain.ModeRemote)
	if err == nil {
		t.Fatal("expected error without CF client")
	}
}

func TestModeSwitcher_InvalidMode(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return nil, nil
		},
	}

	switcher := NewModeSwitcher(nil, nil, lister)
	err := switcher.SwitchTo(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

// [REQ:CFAPI-003] Push route manifest to tunnel config tests
// [REQ:CFAPI-004] Hot-reload verification tests
// (Originally in cf_config_sync_test.go — tests CF push behavior via mode switcher)

func TestModeSwitcher_PushConfigViaSwitchToRemote(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "app", LocalPort: 8080, Enabled: true, PublicURL: "https://app.example.com"},
			}, nil
		},
	}

	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			_ = json.NewDecoder(r.Body).Decode(&received)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	cfClient, _ := adapter.NewCFClient(adapter.WithCFBaseURL(srv.URL))
	switcher := NewModeSwitcher(nil, cfClient, lister)

	if err := switcher.SwitchTo(context.Background(), domain.ModeRemote); err != nil {
		t.Fatalf("SwitchTo remote: %v", err)
	}

	// Verify config was pushed with ingress rules
	cfg, ok := received["config"].(map[string]any)
	if !ok {
		t.Fatal("missing config in payload")
	}
	ingress, ok := cfg["ingress"].([]any)
	if !ok {
		t.Fatal("missing ingress in config")
	}
	if len(ingress) != 2 { // 1 route + catch-all
		t.Errorf("expected 2 rules (1 + catch-all), got %d", len(ingress))
	}
}

func TestModeSwitcher_PushConfigAPIError(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"message":"forbidden"}]}`))
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "bad-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	cfClient, _ := adapter.NewCFClient(adapter.WithCFBaseURL(srv.URL))
	switcher := NewModeSwitcher(nil, cfClient, lister)

	err := switcher.SwitchTo(context.Background(), domain.ModeRemote)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
}
