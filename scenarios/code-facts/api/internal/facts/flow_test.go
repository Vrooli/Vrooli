package facts

import (
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestDeriveFeatureFlowsBuildsBoundedResolvedPath(t *testing.T) {
	graph := &factsv1.LanguageGraph{GraphHash: "fixture-graph", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
		{Id: "func:SearchComposition", Name: "SearchComposition", Path: "composition_api.go", Attributes: map[string]string{"kind": "go_method"}},
		{Id: "func:Composition", Name: "Composition", Path: "composition_api.go", Attributes: map[string]string{"kind": "go_method"}},
		{Id: "func:DescribeControlPlaneCLIComposition", Name: "DescribeControlPlaneCLIComposition", Path: "controlplane_cli.go", Attributes: map[string]string{"kind": "go_func"}},
		{Id: "call:1", Path: "composition_api.go", Attributes: map[string]string{"kind": "go_call", "enclosing_symbol": "SearchComposition", "callee": "s.Composition", "callee_symbol": "func:Composition", "start_line": "77"}},
		{Id: "call:2", Path: "composition_api.go", Attributes: map[string]string{"kind": "go_call", "enclosing_symbol": "Composition", "callee": "DescribeControlPlaneCLIComposition", "callee_symbol": "func:DescribeControlPlaneCLIComposition", "start_line": "35"}},
	}}}
	response := DeriveFeatureFlows([]*factsv1.LanguageGraph{graph}, "SearchComposition", "code-facts", 5)
	if len(response.Results) != 1 {
		t.Fatalf("results=%#v degradations=%#v", response.Results, response.Degradations)
	}
	hit := response.Results[0]
	if hit.ID != "code-facts:flow:SearchComposition:composition_api.go" || hit.GraphHash != "fixture-graph" || hit.Attestation != "DERIVED/PARTIAL" || len(hit.Steps) != 2 {
		t.Fatalf("unexpected flow hit: %#v", hit)
	}
	if hit.Steps[0].To != "Composition" || hit.Steps[1].To != "DescribeControlPlaneCLIComposition" || hit.Steps[1].Resolution != "source_symbol_resolved" || hit.Steps[0].GraphHash != "fixture-graph" || hit.Steps[1].ToGraphHash != "fixture-graph" {
		t.Fatalf("unexpected source path: %#v", hit.Steps)
	}
}

func TestDeriveFeatureFlowsReportsNoMatchingStartAndMissingGraph(t *testing.T) {
	graph := &factsv1.LanguageGraph{GraphHash: "fixture-graph", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
		{Id: "func:SearchComposition", Name: "SearchComposition", Path: "composition_api.go", Attributes: map[string]string{"kind": "go_method"}},
		{Id: "call:1", Path: "composition_api.go", Attributes: map[string]string{"kind": "go_call", "enclosing_symbol": "SearchComposition", "callee": "fmt.Println", "callee_symbol": "external:Println", "start_line": "1"}},
	}}}
	noMatch := DeriveFeatureFlows([]*factsv1.LanguageGraph{graph}, "underwater penguin", "code-facts", 5)
	if len(noMatch.Results) != 0 || len(noMatch.Degradations) != 1 || noMatch.Degradations[0].Code != "no_matching_flow_entry" {
		t.Fatalf("unexpected no-match response: %#v", noMatch)
	}
	missing := DeriveFeatureFlows(nil, "SearchComposition", "code-facts", 5)
	if len(missing.Results) != 0 || len(missing.Degradations) != 1 || missing.Degradations[0].Code != "call_graph_unavailable" {
		t.Fatalf("unexpected missing-graph response: %#v", missing)
	}
}

func TestDeriveFeatureFlowsDisambiguatesSameNamedBasenameResults(t *testing.T) {
	graph := &factsv1.LanguageGraph{GraphHash: "fixture-graph", Graph: &commonv1.CodeGraph{Nodes: []*commonv1.CodeGraphNode{
		{Id: "func:api:Handle", Name: "Handle", Path: "api/main.go", Attributes: map[string]string{"kind": "go_func"}},
		{Id: "func:api:Target", Name: "apiTarget", Path: "api/main.go", Attributes: map[string]string{"kind": "go_func"}},
		{Id: "func:cli:Handle", Name: "Handle", Path: "cli/main.go", Attributes: map[string]string{"kind": "go_func"}},
		{Id: "func:cli:Target", Name: "cliTarget", Path: "cli/main.go", Attributes: map[string]string{"kind": "go_func"}},
		{Id: "call:api", Path: "api/main.go", Attributes: map[string]string{"kind": "go_call", "enclosing_symbol": "Handle", "callee": "apiTarget", "callee_symbol": "func:api:Target", "start_line": "10"}},
		{Id: "call:cli", Path: "cli/main.go", Attributes: map[string]string{"kind": "go_call", "enclosing_symbol": "Handle", "callee": "cliTarget", "callee_symbol": "func:cli:Target", "start_line": "20"}},
	}}}
	response := DeriveFeatureFlows([]*factsv1.LanguageGraph{graph}, "Handle", "sample", 5)
	if len(response.Results) != 2 {
		t.Fatalf("results=%#v degradations=%#v, want both source flows", response.Results, response.Degradations)
	}
	ids := map[string]bool{}
	paths := map[string]bool{}
	for _, hit := range response.Results {
		if ids[hit.ID] {
			t.Fatalf("duplicate flow result ID %q: %#v", hit.ID, response.Results)
		}
		ids[hit.ID] = true
		paths[hit.Path] = true
	}
	if !ids["code-facts:flow:Handle:main.go:api/main.go:func:api:Handle"] || !ids["code-facts:flow:Handle:main.go:cli/main.go:func:cli:Handle"] || !paths["api/main.go"] || !paths["cli/main.go"] {
		t.Fatalf("collision IDs or source paths missing: %#v", response.Results)
	}
}
