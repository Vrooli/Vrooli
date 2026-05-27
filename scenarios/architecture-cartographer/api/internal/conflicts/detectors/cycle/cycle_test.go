package cycle_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

func detectInput(snap graph.GraphSnapshot, m domains.DerivedDomainMap) conflicts.DetectInput {
	return conflicts.DetectInput{
		Scenario:  "demo",
		Snapshot:  snap,
		DomainMap: m,
	}
}

func TestDetect_NoCycleReturnsNoConflict(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:a", Internal: true, Directory: "a"},
			{ID: "pkg:b", Internal: true, Directory: "b"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a", ToPackageID: "pkg:b"},
		},
	}
	got, err := cycle.New().Detect(context.Background(), detectInput(snap, domains.DerivedDomainMap{}))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no conflicts, got %d (%+v)", len(got), got)
	}
}

func TestDetect_TwoPackageCycleEmitsOneConflict(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:a", Internal: true, Directory: "internal/a"},
			{ID: "pkg:b", Internal: true, Directory: "internal/b"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a", ToPackageID: "pkg:b"},
			{From: "pkg:b", ToPackageID: "pkg:a"},
		},
	}
	got, err := cycle.New().Detect(context.Background(), detectInput(snap, domains.DerivedDomainMap{}))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(got))
	}
	if got[0].Type != "cycle" || got[0].Detector != "cycle" {
		t.Fatalf("unexpected: %+v", got[0])
	}
	if len(got[0].SuggestedFixes) == 0 {
		t.Fatal("expected at least one suggested fix")
	}
}

func TestDetect_CrossDomainSubtype(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:a", Internal: true, Directory: "internal/graph"},
			{ID: "pkg:b", Internal: true, Directory: "internal/conflicts"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a", ToPackageID: "pkg:b"},
			{From: "pkg:b", ToPackageID: "pkg:a"},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"internal/graph/**"}},
			{Name: "conflicts", Paths: []string{"internal/conflicts/**"}},
		},
	}
	got, _ := cycle.New().Detect(context.Background(), detectInput(snap, m))
	if got[0].Subtype != "cross-domain" {
		t.Fatalf("want cross-domain, got %q", got[0].Subtype)
	}
	if len(got[0].Domains) != 2 {
		t.Fatalf("want 2 domains, got %v", got[0].Domains)
	}
}

func TestDetect_ExternalEdgesIgnored(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:a", Internal: true},
			{ID: "pkg:ext", Internal: false},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a", ToPackageID: "pkg:ext"},
			{From: "pkg:ext", ToPackageID: "pkg:a"},
		},
	}
	got, _ := cycle.New().Detect(context.Background(), detectInput(snap, domains.DerivedDomainMap{}))
	if len(got) != 0 {
		t.Fatalf("external cycle should be ignored, got %+v", got)
	}
}

func TestDetect_DeterministicOrder(t *testing.T) {
	// Two independent cycles; result ordering should be stable.
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:a", Internal: true, Directory: "a"},
			{ID: "pkg:b", Internal: true, Directory: "b"},
			{ID: "pkg:c", Internal: true, Directory: "c"},
			{ID: "pkg:d", Internal: true, Directory: "d"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a", ToPackageID: "pkg:b"},
			{From: "pkg:b", ToPackageID: "pkg:a"},
			{From: "pkg:c", ToPackageID: "pkg:d"},
			{From: "pkg:d", ToPackageID: "pkg:c"},
		},
	}
	first, _ := cycle.New().Detect(context.Background(), detectInput(snap, domains.DerivedDomainMap{}))
	second, _ := cycle.New().Detect(context.Background(), detectInput(snap, domains.DerivedDomainMap{}))
	if len(first) != len(second) {
		t.Fatalf("non-deterministic conflict count: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Locations[0] != second[i].Locations[0] {
			t.Fatalf("order drift at i=%d: %q vs %q", i, first[i].Locations[0], second[i].Locations[0])
		}
	}
}
