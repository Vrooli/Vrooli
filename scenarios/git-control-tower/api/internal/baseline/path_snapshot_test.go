package baseline

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPathSnapshotCapturesDirtyTextAndExcludesSensitiveContent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "dirty.go"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, objects, err := CapturePathSnapshot(root, "before", "agi", []string{"pkg/**"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 1 || snapshot.Entries[0].ContentRef == "" || string(objects[snapshot.Entries[0].ContentRef]) != "before\n" {
		t.Fatalf("snapshot = %#v objects=%#v", snapshot, objects)
	}
	if _, _, err := CapturePathSnapshot(root, "bad", "agi", []string{".git/**"}, time.Now()); err == nil {
		t.Fatal("sensitive selection accepted")
	}
}

func TestDiffPathSnapshotsLabelsSourceEvidenceOnly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _, err := CapturePathSnapshot(root, "before", "agi", []string{"*.txt"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, _, err := CapturePathSnapshot(root, "after", "agi", []string{"*.txt"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	deltas := DiffPathSnapshots(before, after)
	if len(deltas) != 1 || deltas[0].Status != "modified" {
		t.Fatalf("deltas = %#v", deltas)
	}
}

func TestDiffPathSnapshotsReportsUnambiguousRenames(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.txt")
	if err := os.WriteFile(oldPath, []byte("same bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, _, err := CapturePathSnapshot(root, "before", "agi", []string{"*.txt"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, filepath.Join(root, "new.txt")); err != nil {
		t.Fatal(err)
	}
	after, _, err := CapturePathSnapshot(root, "after", "agi", []string{"*.txt"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	deltas := DiffPathSnapshots(before, after)
	if len(deltas) != 1 || deltas[0].Status != "renamed" || deltas[0].Path != "new.txt" || deltas[0].Before.Path != "old.txt" {
		t.Fatalf("deltas = %#v", deltas)
	}
}

func TestFilterSourceDeltasKeepsPhasePathsAndRenameSources(t *testing.T) {
	deltas := []SourceDelta{{Path: "scenarios/foo/new.go", Status: "renamed", Before: &PathEntry{Path: "scenarios/bar/old.go"}}, {Path: "packages/proto/x.go", Status: "modified"}}
	filtered, err := FilterSourceDeltas(deltas, []string{"scenarios/bar/**"})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].Path != "scenarios/foo/new.go" {
		t.Fatalf("filtered deltas = %#v", filtered)
	}
}
