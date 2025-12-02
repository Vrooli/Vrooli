package models

type ScoreBreakdown struct {
	BaseScore         int           `json:"base_score"`
	ValidationPenalty int           `json:"validation_penalty"`
	Score             int           `json:"score"`
	Classification    string        `json:"classification"`
	Quality           QualityScore  `json:"quality"`
	Coverage          CoverageScore `json:"coverage"`
	Quantity          QuantityScore `json:"quantity"`
	UI                UIScore       `json:"ui"`
}

type QualityScore struct {
	Score               int      `json:"score"`
	Max                 int      `json:"max"`
	RequirementPassRate PassRate `json:"requirement_pass_rate"`
	TargetPassRate      PassRate `json:"target_pass_rate"`
	TestPassRate        PassRate `json:"test_pass_rate"`
}

type PassRate struct {
	Passing int     `json:"passing"`
	Total   int     `json:"total"`
	Rate    float64 `json:"rate"`
	Points  int     `json:"points"`
}

type CoverageScore struct {
	Score             int              `json:"score"`
	Max               int              `json:"max"`
	TestCoverageRatio CoverageRatio    `json:"test_coverage_ratio"`
	DepthScore        DepthScoreDetail `json:"depth_score"`
}

type CoverageRatio struct {
	Ratio  float64 `json:"ratio"`
	Points int     `json:"points"`
}

type DepthScoreDetail struct {
	AvgDepth float64 `json:"avg_depth"`
	Points   int     `json:"points"`
}

type QuantityScore struct {
	Score        int            `json:"score"`
	Max          int            `json:"max"`
	Requirements QuantityMetric `json:"requirements"`
	Targets      QuantityMetric `json:"targets"`
	Tests        QuantityMetric `json:"tests"`
}

type QuantityMetric struct {
	Count     int    `json:"count"`
	Threshold string `json:"threshold"`
	Points    int    `json:"points"`
}

type UIScore struct {
	Score               int                 `json:"score"`
	Max                 int                 `json:"max"`
	TemplateCheck       TemplateCheckResult `json:"template_check"`
	ComponentComplexity ComponentComplexity `json:"component_complexity"`
	APIIntegration      APIIntegration      `json:"api_integration"`
	Routing             RoutingScore        `json:"routing"`
	CodeVolume          CodeVolume          `json:"code_volume"`
}

type TemplateCheckResult struct {
	IsTemplate bool `json:"is_template"`
	Penalty    int  `json:"penalty"`
	Points     int  `json:"points"`
}

type ComponentComplexity struct {
	FileCount int    `json:"file_count"`
	Threshold string `json:"threshold"`
	Points    int    `json:"points"`
}

type APIIntegration struct {
	EndpointCount int `json:"endpoint_count"`
	Points        int `json:"points"`
}

type RoutingScore struct {
	HasRouting bool    `json:"has_routing"`
	RouteCount int     `json:"route_count"`
	Points     float64 `json:"points"`
}

type CodeVolume struct {
	TotalLOC int     `json:"total_loc"`
	Points   float64 `json:"points"`
}

type Recommendation struct {
	Message string  `json:"message"`
	Impact  float64 `json:"impact"`
}

type ScoreResponse struct {
	Scenario            string                    `json:"scenario"`
	Category            string                    `json:"category"`
	Score               float64                   `json:"score"`
	BaseScore           float64                   `json:"base_score"`
	ValidationPenalty   float64                   `json:"validation_penalty"`
	Classification      string                    `json:"classification"`
	Breakdown           ScoreBreakdown            `json:"breakdown"`
	Metrics             map[string]interface{}    `json:"metrics"`
	ValidationAnalysis  ValidationQualityAnalysis `json:"validation_analysis"`
	Recommendations     []Recommendation          `json:"recommendations"`
	PartialResult       map[string]interface{}    `json:"partial_result"`
	CalculatedTimestamp string                    `json:"calculated_at"`
}

type ValidationQualityAnalysis struct {
	HasIssues       bool                   `json:"has_issues"`
	IssueCount      int                    `json:"issue_count"`
	Issues          []ValidationIssue      `json:"issues"`
	Patterns        map[string]interface{} `json:"patterns,omitempty"`
	TotalPenalty    int                    `json:"total_penalty"`
	OverallSeverity string                 `json:"overall_severity"`
}

type ValidationIssue struct {
	Type           string                `json:"type"`
	Severity       string                `json:"severity"`
	Penalty        int                   `json:"penalty"`
	Message        string                `json:"message"`
	Recommendation string                `json:"recommendation"`
	WhyItMatters   string                `json:"why_it_matters"`
	Description    string                `json:"description,omitempty"`
	Count          int                   `json:"count,omitempty"`
	Total          int                   `json:"total,omitempty"`
	Ratio          float64               `json:"ratio,omitempty"`
	ValidSources   []string              `json:"valid_sources,omitempty"`
	InvalidPaths   []InvalidPathInfo     `json:"invalid_paths,omitempty"`
	AffectedReqs   []AffectedRequirement `json:"affected_requirements,omitempty"`
	WorstOffender  *MonolithicTestInfo   `json:"worst_offender,omitempty"`
	Violations     int                   `json:"violations,omitempty"`
}

type InvalidPathInfo struct {
	Path           string   `json:"path"`
	RequirementIDs []string `json:"requirement_ids"`
}

type AffectedRequirement struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Priority      string   `json:"priority"`
	CurrentLayers []string `json:"current_layers"`
	NeededLayers  []string `json:"needed_layers"`
}

type MonolithicTestInfo struct {
	TestRef  string   `json:"test_ref"`
	SharedBy []string `json:"shared_by"`
	Count    int      `json:"count"`
	Severity string   `json:"severity"`
}
