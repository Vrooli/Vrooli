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

func TestBuildMaturityListReportIncludesCapabilitySummariesAndFocus(t *testing.T) {
	report := BuildMaturityListReport(&commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "ui-health",
		Phase:    "ui-health",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel:         "L0",
			NextLevel:            "L1",
			BlockingFindingCodes: []string{"pwa.service_worker_missing"},
			Levels: []*commonv1.LocalMaturityLevel{
				{Id: "L0"},
				{Id: "L1"},
			},
		},
		Capabilities: []*commonv1.CapabilityMaturityAssessment{
			{
				Id:             "interop",
				Label:          "Interop",
				CurrentLevel:   "L2",
				CurrentSummary: "Embeds correctly across deployment contexts.",
				Clean:          true,
				PriorityRank:   1,
				Levels: []*commonv1.LocalMaturityLevel{
					{Id: "L0", StatusLabel: "Unavailable"},
					{Id: "L1", StatusLabel: "Foundation"},
					{Id: "L2", StatusLabel: "Complete"},
				},
			},
			{
				Id:                   "pwa_native_readiness",
				Label:                "PWA Native Readiness",
				CurrentLevel:         "L0",
				NextLevel:            "L1",
				CurrentSummary:       "Install surface is absent.",
				NextUnlock:           "Offline-safe launch and app-shell fallback.",
				BlockingFindingCodes: []string{"pwa.service_worker_missing"},
				PriorityRank:         2,
				Levels: []*commonv1.LocalMaturityLevel{
					{Id: "L0", StatusLabel: "Unavailable"},
					{Id: "L1", StatusLabel: "Basic"},
				},
			},
		},
		HighestPriorityCapability: &commonv1.PriorityFocus{
			CapabilityId:    "pwa_native_readiness",
			CapabilityLabel: "PWA Native Readiness",
			CurrentLevel:    "L0",
			NextLevel:       "L1",
			Reason:          "lowest current capability level",
		},
		Findings: []*commonv1.AssessmentFinding{
			{
				Code:     "pwa.service_worker_missing",
				Severity: "ERROR",
				Title:    "Service worker is missing",
				Maturity: &commonv1.FindingMaturity{
					CapabilityId: "pwa_native_readiness",
					LocalLevel:   "L1",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP,
					Dimension:    "ui",
				},
			},
		},
	})

	if !containsLine(report.Summary, "Highest priority: PWA Native Readiness to L1 - lowest current capability level") {
		t.Fatalf("summary missing priority focus: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "Interop: rung=L2 Complete · blocking=0 · debt=0") {
		t.Fatalf("summary missing complete interop capability: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "  Maximum maturity reached.") {
		t.Fatalf("summary missing maximum maturity line: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "PWA Native Readiness: rung=L0 Unavailable · blocking=1 · debt=0") {
		t.Fatalf("summary missing pwa capability: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "  Next L1 unlocks: Offline-safe launch and app-shell fallback.") {
		t.Fatalf("summary missing next unlock: %#v", report.Summary)
	}
	if len(report.Results) != 2 || report.Results[0] != "PWA Native Readiness / L1 findings (1)" || !strings.Contains(report.Results[1], "capability=pwa_native_readiness") {
		t.Fatalf("results missing capability grouping: %#v", report.Results)
	}
}

func TestBuildMaturityListReportCountsCapabilityDebt(t *testing.T) {
	report := BuildMaturityListReport(&commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "ui-health",
		Phase:    "ui-health",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: "L1",
			Levels:       []*commonv1.LocalMaturityLevel{{Id: "L0"}, {Id: "L1"}},
		},
		Capabilities: []*commonv1.CapabilityMaturityAssessment{
			{
				Id:           "project_standards",
				Label:        "Project Standards",
				CurrentLevel: "L1",
				Levels:       []*commonv1.LocalMaturityLevel{{Id: "L0"}, {Id: "L1", StatusLabel: "Foundation"}},
				PriorityRank: 1,
			},
		},
		Findings: []*commonv1.AssessmentFinding{
			{
				Code:     "standard.a11y_harness_missing",
				Severity: "WARNING",
				Title:    "Accessibility harness missing",
				Maturity: &commonv1.FindingMaturity{
					CapabilityId: "project_standards",
					LocalLevel:   "L1",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_HARDENING_GAP,
					Dimension:    "ui",
				},
			},
		},
	})

	if !containsLine(report.Summary, "Project Standards: rung=L1 Foundation with debt · blocking=0 · debt=1") {
		t.Fatalf("summary missing capability debt: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "  Debt: 1 debt finding.") {
		t.Fatalf("summary missing separate capability debt line: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "Debt by level: L1=1 (warning:1)") {
		t.Fatalf("summary missing legacy debt rollup: %#v", report.Summary)
	}
}

func TestBuildMaturityListReportDoesNotCallDebtCleanMaximum(t *testing.T) {
	report := BuildMaturityListReport(&commonv1.MaturityAssessment{
		Scenario: "demo",
		Provider: "ui-health",
		Phase:    "ui-health",
		Local: &commonv1.LocalMaturityAssessment{
			CurrentLevel: "L5",
			Levels:       []*commonv1.LocalMaturityLevel{{Id: "L5"}},
		},
		Capabilities: []*commonv1.CapabilityMaturityAssessment{
			{
				Id:             "pwa_native_readiness",
				Label:          "PWA Native Readiness",
				CurrentLevel:   "L5",
				CurrentSummary: "Installability, launch, offline, and declared platform capabilities are fully clean.",
				Levels: []*commonv1.LocalMaturityLevel{
					{Id: "L5", StatusLabel: "Complete"},
				},
				PriorityRank: 1,
			},
		},
		Findings: []*commonv1.AssessmentFinding{
			{
				Code:     "pwa_service_worker_offline",
				Severity: "WARNING",
				Title:    "Service worker is missing",
				Maturity: &commonv1.FindingMaturity{
					CapabilityId: "pwa_native_readiness",
					LocalLevel:   "L5",
					GlobalImpact: commonv1.GlobalImpact_GLOBAL_IMPACT_CAPABILITY_GAP,
					Dimension:    "ui",
				},
			},
		},
	})

	if !containsLine(report.Summary, "PWA Native Readiness: rung=L5 Complete with debt · blocking=0 · debt=1") {
		t.Fatalf("summary missing debt-qualified max rung: %#v", report.Summary)
	}
	if containsLine(report.Summary, "  Installability, launch, offline, and declared platform capabilities are fully clean.") {
		t.Fatalf("summary should suppress clean current summary while debt remains: %#v", report.Summary)
	}
	if !containsLine(report.Summary, "  Top rung reached, but 1 advisory debt item remains.") {
		t.Fatalf("summary should not claim clean maximum maturity: %#v", report.Summary)
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
