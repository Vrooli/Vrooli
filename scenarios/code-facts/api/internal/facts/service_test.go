package facts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	_ "modernc.org/sqlite"
)

func TestDescribeDiscoversGenericGoModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")

	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if report.GetTarget().GetScenarioAware() {
		t.Fatal("generic module should not be scenario-aware")
	}
	requireSurface(t, report.GetSurfaces(), "target", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireParseUnit(t, report.GetParseUnits(), "go", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestDescribeDiscoversGenericTypeScriptProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"build":"tsc"}}`)
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"strict":true}}`)
	writeFile(t, filepath.Join(root, "src", "index.ts"), "export const value = 1;\n")

	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	requireParseUnit(t, report.GetParseUnits(), "typescript", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestDescribeScenarioDiscoversSurfacesAndParseUnits(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "demo")

	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: "demo",
			RepoRoot: repo,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES, factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if !report.GetTarget().GetScenarioAware() || report.GetTarget().GetScenario() != "demo" {
		t.Fatalf("target context = %#v, want scenario-aware demo", report.GetTarget())
	}
	if report.GetTarget().GetRootPath() != scenarioRoot {
		t.Fatalf("root path = %q, want %q", report.GetTarget().GetRootPath(), scenarioRoot)
	}
	requireSurface(t, report.GetSurfaces(), "api", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireSurface(t, report.GetSurfaces(), "cli", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireSurface(t, report.GetSurfaces(), "runtime", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireSurface(t, report.GetSurfaces(), "ui", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireSurfaceKind(t, report.GetSurfaces(), "runtime", factsv1.SurfaceKind_SURFACE_KIND_RUNTIME)
	requireParseUnit(t, report.GetParseUnits(), "go", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
	requireParseUnit(t, report.GetParseUnits(), "typescript", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestDescribeFleetImportsUsesExplicitSubset(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "alpha")
	writeScenarioFixtureInRepo(t, repo, "beta")
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			protoImportNode("main.go", "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"),
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).DescribeFleetImports(context.Background(), &factsv1.DescribeFleetImportsRequest{
		Scenarios:      []string{"alpha", "beta", "alpha", ""},
		RepoRoot:       repo,
		LanguageFilter: []string{"go"},
	})
	if err != nil {
		t.Fatalf("DescribeFleetImports() error = %v", err)
	}
	if len(report.GetResults()) != 2 {
		t.Fatalf("results len = %d, want 2", len(report.GetResults()))
	}
	if report.GetResults()[0].GetScenario() != "alpha" || report.GetResults()[1].GetScenario() != "beta" {
		t.Fatalf("scenarios = %q, %q", report.GetResults()[0].GetScenario(), report.GetResults()[1].GetScenario())
	}
	for _, result := range report.GetResults() {
		if result.GetError() != "" {
			t.Fatalf("unexpected per-scenario error for %s: %s", result.GetScenario(), result.GetError())
		}
		requireFactFamily(t, result.GetReport().GetFacts(), factsv1.FactFamily_FACT_FAMILY_IMPORTS)
	}
}

func TestDescribeFleetImportsListsAllAndAppliesLimit(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "alpha")
	writeScenarioFixtureInRepo(t, repo, "beta")
	writeScenarioFixtureInRepo(t, repo, "gamma")

	report, err := NewService().DescribeFleetImports(context.Background(), &factsv1.DescribeFleetImportsRequest{
		RepoRoot: repo,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("DescribeFleetImports() error = %v", err)
	}
	if len(report.GetResults()) != 2 {
		t.Fatalf("results len = %d, want 2", len(report.GetResults()))
	}
	if report.GetResults()[0].GetScenario() != "alpha" || report.GetResults()[1].GetScenario() != "beta" {
		t.Fatalf("scenarios = %q, %q", report.GetResults()[0].GetScenario(), report.GetResults()[1].GetScenario())
	}
}

func TestDescribeFleetImportsReusesCache(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "alpha")
	provider := &countingProvider{fakeProvider: fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result:   &GraphResult{Graph: &commonv1.CodeGraph{}},
	}}
	svc := NewService(WithBroker(NewBroker(provider)))
	req := &factsv1.DescribeFleetImportsRequest{
		Scenarios:      []string{"alpha"},
		RepoRoot:       repo,
		LanguageFilter: []string{"go"},
		UseCache:       true,
	}

	if _, err := svc.DescribeFleetImports(context.Background(), req); err != nil {
		t.Fatalf("first DescribeFleetImports() error = %v", err)
	}
	firstCalls := provider.calls
	if firstCalls == 0 {
		t.Fatal("provider was not called on first request")
	}
	if _, err := svc.DescribeFleetImports(context.Background(), req); err != nil {
		t.Fatalf("second DescribeFleetImports() error = %v", err)
	}
	if provider.calls != firstCalls {
		t.Fatalf("provider calls = %d after cache hit, want %d", provider.calls, firstCalls)
	}
}

func TestDescribeFleetImportsIsolatesPerScenarioErrors(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "ok")

	report, err := NewService().DescribeFleetImports(context.Background(), &factsv1.DescribeFleetImportsRequest{
		Scenarios: []string{"ok", "missing"},
		RepoRoot:  repo,
	})
	if err != nil {
		t.Fatalf("DescribeFleetImports() error = %v", err)
	}
	if len(report.GetResults()) != 2 {
		t.Fatalf("results len = %d, want 2", len(report.GetResults()))
	}
	if report.GetResults()[0].GetError() != "" {
		t.Fatalf("ok error = %q", report.GetResults()[0].GetError())
	}
	if report.GetResults()[1].GetScenario() != "missing" || report.GetResults()[1].GetError() == "" {
		t.Fatalf("missing result = %#v, want per-scenario error", report.GetResults()[1])
	}
}

func TestDescribeFleetImportsRejectsInvalidLimit(t *testing.T) {
	_, err := NewService().DescribeFleetImports(context.Background(), &factsv1.DescribeFleetImportsRequest{Limit: 501})
	if err == nil || !strings.Contains(err.Error(), "limit must be between 0 and 500") {
		t.Fatalf("error = %v, want invalid limit", err)
	}
}

func TestDescribeScenarioReportsUnknownSidecarUnsupported(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "sidecar-demo")
	writeFile(t, filepath.Join(repo, "scenarios", "sidecar-demo", "sidecar", "README.md"), "# custom runtime\n")

	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: "sidecar-demo",
			RepoRoot: repo,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	requireSurface(t, report.GetSurfaces(), "sidecar", factsv1.SurfaceStatus_SURFACE_STATUS_UNSUPPORTED)
}

func TestDescribeFileDomainFamilyWithoutProviderReportsUnsupported(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "domain-demo")

	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: "domain-demo",
			RepoRoot: repo,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	requireFactFamily(t, report.GetFacts(), factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN)
	if len(report.GetWarnings()) != 1 || !strings.Contains(report.GetWarnings()[0].GetMessage(), "architecture-cartographer") {
		t.Fatalf("expected architecture-cartographer unsupported warning, got %+v", report.GetWarnings())
	}
}

func TestDescribeFileDomainFamilyUsesProvider(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "domain-demo")
	provider := fakeFileDomainProvider{
		facts: []*factsv1.GenericFact{{
			Id:      "architecture-cartographer:file_domain:api_internal_orders_service.go",
			Family:  factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN,
			Kind:    "file_domain",
			Subject: "api/internal/orders/service.go",
			Attributes: map[string]string{
				"top_domain":           "orders",
				"tier":                 "auto_place",
				"authority_confidence": "high",
			},
			Evidence: []*factsv1.Evidence{{
				Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
				Confidence: 0.97,
				Analyzer:   "architecture-cartographer.signals",
			}},
		}},
		evidence: []*factsv1.Evidence{{
			Status:     factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN,
			Confidence: 1,
			Analyzer:   "architecture-cartographer",
		}},
	}

	report, err := NewService(WithFileDomainProvider(provider)).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_SCENARIO,
			Scenario: "domain-demo",
			RepoRoot: repo,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if len(report.GetWarnings()) != 0 {
		t.Fatalf("warnings = %#v, want none", report.GetWarnings())
	}
	if len(report.GetFacts()) != 1 {
		t.Fatalf("facts len = %d, want 1", len(report.GetFacts()))
	}
	fact := report.GetFacts()[0]
	if fact.GetKind() != "file_domain" || fact.GetAttributes()["top_domain"] != "orders" || fact.GetAttributes()["authority_confidence"] != "high" {
		t.Fatalf("file_domain fact = %#v", fact)
	}
	if len(report.GetEvidence()) != 1 || report.GetEvidence()[0].GetAnalyzer() != "architecture-cartographer" {
		t.Fatalf("evidence = %#v, want cartographer evidence", report.GetEvidence())
	}
}

func TestNormalizeFamiliesAllIncludesFileDomain(t *testing.T) {
	got := normalizeFamilies([]factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_ALL})
	for _, family := range got {
		if family == factsv1.FactFamily_FACT_FAMILY_FILE_DOMAIN {
			return
		}
	}
	t.Fatalf("FACT_FAMILY_ALL did not include FILE_DOMAIN: %+v", got)
}

func TestDescribePathInsideScenarioSurface(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "nested")
	targetPath := filepath.Join(scenarioRoot, "api", "internal", "thing")
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target: &factsv1.CodeTarget{
			Kind:     factsv1.TargetKind_TARGET_KIND_PATH,
			Path:     targetPath,
			RepoRoot: repo,
		},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_PARSE_UNITS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if report.GetTarget().GetScenario() != "nested" {
		t.Fatalf("scenario = %q, want nested", report.GetTarget().GetScenario())
	}
	requireParseUnit(t, report.GetParseUnits(), "go", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestDescribeNormalizesGoProviderFacts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "go_import_spec:main.go:fmt",
				Name: "fmt",
				Path: "main.go",
				Attributes: map[string]string{
					"kind":        "go_import_spec",
					"import_path": `"fmt"`,
					"start_line":  "3",
				},
			},
			{
				Id:   "go_call:main.go:fmt.Println",
				Name: "fmt.Println",
				Path: "main.go",
				Attributes: map[string]string{
					"kind":   "go_call",
					"callee": "fmt.Println",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if len(report.GetFacts()) != 1 {
		t.Fatalf("facts = %d, want 1 import fact: %#v", len(report.GetFacts()), report.GetFacts())
	}
	fact := report.GetFacts()[0]
	if fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_IMPORTS || fact.GetSubject() != `"fmt"` {
		t.Fatalf("fact = %#v, want fmt import", fact)
	}
	if fact.GetAttributes()["import_path"] != `"fmt"` || fact.GetEvidence()[0].GetRange().GetStartLine() != 3 {
		t.Fatalf("fact attributes/evidence not preserved: %#v", fact)
	}
}

func TestDescribeMapsGoRouteRegistrationFactsToCalls(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/routes\n\ngo 1.25\n")
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "go_route_registration:routes.go:1",
				Name: "POST /upload",
				Path: "routes.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_ROUTE_REGISTRATION",
					"route_path":       "/upload",
					"http_method":      "POST",
					"router_framework": "gorilla/mux",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_CALLS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if len(report.GetFacts()) != 1 {
		t.Fatalf("facts = %d, want 1 route call fact: %#v", len(report.GetFacts()), report.GetFacts())
	}
	fact := report.GetFacts()[0]
	if fact.GetFamily() != factsv1.FactFamily_FACT_FAMILY_CALLS || fact.GetSubject() != "/upload" {
		t.Fatalf("fact = %#v, want route mapped to calls with route_path subject", fact)
	}
	if fact.GetKind() != "go_route_registration" {
		t.Fatalf("kind = %q, want go_route_registration", fact.GetKind())
	}
}

func TestDescribeNormalizesTypeScriptProviderFacts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"build":"tsc"}}`)
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"strict":true}}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result: &GraphResult{Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "ts_import_binding:src/widget.tsx:1:1:Client",
				Name: "Client",
				Path: "src/widget.tsx",
				Attributes: map[string]string{
					"kind":          "TS_NODE_KIND_IMPORT_BINDING",
					"source_module": "./gen/client",
					"import_kind":   "namespace",
				},
			},
			{
				Id:   "ts_call:src/widget.tsx:4:3:createConnectClient",
				Name: "createConnectClient",
				Path: "src/widget.tsx",
				Attributes: map[string]string{
					"kind":                  "TS_NODE_KIND_CALL",
					"callee":                "createConnectClient",
					"enclosing_declaration": "UsageCard",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_CALLS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if len(report.GetFacts()) != 1 {
		t.Fatalf("facts = %d, want 1 call fact: %#v", len(report.GetFacts()), report.GetFacts())
	}
	if got := report.GetFacts()[0].GetAttributes()["enclosing_declaration"]; got != "UsageCard" {
		t.Fatalf("enclosing_declaration = %q, want UsageCard", got)
	}
}

func TestProtoAdoptionRecognizesGeneratedProtoTypeScriptAlias(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "proto-health")
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result: &GraphResult{Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "ts_import_binding:src/api/health.ts:1:1:ResponseSchema",
				Name: "ResponseSchema",
				Path: "src/api/health.ts",
				Attributes: map[string]string{
					"kind":          "ts_import_binding",
					"source_module": "@vrooli/proto-types/proto-health/v1/health/health_pb",
					"import_kind":   "named",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).ProtoAdoption(context.Background(), &factsv1.CheckProtoAdoptionRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		Surfaces: []string{"ui"},
	})
	if err != nil {
		t.Fatalf("ProtoAdoption() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "ui", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestProtoAdoptionMarksSurfaceUnknownWhenAnalyzerUnavailable(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "proto-health")
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		err:      ProviderUnavailableError{Analyzer: "typescript-code-graph", Err: errors.New("sidecar unavailable")},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).ProtoAdoption(context.Background(), &factsv1.CheckProtoAdoptionRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		Surfaces: []string{"ui"},
	})
	if err != nil {
		t.Fatalf("ProtoAdoption() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "ui", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)
	if len(report.GetWarnings()) == 0 || report.GetWarnings()[0].GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN {
		t.Fatalf("warnings = %#v, want unknown analyzer warning", report.GetWarnings())
	}
}

func TestProtoAdoptionScopesGeneratedImportsToSurfaceParseUnit(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	provider := unitProvider{
		language: "go",
		analyzer: "go-code-graph",
		extract: func(unit *factsv1.ParseUnit) *GraphResult {
			if unit.GetRootPath() != filepath.Join(scenarioRoot, "cli") {
				return &GraphResult{Graph: &commonv1.CodeGraph{}}
			}
			return &GraphResult{Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
				protoImportNode("domains/describe/handlers.go", "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"),
			}}}
		},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).ProtoAdoption(context.Background(), &factsv1.CheckProtoAdoptionRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		Surfaces: []string{"api", "cli"},
	})
	if err != nil {
		t.Fatalf("ProtoAdoption() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING)
	requireProofFact(t, report.GetFacts(), "cli", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestDescribeProviderUnavailableIsTypedWarning(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		err:      ProviderUnavailableError{Analyzer: "go-code-graph", Err: errors.New("connection refused")},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if len(report.GetWarnings()) == 0 || report.GetWarnings()[0].GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN {
		t.Fatalf("warnings = %#v, want unknown provider warning", report.GetWarnings())
	}
}

func TestDescribeProviderUnavailableStrictReturnsError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		err:      ProviderUnavailableError{Analyzer: "go-code-graph", Err: errors.New("connection refused")},
	}

	_, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root, Strict: true},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err == nil {
		t.Fatal("Describe() error = nil, want strict provider error")
	}
}

func TestClassifyProviderErrorMapsUnimplementedToUnsupported(t *testing.T) {
	err := classifyProviderError("typescript-code-graph", connect.NewError(connect.CodeUnimplemented, errors.New("workspace_unsupported")))
	var unsupported ProviderUnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("classified error = %T %[1]v, want ProviderUnsupportedError", err)
	}
}

func TestDescribeProviderUnsupportedIsTypedWarning(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"strict":true}}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		err:      ProviderUnsupportedError{Analyzer: "typescript-code-graph", Err: errors.New("workspace_unsupported")},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if len(report.GetWarnings()) == 0 || report.GetWarnings()[0].GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED {
		t.Fatalf("warnings = %#v, want unsupported provider warning", report.GetWarnings())
	}
}

func TestDescribeProviderUnsupportedStrictReturnsError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions":{"strict":true}}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		err:      ProviderUnsupportedError{Analyzer: "typescript-code-graph", Err: errors.New("workspace_unsupported")},
	}

	_, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, Path: root, Strict: true},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err == nil {
		t.Fatal("Describe() error = nil, want strict provider error")
	}
}

func TestDescribeHonorsLanguageFilter(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "proto-health")
	goProvider := &countingProvider{fakeProvider: fakeProvider{language: "go", analyzer: "go-code-graph"}}
	tsProvider := &countingProvider{fakeProvider: fakeProvider{language: "typescript", analyzer: "typescript-code-graph"}}

	_, err := NewService(WithBroker(NewBroker(goProvider, tsProvider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo, LanguageFilter: []string{"go"}},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if goProvider.calls != 3 {
		t.Fatalf("go provider calls = %d, want 3", goProvider.calls)
	}
	if tsProvider.calls != 0 {
		t.Fatalf("typescript provider calls = %d, want 0", tsProvider.calls)
	}
}

func TestDescribeCachesRepeatedSelectiveFactRequest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n\nimport \"fmt\"\n")
	provider := &countingProvider{fakeProvider: fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "graph-one", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "go_import_spec:main.go:fmt",
				Name: "fmt",
				Path: "main.go",
				Attributes: map[string]string{
					"kind":        "GO_NODE_KIND_IMPORT_SPEC",
					"import_path": `"fmt"`,
				},
			},
		}}},
	}}
	svc := NewService(WithBroker(NewBroker(provider)))
	req := &factsv1.DescribeCodeFactsRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root},
		Include:  []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		UseCache: true,
	}

	first, err := svc.Describe(context.Background(), req)
	if err != nil {
		t.Fatalf("first Describe() error = %v", err)
	}
	second, err := svc.Describe(context.Background(), req)
	if err != nil {
		t.Fatalf("second Describe() error = %v", err)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	if first.GetCache().GetHit() {
		t.Fatalf("first cache hit = true, want miss")
	}
	if !second.GetCache().GetHit() || second.GetCache().GetState() != "hit" {
		t.Fatalf("second cache metadata = %#v, want hit", second.GetCache())
	}
	status, err := svc.CacheStatus(context.Background(), &factsv1.GetCacheStatusRequest{Target: req.GetTarget()})
	if err != nil {
		t.Fatalf("CacheStatus() error = %v", err)
	}
	if status.GetEntries() != 2 {
		t.Fatalf("cache entries = %d, want graph + report entries", status.GetEntries())
	}
}

func TestDescribeCacheInvalidatesWhenSourceChanges(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	sourcePath := filepath.Join(root, "main.go")
	writeFile(t, sourcePath, "package main\n\nimport \"fmt\"\n")
	provider := &countingProvider{fakeProvider: fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "graph-one", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "go_import_spec:main.go:fmt",
				Name: "fmt",
				Path: "main.go",
				Attributes: map[string]string{
					"kind":        "GO_NODE_KIND_IMPORT_SPEC",
					"import_path": `"fmt"`,
				},
			},
		}}},
	}}
	svc := NewService(WithBroker(NewBroker(provider)))
	req := &factsv1.DescribeCodeFactsRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root},
		Include:  []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
		UseCache: true,
	}

	if _, err := svc.Describe(context.Background(), req); err != nil {
		t.Fatalf("first Describe() error = %v", err)
	}
	writeFile(t, sourcePath, "package main\n\nimport \"os\"\n")
	second, err := svc.Describe(context.Background(), req)
	if err != nil {
		t.Fatalf("second Describe() error = %v", err)
	}
	if provider.calls != 2 {
		t.Fatalf("provider calls = %d, want 2 after source hash changed", provider.calls)
	}
	if second.GetCache().GetHit() || second.GetCache().GetState() != "miss" {
		t.Fatalf("second cache metadata = %#v, want miss after source change", second.GetCache())
	}
}

func TestSourceFingerprintIgnoresMTimeOnlyChurn(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	writeFile(t, sourcePath, "package main\n")
	first := fileStatSignature(root, []string{sourcePath})
	nextTime := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(sourcePath, nextTime, nextTime); err != nil {
		t.Fatalf("touch source: %v", err)
	}
	second := fileStatSignature(root, []string{sourcePath})
	if second != first {
		t.Fatalf("signature changed after mtime-only touch: %s != %s", second, first)
	}
}

func TestSourceFingerprintChangesWhenContentChanges(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	writeFile(t, sourcePath, "package main\n")
	first := fileStatSignature(root, []string{sourcePath})
	writeFile(t, sourcePath, "package main\n\nfunc main() {}\n")
	second := fileStatSignature(root, []string{sourcePath})
	if second == first {
		t.Fatalf("signature did not change after content edit: %s", second)
	}
}

func TestCacheSupersedeOnPutMemory(t *testing.T) {
	exerciseCacheSupersedeOnPut(t, NewMemoryCacheRepository())
}

func TestCacheSupersedeOnPutSQLite(t *testing.T) {
	db := openCacheTestDB(t)
	exerciseCacheSupersedeOnPut(t, NewSQLiteCacheRepository(db))
}

func TestCacheDistinctLogicalIdentitiesCoexistMemory(t *testing.T) {
	exerciseCacheDistinctLogicalIdentitiesCoexist(t, NewMemoryCacheRepository())
}

func TestCacheDistinctLogicalIdentitiesCoexistSQLite(t *testing.T) {
	db := openCacheTestDB(t)
	exerciseCacheDistinctLogicalIdentitiesCoexist(t, NewSQLiteCacheRepository(db))
}

func TestMigrateCacheSchemaIsIdempotentOnPopulatedLegacyTable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
CREATE TABLE code_facts_cache_entries (
  cache_key TEXT PRIMARY KEY,
  scope TEXT NOT NULL,
  target_root TEXT NOT NULL,
  analyzer TEXT NOT NULL,
  provider TEXT NOT NULL,
  provider_version TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  graph_hash TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  config_hash TEXT NOT NULL,
  family_key TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  warnings_json TEXT NOT NULL DEFAULT '[]',
  extraction_ms INTEGER NOT NULL DEFAULT 0,
  created_at_unix INTEGER NOT NULL,
  last_used_at_unix INTEGER NOT NULL,
  hit_count INTEGER NOT NULL DEFAULT 0
)`)
	if err != nil {
		t.Fatalf("create legacy cache table: %v", err)
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO code_facts_cache_entries (
  cache_key, scope, target_root, analyzer, provider, provider_version, schema_version,
  graph_hash, source_hash, config_hash, family_key, payload_json, warnings_json,
  extraction_ms, created_at_unix, last_used_at_unix, hit_count
) VALUES ('legacy-key', 'graph', '/target', 'analyzer', 'provider', 'provider:v1', ?,
  'graph', 'source', 'config', 'go', '{"nodes":[]}', '[]', 7, 1, 1, 0)`, cacheSchemaVersion)
	if err != nil {
		t.Fatalf("seed legacy cache table: %v", err)
	}
	if err := MigrateCacheSchema(ctx, db); err != nil {
		t.Fatalf("first migration: %v", err)
	}
	if err := MigrateCacheSchema(ctx, db); err != nil {
		t.Fatalf("second migration: %v", err)
	}
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(CacheSchema)); err != nil {
		t.Fatalf("EnsureSchemas after migration: %v", err)
	}
	var logicalKey string
	var payloadBytes int64
	if err := db.QueryRowContext(ctx, `SELECT logical_key, payload_bytes FROM code_facts_cache_entries WHERE cache_key = 'legacy-key'`).Scan(&logicalKey, &payloadBytes); err != nil {
		t.Fatalf("read migrated cache row: %v", err)
	}
	if logicalKey != "legacy-key" {
		t.Fatalf("logical_key = %q, want legacy-key", logicalKey)
	}
	if payloadBytes != int64(len(`{"nodes":[]}`)) {
		t.Fatalf("payload_bytes = %d, want %d", payloadBytes, len(`{"nodes":[]}`))
	}
}

func TestCacheBudgetEvictsLRUAndRefreshChangesOrderMemory(t *testing.T) {
	exerciseCacheBudgetEvictsLRUAndRefreshChangesOrder(t, func(maxBytes int64) CacheRepository {
		return NewMemoryCacheRepository(maxBytes)
	})
}

func TestCacheBudgetEvictsLRUAndRefreshChangesOrderSQLite(t *testing.T) {
	db := openCacheTestDB(t)
	exerciseCacheBudgetEvictsLRUAndRefreshChangesOrder(t, func(maxBytes int64) CacheRepository {
		return NewSQLiteCacheRepository(db, maxBytes)
	})
}

func TestCacheSweepDropsStaleSchemaAndEnforcesBudgetMemory(t *testing.T) {
	repo := NewMemoryCacheRepository(0)
	exerciseCacheSweepDropsStaleSchemaAndEnforcesBudget(t, repo, func(maxBytes int64) CacheRepository {
		mem := repo.(*memoryCacheRepository)
		mem.maxBytes = maxBytes
		return repo
	})
}

func TestCacheSweepDropsStaleSchemaAndEnforcesBudgetSQLite(t *testing.T) {
	db := openCacheTestDB(t)
	repo := NewSQLiteCacheRepository(db, 0)
	exerciseCacheSweepDropsStaleSchemaAndEnforcesBudget(t, repo, func(maxBytes int64) CacheRepository {
		return NewSQLiteCacheRepository(db, maxBytes)
	})
}

func TestCachePayloadRoundTripMemory(t *testing.T) {
	exerciseCachePayloadRoundTrip(t, NewMemoryCacheRepository())
}

func TestCachePayloadRoundTripSQLite(t *testing.T) {
	db := openCacheTestDB(t)
	exerciseCachePayloadRoundTrip(t, NewSQLiteCacheRepository(db, 0))
}

func TestCacheStatusReportsBudgetTotalsAndScopes(t *testing.T) {
	repo := NewMemoryCacheRepository(4096)
	ctx := context.Background()
	targetRoot := t.TempDir()
	graphEntry := cacheTestEntry("status-graph", "status-graph", "source")
	graphEntry.TargetRoot = targetRoot
	reportEntry := cacheTestEntry("status-report", "status-report", "source")
	reportEntry.TargetRoot = targetRoot
	reportEntry.FamilyKey = "FACT_FAMILY_IMPORTS"
	if err := repo.PutGraph(ctx, graphEntry, cacheTestGraph("status-graph")); err != nil {
		t.Fatalf("PutGraph: %v", err)
	}
	if err := repo.PutReport(ctx, reportEntry, &factsv1.CodeFactsReport{
		Target: &factsv1.TargetContext{RootPath: targetRoot},
		Facts:  []*factsv1.GenericFact{{Family: factsv1.FactFamily_FACT_FAMILY_IMPORTS, Subject: "fmt"}},
	}); err != nil {
		t.Fatalf("PutReport: %v", err)
	}
	svc := NewService(WithCacheRepository(repo))
	status, err := svc.CacheStatus(ctx, &factsv1.GetCacheStatusRequest{Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: targetRoot}})
	if err != nil {
		t.Fatalf("CacheStatus: %v", err)
	}
	if status.GetEntries() != 2 || status.GetTotalRows() != 2 {
		t.Fatalf("entries=%d total_rows=%d, want 2/2", status.GetEntries(), status.GetTotalRows())
	}
	if status.GetBudgetBytes() != 4096 {
		t.Fatalf("budget bytes = %d, want 4096", status.GetBudgetBytes())
	}
	if status.GetTotalPayloadBytes() == 0 || status.GetUtilization() <= 0 {
		t.Fatalf("payload bytes=%d utilization=%f, want positive", status.GetTotalPayloadBytes(), status.GetUtilization())
	}
	if len(status.GetScopes()) != 2 {
		t.Fatalf("scopes = %d, want graph and report", len(status.GetScopes()))
	}
}

func TestClearCacheAllDryRunAndDelete(t *testing.T) {
	repo := NewMemoryCacheRepository()
	ctx := context.Background()
	targetRoot := t.TempDir()
	first := cacheTestEntry("clear-a", "clear-a", "source")
	first.TargetRoot = targetRoot
	if err := repo.PutGraph(ctx, first, cacheTestGraph("clear-a")); err != nil {
		t.Fatalf("PutGraph(a): %v", err)
	}
	second := cacheTestEntry("clear-b", "clear-b", "source")
	second.TargetRoot = t.TempDir()
	if err := repo.PutGraph(ctx, second, cacheTestGraph("clear-b")); err != nil {
		t.Fatalf("PutGraph(b): %v", err)
	}
	svc := NewService(WithCacheRepository(repo))
	dryRun, err := svc.ClearCache(ctx, &factsv1.ClearCacheRequest{All: true, DryRun: true})
	if err != nil {
		t.Fatalf("ClearCache(all dry-run): %v", err)
	}
	if dryRun.GetMatchedEntries() != 2 || dryRun.GetClearedEntries() != 0 {
		t.Fatalf("dry-run = %#v, want 2 matched and 0 cleared", dryRun)
	}
	status, err := svc.CacheStatus(ctx, &factsv1.GetCacheStatusRequest{Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: targetRoot}})
	if err != nil {
		t.Fatalf("CacheStatus after dry-run: %v", err)
	}
	if status.GetTotalRows() != 2 {
		t.Fatalf("total rows after dry-run = %d, want 2", status.GetTotalRows())
	}
	cleared, err := svc.ClearCache(ctx, &factsv1.ClearCacheRequest{All: true})
	if err != nil {
		t.Fatalf("ClearCache(all): %v", err)
	}
	if cleared.GetMatchedEntries() != 2 || cleared.GetClearedEntries() != 2 {
		t.Fatalf("cleared = %#v, want 2 matched and 2 cleared", cleared)
	}
	status, err = svc.CacheStatus(ctx, &factsv1.GetCacheStatusRequest{Target: &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: targetRoot}})
	if err != nil {
		t.Fatalf("CacheStatus after clear: %v", err)
	}
	if status.GetTotalRows() != 0 {
		t.Fatalf("total rows after clear = %d, want 0", status.GetTotalRows())
	}
}

func TestSQLiteCacheCompressesGraphPayload(t *testing.T) {
	db := openCacheTestDB(t)
	repo := NewSQLiteCacheRepository(db, 0)
	graph := largeCacheTestGraph(2000)
	rawPayload, _, err := marshalGraphResult(graph)
	if err != nil {
		t.Fatalf("marshal raw graph: %v", err)
	}
	entry := cacheTestEntry("compressed-key", "compressed-unit", "source")
	if err := repo.PutGraph(context.Background(), entry, graph); err != nil {
		t.Fatalf("PutGraph: %v", err)
	}
	entries, err := repo.Status(context.Background(), "/target", "")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Codec != cacheCodecGzip {
		t.Fatalf("codec = %q, want gzip", entries[0].Codec)
	}
	if entries[0].PayloadBytes >= int64(len(rawPayload))/3 {
		t.Fatalf("compressed payload bytes = %d, raw bytes = %d, want at least 3x reduction", entries[0].PayloadBytes, len(rawPayload))
	}
}

func TestClearCacheDryRunDoesNotDeleteEntries(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/generic\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	provider := &countingProvider{fakeProvider: fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result:   &GraphResult{GraphHash: "graph-one", Graph: &commonv1.CodeGraph{}},
	}}
	svc := NewService(WithBroker(NewBroker(provider)))
	target := &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_MODULE, Path: root}
	if _, err := svc.Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target: target, Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS}, UseCache: true,
	}); err != nil {
		t.Fatalf("Describe() error = %v", err)
	}

	dryRun, err := svc.ClearCache(context.Background(), &factsv1.ClearCacheRequest{Target: target, DryRun: true})
	if err != nil {
		t.Fatalf("ClearCache(dry-run) error = %v", err)
	}
	if dryRun.GetMatchedEntries() == 0 || dryRun.GetClearedEntries() != 0 {
		t.Fatalf("dry-run response = %#v, want matched without clear", dryRun)
	}
	status, err := svc.CacheStatus(context.Background(), &factsv1.GetCacheStatusRequest{Target: target})
	if err != nil {
		t.Fatalf("CacheStatus() error = %v", err)
	}
	if status.GetEntries() != dryRun.GetMatchedEntries() {
		t.Fatalf("entries after dry-run = %d, want %d", status.GetEntries(), dryRun.GetMatchedEntries())
	}
}

func TestDescribeUnsupportedParseUnitDoesNotCallProvider(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{"scripts":{"build":"node index.js"}}`)
	writeFile(t, filepath.Join(root, "index.js"), "console.log('hi')\n")
	provider := &countingProvider{fakeProvider: fakeProvider{language: "go", analyzer: "go-code-graph"}}

	report, err := NewService(WithBroker(NewBroker(provider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PATH, Path: root},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_IMPORTS},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
	if len(report.GetWarnings()) == 0 || report.GetWarnings()[0].GetStatus() != factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED {
		t.Fatalf("warnings = %#v, want unsupported parse unit warning", report.GetWarnings())
	}
}

func TestProtoAdoptionProvesGeneratedImportsBySurface(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "proto-health")
	provider := unitProvider{language: "go", analyzer: "go-code-graph", extract: func(unit *factsv1.ParseUnit) *GraphResult {
		nodes := []*commonv1.CodeGraphNode{}
		if filepath.Base(unit.GetRootPath()) == "api" {
			nodes = append(nodes, protoImportNode("api/handlers/health/handler.go", "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/health"))
		}
		if filepath.Base(unit.GetRootPath()) == "cli" {
			nodes = append(nodes, protoImportNode("cli/domains/validation/handlers.go", "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"))
		}
		return &GraphResult{GraphHash: unit.GetId(), Graph: &commonv1.CodeGraph{Nodes: nodes}}
	}}

	report, err := NewService(WithBroker(NewBroker(provider))).ProtoAdoption(context.Background(), &factsv1.CheckProtoAdoptionRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		Surfaces: []string{"api", "cli"},
	})
	if err != nil {
		t.Fatalf("ProtoAdoption() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
	requireProofFact(t, report.GetFacts(), "cli", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestProtoAdoptionReportsMissingGeneratedImport(t *testing.T) {
	repo, _ := writeScenarioFixture(t, "demo")

	report, err := NewService(WithBroker(NewBroker(fakeProvider{language: "go", analyzer: "go-code-graph"}))).ProtoAdoption(context.Background(), &factsv1.CheckProtoAdoptionRequest{
		Target:   &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo", RepoRoot: repo},
		Surfaces: []string{"api"},
	})
	if err != nil {
		t.Fatalf("ProtoAdoption() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "api", factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING)
}

func TestEndpointProofProvesRESTExceptionPayload(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"notes_attach",
  "path":"/api/v1/notes/{id}/attachments",
  "method":"POST",
  "category":"notes",
  "rest_exception":{
    "reason":"multipart_upload",
    "proto_payloads":{
      "request":{"transport":"multipart/form-data","conformance":"transport_only"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.notes.UploadAttachmentResponse","transport":"json","conformance":"protojson"},
      "error":{"proto_full_name":"vrooli.proto_health.v1.shared.ErrorEnvelope","transport":"json","conformance":"protojson"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "endpoint-proof", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "route",
				Name: "Methods",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_ROUTE_REGISTRATION",
					"callee":           "r.HandleFunc(...).Methods",
					"route_path":       "/api/v1/notes/{id}/attachments",
					"http_method":      "POST",
					"handler_expr":     "h.handleUpload",
					"enclosing_symbol": "Module",
				},
			},
			{
				Id:   "response",
				Name: "httpx.WriteProto",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "httpx.WriteProto",
					"argument_types":   "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/notes.UploadAttachmentResponse",
					"enclosing_symbol": "*attachmentsHandler.handleUpload",
				},
			},
			{
				Id:   "error",
				Name: "httpx.WriteError",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "httpx.WriteError",
					"argument_types":   "net/http.ResponseWriter, int, string, string",
					"enclosing_symbol": "*attachmentsHandler.handleUpload",
				},
			},
			{
				Id:   "error-helper",
				Name: "protojson.Marshal",
				Path: "api/internal/httpx/errors.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "protojson.Marshal",
					"argument_types":   "*github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared.ErrorEnvelope",
					"enclosing_symbol": "WriteError",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"notes_attach"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "notes_attach", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestEndpointProofDoesNotBorrowPayloadFromDifferentHandler(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"proto_full_name":"vrooli.proto_health.v1.shared.ErrorEnvelope","transport":"json","conformance":"protojson"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "endpoint-proof", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "health-route",
				Name: "GET /health",
				Path: "api/handlers/health/module.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_ROUTE_REGISTRATION",
					"route_path":       "/health",
					"http_method":      "GET",
					"handler_expr":     "h",
					"enclosing_symbol": "Module",
				},
			},
			{
				Id:   "notes-response",
				Name: "httpx.WriteProto",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "httpx.WriteProto",
					"argument_types":   "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/health.Response",
					"enclosing_symbol": "*attachmentsHandler.handleUpload",
				},
			},
			{
				Id:   "notes-error",
				Name: "httpx.WriteError",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "httpx.WriteError",
					"argument_types":   "net/http.ResponseWriter, int, string, string",
					"enclosing_symbol": "*attachmentsHandler.handleUpload",
				},
			},
			{
				Id:   "error-helper",
				Name: "protojson.Marshal",
				Path: "api/internal/httpx/errors.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "protojson.Marshal",
					"argument_types":   "*github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared.ErrorEnvelope",
					"enclosing_symbol": "WriteError",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)
}

func TestEndpointProofCorrelatesHandlerFactoryClosure(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "endpoint-proof", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "factory",
				Name: "NewHandler",
				Path: "api/handlers/health/module.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "NewHandler",
					"file_id":          "file:handlers/health/module.go",
					"enclosing_symbol": "Module",
					"start_line":       "19",
				},
			},
			{
				Id:   "health-route",
				Name: "GET /health",
				Path: "api/handlers/health/module.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_ROUTE_REGISTRATION",
					"route_path":       "/health",
					"http_method":      "GET",
					"handler_expr":     "h",
					"handler_symbol":   "go_var:package:proto-health/handlers/health:h",
					"file_id":          "file:handlers/health/module.go",
					"enclosing_symbol": "Module",
					"start_line":       "23",
				},
			},
			{
				Id:   "response",
				Name: "httpx.WriteProto",
				Path: "api/handlers/health/handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "httpx.WriteProto",
					"argument_types":   "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/health.Response",
					"enclosing_symbol": "NewHandler",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestEndpointProofAnalyzesTypeScriptProviderByDefault(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), endpointProofFixtureJSON())
	goProvider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result:   endpointProofGraphResult(),
	}
	tsProvider := &countingProvider{fakeProvider: fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		err:      ProviderUnsupportedError{Analyzer: "typescript-code-graph", Err: errors.New("workspace_unsupported")},
	}}

	report, err := NewService(WithBroker(NewBroker(goProvider, tsProvider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"notes_attach"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	if tsProvider.calls != 1 {
		t.Fatalf("typescript provider calls = %d, want 1", tsProvider.calls)
	}
	requireProofFact(t, report.GetFacts(), "notes_attach", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestDescribeEndpointProofAnalyzesTypeScriptProviderByDefault(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), endpointProofFixtureJSON())
	goProvider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result:   endpointProofGraphResult(),
	}
	tsProvider := &countingProvider{fakeProvider: fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		err:      ProviderUnsupportedError{Analyzer: "typescript-code-graph", Err: errors.New("workspace_unsupported")},
	}}

	report, err := NewService(WithBroker(NewBroker(goProvider, tsProvider))).Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		Include:     []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_ENDPOINT_PROOFS},
		EndpointIds: []string{"notes_attach"},
	})
	if err != nil {
		t.Fatalf("Describe() error = %v", err)
	}
	if tsProvider.calls != 1 {
		t.Fatalf("typescript provider calls = %d, want 1", tsProvider.calls)
	}
	requireProofFact(t, report.GetFacts(), "notes_attach", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestEndpointProofProvesTypeScriptExpressResponse(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result:   tsExpressGraphResult("res.json", "@vrooli/proto-types/proto-health/v1/health.Response"),
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestEndpointProofUsesProvenTypeScriptWhenGoImplementationUnknown(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	goProvider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "go-mismatch", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "wrong-route",
				Name: "POST /health",
				Path: "api/server.go",
				Attributes: map[string]string{
					"kind":        "GO_NODE_KIND_ROUTE_REGISTRATION",
					"route_path":  "/health",
					"http_method": "POST",
				},
			},
		}}},
	}
	tsProvider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result:   tsExpressGraphResult("res.json", "@vrooli/proto-types/proto-health/v1/health.Response"),
	}

	report, err := NewService(WithBroker(NewBroker(goProvider, tsProvider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
}

func TestEndpointProofContradictsTypeScriptExpressWrongResponse(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result:   tsExpressGraphResult("res.json", "@vrooli/proto-types/proto-health/v1/notes.UploadAttachmentResponse"),
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED)
}

func TestEndpointProofTypeScriptExpressMissingPayload(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result: &GraphResult{GraphHash: "ts-express", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			tsExpressRouteNode(),
			{
				Id:   "response",
				Name: "res.status",
				Path: "api/server.ts",
				Attributes: map[string]string{
					"kind":                  "TS_NODE_KIND_CALL",
					"callee":                "res.status",
					"argument_summary":      "number",
					"enclosing_declaration": "healthHandler",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_MISSING)
}

func TestEndpointProofTypeScriptExpressImportOnlyUnknown(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result: &GraphResult{GraphHash: "ts-express", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			tsExpressRouteNode(),
			tsImportNode("@vrooli/proto-types/proto-health/v1/health"),
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)
}

func TestEndpointProofWarnsForUnsupportedTypeScriptFramework(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"health",
  "path":"/health",
  "method":"GET",
  "rest_exception":{
    "reason":"ops_probe",
    "proto_payloads":{
      "request":{"transport":"none","conformance":"none"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.health.Response","transport":"json","conformance":"protojson"},
      "error":{"transport":"json","conformance":"external_shape"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "typescript",
		analyzer: "typescript-code-graph",
		result: &GraphResult{GraphHash: "ts-fastify", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "route",
				Name: "GET /health",
				Path: "api/server.ts",
				Attributes: map[string]string{
					"kind":             "TS_NODE_KIND_ROUTE_REGISTRATION",
					"router_framework": "fastify",
					"route_path":       "/health",
					"http_method":      "GET",
					"handler_expr":     "healthHandler",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"health"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "health", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)
	for _, warning := range report.GetWarnings() {
		if warning.GetCode() == "code-facts.endpoint_proof.framework_unsupported" && warning.GetStatus() == factsv1.EvidenceStatus_EVIDENCE_STATUS_UNSUPPORTED {
			return
		}
	}
	t.Fatalf("warnings = %#v, want framework_unsupported unsupported warning", report.GetWarnings())
}

func TestEndpointProofLeavesMismatchedRouteUnknown(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"notes_attach",
  "path":"/api/v1/notes/{id}/attachments",
  "method":"POST",
  "rest_exception":{
    "reason":"multipart_upload",
    "proto_payloads":{
      "request":{"transport":"multipart/form-data","conformance":"transport_only"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.notes.UploadAttachmentResponse","transport":"json","conformance":"protojson"},
      "error":{"proto_full_name":"vrooli.proto_health.v1.shared.ErrorEnvelope","transport":"json","conformance":"protojson"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "endpoint-proof", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "route",
				Name: "GET /api/v1/notes/{id}/attachments",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":        "GO_NODE_KIND_ROUTE_REGISTRATION",
					"route_path":  "/api/v1/notes/{id}/attachments",
					"http_method": "GET",
				},
			},
			{
				Id:   "response",
				Name: "httpx.WriteProto",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":           "GO_NODE_KIND_CALL",
					"callee":         "httpx.WriteProto",
					"argument_types": "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/notes.UploadAttachmentResponse",
				},
			},
			{
				Id:   "error",
				Name: "httpx.WriteError",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":           "GO_NODE_KIND_CALL",
					"callee":         "httpx.WriteError",
					"argument_types": "net/http.ResponseWriter, int, github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared.ErrorEnvelope",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"notes_attach"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "notes_attach", factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN)
}

func TestEndpointProofContradictsWrongResponseType(t *testing.T) {
	repo, scenarioRoot := writeScenarioFixture(t, "proto-health")
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[{
  "id":"notes_attach",
  "path":"/api/v1/notes/{id}/attachments",
  "method":"POST",
  "rest_exception":{
    "reason":"multipart_upload",
    "proto_payloads":{
      "request":{"transport":"multipart/form-data","conformance":"transport_only"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.notes.UploadAttachmentResponse","transport":"json","conformance":"protojson"},
      "error":{"proto_full_name":"vrooli.proto_health.v1.shared.ErrorEnvelope","transport":"json","conformance":"protojson"}
    }
  }
}]}`)
	provider := fakeProvider{
		language: "go",
		analyzer: "go-code-graph",
		result: &GraphResult{GraphHash: "endpoint-proof", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
			{
				Id:   "route",
				Name: "Methods",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_ROUTE_REGISTRATION",
					"callee":           "r.HandleFunc(...).Methods",
					"route_path":       "/api/v1/notes/{id}/attachments",
					"http_method":      "POST",
					"handler_expr":     "h.handleUpload",
					"enclosing_symbol": "Module",
				},
			},
			{
				Id:   "response",
				Name: "httpx.WriteProto",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":             "GO_NODE_KIND_CALL",
					"callee":           "httpx.WriteProto",
					"argument_types":   "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/notes.Note",
					"enclosing_symbol": "handleUpload",
				},
			},
		}}},
	}

	report, err := NewService(WithBroker(NewBroker(provider))).EndpointProof(context.Background(), &factsv1.CheckEndpointProofRequest{
		Target:      &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "proto-health", RepoRoot: repo},
		EndpointIds: []string{"notes_attach"},
	})
	if err != nil {
		t.Fatalf("EndpointProof() error = %v", err)
	}
	requireProofFact(t, report.GetFacts(), "notes_attach", factsv1.EvidenceStatus_EVIDENCE_STATUS_CONTRADICTED)
}

func endpointProofFixtureJSON() string {
	return `{"endpoints":[{
  "id":"notes_attach",
  "path":"/api/v1/notes/{id}/attachments",
  "method":"POST",
  "category":"notes",
  "rest_exception":{
    "reason":"multipart_upload",
    "proto_payloads":{
      "request":{"transport":"multipart/form-data","conformance":"transport_only"},
      "response":{"proto_full_name":"vrooli.proto_health.v1.notes.UploadAttachmentResponse","transport":"json","conformance":"protojson"},
      "error":{"proto_full_name":"vrooli.proto_health.v1.shared.ErrorEnvelope","transport":"json","conformance":"protojson"}
    }
  }
}]}`
}

func endpointProofGraphResult() *GraphResult {
	return &GraphResult{GraphHash: "endpoint-proof", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
		{
			Id:   "route",
			Name: "Methods",
			Path: "api/handlers/notes/attachments_handler.go",
			Attributes: map[string]string{
				"kind":             "GO_NODE_KIND_ROUTE_REGISTRATION",
				"callee":           "r.HandleFunc(...).Methods",
				"route_path":       "/api/v1/notes/{id}/attachments",
				"http_method":      "POST",
				"handler_expr":     "h.handleUpload",
				"enclosing_symbol": "Module",
			},
		},
		{
			Id:   "response",
			Name: "httpx.WriteProto",
			Path: "api/handlers/notes/attachments_handler.go",
			Attributes: map[string]string{
				"kind":             "GO_NODE_KIND_CALL",
				"callee":           "httpx.WriteProto",
				"argument_types":   "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/notes.UploadAttachmentResponse",
				"enclosing_symbol": "*attachmentsHandler.handleUpload",
			},
		},
		{
			Id:   "error",
			Name: "httpx.WriteError",
			Path: "api/handlers/notes/attachments_handler.go",
			Attributes: map[string]string{
				"kind":             "GO_NODE_KIND_CALL",
				"callee":           "httpx.WriteError",
				"argument_types":   "net/http.ResponseWriter, int, string, string",
				"enclosing_symbol": "*attachmentsHandler.handleUpload",
			},
		},
		{
			Id:   "error-helper",
			Name: "protojson.Marshal",
			Path: "api/internal/httpx/errors.go",
			Attributes: map[string]string{
				"kind":             "GO_NODE_KIND_CALL",
				"callee":           "protojson.Marshal",
				"argument_types":   "*github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/shared.ErrorEnvelope",
				"enclosing_symbol": "WriteError",
			},
		},
	}}}
}

func tsExpressGraphResult(responseCallee, responseType string) *GraphResult {
	return &GraphResult{GraphHash: "ts-express", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
		tsExpressRouteNode(),
		tsImportNode("@vrooli/proto-types/proto-health/v1/health"),
		{
			Id:   "response",
			Name: responseCallee,
			Path: "api/server.ts",
			Attributes: map[string]string{
				"kind":                  "TS_NODE_KIND_CALL",
				"callee":                responseCallee,
				"argument_summary":      responseType,
				"enclosing_declaration": "healthHandler",
			},
		},
	}}}
}

func tsExpressRouteNode() *commonv1.CodeGraphNode {
	return &commonv1.CodeGraphNode{
		Id:   "route",
		Name: "GET /health",
		Path: "api/server.ts",
		Attributes: map[string]string{
			"kind":              "TS_NODE_KIND_ROUTE_REGISTRATION",
			"router_framework":  "express",
			"route_path":        "/health",
			"route_path_status": "proven",
			"http_method":       "GET",
			"handler_expr":      "healthHandler",
			"handler_symbol":    "healthHandler",
		},
	}
}

func tsImportNode(sourceModule string) *commonv1.CodeGraphNode {
	return &commonv1.CodeGraphNode{
		Id:   "import:" + sourceModule,
		Name: "Response",
		Path: "api/server.ts",
		Attributes: map[string]string{
			"kind":          "TS_NODE_KIND_IMPORT_BINDING",
			"source_module": sourceModule,
			"local_name":    "Response",
		},
	}
}

func writeScenarioFixture(t *testing.T, name string) (repo string, scenarioRoot string) {
	t.Helper()
	repo = t.TempDir()
	scenarioRoot = writeScenarioFixtureInRepo(t, repo, name)
	return repo, scenarioRoot
}

func writeScenarioFixtureInRepo(t *testing.T, repo string, name string) string {
	t.Helper()
	scenarioRoot := filepath.Join(repo, "scenarios", name)
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "service.json"), `{
  "cli": {"enabled": true, "adapter": {"kind": "go_module", "module_dir": "cli"}}
}`)
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[]}`)
	writeFile(t, filepath.Join(scenarioRoot, "api", "go.mod"), "module demo/api\n\ngo 1.25\n")
	writeFile(t, filepath.Join(scenarioRoot, "cli", "go.mod"), "module demo/cli\n\ngo 1.25\n")
	writeFile(t, filepath.Join(scenarioRoot, "cli", "manifest.json"), `{"groups":[]}`)
	writeFile(t, filepath.Join(scenarioRoot, "runtime", "go.mod"), "module demo/runtime\n\ngo 1.25\n")
	writeFile(t, filepath.Join(scenarioRoot, "ui", "package.json"), `{"scripts":{"build":"vite"}}`)
	writeFile(t, filepath.Join(scenarioRoot, "ui", "tsconfig.json"), `{"compilerOptions":{"jsx":"react-jsx"}}`)
	return scenarioRoot
}

func protoImportNode(path, importPath string) *commonv1.CodeGraphNode {
	return &commonv1.CodeGraphNode{
		Id:   "import:" + path + ":" + importPath,
		Name: filepath.Base(importPath),
		Path: path,
		Attributes: map[string]string{
			"kind":        "GO_NODE_KIND_IMPORT_SPEC",
			"import_path": importPath,
		},
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func requireSurface(t *testing.T, surfaces []*factsv1.Surface, id string, status factsv1.SurfaceStatus) {
	t.Helper()
	for _, surface := range surfaces {
		if surface.GetId() == id {
			if surface.GetStatus() != status {
				t.Fatalf("surface %s status = %s, want %s", id, surface.GetStatus(), status)
			}
			return
		}
	}
	t.Fatalf("surface %s not found in %#v", id, surfaces)
}

func requireSurfaceKind(t *testing.T, surfaces []*factsv1.Surface, id string, kind factsv1.SurfaceKind) {
	t.Helper()
	for _, surface := range surfaces {
		if surface.GetId() == id {
			if surface.GetKind() != kind {
				t.Fatalf("surface %s kind = %s, want %s", id, surface.GetKind(), kind)
			}
			return
		}
	}
	t.Fatalf("surface %s not found in %#v", id, surfaces)
}

func requireParseUnit(t *testing.T, units []*factsv1.ParseUnit, language string, status factsv1.EvidenceStatus) {
	t.Helper()
	for _, unit := range units {
		if unit.GetLanguage() == language {
			if unit.GetStatus() != status {
				t.Fatalf("parse unit %s status = %s, want %s", language, unit.GetStatus(), status)
			}
			return
		}
	}
	t.Fatalf("parse unit %s not found in %#v", language, units)
}

func requireFactFamily(t *testing.T, facts []*factsv1.GenericFact, family factsv1.FactFamily) {
	t.Helper()
	for _, fact := range facts {
		if fact.GetFamily() == family {
			return
		}
	}
	t.Fatalf("fact family %s not found in %#v", family, facts)
}

func requireProofFact(t *testing.T, facts []*factsv1.GenericFact, subject string, status factsv1.EvidenceStatus) {
	t.Helper()
	for _, fact := range facts {
		if fact.GetSubject() != subject {
			continue
		}
		if len(fact.GetEvidence()) == 0 {
			t.Fatalf("proof fact %s has no evidence", subject)
		}
		if got := fact.GetEvidence()[0].GetStatus(); got != status {
			t.Fatalf("proof fact %s status = %s, want %s; fact=%#v", subject, got, status, fact)
		}
		return
	}
	t.Fatalf("proof fact %s not found in %#v", subject, facts)
}

type fakeProvider struct {
	language string
	analyzer string
	result   *GraphResult
	err      error
}

func (f fakeProvider) Language() string     { return f.language }
func (f fakeProvider) AnalyzerName() string { return f.analyzer }

func (f fakeProvider) Extract(context.Context, *factsv1.ParseUnit) (*GraphResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.result != nil {
		return f.result, nil
	}
	return &GraphResult{Graph: &commonv1.CodeGraph{}}, nil
}

type countingProvider struct {
	fakeProvider
	calls int
}

type fakeFileDomainProvider struct {
	facts    []*factsv1.GenericFact
	evidence []*factsv1.Evidence
	warnings []*factsv1.Warning
	err      error
}

func (f fakeFileDomainProvider) DescribeFileDomains(context.Context, *factsv1.TargetContext) ([]*factsv1.GenericFact, []*factsv1.Evidence, []*factsv1.Warning, error) {
	return f.facts, f.evidence, f.warnings, f.err
}

func (f *countingProvider) Extract(ctx context.Context, unit *factsv1.ParseUnit) (*GraphResult, error) {
	f.calls++
	return f.fakeProvider.Extract(ctx, unit)
}

type unitProvider struct {
	language string
	analyzer string
	extract  func(*factsv1.ParseUnit) *GraphResult
}

func (f unitProvider) Language() string     { return f.language }
func (f unitProvider) AnalyzerName() string { return f.analyzer }

func (f unitProvider) Extract(_ context.Context, unit *factsv1.ParseUnit) (*GraphResult, error) {
	return f.extract(unit), nil
}

func openCacheTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), CacheSchema()); err != nil {
		t.Fatalf("apply cache schema: %v", err)
	}
	return db
}

func exerciseCacheSupersedeOnPut(t *testing.T, repo CacheRepository) {
	t.Helper()
	ctx := context.Background()
	first := cacheTestEntry("first-key", "unit-one", "source-one")
	second := cacheTestEntry("second-key", "unit-one", "source-two")
	if err := repo.PutGraph(ctx, first, cacheTestGraph("graph-one")); err != nil {
		t.Fatalf("PutGraph(first): %v", err)
	}
	if err := repo.PutGraph(ctx, second, cacheTestGraph("graph-two")); err != nil {
		t.Fatalf("PutGraph(second): %v", err)
	}
	entries, err := repo.Status(ctx, "/target", "")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Key != second.Key {
		t.Fatalf("remaining key = %q, want %q", entries[0].Key, second.Key)
	}
	if _, _, ok, err := repo.GetGraph(ctx, first.Key); err != nil || ok {
		t.Fatalf("GetGraph(first) ok=%v err=%v, want miss", ok, err)
	}
	got, entry, ok, err := repo.GetGraph(ctx, second.Key)
	if err != nil || !ok {
		t.Fatalf("GetGraph(second) ok=%v err=%v, want hit", ok, err)
	}
	if got.Graph == nil || entry.SourceHash != "source-two" {
		t.Fatalf("new cache entry = graph nil? %v source %q, want graph payload/source-two", got.Graph == nil, entry.SourceHash)
	}
}

func exerciseCacheDistinctLogicalIdentitiesCoexist(t *testing.T, repo CacheRepository) {
	t.Helper()
	ctx := context.Background()
	first := cacheTestEntry("unit-one-key", "unit-one", "source-one")
	second := cacheTestEntry("unit-two-key", "unit-two", "source-two")
	if err := repo.PutGraph(ctx, first, cacheTestGraph("graph-one")); err != nil {
		t.Fatalf("PutGraph(first): %v", err)
	}
	if err := repo.PutGraph(ctx, second, cacheTestGraph("graph-two")); err != nil {
		t.Fatalf("PutGraph(second): %v", err)
	}
	entries, err := repo.Status(ctx, "/target", "")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

func exerciseCacheBudgetEvictsLRUAndRefreshChangesOrder(t *testing.T, newRepo func(maxBytes int64) CacheRepository) {
	t.Helper()
	ctx := context.Background()
	graphA := cacheTestGraph("graph-a")
	graphB := cacheTestGraph("graph-b")
	graphC := cacheTestGraph("graph-c")
	sizeA := graphPayloadBytes(t, graphA)
	sizeB := graphPayloadBytes(t, graphB)
	sizeC := graphPayloadBytes(t, graphC)
	repo := newRepo(sizeA + sizeB + sizeC - 1)
	entryA := cacheTestEntry("key-a", "unit-a", "source-a")
	entryA.CreatedAtUnix = 1
	entryB := cacheTestEntry("key-b", "unit-b", "source-b")
	entryB.CreatedAtUnix = 2
	entryC := cacheTestEntry("key-c", "unit-c", "source-c")
	entryC.CreatedAtUnix = 3
	if err := repo.PutGraph(ctx, entryA, graphA); err != nil {
		t.Fatalf("PutGraph(a): %v", err)
	}
	if err := repo.PutGraph(ctx, entryB, graphB); err != nil {
		t.Fatalf("PutGraph(b): %v", err)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryA.Key); err != nil || !ok {
		t.Fatalf("GetGraph(a) ok=%v err=%v, want refresh hit", ok, err)
	}
	if err := repo.PutGraph(ctx, entryC, graphC); err != nil {
		t.Fatalf("PutGraph(c): %v", err)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryB.Key); err != nil || ok {
		t.Fatalf("GetGraph(b) ok=%v err=%v, want LRU miss", ok, err)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryA.Key); err != nil || !ok {
		t.Fatalf("GetGraph(a after eviction) ok=%v err=%v, want hit", ok, err)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryC.Key); err != nil || !ok {
		t.Fatalf("GetGraph(c after eviction) ok=%v err=%v, want hit", ok, err)
	}
	entries, err := repo.Status(ctx, "/target", "")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if totalCachePayloadBytes(entries) > sizeA+sizeB+sizeC-1 {
		t.Fatalf("payload bytes = %d, want <= %d", totalCachePayloadBytes(entries), sizeA+sizeB+sizeC-1)
	}
}

func exerciseCacheSweepDropsStaleSchemaAndEnforcesBudget(t *testing.T, repo CacheRepository, withBudget func(maxBytes int64) CacheRepository) {
	t.Helper()
	ctx := context.Background()
	graphA := cacheTestGraph("sweep-a")
	graphB := cacheTestGraph("sweep-b")
	graphC := cacheTestGraph("sweep-c")
	sizeB := graphPayloadBytes(t, graphB)
	sizeC := graphPayloadBytes(t, graphC)
	entryA := cacheTestEntry("sweep-a", "sweep-a", "source-a")
	entryA.CreatedAtUnix = 1
	entryA.SchemaVersion = "old-schema"
	entryB := cacheTestEntry("sweep-b", "sweep-b", "source-b")
	entryB.CreatedAtUnix = 2
	entryC := cacheTestEntry("sweep-c", "sweep-c", "source-c")
	entryC.CreatedAtUnix = 3
	if err := repo.PutGraph(ctx, entryA, graphA); err != nil {
		t.Fatalf("PutGraph(stale): %v", err)
	}
	if err := repo.PutGraph(ctx, entryB, graphB); err != nil {
		t.Fatalf("PutGraph(b): %v", err)
	}
	if err := repo.PutGraph(ctx, entryC, graphC); err != nil {
		t.Fatalf("PutGraph(c): %v", err)
	}
	repo = withBudget(sizeB + sizeC - 1)
	sweep, err := repo.Sweep(ctx)
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if sweep.StaleRows != 1 {
		t.Fatalf("stale rows = %d, want 1", sweep.StaleRows)
	}
	if sweep.EvictedRows != 1 {
		t.Fatalf("evicted rows = %d, want 1", sweep.EvictedRows)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryA.Key); err != nil || ok {
		t.Fatalf("GetGraph(stale) ok=%v err=%v, want miss", ok, err)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryB.Key); err != nil || ok {
		t.Fatalf("GetGraph(lru) ok=%v err=%v, want miss", ok, err)
	}
	if _, _, ok, err := repo.GetGraph(ctx, entryC.Key); err != nil || !ok {
		t.Fatalf("GetGraph(protected survivor) ok=%v err=%v, want hit", ok, err)
	}
}

func exerciseCachePayloadRoundTrip(t *testing.T, repo CacheRepository) {
	t.Helper()
	ctx := context.Background()
	graph := cacheTestGraph("round-trip-graph")
	graph.Warnings = []*commonv1.CodeGraphWarning{{Message: "warning"}}
	graphEntry := cacheTestEntry("round-trip-graph-key", "round-trip-graph", "source")
	graphEntry.GraphHash = graph.GraphHash
	if err := repo.PutGraph(ctx, graphEntry, graph); err != nil {
		t.Fatalf("PutGraph: %v", err)
	}
	gotGraph, _, ok, err := repo.GetGraph(ctx, graphEntry.Key)
	if err != nil || !ok {
		t.Fatalf("GetGraph ok=%v err=%v, want hit", ok, err)
	}
	if gotGraph.GraphHash != graph.GraphHash || len(gotGraph.Warnings) != 1 || gotGraph.Warnings[0].GetMessage() != "warning" {
		t.Fatalf("graph round trip = hash %q warnings %#v, want hash %q warning", gotGraph.GraphHash, gotGraph.Warnings, graph.GraphHash)
	}
	report := &factsv1.CodeFactsReport{
		Target: &factsv1.TargetContext{RootPath: "/target"},
		Facts: []*factsv1.GenericFact{{
			Family:  factsv1.FactFamily_FACT_FAMILY_IMPORTS,
			Subject: "fmt",
		}},
	}
	reportEntry := cacheTestEntry("round-trip-report-key", "round-trip-report", "source")
	reportEntry.FamilyKey = "FACT_FAMILY_IMPORTS"
	if err := repo.PutReport(ctx, reportEntry, report); err != nil {
		t.Fatalf("PutReport: %v", err)
	}
	gotReport, _, ok, err := repo.GetReport(ctx, reportEntry.Key)
	if err != nil || !ok {
		t.Fatalf("GetReport ok=%v err=%v, want hit", ok, err)
	}
	if gotReport.GetTarget().GetRootPath() != "/target" || len(gotReport.GetFacts()) != 1 || gotReport.GetFacts()[0].GetSubject() != "fmt" {
		t.Fatalf("report round trip = %#v, want target/fact preserved", gotReport)
	}
}

func cacheTestEntry(key string, identity string, sourceHash string) cacheEntry {
	return cacheEntry{
		Key:           key,
		TargetRoot:    "/target",
		Analyzer:      cacheAnalyzerVersion,
		Provider:      "go-code-graph",
		ProviderVer:   "go-code-graph:phase8",
		SchemaVersion: cacheSchemaVersion,
		GraphHash:     "graph",
		SourceHash:    sourceHash,
		ConfigHash:    "config",
		FamilyKey:     "go",
		Identity:      identity,
	}
}

func cacheTestGraph(hash string) *GraphResult {
	return &GraphResult{
		GraphHash: hash,
		Graph: &commonv1.CodeGraph{
			Nodes: []*commonv1.CodeGraphNode{{Id: hash, Name: hash}},
		},
	}
}

func largeCacheTestGraph(nodes int) *GraphResult {
	graph := &commonv1.CodeGraph{Nodes: make([]*commonv1.CodeGraphNode, 0, nodes)}
	for i := 0; i < nodes; i++ {
		id := fmt.Sprintf("node-%04d", i)
		graph.Nodes = append(graph.Nodes, &commonv1.CodeGraphNode{
			Id:   id,
			Name: "repeated-symbol-name",
			Path: "internal/repeated/file.go",
			Attributes: map[string]string{
				"kind":        "GO_NODE_KIND_IDENTIFIER",
				"description": "this repeated payload is intentionally compressible",
			},
		})
	}
	return &GraphResult{GraphHash: "large-graph", Graph: graph}
}

func graphPayloadBytes(t *testing.T, graph *GraphResult) int64 {
	t.Helper()
	payload, _, err := marshalGraphResult(graph)
	if err != nil {
		t.Fatalf("marshal graph payload: %v", err)
	}
	return int64(len(payload))
}

func totalCachePayloadBytes(entries []cacheEntry) int64 {
	var total int64
	for _, entry := range entries {
		total += entry.PayloadBytes
	}
	return total
}

func BenchmarkSourceFingerprint(b *testing.B) {
	root := b.TempDir()
	paths := make([]string, 0, 250)
	for i := 0; i < 250; i++ {
		path := filepath.Join(root, fmt.Sprintf("file_%03d.go", i))
		if err := os.WriteFile(path, []byte("package bench\n\nfunc F() string { return \"stable\" }\n"), 0o644); err != nil {
			b.Fatalf("write fixture: %v", err)
		}
		paths = append(paths, path)
	}
	b.Run("cold", func(b *testing.B) {
		sourceFileHashMemo = newFileHashMemo(8192)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = fileStatSignature(root, paths)
		}
	})
	b.Run("warm", func(b *testing.B) {
		sourceFileHashMemo = newFileHashMemo(8192)
		_ = fileStatSignature(root, paths)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = fileStatSignature(root, paths)
		}
	})
}
