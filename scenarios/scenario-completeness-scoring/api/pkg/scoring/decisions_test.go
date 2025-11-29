package scoring

import (
	"testing"
)

// =============================================================================
// SCORE CLASSIFICATION DECISION TESTS
// =============================================================================
// [REQ:SCS-CORE-001] Test score classification decisions

func TestDecideClassification(t *testing.T) {
	tests := []struct {
		name           string
		score          int
		wantClass      string
		wantDesc       string
	}{
		// Boundary tests for ProductionReadyThreshold (96)
		{"production_ready_exact", 96, "production_ready", "Production ready, excellent validation coverage"},
		{"production_ready_high", 100, "production_ready", "Production ready, excellent validation coverage"},
		{"production_ready_boundary_below", 95, "nearly_ready", "Nearly ready, final polish and edge cases"},

		// Boundary tests for NearlyReadyThreshold (81)
		{"nearly_ready_exact", 81, "nearly_ready", "Nearly ready, final polish and edge cases"},
		{"nearly_ready_boundary_below", 80, "mostly_complete", "Mostly complete, needs refinement and validation"},

		// Boundary tests for MostlyCompleteThreshold (61)
		{"mostly_complete_exact", 61, "mostly_complete", "Mostly complete, needs refinement and validation"},
		{"mostly_complete_boundary_below", 60, "functional_incomplete", "Functional but incomplete, needs more features/tests"},

		// Boundary tests for FunctionalIncompleteThreshold (41)
		{"functional_incomplete_exact", 41, "functional_incomplete", "Functional but incomplete, needs more features/tests"},
		{"functional_incomplete_boundary_below", 40, "foundation_laid", "Foundation laid, core features in progress"},

		// Boundary tests for FoundationLaidThreshold (21)
		{"foundation_laid_exact", 21, "foundation_laid", "Foundation laid, core features in progress"},
		{"foundation_laid_boundary_below", 20, "early_stage", "Just starting, needs significant development"},

		// Edge cases
		{"early_stage_zero", 0, "early_stage", "Just starting, needs significant development"},
		{"early_stage_negative", -5, "early_stage", "Just starting, needs significant development"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClass, gotDesc := DecideClassification(tt.score)
			if gotClass != tt.wantClass {
				t.Errorf("DecideClassification(%d) classification = %v, want %v", tt.score, gotClass, tt.wantClass)
			}
			if gotDesc != tt.wantDesc {
				t.Errorf("DecideClassification(%d) description = %v, want %v", tt.score, gotDesc, tt.wantDesc)
			}
		})
	}
}

func TestIsProductionReady(t *testing.T) {
	tests := []struct {
		score int
		want  bool
	}{
		{100, true},
		{96, true},  // Exact boundary
		{95, false}, // Just below
		{50, false},
		{0, false},
	}

	for _, tt := range tests {
		got := IsProductionReady(tt.score)
		if got != tt.want {
			t.Errorf("IsProductionReady(%d) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

func TestIsNearlyReady(t *testing.T) {
	tests := []struct {
		score int
		want  bool
	}{
		{95, true},  // In range
		{81, true},  // Lower boundary
		{96, false}, // Above range (production_ready)
		{80, false}, // Below range
		{50, false},
	}

	for _, tt := range tests {
		got := IsNearlyReady(tt.score)
		if got != tt.want {
			t.Errorf("IsNearlyReady(%d) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

func TestRequiresSignificantWork(t *testing.T) {
	tests := []struct {
		score int
		want  bool
	}{
		{60, true},  // Functional incomplete
		{40, true},  // Foundation laid
		{20, true},  // Early stage
		{0, true},   // Zero
		{61, false}, // Mostly complete
		{81, false}, // Nearly ready
		{100, false}, // Production ready
	}

	for _, tt := range tests {
		got := RequiresSignificantWork(tt.score)
		if got != tt.want {
			t.Errorf("RequiresSignificantWork(%d) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

// =============================================================================
// UI SCORING DECISION TESTS
// =============================================================================
// [REQ:SCS-CORE-001D] Test UI scoring decisions

func TestDecideTemplatePoints(t *testing.T) {
	tests := []struct {
		isTemplate   bool
		wantPoints   int
		wantTemplate bool
	}{
		{true, 0, true},
		{false, TemplateUIPoints, false},
	}

	for _, tt := range tests {
		points, isTemplate := DecideTemplatePoints(tt.isTemplate)
		if points != tt.wantPoints {
			t.Errorf("DecideTemplatePoints(%v) points = %d, want %d", tt.isTemplate, points, tt.wantPoints)
		}
		if isTemplate != tt.wantTemplate {
			t.Errorf("DecideTemplatePoints(%v) isTemplate = %v, want %v", tt.isTemplate, isTemplate, tt.wantTemplate)
		}
	}
}

func TestDecideComponentComplexityPoints(t *testing.T) {
	thresholds := ThresholdLevels{OK: 15, Good: 25, Excellent: 40}

	tests := []struct {
		fileCount  int
		wantPoints int
	}{
		{0, 0},                        // Zero files
		{4, 0},                        // Below minimum
		{5, 1},                        // Minimum threshold
		{9, 1},                        // Still basic
		{10, 2},                       // Low threshold
		{14, 2},                       // Still low
		{15, 3},                       // OK threshold
		{24, 3},                       // Still OK
		{25, 4},                       // Good threshold
		{39, 4},                       // Still good
		{40, MaxComponentPoints},      // Excellent threshold
		{100, MaxComponentPoints},     // Well above excellent
	}

	for _, tt := range tests {
		got := DecideComponentComplexityPoints(tt.fileCount, thresholds)
		if got != tt.wantPoints {
			t.Errorf("DecideComponentComplexityPoints(%d, thresholds) = %d, want %d", tt.fileCount, got, tt.wantPoints)
		}
	}
}

func TestDecideAPIIntegrationPoints(t *testing.T) {
	thresholds := ThresholdLevels{OK: 2, Good: 4, Excellent: 8}

	tests := []struct {
		endpointCount int
		wantPoints    int
	}{
		{0, 0},                   // No endpoints
		{1, 2},                   // One endpoint
		{2, 4},                   // OK threshold
		{3, 4},                   // Still OK
		{4, 5},                   // Good threshold
		{7, 5},                   // Still good
		{8, MaxAPIPoints},        // Excellent threshold
		{20, MaxAPIPoints},       // Well above excellent
	}

	for _, tt := range tests {
		got := DecideAPIIntegrationPoints(tt.endpointCount, thresholds)
		if got != tt.wantPoints {
			t.Errorf("DecideAPIIntegrationPoints(%d, thresholds) = %d, want %d", tt.endpointCount, got, tt.wantPoints)
		}
	}
}

func TestDecideRoutingPoints(t *testing.T) {
	tests := []struct {
		routeCount int
		wantPoints float64
	}{
		{0, 0.0},
		{1, 0.5},                    // Basic routing
		{2, 0.5},                    // Still basic
		{3, 1.0},                    // Moderate routing
		{4, 1.0},                    // Still moderate
		{5, MaxRoutingPoints},       // Advanced routing
		{10, MaxRoutingPoints},      // Well above advanced
	}

	for _, tt := range tests {
		got := DecideRoutingPoints(tt.routeCount)
		if got != tt.wantPoints {
			t.Errorf("DecideRoutingPoints(%d) = %f, want %f", tt.routeCount, got, tt.wantPoints)
		}
	}
}

func TestDecideVolumePoints(t *testing.T) {
	thresholds := ThresholdLevels{OK: 300, Good: 600, Excellent: 1200}

	tests := []struct {
		totalLOC   int
		wantPoints float64
	}{
		{0, 0.0},
		{99, 0.0},                   // Below minimum
		{100, 0.5},                  // Minimum threshold
		{299, 0.5},                  // Still minimum
		{300, 1.5},                  // OK threshold
		{599, 1.5},                  // Still OK
		{600, 2.0},                  // Good threshold
		{1199, 2.0},                 // Still good
		{1200, MaxVolumePoints},     // Excellent threshold
		{5000, MaxVolumePoints},     // Well above excellent
	}

	for _, tt := range tests {
		got := DecideVolumePoints(tt.totalLOC, thresholds)
		if got != tt.wantPoints {
			t.Errorf("DecideVolumePoints(%d, thresholds) = %f, want %f", tt.totalLOC, got, tt.wantPoints)
		}
	}
}

// =============================================================================
// THRESHOLD LEVEL DECISION TESTS
// =============================================================================
// [REQ:SCS-CFG-002] Test threshold-based decisions

func TestDecideThresholdLevel(t *testing.T) {
	thresholds := ThresholdLevels{OK: 10, Good: 15, Excellent: 25}

	tests := []struct {
		count     int
		wantLevel string
	}{
		{0, "below"},
		{9, "below"},
		{10, "ok"},
		{14, "ok"},
		{15, "good"},
		{24, "good"},
		{25, "excellent"},
		{100, "excellent"},
	}

	for _, tt := range tests {
		got := DecideThresholdLevel(tt.count, thresholds)
		if got != tt.wantLevel {
			t.Errorf("DecideThresholdLevel(%d, thresholds) = %s, want %s", tt.count, got, tt.wantLevel)
		}
	}
}

func TestIsAboveMinimumThreshold(t *testing.T) {
	thresholds := ThresholdLevels{OK: 10, Good: 15, Excellent: 25}

	tests := []struct {
		count int
		want  bool
	}{
		{9, false},
		{10, true},
		{15, true},
		{25, true},
	}

	for _, tt := range tests {
		got := IsAboveMinimumThreshold(tt.count, thresholds)
		if got != tt.want {
			t.Errorf("IsAboveMinimumThreshold(%d, thresholds) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

func TestIsExcellent(t *testing.T) {
	thresholds := ThresholdLevels{OK: 10, Good: 15, Excellent: 25}

	tests := []struct {
		count int
		want  bool
	}{
		{24, false},
		{25, true},
		{50, true},
	}

	for _, tt := range tests {
		got := IsExcellent(tt.count, thresholds)
		if got != tt.want {
			t.Errorf("IsExcellent(%d, thresholds) = %v, want %v", tt.count, got, tt.want)
		}
	}
}

// =============================================================================
// COVERAGE RATIO DECISION TESTS
// =============================================================================
// [REQ:SCS-CORE-001B] Test coverage dimension decisions

func TestDecideTestCoveragePoints(t *testing.T) {
	tests := []struct {
		testCount  int
		reqCount   int
		wantPoints int
		wantRatio  float64
	}{
		{0, 0, 0, 0.0},         // Zero requirements
		{0, 10, 0, 0.0},        // No tests
		{10, 10, 4, 1.0},       // 1:1 ratio = 50% of max points
		{15, 10, 6, 1.5},       // 1.5:1 ratio = 75% of max points
		{20, 10, MaxTestCoveragePoints, 2.0}, // 2:1 optimal ratio
		{40, 10, MaxTestCoveragePoints, 4.0}, // Above optimal (capped at 2.0)
	}

	for _, tt := range tests {
		gotPoints, gotRatio := DecideTestCoveragePoints(tt.testCount, tt.reqCount)
		if gotPoints != tt.wantPoints {
			t.Errorf("DecideTestCoveragePoints(%d, %d) points = %d, want %d", tt.testCount, tt.reqCount, gotPoints, tt.wantPoints)
		}
		if gotRatio != tt.wantRatio {
			t.Errorf("DecideTestCoveragePoints(%d, %d) ratio = %f, want %f", tt.testCount, tt.reqCount, gotRatio, tt.wantRatio)
		}
	}
}

func TestDecideDepthPoints(t *testing.T) {
	tests := []struct {
		avgDepth   float64
		wantPoints int
	}{
		{0.0, 0},
		{1.0, 2},    // 1/3 of optimal = ~2 points
		{1.5, 4},    // Half of optimal = ~4 points
		{3.0, MaxDepthPoints}, // Optimal depth
		{5.0, MaxDepthPoints}, // Above optimal (capped)
	}

	for _, tt := range tests {
		got := DecideDepthPoints(tt.avgDepth)
		if got != tt.wantPoints {
			t.Errorf("DecideDepthPoints(%f) = %d, want %d", tt.avgDepth, got, tt.wantPoints)
		}
	}
}

// =============================================================================
// QUALITY DIMENSION DECISION TESTS
// =============================================================================
// [REQ:SCS-CORE-001A] Test quality dimension decisions

func TestDecidePassRatePoints(t *testing.T) {
	tests := []struct {
		passing    int
		total      int
		maxPoints  int
		wantPoints int
		wantRate   float64
	}{
		{0, 0, 20, 0, 0.0},       // Zero total
		{0, 10, 20, 0, 0.0},      // Zero passing
		{5, 10, 20, 10, 0.5},     // 50% pass rate
		{10, 10, 20, 20, 1.0},    // 100% pass rate
		{9, 10, 15, 14, 0.9},     // 90% pass rate with 15 max
	}

	for _, tt := range tests {
		gotPoints, gotRate := DecidePassRatePoints(tt.passing, tt.total, tt.maxPoints)
		if gotPoints != tt.wantPoints {
			t.Errorf("DecidePassRatePoints(%d, %d, %d) points = %d, want %d", tt.passing, tt.total, tt.maxPoints, gotPoints, tt.wantPoints)
		}
		if gotRate != tt.wantRate {
			t.Errorf("DecidePassRatePoints(%d, %d, %d) rate = %f, want %f", tt.passing, tt.total, tt.maxPoints, gotRate, tt.wantRate)
		}
	}
}

// =============================================================================
// QUANTITY DIMENSION DECISION TESTS
// =============================================================================
// [REQ:SCS-CORE-001C] Test quantity dimension decisions

func TestDecideQuantityPoints(t *testing.T) {
	tests := []struct {
		count         int
		goodThreshold int
		maxPoints     int
		wantPoints    int
	}{
		{0, 0, 4, 0},        // Zero threshold
		{0, 15, 4, 0},       // Zero count
		{8, 15, 4, 2},       // ~53% of threshold
		{15, 15, 4, 4},      // 100% of threshold
		{30, 15, 4, 4},      // Above threshold (capped)
	}

	for _, tt := range tests {
		got := DecideQuantityPoints(tt.count, tt.goodThreshold, tt.maxPoints)
		if got != tt.wantPoints {
			t.Errorf("DecideQuantityPoints(%d, %d, %d) = %d, want %d", tt.count, tt.goodThreshold, tt.maxPoints, got, tt.wantPoints)
		}
	}
}

// =============================================================================
// RECOMMENDATION DECISION TESTS
// =============================================================================
// [REQ:SCS-ANALYSIS-002] Test recommendation generation decisions

func TestShouldRecommendPassRateImprovement(t *testing.T) {
	tests := []struct {
		rate float64
		want bool
	}{
		{0.89, true},            // Below threshold
		{0.90, false},           // At threshold
		{0.95, false},           // Above threshold
		{1.0, false},            // Perfect
	}

	for _, tt := range tests {
		if got := ShouldRecommendTestPassRateImprovement(tt.rate); got != tt.want {
			t.Errorf("ShouldRecommendTestPassRateImprovement(%f) = %v, want %v", tt.rate, got, tt.want)
		}
		if got := ShouldRecommendRequirementPassRateImprovement(tt.rate); got != tt.want {
			t.Errorf("ShouldRecommendRequirementPassRateImprovement(%f) = %v, want %v", tt.rate, got, tt.want)
		}
		if got := ShouldRecommendTargetPassRateImprovement(tt.rate); got != tt.want {
			t.Errorf("ShouldRecommendTargetPassRateImprovement(%f) = %v, want %v", tt.rate, got, tt.want)
		}
	}
}

func TestShouldRecommendTemplateReplacement(t *testing.T) {
	if got := ShouldRecommendTemplateReplacement(true); !got {
		t.Error("ShouldRecommendTemplateReplacement(true) should be true")
	}
	if got := ShouldRecommendTemplateReplacement(false); got {
		t.Error("ShouldRecommendTemplateReplacement(false) should be false")
	}
}

func TestShouldRecommendMoreUIFiles(t *testing.T) {
	tests := []struct {
		threshold string
		want      bool
	}{
		{"below", true},
		{"ok", false},
		{"good", false},
		{"excellent", false},
	}

	for _, tt := range tests {
		if got := ShouldRecommendMoreUIFiles(tt.threshold); got != tt.want {
			t.Errorf("ShouldRecommendMoreUIFiles(%s) = %v, want %v", tt.threshold, got, tt.want)
		}
	}
}

func TestShouldRecommendAPIIntegration(t *testing.T) {
	if got := ShouldRecommendAPIIntegration(0); !got {
		t.Error("ShouldRecommendAPIIntegration(0) should be true")
	}
	if got := ShouldRecommendAPIIntegration(1); got {
		t.Error("ShouldRecommendAPIIntegration(1) should be false")
	}
}

func TestShouldRecommendMoreTests(t *testing.T) {
	tests := []struct {
		threshold string
		want      bool
	}{
		{"below", true},
		{"ok", true},
		{"good", false},
		{"excellent", false},
	}

	for _, tt := range tests {
		if got := ShouldRecommendMoreTests(tt.threshold); got != tt.want {
			t.Errorf("ShouldRecommendMoreTests(%s) = %v, want %v", tt.threshold, got, tt.want)
		}
	}
}

func TestShouldRecommendBetterCoverage(t *testing.T) {
	tests := []struct {
		ratio float64
		want  bool
	}{
		{1.0, true},
		{1.9, true},
		{2.0, false},
		{3.0, false},
	}

	for _, tt := range tests {
		if got := ShouldRecommendBetterCoverage(tt.ratio); got != tt.want {
			t.Errorf("ShouldRecommendBetterCoverage(%f) = %v, want %v", tt.ratio, got, tt.want)
		}
	}
}

// =============================================================================
// CONSTANTS TESTS
// =============================================================================
// Verify constants are correctly defined

func TestClassificationThresholdConstants(t *testing.T) {
	// Verify thresholds are in descending order and non-overlapping
	if ProductionReadyThreshold <= NearlyReadyThreshold {
		t.Errorf("ProductionReadyThreshold (%d) should be > NearlyReadyThreshold (%d)", ProductionReadyThreshold, NearlyReadyThreshold)
	}
	if NearlyReadyThreshold <= MostlyCompleteThreshold {
		t.Errorf("NearlyReadyThreshold (%d) should be > MostlyCompleteThreshold (%d)", NearlyReadyThreshold, MostlyCompleteThreshold)
	}
	if MostlyCompleteThreshold <= FunctionalIncompleteThreshold {
		t.Errorf("MostlyCompleteThreshold (%d) should be > FunctionalIncompleteThreshold (%d)", MostlyCompleteThreshold, FunctionalIncompleteThreshold)
	}
	if FunctionalIncompleteThreshold <= FoundationLaidThreshold {
		t.Errorf("FunctionalIncompleteThreshold (%d) should be > FoundationLaidThreshold (%d)", FunctionalIncompleteThreshold, FoundationLaidThreshold)
	}
	if FoundationLaidThreshold < 0 {
		t.Errorf("FoundationLaidThreshold (%d) should be >= 0", FoundationLaidThreshold)
	}
}

func TestUIPointConstants(t *testing.T) {
	// Verify UI points sum to approximately 25 (the UI dimension max)
	expectedMax := 25.0
	totalMax := float64(TemplateUIPoints) + float64(MaxComponentPoints) + float64(MaxAPIPoints) + MaxRoutingPoints + MaxVolumePoints

	if totalMax != expectedMax {
		t.Errorf("UI point constants should sum to %f, got %f", expectedMax, totalMax)
	}
}

func TestCoverageConstants(t *testing.T) {
	// Verify coverage points sum to 15 (the coverage dimension max)
	expectedMax := 15
	totalMax := MaxTestCoveragePoints + MaxDepthPoints

	if totalMax != expectedMax {
		t.Errorf("Coverage point constants should sum to %d, got %d", expectedMax, totalMax)
	}
}

func TestQualityConstants(t *testing.T) {
	// Verify quality points sum to 50 (the quality dimension max)
	expectedMax := 50
	totalMax := MaxRequirementPassRatePoints + MaxTargetPassRatePoints + MaxTestPassRatePoints

	if totalMax != expectedMax {
		t.Errorf("Quality point constants should sum to %d, got %d", expectedMax, totalMax)
	}
}

func TestQuantityConstants(t *testing.T) {
	// Verify quantity points sum to 10 (the quantity dimension max)
	expectedMax := 10
	totalMax := MaxRequirementQuantityPoints + MaxTargetQuantityPoints + MaxTestQuantityPoints

	if totalMax != expectedMax {
		t.Errorf("Quantity point constants should sum to %d, got %d", expectedMax, totalMax)
	}
}
