package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:LOCAL-001] Parse existing cloudflared config tests

func TestLocalConfig_Parse(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")

	content := `tunnel: abc-123
credentials-file: /home/user/.cloudflared/abc-123.json
ingress:
  - hostname: app.example.com
    service: http://localhost:8080
  - hostname: api.example.com
    service: http://localhost:9090
  - service: http_status:404
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	mgr := NewLocalConfigManager(WithLocalConfigPath(cfgPath))
	cfg, err := mgr.Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if cfg.Tunnel != "abc-123" {
		t.Errorf("tunnel: want abc-123, got %s", cfg.Tunnel)
	}
	if cfg.CredentialsFile != "/home/user/.cloudflared/abc-123.json" {
		t.Errorf("credentials-file: got %s", cfg.CredentialsFile)
	}
	if len(cfg.Ingress) != 3 {
		t.Fatalf("ingress rules: want 3, got %d", len(cfg.Ingress))
	}
	if cfg.Ingress[0].Hostname != "app.example.com" {
		t.Errorf("first rule hostname: got %s", cfg.Ingress[0].Hostname)
	}
	// Last rule is catch-all (no hostname)
	if cfg.Ingress[2].Hostname != "" {
		t.Errorf("catch-all should have empty hostname, got %s", cfg.Ingress[2].Hostname)
	}
}

func TestLocalConfig_ParseMissing(t *testing.T) {
	mgr := NewLocalConfigManager(WithLocalConfigPath("/nonexistent/config.yml"))
	_, err := mgr.Parse()
	if err == nil {
		t.Fatal("expected error for missing config")
	}
}

func TestLocalConfig_ParseInvalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	mgr := NewLocalConfigManager(WithLocalConfigPath(cfgPath))
	_, err := mgr.Parse()
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
