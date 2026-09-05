package domains_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/domains/mocks"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestService_ExtractDomains(t *testing.T) {
	now := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	loc := &mocks.FakeLocator{Dir: "/repo/scenarios/x"}
	doc := &mocks.FakeExtractor{Src: domains.SourceDomainsDoc, Extraction: mocks.NewExtraction(domains.SourceDomainsDoc, "graph", "conflicts")}
	folders := &mocks.FakeExtractor{Src: domains.SourceAPIFolders, Extraction: mocks.NewExtraction(domains.SourceAPIFolders, "graph")}

	svc := domains.NewService(loc, fixedClock{now}, doc, folders)
	m, err := svc.ExtractDomains(context.Background(), "x")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if m.Scenario != "x" || !m.DerivedAt.Equal(now) {
		t.Fatalf("scenario/derivedAt wrong: %+v", m)
	}
	if m.Authority != domains.SourceDomainsDoc {
		t.Fatalf("authority = %q", m.Authority)
	}
	if !reflect.DeepEqual(m.Names(), []string{"conflicts", "graph"}) {
		t.Fatalf("names = %v", m.Names())
	}
	// Extractors were called with the located dir.
	if len(doc.Calls) != 1 || doc.Calls[0] != "/repo/scenarios/x" {
		t.Fatalf("doc extractor calls = %v", doc.Calls)
	}
}

func TestService_EmptyScenario(t *testing.T) {
	svc := domains.NewService(&mocks.FakeLocator{}, fixedClock{}, &mocks.FakeExtractor{})
	_, err := svc.ExtractDomains(context.Background(), "  ")
	var notFound domains.ErrScenarioNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("want ErrScenarioNotFound, got %v", err)
	}
}

func TestService_LocatorError(t *testing.T) {
	svc := domains.NewService(&mocks.FakeLocator{Err: domains.ErrScenarioNotFound{Scenario: "x"}}, fixedClock{}, &mocks.FakeExtractor{})
	_, err := svc.ExtractDomains(context.Background(), "x")
	if err == nil {
		t.Fatal("expected locator error")
	}
}

func TestService_DomainFor(t *testing.T) {
	m := domains.DerivedDomainMap{
		Domains: []domains.DerivedDomain{
			{Name: "graph", Paths: []string{"scenarios/demo/api/internal/graph/"}},
		},
	}
	if got := m.DomainFor("scenarios/demo/api/internal/graph/service.go"); got != "graph" {
		t.Fatalf("DomainFor = %q, want graph", got)
	}
	if got := m.DomainFor("scenarios/demo/api/internal/other/x.go"); got != "" {
		t.Fatalf("DomainFor = %q, want empty", got)
	}
}

func TestDraftFromMap_ProducesMarkdownAndConfidence(t *testing.T) {
	m, err := domains.Resolve("demo", []domains.Extraction{
		{
			Source: domains.SourceAPIFolders,
			Domains: []domains.ExtractedDomain{
				{Name: "orders", Paths: []string{"api/internal/orders/"}},
			},
		},
		{
			Source: domains.SourceCLIGroups,
			Domains: []domains.ExtractedDomain{
				{Name: "orders", Paths: []string{"cli/domains/orders/"}},
			},
		},
	}, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	draft := domains.DraftFromMap(m)
	if draft.Scenario != "demo" || len(draft.Domains) != 1 {
		t.Fatalf("unexpected draft: %+v", draft)
	}
	if draft.Domains[0].Confidence != "medium" {
		t.Fatalf("confidence = %q, want medium", draft.Domains[0].Confidence)
	}
	if !strings.Contains(draft.Markdown, "| orders | TODO | TODO |") {
		t.Fatalf("markdown did not include TODO inventory row:\n%s", draft.Markdown)
	}
}
