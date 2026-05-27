package boundaries_test

import (
	"reflect"
	"testing"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/signals/boundaries"
)

// fourDomainGraph builds a snapshot with domains a, b, c, kernel where:
//
//	a -> b, a -> c, a -> kernel   (a is a fan-out hub)
//	b -> kernel, c -> kernel      (kernel is widely depended upon)
func fourDomainGraph() (graph.GraphSnapshot, domains.DerivedDomainMap) {
	pkg := func(id, dir string) graph.PackageNode {
		return graph.PackageNode{ID: id, Directory: dir, Internal: true}
	}
	snap := graph.GraphSnapshot{
		Packages: []graph.PackageNode{
			pkg("p-a", "api/internal/a"),
			pkg("p-b", "api/internal/b"),
			pkg("p-c", "api/internal/c"),
			pkg("p-k", "api/internal/kernel"),
		},
		Imports: []graph.ImportEdge{
			{From: "p-a", ToPackageID: "p-b"},
			{From: "p-a", ToPackageID: "p-c"},
			{From: "p-a", ToPackageID: "p-k"},
			{From: "p-b", ToPackageID: "p-k"},
			{From: "p-c", ToPackageID: "p-k"},
		},
	}
	dom := func(name string) domains.DerivedDomain {
		return domains.DerivedDomain{Name: name, Paths: []string{"api/internal/" + name + "/"}}
	}
	m := domains.DerivedDomainMap{
		Scenario: "demo",
		Domains:  []domains.DerivedDomain{dom("a"), dom("b"), dom("c"), {Name: "kernel", Paths: []string{"api/internal/kernel/"}}},
	}
	return snap, m
}

func find(rep boundaries.Report, name string) boundaries.DomainCoupling {
	for _, d := range rep.Domains {
		if d.Domain == name {
			return d
		}
	}
	return boundaries.DomainCoupling{}
}

func hasSmell(dc boundaries.DomainCoupling, kind string) bool {
	for _, s := range dc.Smells {
		if s.Kind == kind {
			return true
		}
	}
	return false
}

func TestAnalyze_CouplingMetrics(t *testing.T) {
	snap, m := fourDomainGraph()
	rep := boundaries.Analyze("demo", snap, m, boundaries.DefaultConfig())

	a := find(rep, "a")
	if a.Efferent != 3 || a.Afferent != 0 {
		t.Fatalf("a: Ce=%d Ca=%d, want 3/0", a.Efferent, a.Afferent)
	}
	if a.Instability != 1.0 {
		t.Fatalf("a instability = %v, want 1.0", a.Instability)
	}
	if a.FanOut != 1.0 {
		t.Fatalf("a fanout = %v, want 1.0 (depends on all 3 others)", a.FanOut)
	}
	if !reflect.DeepEqual(a.DependsOn, []string{"b", "c", "kernel"}) {
		t.Fatalf("a dependsOn = %v", a.DependsOn)
	}

	k := find(rep, "kernel")
	if k.Efferent != 0 || k.Afferent != 3 {
		t.Fatalf("kernel: Ce=%d Ca=%d, want 0/3", k.Efferent, k.Afferent)
	}
	if k.Instability != 0.0 {
		t.Fatalf("kernel instability = %v, want 0", k.Instability)
	}
	b := find(rep, "b")
	if b.Efferent != 1 || b.Afferent != 1 || b.Instability != 0.5 {
		t.Fatalf("b: Ce=%d Ca=%d I=%v, want 1/1/0.5", b.Efferent, b.Afferent, b.Instability)
	}
}

func TestAnalyze_GodDomainSmell(t *testing.T) {
	snap, m := fourDomainGraph()
	rep := boundaries.Analyze("demo", snap, m, boundaries.DefaultConfig())
	a := find(rep, "a")
	if !hasSmell(a, boundaries.SmellGodDomain) {
		t.Fatalf("a should have god_domain smell; smells=%+v", a.Smells)
	}
	if a.HealthScore >= 1.0 {
		t.Fatalf("a health should be penalized, got %v", a.HealthScore)
	}
}

func TestAnalyze_StableKernelNoSmellFullHealth(t *testing.T) {
	snap, m := fourDomainGraph()
	rep := boundaries.Analyze("demo", snap, m, boundaries.DefaultConfig())
	k := find(rep, "kernel")
	if !k.StableKernel {
		t.Fatal("kernel should be detected as a stable kernel")
	}
	if len(k.Smells) != 0 || k.HealthScore != 1.0 {
		t.Fatalf("stable kernel must be smell-free with full health, got smells=%+v score=%v", k.Smells, k.HealthScore)
	}
}

func TestAnalyze_CompositionRootExemptFromGodDomain(t *testing.T) {
	snap, m := fourDomainGraph()
	// Tag the fan-out hub as a composition root → exempt.
	for i := range m.Domains {
		if m.Domains[i].Name == "a" {
			m.Domains[i].Archetype = "composition-root"
		}
	}
	rep := boundaries.Analyze("demo", snap, m, boundaries.DefaultConfig())
	a := find(rep, "a")
	if hasSmell(a, boundaries.SmellGodDomain) {
		t.Fatalf("composition-root domain must be exempt from god_domain; smells=%+v", a.Smells)
	}
}

func TestAnalyze_TestOnlyEdgesExcluded(t *testing.T) {
	snap, m := fourDomainGraph()
	// Add a test-only edge kernel -> a; it must NOT add efferent coupling.
	snap.Imports = append(snap.Imports, graph.ImportEdge{From: "p-k", ToPackageID: "p-a", TestOnly: true})
	rep := boundaries.Analyze("demo", snap, m, boundaries.DefaultConfig())
	if k := find(rep, "kernel"); k.Efferent != 0 {
		t.Fatalf("test-only edge must not count; kernel Ce=%d", k.Efferent)
	}
}
