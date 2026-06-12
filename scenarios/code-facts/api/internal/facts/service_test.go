package facts

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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
	requireSurface(t, report.GetSurfaces(), "ui", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireParseUnit(t, report.GetParseUnits(), "go", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
	requireParseUnit(t, report.GetParseUnits(), "typescript", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
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
					"kind":        "GO_NODE_KIND_IMPORT_SPEC",
					"import_path": `"fmt"`,
					"start_line":  "3",
				},
			},
			{
				Id:   "go_call:main.go:fmt.Println",
				Name: "fmt.Println",
				Path: "main.go",
				Attributes: map[string]string{
					"kind":   "GO_NODE_KIND_CALL",
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
					"kind":        "GO_NODE_KIND_CALL",
					"callee":      "r.HandleFunc(...).Methods",
					"route_path":  "/api/v1/notes/{id}/attachments",
					"http_method": "POST",
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
	requireProofFact(t, report.GetFacts(), "notes_attach", factsv1.EvidenceStatus_EVIDENCE_STATUS_PROVEN)
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
				Id:   "response",
				Name: "httpx.WriteProto",
				Path: "api/handlers/notes/attachments_handler.go",
				Attributes: map[string]string{
					"kind":           "GO_NODE_KIND_CALL",
					"callee":         "httpx.WriteProto",
					"argument_types": "net/http.ResponseWriter, int, *github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/notes.Note",
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

func writeScenarioFixture(t *testing.T, name string) (repo string, scenarioRoot string) {
	t.Helper()
	repo = t.TempDir()
	scenarioRoot = filepath.Join(repo, "scenarios", name)
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "service.json"), `{
  "cli": {"enabled": true, "adapter": {"kind": "go_module", "module_dir": "cli"}}
}`)
	writeFile(t, filepath.Join(scenarioRoot, ".vrooli", "endpoints.json"), `{"endpoints":[]}`)
	writeFile(t, filepath.Join(scenarioRoot, "api", "go.mod"), "module demo/api\n\ngo 1.25\n")
	writeFile(t, filepath.Join(scenarioRoot, "cli", "go.mod"), "module demo/cli\n\ngo 1.25\n")
	writeFile(t, filepath.Join(scenarioRoot, "cli", "manifest.json"), `{"groups":[]}`)
	writeFile(t, filepath.Join(scenarioRoot, "ui", "package.json"), `{"scripts":{"build":"vite"}}`)
	writeFile(t, filepath.Join(scenarioRoot, "ui", "tsconfig.json"), `{"compilerOptions":{"jsx":"react-jsx"}}`)
	return repo, scenarioRoot
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
