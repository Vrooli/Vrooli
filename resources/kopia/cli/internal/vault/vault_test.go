package vault_test

import (
	"context"
	"errors"
	"resource-kopia/cli/internal/vault"
	"resource-kopia/cli/internal/vault/mocks"
	"strings"
	"testing"
)

func TestEnsurePassphraseGeneratesAndStores(t *testing.T) {
	v := mocks.NewFakeVault()
	p, err := vault.EnsurePassphrase(context.Background(), v, "nightly")
	if err != nil {
		t.Fatalf("EnsurePassphrase error = %v", err)
	}
	if len(p) < 32 {
		t.Fatalf("generated passphrase too short: %d", len(p))
	}
	stored, ok := v.Value(vault.PassphrasePath("nightly"), "passphrase")
	if !ok {
		t.Fatal("passphrase was not stored in vault")
	}
	if stored != p {
		t.Fatalf("stored passphrase %q != returned %q", stored, p)
	}
}

func TestEnsurePassphraseReusesExisting(t *testing.T) {
	v := mocks.NewFakeVault()
	v.Seed(vault.PassphrasePath("nightly"), "passphrase", "existing-secret-value-1234567890")
	p, err := vault.EnsurePassphrase(context.Background(), v, "nightly")
	if err != nil {
		t.Fatalf("EnsurePassphrase error = %v", err)
	}
	if p != "existing-secret-value-1234567890" {
		t.Fatalf("expected existing passphrase reused, got %q", p)
	}
	if len(v.Puts) != 0 {
		t.Fatalf("should not write when passphrase exists, puts = %v", v.Puts)
	}
}

func TestEnsurePassphraseNeverEmpty(t *testing.T) {
	v := mocks.NewFakeVault()
	v.Seed(vault.PassphrasePath("nightly"), "passphrase", "   ")
	_, err := vault.EnsurePassphrase(context.Background(), v, "nightly")
	if err == nil {
		t.Fatal("expected error for empty stored passphrase, got nil")
	}
}

func TestRequirePassphraseFailsClosed(t *testing.T) {
	v := mocks.NewFakeVault()
	_, err := vault.RequirePassphrase(context.Background(), v, "missing")
	if !errors.Is(err, vault.ErrPassphraseMissing) {
		t.Fatalf("expected ErrPassphraseMissing, got %v", err)
	}
}

func TestGeneratePassphraseStrength(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		p, err := vault.GeneratePassphrase()
		if err != nil {
			t.Fatalf("GeneratePassphrase error = %v", err)
		}
		if strings.TrimSpace(p) == "" || len(p) < 32 {
			t.Fatalf("weak passphrase: %q", p)
		}
		if seen[p] {
			t.Fatalf("passphrase collision: %q", p)
		}
		seen[p] = true
	}
}

func TestS3CredentialsRoundTrip(t *testing.T) {
	v := mocks.NewFakeVault()
	want := vault.S3Credentials{AccessKeyID: "AKIA", SecretAccessKey: "shhh"}
	if err := vault.PutS3Credentials(context.Background(), v, "offsite", want); err != nil {
		t.Fatalf("PutS3Credentials error = %v", err)
	}
	got, found, err := vault.S3CredentialsFor(context.Background(), v, "offsite")
	if err != nil || !found {
		t.Fatalf("S3CredentialsFor found=%v err=%v", found, err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, want)
	}
}

func TestEnsurePassphraseSurfacesVaultError(t *testing.T) {
	v := mocks.NewFakeVault()
	v.GetErr = errors.New("vault down")
	_, err := vault.EnsurePassphrase(context.Background(), v, "nightly")
	if err == nil {
		t.Fatal("expected error when vault is down")
	}
}
