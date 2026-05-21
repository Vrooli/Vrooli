package manifest_test

import (
	"reflect"
	"testing"

	"architecture-cartographer/internal/manifest"
)

func TestBuildGlossary_LookupCaseInsensitive(t *testing.T) {
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{
			{Name: "graph", Glossary: []string{"GraphSnapshot", "ImportEdge"}},
			{Name: "signals", Glossary: []string{"Signal", "Verdict"}},
		},
	}
	g := manifest.BuildGlossary(m)

	if !g.Match("graph", "graphsnapshot") {
		t.Fatalf("expected case-insensitive match")
	}
	if !g.Match("signals", "Verdict") {
		t.Fatalf("expected exact-case match")
	}
	if g.Match("graph", "Verdict") {
		t.Fatalf("symbol from another domain should not match")
	}
	if g.Match("", "anything") {
		t.Fatalf("empty domain should never match")
	}
	if g.Match("graph", "") {
		t.Fatalf("empty symbol should never match")
	}
}

func TestGlossary_DomainsFor_ReturnsAlphabeticalDomains(t *testing.T) {
	m := manifest.ManifestDefinition{
		Domains: []manifest.DomainSpec{
			{Name: "signals", Glossary: []string{"Verdict"}},
			{Name: "conflicts", Glossary: []string{"Verdict"}},
			{Name: "graph", Glossary: []string{"GraphSnapshot"}},
		},
	}
	g := manifest.BuildGlossary(m)

	got := g.DomainsFor("verdict")
	want := []string{"conflicts", "signals"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DomainsFor(verdict)=%v want %v", got, want)
	}

	if got := g.DomainsFor(""); got != nil {
		t.Fatalf("DomainsFor(\"\") should be nil, got %v", got)
	}
	if got := g.DomainsFor("missing"); got != nil {
		t.Fatalf("DomainsFor(missing) should be nil, got %v", got)
	}
}
