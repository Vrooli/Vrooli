package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultConfigRendersLoopbackFileStorage(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	cfg, err := DefaultConfig(dataDir, 8200)
	if err != nil {
		t.Fatal(err)
	}
	body, err := cfg.RenderHCL()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`storage "file"`, `address = "127.0.0.1:8200"`, `tls_disable = true`, `ui = false`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("rendered config missing %q:\n%s", expected, body)
		}
	}
	if cfg.StoragePath != dataDir {
		t.Fatalf("storage path = %q, want existing data root %q", cfg.StoragePath, dataDir)
	}
}

func TestConfigRejectsNonLoopbackListener(t *testing.T) {
	cfg, err := DefaultConfig(t.TempDir(), 8200)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddr = "0.0.0.0:8200"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("Validate() error = %v, want loopback denial", err)
	}
}

func TestConfigWriteUsesRestrictedPermissions(t *testing.T) {
	cfg, err := DefaultConfig(t.TempDir(), 8200)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "vault.hcl")
	if err := cfg.Write(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestDefaultConfigPreservesExistingCanonicalDataDirectory(t *testing.T) {
	// Native Vault uses the canonical data directory directly, preserving the
	// supported file-storage state without copying bootstrap material or reading
	// a root token during migration.
	dataDir := filepath.Join(t.TempDir(), "vault-file")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "core"), 0o700); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dataDir, "core", "state")
	if err := os.WriteFile(statePath, []byte("state"), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := DefaultConfig(dataDir, 8200)
	if err != nil {
		t.Fatal(err)
	}
	if config.StoragePath != dataDir {
		t.Fatalf("storage path = %q, want existing canonical data directory %q", config.StoragePath, dataDir)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("existing Vault state must remain untouched: %v", err)
	}
}
