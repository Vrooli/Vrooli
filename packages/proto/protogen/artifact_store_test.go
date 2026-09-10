package protogen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testCandidate(t *testing.T, root, sourceDigest, body string) Candidate {
	t.Helper()
	generated := filepath.Join(root, "generated")
	path := filepath.Join(generated, "go", "demo", "demo.pb.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	digest, err := artifactFileDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	return Candidate{
		SourceRoot: generated,
		Metadata: ArtifactMetadata{
			SchemaVersion:   artifactMetadataSchemaVersion,
			ArtifactID:      sourceDigest,
			SourceDigest:    sourceDigest,
			GeneratorDigest: "sha256:generator",
			Outputs: []ArtifactOutput{{
				Path:   "go/demo/demo.pb.go",
				Digest: digest,
				Size:   int64(len(body)),
			}},
		},
	}
}

func TestArtifactStorePublishesAndResolvesImmutableSnapshot(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, t.TempDir(), "a", "first")
	if _, err := store.Publish(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if snapshot.ArtifactID != "a" {
		t.Fatalf("artifact id = %q, want a", snapshot.ArtifactID)
	}
	if got, err := os.ReadFile(filepath.Join(snapshot.GenRoot(), "go/demo/demo.pb.go")); err != nil || string(got) != "first" {
		t.Fatalf("resolved output = %q, err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "snapshots", "a", "metadata.json")); err != nil {
		t.Fatal(err)
	}
}

func TestArtifactStoreActivePublicationFailurePreservesPreviousSelection(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "old", "old")); err != nil {
		t.Fatal(err)
	}
	store.Hooks.BeforeActiveRename = func() error { return errors.New("injected interruption") }
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "new", "new")); !errors.Is(err, ErrArtifactPublication) {
		t.Fatalf("publish error = %v, want publication error", err)
	}
	store.Hooks.BeforeActiveRename = nil
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if snapshot.ArtifactID != "old" {
		t.Fatalf("active artifact = %q, want old", snapshot.ArtifactID)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "snapshots", "new")); err != nil {
		t.Fatalf("validated failed-publication snapshot should remain inspectable: %v", err)
	}
}

func TestArtifactStoreSnapshotPublicationFailurePreservesSelection(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "old", "old")); err != nil {
		t.Fatal(err)
	}
	store.Hooks.BeforeSnapshotRename = func() error { return errors.New("injected snapshot interruption") }
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "new", "new")); !errors.Is(err, ErrArtifactPublication) {
		t.Fatalf("publish error = %v, want publication error", err)
	}
	store.Hooks.BeforeSnapshotRename = nil
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if snapshot.ArtifactID != "old" {
		t.Fatalf("active artifact = %q, want old", snapshot.ArtifactID)
	}
}

func TestArtifactStoreFallsBackFromTamperedActiveSnapshot(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "old", "old")); err != nil {
		t.Fatal(err)
	}
	store.Hooks.BeforeActiveRename = nil
	store.Hooks.BeforeLastGoodRename = func() error { return errors.New("injected last-good interruption") }
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "new", "new")); !errors.Is(err, ErrArtifactPublication) {
		t.Fatalf("publish error = %v, want publication error", err)
	}
	store.Hooks.BeforeLastGoodRename = nil
	path := filepath.Join(store.Root, "snapshots", "new", "gen", "go", "demo", "demo.pb.go")
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if snapshot.ArtifactID != "old" {
		t.Fatalf("fallback artifact = %q, want old", snapshot.ArtifactID)
	}
}

func TestArtifactStoreRejectsMalformedAndIncompleteCandidate(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, t.TempDir(), "bad", "body")
	candidate.Metadata.Outputs[0].Digest = "sha256:not-the-file"
	if _, err := store.Publish(context.Background(), candidate); !errors.Is(err, ErrArtifactDigestMismatch) {
		t.Fatalf("digest error = %v, want digest mismatch", err)
	}
	candidate = testCandidate(t, t.TempDir(), "missing", "body")
	candidate.Metadata.Outputs = append(candidate.Metadata.Outputs, ArtifactOutput{Path: "typescript/missing.ts", Digest: "sha256:missing", Size: 1})
	if _, err := store.Publish(context.Background(), candidate); !errors.Is(err, ErrArtifactIncomplete) {
		t.Fatalf("missing output error = %v, want incomplete", err)
	}
}

func TestArtifactStoreLeaseDefersGarbageCollection(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	store.Retention = time.Hour
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "old", "old")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "new", "new")); err != nil {
		t.Fatal(err)
	}
	// Simulate an old unselected snapshot while keeping a live reader lease on it.
	oldPath := filepath.Join(store.Root, "snapshots", "old")
	oldTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	lease, err := store.acquireLeaseLocked("old")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := store.Collect(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0 while lease is live", removed)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	removed, err = store.Collect(context.Background(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1 after lease release", removed)
	}
}

func TestArtifactStoreRepublishIsIdempotent(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, t.TempDir(), "same", "same")
	if _, err := store.Publish(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(filepath.Join(store.Root, "snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "same" {
		t.Fatalf("snapshots = %#v, want one content-addressed snapshot", entries)
	}
}

func TestArtifactStoreCleanupFailsClosedOnCorruptSelection(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store.Root, "active.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Collect(context.Background(), time.Now()); !errors.Is(err, ErrArtifactRecordCorrupt) {
		t.Fatalf("cleanup error = %v, want corrupt selection", err)
	}
}

func TestArtifactStorePromoteRequiresValidatedSnapshot(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "old", "old")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, t.TempDir(), "new", "new")); err != nil {
		t.Fatal(err)
	}
	if err := store.Promote(context.Background(), "old"); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	if snapshot.ArtifactID != "old" {
		t.Fatalf("promoted artifact = %q, want old", snapshot.ArtifactID)
	}
	if err := store.Promote(context.Background(), "missing"); !errors.Is(err, ErrArtifactIncomplete) {
		t.Fatalf("invalid promotion error = %v, want incomplete artifact", err)
	}
}

func TestArtifactStoreMaterializesCompatibilityViewWithoutMixedTree(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(t.TempDir(), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), Candidate{
		Metadata: testCandidate(t, t.TempDir(), "selected", "selected").Metadata,
		SourceRoot: func() string {
			candidate := testCandidate(t, t.TempDir(), "selected", "selected")
			return candidate.SourceRoot
		}(),
		ValidationReceipt: &ArtifactValidationReceipt{Status: "passed", Checks: []string{"test"}},
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer snapshot.Close()
	target := filepath.Join(t.TempDir(), "packages", "proto", "gen")
	if err := os.MkdirAll(filepath.Join(target, "go", "old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "go", "old", "stale.go"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.MaterializeCompatibilityView(context.Background(), snapshot, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "go", "old", "stale.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale compatibility output still exists, err=%v", err)
	}
	if got, err := os.ReadFile(filepath.Join(target, "go", "demo", "demo.pb.go")); err != nil || string(got) != "selected" {
		t.Fatalf("compatibility output = %q, err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "snapshots", "selected", "validation-receipt.json")); err != nil {
		t.Fatalf("validation receipt missing: %v", err)
	}
}
