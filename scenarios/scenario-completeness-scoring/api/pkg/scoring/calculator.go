package scoring

import (
	"math"
)

// DefaultThresholds returns the default threshold config for "utility" category
func DefaultThresholds() ThresholdConfig {
	return ThresholdConfig{
		Requirements: ThresholdLevels{OK: 10, Good: 15, Excellent: 25},
		Targets:      ThresholdLevels{OK: 8, Good: 12, Excellent: 20},
		Tests:        ThresholdLevels{OK: 15, Good: 25, Excellent: 40},
		UI: UIThresholdConfig{
			FileCount:    ThresholdLevels{OK: 15, Good: 25, Excellent: 40},
			TotalLOC:     ThresholdLevels{OK: 300, Good: 600, Excellent: 1200},
			APIEndpoints: ThresholdLevels{OK: 2, Good: 4, Excellent: 8},
		},
	}
}

// CategoryThresholds maps category names to their threshold configs
var CategoryThresholds = map[string]ThresholdConfig{
	"utility": DefaultThresholds(),
	"business-application": {
		Requirements: ThresholdLevels{OK: 25, Good: 40, Excellent: 60},
		Targets:      ThresholdLevels{OK: 15, Good: 25, Excellent: 35},
		Tests:        ThresholdLevels{OK: 40, Good: 60, Excellent: 100},
		UI: UIThresholdConfig{
			FileCount:    ThresholdLevels{OK: 30, Good: 50, Excellent: 80},
			TotalLOC:     ThresholdLevels{OK: 1500, Good: 3000, Excellent: 6000},
			APIEndpoints: ThresholdLevels{OK: 5, Good: 10, Excellent: 20},
		},
	},
	"automation": {
		Requirements: ThresholdLevels{OK: 15, Good: 25, Excellent: 40},
		Targets:      ThresholdLevels{OK: 12, Good: 18, Excellent: 28},
		Tests:        ThresholdLevels{OK: 25, Good: 40, Excellent: 70},
		UI: UIThresholdConfig{
			FileCount:    ThresholdLevels{OK: 20, Good: 35, Excellent: 60},
			TotalLOC:     ThresholdLevels{OK: 800, Good: 1800, Excellent: 3500},
			APIEndpoints: ThresholdLevels{OK: 4, Good: 8, Excellent: 15},
		},
	},
	"platform": {
		Requirements: ThresholdLevels{OK: 30, Good: 50, Excellent: 80},
		Targets:      ThresholdLevels{OK: 20, Good: 30, Excellent: 45},
		Tests:        ThresholdLevels{OK: 50, Good: 80, Excellent: 120},
		UI: UIThresholdConfig{
			FileCount:    ThresholdLevels{OK: 40, Good: 70, Excellent: 120},
			TotalLOC:     ThresholdLevels{OK: 2000, Good: 4500, Excellent: 8000},
			APIEndpoints: ThresholdLevels{OK: 8, Good: 15, Excellent: 30},
		},
	},
	"developer_tools": {
		Requirements: ThresholdLevels{OK: 20, Good: 30, Excellent: 50},
		Targets:      ThresholdLevels{OK: 12, Good: 20, Excellent: 30},
		Tests:        ThresholdLevels{OK: 30, Good: 50, Excellent: 80},
		UI: UIThresholdConfig{
			FileCount:    ThresholdLevels{OK: 25, Good: 45, Excellent: 75},
			TotalLOC:     ThresholdLevels{OK: 1000, Good: 2500, Excellent: 5000},
			APIEndpoints: ThresholdLevels{OK: 4, Good: 8, Excellent: 15},
		},
	},
}

// GetThresholds returns the threshold config for a category
func GetThresholds(category string) ThresholdConfig {
	if cfg, ok := CategoryThresholds[category]; ok {
		return cfg
	}
	return DefaultThresholds()
}

// CalculateQualityScore computes the quality dimension score (50 points max)
// [REQ:SCS-CORE-001A] Quality dimension scoring
func CalculateQualityScore(metrics Metrics) QualityScore {
	reqPassRate := 0.0
	if metrics.Requirements.Total > 0 {
		reqPassRate = float64(metrics.Requirements.Passing) / float64(metrics.Requirements.Total)
	}
	targetPassRate := 0.0
	if metrics.Targets.Total > 0 {
		targetPassRate = float64(metrics.Targets.Passing) / float64(metrics.Targets.Total)
	}
	testPassRate := 0.0
	if metrics.Tests.Total > 0 {
		testPassRate = float64(metrics.Tests.Passing) / float64(metrics.Tests.Total)
	}

	reqPoints := int(math.Round(reqPassRate * 20))
	targetPoints := int(math.Round(targetPassRate * 15))
	testPoints := int(math.Round(testPassRate * 15))

	return QualityScore{
		Score: reqPoints + targetPoints + testPoints,
		Max:   50,
		RequirementPassRate: PassRate{
			Passing: metrics.Requirements.Passing,
			Total:   metrics.Requirements.Total,
			Rate:    reqPassRate,
			Points:  reqPoints,
		},
		TargetPassRate: PassRate{
			Passing: metrics.Targets.Passing,
			Total:   metrics.Targets.Total,
			Rate:    targetPassRate,
			Points:  targetPoints,
		},
		TestPassRate: PassRate{
			Passing: metrics.Tests.Passing,
			Total:   metrics.Tests.Total,
			Rate:    testPassRate,
			Points:  testPoints,
		},
	}
}

// getMaxDepth calculates the maximum depth of a requirement tree
func getMaxDepth(req RequirementTree) int {
	if len(req.Children) == 0 {
		return 1
	}
	maxChildDepth := 0
	for _, child := range req.Children {
		d := getMaxDepth(child)
		if d > maxChildDepth {
			maxChildDepth = d
		}
	}
	return 1 + maxChildDepth
}

// CalculateCoverageScore computes the coverage dimension score (15 points max)
// [REQ:SCS-CORE-001B] Coverage dimension scoring
func CalculateCoverageScore(metrics Metrics, requirements []RequirementTree) CoverageScore {
	// Test coverage ratio (cap at 2.0x = perfect score)
	testCoverageRatio := 0.0
	if metrics.Requirements.Total > 0 {
		testCoverageRatio = float64(metrics.Tests.Total) / float64(metrics.Requirements.Total)
	}
	cappedRatio := math.Min(testCoverageRatio, 2.0)
	testCoveragePoints := int(math.Round((cappedRatio / 2.0) * 8))

	// Requirement depth score (cap at 3.0 levels = perfect score)
	depthScore := 0
	avgDepth := 0.0
	if len(requirements) > 0 {
		totalDepth := 0
		for _, req := range requirements {
			totalDepth += getMaxDepth(req)
		}
		avgDepth = float64(totalDepth) / float64(len(requirements))
		cappedDepth := math.Min(avgDepth/3.0, 1.0)
		depthScore = int(math.Round(cappedDepth * 7))
	}

	return CoverageScore{
		Score: testCoveragePoints + depthScore,
		Max:   15,
		TestCoverageRatio: CoverageRatio{
			Ratio:  testCoverageRatio,
			Points: testCoveragePoints,
		},
		DepthScore: DepthScoreDetail{
			AvgDepth: avgDepth,
			Points:   depthScore,
		},
	}
}

// getThresholdLevel determines threshold level (excellent, good, ok, below)
func getThresholdLevel(count int, thresholds ThresholdLevels) string {
	if count >= thresholds.Excellent {
		return "excellent"
	}
	if count >= thresholds.Good {
		return "good"
	}
	if count >= thresholds.OK {
		return "ok"
	}
	return "below"
}

// CalculateQuantityScore computes the quantity dimension score (10 points max)
// [REQ:SCS-CORE-001C] Quantity dimension scoring
func CalculateQuantityScore(metrics Metrics, thresholds ThresholdConfig) QuantityScore {
	// Requirements score (4 points max)
	reqRatio := math.Min(float64(metrics.Requirements.Total)/float64(thresholds.Requirements.Good), 1.0)
	reqPoints := int(math.Round(reqRatio * 4))

	// Targets score (3 points max)
	targetRatio := math.Min(float64(metrics.Targets.Total)/float64(thresholds.Targets.Good), 1.0)
	targetPoints := int(math.Round(targetRatio * 3))

	// Tests score (3 points max)
	testRatio := math.Min(float64(metrics.Tests.Total)/float64(thresholds.Tests.Good), 1.0)
	testPoints := int(math.Round(testRatio * 3))

	return QuantityScore{
		Score: reqPoints + targetPoints + testPoints,
		Max:   10,
		Requirements: QuantityMetric{
			Count:     metrics.Requirements.Total,
			Threshold: getThresholdLevel(metrics.Requirements.Total, thresholds.Requirements),
			Points:    reqPoints,
		},
		Targets: QuantityMetric{
			Count:     metrics.Targets.Total,
			Threshold: getThresholdLevel(metrics.Targets.Total, thresholds.Targets),
			Points:    targetPoints,
		},
		Tests: QuantityMetric{
			Count:     metrics.Tests.Total,
			Threshold: getThresholdLevel(metrics.Tests.Total, thresholds.Tests),
			Points:    testPoints,
		},
	}
}

// CalculateUIScore computes the UI dimension score (25 points max)
// [REQ:SCS-CORE-001D] UI dimension scoring
func CalculateUIScore(uiMetrics *UIMetrics, thresholds ThresholdConfig) UIScore {
	if uiMetrics == nil {
		return UIScore{
			Score: 0,
			Max:   25,
			TemplateCheck: TemplateCheckResult{
				IsTemplate: true,
				Penalty:    25,
				Points:     0,
			},
			ComponentComplexity: ComponentComplexity{
				FileCount: 0,
				Threshold: "none",
				Points:    0,
			},
			APIIntegration: APIIntegration{
				EndpointCount: 0,
				Points:        0,
			},
			Routing: RoutingScore{
				HasRouting: false,
				RouteCount: 0,
				Points:     0,
			},
			CodeVolume: CodeVolume{
				TotalLOC: 0,
				Points:   0,
			},
		}
	}

	// Template signature check (10 points) - Binary: 0 if template, 10 if not
	templatePoints := 0
	if !uiMetrics.IsTemplate {
		templatePoints = 10
	}

	// Component count (5 points) - Based on file count vs thresholds
	componentPoints := 0
	if uiMetrics.FileCount >= thresholds.UI.FileCount.Excellent {
		componentPoints = 5
	} else if uiMetrics.FileCount >= thresholds.UI.FileCount.Good {
		componentPoints = 4
	} else if uiMetrics.FileCount >= thresholds.UI.FileCount.OK {
		componentPoints = 3
	} else if uiMetrics.FileCount >= 10 {
		componentPoints = 2
	} else if uiMetrics.FileCount >= 5 {
		componentPoints = 1
	}

	// API integration depth (6 points) - Unique endpoints beyond /health
	apiPoints := 0
	if uiMetrics.APIBeyondHealth >= thresholds.UI.APIEndpoints.Excellent {
		apiPoints = 6
	} else if uiMetrics.APIBeyondHealth >= thresholds.UI.APIEndpoints.Good {
		apiPoints = 5
	} else if uiMetrics.APIBeyondHealth >= thresholds.UI.APIEndpoints.OK {
		apiPoints = 4
	} else if uiMetrics.APIBeyondHealth >= 2 {
		apiPoints = 3
	} else if uiMetrics.APIBeyondHealth >= 1 {
		apiPoints = 2
	}

	// Router complexity (1.5 points) - Lower weight since SPAs can be complete
	routerPoints := 0.0
	if uiMetrics.RouteCount >= 5 {
		routerPoints = 1.5
	} else if uiMetrics.RouteCount >= 3 {
		routerPoints = 1.0
	} else if uiMetrics.RouteCount >= 1 {
		routerPoints = 0.5
	}

	// Code volume (2.5 points) - Total LOC vs threshold
	volumePoints := 0.0
	if uiMetrics.TotalLOC >= thresholds.UI.TotalLOC.Excellent {
		volumePoints = 2.5
	} else if uiMetrics.TotalLOC >= thresholds.UI.TotalLOC.Good {
		volumePoints = 2.0
	} else if uiMetrics.TotalLOC >= thresholds.UI.TotalLOC.OK {
		volumePoints = 1.5
	} else if uiMetrics.TotalLOC >= 100 {
		volumePoints = 0.5
	}

	totalUIScore := int(math.Round(float64(templatePoints+componentPoints+apiPoints) + routerPoints + volumePoints))

	// Determine threshold level for file count
	fileCountThreshold := "none"
	if uiMetrics.FileCount >= thresholds.UI.FileCount.Excellent {
		fileCountThreshold = "excellent"
	} else if uiMetrics.FileCount >= thresholds.UI.FileCount.Good {
		fileCountThreshold = "good"
	} else if uiMetrics.FileCount >= thresholds.UI.FileCount.OK {
		fileCountThreshold = "ok"
	} else if uiMetrics.FileCount > 0 {
		fileCountThreshold = "below"
	}

	return UIScore{
		Score: totalUIScore,
		Max:   25,
		TemplateCheck: TemplateCheckResult{
			IsTemplate: uiMetrics.IsTemplate,
			Penalty:    0,
			Points:     templatePoints,
		},
		ComponentComplexity: ComponentComplexity{
			FileCount:      uiMetrics.FileCount,
			ComponentCount: uiMetrics.ComponentCount,
			PageCount:      uiMetrics.PageCount,
			Threshold:      fileCountThreshold,
			Points:         componentPoints,
		},
		APIIntegration: APIIntegration{
			EndpointCount:  uiMetrics.APIBeyondHealth,
			TotalEndpoints: uiMetrics.APIEndpoints,
			Points:         apiPoints,
		},
		Routing: RoutingScore{
			HasRouting: uiMetrics.HasRouting,
			RouteCount: uiMetrics.RouteCount,
			Points:     routerPoints,
		},
		CodeVolume: CodeVolume{
			TotalLOC: uiMetrics.TotalLOC,
			Points:   volumePoints,
		},
	}
}

// ClassifyScore maps a score to a classification level
func ClassifyScore(score int) (string, string) {
	switch {
	case score >= 96:
		return "production_ready", "Production ready, excellent validation coverage"
	case score >= 81:
		return "nearly_ready", "Nearly ready, final polish and edge cases"
	case score >= 61:
		return "mostly_complete", "Mostly complete, needs refinement and validation"
	case score >= 41:
		return "functional_incomplete", "Functional but incomplete, needs more features/tests"
	case score >= 21:
		return "foundation_laid", "Foundation laid, core features in progress"
	default:
		return "early_stage", "Just starting, needs significant development"
	}
}

// CalculateCompletenessScore computes the overall completeness score
// [REQ:SCS-CORE-001] Calculate completeness scores with 4 dimensions
func CalculateCompletenessScore(metrics Metrics, thresholds ThresholdConfig, validation *ValidationQualityAnalysis) ScoreBreakdown {
	quality := CalculateQualityScore(metrics)
	coverage := CalculateCoverageScore(metrics, metrics.Requirements_)
	quantity := CalculateQuantityScore(metrics, thresholds)
	ui := CalculateUIScore(metrics.UI, thresholds)

	baseScore := quality.Score + coverage.Score + quantity.Score + ui.Score

	// Apply validation quality penalties if provided
	validationPenalty := 0
	if validation != nil {
		validationPenalty = validation.TotalPenalty
	}
	finalScore := baseScore - validationPenalty
	if finalScore < 0 {
		finalScore = 0
	}

	classification, desc := ClassifyScore(finalScore)

	return ScoreBreakdown{
		BaseScore:          baseScore,
		ValidationPenalty:  validationPenalty,
		Score:              finalScore,
		Classification:     classification,
		ClassificationDesc: desc,
		Quality:            quality,
		Coverage:           coverage,
		Quantity:           quantity,
		UI:                 ui,
	}
}

// CalculateCompletenessScoreWithOptions computes the overall completeness score using config options
// [REQ:SCS-CFG-001] Component toggle support - dimensions can be disabled via options
func CalculateCompletenessScoreWithOptions(metrics Metrics, thresholds ThresholdConfig, validation *ValidationQualityAnalysis, opts *ScoringOptions) ScoreBreakdown {
	if opts == nil {
		opts = DefaultScoringOptions()
	}

	// Always calculate all dimensions to show actual metrics
	// Even disabled dimensions show their metrics for transparency
	var quality QualityScore
	var coverage CoverageScore
	var quantity QuantityScore
	var ui UIScore

	// Track which dimensions contribute to the score
	var totalRawScore float64

	// Quality dimension - always calculate, but only count if enabled
	quality = CalculateQualityScore(metrics)
	if opts.QualityEnabled && opts.QualityWeight > 0 {
		// Scale the score based on weight ratio (original max is 50)
		scaledScore := float64(quality.Score) * float64(opts.QualityWeight) / 50.0
		totalRawScore += scaledScore
		quality.Max = opts.QualityWeight
		quality.Score = int(math.Round(scaledScore))
	} else {
		// Keep the calculated metrics but mark as disabled with zero score
		quality.Score = 0
		quality.Max = 0
		quality.Disabled = true
	}

	// Coverage dimension - always calculate, but only count if enabled
	coverage = CalculateCoverageScore(metrics, metrics.Requirements_)
	if opts.CoverageEnabled && opts.CoverageWeight > 0 {
		// Scale the score based on weight ratio (original max is 15)
		scaledScore := float64(coverage.Score) * float64(opts.CoverageWeight) / 15.0
		totalRawScore += scaledScore
		coverage.Max = opts.CoverageWeight
		coverage.Score = int(math.Round(scaledScore))
	} else {
		// Keep the calculated metrics but mark as disabled with zero score
		coverage.Score = 0
		coverage.Max = 0
		coverage.Disabled = true
	}

	// Quantity dimension - always calculate, but only count if enabled
	quantity = CalculateQuantityScore(metrics, thresholds)
	if opts.QuantityEnabled && opts.QuantityWeight > 0 {
		// Scale the score based on weight ratio (original max is 10)
		scaledScore := float64(quantity.Score) * float64(opts.QuantityWeight) / 10.0
		totalRawScore += scaledScore
		quantity.Max = opts.QuantityWeight
		quantity.Score = int(math.Round(scaledScore))
	} else {
		// Keep the calculated metrics but mark as disabled with zero score
		quantity.Score = 0
		quantity.Max = 0
		quantity.Disabled = true
	}

	// UI dimension - always calculate, but only count if enabled
	ui = CalculateUIScore(metrics.UI, thresholds)
	if opts.UIEnabled && opts.UIWeight > 0 {
		// Scale the score based on weight ratio (original max is 25)
		scaledScore := float64(ui.Score) * float64(opts.UIWeight) / 25.0
		totalRawScore += scaledScore
		ui.Max = opts.UIWeight
		ui.Score = int(math.Round(scaledScore))
	} else {
		// Keep the calculated metrics but mark as disabled with zero score
		ui.Score = 0
		ui.Max = 0
		ui.Disabled = true
	}

	baseScore := int(math.Round(totalRawScore))

	// Apply validation quality penalties if provided
	validationPenalty := 0
	if validation != nil {
		validationPenalty = validation.TotalPenalty
	}
	finalScore := baseScore - validationPenalty
	if finalScore < 0 {
		finalScore = 0
	}

	classification, desc := ClassifyScore(finalScore)

	return ScoreBreakdown{
		BaseScore:          baseScore,
		ValidationPenalty:  validationPenalty,
		Score:              finalScore,
		Classification:     classification,
		ClassificationDesc: desc,
		Quality:            quality,
		Coverage:           coverage,
		Quantity:           quantity,
		UI:                 ui,
	}
}

// GenerateRecommendations creates actionable improvement suggestions
func GenerateRecommendations(breakdown ScoreBreakdown, thresholds ThresholdConfig) []Recommendation {
	var recommendations []Recommendation
	priority := 1

	// UI recommendations (highest priority if template detected)
	if breakdown.UI.TemplateCheck.IsTemplate {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Replace template UI with scenario-specific interface",
			Impact:      10,
		})
		priority++
	}

	if breakdown.UI.ComponentComplexity.Threshold == "below" {
		gap := thresholds.UI.FileCount.OK - breakdown.UI.ComponentComplexity.FileCount
		if gap > 0 {
			recommendations = append(recommendations, Recommendation{
				Priority:    priority,
				Description: "Add more UI files to reach minimum threshold",
				Impact:      2,
			})
			priority++
		}
	}

	if breakdown.UI.APIIntegration.EndpointCount == 0 {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Integrate UI with API endpoints beyond /health",
			Impact:      4,
		})
		priority++
	}

	// Quality recommendations
	if breakdown.Quality.TestPassRate.Rate < 0.9 {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Increase test pass rate to 90%+",
			Impact:      5,
		})
		priority++
	}

	if breakdown.Quality.RequirementPassRate.Rate < 0.9 {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Increase requirement pass rate to 90%+",
			Impact:      5,
		})
		priority++
	}

	if breakdown.Quality.TargetPassRate.Rate < 0.9 {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Increase operational target pass rate to 90%+",
			Impact:      4,
		})
		priority++
	}

	// Quantity recommendations
	if breakdown.Quantity.Tests.Threshold == "below" || breakdown.Quantity.Tests.Threshold == "ok" {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Add more tests to reach 'good' threshold",
			Impact:      2,
		})
		priority++
	}

	// Coverage recommendations
	if breakdown.Coverage.TestCoverageRatio.Ratio < 2.0 {
		recommendations = append(recommendations, Recommendation{
			Priority:    priority,
			Description: "Add tests to reach optimal 2:1 test-to-requirement ratio",
			Impact:      3,
		})
	}

	return recommendations
}
