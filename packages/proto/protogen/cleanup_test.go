package protogen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		filepath.Join(protoRoot, "gen", "manifests", "retired.lock.json"),
	} {
		writeCleanupFile(t, path, "stale")
	}

	plan, err := PlanCleanup(CleanupOptions{RepoRoot: root, ProtoRoot: protoRoot, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 6 {
		t.Fatalf("actions = %d, want 6: %#v", len(plan.Actions), plan.Actions)
	}
	if !strings.Contains(strings.Join(cleanupActionPaths(plan), "\n"), "gen") {
		t.Fatalf("plan did not include generated orphan paths: %#v", plan.Actions)
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
