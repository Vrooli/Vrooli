package baseline

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func snapshotFixturePaths(paths ...string) snapshotPathSet {
	return func(_ context.Context, _ string, _ []string, args ...string) (map[string]struct{}, error) {
		result := map[string]struct{}{}
		for _, path := range paths {
			result[path] = struct{}{}
		}
		return result, nil
	}
}

func snapshotFixturePathSet(root string) snapshotPathSet {
	return func(ctx context.Context, _ string, patterns []string, args ...string) (map[string]struct{}, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		wantIgnored := false
		for _, arg := range args {
			if arg == "--ignored" {
				wantIgnored = true
			}
		}
		out := map[string]struct{}{}
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil || rel == "." {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if rel == ".git" || strings.HasPrefix(rel, ".git/") {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			ignored := strings.Contains(rel, "/node_modules/") || strings.HasPrefix(rel, "node_modules/")
			if ignored != wantIgnored || !matchesPath(patterns, rel) {
				return nil
			}
			out[rel] = struct{}{}
			return nil
		})
		return out, err
	}
}

func TestPathSnapshotCapturesDirtyTextAndExcludesSensitiveContent(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "dirty.go"), []byte("before\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot, objects, err := capturePathSnapshotWithPathSet(root, "before", "agi", []string{"pkg/**"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, snapshotFixturePaths("pkg/dirty.go"))
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
	path := filepath.Join(root, "file.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	fixture := snapshotFixturePaths("file.txt")
	before, _, err := capturePathSnapshotWithPathSet(root, "before", "agi", []string{"*.txt"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, _, err := capturePathSnapshotWithPathSet(root, "after", "agi", []string{"*.txt"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, fixture)
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
	fixture := snapshotFixturePaths("old.txt")
	before, _, err := capturePathSnapshotWithPathSet(root, "before", "agi", []string{"*.txt"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, filepath.Join(root, "new.txt")); err != nil {
		t.Fatal(err)
	}
	fixture = snapshotFixturePaths("new.txt")
	after, _, err := capturePathSnapshotWithPathSet(root, "after", "agi", []string{"*.txt"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, fixture)
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
	estimate, err := estimatePathSnapshotContext(context.Background(), root, []string{"scenarios/safe/**"}, PathSnapshotPolicy{}, snapshotFixturePathSet(root))
	if err != nil {
		t.Fatal(err)
	}
	if estimate.EligibleFiles != 1 || estimate.ExcludedIgnoredFiles != 1 || estimate.RequiresRepair() {
		t.Fatalf("safe estimate = %#v", estimate)
	}
	withIgnored, err := estimatePathSnapshotContext(context.Background(), root, []string{"scenarios/safe/**"}, PathSnapshotPolicy{IncludeIgnored: true}, snapshotFixturePathSet(root))
	if err != nil || withIgnored.EligibleFiles != 2 {
		t.Fatalf("ignored opt-in = %#v err=%v", withIgnored, err)
	}
	broad, err := estimatePathSnapshotContext(context.Background(), root, []string{"packages/proto/gen/**"}, PathSnapshotPolicy{}, snapshotFixturePathSet(root))
	if err != nil || !broad.RequiresRepair() || broad.Issues[0].Code != "generated_output_too_broad" || len(broad.Recommendations) != 2 || broad.Recommendations[0].Selection != "packages/proto/gen/go/safe/**" || broad.Recommendations[1].Selection != "packages/proto/gen/manifests/safe.lock.json" {
		t.Fatalf("broad estimate = %#v err=%v", broad, err)
	}
	allScenarios, err := estimatePathSnapshotContext(context.Background(), root, []string{"scenarios/**"}, PathSnapshotPolicy{}, snapshotFixturePathSet(root))
	if err != nil || !allScenarios.RequiresRepair() || len(allScenarios.Recommendations) != 1 || allScenarios.Recommendations[0].Selection != "scenarios/safe/**" {
		t.Fatalf("all scenarios estimate = %#v err=%v", allScenarios, err)
	}
	_, _, err = capturePathSnapshotWithPathSet(root, "broad", "agi", []string{"packages/proto/gen/**"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, snapshotFixturePathSet(root))
	var policyErr *PathSnapshotPolicyError
	if !errors.As(err, &policyErr) || policyErr.Estimate.Issues[0].Code != "generated_output_too_broad" {
		t.Fatalf("capture error = %#v", err)
	}
}

func TestEstimatePathSnapshotHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := estimatePathSnapshotContext(ctx, root, []string{"**"}, PathSnapshotPolicy{}, snapshotFixturePathSet(root))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled estimate error = %v, want context.Canceled", err)
	}
}

func TestRetainedContentMustFitQuotaWhileMetadataDoesNot(t *testing.T) { // [REQ:GCT-SOURCE-EVIDENCE-001]
	root := t.TempDir()
	for i := 0; i < 9; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f-%d.txt", i)), bytes.Repeat([]byte("x"), maxSnapshotFileBytes), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	metadata, err := estimatePathSnapshotContext(context.Background(), root, []string{"*.txt"}, PathSnapshotPolicy{}, snapshotFixturePathSet(root))
	if err != nil || metadata.RequiresRepair() {
		t.Fatalf("metadata estimate = %#v err=%v", metadata, err)
	}
	retained, err := estimatePathSnapshotContext(context.Background(), root, []string{"*.txt"}, PathSnapshotPolicy{RetainContent: true}, snapshotFixturePathSet(root))
	if err != nil || !retained.RequiresRepair() || len(retained.Recommendations) == 0 {
		t.Fatalf("retained estimate = %#v err=%v", retained, err)
	}
	snapshot, _, err := capturePathSnapshotWithPathSet(root, "metadata", "agi", []string{"*.txt"}, PathSnapshotPolicy{}, time.Now(), defaultPathSnapshotLease, snapshotFixturePathSet(root))
	if err != nil || snapshot.PolicyVersion != PathSnapshotPolicyVersion {
		t.Fatalf("metadata snapshot policy version = %#v err=%v", snapshot, err)
	}
}
