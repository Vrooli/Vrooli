// Package validators provides validation quality analysis for scenario completeness scoring.
// It detects testing anti-patterns and gaming behaviors that inflate metrics without genuine validation.
package validators

// ValidationQualityAnalysis holds the complete validation analysis result
type ValidationQualityAnalysis struct {
	HasIssues       bool                   `json:"has_issues"`
	IssueCount      int                    `json:"issue_count"`
	Issues          []ValidationIssue      `json:"issues"`
	Patterns        map[string]interface{} `json:"patterns,omitempty"`
	TotalPenalty    int                    `json:"total_penalty"`
	OverallSeverity string                 `json:"overall_severity"`
}

// ValidationIssue represents a single validation quality issue
type ValidationIssue struct {
	Type            string                 `json:"type"`
	Severity        string                 `json:"severity"`
	Detected        bool                   `json:"detected"`
	Penalty         int                    `json:"penalty"`
	Message         string                 `json:"message"`
	Recommendation  string                 `json:"recommendation"`
	WhyItMatters    string                 `json:"why_it_matters"`
	Description     string                 `json:"description,omitempty"`
	Count           int                    `json:"count,omitempty"`
	Total           int                    `json:"total,omitempty"`
	Ratio           float64                `json:"ratio,omitempty"`
	ValidSources    []string               `json:"valid_sources,omitempty"`
	InvalidPaths    []InvalidPathInfo      `json:"invalid_paths,omitempty"`
	AffectedReqs    []AffectedRequirement  `json:"affected_requirements,omitempty"`
	WorstOffender   *MonolithicTestInfo    `json:"worst_offender,omitempty"`
	Violations      int                    `json:"violations,omitempty"`
	ExtraData       map[string]interface{} `json:"extra_data,omitempty"`
}

// InvalidPathInfo represents a path referenced by requirements that is invalid
type InvalidPathInfo struct {
	Path           string   `json:"path"`
	RequirementIDs []string `json:"requirement_ids"`
}

// AffectedRequirement represents a requirement affected by a validation issue
type AffectedRequirement struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Priority      string   `json:"priority"`
	CurrentLayers []string `json:"current_layers"`
	NeededLayers  []string `json:"needed_layers"`
}

// MonolithicTestInfo represents info about a monolithic test file
type MonolithicTestInfo struct {
	TestRef  string   `json:"test_ref"`
	SharedBy []string `json:"shared_by"`
	Count    int      `json:"count"`
	Severity string   `json:"severity"`
}

// Requirement represents a requirement for validation analysis
type Requirement struct {
	ID                  string       `json:"id"`
	Title               string       `json:"title,omitempty"`
	Status              string       `json:"status,omitempty"`
	Priority            string       `json:"priority,omitempty"`
	Category            string       `json:"category,omitempty"`
	PRDRef              string       `json:"prd_ref,omitempty"`
	OperationalTargetID string       `json:"operational_target_id,omitempty"`
	Children            []string     `json:"children,omitempty"`
	Validation          []Validation `json:"validation,omitempty"`
}

// Validation represents a validation reference
type Validation struct {
	Type       string `json:"type"`
	Ref        string `json:"ref,omitempty"`
	WorkflowID string `json:"workflow_id,omitempty"`
	Status     string `json:"status,omitempty"`
}

// OperationalTarget represents an operational target
type OperationalTarget struct {
	ID           string   `json:"id"`
	Title        string   `json:"title,omitempty"`
	Priority     string   `json:"priority,omitempty"`
	Requirements []string `json:"requirements,omitempty"`
}

// TestRefUsageAnalysis holds analysis of test ref usage across requirements
type TestRefUsageAnalysis struct {
	TotalRefs           int                   `json:"total_refs"`
	TotalRequirements   int                   `json:"total_requirements"`
	AverageReqsPerRef   float64               `json:"average_reqs_per_ref"`
	Violations          []MonolithicTestInfo  `json:"violations"`
	DuplicateRatio      float64               `json:"duplicate_ratio"`
}

// TargetGroupingAnalysis holds analysis of operational target grouping
type TargetGroupingAnalysis struct {
	TotalTargets               int                      `json:"total_targets"`
	OneToOneCount              int                      `json:"one_to_one_count"`
	OneToOneRatio              float64                  `json:"one_to_one_ratio"`
	AcceptableRatio            float64                  `json:"acceptable_ratio"`
	AverageRequirementsPerTarget float64                `json:"average_requirements_per_target"`
	Violations                 []TargetGroupingViolation `json:"violations"`
}

// TargetGroupingViolation represents a target grouping violation
type TargetGroupingViolation struct {
	Type            string   `json:"type"`
	CurrentRatio    float64  `json:"current_ratio"`
	AcceptableRatio float64  `json:"acceptable_ratio"`
	OneToOneCount   int      `json:"one_to_one_count"`
	TotalTargets    int      `json:"total_targets"`
	Message         string   `json:"message"`
	Targets         []TargetMapping `json:"targets"`
}

// TargetMapping represents a target-to-requirement mapping
type TargetMapping struct {
	Target      string `json:"target"`
	Requirement string `json:"requirement"`
}

// TestFileQuality holds test file quality analysis results
type TestFileQuality struct {
	Exists            bool    `json:"exists"`
	LOC               int     `json:"loc"`
	HasAssertions     bool    `json:"has_assertions"`
	HasTestFunctions  bool    `json:"has_test_functions"`
	TestFunctionCount int     `json:"test_function_count"`
	AssertionCount    int     `json:"assertion_count"`
	AssertionDensity  float64 `json:"assertion_density"`
	IsMeaningful      bool    `json:"is_meaningful"`
	QualityScore      int     `json:"quality_score"`
	Reason            string  `json:"reason"`
}

// PlaybookQuality holds e2e playbook quality analysis results
type PlaybookQuality struct {
	Exists       bool   `json:"exists"`
	StepCount    int    `json:"step_count"`
	HasActions   bool   `json:"has_actions"`
	FileSize     int    `json:"file_size"`
	IsMeaningful bool   `json:"is_meaningful"`
	Reason       string `json:"reason"`
}

// ValidationLayerAnalysis holds validation layer analysis results
type ValidationLayerAnalysis struct {
	AutomatedLayers []string `json:"automated_layers"`
	HasManual       bool     `json:"has_manual"`
}

// PenaltyConfig holds penalty configuration for an issue type
type PenaltyConfig struct {
	Description       string  `json:"description"`
	BasePenalty       int     `json:"base_penalty,omitempty"`
	Multiplier        int     `json:"multiplier,omitempty"`
	PerViolation      int     `json:"per_violation,omitempty"`
	PerFile           int     `json:"per_file,omitempty"`
	RatioMultiplier   int     `json:"ratio_multiplier,omitempty"`
	PerCompleteReq    int     `json:"per_complete_requirement,omitempty"`
	MaxCompleteCount  int     `json:"max_complete_count,omitempty"`
	MaxPenalty        int     `json:"max_penalty"`
	Severity          string  `json:"severity,omitempty"`
	SeverityThreshold float64 `json:"severity_threshold,omitempty"`
}

// ValidationConfig holds validation thresholds and ratios
type ValidationConfig struct {
	MinLayers struct {
		P0Requirements int `json:"p0_requirements"`
		P1Requirements int `json:"p1_requirements"`
		P2Requirements int `json:"p2_requirements"`
		P3Requirements int `json:"p3_requirements"`
	} `json:"min_layers"`
	AcceptableRatios struct {
		ManualValidation          float64 `json:"manual_validation"`
		OneToOneTargetMapping     float64 `json:"one_to_one_target_mapping"`
		TestToRequirementTolerance float64 `json:"test_to_requirement_tolerance"`
	} `json:"acceptable_ratios"`
	CompleteWithManualThreshold int `json:"complete_with_manual_threshold"`
	MonolithicTestThreshold     int `json:"monolithic_test_threshold"`
	MonolithicTestHighSeverity  int `json:"monolithic_test_high_severity"`
	TestQuality                 struct {
		MinimumLOC             int     `json:"minimum_loc"`
		MinimumTestFunctions   int     `json:"minimum_test_functions"`
		MinimumAssertionDensity float64 `json:"minimum_assertion_density"`
		MinimumQualityScore    int     `json:"minimum_quality_score"`
	} `json:"test_quality"`
	PlaybookQuality struct {
		MinimumFileSize int `json:"minimum_file_size"`
	} `json:"playbook_quality"`
}

// DefaultPenaltyConfigs returns the default penalty configurations
func DefaultPenaltyConfigs() map[string]PenaltyConfig {
	return map[string]PenaltyConfig{
		"insufficient_test_coverage": {
			Description: "Suspicious 1:1 test-to-requirement ratio",
			BasePenalty: 5,
			MaxPenalty:  5,
			Severity:    "medium",
		},
		"invalid_test_location": {
			Description:       "Requirements reference unsupported test/ directories",
			Multiplier:        25,
			MaxPenalty:        25,
			SeverityThreshold: 0.5,
		},
		"monolithic_test_files": {
			Description:       "Test files validate ≥4 requirements each",
			PerViolation:      2,
			MaxPenalty:        15,
			SeverityThreshold: 6,
		},
		"ungrouped_operational_targets": {
			Description:       "Excessive 1:1 target-to-requirement mappings",
			Multiplier:        10,
			MaxPenalty:        10,
			SeverityThreshold: 0.5,
		},
		"insufficient_validation_layers": {
			Description: "Critical requirements lack multi-layer validation",
			Multiplier:  20,
			MaxPenalty:  20,
			Severity:    "high",
		},
		"superficial_test_implementation": {
			Description: "Test files appear superficial",
			PerFile:     1,
			MaxPenalty:  10,
			Severity:    "medium",
		},
		"missing_test_automation": {
			Description:      "Excessive manual validation usage",
			RatioMultiplier:  10,
			PerCompleteReq:   1,
			MaxCompleteCount: 5,
			MaxPenalty:       15,
			SeverityThreshold: 10,
		},
	}
}

// DefaultValidationConfig returns the default validation configuration
func DefaultValidationConfig() ValidationConfig {
	cfg := ValidationConfig{}
	cfg.MinLayers.P0Requirements = 2
	cfg.MinLayers.P1Requirements = 2
	cfg.MinLayers.P2Requirements = 1
	cfg.MinLayers.P3Requirements = 1
	cfg.AcceptableRatios.ManualValidation = 0.10
	cfg.AcceptableRatios.OneToOneTargetMapping = 0.15
	cfg.AcceptableRatios.TestToRequirementTolerance = 0.10
	cfg.CompleteWithManualThreshold = 5
	cfg.MonolithicTestThreshold = 4
	cfg.MonolithicTestHighSeverity = 6
	cfg.TestQuality.MinimumLOC = 20
	cfg.TestQuality.MinimumTestFunctions = 3
	cfg.TestQuality.MinimumAssertionDensity = 0.1
	cfg.TestQuality.MinimumQualityScore = 4
	cfg.PlaybookQuality.MinimumFileSize = 100
	return cfg
}
