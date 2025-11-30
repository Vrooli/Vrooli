// Package scoring - Decision Boundary Module
// This file contains all explicit decision-making functions for scoring.
// Each decision has a named function that documents:
// - What is being decided
// - What criteria determine the outcome
// - What the possible outcomes are
//
// This makes the scoring system's decision points explicit, testable, and easy to understand.

package scoring

import "math"

// =============================================================================
// SCORE CLASSIFICATION DECISIONS
// =============================================================================
// These constants define the score thresholds for each classification level.
// [REQ:SCS-CORE-001] Score classification thresholds

const (
	// ProductionReadyThreshold: Scores at or above this are production-ready
	// Rationale: 96+ indicates excellent validation coverage with minimal gaps
	ProductionReadyThreshold = 96

	// NearlyReadyThreshold: Scores at or above this (but below production) need final polish
	// Rationale: 81-95 indicates most work is done, just edge cases remain
	NearlyReadyThreshold = 81

	// MostlyCompleteThreshold: Scores at or above this need refinement and validation
	// Rationale: 61-80 indicates core features work but need more coverage
	MostlyCompleteThreshold = 61

	// FunctionalIncompleteThreshold: Scores at or above this are functional but incomplete
	// Rationale: 41-60 indicates basic functionality exists but significant gaps
	FunctionalIncompleteThreshold = 41

	// FoundationLaidThreshold: Scores at or above this have foundation laid
	// Rationale: 21-40 indicates core architecture exists but much work needed
	FoundationLaidThreshold = 21

	// Scores below FoundationLaidThreshold are "early_stage"
)

// DecideClassification determines the classification level for a given score.
// This is the single source of truth for how scores map to classifications.
// [REQ:SCS-CORE-001] Score classification decision
func DecideClassification(score int) (classification string, description string) {
	switch {
	case score >= ProductionReadyThreshold:
		return "production_ready", "Production ready, excellent validation coverage"
	case score >= NearlyReadyThreshold:
		return "nearly_ready", "Nearly ready, final polish and edge cases"
	case score >= MostlyCompleteThreshold:
		return "mostly_complete", "Mostly complete, needs refinement and validation"
	case score >= FunctionalIncompleteThreshold:
		return "functional_incomplete", "Functional but incomplete, needs more features/tests"
	case score >= FoundationLaidThreshold:
		return "foundation_laid", "Foundation laid, core features in progress"
	default:
		return "early_stage", "Just starting, needs significant development"
	}
}

// IsProductionReady decides if a score indicates production readiness.
func IsProductionReady(score int) bool {
	return score >= ProductionReadyThreshold
}

// IsNearlyReady decides if a score indicates near-completion (final polish needed).
func IsNearlyReady(score int) bool {
	return score >= NearlyReadyThreshold && score < ProductionReadyThreshold
}

// RequiresSignificantWork decides if a score indicates significant work is needed.
func RequiresSignificantWork(score int) bool {
	return score < MostlyCompleteThreshold
}

// =============================================================================
// UI SCORING DECISIONS
// =============================================================================
// These constants and functions define the decision points for UI scoring.
// [REQ:SCS-CORE-001D] UI dimension scoring decisions

const (
	// TemplateUIPoints: Points awarded for non-template UI (10 points for real UI, 0 for template)
	TemplateUIPoints = 10

	// MaxComponentPoints: Maximum points for component complexity (5 points)
	MaxComponentPoints = 5

	// MaxAPIPoints: Maximum points for API integration depth (6 points)
	MaxAPIPoints = 6

	// MaxRoutingPoints: Maximum points for routing complexity (1.5 points)
	MaxRoutingPoints = 1.5

	// MaxVolumePoints: Maximum points for code volume (2.5 points)
	MaxVolumePoints = 2.5

	// UI component thresholds (file counts)
	UIFileCountMinimum    = 5   // Minimum files for any points
	UIFileCountLow        = 10  // Low complexity threshold
	UIRouteCountBasic     = 1   // Basic routing (0.5 points)
	UIRouteCountModerate  = 3   // Moderate routing (1.0 point)
	UIRouteCountAdvanced  = 5   // Advanced routing (1.5 points)
	UILOCMinimum          = 100 // Minimum LOC for any volume points
)

// DecideTemplatePoints decides if the UI is a template and returns appropriate points.
// Decision: Template UIs get 0 points, real UIs get 10 points.
// Rationale: Template UIs indicate the developer hasn't customized the interface.
func DecideTemplatePoints(isTemplate bool) (points int, isTemplateUI bool) {
	if isTemplate {
		return 0, true
	}
	return TemplateUIPoints, false
}

// DecideComponentComplexityPoints decides points based on UI file count.
// Decision tree:
// - 0-4 files: 0 points (minimal UI)
// - 5-9 files: 1 point (basic UI)
// - 10-OK threshold: 2 points (developing UI)
// - OK-Good threshold: 3 points (solid UI)
// - Good-Excellent threshold: 4 points (comprehensive UI)
// - Excellent+: 5 points (full-featured UI)
func DecideComponentComplexityPoints(fileCount int, thresholds ThresholdLevels) int {
	switch {
	case fileCount >= thresholds.Excellent:
		return MaxComponentPoints
	case fileCount >= thresholds.Good:
		return 4
	case fileCount >= thresholds.OK:
		return 3
	case fileCount >= UIFileCountLow:
		return 2
	case fileCount >= UIFileCountMinimum:
		return 1
	default:
		return 0
	}
}

// DecideAPIIntegrationPoints decides points based on API endpoint usage.
// Decision: More endpoints integrated = higher points (up to 6).
// Only counts endpoints beyond /health (the minimum required endpoint).
func DecideAPIIntegrationPoints(apiBeyondHealth int, thresholds ThresholdLevels) int {
	switch {
	case apiBeyondHealth >= thresholds.Excellent:
		return MaxAPIPoints
	case apiBeyondHealth >= thresholds.Good:
		return 5
	case apiBeyondHealth >= thresholds.OK:
		return 4
	case apiBeyondHealth >= 2:
		return 3
	case apiBeyondHealth >= 1:
		return 2
	default:
		return 0
	}
}

// DecideRoutingPoints decides points based on route count.
// Decision: More routes = higher points, but capped since SPAs can be complete without many routes.
// Rationale: Routing complexity is a minor indicator (1.5 max) - many valid UIs are single-route.
func DecideRoutingPoints(routeCount int) float64 {
	switch {
	case routeCount >= UIRouteCountAdvanced:
		return MaxRoutingPoints
	case routeCount >= UIRouteCountModerate:
		return 1.0
	case routeCount >= UIRouteCountBasic:
		return 0.5
	default:
		return 0.0
	}
}

// DecideVolumePoints decides points based on total lines of code.
// Decision: More code = higher points, indicating more substantial UI.
// Rationale: LOC is an imperfect metric but correlates with feature completeness.
func DecideVolumePoints(totalLOC int, thresholds ThresholdLevels) float64 {
	switch {
	case totalLOC >= thresholds.Excellent:
		return MaxVolumePoints
	case totalLOC >= thresholds.Good:
		return 2.0
	case totalLOC >= thresholds.OK:
		return 1.5
	case totalLOC >= UILOCMinimum:
		return 0.5
	default:
		return 0.0
	}
}

// =============================================================================
// THRESHOLD LEVEL DECISIONS
// =============================================================================
// These functions decide what level a count falls into based on thresholds.
// [REQ:SCS-CFG-002] Threshold-based decisions

// DecideThresholdLevel determines which level a count falls into.
// Levels: "excellent" >= Excellent, "good" >= Good, "ok" >= OK, "below" otherwise.
// This is the standard decision used across requirements, targets, tests, and UI metrics.
func DecideThresholdLevel(count int, thresholds ThresholdLevels) string {
	switch {
	case count >= thresholds.Excellent:
		return "excellent"
	case count >= thresholds.Good:
		return "good"
	case count >= thresholds.OK:
		return "ok"
	default:
		return "below"
	}
}

// IsAboveMinimumThreshold decides if a count meets at least the OK threshold.
func IsAboveMinimumThreshold(count int, thresholds ThresholdLevels) bool {
	return count >= thresholds.OK
}

// IsExcellent decides if a count meets the excellent threshold.
func IsExcellent(count int, thresholds ThresholdLevels) bool {
	return count >= thresholds.Excellent
}

// =============================================================================
// COVERAGE RATIO DECISIONS
// =============================================================================
// These constants and functions define the decision points for coverage scoring.
// [REQ:SCS-CORE-001B] Coverage dimension decisions

const (
	// OptimalTestToRequirementRatio: The ideal ratio of tests to requirements
	// Rationale: 2:1 means each requirement has ~2 tests (unit + integration/e2e)
	OptimalTestToRequirementRatio = 2.0

	// OptimalRequirementDepth: The ideal average depth of requirement hierarchies
	// Rationale: 3 levels (module > feature > specific) indicates well-decomposed requirements
	OptimalRequirementDepth = 3.0

	// MaxTestCoveragePoints: Maximum points for test coverage ratio (8 points)
	MaxTestCoveragePoints = 8

	// MaxDepthPoints: Maximum points for requirement depth (7 points)
	MaxDepthPoints = 7
)

// DecideTestCoveragePoints calculates points based on test-to-requirement ratio.
// Decision: Ratio up to 2.0x earns full 8 points, proportionally less below.
// Rationale: Encourages multiple test types per requirement without rewarding test proliferation.
func DecideTestCoveragePoints(testCount, requirementCount int) (points int, ratio float64) {
	if requirementCount == 0 {
		return 0, 0.0
	}

	ratio = float64(testCount) / float64(requirementCount)
	cappedRatio := min(ratio, OptimalTestToRequirementRatio)
	normalizedRatio := cappedRatio / OptimalTestToRequirementRatio
	points = int(math.Round(normalizedRatio * float64(MaxTestCoveragePoints)))

	return points, ratio
}

// DecideDepthPoints calculates points based on average requirement depth.
// Decision: Depth up to 3.0 levels earns full 7 points, proportionally less below.
// Rationale: Encourages hierarchical requirement decomposition.
func DecideDepthPoints(avgDepth float64) int {
	cappedDepth := min(avgDepth/OptimalRequirementDepth, 1.0)
	return int(math.Round(cappedDepth * float64(MaxDepthPoints)))
}

// =============================================================================
// QUALITY DIMENSION DECISIONS
// =============================================================================
// These constants define the point allocation for quality scoring.
// [REQ:SCS-CORE-001A] Quality dimension decisions

const (
	// MaxRequirementPassRatePoints: Maximum points for requirement pass rate (20 points)
	MaxRequirementPassRatePoints = 20

	// MaxTargetPassRatePoints: Maximum points for target pass rate (15 points)
	MaxTargetPassRatePoints = 15

	// MaxTestPassRatePoints: Maximum points for test pass rate (15 points)
	MaxTestPassRatePoints = 15
)

// DecidePassRatePoints calculates points based on a pass rate.
// Decision: Points are proportional to pass rate (0-100% maps to 0-maxPoints).
func DecidePassRatePoints(passing, total, maxPoints int) (points int, rate float64) {
	if total == 0 {
		return 0, 0.0
	}
	rate = float64(passing) / float64(total)
	points = int(math.Round(rate * float64(maxPoints)))
	return points, rate
}

// =============================================================================
// QUANTITY DIMENSION DECISIONS
// =============================================================================
// These constants define the point allocation for quantity scoring.
// [REQ:SCS-CORE-001C] Quantity dimension decisions

const (
	// MaxRequirementQuantityPoints: Maximum points for requirement count (4 points)
	MaxRequirementQuantityPoints = 4

	// MaxTargetQuantityPoints: Maximum points for target count (3 points)
	MaxTargetQuantityPoints = 3

	// MaxTestQuantityPoints: Maximum points for test count (3 points)
	MaxTestQuantityPoints = 3
)

// DecideQuantityPoints calculates points based on count vs threshold.
// Decision: Points are proportional to count/Good threshold ratio, capped at maxPoints.
// Rationale: Using "Good" threshold as the 100% target encourages reaching solid coverage.
func DecideQuantityPoints(count, goodThreshold, maxPoints int) int {
	if goodThreshold <= 0 {
		return 0
	}
	ratio := min(float64(count)/float64(goodThreshold), 1.0)
	return int(math.Round(ratio * float64(maxPoints)))
}

// =============================================================================
// RECOMMENDATION DECISIONS
// =============================================================================
// These functions decide when to generate specific recommendations.
// [REQ:SCS-ANALYSIS-002] Recommendation generation decisions

const (
	// RecommendedPassRate: The minimum pass rate before recommending improvements
	RecommendedPassRate = 0.90
)

// ShouldRecommendTestPassRateImprovement decides if we should suggest improving test pass rate.
func ShouldRecommendTestPassRateImprovement(rate float64) bool {
	return rate < RecommendedPassRate
}

// ShouldRecommendRequirementPassRateImprovement decides if we should suggest improving requirement pass rate.
func ShouldRecommendRequirementPassRateImprovement(rate float64) bool {
	return rate < RecommendedPassRate
}

// ShouldRecommendTargetPassRateImprovement decides if we should suggest improving target pass rate.
func ShouldRecommendTargetPassRateImprovement(rate float64) bool {
	return rate < RecommendedPassRate
}

// ShouldRecommendTemplateReplacement decides if we should suggest replacing template UI.
func ShouldRecommendTemplateReplacement(isTemplate bool) bool {
	return isTemplate
}

// ShouldRecommendMoreUIFiles decides if we should suggest adding more UI files.
func ShouldRecommendMoreUIFiles(threshold string) bool {
	return threshold == "below"
}

// ShouldRecommendAPIIntegration decides if we should suggest integrating more API endpoints.
func ShouldRecommendAPIIntegration(endpointCount int) bool {
	return endpointCount == 0
}

// ShouldRecommendMoreTests decides if we should suggest adding more tests.
func ShouldRecommendMoreTests(threshold string) bool {
	return threshold == "below" || threshold == "ok"
}

// ShouldRecommendBetterCoverage decides if we should suggest improving test coverage ratio.
func ShouldRecommendBetterCoverage(ratio float64) bool {
	return ratio < OptimalTestToRequirementRatio
}

// NOTE: This file previously defined custom roundHalf and min helper functions.
// These were removed in favor of Go built-in functions:
// - roundHalf -> math.Round (standard library, handles edge cases correctly)
// - min -> built-in min (Go 1.21+, works with any ordered types including float64)
