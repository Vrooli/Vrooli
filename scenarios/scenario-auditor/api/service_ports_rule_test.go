package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestServicePortsRule(t *testing.T) {
	rule := loadPortsRule(t)

	tests := []struct {
		name               string
		content            string
		path               string
		expectedCount      int
		expectedSubstrings []string
	}{
		{
			name:          "valid configuration",
			content:       `{"ports":{"api":{"env_var":"API_PORT","range":"15000-19999"},"ui":{"env_var":"UI_PORT","range":"35000-39999"}}}`,
			path:          ".vrooli/service.json",
			expectedCount: 0,
		},
		{
			name:               "missing ports block",
			content:            `{"service":{"name":"demo"}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"ports"},
		},
		{
			name:               "invalid json",
			content:            `{`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"valid JSON"},
		},
		{
			name:               "missing api entry",
			content:            `{"ports":{"ui":{"env_var":"UI_PORT","range":"35000-39999"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"api"},
		},
		{
			name:               "wrong api env var",
			content:            `{"ports":{"api":{"env_var":"PORT","range":"15000-19999"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"API_PORT"},
		},
		{
			name:               "api range missing",
			content:            `{"ports":{"api":{"env_var":"API_PORT"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"range"},
		},
		{
			name:               "wrong api range",
			content:            `{"ports":{"api":{"env_var":"API_PORT","range":"10000-12000"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"15000-19999"},
		},
		{
			name:               "ui wrong env var",
			content:            `{"ports":{"api":{"env_var":"API_PORT","range":"15000-19999"},"ui":{"env_var":"PORT","range":"35000-39999"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"UI_PORT"},
		},
		{
			name:               "ui wrong range",
			content:            `{"ports":{"api":{"env_var":"API_PORT","range":"15000-19999"},"ui":{"env_var":"UI_PORT","range":"31000-32000"}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"35000-39999"},
		},
		{
			name:          "ui fixed port without range",
			content:       `{"ports":{"api":{"env_var":"API_PORT","range":"15000-19999"},"ui":{"env_var":"UI_PORT","fixed":38000}}}`,
			path:          ".vrooli/service.json",
			expectedCount: 0,
		},
		{
			name:               "ports api not an object",
			content:            `{"ports":{"api":8080}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"ports.api must be"},
		},
		{
			name:          "ignores non service json",
			content:       `{"ports":{"api":{"env_var":"PORT","range":"100-200"}}}`,
			path:          "config.json",
			expectedCount: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations, err := rule.Check(tc.content, tc.path)
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

func loadPortsRule(t *testing.T) RuleInfo {
	t.Helper()
	rulePath := filepath.Join("..", "rules", "config", "service_ports.go")
	info := RuleInfo{
		ID:       "config-service-ports",
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
