package testcoupling_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
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
			{ID: "pkg:src", RepoPath: "shared/x"},
			{ID: "pkg:conflicts_test", RepoPath: "internal/conflicts"},
		},
		Imports: []graph.ImportEdge{
			{From: "file:test", ToPackageID: "pkg:src", TestOnly: true},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	out := testcoupling.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:src"})
	if len(out.Scores) != 1 {
		t.Fatalf("want 1 score, got %+v", out)
	}
	if out.Scores[0].Domain != "conflicts" {
		t.Fatalf("unexpected domain: %s", out.Scores[0].Domain)
	}
}

func TestScore_NoTestImportersAbstains(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files:    []graph.FileNode{{ID: "file:src", PackageID: "pkg:src"}},
		Packages: []graph.PackageNode{{ID: "pkg:src"}},
	}
	out := testcoupling.New().Score(context.Background(), signals.NewGraphContext("demo", snap, domains.DerivedDomainMap{}), graph.Chunk{FileID: "file:src"})
	if len(out.Scores) != 0 {
		t.Fatalf("want 0 scores, got %+v", out.Scores)
	}
	if out.Abstention == nil {
		t.Fatal("expected abstention when no test importers")
	}
}
