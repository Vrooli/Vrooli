package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:ROUTE-003] Seed routes from cloudflared config on first run
func TestSeedFromCloudflaredConfig(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)

	// Create temp cloudflared config
	configYAML := `tunnel: test-tunnel-id
credentials-file: /home/user/.cloudflared/test-tunnel-id.json
ingress:
  - hostname: agent-manager.itsagitime.com
    service: http://localhost:36238
  - hostname: web-console.itsagitime.com
    service: http://localhost:36240
  - service: http_status:404
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	seeded, err := svc.SeedFromConfig(configPath)
	if err != nil {
		t.Fatalf("SeedFromConfig: %v", err)
	}
	if seeded != 2 {
		t.Errorf("seeded = %d, want 2", seeded)
	}

	// Verify routes were created
	routes, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	// Check specific route properties
	found := map[string]bool{}
	for _, r := range routes {
		found[r.Subdomain] = true
		switch r.Subdomain {
		case "agent-manager":
			if r.LocalPort != 36238 {
				t.Errorf("agent-manager port = %d, want 36238", r.LocalPort)
			}
			if r.PublicURL != "https://agent-manager.itsagitime.com" {
				t.Errorf("agent-manager public_url = %q", r.PublicURL)
			}
		case "web-console":
			if r.LocalPort != 36240 {
				t.Errorf("web-console port = %d, want 36240", r.LocalPort)
			}
		}
	}
	if !found["agent-manager"] || !found["web-console"] {
		t.Error("expected both agent-manager and web-console routes")
	}
}

// [REQ:ROUTE-003] Seed is idempotent — doesn't duplicate existing routes
func TestSeedFromConfigIdempotent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)

	// Pre-create one route
	_, err := svc.Create(RouteInput{
		Subdomain:    "agent-manager",
		ScenarioName: "agent-manager",
		LocalPort:    36238,
	})
	if err != nil {
		t.Fatalf("pre-create: %v", err)
	}

	configYAML := `tunnel: test-tunnel-id
ingress:
  - hostname: agent-manager.itsagitime.com
    service: http://localhost:36238
  - hostname: web-console.itsagitime.com
    service: http://localhost:36240
  - service: http_status:404
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	seeded, err := svc.SeedFromConfig(configPath)
	if err != nil {
		t.Fatalf("SeedFromConfig: %v", err)
	}
	// Only web-console should be seeded (agent-manager already exists)
	if seeded != 1 {
		t.Errorf("seeded = %d, want 1", seeded)
	}

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes total, got %d", len(routes))
	}
}

// [REQ:ROUTE-003] Seed handles missing config file gracefully
func TestSeedFromConfigMissingFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)

	_, err := svc.SeedFromConfig("/nonexistent/config.yml")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}
