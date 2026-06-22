package report

import (
	"strings"
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestBuildMaturityListReportIncludesLocalProgressAndImpact(t *testing.T) {
	report := BuildMaturityListReport(&commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "measures-health",
		Phase:    "measures",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel:         "L1",
			NextLevel:            "L2",
			BlockingFindingCodes: []string{"measures.uncovered-domain"},
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L1"},
				{Id: "L2"},
			},
		},
		Findings: []*commonv1.AssessmentFinding{
			{
				Code:     "measures.uncovered-domain",
				Severity: "ERROR",
				Title:    "Stateful domain is uncovered",
				Location: "cli/manifest.json",
				Maturity: &commonv1.FindingMaturity{
					LocalLevel:   "L2",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP,
					Dimension:    "measures",
				},
			},
		},
		FindingsByGlobalImpact: map[string]int32{"capability_gap": 1},
		RecommendedSkillIds:    []string{"measures-adoption"},
	})

	if !containsLine(report.Summary, "Local maturity: current=L1 · 1 blocking · 0 debt") {
		t.Fatalf("summary missing local progress: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "Next level: L2") {
		t.Fatalf("summary missing next level: %#v", report.Summary)
	}
	if len(report.Results) != 2 || report.Results[0] != "L2 findings (1)" || !strings.Contains(report.Results[1], "impact=capability_gap") {
		t.Fatalf("result missing maturity impact: %#v", report.Results)
	}
	if !containsLine(report.RetrievalHints, "capability_gap: 1") {
		t.Fatalf("hints missing impact count: %#v", report.RetrievalHints)
	}
	if !containsLine(report.RetrievalHints, "skill: measures-adoption") {
		t.Fatalf("hints missing skill: %#v", report.RetrievalHints)
	}
}

func TestBuildMaturityListReportSurfacesDebtWhenComplete(t *testing.T) {
	report := BuildMaturityListReport(&commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "proto-health",
		Phase:    "proto",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: "L5",
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L1"},
				{Id: "L2"},
				{Id: "L3"},
				{Id: "L4"},
				{Id: "L5"},
			},
		},
		Findings: []*commonv1.AssessmentFinding{
			{
				Code:     "proto.annotation.unsupported",
				Severity: "WARNING",
				Title:    "Unsupported annotation",
				Maturity: &commonv1.FindingMaturity{
					LocalLevel:   "L4",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP,
					Dimension:    "proto-health",
				},
			},
			{
				Code:     "proto.message.possibly_unused",
				Severity: "INFO",
				Title:    "Message may be unused",
				Maturity: &commonv1.FindingMaturity{
					LocalLevel:   "L5",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_ADVISORY,
					Dimension:    "proto-health",
				},
			},
		},
	})

	if !containsLine(report.Summary, "Local maturity: current=L5 · 0 blocking · 2 debt") {
		t.Fatalf("summary missing debt count: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "Debt by level: L4=1 (warning:1), L5=1 (info:1)") {
		t.Fatalf("summary missing debt breakdown: %#v", report.Summary)
	}
	wantResults := []string{
		"L4 findings (1)",
		"L5 findings (1)",
	}
	for _, want := range wantResults {
		if !containsLine(report.Results, want) {
			t.Fatalf("results missing %q: %#v", want, report.Results)
		}
	}
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}
