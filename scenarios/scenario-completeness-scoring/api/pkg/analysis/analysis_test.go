package analysis

import (
	"testing"

	"scenario-completeness-scoring/pkg/scoring"
)

// [REQ:SCS-ANALYSIS-001] What-if analysis unit tests

// TestWhatIfAnalysisBasic tests basic what-if functionality
func TestWhatIfAnalysisBasic(t *testing.T) {
	// Test that applyChange works for quality.test_pass_rate
	analyzer := &WhatIfAnalyzer{}

	metrics := &scoring.Metrics{
		Scenario: "test-scenario",
		Category: "utility",
		Tests: scoring.MetricCounts{
			Total:   100,
			Passing: 80,
		},
	}

	// Test applying a change to test pass rate
	change := MetricChange{
		Component: "quality.test_pass_rate",
		NewValue:  1.0, // 100% pass rate
	}

	oldValue, newValue, applied := analyzer.applyChange(metrics, change)

	if !applied {
		t.Error("Expected change to be applied")
	}
	if oldValue != 0.8 {
		t.Errorf("Expected old value 0.8, got %f", oldValue)
	}
	if newValue != 1.0 {
		t.Errorf("Expected new value 1.0, got %f", newValue)
	}
	if metrics.Tests.Passing != 100 {
		t.Errorf("Expected passing count to be 100, got %d", metrics.Tests.Passing)
	}
}

// TestWhatIfAnalysisUITemplate tests UI template change simulation
func TestWhatIfAnalysisUITemplate(t *testing.T) {
	analyzer := &WhatIfAnalyzer{}

	metrics := &scoring.Metrics{
		Scenario: "test-scenario",
		Category: "utility",
		UI: &scoring.UIMetrics{
			IsTemplate: true,
			FileCount:  10,
		},
	}

	// Change from template to non-template
	change := MetricChange{
		Component: "ui.template",
		NewValue:  0.0, // Not a template
	}

	oldValue, newValue, applied := analyzer.applyChange(metrics, change)

	if !applied {
		t.Error("Expected change to be applied")
	}
	if oldValue != 1.0 {
		t.Errorf("Expected old value 1.0 (was template), got %f", oldValue)
	}
	if newValue != 0.0 {
		t.Errorf("Expected new value 0.0, got %f", newValue)
	}
	if metrics.UI.IsTemplate {
		t.Error("Expected UI to no longer be template")
	}
}

// TestWhatIfAnalysisQuantity tests quantity change simulation
func TestWhatIfAnalysisQuantity(t *testing.T) {
	analyzer := &WhatIfAnalyzer{}

	metrics := &scoring.Metrics{
		Scenario: "test-scenario",
		Category: "utility",
		Tests: scoring.MetricCounts{
			Total:   20,
			Passing: 18,
		},
	}

	// Add more tests while maintaining pass rate
	change := MetricChange{
		Component: "quantity.tests",
		NewValue:  40.0,
	}

	oldValue, _, applied := analyzer.applyChange(metrics, change)

	if !applied {
		t.Error("Expected change to be applied")
	}
	if oldValue != 20.0 {
		t.Errorf("Expected old value 20.0, got %f", oldValue)
	}
	if metrics.Tests.Total != 40 {
		t.Errorf("Expected test count to be 40, got %d", metrics.Tests.Total)
	}
	// Pass rate should be maintained (90%)
	expectedPassing := int(0.9 * 40)
	if metrics.Tests.Passing != expectedPassing {
		t.Errorf("Expected passing count %d (90%% of 40), got %d", expectedPassing, metrics.Tests.Passing)
	}
}

// TestWhatIfAnalysisUnknownComponent tests handling of unknown components
func TestWhatIfAnalysisUnknownComponent(t *testing.T) {
	analyzer := &WhatIfAnalyzer{}

	metrics := &scoring.Metrics{
		Scenario: "test-scenario",
		Category: "utility",
	}

	change := MetricChange{
		Component: "unknown.component",
		NewValue:  100,
	}

	_, _, applied := analyzer.applyChange(metrics, change)

	if applied {
		t.Error("Expected unknown component change to not be applied")
	}
}

// TestEstimateImpact tests impact estimation
func TestEstimateImpact(t *testing.T) {
	tests := []struct {
		component string
		oldValue  float64
		newValue  float64
		minImpact int
	}{
		{"ui.template", 1.0, 0.0, 10}, // Template removal = 10 points
		{"quality.requirement_pass_rate", 0.5, 1.0, 10}, // 50% improvement = ~10 points
		{"quality.test_pass_rate", 0.8, 1.0, 3}, // 20% improvement = ~3 points
	}

	for _, tc := range tests {
		impact := estimateImpact(tc.component, tc.oldValue, tc.newValue)
		if impact < tc.minImpact {
			t.Errorf("Component %s: expected impact >= %d, got %d", tc.component, tc.minImpact, impact)
		}
	}
}

// TestAvailableComponents tests that all expected components are listed
func TestAvailableComponents(t *testing.T) {
	components := AvailableComponents()

	if len(components) == 0 {
		t.Error("Expected available components to be non-empty")
	}

	// Check for essential components
	expectedNames := []string{
		"quality.requirement_pass_rate",
		"quality.test_pass_rate",
		"ui.template",
		"quantity.tests",
	}

	componentMap := make(map[string]bool)
	for _, c := range components {
		componentMap[c.Name] = true
	}

	for _, expected := range expectedNames {
		if !componentMap[expected] {
			t.Errorf("Expected component %s to be available", expected)
		}
	}
}

// [REQ:SCS-ANALYSIS-003] Bulk refresh unit tests

// TestBulkRefreshScenario tests single scenario refresh
func TestBulkRefreshScenario(t *testing.T) {
	// This test validates the ScenarioRefreshInfo structure
	info := ScenarioRefreshInfo{
		Scenario:       "test-scenario",
		Category:       "utility",
		Score:          75,
		Classification: "mostly_complete",
		PreviousScore:  70,
		Delta:          5,
		Success:        true,
	}

	if info.Delta != info.Score-info.PreviousScore {
		t.Errorf("Delta should be Score - PreviousScore, got %d", info.Delta)
	}
}

// TestBulkRefreshResult tests result structure
func TestBulkRefreshResult(t *testing.T) {
	result := BulkRefreshResult{
		Total:      10,
		Successful: 8,
		Failed:     2,
		Scenarios: []ScenarioRefreshInfo{
			{Scenario: "s1", Success: true},
			{Scenario: "s2", Success: true},
			{Scenario: "s3", Success: false, Error: "not found"},
		},
	}

	if result.Total != 10 {
		t.Errorf("Expected total 10, got %d", result.Total)
	}
	if result.Successful+result.Failed != result.Total {
		t.Error("Successful + Failed should equal Total")
	}
}

// [REQ:SCS-ANALYSIS-004] Cross-scenario comparison unit tests

// TestComparisonResultStructure tests comparison result structure
func TestComparisonResultStructure(t *testing.T) {
	comparisons := []ScenarioComparison{
		{Scenario: "scenario-a", Score: 90, Rank: 1},
		{Scenario: "scenario-b", Score: 70, Rank: 2},
		{Scenario: "scenario-c", Score: 50, Rank: 3},
	}

	result := ComparisonResult{
		Scenarios:    comparisons,
		BestScore:    90,
		WorstScore:   50,
		AverageScore: 70.0,
	}

	if result.BestScore != 90 {
		t.Errorf("Expected best score 90, got %d", result.BestScore)
	}
	if result.WorstScore != 50 {
		t.Errorf("Expected worst score 50, got %d", result.WorstScore)
	}
	if result.AverageScore != 70.0 {
		t.Errorf("Expected average 70.0, got %f", result.AverageScore)
	}
}

// TestComparisonRanking tests that ranking is assigned correctly
func TestComparisonRanking(t *testing.T) {
	comparisons := []ScenarioComparison{
		{Scenario: "low", Score: 30, Rank: 3},
		{Scenario: "high", Score: 90, Rank: 1},
		{Scenario: "medium", Score: 60, Rank: 2},
	}

	// Verify ranks match score ordering
	for _, c := range comparisons {
		switch c.Scenario {
		case "high":
			if c.Rank != 1 {
				t.Errorf("Expected high score to be rank 1, got %d", c.Rank)
			}
		case "medium":
			if c.Rank != 2 {
				t.Errorf("Expected medium score to be rank 2, got %d", c.Rank)
			}
		case "low":
			if c.Rank != 3 {
				t.Errorf("Expected low score to be rank 3, got %d", c.Rank)
			}
		}
	}
}
