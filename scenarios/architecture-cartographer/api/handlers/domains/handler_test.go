package domains

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"architecture-cartographer/internal/domains"
	internalzones "architecture-cartographer/internal/zones"
	"github.com/gorilla/mux"

	"connectrpc.com/connect"
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type fakeScenarioLocator struct {
	dir string
	err error
}

func (f fakeScenarioLocator) Locate(string) (string, error) { return f.dir, f.err }

func TestClassifyZonesReturnsBoundedScopedEvidenceAndUnknowns(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	for _, dir := range []string{
		filepath.Join(scenarioDir, ".vrooli"),
		filepath.Join(root, "templates", "scenarios", "react-vite"),
		filepath.Join(scenarioDir, "api", "internal", "orders"),
		filepath.Join(scenarioDir, "api", "handlers", "orders"),
		filepath.Join(scenarioDir, "docs"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, ".vrooli", "service.json"), []byte(`{"generation":{"template":{"id":"react-vite"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"zones":{"pathPatterns":{"transport":["api/handlers/<domain>/"],"domain":["api/internal/<domain>/"],"substrate":["api/internal/<segment>/"],"composition-root":["api/internal/<segment>/"]},"builtinSubstrateSegments":[],"compositionRootSegments":["app"]}}`
	if err := os.WriteFile(filepath.Join(root, "templates", "scenarios", "react-vite", "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		filepath.Join(scenarioDir, "api", "internal", "orders", "service.go"),
		filepath.Join(scenarioDir, "api", "handlers", "orders", "routes.go"),
		filepath.Join(scenarioDir, "docs", "guide.md"),
	} {
		if err := os.WriteFile(file, []byte("fixture"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	service := fakeService{m: domains.DerivedDomainMap{Scenario: "demo", Domains: []domains.DerivedDomain{{Name: "orders", Paths: []string{"api/internal/orders/"}, Archetypes: domains.DeclaredArchetypes("service")}}}}
	router := mux.NewRouter()
	Module(service, fakeScenarioLocator{dir: scenarioDir}).Mount(router)
	req := httptest.NewRequest("POST", classifyZonesPath, strings.NewReader(`{"scenario":"demo","paths":["api/internal/orders/service.go","api/handlers/orders/routes.go","docs/guide.md"]}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var got classifyZonesResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Scenario != "demo" || len(got.Results) != 3 {
		t.Fatalf("response scope/results = %q/%d", got.Scenario, len(got.Results))
	}
	if result := got.Results[0]; result.Zone != internalzones.Domain || result.Domain != "orders" || result.Status != "KNOWN" || result.Basis != "domain_path_pattern" {
		t.Fatalf("domain result = %+v", result)
	}
	if result := got.Results[1]; result.Zone != internalzones.Transport || result.Domain != "orders" {
		t.Fatalf("transport result = %+v", result)
	}
	if result := got.Results[2]; result.Status != "UNKNOWN" || result.Basis != "unclassified" {
		t.Fatalf("unknown result = %+v", result)
	}
	if len(got.Degradations) != 1 || got.Degradations[0].Code != "unclassified_paths" || got.Degradations[0].Count != 1 {
		t.Fatalf("degradations = %+v", got.Degradations)
	}
}

func TestClassifyZonesRejectsUnscopedOrEscapingRequests(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.go")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(scenarioDir, "api", "linked.go")); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}
	h := NewHandler(fakeService{m: domains.DerivedDomainMap{Scenario: "demo"}}, fakeScenarioLocator{dir: scenarioDir})
	for _, body := range []string{
		`{"scenario":"../","paths":["api/file.go"]}`,
		`{"scenario":"demo","paths":["../outside.go"]}`,
		`{"scenario":"demo","paths":["api/linked.go"]}`,
		`{"scenario":"demo","paths":["missing.go"]}`,
		`{"scenario":"demo","paths":[]}`,
	} {
		req := httptest.NewRequest("POST", classifyZonesPath, strings.NewReader(body))
		response := httptest.NewRecorder()
		h.ClassifyZones(response, req)
		if response.Code != 400 {
			t.Errorf("body %s status = %d, want 400 (%s)", body, response.Code, response.Body.String())
		}
	}
}

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
				Name:              "graph",
				Paths:             []string{"api/internal/graph/"},
				Glossary:          []string{"GraphSnapshot"},
				Archetypes:        domains.DeclaredArchetypes("service"),
				Provenance:        []domains.Source{domains.SourceDomainsDoc, domains.SourceAPIFolders},
				DeclarationSource: domains.DomainsDocPath,
				DeclarationLine:   17,
			},
		},
		SharedSubstrate: []string{"api/internal/database/"},
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
	if got := d.GetAttestation().GetCitations(); len(got) != 2 || got[0].GetLocator() != domains.DomainsDocPath+":17" {
		t.Fatalf("domain attestation citations = %+v, want exact declaration row and code path", got)
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

func TestArchetypeNameToProto(t *testing.T) {
	tests := []struct {
		name string
		want domainsv1.Archetype
	}{
		{"reporting", domainsv1.Archetype_ARCHETYPE_REPORTING},
		{"service", domainsv1.Archetype_ARCHETYPE_SERVICE},
		{"mutation", domainsv1.Archetype_ARCHETYPE_MUTATION},
		{"classification", domainsv1.Archetype_ARCHETYPE_CLASSIFICATION},
		{"orchestration", domainsv1.Archetype_ARCHETYPE_ORCHESTRATION},
		{"scoring", domainsv1.Archetype_ARCHETYPE_SCORING},
		{"query", domainsv1.Archetype_ARCHETYPE_QUERY},
		{"unknown", domainsv1.Archetype_ARCHETYPE_UNSPECIFIED},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArchetypeNameToProto(tt.name); got != tt.want {
				t.Fatalf("ArchetypeNameToProto(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
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
