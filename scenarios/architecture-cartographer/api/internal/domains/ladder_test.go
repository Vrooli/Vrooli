package domains

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func ext(src Source, ds ...ExtractedDomain) Extraction {
	return Extraction{Source: src, Domains: ds}
}

func dom(name string, paths ...string) ExtractedDomain {
	return ExtractedDomain{Name: name, Paths: paths}
}

func TestResolve_AuthorityIsHighestRungWithDomains(t *testing.T) {
	// DOMAINS.md rung empty; folders rung is authority.
	extractions := []Extraction{
		ext(SourceDomainsDoc), // empty -> not authority
		ext(SourceAPIFolders, dom("graph", "g/"), dom("conflicts", "c/")),
		ext(SourceCLIGroups, dom("graph", "cli/g/")),
	}
	m, err := Resolve("x", extractions, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if m.Authority != SourceAPIFolders {
		t.Fatalf("authority = %q, want api_folders", m.Authority)
	}
	if !reflect.DeepEqual(m.Names(), []string{"conflicts", "graph"}) {
		t.Fatalf("names = %v", m.Names())
	}
}

func TestResolve_Provenance(t *testing.T) {
	extractions := []Extraction{
		ext(SourceDomainsDoc, dom("graph", "g/"), dom("conflicts", "c/")),
		ext(SourceAPIFolders, dom("graph", "g/")),
		ext(SourceCLIGroups, dom("graph", "cli/g/")),
	}
	m, err := Resolve("x", extractions, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if m.Authority != SourceDomainsDoc {
		t.Fatalf("authority = %q", m.Authority)
	}
	var graph, conflicts DerivedDomain
	for _, d := range m.Domains {
		switch d.Name {
		case "graph":
			graph = d
		case "conflicts":
			conflicts = d
		}
	}
	// graph declared by all three rungs.
	if !reflect.DeepEqual(graph.Provenance, []Source{SourceAPIFolders, SourceCLIGroups, SourceDomainsDoc}) {
		t.Fatalf("graph provenance = %v", graph.Provenance)
	}
	// conflicts only in the doc.
	if !reflect.DeepEqual(conflicts.Provenance, []Source{SourceDomainsDoc}) {
		t.Fatalf("conflicts provenance = %v", conflicts.Provenance)
	}
	// Paths come from the authority rung.
	if !reflect.DeepEqual(graph.Paths, []string{"g/"}) {
		t.Fatalf("graph paths = %v (should be authority's)", graph.Paths)
	}
}

func TestResolve_SharedSubstrateFromAnyRung(t *testing.T) {
	docExt := Extraction{
		Source:          SourceDomainsDoc,
		Domains:         []ExtractedDomain{dom("graph", "g/")},
		SharedSubstrate: []string{"api/internal/server/"},
		NonDomains:      []string{"server"},
	}
	m, err := Resolve("x", []Extraction{docExt, ext(SourceAPIFolders, dom("graph", "g/"))}, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !reflect.DeepEqual(m.SharedSubstrate, []string{"api/internal/server/"}) {
		t.Fatalf("shared substrate = %v", m.SharedSubstrate)
	}
}

func TestResolve_NoAuthority(t *testing.T) {
	_, err := Resolve("x", []Extraction{ext(SourceDomainsDoc), ext(SourceAPIFolders)}, time.Time{})
	var noAuth ErrNoAuthority
	if !errors.As(err, &noAuth) {
		t.Fatalf("want ErrNoAuthority, got %v", err)
	}
}

func TestResolve_DeclarationsRecorded(t *testing.T) {
	extractions := []Extraction{
		ext(SourceDomainsDoc, dom("graph", "g/")),
		ext(SourceAPIFolders, dom("graph", "g/"), dom("extra", "e/")),
	}
	m, _ := Resolve("x", extractions, time.Time{})
	if len(m.Declarations) != 2 {
		t.Fatalf("declarations = %d, want 2", len(m.Declarations))
	}
	if !m.Declarations[0].Authoritative || m.Declarations[1].Authoritative {
		t.Fatalf("authoritative flags wrong: %+v", m.Declarations)
	}
	if !reflect.DeepEqual(m.Declarations[1].DomainNames, []string{"extra", "graph"}) {
		t.Fatalf("folder declaration names = %v", m.Declarations[1].DomainNames)
	}
}

func TestRunLadder_PropagatesError(t *testing.T) {
	failing := failingExtractor{src: SourceCLIGroups, err: errors.New("boom")}
	_, err := RunLadder(context.Background(), "/x", []DomainSourceExtractor{&failing})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("want boom error, got %v", err)
	}
}

type failingExtractor struct {
	src Source
	err error
}

func (f *failingExtractor) Source() Source { return f.src }
func (f *failingExtractor) Extract(context.Context, string) (Extraction, error) {
	return Extraction{}, f.err
}
