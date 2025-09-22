package main

import (
	"path/filepath"
	"testing"
)

func TestServiceSetupConditionsRule(t *testing.T) {
	rule := loadSetupConditionsRule(t)

	tests := []struct {
		name               string
		content            string
		path               string
		expectedCount      int
		expectedSubstrings []string
	}{
		{
			name:          "valid condition",
			content:       `{"service":{"name":"scenario-auditor"},"lifecycle":{"setup":{"condition":{"checks":[{"type":"binaries","targets":["api/scenario-auditor-api"]},{"type":"cli","targets":["scenario-auditor"]},{"type":"custom","targets":["anything"]}]}}}}`,
			path:          ".vrooli/service.json",
			expectedCount: 0,
		},
		{
			name:               "missing condition",
			content:            `{"service":{"name":"scenario-auditor"},"lifecycle":{"setup":{}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"lifecycle.setup.condition"},
		},
		{
			name:               "missing service name",
			content:            `{"lifecycle":{"setup":{"condition":{"checks":[{"type":"binaries","targets":["api/scenario-auditor-api"]},{"type":"cli","targets":["scenario-auditor"]}]}}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"service.name"},
		},
		{
			name:               "first check wrong type",
			content:            `{"service":{"name":"scenario-auditor"},"lifecycle":{"setup":{"condition":{"checks":[{"type":"cli","targets":["scenario-auditor"]}]}}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      2,
			expectedSubstrings: []string{"binaries", "cli"},
		},
		{
			name:               "missing binary target",
			content:            `{"service":{"name":"scenario-auditor"},"lifecycle":{"setup":{"condition":{"checks":[{"type":"binaries","targets":["other-api"]},{"type":"cli","targets":["scenario-auditor"]}]}}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"api/scenario-auditor-api"},
		},
		{
			name:               "missing cli target",
			content:            `{"service":{"name":"scenario-auditor"},"lifecycle":{"setup":{"condition":{"checks":[{"type":"binaries","targets":["api/scenario-auditor-api"]},{"type":"cli","targets":["other"]}]}}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"scenario-auditor"},
		},
		{
			name:          "ignores other files",
			content:       `{"service":{"name":"scenario-auditor"}}`,
			path:          "config.json",
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations, err := rule.Check(tc.content, tc.path, "")
			if err != nil {
				t.Fatalf("Check returned error: %v", err)
			}
			if len(violations) != tc.expectedCount {
				t.Fatalf("expected %d violations, got %d", tc.expectedCount, len(violations))
			}
			for _, expected := range tc.expectedSubstrings {
				if !containsViolationSubstring(violations, expected) {
					t.Fatalf("expected violation containing %q, got %#v", expected, violations)
				}
			}
		})
	}
}

func loadSetupConditionsRule(t *testing.T) RuleInfo {
	t.Helper()
	rulePath := filepath.Join("..", "rules", "config", "service_setup_conditions.go")
	info := RuleInfo{
		ID:       "service_setup_conditions",
		FilePath: rulePath,
		Category: "config",
	}

	exec, status := compileGoRule(&info)
	if !status.Valid {
		t.Fatalf("expected rule to compile, got error: %s", status.Error)
	}
	info.executor = exec
	return info
}
