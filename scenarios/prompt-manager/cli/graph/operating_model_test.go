package graph

import (
	"encoding/json"
	"strings"
	"testing"

	clitest "prompt-manager/cli/internal/testutil"
)

func TestCmdOperatingModelValidatePassesFiltersAndFailsOnErrors(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/validate", operatingGraphValidationResponse{
		Graphs: []operatingGraphBlock{{}},
		Validation: operatingGraphValidation{
			Findings: []operatingGraphFinding{{
				Rule:       "graph_edge_unbacked",
				Severity:   "error",
				SourcePath: "docs/marketing/OPERATING_MODEL.md",
				Line:       12,
				Detail:     "edge is not backed",
			}},
			Errors: 1,
		},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelValidate(ctx, []string{"--team", "marketing-crew", "--id", "marketing-operating-model"})
	})
	if err == nil {
		t.Fatal("expected validation errors to return a non-nil error")
	}
	req := ctx.LastRequest()
	if req.Path != "/operating-graphs/validate" || req.Query.Get("team") != "marketing-crew" || req.Query.Get("id") != "marketing-operating-model" {
		t.Fatalf("unexpected request: %+v", req)
	}
	for _, want := range []string{
		"Status",
		"1 error(s)",
		"graph_edge_unbacked",
		"edge is not backed",
		"Fix error findings",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCmdOperatingModelListJSONPreservesAPIShape(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs", operatingGraphListResponse{
		Graphs: []operatingGraphBlock{{
			Metadata: operatingGraphMetadata{
				ID:     "g",
				Scope:  "team",
				Team:   "team-a",
				Mode:   "contract",
				Status: "draft",
				Extra:  map[string]string{"owner": "ops"},
			},
			Graph: operatingGraph{
				ID:        "flowchart",
				Direction: "LR",
				Nodes: []operatingGraphNode{{
					ID:         "M",
					Kind:       "member",
					Value:      "member-a",
					Display:    "Member A",
					RawLabel:   "Member A",
					SourceLine: 12,
					Implicit:   true,
				}},
				Edges: []operatingGraphEdge{{
					From:       "M",
					To:         "T",
					Label:      "sends",
					SourceLine: 13,
				}},
			},
			Source: operatingGraphSource{
				Path:      "docs/test.md",
				Line:      10,
				FenceLine: 11,
			},
		}},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelList(ctx, []string{"--team", "team-a", "--json"})
	})
	if err != nil {
		t.Fatalf("cmdOperatingModelList: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout)
	}
	graph := got["graphs"].([]any)[0].(map[string]any)
	metadata := graph["metadata"].(map[string]any)
	source := graph["source"].(map[string]any)
	if metadata["status"] != "draft" || metadata["extra"].(map[string]any)["owner"] != "ops" || source["fence_line"].(float64) != 11 {
		t.Fatalf("json output lost API fields:\n%s", stdout)
	}
}

func TestCmdOperatingModelValidateJSONPreservesFindingShape(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/validate", operatingGraphValidationResponse{
		Validation: operatingGraphValidation{
			Findings: []operatingGraphFinding{{
				Rule:       "graph_prompt_topic_contract_missing",
				Severity:   "error",
				GraphID:    "g",
				Team:       "team-a",
				NodeID:     "M",
				Member:     "member-a",
				SourcePath: "docs/test.md",
				Line:       12,
				Detail:     "missing prompt section",
			}},
			Errors: 1,
		},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelValidate(ctx, []string{"--json"})
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	for _, want := range []string{
		`"graph_id": "g"`,
		`"team": "team-a"`,
		`"node_id": "M"`,
		`"member": "member-a"`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCmdOperatingModelDiffJSONPreservesFullDiffShape(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/diff", operatingGraphDiffResponse{
		Diff: []operatingGraphDiff{{
			Kind:             "runtime_relationship_missing_in_graph",
			Relationship:     "external_producer",
			Team:             "team-a",
			Member:           "member-a",
			External:         "operator",
			RuntimePath:      "teams/team-a/members/member-a/topics.json",
			AcceptableFields: []string{"external_producers"},
			Suggestions:      []string{"add external:operator -> member:member-a"},
			Detail:           "missing",
		}},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelDiff(ctx, []string{"--json"})
	})
	if err != nil {
		t.Fatalf("cmdOperatingModelDiff: %v", err)
	}
	for _, want := range []string{
		`"runtime_path": "teams/team-a/members/member-a/topics.json"`,
		`"acceptable_fields": [`,
		`"suggestions": [`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCmdOperatingModelDiffRendersHumanOutput(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/diff", operatingGraphDiffResponse{
		Diff: []operatingGraphDiff{{
			Kind:             "graph_relationship_missing_in_runtime",
			Relationship:     "topic_read",
			Team:             "marketing-crew",
			Member:           "researcher",
			Topic:            "marketing/notebook/*",
			SourcePath:       "docs/marketing/OPERATING_MODEL.md",
			Line:             355,
			RuntimePath:      "scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/topics.json",
			AcceptableFields: []string{"intake", "required_read", "evidence_consumed"},
			Suggestions:      []string{"add required_read \"marketing/notebook/*\" to researcher/topics.json"},
			Detail:           "docs/marketing/OPERATING_MODEL.md:355 says topic:marketing/notebook/* -> member:researcher. Runtime has no matching declaration.",
		}},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelDiff(ctx, []string{"--team", "marketing-crew"})
	})
	if err != nil {
		t.Fatalf("cmdOperatingModelDiff: %v", err)
	}
	req := ctx.LastRequest()
	if req.Path != "/operating-graphs/diff" || req.Query.Get("team") != "marketing-crew" {
		t.Fatalf("unexpected request: %+v", req)
	}
	for _, want := range []string{
		"Graph Declares, Runtime Missing",
		"[topic_read] docs/marketing/OPERATING_MODEL.md:355",
		"Runtime file: scenarios/prompt-manager/store/teams/marketing-crew/members/researcher/topics.json",
		"Acceptable runtime fields: intake, required_read, evidence_consumed",
		"Suggested fix: add required_read \"marketing/notebook/*\" to researcher/topics.json",
		"Runtime Declares, Graph Missing",
		"- clean",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCmdOperatingModelDiffRendersCleanNextStep(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/diff", operatingGraphDiffResponse{
		Diff: []operatingGraphDiff{},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelDiff(ctx, []string{"--team", "marketing-crew"})
	})
	if err != nil {
		t.Fatalf("cmdOperatingModelDiff: %v", err)
	}
	for _, want := range []string{
		"Found 0 diff item(s).",
		"Graph Declares, Runtime Missing\n- clean",
		"Runtime Declares, Graph Missing\n- clean",
		"No reconciliation required.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCmdOperatingModelCoveragePassesFiltersAndRendersHumanOutput(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/coverage", operatingGraphCoverageResponse{
		Coverage: []operatingGraphCoverage{{
			GraphID: "marketing-operating-model",
			Team:    "marketing-crew",
			Source:  operatingGraphSource{Path: "docs/marketing/OPERATING_MODEL.md", Line: 42},
			Relationships: []operatingRelationshipCoverage{{
				Relationship:       "topic_read",
				RuntimeDeclared:    45,
				GraphShown:         45,
				Matched:            45,
				ValidationSeverity: "error",
				RuntimeSubtypes: []operatingRelationshipSubtypeCoverage{{
					Relationship:    "topic_required_read",
					RuntimeDeclared: 45,
					Covered:         45,
				}},
			}},
			Prompts: operatingPromptCoverage{
				GraphMembers:               6,
				TopicContractPresent:       6,
				TopicContractSourceMatched: 6,
				TopicContractContentParity: "not_implemented",
			},
			Docs: operatingDocsCoverage{
				MermaidGraph:      "enforced",
				TopicCatalogTable: "reference_only",
				DecisionsTable:    "reference_only",
			},
			Exclusions: []operatingCoverageExclusion{{
				Kind:   "process_nodes",
				Count:  4,
				Detail: "process nodes explain workflow shape but do not map to runtime declarations",
			}},
		}},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelCoverage(ctx, []string{"--team", "marketing-crew", "--id", "marketing-operating-model"})
	})
	if err != nil {
		t.Fatalf("cmdOperatingModelCoverage: %v", err)
	}
	req := ctx.LastRequest()
	if req.Path != "/operating-graphs/coverage" || req.Query.Get("team") != "marketing-crew" || req.Query.Get("id") != "marketing-operating-model" {
		t.Fatalf("unexpected request: %+v", req)
	}
	for _, want := range []string{
		"Analyzed 1 operating graph(s).",
		"Graph: marketing-operating-model team=marketing-crew source=docs/marketing/OPERATING_MODEL.md:42",
		"Relationship Coverage",
		"- topic_read: runtime declared 45, graph shown 45, matched 45, graph-only 0, runtime-only 0 (error)",
		"  - topic_required_read: runtime declared 45, covered 45, runtime-only 0",
		"Prompt Coverage",
		"topic-contract section present: 6/6 graph members",
		"content parity: not_implemented",
		"Docs Coverage",
		"Topic Catalog table: reference_only",
		"Excluded",
		"process_nodes: 4",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCmdOperatingModelCoverageJSONPreservesAPIShape(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/operating-graphs/coverage", operatingGraphCoverageResponse{
		Coverage: []operatingGraphCoverage{{
			GraphID: "g",
			Team:    "team-a",
			Relationships: []operatingRelationshipCoverage{{
				Relationship:    "topic_output",
				RuntimeDeclared: 2,
				GraphShown:      1,
				Matched:         1,
				RuntimeOnly:     1,
				RuntimeSubtypes: []operatingRelationshipSubtypeCoverage{{
					Relationship:    "topic_output",
					RuntimeDeclared: 2,
					Covered:         1,
					RuntimeOnly:     1,
				}},
				ValidationRule:     "graph_declared_output_missing",
				ValidationSeverity: "warning",
				DiffRelationship:   "topic_output",
			}},
			Prompts: operatingPromptCoverage{
				GraphMembers:               1,
				TopicContractContentParity: "not_implemented",
			},
			Docs: operatingDocsCoverage{
				MermaidGraph:      "enforced",
				TopicCatalogTable: "reference_only",
				DecisionsTable:    "reference_only",
			},
		}},
	})

	stdout, _, err := clitest.Output(t, func() error {
		return cmdOperatingModelCoverage(ctx, []string{"--json"})
	})
	if err != nil {
		t.Fatalf("cmdOperatingModelCoverage: %v", err)
	}
	for _, want := range []string{
		`"graph_id": "g"`,
		`"relationship": "topic_output"`,
		`"runtime_only": 1`,
		`"runtime_subtypes": [`,
		`"validation_rule": "graph_declared_output_missing"`,
		`"topic_contract_content_parity": "not_implemented"`,
		`"topic_catalog_table": "reference_only"`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}
