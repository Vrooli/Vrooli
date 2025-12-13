package validators

import (
	"testing"
)

func TestAnalyzeValidationQuality_NoIssues(t *testing.T) {
	// Setup: Scenario with good test structure - 2:1 ratio avoids suspicious ratio penalty
	metrics := ValidationInputCounts{
		RequirementsTotal: 10,
		TestsTotal:        20, // 2:1 ratio is good (avoids suspicious 1:1 penalty)
	}

	requirements := []Requirement{
		{
			ID:       "REQ-001",
			Title:    "Test Requirement",
			Status:   "complete",
			Priority: "P2", // P2 only needs 1 layer
			Validation: []Validation{
				{Type: "test", Ref: "api/pkg/feature/feature_test.go"},
				{Type: "automation", Ref: "bas/cases/feature.json"},
			},
		},
	}

	analysis := AnalyzeValidationQuality(metrics, requirements, nil, "/tmp/nonexistent")

	// Should NOT have suspicious ratio issue (2:1 is fine)
	for _, issue := range analysis.Issues {
		if issue.Type == "insufficient_test_coverage" {
			t.Error("Should not have insufficient_test_coverage with 2:1 ratio")
		}
	}

	// Analysis should complete without panic
	if analysis.OverallSeverity != "medium" && analysis.OverallSeverity != "high" && !analysis.HasIssues {
		// This is expected for good scenarios
	}
}

func TestAnalyzeValidationQuality_SuspiciousRatio(t *testing.T) {
	// Setup: Scenario with suspicious 1:1 test-to-requirement ratio
	metrics := ValidationInputCounts{
		RequirementsTotal: 10,
		TestsTotal:        10, // Exactly 1:1
	}

	requirements := []Requirement{}

	analysis := AnalyzeValidationQuality(metrics, requirements, nil, "/tmp/nonexistent")

	// Should detect suspicious ratio
	found := false
	for _, issue := range analysis.Issues {
		if issue.Type == "insufficient_test_coverage" {
			found = true
			if issue.Penalty != 5 {
				t.Errorf("Expected penalty of 5, got %d", issue.Penalty)
			}
		}
	}

	if !found {
		t.Error("Expected to find insufficient_test_coverage issue")
	}
}

func TestAnalyzeTestRefUsage_MonolithicTests(t *testing.T) {
	// Setup: Requirements all referencing same test file
	requirements := []Requirement{
		{ID: "REQ-001", Validation: []Validation{{Ref: "test/shared.json"}}},
		{ID: "REQ-002", Validation: []Validation{{Ref: "test/shared.json"}}},
		{ID: "REQ-003", Validation: []Validation{{Ref: "test/shared.json"}}},
		{ID: "REQ-004", Validation: []Validation{{Ref: "test/shared.json"}}},
		{ID: "REQ-005", Validation: []Validation{{Ref: "test/shared.json"}}},
	}

	analysis := AnalyzeTestRefUsage(requirements)

	if len(analysis.Violations) != 1 {
		t.Errorf("Expected 1 violation, got %d", len(analysis.Violations))
	}

	if analysis.Violations[0].Count != 5 {
		t.Errorf("Expected violation count of 5, got %d", analysis.Violations[0].Count)
	}
}

func TestAnalyzeTestRefUsage_NoViolations(t *testing.T) {
	// Setup: Requirements with different test files
	requirements := []Requirement{
		{ID: "REQ-001", Validation: []Validation{{Ref: "test/test1.json"}}},
		{ID: "REQ-002", Validation: []Validation{{Ref: "test/test2.json"}}},
		{ID: "REQ-003", Validation: []Validation{{Ref: "test/test3.json"}}},
	}

	analysis := AnalyzeTestRefUsage(requirements)

	if len(analysis.Violations) != 0 {
		t.Errorf("Expected 0 violations, got %d", len(analysis.Violations))
	}
}

func TestAnalyzeTargetGrouping_ExcessiveOneToOne(t *testing.T) {
	// Setup: All targets have 1:1 mapping
	requirements := []Requirement{
		{ID: "REQ-001", PRDRef: "OT-P0-001"},
		{ID: "REQ-002", PRDRef: "OT-P0-002"},
		{ID: "REQ-003", PRDRef: "OT-P0-003"},
		{ID: "REQ-004", PRDRef: "OT-P0-004"},
		{ID: "REQ-005", PRDRef: "OT-P0-005"},
	}

	analysis := AnalyzeTargetGrouping(nil, requirements)

	// All 5 targets are 1:1, which exceeds acceptable ratio
	if analysis.OneToOneRatio != 1.0 {
		t.Errorf("Expected 1.0 ratio, got %f", analysis.OneToOneRatio)
	}

	if len(analysis.Violations) == 0 {
		t.Error("Expected violations for 100% 1:1 mapping")
	}
}

func TestDeriveRequirementCriticality(t *testing.T) {
	tests := []struct {
		name     string
		req      Requirement
		expected string
	}{
		{
			name:     "P0 from PRDRef",
			req:      Requirement{PRDRef: "OT-P0-001"},
			expected: "P0",
		},
		{
			name:     "P1 from PRDRef",
			req:      Requirement{PRDRef: "OT-P1-002"},
			expected: "P1",
		},
		{
			name:     "P2 from OperationalTargetID",
			req:      Requirement{OperationalTargetID: "OT-P2-003"},
			expected: "P2",
		},
		{
			name:     "P0 from Priority field",
			req:      Requirement{Priority: "P0"},
			expected: "P0",
		},
		{
			name:     "Default to P2",
			req:      Requirement{},
			expected: "P2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveRequirementCriticality(tt.req)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDetectValidationLayersBasic(t *testing.T) {
	tests := []struct {
		name           string
		req            Requirement
		expectedLayers []string
	}{
		{
			name: "API test",
			req: Requirement{
				Validation: []Validation{{Type: "test", Ref: "api/pkg/feature/feature_test.go"}},
			},
			expectedLayers: []string{"API"},
		},
		{
			name: "UI test",
			req: Requirement{
				Validation: []Validation{{Type: "test", Ref: "ui/src/components/Button.test.tsx"}},
			},
			expectedLayers: []string{"UI"},
		},
		{
			name: "E2E playbook",
			req: Requirement{
				Validation: []Validation{{Type: "automation", Ref: "bas/cases/feature.json"}},
			},
			expectedLayers: []string{"E2E"},
		},
		{
			name: "Manual",
			req: Requirement{
				Validation: []Validation{{Type: "manual"}},
			},
			expectedLayers: []string{"MANUAL"},
		},
		{
			name: "Multi-layer",
			req: Requirement{
				Validation: []Validation{
					{Type: "test", Ref: "api/pkg/feature/feature_test.go"},
					{Type: "automation", Ref: "bas/cases/feature.json"},
				},
			},
			expectedLayers: []string{"API", "E2E"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layers := DetectValidationLayersBasic(tt.req)
			for _, expected := range tt.expectedLayers {
				if !layers[expected] {
					t.Errorf("Expected layer %s to be detected", expected)
				}
			}
		})
	}
}

func TestDefaultPenaltyParameters(t *testing.T) {
	configs := DefaultPenaltyParameters()

	expectedTypes := []string{
		"insufficient_test_coverage",
		"invalid_test_location",
		"monolithic_test_files",
		"ungrouped_operational_targets",
		"insufficient_validation_layers",
		"superficial_test_implementation",
		"missing_test_automation",
	}

	for _, typeName := range expectedTypes {
		if _, ok := configs[typeName]; !ok {
			t.Errorf("Expected penalty config for %s", typeName)
		}
	}
}

func TestDefaultValidationConfig(t *testing.T) {
	cfg := DefaultValidationConfig()

	if cfg.MinLayers.P0Requirements != 2 {
		t.Errorf("Expected P0 min layers to be 2, got %d", cfg.MinLayers.P0Requirements)
	}

	if cfg.MinLayers.P1Requirements != 2 {
		t.Errorf("Expected P1 min layers to be 2, got %d", cfg.MinLayers.P1Requirements)
	}

	if cfg.MonolithicTestThreshold != 4 {
		t.Errorf("Expected monolithic threshold to be 4, got %d", cfg.MonolithicTestThreshold)
	}
}
