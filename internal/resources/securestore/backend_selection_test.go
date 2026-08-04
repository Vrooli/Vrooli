package securestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackendSelectionRoundTripsWithoutCredentialMaterial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "credential-backend.json")
	previous := backendSelectionPath
	backendSelectionPath = func() (string, error) { return path, nil }
	t.Cleanup(func() { backendSelectionPath = previous })

	if _, found, err := SelectedBackend(); err != nil || found {
		t.Fatalf("missing selection = found %t, err %v; want absent without error", found, err)
	}
	if err := SelectBackend(BackendEncryptedFile, "native backend was not writable during setup"); err != nil {
		t.Fatalf("SelectBackend: %v", err)
	}
	backend, found, err := SelectedBackend()
	if err != nil || !found || backend != BackendEncryptedFile {
		t.Fatalf("SelectedBackend = %q, found %t, err %v; want encrypted-file", backend, found, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read selection: %v", err)
	}
	if string(data) == "" || filepath.Base(path) != "credential-backend.json" {
		t.Fatalf("selection file was not written")
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("selection permissions = %v, err %v; want 0600", info.Mode().Perm(), err)
	}
}

func TestBackendSelectionRejectsUnknownBackend(t *testing.T) {
	if err := SelectBackend("split-brain", "test"); err == nil {
		t.Fatal("SelectBackend accepted an unknown backend")
	}
}
