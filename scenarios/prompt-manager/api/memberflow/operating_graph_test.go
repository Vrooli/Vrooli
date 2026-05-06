package memberflow

import (
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/teamcontract"
)

func TestParseOperatingMermaidTypedNodesAndEdges(t *testing.T) {
	graph, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		`  R["member:researcher<br/>Researcher"]`,
		`  RI["topic:research-inbox/*"]`,
		"  RI --> R",
	}, 1)
	if err != nil {
		t.Fatalf("ParseOperatingMermaid: %v", err)
	}
	if graph.Direction != "LR" {
		t.Fatalf("direction = %q", graph.Direction)
	}
	if len(graph.Nodes) != 2 || len(graph.Edges) != 1 {
		t.Fatalf("nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
	}
	var found bool
	for _, n := range graph.Nodes {
		if n.ID == "R" {
			found = true
			if n.Kind != "member" || n.Value != "researcher" || n.Display != "Researcher" {
				t.Fatalf("bad R node: %+v", n)
			}
		}
	}
	if !found {
		t.Fatalf("R node missing: %+v", graph.Nodes)
	}
}

func TestValidateOperatingGraphsDetectsUnbackedEdge(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  M["member:member-a"]`,
			`  T["topic:unknown/*"]`,
			"  M --> T",
		}),
	}
	runtime := OperatingGraphRuntime{
		RepoRoot: t.TempDir(),
		Members: []MemberTopics{{
			Ref:    MemberRef{Team: "team-a", Member: "member-a"},
			Topics: Topics{},
		}},
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members:         map[string]teamcontract.MemberContract{"member-a": {}},
				DecisionContext: map[string]teamcontract.DecisionContext{},
			}},
		},
	}
	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	if result.Errors == 0 {
		t.Fatalf("expected validation errors, got %+v", result)
	}
	assertOperatingFinding(t, result, "graph_topic_unresolved")
	assertOperatingFinding(t, result, "graph_edge_unbacked")
}

func TestExtractOperatingGraphBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "OPERATING_MODEL.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
`+"```"+`
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/x.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 1 || blocks[0].Metadata.ID != "g" || blocks[0].Graph.Nodes[0].Kind != "member" {
		t.Fatalf("unexpected blocks: %+v", blocks)
	}
}

func mustParseGraph(t *testing.T, lines []string) OperatingGraph {
	t.Helper()
	graph, err := ParseOperatingMermaid("g", lines, 1)
	if err != nil {
		t.Fatalf("parse graph: %v", err)
	}
	return graph
}

func assertOperatingFinding(t *testing.T, result OperatingGraphValidationResult, rule string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Rule == rule {
			return
		}
	}
	t.Fatalf("finding %q missing from %+v", rule, result.Findings)
}
