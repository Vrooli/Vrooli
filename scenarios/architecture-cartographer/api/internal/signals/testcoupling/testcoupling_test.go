package testcoupling_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/testcoupling"
)

func TestScore_TestFilesInDomainProduceScore(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:src", PackageID: "pkg:src"},
			{ID: "file:test", PackageID: "pkg:conflicts_test", IsTest: true},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:src", Directory: "shared/x"},
			{ID: "pkg:conflicts_test", Directory: "internal/conflicts"},
		},
		Imports: []graph.ImportEdge{
			{From: "file:test", ToPackageID: "pkg:src", TestOnly: true},
		},
	}
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	out := testcoupling.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:src"})
	if len(out) != 1 {
		t.Fatalf("want 1 score, got %+v", out)
	}
	if out[0].Domain != "conflicts" {
		t.Fatalf("unexpected domain: %s", out[0].Domain)
	}
}

func TestScore_NoTestImportersReturnsEmpty(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{{ID: "file:src", PackageID: "pkg:src"}},
		Packages: []graph.PackageNode{{ID: "pkg:src"}},
	}
	out := testcoupling.New().Score(context.Background(), signals.NewGraphContext("demo", snap, manifest.ManifestDefinition{}), graph.Chunk{FileID: "file:src"})
	if len(out) != 0 {
		t.Fatalf("want 0 scores, got %+v", out)
	}
}
