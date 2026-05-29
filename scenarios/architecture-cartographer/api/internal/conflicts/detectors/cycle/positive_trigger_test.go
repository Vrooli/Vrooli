package cycle_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
)

// TestPositiveTrigger_ThreeNodeCrossDomainSCC documents the canonical
// drift the cycle detector must catch: three packages in three
// different domains forming an SCC. Closes the "detector coverage
// opaque" gap (Plan Problem 3): proves the detector fires on deliberate
// cross-domain drift and emits a useful conflict shape.
func TestPositiveTrigger_ThreeNodeCrossDomainSCC(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:a", RepoPath: "internal/alpha"},
			{ID: "pkg:b", RepoPath: "internal/bravo"},
			{ID: "pkg:c", RepoPath: "internal/charlie"},
		},
		Imports: []graph.ImportEdge{
			{From: "pkg:a", ToPackageID: "pkg:b"},
			{From: "pkg:b", ToPackageID: "pkg:c"},
			{From: "pkg:c", ToPackageID: "pkg:a"},
		},
	}
	m := domains.DerivedDomainMap{
		Scenario: "demo",
		Domains: []domains.DerivedDomain{
			{Name: "alpha", Paths: []string{"internal/alpha/**"}},
			{Name: "bravo", Paths: []string{"internal/bravo/**"}},
			{Name: "charlie", Paths: []string{"internal/charlie/**"}},
		},
	}
	got, err := cycle.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo", Snapshot: snap, DomainMap: m,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 cycle conflict, got %d (%+v)", len(got), got)
	}
	c := got[0]
	if c.Type != "cycle" || c.Subtype != "cross-domain" {
		t.Fatalf("want cycle/cross-domain, got %s/%s", c.Type, c.Subtype)
	}
	if c.Severity != conflicts.SeverityError {
		t.Fatalf("cycle must be error severity, got %s", c.Severity)
	}
	if len(c.Locations) != 3 {
		t.Fatalf("want 3 locations for 3-node SCC, got %v", c.Locations)
	}
	if len(c.Domains) != 3 {
		t.Fatalf("want 3 domain tags (cross-domain), got %v", c.Domains)
	}
	if len(c.Evidence) < 1 {
		t.Fatalf("want at least one evidence row, got %+v", c.Evidence)
	}
}

// TestPositiveTrigger_TwoNodeWithinDomain documents that an in-domain
// cycle classifies as `within-domain` (not cross-domain). Catches
// regression where the classifier picks the wrong subtype.
func TestPositiveTrigger_TwoNodeWithinDomain(t *testing.T) {
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			{ID: "pkg:x", RepoPath: "internal/widgets/x"},
			{ID: "pkg:y", RepoPath: "internal/widgets/y"},
		},
		Imports: []graph.ImportEdge{
			// SymbolIDs present → classifier rules out type-only and falls
			// through to within-domain (single shared domain).
			{From: "pkg:x", ToPackageID: "pkg:y", SymbolIDs: []string{"sym:Y.Func"}},
			{From: "pkg:y", ToPackageID: "pkg:x", SymbolIDs: []string{"sym:X.Func"}},
		},
	}
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{{Name: "widgets", Paths: []string{"internal/widgets/**"}}},
	}
	got, err := cycle.New().Detect(context.Background(), conflicts.DetectInput{
		Scenario: "demo", Snapshot: snap, DomainMap: m,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 1 || got[0].Subtype != "within-domain" {
		t.Fatalf("want 1 within-domain cycle, got %+v", got)
	}
}
