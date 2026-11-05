package protogen

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, writableTempDir(t), "a", "first")
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "old", "old")); err != nil {
		t.Fatal(err)
	}
	store.Hooks.BeforeActiveRename = func() error { return errors.New("injected interruption") }
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "new", "new")); !errors.Is(err, ErrArtifactPublication) {
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "old", "old")); err != nil {
		t.Fatal(err)
	}
	store.Hooks.BeforeSnapshotRename = func() error { return errors.New("injected snapshot interruption") }
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "new", "new")); !errors.Is(err, ErrArtifactPublication) {
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "old", "old")); err != nil {
		t.Fatal(err)
	}
	store.Hooks.BeforeActiveRename = nil
	store.Hooks.BeforeLastGoodRename = func() error { return errors.New("injected last-good interruption") }
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "new", "new")); !errors.Is(err, ErrArtifactPublication) {
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, writableTempDir(t), "bad", "body")
	candidate.Metadata.Outputs[0].Digest = "sha256:not-the-file"
	if _, err := store.Publish(context.Background(), candidate); !errors.Is(err, ErrArtifactDigestMismatch) {
		t.Fatalf("digest error = %v, want digest mismatch", err)
	}
	candidate = testCandidate(t, writableTempDir(t), "missing", "body")
	candidate.Metadata.Outputs = append(candidate.Metadata.Outputs, ArtifactOutput{Path: "typescript/missing.ts", Digest: "sha256:missing", Size: 1})
	if _, err := store.Publish(context.Background(), candidate); !errors.Is(err, ErrArtifactIncomplete) {
		t.Fatalf("missing output error = %v, want incomplete", err)
	}
}

func TestArtifactStoreLeaseDefersGarbageCollection(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	store.Retention = time.Hour
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "old", "old")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "new", "new")); err != nil {
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	candidate := testCandidate(t, writableTempDir(t), "same", "same")
	if _, err := store.Publish(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	first, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstGeneratedAt := first.Metadata.GeneratedAt
	first.Close()
	candidate.Metadata.GeneratedAt = firstGeneratedAt.Add(time.Hour)
	if _, err := store.Publish(context.Background(), candidate); err != nil {
		t.Fatal(err)
	}
	second, err := store.Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deferredGeneratedAt := second.Metadata.GeneratedAt
	second.Close()
	if !deferredGeneratedAt.Equal(firstGeneratedAt) {
		t.Fatalf("republish changed immutable metadata timestamp = %s, want %s", deferredGeneratedAt, firstGeneratedAt)
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "old", "old")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "new", "new")); err != nil {
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
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), Candidate{
		Metadata: testCandidate(t, writableTempDir(t), "selected", "selected").Metadata,
		SourceRoot: func() string {
			candidate := testCandidate(t, writableTempDir(t), "selected", "selected")
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
	target := filepath.Join(writableTempDir(t), "packages", "proto", "gen")
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
	stamp, err := os.ReadFile(filepath.Join(target, compatibilityStampName))
	if err != nil || !strings.Contains(string(stamp), `"artifact_id": "selected"`) {
		t.Fatalf("compatibility stamp = %q, err=%v", stamp, err)
	}
	if _, err := os.Stat(filepath.Join(store.Root, "snapshots", "selected", "validation-receipt.json")); err != nil {
		t.Fatalf("validation receipt missing: %v", err)
	}
}

// writableTempDir returns a t.TempDir whose tree is made owner-writable again
// before TempDir's own cleanup runs (cleanups run LIFO), so read-only snapshot
// content never strands directories in the system temp root.
func writableTempDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Cleanup(func() {
		_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if info, infoErr := entry.Info(); infoErr == nil {
				_ = os.Chmod(path, info.Mode().Perm()|0o700)
			}
			return nil
		})
	})
	return root
}

func ageSnapshot(t *testing.T, store *Store, artifactID string, age time.Duration) {
	t.Helper()
	when := time.Now().Add(-age)
	if err := os.Chtimes(filepath.Join(store.Root, "snapshots", artifactID), when, when); err != nil {
		t.Fatal(err)
	}
}

func snapshotNames(t *testing.T, store *Store) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(store.Root, "snapshots"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

func TestArtifactStorePublishCollectsExpiredSnapshots(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b", "c"} {
		if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), id, id)); err != nil {
			t.Fatal(err)
		}
	}
	ageSnapshot(t, store, "a", 48*time.Hour)
	ageSnapshot(t, store, "b", 48*time.Hour)
	ageSnapshot(t, store, "c", 48*time.Hour)
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "d", "d")); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(snapshotNames(t, store), ","), "d"; got != want {
		t.Fatalf("snapshots after publish = %s, want %s", got, want)
	}
}

func TestArtifactStoreCollectCapsSnapshotCountButKeepsProtected(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{"s1", "s2", "s3", "s4", "s5"}
	for index, id := range ids {
		if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), id, id)); err != nil {
			t.Fatal(err)
		}
		// Distinct ages inside the retention window; s1 oldest.
		ageSnapshot(t, store, id, time.Duration(len(ids)-index)*time.Minute)
	}
	store.MaxSnapshots = 2
	lease, err := store.acquireLeaseLocked("s1")
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	if _, err := store.Collect(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	// s5 is active/last-good, s1 is leased; of the rest only the newest two remain.
	if got, want := strings.Join(snapshotNames(t, store), ","), "s1,s3,s4,s5"; got != want {
		t.Fatalf("snapshots = %s, want %s", got, want)
	}
}

func TestArtifactStoreCollectRemovesExpiredLeaseFiles(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "a", "a")); err != nil {
		t.Fatal(err)
	}
	expired, err := store.acquireLeaseLocked("a")
	if err != nil {
		t.Fatal(err)
	}
	live, err := store.acquireLeaseLocked("a")
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	expired.record.ExpiresAt = time.Now().Add(-time.Hour)
	if err := writeJSON(expired.path, expired.record); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Collect(context.Background(), time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(expired.path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expired lease stat err = %v, want removed", err)
	}
	if _, err := os.Stat(live.path); err != nil {
		t.Fatalf("live lease removed: %v", err)
	}
}

func TestArtifactStoreCollectionFailureDoesNotFailPublish(t *testing.T) {
	store, err := NewArtifactStore(filepath.Join(writableTempDir(t), "proto"))
	if err != nil {
		t.Fatal(err)
	}
	var log strings.Builder
	store.Logger = &log
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "a", "a")); err != nil {
		t.Fatal(err)
	}
	expired, err := store.acquireLeaseLocked("a")
	if err != nil {
		t.Fatal(err)
	}
	expired.record.ExpiresAt = time.Now().Add(-time.Hour)
	if err := writeJSON(expired.path, expired.record); err != nil {
		t.Fatal(err)
	}
	leaseRoot := filepath.Join(store.Root, "leases")
	if err := os.Chmod(leaseRoot, 0o555); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Publish(context.Background(), testCandidate(t, writableTempDir(t), "b", "b")); err != nil {
		t.Fatalf("publish failed because of collection: %v", err)
	}
	if !strings.Contains(log.String(), "artifact collection after publish failed") {
		t.Fatalf("log = %q, want collection warning", log.String())
	}
	status, err := store.Status(context.Background())
	if err != nil || status.Active.Artifact != "b" {
		t.Fatalf("active after publish = %+v (%v), want b", status.Active, err)
	}
}
