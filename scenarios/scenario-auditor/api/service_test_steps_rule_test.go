//go:build rule_runtime_tests
// +build rule_runtime_tests

package main

import (
	"path/filepath"
	"testing"
)

func TestServiceTestStepsRule(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	t.Setenv("VROOLI_ROOT", root)

	rule := loadTestStepsRule(t)

	tests := []struct {
		name               string
		content            string
		path               string
		expectedCount      int
		expectedSubstrings []string
	}{
		{
			name:               "missing lifecycle",
			content:            `{"service":{"name":"demo"}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"lifecycle"},
		},
		{
			name:               "missing test object",
			content:            `{"lifecycle":{}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"lifecycle.test"},
		},
		{
			name:               "missing steps",
			content:            `{"lifecycle":{"test":{}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"steps"},
		},
		{
			name:               "steps not array",
			content:            `{"lifecycle":{"test":{"steps":"oops"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"array"},
		},
		{
			name:               "required step missing",
			content:            `{"lifecycle":{"test":{"steps":[{"name":"custom","run":"script.sh"}]}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"run-tests"},
		},
		{
			name:          "valid with extra step",
			content:       `{"lifecycle":{"test":{"steps":[{"name":"run-tests","run":"test/run-tests.sh","description":"Execute comprehensive phased testing (structure, dependencies, unit, integration, business, performance)"},{"name":"notify","run":"scripts/notify.sh"}]}}}`,
			path:          ".vrooli/service.json",
			expectedCount: 0,
		},
		{
			name:          "ignores unrelated file",
			content:       `{"lifecycle":{"test":{"steps":[]}}}`,
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

func loadTestStepsRule(t *testing.T) RuleInfo {
	t.Helper()
	rulePath := filepath.Join("..", "rules", "config", "service_test_steps.go")
	info := RuleInfo{
		ID:       "config-test-steps",
		FilePath: sanitizeRuleForTest(t, rulePath),
		Category: "config",
	}

	exec, status := compileGoRule(&info)
	if !status.Valid {
		t.Fatalf("expected rule to compile, got error: %s", status.Error)
	}
	info.executor = exec
	return info
}
