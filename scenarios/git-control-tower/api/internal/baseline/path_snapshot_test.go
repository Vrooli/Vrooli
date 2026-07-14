package baseline

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func initSnapshotGitRepo(t *testing.T, root string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", root, "init", "--quiet").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
}

func TestPathSnapshotCapturesDirtyTextAndExcludesSensitiveContent(t *testing.T) {
	root := t.TempDir()
	initSnapshotGitRepo(t, root)
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
	if len(snapshot.Entries) != 1 || snapshot.Entries[0].ContentRef != "" || len(objects) != 0 {
		t.Fatalf("snapshot = %#v objects=%#v", snapshot, objects)
	}
	if _, _, err := CapturePathSnapshot(root, "bad", "agi", []string{".git/**"}, time.Now()); err == nil {
		t.Fatal("sensitive selection accepted")
	}
}

func TestDiffPathSnapshotsLabelsSourceEvidenceOnly(t *testing.T) {
	root := t.TempDir()
	initSnapshotGitRepo(t, root)
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
	initSnapshotGitRepo(t, root)
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

func TestEstimatePathSnapshotUsesGitCandidatesAndFlagsBroadScopes(t *testing.T) { // [REQ:GCT-SOURCE-EVIDENCE-001]
	root := t.TempDir()
	initSnapshotGitRepo(t, root)
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("node_modules/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "safe", "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "safe", "main.go"), []byte("package safe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scenarios", "safe", "node_modules", "large.js"), []byte("ignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", "proto", "gen", "go", "safe"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages", "proto", "gen", "go", "safe", "x.pb.go"), []byte("generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "packages", "proto", "gen", "manifests"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packages", "proto", "gen", "manifests", "safe.lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	estimate, err := EstimatePathSnapshot(root, []string{"scenarios/safe/**"}, PathSnapshotPolicy{})
	if err != nil {
		t.Fatal(err)
	}
	if estimate.EligibleFiles != 1 || estimate.ExcludedIgnoredFiles != 1 || estimate.RequiresRepair() {
		t.Fatalf("safe estimate = %#v", estimate)
	}
	withIgnored, err := EstimatePathSnapshot(root, []string{"scenarios/safe/**"}, PathSnapshotPolicy{IncludeIgnored: true})
	if err != nil || withIgnored.EligibleFiles != 2 {
		t.Fatalf("ignored opt-in = %#v err=%v", withIgnored, err)
	}
	broad, err := EstimatePathSnapshot(root, []string{"packages/proto/gen/**"}, PathSnapshotPolicy{})
	if err != nil || !broad.RequiresRepair() || broad.Issues[0].Code != "generated_output_too_broad" || len(broad.Recommendations) != 2 || broad.Recommendations[0].Selection != "packages/proto/gen/go/safe/**" || broad.Recommendations[1].Selection != "packages/proto/gen/manifests/safe.lock.json" {
		t.Fatalf("broad estimate = %#v err=%v", broad, err)
	}
	allScenarios, err := EstimatePathSnapshot(root, []string{"scenarios/**"}, PathSnapshotPolicy{})
	if err != nil || !allScenarios.RequiresRepair() || len(allScenarios.Recommendations) != 1 || allScenarios.Recommendations[0].Selection != "scenarios/safe/**" {
		t.Fatalf("all scenarios estimate = %#v err=%v", allScenarios, err)
	}
	_, _, err = CapturePathSnapshot(root, "broad", "agi", []string{"packages/proto/gen/**"}, time.Now())
	var policyErr *PathSnapshotPolicyError
	if !errors.As(err, &policyErr) || policyErr.Estimate.Issues[0].Code != "generated_output_too_broad" {
		t.Fatalf("capture error = %#v", err)
	}
}

func TestRetainedContentMustFitQuotaWhileMetadataDoesNot(t *testing.T) { // [REQ:GCT-SOURCE-EVIDENCE-001]
	root := t.TempDir()
	initSnapshotGitRepo(t, root)
	for i := 0; i < 9; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f-%d.txt", i)), bytes.Repeat([]byte("x"), maxSnapshotFileBytes), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	metadata, err := EstimatePathSnapshot(root, []string{"*.txt"}, PathSnapshotPolicy{})
	if err != nil || metadata.RequiresRepair() {
		t.Fatalf("metadata estimate = %#v err=%v", metadata, err)
	}
	retained, err := EstimatePathSnapshot(root, []string{"*.txt"}, PathSnapshotPolicy{RetainContent: true})
	if err != nil || !retained.RequiresRepair() || len(retained.Recommendations) == 0 {
		t.Fatalf("retained estimate = %#v err=%v", retained, err)
	}
	snapshot, _, err := CapturePathSnapshot(root, "metadata", "agi", []string{"*.txt"}, time.Now())
	if err != nil || snapshot.PolicyVersion != PathSnapshotPolicyVersion {
		t.Fatalf("metadata snapshot policy version = %#v err=%v", snapshot, err)
	}
}
