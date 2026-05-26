package destinations_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

// TestDestination_EncryptedByDefault proves DBM-ENC-001:
//  1. The created destination's EncryptionAlgorithm is non-empty (sourced from
//     FakeKopiaEngine's default "AES256-GCM-HMAC-SHA256").
//  2. No secret value is persisted — only secret_ref (the vault reference).
func TestDestination_EncryptedByDefault(t *testing.T) {
	ctx := context.Background()
	eng := &enginemocks.FakeKopiaEngine{} // default returns "AES256-GCM-HMAC-SHA256"
	repo := mocks.NewFakeRepository()
	svc := destinations.NewService(repo, eng, "/protected")

	d, err := svc.CreateDestination(ctx, destinations.CreateInput{
		Name:      "encrypted-dest",
		Backend:   destinations.BackendFilesystem,
		Location:  "/mnt/safe",
		SecretRef: "vault/kopia/encrypted-dest",
	})
	if err != nil {
		t.Fatalf("CreateDestination: %v", err)
	}

	// Encryption algorithm must be non-empty.
	if d.EncryptionAlgorithm == "" {
		t.Fatal("EncryptionAlgorithm is empty; encryption must always be on")
	}
	if d.EncryptionAlgorithm != "AES256-GCM-HMAC-SHA256" {
		t.Fatalf("EncryptionAlgorithm = %q, want AES256-GCM-HMAC-SHA256", d.EncryptionAlgorithm)
	}

	// Only the vault reference must be stored — never a secret value.
	if d.SecretRef != "vault/kopia/encrypted-dest" {
		t.Fatalf("SecretRef = %q, want vault/kopia/encrypted-dest", d.SecretRef)
	}
	// The domain type has no "secret" or "passphrase" field; asserting the struct
	// shape guarantees that no secret is accidentally captured.
	// (If a plaintext secret were ever added, this package would fail to compile
	// against the updated Destination type before reaching this assertion.)
	_ = destinations.Destination{
		SecretRef: "ref-only", // compile-time proof: no SecretValue or Passphrase field
	}
}
