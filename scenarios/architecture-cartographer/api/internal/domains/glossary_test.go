package domains

import (
	"reflect"
	"testing"
)

func TestGlossary(t *testing.T) {
	m := DerivedDomainMap{
		Domains: []DerivedDomain{
			{Name: "graph", Glossary: []string{"GraphSnapshot", "ImportEdge"}},
			{Name: "conflicts", Glossary: []string{"Detector", "ImportEdge"}},
		},
	}
	g := BuildGlossary(m)

	if !g.Match("graph", "graphsnapshot") {
		t.Fatal("expected case-insensitive match for graph/GraphSnapshot")
	}
	if g.Match("graph", "Detector") {
		t.Fatal("graph does not declare Detector")
	}
	if g.Match("", "x") || g.Match("graph", "") {
		t.Fatal("empty domain/symbol must not match")
	}
	// ImportEdge declared by both domains.
	if got := g.DomainsFor("ImportEdge"); !reflect.DeepEqual(got, []string{"conflicts", "graph"}) {
		t.Fatalf("DomainsFor(ImportEdge) = %v", got)
	}
	if got := g.DomainsFor("nope"); got != nil {
		t.Fatalf("DomainsFor(nope) = %v, want nil", got)
	}
}
