package resources

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/internal/secrets"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

// TestKopiaCredentialRecoveryDrill isolates both sides of a Kopia destination
// state, destroys them, and proves that the operator-held credential bundle
// restores the only irreplaceable part: the repository passphrase. Registry
// metadata is non-secret and is restored as ordinary state; the test refuses to
// operate unless every destructive path is under t.TempDir.
func TestKopiaCredentialRecoveryDrill(t *testing.T) {
	root := t.TempDir()
	credentialPath := filepath.Join(root, "credentials")
	kopiaState := filepath.Join(root, "kopia-state")
	kopiaCache := filepath.Join(root, "kopia-cache")
	bundlePath := filepath.Join(root, "recovery.bundle")
	for _, path := range []string{credentialPath, kopiaState, kopiaCache} {
		if !withinTemp(root, path) {
			t.Fatalf("isolation path escaped temp root: %q", path)
		}
	}
	if err := os.MkdirAll(credentialPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(credentialPath, "isolated-store.marker"), []byte("isolated"), 0o600); err != nil {
		t.Fatal(err)
	}

	registryPath := filepath.Join(kopiaState, "registry.json")
	registry := kopiaregistry.New(registryPath)
	entry := kopiaregistry.Entry{Name: "drill-repository", Backend: kopiaregistry.BackendFilesystem, ConfigFile: filepath.Join(kopiaState, "repository.config"), CacheDir: kopiaCache, Path: filepath.Join(root, "repository")}
	if err := registry.Upsert(entry); err != nil {
		t.Fatal(err)
	}
	registryBytes, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}

	store := &memorySecureStore{values: map[string]string{}}
	authority, err := credentialauthority.NewAuthority(store)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := kopiaregistry.PassphraseIdentity(entry.Name)
	if err != nil {
		t.Fatal(err)
	}
	const passphrase = "kopia-drill-passphrase"
	if err := authority.Put(identity, kopiaregistry.PassphraseField, passphrase); err != nil {
		t.Fatal(err)
	}
	bundle, err := authority.ExportRecovery([]credentialauthority.RecoveryEntry{{Identity: identity, Field: kopiaregistry.PassphraseField}}, "operator-bundle-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bundle, []byte(passphrase)) {
		t.Fatal("recovery bundle contains the Kopia passphrase in plaintext")
	}
	if err := os.WriteFile(bundlePath, bundle, 0o600); err != nil {
		t.Fatal(err)
	}

	// Disaster: local credential, state, and cache are gone. The repository
	// bytes are intentionally outside those directories and remain available.
	for _, path := range []string{credentialPath, kopiaState, kopiaCache} {
		if err := os.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(credentialPath); !os.IsNotExist(err) {
		t.Fatalf("isolated credential store still exists after destruction: %v", err)
	}
	if _, _, err := kopiaregistry.New(registryPath).Get(entry.Name); err != nil {
		t.Fatal(err)
	}
	if found, err := (&memorySecureStore{values: map[string]string{}}).Get(string(identity), kopiaregistry.PassphraseField); err == nil {
		t.Fatalf("destroyed credential store returned a value of length %d", len(found))
	}

	// Replacement host: restore the non-secret registry state and the encrypted
	// credential bundle into fresh isolated stores.
	if err := os.MkdirAll(filepath.Dir(registryPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registryPath, registryBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	replacementStore := &memorySecureStore{values: map[string]string{}}
	replacementAuthority, err := credentialauthority.NewAuthority(replacementStore)
	if err != nil {
		t.Fatal(err)
	}
	restoredBundle, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := replacementAuthority.RestoreRecovery(restoredBundle, "operator-bundle-passphrase"); err != nil {
		t.Fatal(err)
	}
	restored, err := replacementAuthority.Resolve(identity, kopiaregistry.PassphraseField)
	if err != nil || restored != passphrase {
		t.Fatalf("restored passphrase length=%d err=%v", len(restored), err)
	}
	if _, found, err := kopiaregistry.New(registryPath).Get(entry.Name); err != nil || !found {
		t.Fatalf("restored registry found=%v err=%v", found, err)
	}
}

func withinTemp(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !bytes.HasPrefix([]byte(rel), []byte(".."+string(filepath.Separator)))
}
