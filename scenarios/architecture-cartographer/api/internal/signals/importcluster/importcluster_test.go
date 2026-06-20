package importcluster_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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

// TestScore_DenseGraphCompletesQuickly is the performance-regression
// guard for the import-cluster DoS. The pre-fix Louvain recomputed global
// O(N²) modularity once per node plus once per candidate move every sweep,
// so a few hundred densely-connected packages pegged a core for minutes.
// With incremental ΔQ this is sub-second; bounding it under a 15s deadline
// (Score abstains with a context error if clustering overruns) keeps the
// assertion robust against CI jitter while still catching an O(N³) regression.
func TestScore_DenseGraphCompletesQuickly(t *testing.T) {
	const n = 400
	snap := graph.GraphSnapshot{}
	for i := 0; i < n; i++ {
		pkg := fmt.Sprintf("pkg:%03d", i)
		file := fmt.Sprintf("file:%03d", i)
		dom := "alpha"
		if i%2 == 1 {
			dom = "beta"
		}
		snap.Files = append(snap.Files, graph.FileNode{ID: file, Path: fmt.Sprintf("internal/%s/p%03d.go", dom, i), PackageID: pkg})
		snap.Packages = append(snap.Packages, graph.PackageNode{ID: pkg, RepoPath: fmt.Sprintf("internal/%s/p%03d", dom, i)})
	}
	// Dense edges (~60% of pairs) — the worst case for the old recompute.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if (i*7+j*3)%5 < 3 {
				snap.Imports = append(snap.Imports, graph.ImportEdge{From: fmt.Sprintf("pkg:%03d", i), ToPackageID: fmt.Sprintf("pkg:%03d", j)})
			}
		}
	}
	m := domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
		{Name: "alpha", Paths: []string{"internal/alpha/**"}},
		{Name: "beta", Paths: []string{"internal/beta/**"}},
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	start := time.Now()
	out := importcluster.New().Score(ctx, signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:000"})
	elapsed := time.Since(start)
	if ctx.Err() != nil {
		t.Fatalf("clustering overran the deadline (%s) — O(N²)/sweep regression?", elapsed)
	}
	if len(out.Scores) == 0 {
		t.Fatalf("expected a cluster score for a densely connected package, got abstention: %+v", out.Abstention)
	}
	t.Logf("N=%d dense clustering in %s", n, elapsed)
}

// TestScore_IsolatedPackageGetsSingletonCluster guards the isolated-node
// fast path: a package with no internal import edges is excluded from the
// expensive Louvain pass but still receives its own singleton cluster, so
// its file scores its own domain rather than abstaining.
func TestScore_IsolatedPackageGetsSingletonCluster(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:lonely", Path: "internal/solo/s.go", PackageID: "pkg:solo"},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:solo", RepoPath: "internal/solo/s"},
			{ID: "pkg:c1", RepoPath: "internal/conflicts/a"},
			{ID: "pkg:c2", RepoPath: "internal/conflicts/b"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:c1", ToPackageID: "pkg:c2"}, // pkg:solo participates in no edge
		},
	}
	m := domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
		{Name: "solo", Paths: []string{"internal/solo/**"}},
		{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
	}}
	out := importcluster.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:lonely"})
	if len(out.Scores) != 1 {
		t.Fatalf("want 1 singleton-cluster score, got %+v (abstention=%+v)", out.Scores, out.Abstention)
	}
	if out.Scores[0].Domain != "solo" || out.Scores[0].Value != 1 {
		t.Fatalf("isolated package should score its own domain at full weight, got %+v", out.Scores[0])
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

func TestScore_CanceledContextAbstains(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := importcluster.New().Score(ctx,
		signals.NewGraphContext("demo", graph.GraphSnapshot{}, domains.DerivedDomainMap{}),
		graph.Chunk{FileID: "file:none", Path: "none.go"},
	)
	if len(out.Scores) != 0 {
		t.Fatalf("canceled context must not produce scores, got %+v", out.Scores)
	}
	if out.Abstention == nil {
		t.Fatal("expected cancellation abstention")
	}
}
