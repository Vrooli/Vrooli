package couplingsmell_test

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/couplingsmell"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals/boundaries"
)

func godDomainInput() conflicts.DetectInput {
	pkg := func(id, dir string) graph.PackageNode {
		return graph.PackageNode{ID: id, RepoPath: dir}
	}
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			pkg("p-a", "api/internal/a"),
			pkg("p-b", "api/internal/b"),
			pkg("p-c", "api/internal/c"),
			pkg("p-d", "api/internal/d"),
		},
		Imports: []graph.ImportEdge{
			{From: "p-a", ToPackageID: "p-b"},
			{From: "p-a", ToPackageID: "p-c"},
			{From: "p-a", ToPackageID: "p-d"},
		},
	}
	dom := func(n string) domains.DerivedDomain {
		return domains.DerivedDomain{Name: n, Paths: []string{"api/internal/" + n + "/"}}
	}
	m := domains.DerivedDomainMap{
		Scenario: "demo",
		Domains:  []domains.DerivedDomain{dom("a"), dom("b"), dom("c"), dom("d")},
	}
	return conflicts.DetectInput{Scenario: "demo", Snapshot: snap, DomainMap: m}
}

func TestDetect_EmitsGodDomainConflict(t *testing.T) {
	// a depends on every peer in a 4-domain graph → fan-out 1.0 → god_domain.
	got, err := couplingsmell.New().Detect(context.Background(), godDomainInput())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	var found *conflicts.Conflict
	for i := range got {
		if got[i].Subtype == "god_domain" {
			found = &got[i]
		}
	}
	if found == nil {
		t.Fatalf("expected god_domain coupling_smell, got %+v", got)
	}
	if found.Type != "coupling_smell" || found.Severity != conflicts.SeverityWarn {
		t.Fatalf("unexpected conflict: %+v", found)
	}
	if len(found.Domains) != 1 || found.Domains[0] != "a" {
		t.Fatalf("expected domain a, got %v", found.Domains)
	}
}

func TestDetect_UnstableDependencyMapsToInfoSeverity(t *testing.T) {
	// hub: Ce=5, Ca=2 → I≈0.714 ≥ warn band → unstable_dependency (info).
	pkg := func(id, dir string) graph.PackageNode {
		return graph.PackageNode{ID: id, RepoPath: dir}
	}
	names := []string{"hub", "l1", "l2", "l3", "l4", "l5", "u1", "u2"}
	var pkgs []graph.PackageNode
	var doms []domains.DerivedDomain
	for _, n := range names {
		pkgs = append(pkgs, pkg("p-"+n, "api/internal/"+n))
		doms = append(doms, domains.DerivedDomain{Name: n, Paths: []string{"api/internal/" + n + "/"}})
	}
	snap := graph.GraphSnapshot{Packages: pkgs, Imports: []graph.ImportEdge{
		{From: "p-hub", ToPackageID: "p-l1"},
		{From: "p-hub", ToPackageID: "p-l2"},
		{From: "p-hub", ToPackageID: "p-l3"},
		{From: "p-hub", ToPackageID: "p-l4"},
		{From: "p-hub", ToPackageID: "p-l5"},
		{From: "p-u1", ToPackageID: "p-hub"},
		{From: "p-u2", ToPackageID: "p-hub"},
	}}
	in := conflicts.DetectInput{Scenario: "demo", Snapshot: snap, DomainMap: domains.DerivedDomainMap{Scenario: "demo", Domains: doms}}

	// NewWithConfig with explicit defaults exercises the control-surface ctor.
	got, err := couplingsmell.NewWithConfig(boundaries.DefaultConfig()).Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	var found *conflicts.Conflict
	for i := range got {
		if got[i].Domains[0] == "hub" && got[i].Subtype == "unstable_dependency" {
			found = &got[i]
		}
	}
	if found == nil {
		t.Fatalf("expected unstable_dependency coupling_smell for hub, got %+v", got)
	}
	if found.Severity != conflicts.SeverityInfo {
		t.Fatalf("unstable_dependency must map to info severity, got %q", found.Severity)
	}
}

// TestDetect_EveryConflictHasSuggestedFix asserts every coupling_smell
// finding carries at least one templated fix naming the offending edges.
func TestDetect_EveryConflictHasSuggestedFix(t *testing.T) {
	got, err := couplingsmell.New().Detect(context.Background(), godDomainInput())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected at least one coupling_smell to assert on")
	}
	for _, c := range got {
		if len(c.SuggestedFixes) < 1 {
			t.Errorf("subtype %q: missing SuggestedFixes", c.Subtype)
			continue
		}
		if c.SuggestedFixes[0].Summary == "" {
			t.Errorf("subtype %q: first fix has empty Summary", c.Subtype)
		}
	}
}

func TestDetect_HealthyNoConflicts(t *testing.T) {
	pkg := func(id, dir string) graph.PackageNode {
		return graph.PackageNode{ID: id, RepoPath: dir}
	}
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{pkg("p-a", "api/internal/a"), pkg("p-b", "api/internal/b")},
		Imports:  []graph.ImportEdge{{From: "p-a", ToPackageID: "p-b"}},
	}
	m := domains.DerivedDomainMap{Domains: []domains.DerivedDomain{
		{Name: "a", Paths: []string{"api/internal/a/"}},
		{Name: "b", Paths: []string{"api/internal/b/"}},
	}}
	got, err := couplingsmell.New().Detect(context.Background(), conflicts.DetectInput{Scenario: "demo", Snapshot: snap, DomainMap: m})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("a single dependency in a two-domain graph is not a god-domain smell, got %+v", got)
	}
}

func TestDetect_CartographerConflictsDomainFanOutSurfaces(t *testing.T) {
	pkg := func(id, dir string) graph.PackageNode {
		return graph.PackageNode{ID: id, RepoPath: dir}
	}
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			pkg("p-conflicts", "api/internal/conflicts"),
			pkg("p-domains", "api/internal/domains"),
			pkg("p-graph", "api/internal/graph"),
			pkg("p-signals", "api/internal/signals"),
			pkg("p-audit", "api/internal/audit"),
		},
		Imports: []graph.ImportEdge{
			{From: "p-conflicts", ToPackageID: "p-domains"},
			{From: "p-conflicts", ToPackageID: "p-graph"},
			{From: "p-conflicts", ToPackageID: "p-signals"},
		},
	}
	dom := func(name string) domains.DerivedDomain {
		return domains.DerivedDomain{Name: name, Paths: []string{"api/internal/" + name + "/"}}
	}
	in := conflicts.DetectInput{
		Scenario: "architecture-cartographer",
		Snapshot: snap,
		DomainMap: domains.DerivedDomainMap{Scenario: "architecture-cartographer", Domains: []domains.DerivedDomain{
			dom("audit"),
			dom("conflicts"),
			dom("domains"),
			dom("graph"),
			dom("signals"),
		}},
	}
	got, err := couplingsmell.New().Detect(context.Background(), in)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	for _, c := range got {
		if c.Subtype == "god_domain" && len(c.Domains) == 1 && c.Domains[0] == "conflicts" {
			return
		}
	}
	t.Fatalf("expected conflicts domain fan-out to surface as god_domain, got %+v", got)
}
