package domains

import (
	"context"
	"strings"
	"testing"
	"time"

	"architecture-cartographer/internal/domains"

	"connectrpc.com/connect"
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type fakeService struct {
	m   domains.DerivedDomainMap
	err error
}

func (f fakeService) ExtractDomains(context.Context, string) (domains.DerivedDomainMap, error) {
	return f.m, f.err
}

func (f fakeService) GetDomainMap(context.Context, string) (domains.DerivedDomainMap, error) {
	return f.m, f.err
}

func (f fakeService) DraftDomains(context.Context, string) (domains.DomainDraft, error) {
	if f.err != nil {
		return domains.DomainDraft{}, f.err
	}
	return domains.DraftFromMap(f.m), nil
}

func TestHandler_ExtractDomains_Translates(t *testing.T) {
	m := domains.DerivedDomainMap{
		Scenario:  "x",
		Authority: domains.SourceDomainsDoc,
		DerivedAt: time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC),
		Domains: []domains.DerivedDomain{
			{
				Name:       "graph",
				Paths:      []string{"api/internal/graph/"},
				Glossary:   []string{"GraphSnapshot"},
				Archetypes: domains.DeclaredArchetypes("service"),
				Provenance: []domains.Source{domains.SourceDomainsDoc, domains.SourceAPIFolders},
			},
		},
		SharedSubstrate: []string{"api/internal/server/"},
		NonDomains:      []string{"server"},
		Declarations: []domains.DomainDeclaration{
			{Source: domains.SourceDomainsDoc, DomainNames: []string{"graph"}, Authoritative: true},
		},
	}
	h := NewHandler(fakeService{m: m})
	resp, err := h.ExtractDomains(context.Background(), connect.NewRequest(&domainsv1.ExtractDomainsRequest{Scenario: "x"}))
	if err != nil {
		t.Fatalf("ExtractDomains: %v", err)
	}
	got := resp.Msg.GetDomainMap()
	if got.GetScenario() != "x" {
		t.Fatalf("scenario = %q", got.GetScenario())
	}
	if got.GetAuthority() != domainsv1.DomainSource_DOMAIN_SOURCE_DOMAINS_DOC {
		t.Fatalf("authority = %v", got.GetAuthority())
	}
	if len(got.GetDomains()) != 1 {
		t.Fatalf("domains = %d", len(got.GetDomains()))
	}
	d := got.GetDomains()[0]
	if d.GetName() != "graph" || len(d.GetArchetypes()) != 1 || d.GetArchetypes()[0].GetArchetype() != domainsv1.Archetype_ARCHETYPE_SERVICE {
		t.Fatalf("domain = %+v", d)
	}
	if d.GetAttestation() == nil || d.GetAttestation().GetBasis() == commonv1.Basis_BASIS_UNSPECIFIED {
		t.Fatalf("domain attestation missing or unspecified: %+v", d.GetAttestation())
	}
	if got.GetAttestation() == nil || got.GetAttestation().GetBasis() != commonv1.Basis_BASIS_DERIVED {
		t.Fatalf("map attestation should be DERIVED: %+v", got.GetAttestation())
	}
	if len(d.GetProvenance()) != 2 {
		t.Fatalf("provenance = %v", d.GetProvenance())
	}
	if got.GetDerivedAt() == nil {
		t.Fatal("derived_at should be set")
	}
}

func TestHandler_ConvergenceReport(t *testing.T) {
	// Authority (doc) declares graph + conflicts; folders only have graph,
	// so conflicts is a missing_implementation finding.
	mk := func(src domains.Source, names ...string) domains.Extraction {
		ex := domains.Extraction{Source: src}
		for _, n := range names {
			ex.Domains = append(ex.Domains, domains.ExtractedDomain{Name: n, Paths: []string{n + "/"}})
		}
		return ex
	}
	m, err := domains.Resolve("x", []domains.Extraction{
		mk(domains.SourceDomainsDoc, "graph", "conflicts"),
		mk(domains.SourceAPIFolders, "graph"),
	}, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	h := NewHandler(fakeService{m: m})
	resp, err := h.ConvergenceReport(context.Background(), connect.NewRequest(&domainsv1.ConvergenceReportRequest{Scenario: "x"}))
	if err != nil {
		t.Fatalf("ConvergenceReport: %v", err)
	}
	if resp.Msg.GetAuthority() != domainsv1.DomainSource_DOMAIN_SOURCE_DOMAINS_DOC {
		t.Fatalf("authority = %v", resp.Msg.GetAuthority())
	}
	var found bool
	for _, f := range resp.Msg.GetFindings() {
		if f.GetKind() == domains.FindingMissingImplementation && f.GetDomain() == "conflicts" {
			found = true
			if f.GetSeverity() != domainsv1.ConvergenceSeverity_CONVERGENCE_SEVERITY_WARN {
				t.Fatalf("expected warn severity, got %v", f.GetSeverity())
			}
		}
	}
	if !found {
		t.Fatalf("expected missing_implementation for conflicts, got %+v", resp.Msg.GetFindings())
	}
}

func TestHandler_DraftDomains(t *testing.T) {
	m, err := domains.Resolve("x", []domains.Extraction{
		{
			Source: domains.SourceAPIFolders,
			Domains: []domains.ExtractedDomain{
				{Name: "graph", Paths: []string{"api/internal/graph/"}},
			},
		},
		{
			Source: domains.SourceCLIGroups,
			Domains: []domains.ExtractedDomain{
				{Name: "graph", Paths: []string{"cli/domains/graph/"}},
			},
		},
	}, time.Time{})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	h := NewHandler(fakeService{m: m})
	resp, err := h.DraftDomains(context.Background(), connect.NewRequest(&domainsv1.DraftDomainsRequest{Scenario: "x"}))
	if err != nil {
		t.Fatalf("DraftDomains: %v", err)
	}
	if resp.Msg.GetScenario() != "x" || len(resp.Msg.GetDomains()) != 1 {
		t.Fatalf("unexpected draft response: %+v", resp.Msg)
	}
	d := resp.Msg.GetDomains()[0]
	if d.GetName() != "graph" || d.GetConfidence() != "medium" {
		t.Fatalf("unexpected proposed domain: %+v", d)
	}
	if !strings.Contains(resp.Msg.GetMarkdown(), "| graph |") {
		t.Fatalf("draft markdown missing graph row:\n%s", resp.Msg.GetMarkdown())
	}
}

func TestHandler_EmptyScenario(t *testing.T) {
	h := NewHandler(fakeService{})
	_, err := h.GetDomainMap(context.Background(), connect.NewRequest(&domainsv1.GetDomainMapRequest{Scenario: ""}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", err)
	}
}

func TestHandler_ServiceErrorMapping(t *testing.T) {
	h := NewHandler(fakeService{err: domains.ErrScenarioNotFound{Scenario: "x"}})
	_, err := h.ExtractDomains(context.Background(), connect.NewRequest(&domainsv1.ExtractDomainsRequest{Scenario: "x"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("want NotFound, got %v", err)
	}
}
