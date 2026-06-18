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

func TestScore_LouvainSplitsConnectedGraphIntoCommunities(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:a1", Path: "internal/alpha/a1.go", PackageID: "pkg:a1"},
			{ID: "file:b1", Path: "internal/beta/b1.go", PackageID: "pkg:b1"},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:a1", RepoPath: "internal/alpha/a1"},
			{ID: "pkg:a2", RepoPath: "internal/alpha/a2"},
			{ID: "pkg:a3", RepoPath: "internal/alpha/a3"},
			{ID: "pkg:b1", RepoPath: "internal/beta/b1"},
			{ID: "pkg:b2", RepoPath: "internal/beta/b2"},
			{ID: "pkg:b3", RepoPath: "internal/beta/b3"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a1", ToPackageID: "pkg:a2"},
			{From: "pkg:a1", ToPackageID: "pkg:a3"},
			{From: "pkg:a2", ToPackageID: "pkg:a3"},
			{From: "pkg:b1", ToPackageID: "pkg:b2"},
			{From: "pkg:b1", ToPackageID: "pkg:b3"},
			{From: "pkg:b2", ToPackageID: "pkg:b3"},
			// The graph is one connected component, but modularity should
			// keep the two dense domain communities separate.
			{From: "pkg:a3", ToPackageID: "pkg:b3"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "alpha", Paths: []string{"internal/alpha/**"}},
			{Name: "beta", Paths: []string{"internal/beta/**"}},
		},
	}

	alpha := importcluster.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:a1"})
	if len(alpha.Scores) != 1 {
		t.Fatalf("want alpha community score only, got %+v", alpha.Scores)
	}
	if alpha.Scores[0].Domain != "alpha" || alpha.Scores[0].Value != 1 {
		t.Fatalf("alpha score = %+v, want pure alpha community", alpha.Scores[0])
	}

	beta := importcluster.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:b1"})
	if len(beta.Scores) != 1 {
		t.Fatalf("want beta community score only, got %+v", beta.Scores)
	}
	if beta.Scores[0].Domain != "beta" || beta.Scores[0].Value != 1 {
		t.Fatalf("beta score = %+v, want pure beta community", beta.Scores[0])
	}
}

func TestScore_LouvainCommunitiesAreDeterministic(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{{ID: "file:a1", Path: "internal/alpha/a1.go", PackageID: "pkg:a1"}},
		Packages: []graph.PackageNode{
			{ID: "pkg:a1", RepoPath: "internal/alpha/a1"},
			{ID: "pkg:a2", RepoPath: "internal/alpha/a2"},
			{ID: "pkg:a3", RepoPath: "internal/alpha/a3"},
			{ID: "pkg:b1", RepoPath: "internal/beta/b1"},
			{ID: "pkg:b2", RepoPath: "internal/beta/b2"},
			{ID: "pkg:b3", RepoPath: "internal/beta/b3"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a3", ToPackageID: "pkg:b3"},
			{From: "pkg:b2", ToPackageID: "pkg:b3"},
			{From: "pkg:a2", ToPackageID: "pkg:a3"},
			{From: "pkg:b1", ToPackageID: "pkg:b3"},
			{From: "pkg:a1", ToPackageID: "pkg:a3"},
			{From: "pkg:b1", ToPackageID: "pkg:b2"},
			{From: "pkg:a1", ToPackageID: "pkg:a2"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "alpha", Paths: []string{"internal/alpha/**"}},
			{Name: "beta", Paths: []string{"internal/beta/**"}},
		},
	}
	var first []signals.Score
	for i := 0; i < 25; i++ {
		got := importcluster.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:a1"}).Scores
		if i == 0 {
			first = got
			continue
		}
		if len(got) != len(first) {
			t.Fatalf("run %d score count = %d, want %d", i, len(got), len(first))
		}
		for j := range got {
			if got[j].Domain != first[j].Domain || got[j].Value != first[j].Value || got[j].Reason != first[j].Reason {
				t.Fatalf("run %d score[%d] = %+v, want %+v", i, j, got[j], first[j])
			}
		}
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
