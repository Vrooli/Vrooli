package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceHealthLifecycleRule(t *testing.T) {
	rule := loadHealthRule(t)

	tests := []struct {
		name               string
		content            string
		expectedCount      int
		expectedSubstrings []string
	}{
		{
			name:          "valid api only",
			content:       `{"lifecycle":{"health":{"endpoints":{"api":"/health"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000}]}}}`,
			expectedCount: 0,
		},
		{
			name:          "valid with ui",
			content:       `{"ports":{"ui":{"env_var":"UI_PORT"}},"lifecycle":{"health":{"endpoints":{"api":"/health","ui":"/health"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000},{"name":"ui_endpoint","type":"http","target":"http://localhost:${UI_PORT}/health","critical":false,"timeout":3000,"interval":45000}]}}}`,
			expectedCount: 0,
		},
		{
			name:               "missing lifecycle",
			content:            `{}`,
			expectedCount:      1,
			expectedSubstrings: []string{"lifecycle"},
		},
		{
			name:               "missing health",
			content:            `{"lifecycle":{}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"health"},
		},
		{
			name:               "missing endpoints",
			content:            `{"lifecycle":{"health":{"checks":[]}}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"endpoints"},
		},
		{
			name:               "missing api endpoint",
			content:            `{"lifecycle":{"health":{"endpoints":{},"checks":[]}}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"api endpoint"},
		},
		{
			name:               "wrong api path",
			content:            `{"lifecycle":{"health":{"endpoints":{"api":"/status"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000}]}}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"must be '/health'"},
		},
		{
			name:               "missing api check",
			content:            `{"lifecycle":{"health":{"endpoints":{"api":"/health"},"checks":[{"name":"other","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000}]}}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"api_endpoint"},
		},
		{
			name:               "api check wrong target",
			content:            `{"lifecycle":{"health":{"endpoints":{"api":"/health"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:1234/status","critical":true,"timeout":5000,"interval":30000}]}}}`,
			expectedCount:      2,
			expectedSubstrings: []string{"${API_PORT}", "/health"},
		},
		{
			name:               "ui missing endpoint",
			content:            `{"ports":{"ui":{"env_var":"UI_PORT"}},"lifecycle":{"health":{"endpoints":{"api":"/health"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000}]}}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"ui endpoint"},
		},
		{
			name:               "ui missing check",
			content:            `{"ports":{"ui":{"env_var":"UI_PORT"}},"lifecycle":{"health":{"endpoints":{"api":"/health","ui":"/health"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000}]}}}`,
			expectedCount:      1,
			expectedSubstrings: []string{"ui_endpoint"},
		},
		{
			name:               "ui check wrong target",
			content:            `{"ports":{"ui":{"env_var":"UI_PORT"}},"lifecycle":{"health":{"endpoints":{"api":"/health","ui":"/health"},"checks":[{"name":"api_endpoint","type":"http","target":"http://localhost:${API_PORT}/health","critical":true,"timeout":5000,"interval":30000},{"name":"ui_endpoint","type":"http","target":"http://localhost:39999/status","critical":false,"timeout":3000,"interval":45000}]}}}`,
			expectedCount:      2,
			expectedSubstrings: []string{"${UI_PORT}", "/health"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations, err := rule.Check(tc.content, ".vrooli/service.json")
			if err != nil {
				t.Fatalf("Check returned error: %v", err)
			}
			if len(violations) != tc.expectedCount {
				t.Fatalf("expected %d violations, got %d (%#v)", tc.expectedCount, len(violations), violations)
			}
			for _, expected := range tc.expectedSubstrings {
				if !containsViolationSubstring(violations, expected) {
					t.Fatalf("expected violation containing %q, got %#v", expected, violations)
				}
			}
		})
	}
}

func loadHealthRule(t *testing.T) RuleInfo {
	t.Helper()
	rulePath := filepath.Join("..", "rules", "config", "service_health_lifecycle.go")
	info := RuleInfo{
		ID:       "service_health_lifecycle",
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

func containsViolationSubstring(list []Violation, needle string) bool {
	for _, v := range list {
		if strings.Contains(v.Message, needle) || strings.Contains(v.Description, needle) {
			return true
		}
	}
	return false
}
