package graph_test

import (
	"testing"

	"architecture-cartographer/internal/graph"
)

func TestNormalizeInfersTestFilesFromPath(t *testing.T) {
	snap := graph.Normalize("demo", graph.RawGraph{
		Files: []graph.FileNode{
			{ID: "file:a", Path: "api/internal/a/a.go"},
			{ID: "file:go-test", Path: "api/internal/a/a_test.go"},
			{ID: "file:ts-test", Path: "ui/src/a.spec.tsx"},
		},
	})
	tests := map[string]bool{}
	for _, f := range snap.Files {
		tests[f.ID] = f.IsTest
	}
	if tests["file:a"] {
		t.Fatalf("non-test file inferred as test")
	}
	if !tests["file:go-test"] || !tests["file:ts-test"] {
		t.Fatalf("test filename inference failed: %+v", tests)
	}
}

func TestNormalizeInfersTestOnlyImportsFromTestFileSource(t *testing.T) {
	snap := graph.Normalize("demo", graph.RawGraph{
		Files: []graph.FileNode{
			{ID: "file:test", Path: "ui/src/a.test.ts", PackageID: "pkg:test"},
			{ID: "file:prod", Path: "ui/src/a.ts", PackageID: "pkg:prod"},
		},
		Imports: []graph.ImportEdge{
			{From: "file:test", ToPackageID: "pkg:prod"},
			{From: "file:prod", ToPackageID: "pkg:test"},
		},
	})
	got := map[string]bool{}
	for _, e := range snap.Imports {
		got[e.From] = e.TestOnly
	}
	if !got["file:test"] {
		t.Fatalf("test-file import should be inferred test-only: %+v", got)
	}
	if got["file:prod"] {
		t.Fatalf("prod-file import should not be inferred test-only: %+v", got)
	}
}
