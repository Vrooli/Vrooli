package importcluster_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/importcluster"
)

func TestScore_ClusterMembershipDrivesDomainShare(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:x", PackageID: "pkg:x"},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:x", RepoPath: "shared/x"},
			{ID: "pkg:c1", RepoPath: "internal/conflicts/a"},
			{ID: "pkg:c2", RepoPath: "internal/conflicts/b"},
			{ID: "pkg:g1", RepoPath: "internal/graph/a"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:c1", ToPackageID: "pkg:x"},
			{From: "pkg:x", ToPackageID: "pkg:c2"},
			// Disconnected node — different cluster.
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
			{Name: "graph", Paths: []string{"internal/graph/**"}},
		},
	}
	out := importcluster.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:x"})
	if len(out.Scores) != 1 {
		t.Fatalf("want 1 score (conflicts cluster), got %+v", out)
	}
	if out.Scores[0].Domain != "conflicts" {
		t.Fatalf("unexpected: %s", out.Scores[0].Domain)
	}
}

func TestScore_NoClusterAbstains(t *testing.T) {
	out := importcluster.New().Score(context.Background(),
		signals.NewGraphContext("demo", graph.GraphSnapshot{}, domains.DerivedDomainMap{}),
		graph.Chunk{FileID: "file:none"},
	)
	if len(out.Scores) != 0 {
		t.Fatalf("expected no scores, got %+v", out.Scores)
	}
	if out.Abstention == nil {
		t.Fatal("expected abstention when file has no cluster")
	}
}
