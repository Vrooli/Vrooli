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
		Diff: []operatingGraphDiff{{Kind: "declared_output_missing", Detail: "member output missing"}},
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
	if !strings.Contains(stdout, "[declared_output_missing] member output missing") {
		t.Fatalf("stdout missing diff:\n%s", stdout)
	}
}
