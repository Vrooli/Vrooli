package main

import (
	"testing"
)

// [REQ:LOCAL-002] Generate config from route manifest tests

func TestLocalConfig_GenerateFromRoutes(t *testing.T) {
	mgr := NewLocalConfigManager()

	routes := []Route{
		{Subdomain: "app-a", ScenarioName: "scenario-a", LocalPort: 8080, Enabled: true},
		{Subdomain: "app-b", ScenarioName: "scenario-b", LocalPort: 9090, Enabled: true, PublicURL: "https://custom.example.com"},
		{Subdomain: "disabled", ScenarioName: "scenario-c", LocalPort: 7070, Enabled: false},
	}

	cfg := mgr.GenerateFromRoutes(routes, nil)

	// Should have 2 enabled + 1 catch-all = 3 rules
	if len(cfg.Ingress) != 3 {
		t.Fatalf("expected 3 ingress rules, got %d", len(cfg.Ingress))
	}

	// First rule uses default hostname
	if cfg.Ingress[0].Hostname != "app-a.vrooli.com" {
		t.Errorf("rule[0] hostname: want app-a.vrooli.com, got %s", cfg.Ingress[0].Hostname)
	}
	if cfg.Ingress[0].Service != "http://localhost:8080" {
		t.Errorf("rule[0] service: got %s", cfg.Ingress[0].Service)
	}

	// Second rule uses custom hostname from PublicURL
	if cfg.Ingress[1].Hostname != "custom.example.com" {
		t.Errorf("rule[1] hostname: want custom.example.com, got %s", cfg.Ingress[1].Hostname)
	}

	// Last rule is catch-all
	if cfg.Ingress[2].Service != "http_status:404" {
		t.Errorf("catch-all service: got %s", cfg.Ingress[2].Service)
	}
}

func TestLocalConfig_GeneratePreservesSettings(t *testing.T) {
	mgr := NewLocalConfigManager()

	existing := &CloudflaredConfig{
		Tunnel:          "my-tunnel",
		CredentialsFile: "/path/to/creds.json",
		WarpRouting:     map[string]any{"enabled": true},
	}

	cfg := mgr.GenerateFromRoutes([]Route{}, existing)

	if cfg.Tunnel != "my-tunnel" {
		t.Errorf("tunnel not preserved: got %s", cfg.Tunnel)
	}
	if cfg.CredentialsFile != "/path/to/creds.json" {
		t.Errorf("credentials-file not preserved: got %s", cfg.CredentialsFile)
	}
	if cfg.WarpRouting == nil || cfg.WarpRouting["enabled"] != true {
		t.Error("warp-routing not preserved")
	}
	// Should still have catch-all
	if len(cfg.Ingress) != 1 || cfg.Ingress[0].Service != "http_status:404" {
		t.Error("expected catch-all rule")
	}
}
