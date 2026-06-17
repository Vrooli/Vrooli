package cliapp

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

	if !containsLine(report.Summary, "Local maturity: current=L1 next=L2") {
		t.Fatalf("summary missing local progress: %#v", report.Summary)
	}
	if len(report.Results) != 1 || !strings.Contains(report.Results[0], "impact=capability_gap") {
		t.Fatalf("result missing maturity impact: %#v", report.Results)
	}
	if !containsLine(report.RetrievalHints, "capability_gap: 1") {
		t.Fatalf("hints missing impact count: %#v", report.RetrievalHints)
	}
	if !containsLine(report.RetrievalHints, "skill: measures-adoption") {
		t.Fatalf("hints missing skill: %#v", report.RetrievalHints)
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
