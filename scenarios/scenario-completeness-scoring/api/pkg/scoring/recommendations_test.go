package scoring

import (
	"testing"
)

// [REQ:SCS-ANALYSIS-002] Test recommendations generation
func TestGenerateRecommendations(t *testing.T) {
	// Template UI breakdown
	breakdown := ScoreBreakdown{
		Quality: QualityScore{
			RequirementPassRate: PassRate{Rate: 0.7},
			TargetPassRate:      PassRate{Rate: 0.8},
			TestPassRate:        PassRate{Rate: 0.6},
		},
		UI: UIScore{
			TemplateCheck:       TemplateCheckResult{IsTemplate: true},
			ComponentComplexity: ComponentComplexity{Threshold: "below", FileCount: 5},
			APIIntegration:      APIIntegration{EndpointCount: 0},
		},
		Quantity: QuantityScore{
			Tests: QuantityMetric{Threshold: "below"},
		},
		Coverage: CoverageScore{
			TestCoverageRatio: CoverageRatio{Ratio: 1.0},
		},
	}
	thresholds := DefaultThresholds()

	recommendations := GenerateRecommendations(breakdown, thresholds)

	if len(recommendations) == 0 {
		t.Error("Expected recommendations for low-scoring scenario")
	}

	// Check that template recommendation is first (highest priority)
	hasTemplate := false
	for _, rec := range recommendations {
		if rec.Priority == 1 && rec.Impact == 10 {
			hasTemplate = true
		}
	}
	if !hasTemplate {
		t.Error("Expected template replacement to be first recommendation")
	}
}

// [REQ:SCS-ANALYSIS-002] Test recommendations for high-scoring scenario
func TestGenerateRecommendationsHighScore(t *testing.T) {
	// High-scoring breakdown (no template, good pass rates)
	breakdown := ScoreBreakdown{
		Quality: QualityScore{
			RequirementPassRate: PassRate{Rate: 0.95},
			TargetPassRate:      PassRate{Rate: 0.95},
			TestPassRate:        PassRate{Rate: 0.95},
		},
		UI: UIScore{
			TemplateCheck:       TemplateCheckResult{IsTemplate: false},
			ComponentComplexity: ComponentComplexity{Threshold: "good", FileCount: 30},
			APIIntegration:      APIIntegration{EndpointCount: 5},
		},
		Quantity: QuantityScore{
			Tests: QuantityMetric{Threshold: "excellent"},
		},
		Coverage: CoverageScore{
			TestCoverageRatio: CoverageRatio{Ratio: 2.5},
		},
	}
	thresholds := DefaultThresholds()

	recommendations := GenerateRecommendations(breakdown, thresholds)

	// High-scoring scenario should have fewer recommendations
	if len(recommendations) > 3 {
		t.Errorf("Expected few recommendations for high-scoring scenario, got %d", len(recommendations))
	}
}

// [REQ:SCS-ANALYSIS-002] Test recommendation impact values
func TestRecommendationImpactValues(t *testing.T) {
	breakdown := ScoreBreakdown{
		Quality: QualityScore{
			RequirementPassRate: PassRate{Rate: 0.5},
			TargetPassRate:      PassRate{Rate: 0.5},
			TestPassRate:        PassRate{Rate: 0.5},
		},
		UI: UIScore{
			TemplateCheck:       TemplateCheckResult{IsTemplate: true},
			ComponentComplexity: ComponentComplexity{Threshold: "below", FileCount: 3},
			APIIntegration:      APIIntegration{EndpointCount: 0},
		},
		Quantity: QuantityScore{
			Tests: QuantityMetric{Threshold: "below"},
		},
		Coverage: CoverageScore{
			TestCoverageRatio: CoverageRatio{Ratio: 0.5},
		},
	}
	thresholds := DefaultThresholds()

	recommendations := GenerateRecommendations(breakdown, thresholds)

	// Template replacement should have impact of 10 (highest single fix)
	for _, rec := range recommendations {
		if rec.Impact == 10 {
			if rec.Priority != 1 {
				t.Error("10-point impact recommendation should be first priority")
			}
			break
		}
	}

	// All recommendations should have positive impact
	for _, rec := range recommendations {
		if rec.Impact <= 0 {
			t.Errorf("Recommendation should have positive impact, got %d", rec.Impact)
		}
	}
}

// [REQ:SCS-ANALYSIS-002] Test recommendation priority ordering
func TestRecommendationPriority(t *testing.T) {
	breakdown := ScoreBreakdown{
		Quality: QualityScore{
			RequirementPassRate: PassRate{Rate: 0.5},
			TargetPassRate:      PassRate{Rate: 0.5},
			TestPassRate:        PassRate{Rate: 0.5},
		},
		UI: UIScore{
			TemplateCheck:       TemplateCheckResult{IsTemplate: true},
			ComponentComplexity: ComponentComplexity{Threshold: "below", FileCount: 5},
			APIIntegration:      APIIntegration{EndpointCount: 0},
		},
		Quantity: QuantityScore{
			Tests: QuantityMetric{Threshold: "ok"},
		},
		Coverage: CoverageScore{
			TestCoverageRatio: CoverageRatio{Ratio: 1.0},
		},
	}
	thresholds := DefaultThresholds()

	recommendations := GenerateRecommendations(breakdown, thresholds)

	// Verify priorities are sequential
	for i, rec := range recommendations {
		if rec.Priority != i+1 {
			t.Errorf("Expected priority %d, got %d", i+1, rec.Priority)
		}
	}
}
