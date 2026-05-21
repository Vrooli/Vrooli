package importcluster_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/signals"
	"architecture-cartographer/internal/signals/importcluster"
)

func TestScore_ClusterMembershipDrivesDomainShare(t *testing.T) {
	snap := graph.GraphSnapshot{
		Files: []graph.FileNode{
			{ID: "file:x", PackageID: "pkg:x"},
		},
		Packages: []graph.PackageNode{
			{ID: "pkg:x", Internal: true, Directory: "shared/x"},
			{ID: "pkg:c1", Internal: true, Directory: "internal/conflicts/a"},
			{ID: "pkg:c2", Internal: true, Directory: "internal/conflicts/b"},
			{ID: "pkg:g1", Internal: true, Directory: "internal/graph/a"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:c1", ToPackageID: "pkg:x"},
			{From: "pkg:x", ToPackageID: "pkg:c2"},
			// Disconnected node — different cluster.
		},
	}
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
			{Name: "graph", Paths: []string{"internal/graph/**"}},
		},
	}
	out := importcluster.New().Score(context.Background(), signals.NewGraphContext("demo", snap, m), graph.Chunk{FileID: "file:x"})
	if len(out) != 1 {
		t.Fatalf("want 1 score (conflicts cluster), got %+v", out)
	}
	if out[0].Domain != "conflicts" {
		t.Fatalf("unexpected: %s", out[0].Domain)
	}
}

func TestScore_NoClusterReturnsEmpty(t *testing.T) {
	out := importcluster.New().Score(context.Background(),
		signals.NewGraphContext("demo", graph.GraphSnapshot{}, manifest.ManifestDefinition{}),
		graph.Chunk{FileID: "file:none"},
	)
	if len(out) != 0 {
		t.Fatalf("expected empty, got %+v", out)
	}
}
