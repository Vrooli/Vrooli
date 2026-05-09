package hygienecli

import (
	"bytes"
	"strings"
	"testing"

	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	"github.com/vrooli/vrooli/internal/cliout"
)

func TestParseRequestSupportsDetailAndScopeFlags(t *testing.T) {
	req, err := ParseRequest([]string{"--details", "--plans-only", "--fail-on", "warning"})
	if err != nil {
		t.Fatalf("ParseRequest: %v", err)
	}
	if req.OutputMode != OutputModeDetails {
		t.Fatalf("OutputMode = %q, want %q", req.OutputMode, OutputModeDetails)
	}
	if !req.PlansOnly {
		t.Fatalf("PlansOnly = false, want true")
	}
	if req.FailOn != hygieneapp.SeverityWarning {
		t.Fatalf("FailOn = %q, want warning", req.FailOn)
	}
}

func TestParseRequestRejectsConflictingModes(t *testing.T) {
	if _, err := ParseRequest([]string{"--summary", "--details"}); err == nil {
		t.Fatalf("ParseRequest accepted conflicting output modes")
	}
	if _, err := ParseRequest([]string{"--plans-only", "--contract-only"}); err == nil {
		t.Fatalf("ParseRequest accepted conflicting scope modes")
	}
}

func TestRenderDefaultShowsFindingsPlanSummaryAndNextSteps(t *testing.T) {
	report := hygieneapp.Report{
		Success:          false,
		Root:             "/repo",
		BlockingFailures: 1,
		Warnings:         1,
		Findings: []hygieneapp.Finding{
			{
				Severity:   hygieneapp.SeverityError,
				Code:       "repo_contract_personal_absolute_paths",
				Locations:  []string{"internal/setup/example_test.go:12"},
				Message:    "personal absolute paths found",
				Why:        "Committed personal home paths make tests machine-specific.",
				Fixability: hygieneapp.FixabilityManual,
				NextActions: []hygieneapp.Action{{
					Code:    "remove_personal_paths",
					Message: "Replace personal absolute paths.",
				}},
			},
			{
				Severity:   hygieneapp.SeverityWarning,
				Code:       "plan_candidates",
				Message:    "12 likely scratch plan candidates found",
				Fixability: hygieneapp.FixabilityAutomatic,
				NextActions: []hygieneapp.Action{{
					Code:    "import_plan_candidates",
					Message: "Import untracked scratch plan candidates into user-scoped plan storage.",
					Command: "vrooli hygiene --fix-safe --plans",
				}},
			},
		},
		PlanCandidates: []hygieneapp.PlanCandidate{
			{Path: "docs/plans/a.md", Status: "untracked"},
			{Path: "docs/plans/b.md", Status: "modified"},
		},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeDefault); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	for _, want := range []string{
		"Blocking issues:",
		"repo_contract_personal_absolute_paths [manual]",
		"Locations: internal/setup/example_test.go:12",
		"Why: Committed personal home paths make tests machine-specific.",
		"Warnings:",
		"plan_candidates [automatic]",
		"Plan candidates: 2",
		"- 1 modified",
		"- 1 untracked",
		"Modified plan candidates:",
		"Next steps:",
		"vrooli hygiene --fix-safe --plans",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("rendered output missing %q:\n%s", want, text)
		}
	}
}

func TestRenderNextOnlySuppressesStatus(t *testing.T) {
	report := hygieneapp.Report{
		Root: "/repo",
		Findings: []hygieneapp.Finding{{
			Severity: hygieneapp.SeverityWarning,
			Code:     "plan_candidates",
			Message:  "plans found",
			NextActions: []hygieneapp.Action{{
				Code:    "import_plan_candidates",
				Command: "vrooli hygiene --fix-safe --plans",
			}},
		}},
	}
	var out bytes.Buffer
	if err := Render(&out, cliout.FormatHuman, report, OutputModeNext); err != nil {
		t.Fatalf("Render: %v", err)
	}
	text := out.String()
	if strings.Contains(text, "Status:") {
		t.Fatalf("--next output included status:\n%s", text)
	}
	if !strings.Contains(text, "vrooli hygiene --fix-safe --plans") {
		t.Fatalf("--next output missing next command:\n%s", text)
	}
}
