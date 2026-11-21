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
		RelativePath:        "vrooli-backups",
		RepositoryPath:      repoPath,
		EncryptionAlgorithm: "AES256-GCM-HMAC-SHA256",
		SecretRef:           "vrooli/kopia/elements-local:repository-passphrase",
		CreatedAt:           time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC),
		Host:                "host-1",
		User:                "operator",
	}
}

func writeRepoMarker(t *testing.T, repoPath string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoPath, "kopia.repository.f"), []byte(`{"uniqueID":"fixture-repository"}`), 0o600); err != nil {
		t.Fatal(err)
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
	writeRepoMarker(t, repoPath)
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
	if manifest["repository_unique_id"] != "fixture-repository" {
		t.Errorf("manifest repository_unique_id = %v, want fixture repository identity", manifest["repository_unique_id"])
	}
	if manifest["relative_path"] != "vrooli-backups" {
		t.Errorf("manifest relative_path = %v, want vrooli-backups", manifest["relative_path"])
	}
	if manifest["secret_ref"] == "" {
		t.Error("manifest should carry the credential reference")
	}
}

func TestFSBundleWriterRefusesNonemptyRepositoryPathBeforeCreate(t *testing.T) {
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	if err := os.MkdirAll(repoPath, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(repoPath, "kopia.repository.f")
	contents := []byte("existing repository must be preserved")
	if err := os.WriteFile(marker, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (&destinations.FSBundleWriter{}).PrepareRepository(context.Background(), root, repoPath); err == nil {
		t.Fatal("PrepareRepository accepted a nonempty repository path")
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != string(contents) {
		t.Fatalf("existing repository marker changed: %q / %v", got, err)
	}
}

func TestFSBundleWriterAllowsUnrelatedFilesAtMixedUseDriveRoot(t *testing.T) {
	root := t.TempDir()
	keepPath := filepath.Join(root, "operator-data", "notes.txt")
	if err := os.MkdirAll(filepath.Dir(keepPath), 0o700); err != nil {
		t.Fatal(err)
	}
	keep := []byte("user data that is not part of the Kopia repository")
	if err := os.WriteFile(keepPath, keep, 0o600); err != nil {
		t.Fatal(err)
	}
	repoPath := destinations.RepositoryPathFor(root, "mixed-use")
	if err := (&destinations.FSBundleWriter{}).PrepareRepository(context.Background(), root, repoPath); err != nil {
		t.Fatalf("prepare repository on mixed-use drive: %v", err)
	}
	if got, err := os.ReadFile(keepPath); err != nil || string(got) != string(keep) {
		t.Fatalf("unrelated drive content changed: %q / %v", got, err)
	}
	entries, err := os.ReadDir(repoPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("new repository path contains unexpected content: %v", entries)
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
	writeRepoMarker(t, repoPath)
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
	writeRepoMarker(t, repoPath)
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

func TestFSBundleWriter_RefreshMetadataRepairsOnlyManagedCredentialReference(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}
	meta := newMeta(root, repoPath)
	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, repoPath)
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}

	legacy := meta
	legacy.SecretRef = "secret/resources/kopia/repo/elements-local/passphrase"
	if err := w.RefreshMetadata(ctx, legacy); err != nil {
		t.Fatalf("seed legacy metadata: %v", err)
	}
	updated := meta
	updated.SecretRef = "vrooli/kopia/elements-local:repository-passphrase"
	if err := w.RefreshMetadata(ctx, updated); err != nil {
		t.Fatalf("RefreshMetadata: %v", err)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(root, destinations.BundleManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if got := manifest["secret_ref"]; got != updated.SecretRef {
		t.Fatalf("manifest secret_ref = %v, want %q", got, updated.SecretRef)
	}
	recovery, err := os.ReadFile(filepath.Join(root, destinations.BundleRecoveryFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(recovery), "secret/resources/kopia") || !strings.Contains(string(recovery), updated.SecretRef) {
		t.Fatalf("recovery document did not refresh authority reference: %s", recovery)
	}
}

func TestFSBundleWriter_RefreshMetadataRejectsDifferentDestination(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}
	meta := newMeta(root, repoPath)
	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, repoPath)
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	other := meta
	other.DestinationID = "other"
	if err := w.RefreshMetadata(ctx, other); err == nil {
		t.Fatal("RefreshMetadata accepted a different destination identity")
	}
}

func TestFSBundleWriter_AnchorsLegacyRepositoryOnlyAtOriginalPath(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}
	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, repoPath)
	meta := newMeta(root, repoPath)
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, destinations.BundleManifestFile)
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	delete(manifest, "repository_unique_id") // old bundle written before identity anchoring
	delete(manifest, "relative_path")        // ...and before the locator was recorded
	contents, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	legacyManifest := append(append([]byte(nil), contents...), '\n')
	if err := os.WriteFile(manifestPath, legacyManifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.ValidateRelocation(ctx, meta); err != nil {
		t.Fatalf("legacy bundle at unchanged path: %v", err)
	}
	if err := w.RebindMetadata(ctx, meta); err != nil {
		t.Fatalf("anchor legacy bundle at unchanged path: %v", err)
	}
	contents, err = os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifest = nil
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["repository_unique_id"] != "fixture-repository" {
		t.Fatalf("repository identity was not anchored: %v", manifest)
	}

	moved := meta
	moved.BundleRoot = filepath.Join(t.TempDir(), "relocated")
	moved.RepositoryPath = destinations.RepositoryPathFor(moved.BundleRoot, meta.Name)
	if err := os.MkdirAll(moved.BundleRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moved.BundleRoot, destinations.BundleManifestFile), legacyManifest, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(moved.RepositoryPath, 0o700); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, moved.RepositoryPath)
	if err := w.ValidateRelocation(ctx, moved); err == nil {
		t.Fatal("legacy bundle with no prior identity anchor was allowed to relocate")
	}
}

func TestFSBundleWriter_AcceptsLegacyManifestWithoutRelativePath(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repoPath := destinations.RepositoryPathFor(root, "elements-local")
	w := &destinations.FSBundleWriter{}
	if err := w.PrepareRepository(ctx, root, repoPath); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, repoPath)
	meta := newMeta(root, repoPath)
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, destinations.BundleManifestFile)
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	delete(manifest, "relative_path")
	delete(manifest, "repository_unique_id")
	contents, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(contents, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.ValidateRelocation(ctx, meta); err != nil {
		t.Fatalf("legacy manifest without relative_path should be readable at its unchanged path: %v", err)
	}
}

func TestFSBundleWriter_RebindRetryConvergesAfterPartialMetadataWrite(t *testing.T) {
	ctx := context.Background()
	w := &destinations.FSBundleWriter{}
	oldRoot := t.TempDir()
	oldRepo := destinations.RepositoryPathFor(oldRoot, "elements-local")
	if err := w.PrepareRepository(ctx, oldRoot, oldRepo); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, oldRepo)
	meta := newMeta(oldRoot, oldRepo)
	if err := w.WriteMetadata(ctx, meta); err != nil {
		t.Fatal(err)
	}

	newRoot := t.TempDir()
	newRepo := destinations.RepositoryPathFor(newRoot, "elements-local")
	if err := os.MkdirAll(newRepo, 0o700); err != nil {
		t.Fatal(err)
	}
	writeRepoMarker(t, newRepo)
	for _, name := range []string{destinations.BundleManifestFile, destinations.BundleReadmeFile, destinations.BundleRecoveryFile} {
		data, err := os.ReadFile(filepath.Join(oldRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(newRoot, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	const unrelated = "operator-notes.txt"
	if err := os.WriteFile(filepath.Join(newRoot, unrelated), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	moved := meta
	moved.BundleRoot, moved.RepositoryPath = newRoot, newRepo
	if err := w.ValidateRelocation(ctx, moved); err != nil {
		t.Fatalf("ValidateRelocation: %v", err)
	}
	if err := os.Remove(filepath.Join(newRoot, destinations.BundleRecoveryFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(newRoot, destinations.BundleRecoveryFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := w.RebindMetadata(ctx, moved); err == nil {
		t.Fatal("expected injected recovery-file projection failure")
	}
	if err := os.Remove(filepath.Join(newRoot, destinations.BundleRecoveryFile)); err != nil {
		t.Fatal(err)
	}
	if err := w.RebindMetadata(ctx, moved); err != nil {
		t.Fatalf("retry RebindMetadata: %v", err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(newRoot, destinations.BundleManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["repository_unique_id"] != "fixture-repository" || manifest["repository_path"] != newRepo {
		t.Fatalf("rebound manifest = %v", manifest)
	}
	if notes, err := os.ReadFile(filepath.Join(newRoot, unrelated)); err != nil || string(notes) != "keep" {
		t.Fatalf("unrelated file changed: %q / %v", notes, err)
	}
}

func TestFSBundleWriter_RejectsRepoAtBundleRoot(t *testing.T) {
	ctx := context.Background()
	for _, marker := range []string{"kopia.repository", "kopia.repository.f"} {
		t.Run(marker, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, marker), []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
			w := &destinations.FSBundleWriter{}
			if err := w.PrepareRepository(ctx, root, destinations.RepositoryPathFor(root, "elements-local")); err == nil {
				t.Fatal("expected rejection when bundle root is already a kopia repository")
			}
		})
	}
}

func TestFSBundleWriterNeverAdoptsAManagedNameWithTheWrongType(t *testing.T) {
	ctx := context.Background()
	w := &destinations.FSBundleWriter{}
	cases := []struct {
		name  string
		setup func(root string) error
	}{
		{"repositories is a file", func(root string) error {
			return os.WriteFile(filepath.Join(root, "repositories"), []byte("not a directory"), 0o600)
		}},
		{"README is a directory", func(root string) error {
			return os.MkdirAll(filepath.Join(root, destinations.BundleReadmeFile), 0o700)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := tc.setup(root); err != nil {
				t.Fatal(err)
			}
			err := w.PrepareRepository(ctx, root, destinations.RepositoryPathFor(root, "repo"))
			if err == nil {
				t.Fatal("a managed-name type conflict must be refused")
			}
			if !strings.Contains(err.Error(), "bundle root") {
				t.Fatalf("error should name the bundle-root conflict: %v", err)
			}
		})
	}
}

func TestFSBundleWriterRefusesARepositoryOutsideItsBundle(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "elsewhere", "repo.kopia")
	err := (&destinations.FSBundleWriter{}).PrepareRepository(ctx, root, outside)
	if err == nil {
		t.Fatal("a repository path outside the bundle must be refused")
	}
}

func TestForeignBundleEntriesListsOnlyUnmanagedNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"operator-data", "photo.jpg"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, destinations.BundleManifestFile), 0o700); err != nil {
		t.Fatal(err)
	}
	foreign, err := destinations.ForeignBundleEntries(root)
	if err != nil {
		t.Fatal(err)
	}
	// The manifest name is managed even when its type is wrong; the guard for
	// that lives in PrepareRepository.
	if len(foreign) != 2 || foreign[0] != "operator-data" || foreign[1] != "photo.jpg" {
		t.Fatalf("foreign entries = %v", foreign)
	}
}
