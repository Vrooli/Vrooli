package credentials

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	kopiaregistry "github.com/vrooli/vrooli/packages/kopiaregistry-go"
)

type fakeStore struct {
	value  string
	putErr error
	values map[string]string
}

func (f *fakeStore) Put(identity credentialauthority.Identity, field, value string) error {
	if f.putErr != nil {
		return f.putErr
	}
	if f.values == nil {
		f.values = map[string]string{}
	}
	f.values[string(identity)+":"+field] = value
	return nil
}

func (f *fakeStore) Resolve(identity credentialauthority.Identity, field string) (string, error) {
	if value, ok := f.values[string(identity)+":"+field]; ok {
		return value, nil
	}
	if f.value == "" {
		return "", credentialauthority.ErrUnconfigured
	}
	return f.value, nil
}

func (f *fakeStore) Delete(identity credentialauthority.Identity, field string) error {
	delete(f.values, string(identity)+":"+field)
	return nil
}

func TestGeneratePassphraseStrength(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		value, err := GeneratePassphrase()
		if err != nil || strings.TrimSpace(value) == "" || len(value) < 32 {
			t.Fatalf("GeneratePassphrase() = %q, %v", value, err)
		}
		if seen[value] {
			t.Fatalf("passphrase collision: %q", value)
		}
		seen[value] = true
	}
}

func TestValidateStoredPassphraseRequiresMatchingReadback(t *testing.T) {
	identity, err := kopiaregistry.PassphraseIdentity("nightly")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStoredPassphrase(&fakeStore{value: "passphrase"}, identity, "passphrase"); err != nil {
		t.Fatal(err)
	}
	for _, store := range []Store{&fakeStore{value: "other"}, &fakeStore{putErr: errors.New("locked")}} {
		if err := ValidateStoredPassphrase(store, identity, "passphrase"); err == nil {
			t.Fatal("ValidateStoredPassphrase accepted a failed readback")
		}
	}
}

func TestS3CredentialsRoundTripUsesAuthorityIdentity(t *testing.T) {
	store := &fakeStore{values: map[string]string{}}
	want := S3Credentials{AccessKeyID: "AKIAEXAMPLE", SecretAccessKey: "secret-value"}
	if err := PutS3Credentials(store, "offsite", want); err != nil {
		t.Fatal(err)
	}
	got, found, err := S3CredentialsFor(store, "offsite")
	if err != nil || !found || got != want {
		t.Fatalf("S3CredentialsFor() = %#v, found=%v, err=%v; want %#v", got, found, err, want)
	}
	if err := DeleteS3Credentials(store, "offsite"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := S3CredentialsFor(store, "offsite"); err != nil || found {
		t.Fatalf("deleted S3 credentials = found=%v, err=%v", found, err)
	}
}

func TestS3CredentialsProviderFailureIsNotMissing(t *testing.T) {
	store := &fakeStore{value: "unused"}
	store.putErr = errors.New("locked")
	if err := PutS3Credentials(store, "offsite", S3Credentials{AccessKeyID: "a", SecretAccessKey: "b"}); err == nil {
		t.Fatal("PutS3Credentials succeeded with a locked authority")
	}
}

func TestResolvePassphraseMigratesLegacySidecarUntilOperationSucceeds(t *testing.T) {
	store := &fakeStore{values: map[string]string{}}
	legacyPath := filepath.Join(t.TempDir(), "repository.config.kopia-password")
	if err := os.WriteFile(legacyPath, []byte("legacy-passphrase\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ResolvePassphrase(store, "elements-local", legacyPath)
	if err != nil || got != "legacy-passphrase" {
		t.Fatalf("ResolvePassphrase() = %q, %v", got, err)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("legacy sidecar disappeared before operation success, stat error = %v", err)
	}
	if err := RetireLegacyPassphrase(store, "elements-local", legacyPath); err != nil {
		t.Fatalf("RetireLegacyPassphrase() = %v", err)
	}
	if _, err := os.Stat(legacyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy sidecar still exists after retirement, stat error = %v", err)
	}
	identity, err := kopiaregistry.PassphraseIdentity("elements-local")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.Resolve(identity, kopiaregistry.PassphraseField); err != nil || got != "legacy-passphrase" {
		t.Fatalf("authority passphrase = %q, %v", got, err)
	}
}

func TestResolvePassphraseUsesAuthorityWhenLegacySidecarDiffers(t *testing.T) {
	store := &fakeStore{values: map[string]string{}}
	identity, err := kopiaregistry.PassphraseIdentity("elements-local")
	if err != nil {
		t.Fatal(err)
	}
	store.values[string(identity)+":"+kopiaregistry.PassphraseField] = "authority-passphrase"
	legacyPath := filepath.Join(t.TempDir(), "repository.config.kopia-password")
	if err := os.WriteFile(legacyPath, []byte("different-passphrase\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolvePassphrase(store, "elements-local", legacyPath)
	if err != nil || got != "authority-passphrase" {
		t.Fatalf("ResolvePassphrase() = %q, %v", got, err)
	}
	if _, err := os.Stat(legacyPath); err != nil {
		t.Fatalf("conflicting legacy sidecar was removed: %v", err)
	}
}

func TestRetireLegacyPassphraseRemovesConflictingSidecarAfterVerifiedOperation(t *testing.T) {
	store := &fakeStore{values: map[string]string{}}
	identity, err := kopiaregistry.PassphraseIdentity("elements-local")
	if err != nil {
		t.Fatal(err)
	}
	store.values[string(identity)+":"+kopiaregistry.PassphraseField] = "authority-passphrase"
	legacyPath := filepath.Join(t.TempDir(), "repository.config.kopia-password")
	if err := os.WriteFile(legacyPath, []byte("different-passphrase\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RetireLegacyPassphrase(store, "elements-local", legacyPath); err != nil {
		t.Fatalf("RetireLegacyPassphrase() = %v", err)
	}
	if _, err := os.Stat(legacyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy sidecar still exists, stat error = %v", err)
	}
}
