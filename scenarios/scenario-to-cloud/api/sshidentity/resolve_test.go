package sshidentity

import (
	"os"
	"path/filepath"
	"testing"

	"scenario-to-cloud/domain"
)

func TestDefaultResolver_PrefersManifestKeyPath(t *testing.T) {
	r := DefaultResolver{}
	m := domain.CloudManifest{Target: domain.ManifestTarget{VPS: &domain.ManifestVPS{KeyPath: "~/.ssh/id_ed25519"}}}
	resolved, err := r.Resolve(m, nil)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if resolved.AuthMode != AuthModeExplicitKey {
		t.Fatalf("AuthMode=%q, want %q", resolved.AuthMode, AuthModeExplicitKey)
	}
	if resolved.KeyPath == "" {
		t.Fatal("expected key path from manifest")
	}
}

func TestDefaultResolver_FallsBackToPersistedExplicitKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "id_test")
	if err := os.WriteFile(keyPath, []byte("private"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	r := DefaultResolver{}
	m := domain.CloudManifest{Target: domain.ManifestTarget{VPS: &domain.ManifestVPS{}}}
	existing := &DeploymentSSHIdentity{AuthMode: AuthModeExplicitKey, VerificationState: VerificationAuthorized, KeyPath: keyPath}

	resolved, err := r.Resolve(m, existing)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if resolved.AuthMode != AuthModeExplicitKey {
		t.Fatalf("AuthMode=%q, want explicit_key", resolved.AuthMode)
	}
	if resolved.KeyPath != keyPath {
		t.Fatalf("KeyPath=%q, want %q", resolved.KeyPath, keyPath)
	}
	if resolved.VerificationState != VerificationUnknown {
		t.Fatalf("VerificationState=%q, want unknown", resolved.VerificationState)
	}
}

func TestDefaultResolver_UsesAmbientModeWhenNoExplicitKey(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	r := DefaultResolver{}
	m := domain.CloudManifest{Target: domain.ManifestTarget{VPS: &domain.ManifestVPS{}}}
	resolved, err := r.Resolve(m, nil)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if resolved.AuthMode != AuthModeDefaultSSH {
		t.Fatalf("AuthMode=%q, want %q", resolved.AuthMode, AuthModeDefaultSSH)
	}
}
