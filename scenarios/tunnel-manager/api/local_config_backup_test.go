package main

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:LOCAL-003] Config write with backup tests

func TestLocalConfig_WriteWithBackup(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")

	// Write initial config
	if err := os.WriteFile(cfgPath, []byte("tunnel: original\n"), 0o644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	mgr := NewLocalConfigManager(WithLocalConfigPath(cfgPath))
	cfg := &CloudflaredConfig{
		Tunnel:  "updated",
		Ingress: []CloudflaredIngress{{Service: "http_status:404"}},
	}

	if err := mgr.WriteWithBackup(cfg, 5); err != nil {
		t.Fatalf("WriteWithBackup: %v", err)
	}

	// Verify new config was written
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read new config: %v", err)
	}
	if string(data) == "tunnel: original\n" {
		t.Error("config was not updated")
	}

	// Verify backup was created
	matches, _ := filepath.Glob(filepath.Join(dir, "config.yml.backup.*"))
	if len(matches) != 1 {
		t.Errorf("expected 1 backup, found %d", len(matches))
	}

	// Verify backup contains original content
	backupData, _ := os.ReadFile(matches[0])
	if string(backupData) != "tunnel: original\n" {
		t.Errorf("backup content mismatch: got %s", string(backupData))
	}
}

func TestLocalConfig_WriteCreatesDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "subdir", "config.yml")

	mgr := NewLocalConfigManager(WithLocalConfigPath(cfgPath))
	cfg := &CloudflaredConfig{
		Ingress: []CloudflaredIngress{{Service: "http_status:404"}},
	}

	if err := mgr.WriteWithBackup(cfg, 5); err != nil {
		t.Fatalf("WriteWithBackup (new dir): %v", err)
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}
}

func TestLocalConfig_PruneBackups(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yml")

	// Create 7 fake backups
	for i := 0; i < 7; i++ {
		name := filepath.Join(dir, "config.yml.backup.2026010"+string(rune('0'+i))+"-120000")
		if err := os.WriteFile(name, []byte("backup"), 0o644); err != nil {
			t.Fatalf("write backup %d: %v", i, err)
		}
	}
	if err := os.WriteFile(cfgPath, []byte("current"), 0o644); err != nil {
		t.Fatalf("write current config: %v", err)
	}

	mgr := NewLocalConfigManager(WithLocalConfigPath(cfgPath))
	cfg := &CloudflaredConfig{
		Ingress: []CloudflaredIngress{{Service: "http_status:404"}},
	}

	// Write with max 3 backups
	if err := mgr.WriteWithBackup(cfg, 3); err != nil {
		t.Fatalf("WriteWithBackup: %v", err)
	}

	// Should have at most 3 old backups + 1 new = 4, but pruning happens before new write
	matches, _ := filepath.Glob(filepath.Join(dir, "config.yml.backup.*"))
	// After prune(3): keep 3 of 7, then add 1 new = 4
	if len(matches) > 4 {
		t.Errorf("expected at most 4 backups after prune, found %d", len(matches))
	}
}
