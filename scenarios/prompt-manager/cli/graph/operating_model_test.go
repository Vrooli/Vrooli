package graph

import (
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
