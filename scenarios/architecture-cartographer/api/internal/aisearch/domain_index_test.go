package aisearch

import (
	"context"
	"strings"
	"testing"

	"architecture-cartographer/internal/domains"
)

// fakeProvider serves canned domain maps per scenario.
type fakeProvider struct {
	maps map[string]domains.DerivedDomainMap
	err  map[string]error
}

func (f fakeProvider) ExtractDomains(_ context.Context, scenario string) (domains.DerivedDomainMap, error) {
	if f.err != nil {
		if e := f.err[scenario]; e != nil {
			return domains.DerivedDomainMap{}, e
		}
	}
	return f.maps[scenario], nil
}

// fakeLister returns a fixed scenario list.
type fakeLister struct{ names []string }

func (f fakeLister) List(_ context.Context) ([]string, error) { return f.names, nil }

func sampleMap() domains.DerivedDomainMap {
	return domains.DerivedDomainMap{
		Scenario:            "plan-manager",
		Authority:           domains.SourceDomainsDoc,
		AuthorityConfidence: domains.ConfidenceHigh,
		Domains: []domains.DerivedDomain{
			{
				Name:           "authoring",
				Responsibility: "Guided composer wizard: section-by-section flow.",
				Purpose:        "Make plan authoring cheap for local models.",
				OwnsData:       "draft sessions",
				Glossary:       []string{"AuthoringSession", "PhaseDraft"},
				Surfaces:       []string{"api", "cli"},
				Paths:          []string{"api/internal/authoring"},
				Archetypes:     domains.DeclaredArchetypes("service"),
			},
		},
	}
}

func TestToDomainRecord(t *testing.T) {
	m := sampleMap()
	r := toDomainRecord("plan-manager", m, m.Domains[0])
	if r.ID != "plan-manager/authoring" {
		t.Fatalf("ID = %q, want plan-manager/authoring", r.ID)
	}
	if r.Scenario != "plan-manager" || r.Name != "authoring" {
		t.Fatalf("scenario/name = %q/%q", r.Scenario, r.Name)
	}
	if r.Responsibility == "" || r.Purpose == "" {
		t.Fatalf("responsibility/purpose should be populated: %+v", r)
	}
	if r.Archetype != "service" {
		t.Fatalf("archetype = %q, want service", r.Archetype)
	}
	if r.Authority != string(domains.SourceDomainsDoc) || r.Confidence != string(domains.ConfidenceHigh) {
		t.Fatalf("authority/confidence = %q/%q", r.Authority, r.Confidence)
	}
}

func TestComposeDomainEmbeddingText(t *testing.T) {
	m := sampleMap()
	r := toDomainRecord("plan-manager", m, m.Domains[0])
	body := composeDomainEmbeddingText(r)

	// The natural-language identity + responsibility/purpose prose MUST be present
	// (the term-agnostic anchor).
	for _, want := range []string{"plan-manager authoring", "Guided composer wizard", "cheap for local models", "Owns: draft sessions", "Vocabulary: AuthoringSession"} {
		if !strings.Contains(body, want) {
			t.Errorf("embedding text missing %q\n--- body ---\n%s", want, body)
		}
	}
	// Structured facets MUST NOT be embedded as prose (plan §12): archetype and
	// surfaces are metadata filters, not retrieval signal.
	if strings.Contains(body, "service") {
		t.Errorf("archetype 'service' leaked into embedding text:\n%s", body)
	}
	if strings.Contains(strings.ToLower(body), "surfaces") {
		t.Errorf("surfaces leaked into embedding text:\n%s", body)
	}
}

func TestDomainContentHashStable(t *testing.T) {
	m := sampleMap()
	r := toDomainRecord("plan-manager", m, m.Domains[0])
	h1 := domainContentHash(r)
	h2 := domainContentHash(r)
	if h1 != h2 {
		t.Fatalf("hash not stable: %q vs %q", h1, h2)
	}
	if !strings.HasPrefix(h1, "sha256:") {
		t.Fatalf("hash missing prefix: %q", h1)
	}
	// A meaningful field change must move the hash (drift gate).
	r.Responsibility = "different responsibility"
	if domainContentHash(r) == h1 {
		t.Fatalf("hash did not change after responsibility edit")
	}
}

func TestDomainToSourceDoc(t *testing.T) {
	m := sampleMap()
	r := toDomainRecord("plan-manager", m, m.Domains[0])
	doc := domainToSourceDoc(r, composeDomainEmbeddingText)
	if doc.ID != "plan-manager/authoring" {
		t.Fatalf("doc.ID = %q", doc.ID)
	}
	if doc.Kind != domainKind {
		t.Fatalf("doc.Kind = %q, want %q", doc.Kind, domainKind)
	}
	if doc.Body == "" || doc.ContentHash == "" {
		t.Fatalf("doc body/hash empty: %+v", doc)
	}
	if doc.Meta["scenario"] != "plan-manager" || doc.Meta["id"] != "plan-manager/authoring" {
		t.Fatalf("meta wrong: %+v", doc.Meta)
	}
}

func TestDomainSourceLoadAll(t *testing.T) {
	pm := sampleMap()
	sh := domains.DerivedDomainMap{
		Scenario: "search-hub",
		Domains: []domains.DerivedDomain{
			{Name: "routing", Responsibility: "Classify a free query and fan out."},
		},
	}
	src := newDomainSource(
		fakeProvider{maps: map[string]domains.DerivedDomainMap{"plan-manager": pm, "search-hub": sh}},
		fakeLister{names: []string{"plan-manager", "search-hub", "missing"}},
		nil,
	)
	docs, err := src.LoadAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// plan-manager(1) + search-hub(1); "missing" yields an empty map (no domains).
	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2: %+v", len(docs), docs)
	}
	ids := map[string]bool{}
	for _, d := range docs {
		ids[d.ID] = true
	}
	if !ids["plan-manager/authoring"] || !ids["search-hub/routing"] {
		t.Fatalf("missing expected ids: %v", ids)
	}
}

func TestPayloadToHit(t *testing.T) {
	m := sampleMap()
	r := toDomainRecord("plan-manager", m, m.Domains[0])
	hit := payloadToHit(pointIDForDomain(r.ID), 0.83, domainMeta(r))
	if hit.ID != "plan-manager/authoring" {
		t.Fatalf("hit.ID = %q (should project the payload id, not the point uuid)", hit.ID)
	}
	if hit.Scenario != "plan-manager" || hit.Name != "authoring" {
		t.Fatalf("hit scenario/name = %q/%q", hit.Scenario, hit.Name)
	}
	if hit.ScorePercent != 83 {
		t.Fatalf("score percent = %d, want 83", hit.ScorePercent)
	}
	if len(hit.Paths) == 0 {
		t.Fatalf("paths not projected")
	}
}

func TestScoreDomainTextFallback(t *testing.T) {
	m := sampleMap()
	r := toDomainRecord("plan-manager", m, m.Domains[0])
	// A query overlapping the identity + responsibility scores > 0.
	if s := scoreDomain(r, tokenize("authoring composer wizard")); s <= 0 {
		t.Fatalf("expected positive score, got %v", s)
	}
	// Pure gibberish scores 0 (junk rejection in the offline leg).
	if s := scoreDomain(r, tokenize("xyzzy florbnax qwertyuiop")); s != 0 {
		t.Fatalf("expected zero score for gibberish, got %v", s)
	}
}
