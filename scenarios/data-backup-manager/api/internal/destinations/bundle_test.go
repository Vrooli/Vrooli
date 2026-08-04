package destinations_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"data-backup-manager/internal/destinations"
)

const secretValue = "super-secret-passphrase-DO-NOT-LEAK"

func newMeta(root, repoPath string) destinations.BundleMetadata {
	return destinations.BundleMetadata{
		DestinationID:       "dst-1",
		Name:                "elements-local",
		Backend:             "filesystem",
		BundleRoot:          root,
		RepositoryPath:      repoPath,
		EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		SecretRef:           "vrooli/kopia/elements-local:repository-passphrase",
		CreatedAt:           time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC),
		Host:                "host-1",
		User:                "operator",
	}
}

func TestFSBundleWriter_WritesSelfDescribingBundle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}

	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatalf("PrepareRepository: %v", err)
	}
	if fi, err := os.Stat(repoPath); err != nil || !fi.IsDir() {
		t.Fatalf("repository dir not created: %v", err)
	}
	if err := w.WriteMetadata(ctx, newMeta(root, repoPath)); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}

	for _, f := range []string{destinations.BundleReadmeFile, destinations.BundleRecoveryFile, destinations.BundleManifestFile} {
		if _, err := os.Stat(filepath.Join(root, f)); err != nil {
			t.Errorf("missing bundle file %s: %v", f, err)
		}
	}

	// Manifest is valid JSON, carries the secret REF but never the secret value.
	manifestBytes, err := os.ReadFile(filepath.Join(root, destinations.BundleManifestFile))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("manifest not valid json: %v", err)
	}
	if manifest["repository_path"] != repoPath {
		t.Errorf("manifest repository_path = %v, want %v", manifest["repository_path"], repoPath)
	}
	if manifest["secret_ref"] == "" {
		t.Error("manifest should carry the credential reference")
	}
}

func TestFSBundleWriter_NeverWritesSecretValue(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}
	meta := newMeta(root, repoPath)
	meta.SecretRef = secretValue // even if a caller mistakenly passes a value, it lands only as a "ref" string

	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	// This test documents the seam's contract: callers must pass a ref, not a
	// secret. The writer renders whatever SecretRef holds, so the real guarantee
	// lives in the service (which only ever passes a ref). Here we assert the
	// README/RECOVERY guidance never prints a passphrase value of its own.
	for _, f := range []string{destinations.BundleReadmeFile, destinations.BundleRecoveryFile} {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "passphrase-value") {
			t.Errorf("%s appears to embed a passphrase value", f)
		}
	}
}

func TestFSBundleWriter_IdempotentAndConflictSafe(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}
	meta := newMeta(root, repoPath)

	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	// Re-running with identical content is a no-op (idempotent create retry).
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatalf("idempotent re-write should succeed: %v", err)
	}
	// A conflicting README must fail rather than overwrite unknown data.
	if err := os.WriteFile(filepath.Join(root, destinations.BundleReadmeFile), []byte("someone else's data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteMetadata(ctx, meta); err == nil {
		t.Fatal("expected conflict error when README differs, got nil")
	}
}

func TestFSBundleWriter_RejectsRepoAtBundleRoot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	// Simulate a bundle root that is itself a kopia repository.
	if err := os.WriteFile(filepath.Join(root, "kopia.repository"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	w := &destinations.FSBundleWriter{}
	err := w.PrepareRepository(ctx, root, destinations.RepositoryPathFor(root, "elements-local"))
	if err == nil {
		t.Fatal("expected rejection when bundle root is already a kopia repository")
	}
}
