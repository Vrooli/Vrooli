package graph

import (
	"strings"
	"testing"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// The split exists so one command can gate and the other can report. These two
// tests pin both halves: a declaration error must fail `graph topics`, and a
// runtime error must not.
func TestTopicsGatesOnDeclarationFindingsOnly(t *testing.T) {
	for _, tc := range []struct {
		name    string
		finding topicFinding
		wantErr bool
	}{
		{
			name:    "declaration error gates",
			finding: topicFinding{Rule: "orphan_input", Severity: "error", Kind: kindDeclaration, Detail: "no producer"},
			wantErr: true,
		},
		{
			name:    "runtime error does not gate",
			finding: topicFinding{Rule: "actual_writer_undeclared", Severity: "error", Kind: kindRuntime, Detail: "undeclared write"},
			wantErr: false,
		},
		{
			name:    "declaration warning does not gate",
			finding: topicFinding{Rule: "orphan_output", Severity: "warning", Kind: kindDeclaration, Detail: "no consumer"},
			wantErr: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := clitest.NewContext(t)
			ctx.Respond("GET", "/topics/graph", topicsGraphResponse{
				Validation: topicValidation{Findings: []topicFinding{tc.finding}, Errors: 1},
			})
			_, _, err := clitest.Output(t, func() error { return cmdTopics(ctx, nil) })
			if tc.wantErr && err == nil {
				t.Error("expected a non-zero exit")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected exit 0, got %v", err)
			}
		})
	}
}

func TestRuntimeReportsRuntimeFindingsAndAlwaysExitsZero(t *testing.T) {
	ctx := clitest.NewContext(t)
	ctx.Respond("GET", "/topics/graph", topicsGraphResponse{
		Validation: topicValidation{
			Findings: []topicFinding{
				{Rule: "actual_writer_undeclared", Severity: "error", Kind: kindRuntime, Team: "t", Member: "m", Prefix: "x/*", Detail: "undeclared write"},
				{Rule: "actual_writer_undeclared", Severity: "error", Kind: kindRuntime, Team: "t", Member: "n", Prefix: "y/*", Detail: "undeclared write"},
				{Rule: "orphan_input", Severity: "error", Kind: kindDeclaration, Detail: "no producer"},
			},
			Errors: 3,
		},
	})

	stdout, _, err := clitest.Output(t, func() error { return cmdRuntime(ctx, nil) })
	if err != nil {
		t.Fatalf("graph runtime must always exit 0, got %v", err)
	}
	if !strings.Contains(stdout, "actual_writer_undeclared") {
		t.Errorf("runtime report omits the runtime rule:\n%s", stdout)
	}
	// A declaration finding belongs to the gate, not the report.
	if strings.Contains(stdout, "orphan_input") {
		t.Errorf("runtime report leaked a declaration finding:\n%s", stdout)
	}
	if !strings.Contains(stdout, "2 runtime finding") {
		t.Errorf("runtime report does not count its findings:\n%s", stdout)
	}
}

// An uncatalogued finding has no kind. It must reach neither surface rather
// than defaulting into the gate, where it could fail a build with no
// description, actuator, or documentation behind it.
func TestUncataloguedFindingJoinsNeitherSurface(t *testing.T) {
	ctx := clitest.NewContext(t)
	resp := topicsGraphResponse{
		Validation: topicValidation{
			Findings: []topicFinding{{Rule: "not_catalogued", Severity: "error", Detail: "orphaned"}},
			Errors:   1,
		},
	}
	ctx.Respond("GET", "/topics/graph", resp)
	if _, _, err := clitest.Output(t, func() error { return cmdTopics(ctx, nil) }); err != nil {
		t.Errorf("an uncatalogued finding must not gate: %v", err)
	}

	ctx2 := clitest.NewContext(t)
	ctx2.Respond("GET", "/topics/graph", resp)
	stdout, _, err := clitest.Output(t, func() error { return cmdRuntime(ctx2, nil) })
	if err != nil {
		t.Fatalf("graph runtime must always exit 0: %v", err)
	}
	if strings.Contains(stdout, "not_catalogued") {
		t.Errorf("runtime report claimed an uncatalogued finding:\n%s", stdout)
	}
}
