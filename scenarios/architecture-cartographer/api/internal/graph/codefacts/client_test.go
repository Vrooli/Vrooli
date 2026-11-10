package codefacts

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"architecture-cartographer/internal/graph"
	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	"github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts/facts_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type describeFunc func(context.Context, *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error)
type fakeService struct {
	facts_v1connect.UnimplementedCodeFactsServiceHandler
	describe describeFunc
}

func (f fakeService) DescribeCodeFacts(ctx context.Context, req *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
	return f.describe(ctx, req)
}
func server(t *testing.T, fn describeFunc) string {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := facts_v1connect.NewCodeFactsServiceHandler(fakeService{describe: fn})
	mux.Handle(path, handler)
	s := httptest.NewServer(mux)
	t.Cleanup(s.Close)
	return s.URL
}

func graphReport(language, directory string) *factsv1.CodeFactsReport {
	return &factsv1.CodeFactsReport{
		Target: &factsv1.TargetContext{RootPath: "/repo/scenarios/demo"},
		Cache:  &factsv1.CacheMetadata{SourceHash: "source", ConfigHash: "config"},
		Graphs: []*factsv1.LanguageGraph{{
			ParseUnit:         &factsv1.ParseUnit{RootPath: "/repo/scenarios/demo/" + directory, Language: language},
			ExtractionProfile: "structural", ExtractionMs: 7,
			OmittedInformation: []*factsv1.GraphInformationOmission{{Capability: "resolved_usage_facts", Reason: "structural profile"}},
			Graph: &commonv1.CodeGraph{
				Nodes: []*commonv1.CodeGraphNode{
					{Id: "file:main.go", Kind: commonv1.NodeKind_NODE_KIND_FILE, Path: "main.go", Attributes: map[string]string{"package_id": "package:demo", "lines": "24", "is_test": "true"}},
					{Id: "package:demo", Kind: commonv1.NodeKind_NODE_KIND_PACKAGE, Name: "demo", Attributes: map[string]string{"import_path": "demo"}},
					{Id: "function:Run", Kind: commonv1.NodeKind_NODE_KIND_PACKAGE, Name: "Run", Attributes: map[string]string{"kind": "function", "file_id": "file:main.go", "package_id": "package:demo", "exported": "true"}},
					{Id: "module:demo", Kind: commonv1.NodeKind_NODE_KIND_MODULE, Name: "module-demo"},
				},
				Edges: []*commonv1.CodeGraphEdge{{Kind: commonv1.EdgeKind_EDGE_KIND_IMPORT, FromNodeId: "package:demo", ToNodeId: "package:fmt", Attributes: map[string]string{"test_only": "true", "symbol_ids": "function:Run", "symbol_kinds": "function"}}},
			},
		}},
	}
}

func TestClient_SharedGraphsPreserveScopeAndMetadata(t *testing.T) {
	for _, language := range []graph.Language{graph.LanguageGo, graph.LanguageTypeScript} {
		t.Run(string(language), func(t *testing.T) {
			name := string(language)
			if language == graph.LanguageTypeScript {
				name = "typescript"
			}
			url := server(t, func(_ context.Context, req *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
				if !req.Msg.GetIncludeGraphs() || !req.Msg.GetUseCache() || req.Msg.GetTarget().GetScenario() != "demo" || req.Msg.GetTarget().GetRepoRoot() != "/repo" || req.Msg.GetTarget().GetLanguageFilter()[0] != name {
					t.Errorf("wrong shared substrate request: %v", req.Msg)
				}
				return connect.NewResponse(graphReport(name, "api")), nil
			})
			client := New(discovery.NewStaticResolver(url), "/repo", language, nil)
			raw, err := client.Extract(context.Background(), "demo")
			if err != nil {
				t.Fatal(err)
			}
			if client.Name() != name || client.SupportedLanguages()[0] != language {
				t.Fatal("wrong adapter language")
			}
			if len(raw.Files) != 1 || raw.Files[0].Path != "api/main.go" || raw.Files[0].Lines != 24 || !raw.Files[0].IsTest {
				t.Fatalf("file evidence lost: %+v", raw.Files)
			}
			if len(raw.Packages) != 2 || raw.Packages[0].RepoPath != "api" || len(raw.Symbols) != 1 || !raw.Symbols[0].Exported {
				t.Fatalf("package/symbol discrimination lost: %+v", raw)
			}
			if raw.Symbols[0].FileID != raw.Files[0].ID || raw.Symbols[0].PackageID != raw.Packages[0].ID {
				t.Fatal("identity references detached")
			}
			if len(raw.Imports) != 1 || !raw.Imports[0].TestOnly || raw.Imports[0].SymbolIDs[0] != raw.Symbols[0].ID {
				t.Fatalf("import evidence lost: %+v", raw.Imports)
			}
			if raw.ExtractionMS != 7 || len(raw.OmittedInformation) != 1 || raw.ExtractionProfiles[0] != "structural" || raw.SourceFingerprint != "source:config" {
				t.Fatalf("extraction contract lost: %+v", raw)
			}
		})
	}
}

func TestProject_DistinguishesModulesWithIdenticalLocalIDs(t *testing.T) {
	report := graphReport("go", "api")
	report.Graphs = append(report.Graphs, graphReport("go", "cli").Graphs...)
	raw, err := project(report, graph.LanguageGo)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw.Files) != 2 || raw.Files[0].ID == raw.Files[1].ID || raw.Files[0].Path == raw.Files[1].Path || raw.Packages[0].ID == raw.Packages[2].ID {
		t.Fatalf("module identities collapsed: %+v", raw)
	}
}

func TestProject_MissingLanguageAndPartialProvider(t *testing.T) {
	raw, err := project(&factsv1.CodeFactsReport{}, graph.LanguageGo)
	if err != nil || len(raw.Files) != 0 || raw.SourceFingerprint != "" {
		t.Fatalf("empty language result: %+v %v", raw, err)
	}
	report := graphReport("go", "api")
	report.Warnings = []*factsv1.Warning{{Code: "go-code-graph.unavailable", Status: factsv1.EvidenceStatus_EVIDENCE_STATUS_UNKNOWN}}
	raw, err = project(report, graph.LanguageGo)
	if err != nil || len(raw.Files) != 1 || len(raw.SkippedAdapters) != 1 {
		t.Fatalf("partial result discarded: %+v %v", raw, err)
	}
	report.Graphs[0].ParseUnit.RootPath = "/outside"
	if _, err := project(report, graph.LanguageGo); err == nil {
		t.Fatal("accepted parse unit outside target")
	}
}

func TestClient_ErrorMappingAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		code connect.Code
		want string
	}{{connect.CodeUnavailable, "scenario_unreachable"}, {connect.CodeUnimplemented, "unimplemented"}, {connect.CodeDeadlineExceeded, "scenario_timeout"}, {connect.CodeInvalidArgument, "invalid_argument"}, {connect.CodeInternal, "internal"}, {connect.CodeCanceled, "canceled"}} {
		t.Run(tc.code.String(), func(t *testing.T) {
			url := server(t, func(context.Context, *connect.Request[factsv1.DescribeCodeFactsRequest]) (*connect.Response[factsv1.CodeFactsReport], error) {
				return nil, connect.NewError(tc.code, errors.New("provider error"))
			})
			_, err := New(discovery.NewStaticResolver(url), "/repo", graph.LanguageGo, nil).Extract(context.Background(), "demo")
			var ie graph.IntegrationError
			if !errors.As(err, &ie) || ie.Kind != tc.want || ie.Scenario != ScenarioName {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}
}
