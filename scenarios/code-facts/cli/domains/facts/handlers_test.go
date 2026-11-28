package facts

import (
	"strings"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCompositionReportLabelsControlPlaneEvidence(t *testing.T) {
	report := (&handlers{}).compositionReport(nil, &factsv1.CompositionReport{
		Bindings:      []*factsv1.CompositionBinding{{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: "resource", Status: "concrete"}},
		AnalysisScope: "go-syntax-v1; top-level and supported nested command specs are joined to explicit source registrations; nested/runtime argument dispatch and execution are not analyzed",
	})
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "control-plane CLI source composition") {
		t.Fatalf("unexpected composition summary: %#v", report.Summary)
	}
}

func TestCompositionReportPreservesNestedProvenanceAndUnknownBoundary(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	report := (&handlers{}).compositionReport(nil, &factsv1.CompositionReport{
		Bindings: []*factsv1.CompositionBinding{
			{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: "resource:archive:gc", InterfaceType: "nested commandtree.Spec handler=ArchiveCommandGC", Provider: "archiveGCCommand", ProviderExpression: "source registration only; runtime invocation unknown", Status: "concrete", SourcePath: "internal/cli/resourcehandlers/handlers_writen8nruntimeresponse.go", Line: 276, SourceHash: digest, Provenance: "nested-command-spec-handler-table"},
			{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: "resource:nested-scan-boundary", ProviderExpression: "PARTIAL: other registration forms remain UNKNOWN", Status: "unknown", SourcePath: "internal/cli/vroolicli/app.go", Line: 756, SourceHash: digest, Provenance: "nested-command-scan-boundary"},
		},
		AnalysisScope: "go-syntax-v1; top-level and supported nested command specs are joined to explicit source registrations; unsupported forms degrade to UNKNOWN; nested/runtime argument dispatch and execution are not analyzed",
	})
	joined := strings.Join(report.Results, "\n")
	for _, want := range []string{"resource:archive:gc", "handlers_writen8nruntimeresponse.go:276", digest, "nested-command-spec-handler-table", "runtime invocation unknown", "resource:nested-scan-boundary", "remain UNKNOWN"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("CLI composition report omitted %q: %q", want, joined)
		}
	}
	if len(report.Summary) != 1 || !strings.Contains(report.Summary[0], "nested/runtime argument dispatch and execution are not analyzed") {
		t.Fatalf("CLI summary hid the static/runtime boundary: %#v", report.Summary)
	}
}

func TestCompositionReportPreservesLocalInitializerHopAndUnknownIndirectForm(t *testing.T) {
	const digest = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	report := (&handlers{}).compositionReport(nil, &factsv1.CompositionReport{
		AnalysisScope: "go-syntax-v3; unique top-level same-function literal locals only; call-return and dynamic forms UNKNOWN",
		Bindings: []*factsv1.CompositionBinding{
			{Environment: "production", Consumer: "Deps", Dependency: "Store", Provider: "MemoryStore", ProviderExpression: "store", Status: "concrete", SourcePath: "api/wiring.go", Line: 18, SourceHash: digest, Provenance: "resolved_local_initializer=api/wiring.go:16#sha256:" + digest + "; type_check=UNKNOWN"},
			{Environment: "production", Consumer: "Deps", Dependency: "Store", ProviderExpression: "NewStore()", Status: "unknown", SourcePath: "api/wiring.go", Line: 24, SourceHash: digest, Provenance: "indirect_calls=UNKNOWN; entrypoint_reachability=PARTIAL"},
		},
	})
	joined := strings.Join(report.Results, "\n")
	for _, want := range []string{"resolved_local_initializer=api/wiring.go:16#sha256:" + digest, "NewStore()", "unknown", "type_check=UNKNOWN", "entrypoint_reachability=PARTIAL"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("CLI composition output omitted %q: %q", want, joined)
		}
	}
}

func TestFlowSearchReportLabelsRuntimeReachabilityPartial(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{
		"results":      []any{map[string]any{"title": "SearchComposition source call path", "text": "SearchComposition → Composition; DERIVED/PARTIAL", "path": "composition_api.go", "graph_hash": "graph", "attestation": "DERIVED/PARTIAL"}},
		"degradations": []any{map[string]any{"code": "call_graph_unavailable", "message": "missing graph"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := (&handlers{}).flowSearchReport(nil, response)
	if len(report.Result) < 2 || !strings.Contains(report.Result[0], "source reachability remains partial") || !strings.Contains(report.Result[1], "DERIVED/PARTIAL") {
		t.Fatalf("unexpected feature-flow report: %#v", report)
	}
}

func TestPersistencePatternReportKeepsUnknownLineageVisible(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{
		"scope":   "scenario:code-facts static only",
		"results": []any{},
		"unknowns": []any{map[string]any{
			"entity": "orders", "status": "UNKNOWN", "reason": "exact_repository_table_reference_not_proven",
			"schema": map[string]any{"path": "api/internal/catalog/schema.sql", "line": 8, "sha256": "schemahash"},
		}},
		"degradations": []any{"lineage_unknown"},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := (&handlers{}).persistencePatternsReport(nil, response)
	if len(report.Result) < 2 || !strings.Contains(report.Result[0], "1 UNKNOWN lineage candidate") || !strings.Contains(report.Result[1], "UNKNOWN lineage orders") || !strings.Contains(report.Result[1], "schema.sql:8") {
		t.Fatalf("unknown lineage was hidden from CLI output: %#v", report.Result)
	}
}

func TestEndpointAnatomyReportPreservesConnectMountChainAndUnknownBoundary(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{
		"scope": "bounded Connect source",
		"results": []any{map[string]any{
			"transport": "connect", "connect_service": "CodeFactsService", "connect_method": "Search",
			"route_path": "/vrooli.code_facts.v1.facts.CodeFactsService/Search", "state": "CONNECT_STATIC_MOUNT_REGISTERED",
			"connect_chain": []any{
				map[string]any{"path": "facts.connect.go", "line": 40, "basis": "generated Procedure path", "sha256": "routehash"},
				map[string]any{"path": "module.go", "line": 71, "basis": "RegisterServices", "sha256": "modulehash"},
			},
		}},
		"degradations": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := (&handlers{}).endpointAnatomyReport(nil, response)
	if len(report.Result) < 2 || !strings.Contains(report.Result[1], "Connect CodeFactsService.Search") || !strings.Contains(report.Result[1], "facts.connect.go:40 sha256:routehash → module.go:71 sha256:modulehash") || !strings.Contains(report.Result[1], "runtime mount remains unverified") {
		t.Fatalf("Connect chain or runtime boundary was lost: %#v", report.Result)
	}
}

func TestEndpointAnatomyReportKeepsDuplicateRouteCandidatesExplicit(t *testing.T) {
	response, err := structpb.NewStruct(map[string]any{
		"scope": "bounded HTTP source",
		"results": []any{map[string]any{
			"method": "POST", "route_path": "/api/v1/duplicate", "state": "ROUTE_AMBIGUOUS", "route_candidate_total": 2, "route_candidates_omitted": 0,
			"route_candidates": []any{
				map[string]any{"handler": "firstHandler", "registration": map[string]any{"path": "module.go", "line": 20, "sha256": "route-a"}, "handler_source": map[string]any{"path": "first.go", "line": 4, "sha256": "handler-a"}},
				map[string]any{"handler": "secondHandler", "registration": map[string]any{"path": "module.go", "line": 21, "sha256": "route-b"}, "handler_source": map[string]any{"path": "second.go", "line": 4, "sha256": "handler-b"}},
			},
		}},
		"degradations": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := (&handlers{}).endpointAnatomyReport(nil, response)
	if len(report.Result) < 2 || !strings.Contains(report.Result[1], "ROUTE_AMBIGUOUS") || !strings.Contains(report.Result[1], "firstHandler at module.go:20 sha256:route-a") || !strings.Contains(report.Result[1], "secondHandler at module.go:21 sha256:route-b") || !strings.Contains(report.Result[1], "no handler selected") {
		t.Fatalf("ambiguous route candidates were hidden or collapsed: %#v", report.Result)
	}
}
