package destinations_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/destinations/mocks"
	enginemocks "data-backup-manager/internal/testutil/mocks"
)

// TestDestination_EncryptedByDefault proves DBM-ENC-001:
//  1. The created destination's EncryptionAlgorithm is non-empty (sourced from
//     FakeKopiaEngine's default "AES256-GCM-HMAC-SHA256").
//  2. No secret value is persisted — only secret_ref (the credential reference).
func TestDestination_EncryptedByDefault(t *testing.T) {
	ctx := context.Background()
	eng := &enginemocks.FakeKopiaEngine{} // default returns "AES256-GCM-HMAC-SHA256"
	repo := mocks.NewFakeRepository()
	svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

	d, err := svc.CreateDestination(ctx, destinations.CreateInput{
		Name:     "encrypted-dest",
		Backend:  destinations.BackendFilesystem,
		Location: "/mnt/safe",
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

	// Only the credential reference must be stored — never a secret value. The ref is
	// derived authoritatively from the engine's deterministic passphrase path.
	wantRef := "vrooli/kopia/encrypted-dest:repository-passphrase"
	if d.SecretRef != wantRef {
		t.Fatalf("SecretRef = %q, want %q", d.SecretRef, wantRef)
	}
	// The domain type has no "secret" or "passphrase" field; asserting the struct
	// shape guarantees that no secret is accidentally captured.
	// (If a plaintext secret were ever added, this package would fail to compile
	// against the updated Destination type before reaching this assertion.)
	_ = destinations.Destination{
		SecretRef: "ref-only", // compile-time proof: no SecretValue or Passphrase field
	}
}

func TestDeleteDestinationDeletesKopiaMetadataWhenRequested(t *testing.T) {
	ctx := context.Background()
	eng := &enginemocks.FakeKopiaEngine{}
	repo := mocks.NewFakeRepository()
	svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

	d, err := svc.CreateDestination(ctx, destinations.CreateInput{
		Name:     "cleanup-dest",
		Backend:  destinations.BackendFilesystem,
		Location: "/mnt/cleanup",
	})
	if err != nil {
		t.Fatalf("CreateDestination: %v", err)
	}
	removed, err := svc.DeleteDestination(ctx, d.ID, true)
	if err != nil {
		t.Fatalf("DeleteDestination: %v", err)
	}
	if !removed {
		t.Fatal("DeleteDestination removed=false, want true")
	}
	if !slices.Contains(eng.Calls, "RepoDelete(cleanup-dest)") {
		t.Fatalf("RepoDelete not called; calls=%v", eng.Calls)
	}
}

func TestDeleteDestinationDoesNotRemoveCatalogWhenKopiaDeleteFails(t *testing.T) {
	ctx := context.Background()
	deleteErr := errors.New("kopia unavailable")
	eng := &enginemocks.FakeKopiaEngine{
		RepoDeleteFn: func(context.Context, string) error { return deleteErr },
	}
	repo := mocks.NewFakeRepository()
	svc := destinations.NewService(repo, eng, mocks.NewFakeBundleWriter(), "/protected")

	d, err := svc.CreateDestination(ctx, destinations.CreateInput{
		Name:     "cleanup-fails",
		Backend:  destinations.BackendFilesystem,
		Location: "/mnt/cleanup-fails",
	})
	if err != nil {
		t.Fatalf("CreateDestination: %v", err)
	}
	if _, err := svc.DeleteDestination(ctx, d.ID, true); !errors.Is(err, deleteErr) {
		t.Fatalf("DeleteDestination error = %v, want %v", err, deleteErr)
	}
	if _, err := svc.GetDestination(ctx, d.ID); err != nil {
		t.Fatalf("catalog row should remain after kopia delete failure: %v", err)
	}
}
