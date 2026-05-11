package memberflow

import (
	"os"
	"path/filepath"
	"prompt-manager/teamcontract"
	"strings"
	"testing"
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

func TestParseOperatingMermaidSubgraphsAndShapes(t *testing.T) {
	graph, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		`  subgraph INFLOWS["Inflows / Producers"]`,
		"    %% @node OP external:operator",
		"    OP([Operator])",
		"  end",
		"  subgraph TOPICS[Topics]",
		"    %% @node RI topic:research-inbox/*",
		"    RI[(research-inbox/*)]",
		"  end",
		"  %% @node D decision:audience-update",
		"  D{audience-update}",
		"  %% @node MON team:monetization",
		"  MON[[Monetization team]]",
		"  %% @node P por:docs/marketing/STRATEGY.md",
		"  P[/docs/marketing/STRATEGY.md/]",
		"  OP -->|produces| RI",
		"  RI --> D",
		"  RI --> MON",
	}, 1)
	if err != nil {
		t.Fatalf("ParseOperatingMermaid: %v", err)
	}
	if len(graph.Groups) != 2 {
		t.Fatalf("groups=%d, want 2: %+v", len(graph.Groups), graph.Groups)
	}
	if graph.Groups[0].ID != "INFLOWS" || graph.Groups[0].Display != "Inflows / Producers" || strings.Join(graph.Groups[0].NodeIDs, ",") != "OP" {
		t.Fatalf("bad inflow group: %+v", graph.Groups[0])
	}
	if node := operatingNodeByID(t, graph, "RI"); node.Shape != OperatingGraphNodeShapeCylinder || node.Kind != OperatingGraphNodeKindTopic {
		t.Fatalf("bad topic node shape: %+v", node)
	}
	if node := operatingNodeByID(t, graph, "D"); node.Shape != OperatingGraphNodeShapeDiamond || node.Kind != OperatingGraphNodeKindDecision {
		t.Fatalf("bad decision node shape: %+v", node)
	}
	if node := operatingNodeByID(t, graph, "MON"); node.Shape != OperatingGraphNodeShapeSubroutine || node.Kind != OperatingGraphNodeKindTeam {
		t.Fatalf("bad team node shape: %+v", node)
	}
	if node := operatingNodeByID(t, graph, "P"); node.Shape != OperatingGraphNodeShapeDocument || node.Kind != OperatingGraphNodeKindPOR {
		t.Fatalf("bad por node shape: %+v", node)
	}
	if graph.Edges[0].Label != "produces" {
		t.Fatalf("edge label = %q", graph.Edges[0].Label)
	}
}

func TestParseOperatingMermaidRejectsMalformedSubgraphs(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "nested",
			lines: []string{
				"flowchart LR",
				"subgraph A[Group A]",
				"subgraph B[Group B]",
				"end",
				"end",
			},
		},
		{
			name: "unclosed",
			lines: []string{
				"flowchart LR",
				"subgraph A[Group A]",
				`A["member:a"]`,
			},
		},
		{
			name: "unmatched end",
			lines: []string{
				"flowchart LR",
				"end",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseOperatingMermaid("g", tc.lines, 1); err == nil {
				t.Fatalf("expected subgraph parse error")
			}
		})
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

func TestExtractOperatingGraphDocsParsesTopicCatalogAndDecisions(t *testing.T) {
	graph, err := ParseOperatingMermaid("g", []string{
		"flowchart LR",
		"  %% @node R member:researcher",
		"  R[Researcher]",
		"  %% @node PUB member:publisher",
		"  PUB[Publisher]",
		"  %% @node OP external:operator",
		"  OP[Operator]",
	}, 1)
	if err != nil {
		t.Fatalf("ParseOperatingMermaid: %v", err)
	}
	docs := ExtractOperatingGraphDocsForGraph(strings.Split(`
## Topic Catalog

Prose before the table is allowed.

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:research-inbox/*`"+` | live | operator, researcher | researcher | Raw signal. |
| `+"`topic[future]:ad-run/<lane>/*`"+` | target | advertisers | publisher | Run summaries. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `+"`content-publish-proposal`"+` | advertiser or publisher | Publish gate. | Draft and release evidence. | Publisher updates `+"`topic:publish-log/*`"+`. |
`, "\n"), OperatingGraphMetadata{Extra: map[string]string{
		"actor_alias.advertisers": "group:advertisers",
		"actor_alias.advertiser":  "group:advertisers",
		"actor_group.advertisers": "member:oss-advertiser, member:subscription-advertiser",
	}}, graph)

	if !docs.TopicCatalog.Present || len(docs.TopicCatalog.Rows) != 2 {
		t.Fatalf("bad topic catalog parse: %+v", docs.TopicCatalog)
	}
	if docs.TopicCatalog.Rows[0].StatusKind != OperatingTopicStatusLive || docs.TopicCatalog.Rows[1].StatusKind != OperatingTopicStatusTarget {
		t.Fatalf("bad status parse: %+v", docs.TopicCatalog.Rows)
	}
	if row := docs.TopicCatalog.Rows[1]; row.Topic != "ad-run/<lane>/*" || row.Qualifier != "future" || len(row.Writers) != 1 || row.Writers[0].Kind != "group" {
		t.Fatalf("bad topic row: %+v", row)
	}
	if row := docs.TopicCatalog.Rows[1]; len(row.Readers) != 1 || row.Readers[0].Kind != OperatingActorKindMember || row.Readers[0].Value != "publisher" {
		t.Fatalf("reader should resolve from graph node label: %+v", row.Readers)
	}
	if !docs.Decisions.Present || len(docs.Decisions.Rows) != 1 {
		t.Fatalf("bad decisions parse: %+v", docs.Decisions)
	}
	if row := docs.Decisions.Rows[0]; row.Decision != "content-publish-proposal" || len(row.Owners) != 2 || row.ExpectedEvidenceTrigger == "" || row.AcceptedEffect == "" {
		t.Fatalf("bad decision row: %+v", row)
	}
}

func TestExtractOperatingGraphBlocksScopesDocsTablesPerGraph(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g1
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A1["member:a"]
  T1["topic:first/*"]
  D1["decision:first-decision"]
  T1 --> A1
  A1 --> D1
`+"```"+`

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:first/*`"+` | live | member:a | member:a | First. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `+"`first-decision`"+` | member:a | First decision. | First evidence. | Member writes `+"`topic:first/*`"+`. |

<!-- prompt-manager-graph:
id: g2
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A2["member:a"]
  T2["topic:second/*"]
  D2["decision:second-decision"]
  T2 --> A2
  A2 --> D2
`+"```"+`

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:second/*`"+` | live | member:a | member:a | Second. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `+"`second-decision`"+` | member:a | Second decision. | Second evidence. | Member writes `+"`topic:second/*`"+`. |
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	blocks, err := ExtractOperatingGraphBlocks(path, "docs/test/multi.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want 2", len(blocks))
	}
	if got := blocks[0].Docs.TopicCatalog.Rows; len(got) != 1 || got[0].Topic != "first/*" {
		t.Fatalf("g1 topic catalog leaked or missing rows: %+v", got)
	}
	if got := blocks[0].Docs.Decisions.Rows; len(got) != 1 || got[0].Decision != "first-decision" {
		t.Fatalf("g1 decisions leaked or missing rows: %+v", got)
	}
	if got := blocks[1].Docs.TopicCatalog.Rows; len(got) != 1 || got[0].Topic != "second/*" {
		t.Fatalf("g2 topic catalog leaked or missing rows: %+v", got)
	}
	if got := blocks[1].Docs.Decisions.Rows; len(got) != 1 || got[0].Decision != "second-decision" {
		t.Fatalf("g2 decisions leaked or missing rows: %+v", got)
	}
}

func TestExtractOperatingGraphBlocksDoesNotBorrowLaterDocsTables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g1
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A1["member:a"]
  T1["topic:first/*"]
  D1["decision:first-decision"]
  T1 --> A1
  A1 --> D1
`+"```"+`

<!-- prompt-manager-graph:
id: g2
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A2["member:a"]
  T2["topic:second/*"]
  D2["decision:second-decision"]
  T2 --> A2
  A2 --> D2
`+"```"+`

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:first/*`"+` | live | member:a | member:a | This row belongs to the later graph's docs range. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
| `+"`first-decision`"+` | member:a | Later decision. | Later evidence. | Member writes `+"`topic:first/*`"+`. |
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	blocks, err := ExtractOperatingGraphBlocks(path, "docs/test/multi.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want 2", len(blocks))
	}
	if blocks[0].Docs.TopicCatalog.Present || blocks[0].Docs.Decisions.Present {
		t.Fatalf("g1 borrowed later docs tables: %+v", blocks[0].Docs)
	}

	result := ValidateOperatingGraphs(blocks, OperatingGraphRuntime{}, "team-a", "g1")
	assertOperatingFinding(t, result, "graph_topic_catalog_missing")
	assertOperatingFinding(t, result, "graph_decisions_table_missing")
}

func TestValidateOperatingGraphsDocsTablesDetectDrift(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  T["topic:research-inbox/*"]`,
		`  D["decision:audience-update"]`,
		"  T --> M",
		"  M --> D",
	})
	block.Docs = OperatingGraphDocs{
		TopicCatalog: OperatingTopicCatalogTable{Present: true, Rows: []OperatingTopicCatalogRow{{
			Topic:      "ghost/*",
			RawTopic:   "`topic:ghost/*`",
			Status:     "live",
			SourceLine: 20,
		}}},
		Decisions: OperatingDecisionTable{Present: true, Rows: []OperatingDecisionRow{{
			Decision:   "audience-update",
			Owners:     []OperatingActorReference{{Kind: "member", Value: "brand-manager", Raw: "brand-manager"}},
			SourceLine: 30,
		}}},
	}
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{
			Intake:         []IntakeEntry{{Prefix: "research-inbox/*"}},
			DecisionsOwned: []string{"audience-update"},
		},
	}})
	runtime.Contracts = TeamContractRegistry{"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
		Members: map[string]teamcontract.MemberContract{
			"researcher":    {},
			"brand-manager": {},
		},
		DecisionContext: map[string]teamcontract.DecisionContext{"audience-update": {}},
	}}}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFinding(t, result, "graph_topic_catalog_drift")
	assertOperatingFinding(t, result, "graph_decisions_table_owner_drift")
}

func TestValidateOperatingGraphsDocsCatalogStatusRules(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  LIVE["topic:live/*"]`,
		`  FUT["topic[future]:future/*"]`,
		`  OLD["topic[old]:old/*"]`,
		"  LIVE --> M",
	})
	block.Docs = OperatingGraphDocs{TopicCatalog: OperatingTopicCatalogTable{Present: true, Rows: []OperatingTopicCatalogRow{
		{Topic: "live/*", Qualifier: "future", RawTopic: "`topic[future]:live/*`", Status: "live", StatusKind: OperatingTopicStatusLive, SourceLine: 20},
		{Topic: "future/*", RawTopic: "`topic:future/*`", Status: "target", StatusKind: OperatingTopicStatusTarget, SourceLine: 21},
		{Topic: "old/*", Qualifier: "old", RawTopic: "`topic[old]:old/*`", Status: "mystery", StatusKind: OperatingTopicStatusUnknown, SourceLine: 22},
		{Topic: "transitional/*", RawTopic: "`topic:transitional/*`", Status: "live transitional", StatusKind: OperatingTopicStatusLiveTransitional, SourceLine: 23},
	}}}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, operatingDiffRuntime(t, nil), "team-a", "g")
	assertOperatingFinding(t, result, "graph_topic_catalog_unknown_status")
	assertOperatingFinding(t, result, "graph_topic_catalog_status_qualifier_drift")
	assertOperatingFinding(t, result, "graph_topic_catalog_transitional_without_target")
	assertOperatingFinding(t, result, "graph_topic_catalog_live_status_unbacked")
}

func TestValidateOperatingGraphsDocsCatalogActorParity(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  R["member:researcher"]`,
		`  BM["member:brand-manager"]`,
		`  OSS["member:oss-advertiser"]`,
		`  SUB["member:subscription-advertiser"]`,
		`  OP["external:operator"]`,
		`  MON["team:monetization"]`,
		`  IN["topic:research-inbox/*"]`,
		`  HOOK["topic:hook-record/*"]`,
		`  MB["topic:monetization-benchmark/*"]`,
		"  OP --> IN",
		"  OP --> R",
		"  IN --> R",
		"  R --> HOOK",
		"  HOOK --> OSS",
		"  HOOK --> SUB",
		"  MB --> MON",
	})
	block.Metadata.Extra = map[string]string{
		"actor_alias.advertisers": "group:advertisers",
		"actor_group.advertisers": "member:oss-advertiser, member:subscription-advertiser",
	}
	block.Docs = OperatingGraphDocs{TopicCatalog: OperatingTopicCatalogTable{Present: true, Rows: []OperatingTopicCatalogRow{
		{
			Topic: "research-inbox/*", Status: "live", StatusKind: OperatingTopicStatusLive, RawTopic: "`topic:research-inbox/*`", SourceLine: 20,
			Writers: []OperatingActorReference{{Kind: OperatingActorKindExternal, Value: "operator", Raw: "operator"}},
			Readers: []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "researcher", Raw: "researcher"}},
		},
		{
			Topic: "hook-record/*", Status: "live", StatusKind: OperatingTopicStatusLive, RawTopic: "`topic:hook-record/*`", SourceLine: 21,
			Writers: []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "researcher", Raw: "researcher"}},
			Readers: []OperatingActorReference{{Kind: OperatingActorKindGroup, Value: "advertisers", Raw: "advertisers"}},
		},
		{
			Topic: "monetization-benchmark/*", Status: "live", StatusKind: OperatingTopicStatusLive, RawTopic: "`topic:monetization-benchmark/*`", SourceLine: 22,
			Readers: []OperatingActorReference{{Kind: OperatingActorKindTeam, Value: "monetization", Raw: "monetization"}},
		},
		{
			Topic: "hook-record/*", Status: "live", StatusKind: OperatingTopicStatusLive, RawTopic: "`topic:hook-record/*`", SourceLine: 23,
			Readers: []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "brand-manager", Raw: "brand-manager"}},
		},
		{
			Topic: "hook-record/*", Status: "live", StatusKind: OperatingTopicStatusLive, RawTopic: "`topic:hook-record/*`", SourceLine: 24,
			Readers: []OperatingActorReference{{Kind: OperatingActorKindExternal, Value: "operator", Raw: "operator"}},
		},
	}}}
	runtime := operatingDiffRuntime(t, []MemberTopics{
		{
			Ref: MemberRef{Team: "team-a", Member: "researcher"},
			Topics: Topics{
				Intake:            []IntakeEntry{{Prefix: "research-inbox/*"}},
				Output:            []OutputEntry{{Prefix: "hook-record/*", DestinationKind: DestinationKnowledge}, {Prefix: "monetization-benchmark/*", DestinationKind: DestinationKnowledge, DestinationTeam: operatingStringPtr("monetization")}},
				ExternalProducers: []string{"operator"},
			},
		},
		{Ref: MemberRef{Team: "team-a", Member: "oss-advertiser"}, Topics: Topics{RequiredRead: []RequiredReadEntry{{Prefix: "hook-record/*"}}}},
		{Ref: MemberRef{Team: "team-a", Member: "subscription-advertiser"}, Topics: Topics{RequiredRead: []RequiredReadEntry{{Prefix: "hook-record/*"}}}},
		{Ref: MemberRef{Team: "team-a", Member: "brand-manager"}, Topics: Topics{}},
	})

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFinding(t, result, "graph_topic_catalog_reader_drift")
	assertOperatingFindingAbsent(t, result, "graph_topic_catalog_actor_unsupported")
	assertOperatingFindingAbsent(t, result, "graph_topic_catalog_writer_drift")
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
				Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: OperatingGraphMode(tc.mode)},
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

func TestValidateOperatingGraphsDetectsMissingContractMember(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
		}),
		Source: OperatingGraphSource{Path: "docs/test/OPERATING_MODEL.md", FenceLine: 2},
	}
	runtime := OperatingGraphRuntime{
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members: map[string]teamcontract.MemberContract{
					"member-a": {},
					"member-b": {},
				},
			}},
		},
		PromptSections: map[MemberRef][]OperatingGraphPromptSection{
			{Team: "team-a", Member: "member-a"}: {{
				Team:       "team-a",
				Member:     "member-a",
				Kind:       operatingGraphPromptSectionKindTopicContract,
				SourcePath: expectedTopicContractSourcePath("team-a", "member-a"),
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFinding(t, result, "graph_declared_member_missing")
}

func TestValidateOperatingGraphsCheckableMayOmitContractMembers(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "checkable"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
		}),
	}
	runtime := OperatingGraphRuntime{
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members: map[string]teamcontract.MemberContract{"member-b": {}},
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingAbsent(t, result, "graph_declared_member_missing")
}

func TestValidateOperatingGraphsDetectsUnsupportedTypedEdgeSemantics(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "checkable"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
			`  B["member:member-b"]`,
			"  A --> B",
		}),
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "graph_unsupported_edge_semantics")
	assertOperatingFindingAbsent(t, result, "graph_edge_unbacked")
}

func TestValidateOperatingGraphsWarnsOnShapeConventionDrift(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "checkable"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  T["topic:research-inbox/*"]`,
			`  M["member:researcher"]`,
			"  T --> M",
		}),
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "graph_node_shape_convention_drift")
}

func TestValidateOperatingGraphsIgnoresNonActionableUnsupportedEdges(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "process endpoint",
			lines: []string{
				"flowchart LR",
				`  P["process:planning"]`,
				`  T["topic:research-inbox/*"]`,
				"  P --> T",
			},
		},
		{
			name: "future topic endpoint",
			lines: []string{
				"flowchart LR",
				`  T["topic[future]:publish-performance/*"]`,
				`  P["process:learning"]`,
				"  T --> P",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			block := OperatingGraphBlock{
				Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "checkable"},
				Graph:    mustParseGraph(t, tc.lines),
			}
			result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, OperatingGraphRuntime{}, "team-a", "g")
			assertOperatingFindingAbsent(t, result, "graph_unsupported_edge_semantics")
		})
	}
}

func TestValidateOperatingGraphsSupportedUnbackedEdgeIsNotUnsupported(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "checkable"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  M["member:member-a"]`,
			`  T["topic:hook-record/*"]`,
			"  M --> T",
		}),
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "graph_edge_unbacked")
	assertOperatingFindingAbsent(t, result, "graph_unsupported_edge_semantics")
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
		`  T["topic:marketing-craft-observation/*"]`,
		"  T --> M",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref:    MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{},
	}})

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	diff := assertOperatingDiff(t, diffs, "graph_relationship_missing_in_runtime", operatingRelTopicRead)
	if diff.Member != "researcher" || diff.Topic != "marketing-craft-observation/*" || diff.RuntimePath == "" || len(diff.Suggestions) == 0 {
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
			Prefix: "marketing-craft-observation/*",
		}}},
	}})

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	diff := assertOperatingDiff(t, diffs, "runtime_relationship_missing_in_graph", operatingRelTopicRead)
	if diff.Member != "researcher" || diff.Topic != "marketing-craft-observation/*" || diff.RuntimePath == "" || len(diff.Suggestions) == 0 {
		t.Fatalf("unexpected diff: %+v", diff)
	}
}

func TestDiffOperatingGraphsGraphReadMatchesRuntimeReadSubtypes(t *testing.T) {
	for name, topics := range map[string]Topics{
		"intake": {Intake: []IntakeEntry{{Prefix: "marketing-craft-observation/*"}}},
		"required_read": {RequiredRead: []RequiredReadEntry{{
			Prefix: "marketing-craft-observation/*",
		}}},
		"evidence_consumed": {EvidenceConsumed: []EvidenceConsumedEntry{{
			Prefix: "marketing-craft-observation/*",
		}}},
	} {
		t.Run(name, func(t *testing.T) {
			block := operatingDiffBlock(t, []string{
				"flowchart LR",
				`  M["member:researcher"]`,
				`  T["topic:marketing-craft-observation/*"]`,
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

func TestDiffOperatingGraphsExternalTopicDoesNotSatisfyMemberExternalProducer(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  OP["external:operator"]`,
		`  IN["topic:research-inbox/*"]`,
		`  A["member:member-a"]`,
		`  B["member:member-b"]`,
		"  OP --> IN",
		"  OP --> A",
		"  IN --> A",
		"  IN --> B",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{
		{
			Ref: MemberRef{Team: "team-a", Member: "member-a"},
			Topics: Topics{
				Intake:            []IntakeEntry{{Prefix: "research-inbox/*"}},
				ExternalProducers: []string{"operator"},
			},
		},
		{
			Ref: MemberRef{Team: "team-a", Member: "member-b"},
			Topics: Topics{
				Intake:            []IntakeEntry{{Prefix: "research-inbox/*"}},
				ExternalProducers: []string{"operator"},
			},
		},
	})

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	diff := assertOperatingDiff(t, diffs, "runtime_relationship_missing_in_graph", operatingRelExternalProducer)
	if diff.Member != "member-b" || diff.External != "operator" {
		t.Fatalf("unexpected external producer diff: %+v", diff)
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingDetail(t, result, "graph_declared_external_producer_missing", "member-b")
}

func TestDiffOperatingGraphsExternalTopicAloneDoesNotSatisfyExternalMember(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  OP["external:operator"]`,
		`  IN["topic:research-inbox/*"]`,
		`  A["member:member-a"]`,
		"  OP --> IN",
		"  IN --> A",
	})
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "member-a"},
		Topics: Topics{
			Intake:            []IntakeEntry{{Prefix: "research-inbox/*"}},
			ExternalProducers: []string{"operator"},
		},
	}})

	diffs := DiffOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	diff := assertOperatingDiff(t, diffs, "runtime_relationship_missing_in_graph", operatingRelExternalProducer)
	if diff.Member != "member-a" || diff.External != "operator" {
		t.Fatalf("unexpected external producer diff: %+v", diff)
	}
}

func TestValidateOperatingGraphsPromptTopicContractSections(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
			`  B["member:member-b"]`,
			`  C["member:member-c"]`,
		}),
	}
	runtime := OperatingGraphRuntime{
		PromptSections: map[MemberRef][]OperatingGraphPromptSection{
			{Team: "team-a", Member: "member-a"}: {{
				Team:       "team-a",
				Member:     "member-a",
				Kind:       operatingGraphPromptSectionKindTopicContract,
				SourcePath: expectedTopicContractSourcePath("team-a", "member-a"),
			}},
			{Team: "team-a", Member: "member-b"}: {{
				Team:       "team-a",
				Member:     "member-b",
				Kind:       operatingGraphPromptSectionKindTopicContract,
				SourcePath: "teams/team-a/members/member-b/old-topics.json",
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingDetail(t, result, "graph_prompt_topic_contract_missing", "member-c")
	assertOperatingFindingDetail(t, result, "graph_prompt_topic_contract_source_mismatch", "member-b")
}

func TestValidateOperatingGraphsPromptTopicContractSectionsPassWhenPresent(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
		}),
	}
	runtime := OperatingGraphRuntime{
		PromptSections: map[MemberRef][]OperatingGraphPromptSection{
			{Team: "team-a", Member: "member-a"}: {{
				Team:       "team-a",
				Member:     "member-a",
				Kind:       operatingGraphPromptSectionKindTopicContract,
				SourcePath: expectedTopicContractSourcePath("team-a", "member-a"),
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingAbsent(t, result, "graph_prompt_topic_contract_missing")
	assertOperatingFindingAbsent(t, result, "graph_prompt_topic_contract_source_mismatch")
}

func TestValidateOperatingGraphsPromptTopicContractDerivedSectionsAreNotLiveProof(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
		}),
	}
	runtime := OperatingGraphRuntime{
		Members: []MemberTopics{{Ref: MemberRef{Team: "team-a", Member: "member-a"}}},
		PromptSections: map[MemberRef][]OperatingGraphPromptSection{
			{Team: "team-a", Member: "member-a"}: {{
				Team:       "team-a",
				Member:     "member-a",
				Kind:       operatingGraphPromptSectionKindTopicContract,
				SourcePath: expectedTopicContractSourcePath("team-a", "member-a"),
				SourceKind: OperatingGraphPromptSectionSourceDerived,
				Content:    "# Topic Contract\n\nderived content",
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingAbsent(t, result, "graph_prompt_topic_contract_missing")
	assertOperatingFindingAbsent(t, result, "graph_prompt_topic_contract_content_mismatch")

	coverage := BuildOperatingGraphCoverage([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	if coverage[0].Prompts.TopicContractSourceKind != string(OperatingGraphPromptSectionSourceDerived) {
		t.Fatalf("prompt source kind=%q", coverage[0].Prompts.TopicContractSourceKind)
	}
	if coverage[0].Prompts.TopicContractContentParity != OperatingCoverageStatusUnavailable {
		t.Fatalf("content parity=%q", coverage[0].Prompts.TopicContractContentParity)
	}
}

func TestValidateOperatingGraphsPromptTopicContractContentMismatch(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  A["member:member-a"]`,
		}),
	}
	member := MemberTopics{
		Ref: MemberRef{Team: "team-a", Member: "member-a"},
		Topics: Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*", Taxonomy: "research"}},
		},
	}
	runtime := OperatingGraphRuntime{
		Members: []MemberTopics{member},
		PromptSections: map[MemberRef][]OperatingGraphPromptSection{
			member.Ref: {{
				Team:       "team-a",
				Member:     "member-a",
				Kind:       operatingGraphPromptSectionKindTopicContract,
				SourcePath: expectedTopicContractSourcePath("team-a", "member-a"),
				Content:    "# Topic Contract\n\nstale content",
			}},
		},
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFinding(t, result, "graph_prompt_topic_contract_content_mismatch")
}

func TestDefaultOperatingGraphRulesRegistersBaselineContractRules(t *testing.T) {
	want := []string{
		"graph_untyped_node",
		"graph_unknown_node_kind",
		"graph_node_shape_convention_drift",
		"graph_unknown_member",
		"graph_unknown_decision",
		"graph_unknown_team",
		"graph_unknown_por",
		"graph_topic_unresolved",
		"graph_future_topic_live_edge",
		"graph_unsupported_edge_semantics",
		"graph_edge_unbacked",
		"graph_declared_member_missing",
		"graph_declared_intake_missing",
		"graph_declared_required_read_missing",
		"graph_declared_evidence_missing",
		"graph_declared_output_missing",
		"graph_declared_decision_owned_missing",
		"graph_declared_decision_consumed_missing",
		"graph_declared_capability_gap_missing",
		"graph_declared_external_producer_missing",
		"graph_declared_cross_team_output_missing",
		"graph_topic_catalog_missing",
		"graph_topic_catalog_invalid_topic",
		"graph_topic_catalog_drift",
		"graph_topic_catalog_unknown_status",
		"graph_topic_catalog_status_qualifier_drift",
		"graph_topic_catalog_live_status_unbacked",
		"graph_topic_catalog_transitional_without_target",
		"graph_topic_catalog_purpose_drift",
		"graph_docs_unknown_actor",
		"graph_topic_catalog_writer_drift",
		"graph_topic_catalog_reader_drift",
		"graph_topic_catalog_actor_unsupported",
		"graph_decisions_table_missing",
		"graph_decisions_table_drift",
		"graph_decisions_table_owner_drift",
		"graph_prompt_topic_contract_missing",
		"graph_prompt_topic_contract_source_mismatch",
		"graph_prompt_topic_contract_content_mismatch",
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

func TestDefaultOperatingModelRulesRegistersBaselineContractRules(t *testing.T) {
	want := []string{
		"operating_model_required_section_missing",
		"operating_model_duplicate_section",
		"operating_model_decisions_header_drift",
		"operating_model_decisions_empty",
		"operating_model_decisions_row_incomplete",
		"operating_model_decisions_effect_weak",
		"operating_model_external_inputs_table_missing",
		"operating_model_external_inputs_header_drift",
		"operating_model_external_inputs_empty",
		"operating_model_external_inputs_row_incomplete",
		"operating_model_external_inputs_producer_unbacked",
		"operating_model_external_inputs_entry_unbacked",
		"operating_model_external_inputs_drainer_unbacked",
		"operating_model_outputs_table_missing",
		"operating_model_outputs_header_drift",
		"operating_model_outputs_empty",
		"operating_model_outputs_row_incomplete",
		"operating_model_outputs_surface_unbacked",
		"operating_model_outputs_consumer_unbacked",
		"operating_model_feedback_steps_missing",
		"operating_model_feedback_step_unanchored",
		"operating_model_feedback_reference_unbacked",
		"operating_model_gaps_items_missing",
		"operating_model_gap_item_unanchored",
		"operating_model_gap_item_target_state_missing",
		"operating_model_adoption_command_missing",
		"operating_model_plan_of_record_missing",
		"operating_model_readme_link_missing",
	}
	rules := DefaultOperatingModelRules()
	if len(rules) != len(want) {
		t.Fatalf("rules=%d, want %d", len(rules), len(want))
	}
	seen := map[string]bool{}
	contractModel := operatingModelDocumentFixture(t)
	contractModel.Graphs[0].Metadata.Mode = OperatingGraphModeContract
	explanatoryModel := contractModel
	explanatoryModel.Graphs = append([]OperatingGraphBlock(nil), contractModel.Graphs...)
	explanatoryModel.Graphs[0].Metadata.Mode = OperatingGraphModeExplanatory
	for i, rule := range rules {
		if rule.ID() != want[i] {
			t.Fatalf("rule[%d]=%q, want %q", i, rule.ID(), want[i])
		}
		if seen[rule.ID()] {
			t.Fatalf("duplicate operating model rule id %q", rule.ID())
		}
		seen[rule.ID()] = true
		if rule.ID() == "" || rule.Group() == "" || rule.DefaultSeverity() == "" {
			t.Fatalf("rule[%d] has incomplete metadata: id=%q group=%q severity=%q", i, rule.ID(), rule.Group(), rule.DefaultSeverity())
		}
		if !rule.AppliesTo(contractModel) {
			t.Fatalf("rule[%d] should apply to contract models", i)
		}
		if rule.AppliesTo(explanatoryModel) {
			t.Fatalf("rule[%d] should not apply to explanatory models", i)
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

func TestValidateOperatingGraphsTopicCatalogPurposeDrift(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  M["member:member-a"]`,
			`  T["topic:audience-scan/*"]`,
			"  M --> T",
			"  T --> M",
		}),
		Docs: OperatingGraphDocs{TopicCatalog: OperatingTopicCatalogTable{
			Present: true,
			Rows: []OperatingTopicCatalogRow{{
				Topic:      "audience-scan/*",
				Status:     "live",
				StatusKind: OperatingTopicStatusLive,
				Writers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "member-a", Raw: "member:a"}},
				Readers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "member-a", Raw: "member:a"}},
				Purpose:    "Documented purpose.",
				SourceLine: 12,
				RawTopic:   "`topic:audience-scan/*`",
			}},
		}, Decisions: OperatingDecisionTable{Present: true}},
		Source: OperatingGraphSource{Path: "docs/team/OPERATING_MODEL.md", FenceLine: 1},
	}
	runtime := OperatingGraphRuntime{
		Members: []MemberTopics{{
			Ref: MemberRef{Team: "team-a", Member: "member-a"},
			Topics: Topics{
				RequiredRead: []RequiredReadEntry{{Prefix: "audience-scan/*"}},
				Output:       []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge}},
			},
		}},
		Contracts: TeamContractRegistry{
			"team-a": {
				TeamID: "team-a",
				Contract: &teamcontract.OperatingContract{
					Members:         map[string]teamcontract.MemberContract{"member-a": {}},
					DecisionContext: map[string]teamcontract.DecisionContext{},
				},
				TopicCatalog: []TopicCatalogEntry{{
					Prefix:  "audience-scan/*",
					Status:  "live",
					Purpose: "Structured purpose.",
				}},
			},
		},
		PromptSections: derivedTopicContractPromptSections([]MemberTopics{{
			Ref: MemberRef{Team: "team-a", Member: "member-a"},
			Topics: Topics{
				RequiredRead: []RequiredReadEntry{{Prefix: "audience-scan/*"}},
				Output:       []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge}},
			},
		}}),
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFinding(t, result, "graph_topic_catalog_purpose_drift")

	coverage := BuildOperatingGraphCoverage([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	if len(coverage) != 1 {
		t.Fatalf("coverage length=%d, want 1", len(coverage))
	}
	if coverage[0].Docs.TopicCatalogPurposeMismatch != 1 || coverage[0].Docs.TopicCatalogPurposeMatched != 0 {
		t.Fatalf("unexpected purpose coverage: %+v", coverage[0].Docs)
	}
}

func TestValidateOperatingGraphsTopicCatalogPurposeParityClean(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  M["member:member-a"]`,
			`  T["topic:audience-scan/*"]`,
			"  M --> T",
			"  T --> M",
		}),
		Docs: OperatingGraphDocs{TopicCatalog: OperatingTopicCatalogTable{
			Present: true,
			Rows: []OperatingTopicCatalogRow{{
				Topic:      "audience-scan/*",
				Status:     "live",
				StatusKind: OperatingTopicStatusLive,
				Writers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "member-a", Raw: "member:a"}},
				Readers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "member-a", Raw: "member:a"}},
				Purpose:    "Structured purpose.",
				SourceLine: 12,
				RawTopic:   "`topic:audience-scan/*`",
			}},
		}, Decisions: OperatingDecisionTable{Present: true}},
		Source: OperatingGraphSource{Path: "docs/team/OPERATING_MODEL.md", FenceLine: 1},
	}
	member := MemberTopics{
		Ref: MemberRef{Team: "team-a", Member: "member-a"},
		Topics: Topics{
			RequiredRead: []RequiredReadEntry{{Prefix: "audience-scan/*"}},
			Output:       []OutputEntry{{Prefix: "audience-scan/*", DestinationKind: DestinationKnowledge}},
		},
	}
	runtime := OperatingGraphRuntime{
		Members: []MemberTopics{member},
		Contracts: TeamContractRegistry{
			"team-a": {
				TeamID: "team-a",
				Contract: &teamcontract.OperatingContract{
					Members:         map[string]teamcontract.MemberContract{"member-a": {}},
					DecisionContext: map[string]teamcontract.DecisionContext{},
				},
				TopicCatalog: []TopicCatalogEntry{{
					Prefix:  "audience-scan/*",
					Status:  "live",
					Purpose: "Structured purpose.",
				}},
			},
		},
		PromptSections: derivedTopicContractPromptSections([]MemberTopics{member}),
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingAbsent(t, result, "graph_topic_catalog_purpose_drift")
	coverage := BuildOperatingGraphCoverage([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	if coverage[0].Docs.TopicCatalogPurposeMatched != 1 || coverage[0].Docs.TopicCatalogPurposeMismatch != 0 {
		t.Fatalf("unexpected purpose coverage: %+v", coverage[0].Docs)
	}
}

func TestValidateOperatingGraphsTreatsLiveSystemTopicWriterAsStagedBoundary(t *testing.T) {
	block := OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph: mustParseGraph(t, []string{
			"flowchart LR",
			`  P["member:publisher"]`,
			`  T["topic:decision-application/<decision-id>"]`,
			"  T --> P",
		}),
		Docs: OperatingGraphDocs{TopicCatalog: OperatingTopicCatalogTable{
			Present: true,
			Rows: []OperatingTopicCatalogRow{{
				Topic:      "decision-application/<decision-id>",
				Status:     "live system",
				StatusKind: OperatingTopicStatusLiveSystem,
				Writers:    []OperatingActorReference{{Kind: OperatingActorKindExternal, Value: "decision-workflow", Raw: "decision workflow"}},
				Readers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "publisher", Raw: "publisher"}},
				Purpose:    "Structured purpose.",
				SourceLine: 12,
				RawTopic:   "`topic:decision-application/<decision-id>`",
			}},
		}, Decisions: OperatingDecisionTable{Present: true}},
		Source: OperatingGraphSource{Path: "docs/team/OPERATING_MODEL.md", FenceLine: 1},
	}
	member := MemberTopics{
		Ref: MemberRef{Team: "team-a", Member: "publisher"},
		Topics: Topics{
			RequiredRead: []RequiredReadEntry{{Prefix: "decision-application/<decision-id>"}},
		},
	}
	runtime := OperatingGraphRuntime{
		Members: []MemberTopics{member},
		Contracts: TeamContractRegistry{
			"team-a": {
				TeamID: "team-a",
				Contract: &teamcontract.OperatingContract{
					Members:         map[string]teamcontract.MemberContract{"publisher": {}},
					DecisionContext: map[string]teamcontract.DecisionContext{},
				},
				TopicCatalog: []TopicCatalogEntry{{
					Prefix:  "decision-application/<decision-id>",
					Status:  "live system",
					Purpose: "Structured purpose.",
				}},
			},
		},
		PromptSections: derivedTopicContractPromptSections([]MemberTopics{member}),
	}

	result := ValidateOperatingGraphs([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	assertOperatingFindingAbsent(t, result, "graph_topic_catalog_writer_drift")
	assertOperatingFindingAbsent(t, result, "graph_topic_catalog_actor_unsupported")
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
			`  READ["topic:marketing-craft-observation/*"]`,
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
				Prefix: "marketing-craft-observation/*",
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
				Prefix: "marketing-craft-observation/*",
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
				Prefix: "marketing-craft-observation/*",
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

func TestBuildOperatingGraphCoverageReportsRelationshipPromptDocsAndExclusions(t *testing.T) {
	runtime := operatingDiffRuntime(t, []MemberTopics{{
		Ref: MemberRef{Team: "team-a", Member: "researcher"},
		Topics: Topics{
			Intake: []IntakeEntry{{Prefix: "research-inbox/*"}},
			Output: []OutputEntry{
				{Prefix: "hook-record/*", DestinationKind: DestinationKnowledge},
				{Prefix: "campaign-draft/*", DestinationKind: DestinationKnowledge},
			},
		},
	}})
	runtime.PromptSections = map[MemberRef][]OperatingGraphPromptSection{
		{Team: "team-a", Member: "researcher"}: {{
			Kind:       operatingGraphPromptSectionKindTopicContract,
			SourcePath: expectedTopicContractSourcePath("team-a", "researcher"),
		}},
	}
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
		`  IN["topic:research-inbox/*"]`,
		`  OUT["topic:hook-record/*"]`,
		`  GHOST["topic:ghost-record/*"]`,
		`  PROC["process:learning-synthesis"]`,
		`  FUT["topic[future]:publish-performance/*"]`,
		`  OLD["topic[old]:oss-ad-run/*"]`,
		`  EXTREF["topic[external]:outside-signal/*"]`,
		"  IN --> M",
		"  M --> OUT",
		"  M --> GHOST",
		"  PROC --> M",
		"  FUT --> M",
		"  OLD --> M",
		"  EXTREF --> M",
	})

	coverage := BuildOperatingGraphCoverage([]OperatingGraphBlock{block}, runtime, "team-a", "g")
	if len(coverage) != 1 {
		t.Fatalf("coverage=%d, want 1: %+v", len(coverage), coverage)
	}
	topicRead := operatingCoverageByRelationship(coverage[0].Relationships, string(operatingRelTopicRead))
	if topicRead.RuntimeDeclared != 1 || topicRead.GraphShown != 1 || topicRead.Matched != 1 || topicRead.GraphOnly != 0 || topicRead.RuntimeOnly != 0 {
		t.Fatalf("unexpected topic_read coverage: %+v", topicRead)
	}
	if len(topicRead.RuntimeSubtypes) != 3 || operatingSubtypeCoverageByRelationship(topicRead.RuntimeSubtypes, string(operatingRelTopicIntake)).Covered != 1 {
		t.Fatalf("unexpected topic_read subtype coverage: %+v", topicRead.RuntimeSubtypes)
	}
	topicOutput := operatingCoverageByRelationship(coverage[0].Relationships, string(operatingRelTopicOutput))
	if topicOutput.RuntimeDeclared != 2 || topicOutput.GraphShown != 2 || topicOutput.Matched != 1 || topicOutput.GraphOnly != 1 || topicOutput.RuntimeOnly != 1 {
		t.Fatalf("unexpected topic_output coverage: %+v", topicOutput)
	}
	if coverage[0].Prompts.GraphMembers != 1 || coverage[0].Prompts.TopicContractPresent != 1 || coverage[0].Prompts.TopicContractSourceMatched != 1 {
		t.Fatalf("unexpected prompt coverage: %+v", coverage[0].Prompts)
	}
	if coverage[0].Prompts.TopicContractContentParity != OperatingCoverageStatusUnavailable {
		t.Fatalf("content parity=%q", coverage[0].Prompts.TopicContractContentParity)
	}
	if coverage[0].Docs.MermaidGraph != OperatingCoverageStatusEnforced || coverage[0].Docs.TopicCatalogTable != OperatingCoverageStatusMissing || coverage[0].Docs.DecisionsTable != OperatingCoverageStatusMissing {
		t.Fatalf("unexpected docs coverage: %+v", coverage[0].Docs)
	}
	for kind, want := range map[string]int{
		"process_nodes":        1,
		"future_topic_nodes":   1,
		"old_topic_nodes":      1,
		"external_topic_nodes": 1,
		"non_actionable_edges": 4,
	} {
		if got := operatingCoverageExclusionCount(coverage[0].Exclusions, kind); got != want {
			t.Fatalf("exclusion %s=%d, want %d: %+v", kind, got, want, coverage[0].Exclusions)
		}
	}
}

func TestBuildOperatingGraphCoverageSkipsExplanatoryGraphs(t *testing.T) {
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  M["member:researcher"]`,
	})
	block.Metadata.Mode = "explanatory"

	coverage := BuildOperatingGraphCoverage([]OperatingGraphBlock{block}, OperatingGraphRuntime{}, "team-a", "g")
	if len(coverage) != 0 {
		t.Fatalf("coverage=%d, want 0: %+v", len(coverage), coverage)
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

func TestExtractOperatingModelDocumentsParsesFullSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "OPERATING_MODEL.md")
	if err := os.WriteFile(path, []byte(`# Team Operating Model

## Mission

Keep the contract understandable.

## Scope

One team operating model.

## Operating Loops

1. Read the input.
2. Drain the output.

## Operating Graph

<!-- prompt-manager-graph:
id: g
scope: team
team: team-a
mode: contract
status: draft
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
  T["topic:first/*"]
  T --> A
`+"```"+`

## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:first/*`"+` | live | member:a | member:a | First. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|

## External Inputs / Triggers

| Producer / trigger | Entry surface | Drainer | Routing rule |
|---|---|---|---|

## Outputs / Downstream Consumers

| Output | Surface | Consumer | Purpose |
|---|---|---|---|

## Feedback / Capability Improvement Loop

1. Review `+"`topic:first/*`"+`.

## Current Implementation Gaps

1. `+"`topic[future]:second/*`"+` remains target-state until a producer exists.

## Adoption / Validation

- `+"`prompt-manager graph operating-model validate --team team-a --id g`"+`
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	models, err := ExtractOperatingModelDocuments(path, "docs/team/OPERATING_MODEL.md")
	if err != nil {
		t.Fatalf("ExtractOperatingModelDocuments: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("models=%d, want 1", len(models))
	}
	model := models[0]
	if model.ID != "g" || model.Team != "team-a" || model.Status != "draft" {
		t.Fatalf("unexpected model identity: %+v", model)
	}
	if !model.Sections.Mission.Present || !model.Sections.Graph.Present || model.Sections.Graph.Heading != "Operating Graph" {
		t.Fatalf("expected mission and graph sections: %+v", model.Sections)
	}
	if !model.Sections.TopicCatalog.Present || len(model.Sections.TopicCatalog.Rows) != 1 || model.Sections.TopicCatalog.Rows[0].Topic != "first/*" {
		t.Fatalf("unexpected topic catalog: %+v", model.Sections.TopicCatalog)
	}
	if !model.Sections.ExternalInputs.Present || !model.Sections.Outputs.Present || !model.Sections.FeedbackLoop.Present || !model.Sections.Gaps.Present || !model.Sections.Adoption.Present {
		t.Fatalf("expected full document sections: %+v", model.Sections)
	}
	if !model.Sections.ExternalInputs.Table || model.Sections.ExternalInputs.HeaderLine == 0 {
		t.Fatalf("expected external inputs table: %+v", model.Sections.ExternalInputs)
	}
	if !model.Sections.Outputs.Table || model.Sections.Outputs.HeaderLine == 0 {
		t.Fatalf("expected outputs table: %+v", model.Sections.Outputs)
	}
	if len(model.Sections.Adoption.Commands) != 1 {
		t.Fatalf("expected adoption command: %+v", model.Sections.Adoption)
	}
	if len(model.Sections.FeedbackLoop.Steps) != 1 || strings.Join(model.Sections.FeedbackLoop.Steps[0].References, ",") != "topic:first/*" {
		t.Fatalf("expected feedback step reference: %+v", model.Sections.FeedbackLoop)
	}
	if len(model.Sections.Gaps.Items) != 1 || strings.Join(model.Sections.Gaps.Items[0].References, ",") != "topic[future]:second/*" || !model.Sections.Gaps.Items[0].TargetState {
		t.Fatalf("expected structured gap item: %+v", model.Sections.Gaps)
	}
}

func TestValidateOperatingModelsRejectsMissingRequiredSection(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.ExternalInputs = OperatingExternalInputsTable{}

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_required_section_missing")
	assertOperatingFindingDetail(t, result, "operating_model_required_section_missing", "External Inputs / Triggers")
}

func TestValidateOperatingModelsRejectsDuplicateRequiredSections(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Mission.Duplicates = []int{3}

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_duplicate_section")
	assertOperatingFindingDetail(t, result, "operating_model_duplicate_section", "Mission")
}

func TestValidateOperatingModelsRejectsNonCanonicalSectionTables(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Decisions.Headers = []string{"decision context", "owner", "purpose"}
	models[0].Sections.ExternalInputs.Headers = []string{"input", "enters through", "first handler", "notes"}
	models[0].Sections.Outputs.Headers = []string{"output", "downstream consumer", "path"}

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_decisions_header_drift")
	assertOperatingFinding(t, result, "operating_model_external_inputs_header_drift")
	assertOperatingFinding(t, result, "operating_model_outputs_header_drift")
}

func TestValidateOperatingModelsRejectsIncompleteDecisionMetadata(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Decisions.Rows[0].ExpectedEvidenceTrigger = ""

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_decisions_row_incomplete")
	assertOperatingFindingDetail(t, result, "operating_model_decisions_row_incomplete", "expected evidence / trigger")
}

func TestValidateOperatingModelsRejectsWeakDecisionAcceptedEffect(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Decisions.Rows[0].AcceptedEffect = "Operator approves it."

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_decisions_effect_weak")
}

func TestValidateOperatingModelsRejectsUnanchoredFeedbackStep(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.FeedbackLoop.Steps[0].References = nil

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_feedback_step_unanchored")
}

func TestValidateOperatingModelsRejectsUnbackedFeedbackReference(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.FeedbackLoop.Steps[0].References = []string{"ghost-surface/*"}

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_feedback_reference_unbacked")
}

func TestValidateOperatingModelsRejectsUnstructuredGapItems(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Gaps.Items = nil

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_gaps_items_missing")
}

func TestValidateOperatingModelsRejectsUnanchoredGapItem(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Gaps.Items[0].References = nil

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_gap_item_unanchored")
}

func TestValidateOperatingModelsRejectsGapItemWithoutTargetState(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Gaps.Items[0].TargetState = false

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_gap_item_target_state_missing")
}

func TestValidateOperatingModelsRejectsMissingAdoptionCommands(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Adoption.Commands = models[0].Sections.Adoption.Commands[:1]

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_adoption_command_missing")
	assertOperatingFindingDetail(t, result, "operating_model_adoption_command_missing", "diff")
}

func TestOperatingModelGoldenFixtureValidatesAndCoversAllSections(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	runtime := operatingModelDiscoverabilityRuntime(t, model, true, true)

	result := ValidateOperatingModels([]OperatingModelDocument{model}, runtime, "team-a", "g")
	if result.Errors != 0 || result.Warnings != 0 {
		t.Fatalf("golden operating-model fixture should validate cleanly: %+v", result.Findings)
	}

	coverage := BuildOperatingModelCoverage([]OperatingModelDocument{model}, runtime, "team-a", "g")
	if len(coverage) != 1 {
		t.Fatalf("coverage length=%d, want 1: %+v", len(coverage), coverage)
	}
	docs := coverage[0].Docs
	if docs.RequiredSectionsPresent != docs.RequiredSectionsTotal || docs.RequiredSectionsTotal == 0 {
		t.Fatalf("golden fixture should cover every required section: %+v", docs)
	}
	if docs.TopicCatalogRows != 1 || docs.TopicCatalogMatched != 1 || docs.TopicCatalogGraphOnly != 0 || docs.TopicCatalogDocsOnly != 0 {
		t.Fatalf("golden fixture topic coverage drifted: %+v", docs)
	}
	if docs.DecisionsRows != 1 || docs.DecisionsMatched != 1 || docs.DecisionsGraphOnly != 0 || docs.DecisionsDocsOnly != 0 {
		t.Fatalf("golden fixture decision coverage drifted: %+v", docs)
	}
	if docs.ExternalInputsRows != 1 || docs.ExternalInputsBackedRows != 1 || docs.ExternalInputsUnbackedRows != 0 {
		t.Fatalf("golden fixture external input coverage drifted: %+v", docs)
	}
	if docs.OutputsRows != 1 || docs.OutputsBackedRows != 1 || docs.OutputsUnbackedRows != 0 {
		t.Fatalf("golden fixture output coverage drifted: %+v", docs)
	}
	if docs.FeedbackSteps != 1 || docs.FeedbackAnchoredSteps != 1 || docs.FeedbackUnbackedReferences != 0 {
		t.Fatalf("golden fixture feedback coverage drifted: %+v", docs)
	}
	if docs.GapsItems != 1 || docs.GapsAnchoredItems != 1 || docs.GapsTargetStateItems != 1 {
		t.Fatalf("golden fixture gap coverage drifted: %+v", docs)
	}
	if docs.AdoptionValidationCommands != 3 ||
		docs.PlanOfRecordRegistration != OperatingCoverageStatusEnforced ||
		docs.ReadmeDiscoverability != OperatingCoverageStatusEnforced {
		t.Fatalf("golden fixture discoverability coverage drifted: %+v", docs)
	}
}

func TestOperatingModelRegisteredRulesHaveFailureFixtures(t *testing.T) {
	cases := []struct {
		rule    string
		mutate  func(*OperatingModelDocument)
		runtime func(t *testing.T, model OperatingModelDocument) OperatingGraphRuntime
	}{
		{rule: "operating_model_required_section_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs = OperatingExternalInputsTable{}
		}},
		{rule: "operating_model_duplicate_section", mutate: func(model *OperatingModelDocument) {
			model.Sections.Mission.Duplicates = []int{3}
		}},
		{rule: "operating_model_decisions_header_drift", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Headers = []string{"decision context", "owner", "purpose"}
		}},
		{rule: "operating_model_decisions_empty", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Rows = nil
		}},
		{rule: "operating_model_decisions_row_incomplete", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Rows[0].ExpectedEvidenceTrigger = ""
		}},
		{rule: "operating_model_decisions_effect_weak", mutate: func(model *OperatingModelDocument) {
			model.Sections.Decisions.Rows[0].AcceptedEffect = "Operator approves it."
		}},
		{rule: "operating_model_external_inputs_table_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Table = false
		}},
		{rule: "operating_model_external_inputs_header_drift", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Headers = []string{"input", "enters through", "first handler", "notes"}
		}},
		{rule: "operating_model_external_inputs_empty", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows = nil
		}},
		{rule: "operating_model_external_inputs_row_incomplete", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].RoutingRule = ""
		}},
		{rule: "operating_model_external_inputs_producer_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].ProducerTrigger = "`external:ghost-system`"
		}},
		{rule: "operating_model_external_inputs_entry_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].EntrySurface = "`topic:ghost-inbox/*`"
		}},
		{rule: "operating_model_external_inputs_drainer_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.ExternalInputs.Rows[0].Drainer = "member:ghost"
		}},
		{rule: "operating_model_outputs_table_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Table = false
		}},
		{rule: "operating_model_outputs_header_drift", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Headers = []string{"output", "downstream consumer", "path"}
		}},
		{rule: "operating_model_outputs_empty", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows = nil
		}},
		{rule: "operating_model_outputs_row_incomplete", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows[0].Purpose = ""
		}},
		{rule: "operating_model_outputs_surface_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows[0].Surface = "`topic:ghost-output/*`"
		}},
		{rule: "operating_model_outputs_consumer_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.Outputs.Rows[0].Consumer = "ghost"
		}},
		{rule: "operating_model_feedback_steps_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.FeedbackLoop.Steps = nil
		}},
		{rule: "operating_model_feedback_step_unanchored", mutate: func(model *OperatingModelDocument) {
			model.Sections.FeedbackLoop.Steps[0].References = nil
		}},
		{rule: "operating_model_feedback_reference_unbacked", mutate: func(model *OperatingModelDocument) {
			model.Sections.FeedbackLoop.Steps[0].References = []string{"ghost-surface/*"}
		}},
		{rule: "operating_model_gaps_items_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Gaps.Items = nil
		}},
		{rule: "operating_model_gap_item_unanchored", mutate: func(model *OperatingModelDocument) {
			model.Sections.Gaps.Items[0].References = nil
		}},
		{rule: "operating_model_gap_item_target_state_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Gaps.Items[0].TargetState = false
		}},
		{rule: "operating_model_adoption_command_missing", mutate: func(model *OperatingModelDocument) {
			model.Sections.Adoption.Commands = model.Sections.Adoption.Commands[:1]
		}},
		{rule: "operating_model_plan_of_record_missing", runtime: func(t *testing.T, model OperatingModelDocument) OperatingGraphRuntime {
			return operatingModelDiscoverabilityRuntime(t, model, false, true)
		}},
		{rule: "operating_model_readme_link_missing", runtime: func(t *testing.T, model OperatingModelDocument) OperatingGraphRuntime {
			return operatingModelDiscoverabilityRuntime(t, model, true, false)
		}},
	}

	fixtured := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			model := operatingModelDocumentFixture(t)
			if tc.mutate != nil {
				tc.mutate(&model)
			}
			runtime := OperatingGraphRuntime{}
			if tc.runtime != nil {
				runtime = tc.runtime(t, model)
			}
			result := ValidateOperatingModels([]OperatingModelDocument{model}, runtime, "team-a", "g")
			assertOperatingFinding(t, result, tc.rule)
			fixtured[tc.rule] = true
		})
	}

	for _, rule := range DefaultOperatingModelRules() {
		if !fixtured[rule.ID()] {
			t.Fatalf("registered operating-model rule %q has no deliberate failure fixture", rule.ID())
		}
	}
}

func TestValidateOperatingModelsChecksPlanOfRecordManifest(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writeFile("docs/team-a/manifest.json", `{
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "team": "team-a"},
  "sections": [
    {
      "id": "entrypoint",
      "path": ".",
      "documents": [{
        "path": "README.md",
        "required": true,
        "validation": {"requiredHeadings": ["Start here for agents", "Folder map"]}
      }]
    },
    {
      "id": "operating",
      "path": "operating/",
      "required": true,
      "documents": [{"path": "OPERATING_MODEL.md", "required": true}]
    },
    {
      "id": "taxonomies",
      "path": "taxonomies/",
      "packages": [{"id": "signal", "path": "signal/", "requiredFiles": ["README.md", "taxonomy.json"]}]
    }
  ]
}`)
	writeFile("docs/team-a/README.md", "# Team A\n\n## Start here for agents\n")
	writeFile("docs/team-a/operating/OPERATING_MODEL.md", "# Operating\n")
	writeFile("docs/team-a/taxonomies/signal/README.md", "# Signal\n")

	model := operatingModelDocumentFixture(t)
	model.Team = "team-a"
	model.Source.Path = "docs/team-a/operating/OPERATING_MODEL.md"
	result := ValidateOperatingModels([]OperatingModelDocument{model}, OperatingGraphRuntime{RepoRoot: repoRoot}, "team-a", "g")

	assertOperatingFinding(t, result, "por_required_heading_missing")
	assertOperatingFinding(t, result, "por_package_required_file_missing")
}

func TestValidateOperatingModelsRejectsPlanOfRecordHardCutoverDrift(t *testing.T) {
	repoRoot := t.TempDir()
	writeFile := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writeFile("docs/team-a/manifest.json", `{
  "contract": {"kind": "team-plan-of-record", "schema": "team-plan-of-record/v1", "team": "team-a"},
  "sections": [
    {"id": "entrypoint", "path": ".", "required": true, "documents": [{"path": "README.md", "required": true}]},
    {"id": "operating", "path": "operating/", "required": true, "documents": [{"path": "OPERATING_MODEL.md", "required": true}]}
  ]
}`)
	writeFile("docs/team-a/README.md", "# Team A\n\nThis current file points to docs/marketing/notebook.\n")
	writeFile("docs/team-a/operating/OPERATING_MODEL.md", "# Operating\n")
	writeFile("docs/team-a/notebook/NOTE.md", "# Note\n")

	model := operatingModelDocumentFixture(t)
	model.Team = "team-a"
	model.Source.Path = "docs/team-a/operating/OPERATING_MODEL.md"
	result := ValidateOperatingModels([]OperatingModelDocument{model}, OperatingGraphRuntime{RepoRoot: repoRoot}, "team-a", "g")

	assertOperatingFinding(t, result, "por_hard_cutover_drift")
	assertOperatingFinding(t, result, "por_notebook_surface")
}

func TestLoadAllTaxonomiesDiscoversPackagedTaxonomyJSON(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, "docs", "team-a", "taxonomies", "signal", "taxonomy.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir taxonomy dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"id":"team-a-signal","signalTypes":[{"id":"bug"}]}`), 0o644); err != nil {
		t.Fatalf("write taxonomy: %v", err)
	}

	registry, err := LoadAllTaxonomies(repoRoot)
	if err != nil {
		t.Fatalf("LoadAllTaxonomies: %v", err)
	}
	if _, ok := registry["team-a-signal"]; !ok {
		t.Fatalf("expected packaged taxonomy to load, got ids=%v", registry.IDs())
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
	path := filepath.Join("..", "..", "..", "..", "docs", "marketing", "operating", "OPERATING_MODEL.md")
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/marketing/operating/OPERATING_MODEL.md")
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

func TestMarketingOperatingModelCentralizesTypedObservationDrainage(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "docs", "marketing", "operating", "OPERATING_MODEL.md")
	blocks, err := ExtractOperatingGraphBlocks(path, "docs/marketing/operating/OPERATING_MODEL.md")
	if err != nil {
		t.Fatalf("ExtractOperatingGraphBlocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	rels := NewOperatingRelationshipSet(BuildGraphOperatingRelationships(blocks[0]))
	wantBrandManagerRead := OperatingRelationship{Kind: operatingRelTopicRead, Team: "marketing-crew", Member: "brand-manager", Topic: "marketing-craft-observation/*"}
	if !operatingRelationshipSetContains(rels, wantBrandManagerRead) {
		t.Fatalf("marketing craft observation must drain through brand-manager; relationships=%+v", rels.All())
	}
	for _, member := range []string{"researcher", "oss-advertiser", "subscription-advertiser", "publisher"} {
		forbidden := OperatingRelationship{Kind: operatingRelTopicRead, Team: "marketing-crew", Member: member, Topic: "marketing-craft-observation/*"}
		if operatingRelationshipSetContains(rels, forbidden) {
			t.Fatalf("raw marketing craft observation should not be a direct runtime read for %s", member)
		}
	}
}

func TestValidateOperatingModelsKeepsMarketingScenarioQAMetaGreen(t *testing.T) {
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
	models, err := LoadOperatingModelDocuments(repoRoot)
	if err != nil {
		t.Fatalf("LoadOperatingModelDocuments: %v", err)
	}
	runtime, err := BuildOperatingGraphRuntime(repoRoot, storeDir)
	if err != nil {
		t.Fatalf("BuildOperatingGraphRuntime: %v", err)
	}

	cases := []struct {
		team string
		id   string
	}{
		{team: "marketing-crew", id: "marketing-operating-model"},
		{team: "scenario-qa", id: "scenario-qa-operating-model"},
		{team: "meta-optimization", id: "meta-optimization-operating-model"},
	}
	for _, tc := range cases {
		t.Run(tc.team, func(t *testing.T) {
			result := ValidateOperatingModels(models, runtime, tc.team, tc.id)
			if result.Errors != 0 || result.Warnings != 0 {
				t.Fatalf("unexpected validation findings for %s/%s: %+v", tc.team, tc.id, result.Findings)
			}

			diffs := DiffOperatingGraphs(blocks, runtime, tc.team, tc.id)
			if len(diffs) != 0 {
				t.Fatalf("unexpected diff for %s/%s: %+v", tc.team, tc.id, diffs)
			}

			coverage := BuildOperatingModelCoverage(models, runtime, tc.team, tc.id)
			if len(coverage) != 1 {
				t.Fatalf("coverage length=%d, want 1: %+v", len(coverage), coverage)
			}
			docCoverage := coverage[0].Docs
			if docCoverage.TopicCatalogRows != docCoverage.TopicCatalogMatched || docCoverage.TopicCatalogGraphOnly != 0 || docCoverage.TopicCatalogDocsOnly != 0 {
				t.Fatalf("unexpected topic catalog coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.DecisionsRows != docCoverage.DecisionsMatched || docCoverage.DecisionsGraphOnly != 0 || docCoverage.DecisionsDocsOnly != 0 {
				t.Fatalf("unexpected decision coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.DecisionsMetadataComplete != docCoverage.DecisionsRows || docCoverage.DecisionsMetadataIncomplete != 0 || docCoverage.DecisionsAcceptedEffectWeak != 0 {
				t.Fatalf("unexpected decision metadata coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.RequiredSectionsPresent != docCoverage.RequiredSectionsTotal || docCoverage.RequiredSectionsTotal == 0 {
				t.Fatalf("unexpected required-section coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.ExternalInputsTable != OperatingCoverageStatusEnforced ||
				docCoverage.ExternalInputsRows == 0 ||
				docCoverage.ExternalInputsBackedRows != docCoverage.ExternalInputsRows ||
				docCoverage.ExternalInputsUnbackedRows != 0 {
				t.Fatalf("unexpected external-input coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.OutputsTable != OperatingCoverageStatusEnforced ||
				docCoverage.OutputsRows == 0 ||
				docCoverage.OutputsBackedRows != docCoverage.OutputsRows ||
				docCoverage.OutputsUnbackedRows != 0 {
				t.Fatalf("unexpected outputs coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.FeedbackSteps == 0 || docCoverage.FeedbackAnchoredSteps != docCoverage.FeedbackSteps || docCoverage.FeedbackUnbackedReferences != 0 {
				t.Fatalf("unexpected feedback coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.GapsItems == 0 || docCoverage.GapsAnchoredItems != docCoverage.GapsItems || docCoverage.GapsTargetStateItems != docCoverage.GapsItems {
				t.Fatalf("unexpected gaps coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			if docCoverage.AdoptionValidationCommands != 3 || docCoverage.PlanOfRecordRegistration != OperatingCoverageStatusEnforced || docCoverage.ReadmeDiscoverability != OperatingCoverageStatusEnforced {
				t.Fatalf("unexpected adoption/discoverability coverage for %s/%s: %+v", tc.team, tc.id, docCoverage)
			}
			for _, rel := range coverage[0].Relationships {
				if rel.GraphOnly != 0 || rel.RuntimeOnly != 0 {
					t.Fatalf("unexpected relationship coverage drift for %s/%s relationship %s: %+v", tc.team, tc.id, rel.Relationship, rel)
				}
			}
		})
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

func assertOperatingFindingAbsent(t *testing.T, result OperatingGraphValidationResult, rule string) {
	t.Helper()
	for _, f := range result.Findings {
		if f.Rule == rule {
			t.Fatalf("unexpected finding %q in %+v", rule, result.Findings)
		}
	}
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

func operatingCoverageExclusionCount(exclusions []OperatingCoverageExclusion, kind string) int {
	for _, exclusion := range exclusions {
		if exclusion.Kind == kind {
			return exclusion.Count
		}
	}
	return 0
}

func operatingSubtypeCoverageByRelationship(rels []OperatingRelationshipSubtypeCoverage, relationship string) OperatingRelationshipSubtypeCoverage {
	for _, rel := range rels {
		if rel.Relationship == relationship {
			return rel
		}
	}
	return OperatingRelationshipSubtypeCoverage{}
}

func TestExtractOperatingModelDocumentsRejectsDuplicateContractGraphs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "OPERATING_MODEL.md")
	if err := os.WriteFile(path, []byte(`<!-- prompt-manager-graph:
id: g1
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
  T1["topic:first/*"]
  T1 --> A
`+"```"+`
## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:first/*`"+` | live | member:a | member:a | First. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|

<!-- prompt-manager-graph:
id: g2
scope: team
team: team-a
mode: contract
-->
`+"```mermaid"+`
flowchart LR
  A["member:a"]
  T2["topic:second/*"]
  T2 --> A
`+"```"+`
## Topic Catalog

| Topic family | Status | Owner / primary writer | Primary readers | Purpose |
|---|---|---|---|---|
| `+"`topic:second/*`"+` | live | member:a | member:a | Second. |

## Decisions

| Decision context | Owner | Purpose | Expected evidence / trigger | Accepted effect |
|---|---|---|---|---|
`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, err := ExtractOperatingModelDocuments(path, "docs/x/OPERATING_MODEL.md")
	if err == nil || !strings.Contains(err.Error(), "one contract graph") {
		t.Fatalf("expected duplicate contract graph rejection, got %v", err)
	}
}

func operatingDiffBlock(t *testing.T, lines []string) OperatingGraphBlock {
	t.Helper()
	return OperatingGraphBlock{
		Metadata: OperatingGraphMetadata{ID: "g", Scope: "team", Team: "team-a", Mode: "contract"},
		Graph:    mustParseGraph(t, lines),
		Source:   OperatingGraphSource{Path: "docs/test/OPERATING_MODEL.md", Line: 1, FenceLine: 2},
	}
}

func operatingModelDocumentFixture(t *testing.T) OperatingModelDocument {
	t.Helper()
	block := operatingDiffBlock(t, []string{
		"flowchart LR",
		`  OP(["external:operator"])`,
		`  A["member:a"]`,
		`  T[("topic:first/*")]`,
		`  D{"decision:model-update"}`,
		"  OP --> A",
		"  A --> T",
		"  T --> A",
		"  A --> D",
		"  D --> A",
	})
	block.Docs = OperatingGraphDocs{
		TopicCatalog: OperatingTopicCatalogTable{
			Present:    true,
			HeaderLine: 10,
			Rows: []OperatingTopicCatalogRow{{
				Topic:      "first/*",
				Status:     "live",
				StatusKind: OperatingTopicStatusLive,
				Writers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "a", Raw: "member:a"}},
				Readers:    []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "a", Raw: "member:a"}},
				Purpose:    "First.",
				SourceLine: 12,
				RawTopic:   "`topic:first/*`",
			}},
		},
		Decisions: OperatingDecisionTable{
			Present:    true,
			HeaderLine: 14,
			Headers:    []string{"decision context", "owner", "purpose", "expected evidence / trigger", "accepted effect"},
			Rows: []OperatingDecisionRow{{
				Decision:                "model-update",
				Owners:                  []OperatingActorReference{{Kind: OperatingActorKindMember, Value: "a", Raw: "member:a"}},
				Purpose:                 "Update the model contract.",
				ExpectedEvidenceTrigger: "Evidence from `topic:first/*`.",
				AcceptedEffect:          "Operator-approved operating-model document update.",
				SourceLine:              16,
				RawDecision:             "`model-update`",
			}},
		},
	}
	return OperatingModelDocument{
		ID:     "g",
		Team:   "team-a",
		Source: OperatingModelSource{Path: "docs/test/OPERATING_MODEL.md", Line: 1},
		Sections: OperatingModelSections{
			Mission:        OperatingMarkdownSection{Heading: "Mission", Present: true, Line: 1},
			Scope:          OperatingMarkdownSection{Heading: "Scope", Present: true, Line: 2},
			OperatingLoops: OperatingMarkdownSection{Heading: "Operating Loops", Present: true, Line: 3},
			Graph:          OperatingGraphSection{OperatingGraphBlock: block, Heading: "Operating Graph", Present: true},
			TopicCatalog:   block.Docs.TopicCatalog,
			Decisions:      block.Docs.Decisions,
			ExternalInputs: OperatingExternalInputsTable{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "External Inputs / Triggers", Present: true, Line: 16},
				HeaderLine:               17,
				Headers:                  []string{"producer / trigger", "entry surface", "drainer", "routing rule"},
				Table:                    true,
				Rows: []OperatingExternalInputRow{{
					ProducerTrigger: "Operator",
					EntrySurface:    "direct member context",
					Drainer:         "member:a",
					RoutingRule:     "Route directly.",
					SourceLine:      19,
				}},
			},
			Outputs: OperatingOutputsTable{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Outputs / Downstream Consumers", Present: true, Line: 20},
				HeaderLine:               21,
				Headers:                  []string{"output", "surface", "consumer", "purpose"},
				Table:                    true,
				Rows: []OperatingOutputRow{{
					Output:     "Output",
					Surface:    "`topic:first/*`",
					Consumer:   "member:a",
					Purpose:    "Read it.",
					SourceLine: 23,
				}},
			},
			FeedbackLoop: OperatingFeedbackSection{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Feedback / Capability Improvement Loop", Present: true, Line: 18},
				Steps: []OperatingFeedbackStep{{
					Text:       "Review `topic:first/*`.",
					References: []string{"topic:first/*"},
					SourceLine: 19,
				}},
			},
			Gaps: OperatingGapsSection{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Current Implementation Gaps", Present: true, Line: 19},
				Items: []OperatingGapItem{{
					Text:        "`topic[future]:second/*` remains target-state until a producer exists.",
					References:  []string{"topic[future]:second/*"},
					TargetState: true,
					SourceLine:  20,
				}},
			},
			Adoption: OperatingAdoptionSection{
				OperatingMarkdownSection: OperatingMarkdownSection{Heading: "Adoption / Validation", Present: true, Line: 24},
				Commands: []OperatingAdoptionCommand{
					{Command: "prompt-manager graph operating-model validate --team team-a --id g", SourceLine: 25},
					{Command: "prompt-manager graph operating-model diff --team team-a --id g", SourceLine: 26},
					{Command: "prompt-manager graph operating-model coverage --team team-a --id g", SourceLine: 27},
				},
			},
		},
		Graphs: []OperatingGraphBlock{block},
	}
}

func operatingModelDiscoverabilityRuntime(t *testing.T, model OperatingModelDocument, includePlanOfRecord, includeReadmeLink bool) OperatingGraphRuntime {
	t.Helper()
	repoRoot := t.TempDir()
	readmePath := operatingModelTeamReadmePath(model.Source.Path)
	if readmePath != "" && includeReadmeLink {
		absReadmePath := filepath.Join(repoRoot, filepath.FromSlash(readmePath))
		if err := os.MkdirAll(filepath.Dir(absReadmePath), 0o755); err != nil {
			t.Fatalf("create README dir: %v", err)
		}
		if err := os.WriteFile(absReadmePath, []byte("See OPERATING_MODEL.md.\n"), 0o644); err != nil {
			t.Fatalf("write README fixture: %v", err)
		}
	}
	loaded := &LoadedTeamContract{TeamID: model.Team}
	members := []MemberTopics{{
		Ref: MemberRef{Team: model.Team, Member: "a"},
		Topics: Topics{
			Intake:            []IntakeEntry{{Prefix: "first/*"}},
			Output:            []OutputEntry{{Prefix: "first/*", DestinationKind: DestinationKnowledge}},
			DecisionsOwned:    []string{"model-update"},
			DecisionsConsumed: []string{"model-update"},
			ExternalProducers: []string{"operator"},
		},
		Exists: true,
	}}
	loaded.Contract = &teamcontract.OperatingContract{
		DecisionContext: map[string]teamcontract.DecisionContext{"model-update": {OwnerMemberIDs: []string{"a"}}},
		Members:         map[string]teamcontract.MemberContract{"a": {}},
	}
	loaded.TopicCatalog = []TopicCatalogEntry{{
		Prefix:  "first/*",
		Status:  "live",
		Purpose: "First.",
	}}
	if includePlanOfRecord {
		loaded.PlanOfRecordDocuments = []teamcontract.PlanOfRecordDocument{{
			ID: "operating-model",
			Paths: []teamcontract.PathRef{{
				Base: "repo-root",
				Path: model.Source.Path,
			}},
		}}
	}
	return OperatingGraphRuntime{
		RepoRoot:       repoRoot,
		Members:        members,
		PromptSections: derivedTopicContractPromptSections(members, TeamContractRegistry{model.Team: loaded}),
		Contracts: TeamContractRegistry{
			model.Team: loaded,
		},
	}
}

func TestValidateOperatingModelsRejectsUnbackedExternalInputs(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.ExternalInputs.Rows[0].ProducerTrigger = "`external:ghost-system`"
	models[0].Sections.ExternalInputs.Rows[0].EntrySurface = "`topic:ghost-inbox/*`"
	models[0].Sections.ExternalInputs.Rows[0].Drainer = "member:ghost"

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_external_inputs_producer_unbacked")
	assertOperatingFinding(t, result, "operating_model_external_inputs_entry_unbacked")
	assertOperatingFinding(t, result, "operating_model_external_inputs_drainer_unbacked")
}

func TestValidateOperatingModelsRejectsUnbackedOutputs(t *testing.T) {
	models := []OperatingModelDocument{operatingModelDocumentFixture(t)}
	models[0].Sections.Outputs.Rows[0].Surface = "`topic:ghost-output/*`"
	models[0].Sections.Outputs.Rows[0].Consumer = "member:ghost"

	result := ValidateOperatingModels(models, OperatingGraphRuntime{}, "team-a", "g")
	assertOperatingFinding(t, result, "operating_model_outputs_surface_unbacked")
	assertOperatingFinding(t, result, "operating_model_outputs_consumer_unbacked")
}

func TestOperatingModelReferenceIndexNormalizesDocumentReferences(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	model.Sections.Outputs.Rows[0].Surface = "`topic:first/*`"
	index := NewOperatingModelReferenceIndex(model, OperatingGraphRuntime{})

	assertOperatingModelReference(t, index, OperatingModelReferenceKindTopic, "", "first/*", "topic_catalog")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindDecision, "", "model-update", "decisions")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindMember, "", "a", "topic_catalog")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindTopic, OperatingGraphQualifierFuture, "second/*", "gaps")
	assertOperatingModelReference(t, index, OperatingModelReferenceKindCommand, "", "prompt-manager graph operating-model validate --team team-a --id g", "adoption")

	inputAssurance := index.ExternalInputAssurance(model.Sections.ExternalInputs.Rows[0])
	if !inputAssurance.Backed() {
		t.Fatalf("expected fixture external input row to be fully backed: %+v", inputAssurance)
	}
	outputAssurance := index.OutputAssurance(model.Sections.Outputs.Rows[0])
	if !outputAssurance.Backed() {
		t.Fatalf("expected fixture output row to be fully backed: %+v", outputAssurance)
	}
	if !index.FeedbackReferenceAssurance("topic:first/*").Backed {
		t.Fatalf("expected feedback topic reference to be backed")
	}
}

func TestOperatingModelReferenceIndexOwnsCoverageAssurance(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	model.Sections.ExternalInputs.Rows[0].ProducerTrigger = "`external:ghost-system`"
	model.Sections.Outputs.Rows[0].Surface = "`topic:ghost-output/*`"
	model.Sections.FeedbackLoop.Steps[0].References = []string{"topic:first/*", "topic:never-seen/*"}
	index := NewOperatingModelReferenceIndex(model, OperatingGraphRuntime{})

	inputBacked, inputUnbacked := externalInputsCoverageCounts(index)
	if inputBacked != 0 || inputUnbacked != 1 {
		t.Fatalf("expected coverage to count reference-index external input assurance, got backed=%d unbacked=%d", inputBacked, inputUnbacked)
	}
	outputBacked, outputUnbacked := outputsCoverageCounts(index)
	if outputBacked != 0 || outputUnbacked != 1 {
		t.Fatalf("expected coverage to count reference-index output assurance, got backed=%d unbacked=%d", outputBacked, outputUnbacked)
	}
	steps, anchored, unbackedRefs := feedbackLoopCoverageCounts(index)
	if steps != 1 || anchored != 1 || unbackedRefs != 1 {
		t.Fatalf("expected coverage to count reference-index feedback assurance, got steps=%d anchored=%d unbacked=%d", steps, anchored, unbackedRefs)
	}
}

func TestOperatingModelReferenceIndexExpandsRuntimeActorGroups(t *testing.T) {
	model := operatingModelDocumentFixture(t)
	model.Graphs[0].Metadata.Extra = map[string]string{
		"actor_alias.available-drainers": "group:available-drainers",
		"actor_group.available-drainers": "team-members",
	}
	model.Sections.ExternalInputs.Rows[0].Drainer = "available-drainers"

	withoutRuntime := NewOperatingModelReferenceIndex(model, OperatingGraphRuntime{}).ExternalInputAssurance(model.Sections.ExternalInputs.Rows[0])
	if withoutRuntime.Drainer {
		t.Fatalf("expected unresolved team-members group to leave drainer unbacked: %+v", withoutRuntime)
	}

	runtime := OperatingGraphRuntime{
		Contracts: TeamContractRegistry{
			"team-a": {TeamID: "team-a", Contract: &teamcontract.OperatingContract{
				Members: map[string]teamcontract.MemberContract{"a": {}},
			}},
		},
	}
	withRuntime := NewOperatingModelReferenceIndex(model, runtime).ExternalInputAssurance(model.Sections.ExternalInputs.Rows[0])
	if !withRuntime.Drainer {
		t.Fatalf("expected runtime-expanded team-members group to back drainer: %+v", withRuntime)
	}
}

func assertOperatingModelReference(t *testing.T, index OperatingModelReferenceIndex, kind OperatingModelReferenceKind, qualifier OperatingGraphQualifier, value, surface string) {
	t.Helper()
	for _, ref := range index.References {
		if ref.Kind == kind && ref.Qualifier == qualifier && ref.Value == value && ref.Surface == surface {
			return
		}
	}
	t.Fatalf("missing normalized reference kind=%s qualifier=%s value=%q surface=%q in %+v", kind, qualifier, value, surface, index.References)
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
