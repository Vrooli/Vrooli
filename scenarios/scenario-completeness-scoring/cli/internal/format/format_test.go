package format

import (
	"bytes"
	"strings"
	"testing"
)

func sampleResponse() ScoreResponse {
	return ScoreResponse{
		Scenario:          "test-genie",
		Category:          "utility",
		Score:             44,
		BaseScore:         59,
		ValidationPenalty: 15,
		Classification:    "functional_incomplete",
		Breakdown: ScoreBreakdown{
			BaseScore:      59,
			Classification: "functional_incomplete",
			Quality: QualityScore{
				Score: 27, Max: 50,
				RequirementPassRate: PassRate{Passing: 1, Total: 3, Rate: 0.333, Points: 7},
				TargetPassRate:      PassRate{Passing: 1, Total: 3, Rate: 0.333, Points: 5},
				TestPassRate:        PassRate{Passing: 3, Total: 3, Rate: 1.0, Points: 15},
			},
			Coverage: CoverageScore{
				Score: 6, Max: 15,
				TestCoverageRatio: CoverageRatio{Ratio: 1.0, Points: 4},
				DepthScore:        DepthScoreDetail{AvgDepth: 1.0, Points: 2},
			},
			Quantity: QuantityScore{
				Score: 2, Max: 10,
				Requirements: QuantityMetric{Count: 3, Threshold: "below", Points: 1},
				Targets:      QuantityMetric{Count: 3, Threshold: "below", Points: 1},
				Tests:        QuantityMetric{Count: 3, Threshold: "below", Points: 0},
			},
			UI: UIScore{
				Score: 24, Max: 25,
				TemplateCheck:       TemplateCheckResult{IsTemplate: false, Points: 10},
				ComponentComplexity: ComponentComplexity{FileCount: 85, Threshold: "excellent", Points: 5},
				APIIntegration:      APIIntegration{EndpointCount: 23, Points: 6},
				Routing:             RoutingScore{HasRouting: false, RouteCount: 0, Points: 0},
				CodeVolume:          CodeVolume{TotalLOC: 12914, Points: 2.5},
			},
		},
		ValidationAnalysis: ValidationQualityAnalysis{
			HasIssues:       true,
			IssueCount:      2,
			TotalPenalty:    15,
			OverallSeverity: "high",
			Issues: []ValidationIssue{
				{
					Type:           "ungrouped_operational_targets",
					Severity:       "high",
					Penalty:        10,
					Message:        "100% of operational targets have 1:1 requirement mapping",
					Recommendation: "Group related requirements under shared operational targets",
					WhyItMatters:   "Operational targets should encompass multiple requirements.",
				},
				{
					Type:     "insufficient_test_coverage",
					Severity: "medium",
					Penalty:  5,
					Message:  "Suspicious 1:1 test-to-requirement ratio",
				},
			},
		},
	}
}

func TestFormatScoreSummaryContainsBanner(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	var buf bytes.Buffer
	FormatScoreSummary(&buf, sampleResponse())
	out := buf.String()

	for _, want := range []string{
		"📊 COMPLETENESS SCORE: 44/100",
		"functional incomplete",
		"Final Score:        44/100",
		"Base Score:         59/100",
		"Validation Penalty: -15pts",
		"Penalty breakdown:",
		"ungrouped operational targets: -10 pts",
		"insufficient test coverage: -5 pts",
		"Status: Functional but incomplete, needs more features/tests",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatScoreSummary missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatBaseMetricsIncludesAllSections(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	var buf bytes.Buffer
	FormatBaseMetrics(&buf, sampleResponse().Breakdown)
	out := buf.String()

	for _, want := range []string{
		"Quality Metrics (27/50):",
		"Requirements: 3 total, 1 passing (33%)",
		"[Target: 90%+]",
		"Tests: 3 total, 3 passing (100%)",
		"Coverage Metrics (6/15):",
		"Test Coverage: 1.0x",
		"Depth Score: 1.0 avg levels",
		"Quantity Metrics (2/10):",
		"Requirements: 3 (Below)",
		"UI Metrics (24/25):",
		"Template: Custom",
		"Files: 85 files (Excellent)",
		"API Integration: 23 endpoints",
		"Routing: 0 routes",
		"LOC: 12914 total",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatBaseMetrics missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatValidationIssuesHighAndMedium(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	var buf bytes.Buffer
	FormatValidationIssues(&buf, sampleResponse().ValidationAnalysis, false)
	out := buf.String()

	for _, want := range []string{
		"📋 UNDERSTANDING THIS REPORT",
		"VALIDATION ISSUES DETECTED",
		"Top Issues (Fix These First):",
		"🔴 100% of operational targets have 1:1 requirement mapping",
		"Penalty: -10 pts",
		"Why this matters:",
		"Next Steps:",
		"🟡 Minor Issues (1 issues, -5pts total)",
		"🟡 Suspicious 1:1 test-to-requirement ratio",
		"Run with --verbose",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatValidationIssues missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatValidationIssuesNoIssues(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	var buf bytes.Buffer
	FormatValidationIssues(&buf, ValidationQualityAnalysis{HasIssues: false}, false)
	out := buf.String()

	if !strings.Contains(out, "No Validation Issues Detected") {
		t.Errorf("expected clean-report banner, got:\n%s", out)
	}
	if strings.Contains(out, "VALIDATION ISSUES DETECTED") {
		t.Errorf("should not print issues banner when has_issues=false:\n%s", out)
	}
}

func TestFormatActionPlanWithKnownIssueTypes(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	resp := sampleResponse()
	resp.ValidationAnalysis.Issues = []ValidationIssue{
		{Type: "invalid_test_location", Severity: "high", Penalty: 10, Count: 5},
		{Type: "insufficient_validation_layers", Severity: "high", Penalty: 8, Count: 2, Total: 6},
		{Type: "monolithic_test_files", Severity: "medium", Penalty: 4, Violations: 3},
	}

	var buf bytes.Buffer
	FormatActionPlan(&buf, resp)
	out := buf.String()

	for _, want := range []string{
		"🎯 RECOMMENDED ACTION PLAN",
		"Phase 1: Fix Test Locations (+10pts estimated)",
		"Phase 2: Add Multi-Layer Validation (+8pts estimated)",
		"Phase 3: Create Focused Tests (+4pts estimated)",
		"Estimated Score After Fixes: ~59/100",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("FormatActionPlan missing %q in:\n%s", want, out)
		}
	}
}

func TestFormatActionPlanNoIssues(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	var buf bytes.Buffer
	FormatActionPlan(&buf, ScoreResponse{ValidationAnalysis: ValidationQualityAnalysis{HasIssues: false}})
	out := buf.String()

	if !strings.Contains(out, "No priority actions needed") {
		t.Errorf("expected clean action plan, got:\n%s", out)
	}
}

func TestFormatComparisonContextBranches(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	tests := []struct {
		name    string
		penalty int
		score   float64
		want    string
	}{
		{"high penalty", 60, 30, "🎓 Study browser-automation-studio"},
		{"excellent", 0, 90, "🌟 Excellent work!"},
		{"good", 15, 60, "✨ This scenario has good test structure"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			FormatComparisonContext(&buf, ValidationQualityAnalysis{TotalPenalty: tc.penalty}, tc.score)
			out := buf.String()
			if !strings.Contains(out, tc.want) {
				t.Errorf("missing %q in:\n%s", tc.want, out)
			}
		})
	}
}

func TestSetColorEnabledStripsAnsi(t *testing.T) {
	SetColorEnabled(true)
	if !ColorEnabled() {
		t.Fatal("expected colors enabled")
	}
	var withColor bytes.Buffer
	FormatScoreSummary(&withColor, sampleResponse())
	if !strings.Contains(withColor.String(), "\033[") {
		t.Error("expected ANSI escapes with color enabled")
	}

	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })
	if ColorEnabled() {
		t.Fatal("expected colors disabled")
	}
	var noColor bytes.Buffer
	FormatScoreSummary(&noColor, sampleResponse())
	if strings.Contains(noColor.String(), "\033[") {
		t.Errorf("did not expect ANSI escapes when color disabled:\n%s", noColor.String())
	}
}

func TestVerboseIssueRendersOptionalFields(t *testing.T) {
	SetColorEnabled(false)
	t.Cleanup(func() { SetColorEnabled(true) })

	issue := ValidationIssue{
		Type:        "invalid_test_location",
		Severity:    "high",
		Penalty:     10,
		Message:     "Requirements reference CLI wrapper tests",
		Description: "Background explaining the gaming pattern.",
		ValidSources: []string{
			"api/**/*_test.go",
			"ui/src/**/*.test.tsx",
		},
		InvalidPaths: []InvalidPathInfo{
			{Path: "cli/foo.go", RequirementIDs: []string{"REQ-1", "REQ-2"}},
		},
		AffectedReqs: []AffectedRequirement{
			{ID: "REQ-1", Title: "Example", CurrentLayers: []string{"cli"}, NeededLayers: []string{"api", "ui"}},
		},
		WorstOffender: &MonolithicTestInfo{TestRef: "api/big_test.go", Count: 8},
		Violations:    3,
	}

	var buf bytes.Buffer
	formatIssueDetail(&buf, issue, true)
	out := buf.String()

	for _, want := range []string{
		"🔴 Requirements reference CLI wrapper tests",
		"Invalid paths found:",
		"cli/foo.go (referenced by 2 requirements)",
		"Valid test locations:",
		"api/**/*_test.go",
		"Affected requirements (first 5):",
		"REQ-1 (Example)",
		"has: cli → needs: api + ui",
		"Affected files:",
		"api/big_test.go (validates 8 requirements)",
		"... and 2 more test files",
		"Background:",
		"Background explaining the gaming pattern.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("formatIssueDetail missing %q in:\n%s", want, out)
		}
	}
}
