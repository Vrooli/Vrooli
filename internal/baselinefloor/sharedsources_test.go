package baselinefloor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A frozen root keeps the generated outputs the serving build consumes
// (nothing regenerates a frozen root) and drops installed dependencies.
func TestCaptureSharedSourcesKeepsGeneratedOutputsAndDropsDependencies(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "packages/proto/gen/typescript/demo.ts", "export {}", 0o644)
	writeFile(t, repo, "packages/proto/gen/.vrooli-proto-artifact.json", `{"artifact_id":"abc"}`, 0o644)
	writeFile(t, repo, "packages/rcl/dist/index.js", "export {}", 0o644)
	writeFile(t, repo, "packages/rcl/node_modules/react/index.js", "x", 0o644)
	writeFile(t, repo, "packages/rcl/.vrooli/package.json", "{}", 0o644)
	writeFile(t, repo, "go.mod", "module example.com/root\n", 0o644)
	dest := filepath.Join(t.TempDir(), "shared-sources")

	roots, total, err := CaptureSharedSources(repo, dest, []SharedRoot{
		{Path: "packages/rcl", Reason: "file:@vrooli/rcl"},
		{Path: "packages/proto/gen", Reason: "alias:ui/vite.config.ts"},
		{Path: "go.mod", Reason: "go:module ."},
	}, false)
	if err != nil {
		t.Fatalf("CaptureSharedSources: %v", err)
	}
	if len(roots) != 3 || total == 0 {
		t.Fatalf("roots=%+v total=%d", roots, total)
	}
	for _, kept := range []string{"packages/proto/gen/typescript/demo.ts", "packages/proto/gen/.vrooli-proto-artifact.json", "packages/rcl/dist/index.js", "packages/rcl/.vrooli/package.json", "go.mod"} {
		if _, err := os.Stat(filepath.Join(dest, kept)); err != nil {
			t.Errorf("snapshot lacks %s: %v", kept, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "packages/rcl/node_modules")); !os.IsNotExist(err) {
		t.Errorf("snapshot holds node_modules (err=%v)", err)
	}

	// A recapture replaces the snapshot as a whole.
	writeFile(t, repo, "packages/rcl/dist/index.js", "export const v2 = 1", 0o644)
	if _, _, err := CaptureSharedSources(repo, dest, []SharedRoot{{Path: "packages/rcl", Reason: "file:@vrooli/rcl"}}, false); err != nil {
		t.Fatalf("recapture: %v", err)
	}
	if got := readFile(t, filepath.Join(dest, "packages/rcl/dist/index.js")); got != "export const v2 = 1" {
		t.Errorf("recapture kept stale output %q", got)
	}
	if _, err := os.Stat(filepath.Join(dest, "go.mod")); !os.IsNotExist(err) {
		t.Errorf("recapture kept a root the new capture does not freeze (err=%v)", err)
	}
}

// A failed capture leaves the previous snapshot in place.
func TestCaptureSharedSourcesFailureKeepsPreviousSnapshot(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "packages/a/a.go", "package a", 0o644)
	dest := filepath.Join(t.TempDir(), "shared-sources")
	if _, _, err := CaptureSharedSources(repo, dest, []SharedRoot{{Path: "packages/a"}}, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CaptureSharedSources(repo, dest, []SharedRoot{{Path: "packages/missing"}}, false); err == nil {
		t.Fatal("expected an error for a missing root")
	}
	if _, err := os.Stat(filepath.Join(dest, "packages/a/a.go")); err != nil {
		t.Fatalf("failed capture removed the previous snapshot: %v", err)
	}
	for _, bad := range []string{"", ".", "..", "../x", "/abs"} {
		if _, _, err := CaptureSharedSources(repo, dest, []SharedRoot{{Path: bad}}, false); err == nil {
			t.Errorf("root %q: expected error", bad)
		}
	}
}

func TestCaptureRecordRoundTripAndLegacyEngagement(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, ok, err := store.ReadCaptureRecord("demo", "goal"); err != nil || ok {
		t.Fatalf("legacy engagement must read as no record: ok=%t err=%v", ok, err)
	}
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	record := CaptureRecord{CaptureID: NewCaptureID(now), CapturedAt: now, SharedSources: SharedSources{Status: SharedSourcesFrozen, Roots: []SharedRoot{{Path: "go.mod"}}}}
	if err := store.WriteCaptureRecord("demo", "goal", record); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.ReadCaptureRecord("demo", "goal")
	if err != nil || !ok || got.CaptureID != record.CaptureID || !got.SharedSources.Frozen() {
		t.Fatalf("round trip = %+v ok=%t err=%v", got, ok, err)
	}
	if NewCaptureID(now) == record.CaptureID {
		t.Fatal("capture IDs must be unique")
	}
}

// Store.Clean removes an engagement even when a copied module left
// read-only directories behind.
func TestStoreCleanRemovesReadOnlyTrees(t *testing.T) {
	store := NewStore(t.TempDir())
	ro := filepath.Join(store.SharedSourcesPath("demo", "goal"), "packages", "mod")
	writeFile(t, ro, "x.go", "package x", 0o444)
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	if err := store.Clean("demo", "goal"); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if _, err := os.Stat(store.EngagementDir("demo", "goal")); !os.IsNotExist(err) {
		t.Fatalf("engagement remains (err=%v)", err)
	}
}
