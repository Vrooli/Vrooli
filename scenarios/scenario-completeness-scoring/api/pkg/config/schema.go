package config

// ConfigSchema describes the scoring configuration in a UI-friendly way.
// The UI renders sections and fields from this schema so configuration is
// self-explanatory and fully toggleable at every level.

type ConfigSchema struct {
	Version  string          `json:"version"`
	Sections []SectionSchema `json:"sections"`
}

type SectionSchema struct {
	Key         string        `json:"key"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	EnabledPath string        `json:"enabled_path,omitempty"`
	WeightPath  string        `json:"weight_path,omitempty"`
	Fields      []FieldSchema `json:"fields"`
}

type FieldSchema struct {
	Key         string `json:"key"`
	Path        string `json:"path"`
	Type        string `json:"type"` // "boolean" | "integer"
	Title       string `json:"title"`
	Description string `json:"description"`
	Min         *int   `json:"min,omitempty"`
	Max         *int   `json:"max,omitempty"`
}

func intPtr(v int) *int { return &v }

func Schema() ConfigSchema {
	return ConfigSchema{
		Version: "1.0.0",
		Sections: []SectionSchema{
			{
				Key:         "weights",
				Title:       "Weights",
				Description: "How much each dimension contributes to the final score. Weights must sum to 100.",
				Fields: []FieldSchema{
					{Key: "quality", Path: "weights.quality", Type: "integer", Title: "Quality", Description: "Pass-rate quality signals (requirements, targets, tests).", Min: intPtr(0), Max: intPtr(100)},
					{Key: "coverage", Path: "weights.coverage", Type: "integer", Title: "Coverage", Description: "How well tests cover requirements (ratio + depth).", Min: intPtr(0), Max: intPtr(100)},
					{Key: "quantity", Path: "weights.quantity", Type: "integer", Title: "Quantity", Description: "Breadth of requirements/targets/tests.", Min: intPtr(0), Max: intPtr(100)},
					{Key: "ui", Path: "weights.ui", Type: "integer", Title: "UI", Description: "UI sophistication and integration signals.", Min: intPtr(0), Max: intPtr(100)},
				},
			},
			{
				Key:         "quality",
				Title:       "Quality",
				Description: "Reward high pass rates across requirements, targets, and tests.",
				EnabledPath: "components.quality.enabled",
				WeightPath:  "weights.quality",
				Fields: []FieldSchema{
					{Key: "requirement_pass_rate", Path: "components.quality.requirement_pass_rate", Type: "boolean", Title: "Requirement pass rate", Description: "How many requirements pass validation."},
					{Key: "target_pass_rate", Path: "components.quality.target_pass_rate", Type: "boolean", Title: "Target pass rate", Description: "How many operational targets pass validation."},
					{Key: "test_pass_rate", Path: "components.quality.test_pass_rate", Type: "boolean", Title: "Test pass rate", Description: "How many automated tests pass."},
				},
			},
			{
				Key:         "coverage",
				Title:       "Coverage",
				Description: "Reward test coverage depth and a healthy tests-to-requirements ratio.",
				EnabledPath: "components.coverage.enabled",
				WeightPath:  "weights.coverage",
				Fields: []FieldSchema{
					{Key: "test_coverage_ratio", Path: "components.coverage.test_coverage_ratio", Type: "boolean", Title: "Test coverage ratio", Description: "Tests per requirement (capped)."},
					{Key: "requirement_depth", Path: "components.coverage.requirement_depth", Type: "boolean", Title: "Requirement depth", Description: "Depth/structure of the requirement tree."},
				},
			},
			{
				Key:         "quantity",
				Title:       "Quantity",
				Description: "Reward having enough requirements, targets, and tests for the scenario category.",
				EnabledPath: "components.quantity.enabled",
				WeightPath:  "weights.quantity",
				Fields: []FieldSchema{
					{Key: "requirements", Path: "components.quantity.requirements", Type: "boolean", Title: "Requirements count", Description: "Enough requirements for the scenario category."},
					{Key: "targets", Path: "components.quantity.targets", Type: "boolean", Title: "Targets count", Description: "Enough operational targets for the scenario category."},
					{Key: "tests", Path: "components.quantity.tests", Type: "boolean", Title: "Tests count", Description: "Enough tests for the scenario category."},
				},
			},
			{
				Key:         "ui",
				Title:       "UI",
				Description: "Reward non-template UI, component complexity, API integration, routing, and meaningful code volume.",
				EnabledPath: "components.ui.enabled",
				WeightPath:  "weights.ui",
				Fields: []FieldSchema{
					{Key: "template_detection", Path: "components.ui.template_detection", Type: "boolean", Title: "Template detection", Description: "Detect and reward non-template UI."},
					{Key: "component_complexity", Path: "components.ui.component_complexity", Type: "boolean", Title: "Component complexity", Description: "UI file/component/page counts as a proxy for complexity."},
					{Key: "api_integration", Path: "components.ui.api_integration", Type: "boolean", Title: "API integration", Description: "UI usage of API endpoints beyond /health."},
					{Key: "routing", Path: "components.ui.routing", Type: "boolean", Title: "Routing", Description: "Route count as a light signal of UI breadth."},
					{Key: "code_volume", Path: "components.ui.code_volume", Type: "boolean", Title: "Code volume", Description: "Total UI LOC as a weak proxy for substance."},
				},
			},
			{
				Key:         "penalties",
				Title:       "Penalties",
				Description: "Deduct points for validation anti-patterns that inflate metrics without real coverage.",
				EnabledPath: "penalties.enabled",
				Fields: []FieldSchema{
					{Key: "insufficient_test_coverage", Path: "penalties.insufficient_test_coverage", Type: "boolean", Title: "Insufficient test coverage", Description: "Suspicious 1:1 test-to-requirement ratio."},
					{Key: "invalid_test_location", Path: "penalties.invalid_test_location", Type: "boolean", Title: "Invalid test location", Description: "Requirements reference unsupported test paths."},
					{Key: "monolithic_test_files", Path: "penalties.monolithic_test_files", Type: "boolean", Title: "Monolithic test files", Description: "Single tests validate many requirements (weak signal)."},
					{Key: "single_layer_validation", Path: "penalties.single_layer_validation", Type: "boolean", Title: "Insufficient validation layers", Description: "Critical requirements lack multi-layer validation."},
					{Key: "target_mapping_ratio", Path: "penalties.target_mapping_ratio", Type: "boolean", Title: "Ungrouped operational targets", Description: "Excessive 1:1 target-to-requirement mappings."},
					{Key: "superficial_test_implementation", Path: "penalties.superficial_test_implementation", Type: "boolean", Title: "Superficial test implementation", Description: "Tests appear superficial (low assertions/structure)."},
					{Key: "manual_validations", Path: "penalties.manual_validations", Type: "boolean", Title: "Missing test automation", Description: "Excessive manual validation usage."},
				},
			},
			{
				Key:         "circuit_breaker",
				Title:       "Circuit Breaker",
				Description: "Auto-disable flaky collectors after repeated failures so scoring keeps working.",
				EnabledPath: "circuit_breaker.enabled",
				Fields: []FieldSchema{
					{Key: "auto_disable", Path: "circuit_breaker.auto_disable", Type: "boolean", Title: "Auto-disable", Description: "Automatically disable collectors that repeatedly fail."},
					{Key: "fail_threshold", Path: "circuit_breaker.fail_threshold", Type: "integer", Title: "Fail threshold", Description: "Failures before tripping.", Min: intPtr(1), Max: intPtr(20)},
					{Key: "retry_interval_seconds", Path: "circuit_breaker.retry_interval_seconds", Type: "integer", Title: "Retry interval (s)", Description: "Seconds before retrying a tripped collector.", Min: intPtr(1), Max: intPtr(86400)},
				},
			},
		},
	}
}

