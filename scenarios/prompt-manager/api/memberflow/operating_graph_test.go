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

func TestValidateOperatingGraphsModeSemantics(t *testing.T) {
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{Intake: []IntakeEntry{{
			Prefix: "research-inbox/*",
		}}},
	}})

	for _, tc := range []struct {
		name      string
		mode      string
		wantRules []string
	}{
		{
			name: "explanatory skips all contract checks",
			mode: "explanatory",
		},
		{
			name: "checkable validates present invalid edges but skips completeness",
			mode: "checkable",
			wantRules: []string{
				"graph_topic_unresolved",
				"graph_edge_unbacked",
			},
		},
		{
			name: "contract validates present invalid edges and completeness",
			mode: "contract",
			wantRules: []string{
				"graph_topic_unresolved",
				"graph_edge_unbacked",
				"graph_declared_intake_missing",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			block := OperatingGraphBlock{
				Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: tc.mode},
				Graph: mustParseGraph(t, []string{
					"flowchart LR",
					`  M["member:researcher"]`,
					`  T["topic:unknown/*"]`,
					"  M --> T",
				}),
			}

			result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
			if len(tc.wantRules) == 0 {
				if len(result.Findings) != 0 || result.Errors != 0 || result.Warnings != 0 {
					t.Fatalf("expected clean result, got %+v", result)
				}
				return
			}
			for _, rule := range tc.wantRules {
				assertOperatingFinding(t, result, rule)
			}
		})
	}
}

func TestExtractOperatingGraphBlocksRejectsMalformedMarkedGraphs(t *testing.T) {
	for _, mode := range []string{"explanatory", "checkable", "contract"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "OPERATING_MODEL.md")
			if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: `+mode+`
-->
`+"```mermaid"+`
not-a-flowchart
`+"```"+`
`), 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}
			_, err := ExtractOperatingGraphBlocks(path, "docs/x.md")
			if err == nil {
				t.Fatalf("expected malformed marked %s graph to fail", mode)
			}
		})
	}
}

func TestExtractOperatingGraphBlocksIgnoresUnmarkedMermaid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("```mermaid\nnot-a-flowchart\n```\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/x.md")
	if err != nil {
		t.Fatalf("unmarked mermaid should be ignored, got %v", err)
	}
	if len(blocks) != 0 {
		t.Fatalf("blocks=%d, want 0", len(blocks))
	}
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

func TestDefaultOperatingGraphRulesRegistersBaselineContractRules(t *testing.T) {
	want := []string{
		"graph_untyped_node",
		"graph_unknown_node_kind",
		"graph_unknown_member",
		"graph_unknown_decision",
		"graph_unknown_team",
		"graph_unknown_por",
		"graph_topic_unresolved",
		"graph_future_topic_live_edge",
		"graph_edge_unbacked",
		"graph_declared_intake_missing",
		"graph_declared_required_read_missing",
		"graph_declared_evidence_missing",
		"graph_declared_output_missing",
		"graph_declared_decision_owned_missing",
		"graph_declared_decision_consumed_missing",
		"graph_declared_capability_gap_missing",
		"graph_declared_external_producer_missing",
		"graph_declared_cross_team_output_missing",
	}
	rules := DefaultOperatingGraphRules()
	if len(rules) != len(want) {
		t.Fatalf("rules=%d, want %d", len(rules), len(want))
	}
	for i, rule := range rules {
		if rule.ID() != want[i] {
			t.Fatalf("rule[%d]=%q, want %q", i, rule.ID(), want[i])
		}
		if rule.ID() == "" || rule.Group() == "" || rule.DefaultSeverity() == "" {
			t.Fatalf("rule[%d] has incomplete metadata: id=%q group=%q severity=%q", i, rule.ID(), rule.Group(), rule.DefaultSeverity())
		}
	}
}

func TestValidateOperatingGraphsDecisionNodesAreTeamScoped(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "checkable"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  D["decision:shared-decision"]`,
		}),
	}
	runtime := OperatingGraphRuntime{
		RepoRoot: t.TempDir(),
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members:         map[string]teamcontract.MemberContract{},
				DecisionContext: map[string]teamcontract.DecisionContext{},
			}},
			"team-b": {TeamID: "team-b", Contract: &teamcontract.OperatingContract{
				Members: map[string]teamcontract.MemberContract{},
				DecisionContext: map[string]teamcontract.DecisionContext{
					"shared-decision": {},
				},
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingDetail(t, result, "graph_unknown_decision", "team-scoped graph")
}

func TestOperatingRelationshipSetDedupesAndQueries(t *testing.T) {
	set := NewOperatingRelationshipSet([]OperatingRelationship{
		{Kind: operatingRelTopicRead, Team: "team-a", Member: "researcher", Topic: "hook-record/*"},
		{Kind: operatingRelTopicRead, Team: "team-a", Member: "researcher", Topic: "hook-record/*"},
		{Kind: operatingRelTopicOutput, Team: "team-a", Member: "publisher", Topic: "publish-log/*"},
	})
	if got := len(set.All()); got != 2 {
		t.Fatalf("deduped relationships=%d, want 2", got)
	}
	if got := len(set.ByMember("researcher")); got != 1 {
		t.Fatalf("researcher relationships=%d, want 1", got)
	}
	if got := len(set.ByKind(operatingRelTopicOutput)); got != 1 {
		t.Fatalf("topic outputs=%d, want 1", got)
	}
}

func TestOperatingGraphContractIndexQueriesNodesAndRelationships(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  T["topic:research-inbox/*"]`,
		"  T --> M",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{Intake: []IntakeEntry{{
			Prefix: "research-inbox/*",
		}}},
	}})

	ctx := NewOperatingGraphContractContext(block, runtime)
	if _, ok := ctx.Index.Node("member", "researcher"); !ok {
		t.Fatalf("member node missing from index")
	}
	read := OperatingRelationship{Kind: operatingRelTopicRead, Team: "team-a", Member: "researcher", Topic: "research-inbox/*"}
	if !ctx.Index.RuntimeHasRelationship(read, ctx.Matcher) {
		t.Fatalf("runtime relationship missing from index")
	}
	if !ctx.Index.GraphHasRelationship(read) {
		t.Fatalf("graph relationship missing from index")
	}
}

func TestValidateOperatingGraphsCompletenessUsesIndexedGraphRelationships(t *testing.T) {
	porPath := "docs/marketing/OPERATING_MODEL.md"
	targetTeam := "monetization"
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  M["member:researcher"]`,
			`  EXT["external:operator"]`,
			`  IN["topic:research-inbox/*"]`,
			`  READ["topic:marketing/notebook/*"]`,
			`  EV["topic:challenge-report/*"]`,
			`  OUT["topic:hook-record/*"]`,
			`  MON["team:monetization"]`,
			`  POR["por:docs/marketing/OPERATING_MODEL.md"]`,
			`  OWN["decision:audience-update"]`,
			`  CONSUME["decision:capability-gap"]`,
			"  IN --> M",
			"  READ --> M",
			"  EV --> M",
			"  M --> OUT",
			"  OUT --> MON",
			"  M --> POR",
			"  M --> OWN",
			"  CONSUME --> M",
			"  M --> CONSUME",
			"  EXT --> M",
		}),
	}
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*"}},
			RequiredRead: []RequiredReadEntry{{
				Prefix: "marketing/notebook/*",
			}},
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "challenge-report/*",
				ForDecisions: []string{"capability-gap"},
			}},
			Output: []OutputEntry{
				{Prefix: "hook-record/*", DestinationKind: DestinationKnowledge, DestinationTeam: &targetTeam},
				{Prefix: "por-update/*", DestinationKind: DestinationPORFile, DestinationPath: &porPath},
			},
			DecisionsOwned:       []string{"audience-update"},
			DecisionsConsumed:    []string{"capability-gap"},
			RaisesCapabilityGaps: true,
			ExternalProducers:    []string{"operator"},
		},
	}})
	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	for _, rule := range []string{
		"graph_declared_intake_missing",
		"graph_declared_required_read_missing",
		"graph_declared_evidence_missing",
		"graph_declared_output_missing",
		"graph_declared_decision_owned_missing",
		"graph_declared_decision_consumed_missing",
		"graph_declared_capability_gap_missing",
		"graph_declared_external_producer_missing",
		"graph_declared_cross_team_output_missing",
	} {
		for _, finding := range result.Findings {
			if finding.Rule == rule {
				t.Fatalf("unexpected completeness finding %q: %+v", rule, result.Findings)
			}
		}
	}
}

func TestValidateOperatingGraphsCompletenessCoversRuntimeRelationshipsComparedByDiff(t *testing.T) {
	targetTeam := "monetization"
	porPath := "docs/marketing/OPERATING_MODEL.md"
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*"}},
			RequiredRead: []RequiredReadEntry{{
				Prefix: "marketing/notebook/*",
			}},
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "challenge-report/*",
				ForDecisions: []string{"capability-gap"},
			}},
			Output: []OutputEntry{
				{Prefix: "hook-record/*", DestinationKind: DestinationKnowledge, DestinationTeam: &targetTeam},
				{Prefix: "por-update/*", DestinationKind: DestinationPORFile, DestinationPath: &porPath},
			},
			DecisionsOwned:       []string{"audience-update"},
			DecisionsConsumed:    []string{"capability-gap"},
			RaisesCapabilityGaps: true,
			ExternalProducers:    []string{"operator"},
		},
	}})
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  MON["team:monetization"]`,
		`  EXT["external:operator"]`,
	})

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	for _, rule := range []string{
		"graph_declared_intake_missing",
		"graph_declared_required_read_missing",
		"graph_declared_evidence_missing",
		"graph_declared_output_missing",
		"graph_declared_decision_owned_missing",
		"graph_declared_decision_consumed_missing",
		"graph_declared_capability_gap_missing",
		"graph_declared_external_producer_missing",
		"graph_declared_cross_team_output_missing",
	} {
		assertOperatingFinding(t, result, rule)
	}

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	for _, relationship := range []OperatingRelationshipKind{
		operatingRelTopicRead,
		operatingRelTopicOutput,
		operatingRelPOROutput,
		operatingRelDecisionOwned,
		operatingRelDecisionConsumed,
		operatingRelCapabilityGapRaised,
		operatingRelExternalProducer,
		operatingRelCrossTeamOutput,
	} {
		assertOperatingDiff(t, diffs, "runtime_relationship_missing_in_graph", relationship)
	}
}

func TestBuildRuntimeOperatingRelationshipsExtractsTopicsContractFields(t *testing.T) {
	targetTeam := "monetization"
	porPath := "docs/marketing/OPERATING_MODEL.md"
	rels := BuildRuntimeOperatingRelationships(operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*"}},
			RequiredRead: []RequiredReadEntry{{
				Prefix: "marketing/notebook/*",
			}},
			EvidenceConsumed: []EvidenceConsumedEntry{{
				Prefix:       "challenge-report/*",
				ForDecisions: []string{"audience-update"},
			}},
			Output: []OutputEntry{
				{Prefix: "hook-record/*", DestinationKind: DestinationKnowledge, DestinationTeam: &targetTeam},
				{Prefix: "por-update/*", DestinationKind: DestinationPORFile, DestinationPath: &porPath},
			},
			DecisionsOwned:       []string{"audience-update"},
			DecisionsConsumed:    []string{"campaign-launch-proposal"},
			RaisesCapabilityGaps: true,
			ExternalProducers:    []string{"bookmark-hub"},
		},
	}}), "team-a")

	set := NewOperatingRelationshipSet(rels)
	for _, kind := range []OperatingRelationshipKind{
		operatingRelTopicIntake,
		operatingRelTopicRequiredRead,
		operatingRelTopicEvidenceConsumed,
		operatingRelTopicOutput,
		operatingRelPOROutput,
		operatingRelDecisionOwned,
		operatingRelDecisionConsumed,
		operatingRelCapabilityGapRaised,
		operatingRelExternalProducer,
		operatingRelExternalProducerIntake,
		operatingRelCrossTeamOutput,
	} {
		if got := len(set.ByKind(kind)); got == 0 {
			t.Fatalf("runtime relationship kind %q missing from %+v", kind, rels)
		}
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

func TestExtractOperatingGraphBlocksRejectsUnsupportedAllowMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "OPERATING_MODEL.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: contract
allow: graph_edge_unbacked
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
`+"```"+`
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := ExtractOperatingGraphBlocks(path, "docs/x.md")
	if err == nil || !strings.Contains(err.Error(), `metadata field "allow" is not supported`) {
		t.Fatalf("expected unsupported allow metadata error, got %v", err)
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

func TestMarketingOperatingModelCentralizesNotebookDrainage(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "marketing", "OPERATING_MODEL.md")
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/marketing/OPERATING_MODEL.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	rels := NewOperatingRelationshipSet(BuildGraphOperatingRelationships(blocks[0]))
	wantBrandManagerRead := OperatingRelationship{Kind: operatingRelTopicRead, Team: "marketing-crew", Member: "brand-manager", Topic: "marketing/notebook/*"}
	if !operatingRelationshipSetContains(rels, wantBrandManagerRead) {
		t.Fatalf("marketing notebook must drain through brand-manager; relationships=%+v", rels.All())
	}
	for _, member := range []string{"researcher", "oss-advertiser", "subscription-advertiser", "publisher"} {
		forbidden := OperatingRelationship{Kind: operatingRelTopicRead, Team: "marketing-crew", Member: member, Topic: "marketing/notebook/*"}
		if operatingRelationshipSetContains(rels, forbidden) {
			t.Fatalf("raw marketing notebook should not be a direct runtime read for %s", member)
		}
	}
}

func TestBundledMarketingOperatingModelIsReconciled(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	storeDir := filepath.Join(repoRoot, "scenarios", "prompt-manager", "store")
	if _, err := os.Stat(storeDir); err != nil {
		t.Skipf("bundled prompt-manager store not available: %v", err)
	}
	blocks, err := LoadOperatingGraphBlocks(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingGraphBlocks: %v", err)
	}
	runtime, err := BuildOperatingGraphRuntime(repoRoot, storeDir)
	if err != nil {
		t.Fatalf("BuildOperatingGraphRuntime: %v", err)
	}

	result := ValidateOperatingGraphs(blocks, runtime, "marketing-crew", "marketing-operating-model")
	if result.Errors != 0 || result.Warnings != 0 {
		t.Fatalf("marketing validation counts changed: errors=%d warnings=%d findings=%+v", result.Errors, result.Warnings, result.Findings)
	}

	diffs := DiffOperatingGraphs(blocks, runtime, "marketing-crew", "marketing-operating-model")
	if got := countOperatingDiffs(diffs, "graph_relationship_missing_in_runtime"); got != 0 {
		t.Fatalf("graph-to-runtime diff count=%d, want 0: %+v", got, diffs)
	}
	if got := countOperatingDiffs(diffs, "runtime_relationship_missing_in_graph"); got != 0 {
		t.Fatalf("runtime-to-graph diff count=%d, want 0: %+v", got, diffs)
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

func assertOperatingFindingDetail(t *testing.T, result OperatingGraphValidationResult, rule, detailFragment string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Rule == rule && strings.Contains(f.Detail, detailFragment) {
			return
		}
	}
	t.Fatalf("finding rule=%q detail containing %q missing from %+v", rule, detailFragment, result.Findings)
}

func countOperatingDiffs(diffs []OperatingGraphContractDiff, kind string) int {
	var count int
	for _, diff := range diffs {
		if diff.Kind == kind {
			count++
		}
	}
	return count
}

func operatingRelationshipSetContains(set OperatingRelationshipSet, rel OperatingRelationship) bool {
	for _, candidate := range set.All() {
		if operatingGraphRelationshipsEquivalent(candidate, rel) {
			return true
		}
	}
	return false
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

func assertOperatingDiff(t *testing.T, diffs []OperatingGraphContractDiff, kind string, relationship OperatingRelationshipKind) OperatingGraphContractDiff {
	t.Helper()
	for _, diff := range diffs {
		if diff.Kind == kind && diff.Relationship == string(relationship) {
			return diff
		}
	}
	t.Fatalf("diff kind=%q relationship=%q missing from %+v", kind, relationship, diffs)
	return OperatingGraphContractDiff{}
}

func operatingStringPtr(value string) *string {
	return &value
}
