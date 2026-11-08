package protogen

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanCleanupFindsOrphanOwnerFootprintAndEmptySchemaOwner(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	writeCleanupFile(t, filepath.Join(protoRoot, "schemas", "active", "v1", "service.proto"), "syntax = \"proto3\";\n")
	if err := os.MkdirAll(filepath.Join(protoRoot, "schemas", "retired"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(protoRoot, "gen", "go", "retired", "v1", "service.pb.go"),
		filepath.Join(protoRoot, "gen", "typescript", "retired", "v1", "service_pb.ts"),
		filepath.Join(protoRoot, "gen", "python", "retired", "v1", "service_pb2.py"),
		filepath.Join(protoRoot, "gen", "python", "forgotten", "v1", "service_pb2.py"),
		filepath.Join(protoRoot, "gen", "manifests", "retired.lock.json"),
	} {
		writeCleanupFile(t, path, "stale")
	}

	plan, err := PlanCleanup(CleanupOptions{RepoRoot: root, ProtoRoot: protoRoot, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 7 {
		t.Fatalf("actions = %d, want 7: %#v", len(plan.Actions), plan.Actions)
	}
	if !strings.Contains(strings.Join(cleanupActionPaths(plan), "\n"), "gen") {
		t.Fatalf("plan did not include generated orphan paths: %#v", plan.Actions)
	}
	if !strings.Contains(strings.Join(cleanupActionPaths(plan), "\n"), "gen/python/forgotten") {
		t.Fatalf("plan did not include unowned generated output: %#v", plan.Actions)
	}
	if !strings.Contains(strings.Join(cleanupActionPaths(plan), "\n"), "schemas/retired") {
		t.Fatalf("plan did not include empty retired owner: %#v", plan.Actions)
	}
}

func TestPlanCleanupKeepsActiveOwnerAndSupportsScenarioFilter(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	writeCleanupFile(t, filepath.Join(protoRoot, "schemas", "active", "v1", "service.proto"), "syntax = \"proto3\";\n")
	writeCleanupFile(t, filepath.Join(protoRoot, "schemas", "retired", "v1", "service.proto"), "syntax = \"proto3\";\n")
	writeCleanupFile(t, filepath.Join(protoRoot, "gen", "go", "retired", "v1", "service.pb.go"), "stale")
	writeCleanupFile(t, filepath.Join(protoRoot, "gen", "go", "other", "v1", "service.pb.go"), "stale")

	plan, err := PlanCleanup(CleanupOptions{RepoRoot: root, ProtoRoot: protoRoot, Scenarios: []string{"retired"}, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, action := range plan.Actions {
		if action.Owner != "retired" && action.Kind != "regenerate_generated_tree" {
			t.Fatalf("unselected action survived filter: %#v", action)
		}
	}
	if strings.Contains(strings.Join(cleanupActionPaths(plan), "\n"), "gen/go/other") {
		t.Fatalf("scenario filter leaked other owner: %#v", plan.Actions)
	}
}

func TestRemoveEmptyDirectoryTreePreservesFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "retired", "v1")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := removeEmptyDirectoryTree(filepath.Dir(root)); err != nil {
		t.Fatalf("remove empty directory tree: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(root)); !os.IsNotExist(err) {
		t.Fatalf("empty directory tree still exists, stat error = %v", err)
	}

	root = filepath.Join(t.TempDir(), "retired", "v1")
	writeCleanupFile(t, filepath.Join(root, "keep.proto"), "syntax = \"proto3\";\n")
	if err := removeEmptyDirectoryTree(filepath.Dir(root)); err == nil {
		t.Fatal("remove empty directory tree succeeded despite source file")
	}
	if _, err := os.Stat(filepath.Join(root, "keep.proto")); err != nil {
		t.Fatalf("source file was not preserved: %v", err)
	}
}

func writeCleanupFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func cleanupActionPaths(plan CleanupPlan) []string {
	paths := make([]string, 0, len(plan.Actions))
	for _, action := range plan.Actions {
		paths = append(paths, action.Path)
	}
	return paths
}

func TestCleanupRemovesOnlyOrphanedInTreeStagingTreesWithoutRegenerating(t *testing.T) {
	root := t.TempDir()
	protoRoot := filepath.Join(root, "packages", "proto")
	writeCleanupFile(t, filepath.Join(protoRoot, "schemas", "active", "v1", "service.proto"), "syntax = \"proto3\";\n")
	legacyVerify := writeStageTree(t, filepath.Join(root, "packages"), ".proto-gen-verify-123", unownedStageGrace+time.Hour)
	legacyView := writeStageTree(t, protoRoot, ".proto-compatibility-456", unownedStageGrace+time.Hour)
	live, err := createStage(filepath.Join(root, "packages"), generationStagePrefix)
	if err != nil {
		t.Fatal(err)
	}
	defer live.Remove()

	options := CleanupOptions{RepoRoot: root, ProtoRoot: protoRoot}
	plan, err := PlanCleanup(options)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"packages/.proto-gen-verify-123", "packages/proto/.proto-compatibility-456"}
	if got := cleanupActionPaths(plan); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("planned paths = %v, want %v (live tree excluded, no regeneration)", got, want)
	}
	result, err := ApplyCleanup(context.Background(), options, plan, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || result.GenerationRan {
		t.Fatalf("result applied=%t generation=%t, want applied without generation", result.Applied, result.GenerationRan)
	}
	for _, path := range []string{legacyVerify, legacyView} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("orphaned staging tree %s survived apply: %v", path, err)
		}
	}
	if _, err := os.Stat(live.Dir); err != nil {
		t.Fatalf("apply removed a live staging tree: %v", err)
	}
}
