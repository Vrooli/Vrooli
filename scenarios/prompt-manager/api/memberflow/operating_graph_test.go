package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/teamcontract"
)

func TestParseOperatingMermaidTypedNodesAndEdges(t *testing.T) {
	graph, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		"  %% @node R member:researcher",
		"  R[Researcher]",
		"  %% @node RI topic:research-inbox/*",
		"  RI[research-inbox/*]",
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

func TestParseOperatingMermaidInlineTypedLabel(t *testing.T) {
	graph, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		`  R["member:researcher<br/>Researcher"]`,
		`  RI["topic:research-inbox/*"]`,
		"  RI --> R",
	}, 1)
	if err != nil {
		t.Fatalf("ParseOperatingMermaid: %v", err)
	}
	node := operatingNodeByID(t, graph, "R")
	if node.Kind != "member" || node.Value != "researcher" || node.Display != "Researcher" {
		t.Fatalf("bad inline typed node: %+v", node)
	}
}

func TestParseOperatingMermaidAnnotationRejectsConflictingInlineToken(t *testing.T) {
	_, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		"  %% @node R member:researcher",
		`  R["member:brand-manager<br/>Researcher"]`,
	}, 1)
	if err == nil {
		t.Fatalf("expected conflicting annotation and inline token error")
	}
}

func TestParseOperatingMermaidAnnotationRejectsDuplicate(t *testing.T) {
	_, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		"  %% @node R member:researcher",
		"  %% @node R member:brand-manager",
		"  R[Researcher]",
	}, 1)
	if err == nil {
		t.Fatalf("expected duplicate annotation error")
	}
}

func TestParseOperatingMermaidAnnotationRequiresDeclaredNode(t *testing.T) {
	_, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		"  %% @node R member:researcher",
		"  R --> T",
		"  T[topic]",
	}, 1)
	if err == nil {
		t.Fatalf("expected annotation without declared node error")
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

func TestDiffOperatingGraphsDetectsGraphReadMissingInRuntime(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  T["topic:marketing/notebook/*"]`,
		"  T --> M",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref:    MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{},
	}})

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	diff := assertOperatingDiff(t, diffs, "graph_relationship_missing_in_runtime", operatingRelTopicRead)
	if diff.Member != "researcher" || diff.Topic != "marketing/notebook/*" || diff.RuntimePath == "" || len(diff.Suggestions) == 0 {
		t.Fatalf("unexpected diff: %+v", diff)
	}
}

func TestDiffOperatingGraphsDetectsRuntimeRequiredReadMissingInGraph(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{RequiredRead: []RequiredReadEntry{{
			Prefix: "marketing/notebook/*",
		}}},
	}})

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	diff := assertOperatingDiff(t, diffs, "runtime_relationship_missing_in_graph", operatingRelTopicRead)
	if diff.Member != "researcher" || diff.Topic != "marketing/notebook/*" || diff.RuntimePath == "" || len(diff.Suggestions) == 0 {
		t.Fatalf("unexpected diff: %+v", diff)
	}
}

func TestDiffOperatingGraphsGraphReadMatchesRuntimeReadSubtypes(t *testing.T) {
	for name, topics := range map[string]Topics{
		"intake": {Intake: []IntakeEntry{{Prefix: "marketing/notebook/*"}}},
		"required_read": {RequiredRead: []RequiredReadEntry{{
			Prefix: "marketing/notebook/*",
		}}},
		"evidence_consumed": {EvidenceConsumed: []EvidenceConsumedEntry{{
			Prefix: "marketing/notebook/*",
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			block := operatingDiffBlock(t, []string{
				"flowchart LR",
				`  M["member:researcher"]`,
				`  T["topic:marketing/notebook/*"]`,
				"  T --> M",
			})
			runtime := operatingDiffRuntime(t, []MemberTopics{{
				Ref:    MemberRef{Team: "team-a", Member: "researcher"},
				Topics: topics,
			}})

			if diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g"); len(diffs) != 0 {
				t.Fatalf("expected clean diff, got %+v", diffs)
			}
		})
	}
}

func TestDiffOperatingGraphsMatchesRuntimeOutputCapabilityGapCrossTeamAndPOR(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  T["topic:campaign-draft/*"]`,
		`  CAP["decision:capability-gap"]`,
		`  MON["team:monetization"]`,
		`  P["por:docs/marketing/OPERATING_MODEL.md"]`,
		"  M --> T",
		"  M --> CAP",
		"  T --> MON",
		"  M --> P",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{
			Output: []OutputEntry{
				{Prefix: "campaign-draft/*", DestinationKind: DestinationKnowledge, DestinationTeam: operatingStringPtr("monetization")},
				{Prefix: "por-update/*", DestinationKind: DestinationPORFile, DestinationPath: operatingStringPtr("docs/marketing/OPERATING_MODEL.md")},
			},
			RaisesCapabilityGaps: true,
		},
	}})

	if diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g"); len(diffs) != 0 {
		t.Fatalf("expected clean diff, got %+v", diffs)
	}
}

func TestDiffOperatingGraphsSkipsFutureGraphRelationships(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  T["topic[future]:publish-performance/*"]`,
		"  T --> M",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref:    MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{},
	}})

	if diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g"); len(diffs) != 0 {
		t.Fatalf("expected clean diff, got %+v", diffs)
	}
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

func TestMarketingOperatingModelUsesReadableAnnotatedLabels(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "marketing", "OPERATING_MODEL.md")
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/marketing/OPERATING_MODEL.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	graph := blocks[0].Graph
	if len(graph.Nodes) == 0 || len(graph.Edges) == 0 {
		t.Fatalf("expected populated marketing graph, nodes=%d edges=%d", len(graph.Nodes), len(graph.Edges))
	}
	mon := operatingNodeByID(t, graph, "MON")
	if mon.Kind != "team" || mon.Value != "monetization" || mon.Display != "Monetization team" {
		t.Fatalf("bad MON node: %+v", mon)
	}
	if strings.Contains(mon.Display, "team:") || strings.Contains(mon.RawLabel, "team:monetization") {
		t.Fatalf("MON visual label should not contain machine token: %+v", mon)
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

func operatingNodeByID(t *testing.T, graph OperatingGraph, id string) OperatingGraphNode {
	t.Helper()
	for _, node := range graph.Nodes {
		if node.ID == id {
			return node
		}
	}
	t.Fatalf("node %q missing from %+v", id, graph.Nodes)
	return OperatingGraphNode{}
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

func operatingDiffBlock(t *testing.T, lines []string) OperatingGraphBlock {
	t.Helper()
	return OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph:    mustParseGraph(t, lines),
		Source:   OperatingGraphSource{Path: "docs/test/OPERATING_MODEL.md", Line: 1, FenceLine: 2},
	}
}

func operatingDiffRuntime(t *testing.T, members []MemberTopics) OperatingGraphRuntime {
	t.Helper()
	repoRoot := t.TempDir()
	storeDir := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store")
	return OperatingGraphRuntime{
		RepoRoot: repoRoot,
		StoreDir: storeDir,
		Members:  members,
	}
}

func assertOperatingDiff(t *testing.T, diffs []OperatingGraphContractDiff, kind, relationship string) OperatingGraphContractDiff {
	t.Helper()
	for _, diff := range diffs {
		if diff.Kind == kind && diff.Relationship == relationship {
			return diff
		}
	}
	t.Fatalf("diff kind=%q relationship=%q missing from %+v", kind, relationship, diffs)
	return OperatingGraphContractDiff{}
}

func operatingStringPtr(value string) *string {
	return &value
}
