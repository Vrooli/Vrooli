// Package scoring provides completeness score calculation for Vrooli scenarios.
// It implements the same algorithm as scripts/scenarios/lib/completeness.js
package scoring

// ScoreBreakdown represents the complete score calculation result
type ScoreBreakdown struct {
	BaseScore          int              `json:"base_score"`
	ValidationPenalty  int              `json:"validation_penalty"`
	Score              int              `json:"score"`
	Classification     string           `json:"classification"`
	ClassificationDesc string           `json:"classification_description"`
	Quality            QualityScore     `json:"quality"`
	Coverage           CoverageScore    `json:"coverage"`
	Quantity           QuantityScore    `json:"quantity"`
	UI                 UIScore          `json:"ui"`
	Warnings           []Warning        `json:"warnings,omitempty"`
	Recommendations    []Recommendation `json:"recommendations,omitempty"`
}

// QualityScore represents the quality dimension (50 points max)
type QualityScore struct {
	Score               int       `json:"score"`
	Max                 int       `json:"max"`
	Disabled            bool      `json:"disabled,omitempty"`
	RequirementPassRate PassRate  `json:"requirement_pass_rate"`
	TargetPassRate      PassRate  `json:"target_pass_rate"`
	TestPassRate        PassRate  `json:"test_pass_rate"`
}

// PassRate represents a pass/fail rate metric
type PassRate struct {
	Passing int     `json:"passing"`
	Total   int     `json:"total"`
	Rate    float64 `json:"rate"`
	Points  int     `json:"points"`
}

// CoverageScore represents the coverage dimension (15 points max)
type CoverageScore struct {
	Score             int              `json:"score"`
	Max               int              `json:"max"`
	Disabled          bool             `json:"disabled,omitempty"`
	TestCoverageRatio CoverageRatio    `json:"test_coverage_ratio"`
	DepthScore        DepthScoreDetail `json:"depth_score"`
}

// CoverageRatio represents the test-to-requirement ratio
type CoverageRatio struct {
	Ratio  float64 `json:"ratio"`
	Points int     `json:"points"`
}

// DepthScoreDetail represents requirement depth scoring
type DepthScoreDetail struct {
	AvgDepth float64 `json:"avg_depth"`
	Points   int     `json:"points"`
}

// QuantityScore represents the quantity dimension (10 points max)
type QuantityScore struct {
	Score        int            `json:"score"`
	Max          int            `json:"max"`
	Disabled     bool           `json:"disabled,omitempty"`
	Requirements QuantityMetric `json:"requirements"`
	Targets      QuantityMetric `json:"targets"`
	Tests        QuantityMetric `json:"tests"`
}

// QuantityMetric represents a count-based metric with threshold
type QuantityMetric struct {
	Count     int    `json:"count"`
	Threshold string `json:"threshold"`
	Points    int    `json:"points"`
}

// UIScore represents the UI dimension (25 points max)
type UIScore struct {
	Score               int                 `json:"score"`
	Max                 int                 `json:"max"`
	Disabled            bool                `json:"disabled,omitempty"`
	TemplateCheck       TemplateCheckResult `json:"template_check"`
	ComponentComplexity ComponentComplexity `json:"component_complexity"`
	APIIntegration      APIIntegration      `json:"api_integration"`
	Routing             RoutingScore        `json:"routing"`
	CodeVolume          CodeVolume          `json:"code_volume"`
}

// TemplateCheckResult represents template detection
type TemplateCheckResult struct {
	IsTemplate bool `json:"is_template"`
	Penalty    int  `json:"penalty"`
	Points     int  `json:"points"`
}

// ComponentComplexity represents UI component metrics
type ComponentComplexity struct {
	FileCount      int    `json:"file_count"`
	ComponentCount int    `json:"component_count,omitempty"`
	PageCount      int    `json:"page_count,omitempty"`
	Threshold      string `json:"threshold"`
	Points         int    `json:"points"`
}

// APIIntegration represents API endpoint usage in UI
type APIIntegration struct {
	EndpointCount  int `json:"endpoint_count"`
	TotalEndpoints int `json:"total_endpoints,omitempty"`
	Points         int `json:"points"`
}

// RoutingScore represents routing complexity
type RoutingScore struct {
	HasRouting bool    `json:"has_routing"`
	RouteCount int     `json:"route_count"`
	Points     float64 `json:"points"`
}

// CodeVolume represents lines of code metrics
type CodeVolume struct {
	TotalLOC int     `json:"total_loc"`
	Points   float64 `json:"points"`
}

// Warning represents a scoring warning
type Warning struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Action  string `json:"action,omitempty"`
}

// Recommendation represents an actionable improvement suggestion
type Recommendation struct {
	Priority    int    `json:"priority"`
	Description string `json:"description"`
	Impact      int    `json:"impact,omitempty"`
}

// Metrics represents the collected metrics for scoring
type Metrics struct {
	Scenario     string            `json:"scenario"`
	Category     string            `json:"category"`
	Requirements MetricCounts      `json:"requirements"`
	Targets      MetricCounts      `json:"targets"`
	Tests        MetricCounts      `json:"tests"`
	UI           *UIMetrics        `json:"ui,omitempty"`
	LastTestRun  string            `json:"last_test_run,omitempty"`
	Requirements_ []RequirementTree `json:"-"` // For depth calculation
}

// MetricCounts represents passing/total counts
type MetricCounts struct {
	Total   int `json:"total"`
	Passing int `json:"passing"`
}

// UIMetrics represents raw UI metrics from file analysis
type UIMetrics struct {
	IsTemplate      bool `json:"is_template"`
	FileCount       int  `json:"file_count"`
	ComponentCount  int  `json:"component_count"`
	PageCount       int  `json:"page_count"`
	APIEndpoints    int  `json:"api_endpoints"`
	APIBeyondHealth int  `json:"api_beyond_health"`
	HasRouting      bool `json:"has_routing"`
	RouteCount      int  `json:"route_count"`
	TotalLOC        int  `json:"total_loc"`
}

// RequirementTree represents a requirement with children for depth calculation
type RequirementTree struct {
	ID       string            `json:"id"`
	Children []RequirementTree `json:"children,omitempty"`
}

// ThresholdConfig holds category-specific thresholds
type ThresholdConfig struct {
	Requirements ThresholdLevels   `json:"requirements"`
	Targets      ThresholdLevels   `json:"targets"`
	Tests        ThresholdLevels   `json:"tests"`
	UI           UIThresholdConfig `json:"ui"`
}

// ThresholdLevels defines ok/good/excellent thresholds
type ThresholdLevels struct {
	OK        int `json:"ok"`
	Good      int `json:"good"`
	Excellent int `json:"excellent"`
}

// UIThresholdConfig defines UI-specific thresholds
type UIThresholdConfig struct {
	FileCount    ThresholdLevels `json:"file_count"`
	TotalLOC     ThresholdLevels `json:"total_loc"`
	APIEndpoints ThresholdLevels `json:"api_endpoints"`
}

// ValidationQualityAnalysis holds penalty information
type ValidationQualityAnalysis struct {
	TotalPenalty int             `json:"total_penalty"`
	Penalties    []PenaltyDetail `json:"penalties,omitempty"`
}

// PenaltyDetail represents a single penalty
type PenaltyDetail struct {
	Type    string `json:"type"`
	Points  int    `json:"points"`
	Message string `json:"message"`
}

// ScoringOptions controls which dimensions are enabled and their weights
// This is used to wire configuration into the scoring logic
type ScoringOptions struct {
	// Enabled flags for each dimension
	QualityEnabled  bool
	CoverageEnabled bool
	QuantityEnabled bool
	UIEnabled       bool

	// Weights for each dimension (should sum to 100 when accounting for enabled dimensions)
	QualityWeight  int
	CoverageWeight int
	QuantityWeight int
	UIWeight       int
}

// DefaultScoringOptions returns options with all dimensions enabled at default weights
func DefaultScoringOptions() *ScoringOptions {
	return &ScoringOptions{
		QualityEnabled:  true,
		CoverageEnabled: true,
		QuantityEnabled: true,
		UIEnabled:       true,
		QualityWeight:   50,
		CoverageWeight:  15,
		QuantityWeight:  10,
		UIWeight:        25,
	}
}
