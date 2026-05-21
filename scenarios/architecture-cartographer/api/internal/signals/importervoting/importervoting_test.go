package importervoting_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/importervoting"
)

func TestScore_MajorityVotesForDomain(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:x", PackageID: "pkg:x"},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:x", Directory: "shared/x"},
			{ID: "pkg:conflicts", Directory: "internal/conflicts"},
			{ID: "pkg:graph", Directory: "internal/graph"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:conflicts", ToPackageID: "pkg:x"},
			{From: "pkg:graph", ToPackageID: "pkg:x"},
		},
	}
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
			{Name: "graph", Paths: []string{"internal/graph/**"}},
		},
	}
	gctx := signals.NewGraphContext("demo", snap, m)
	out := importervoting.New().Score(context.Background(), gctx, graph.Chunk{FileID: "file:x"})
	if len(out) != 2 {
		t.Fatalf("want 2 scores, got %+v", out)
	}
	for _, s := range out {
		if s.Value != 0.5 {
			t.Fatalf("each domain should have 0.5 vote share, got %f for %s", s.Value, s.Domain)
		}
	}
}

func TestScore_NoImportersReturnsEmpty(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files:    []graph.FileNode{{ID: "file:x", PackageID: "pkg:x"}},
		Packages: []graph.PackageNode{{ID: "pkg:x"}},
	}
	out := importervoting.New().Score(context.Background(), signals.NewGraphContext("demo", snap, manifest.ManifestDefinition{}), graph.Chunk{FileID: "file:x"})
	if len(out) != 0 {
		t.Fatalf("want 0 scores, got %+v", out)
	}
}
