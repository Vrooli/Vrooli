package importervoting_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/importervoting"
)

func TestScore_MajorityVotesForDomain(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:x", PackageID: "pkg:x"},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:x", RepoPath: "shared/x"},
			{ID: "pkg:conflicts", RepoPath: "internal/conflicts"},
			{ID: "pkg:graph", RepoPath: "internal/graph"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:conflicts", ToPackageID: "pkg:x"},
			{From: "pkg:graph", ToPackageID: "pkg:x"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
			{Name: "graph", Paths: []string{"internal/graph/**"}},
		},
	}
	gctx := signals.NewGraphContext("demo", snap, m)
	out := importervoting.New().Score(context.Background(), gctx, graph.Chunk{FileID: "file:x"})
	if len(out.Scores) != 2 {
		t.Fatalf("want 2 scores, got %+v", out)
	}
	for _, s := range out.Scores {
		if s.Value != 0.5 {
			t.Fatalf("each domain should have 0.5 vote share, got %f for %s", s.Value, s.Domain)
		}
	}
}

func TestScore_NoImportersAbstains(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files:    []graph.FileNode{{ID: "file:x", PackageID: "pkg:x"}},
		Packages: []graph.PackageNode{{ID: "pkg:x"}},
	}
	out := importervoting.New().Score(context.Background(), signals.NewGraphContext("demo", snap, domains.DerivedDomainMap{}), graph.Chunk{FileID: "file:x"})
	if len(out.Scores) != 0 {
		t.Fatalf("want 0 scores, got %+v", out.Scores)
	}
	if out.Abstention == nil {
		t.Fatal("expected abstention when no importers")
	}
}
