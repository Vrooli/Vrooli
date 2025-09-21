package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceSetupStepsRule(t *testing.T) {
	rule := loadSetupStepsRule(t)

	tests := []struct {
		name               string
		content            string
		path               string
		expectedCount      int
		expectedSubstrings []string
	}{
		{
			name:          "valid setup",
			content:       `{"service":{"name":"file-tools"},"lifecycle":{"setup":{"steps":[{"name":"install-cli","run":"cd cli && ./install.sh","description":"Install CLI command globally","condition":{"file_exists":"cli/install.sh"}},{"name":"build-api","run":"cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go","description":"Build Go API server","condition":{"file_exists":"api/go.mod"}},{"name":"show-urls","run":"echo done"}]}}}`,
			path:          ".vrooli/service.json",
			expectedCount: 0,
		},
		{
			name:               "missing setup steps",
			content:            `{"service":{"name":"file-tools"},"lifecycle":{}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"lifecycle.setup.steps"},
		},
		{
			name:               "missing install cli",
			content:            `{"service":{"name":"file-tools"},"lifecycle":{"setup":{"steps":[{"name":"build-api","run":"cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go","description":"Build Go API server","condition":{"file_exists":"api/go.mod"}},{"name":"show-urls","run":"echo done"}]}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"install-cli"},
		},
		{
			name:               "install cli wrong run",
			content:            `{"service":{"name":"file-tools"},"lifecycle":{"setup":{"steps":[{"name":"install-cli","run":"./install.sh","description":"Install CLI command globally","condition":{"file_exists":"cli/install.sh"}},{"name":"build-api","run":"cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go","description":"Build Go API server","condition":{"file_exists":"api/go.mod"}},{"name":"show-urls","run":"echo done"}]}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"install-cli"},
		},
		{
			name:               "build api wrong binary",
			content:            `{"service":{"name":"file-tools"},"lifecycle":{"setup":{"steps":[{"name":"install-cli","run":"cd cli && ./install.sh","description":"Install CLI command globally","condition":{"file_exists":"cli/install.sh"}},{"name":"build-api","run":"cd api && go mod download && go build -o other-api ./cmd/server/main.go","description":"Build Go API server","condition":{"file_exists":"api/go.mod"}},{"name":"show-urls","run":"echo done"}]}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"build-api"},
		},
		{
			name:               "show urls not last",
			content:            `{"service":{"name":"file-tools"},"lifecycle":{"setup":{"steps":[{"name":"show-urls","run":"echo"},{"name":"install-cli","run":"cd cli && ./install.sh","description":"Install CLI command globally","condition":{"file_exists":"cli/install.sh"}},{"name":"build-api","run":"cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go","description":"Build Go API server","condition":{"file_exists":"api/go.mod"}}]}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"show-urls"},
		},
		{
			name:               "service name missing",
			content:            `{"lifecycle":{"setup":{"steps":[{"name":"install-cli","run":"cd cli && ./install.sh","description":"Install CLI command globally","condition":{"file_exists":"cli/install.sh"}},{"name":"build-api","run":"cd api && go mod download && go build -o file-tools-api ./cmd/server/main.go","description":"Build Go API server","condition":{"file_exists":"api/go.mod"}},{"name":"show-urls","run":"echo done"}]}}}`,
			path:               ".vrooli/service.json",
			expectedCount:      1,
			expectedSubstrings: []string{"service.name"},
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

func loadSetupStepsRule(t *testing.T) RuleInfo {
	t.Helper()
	rulePath := filepath.Join("..", "rules", "config", "setup_steps.go")
	info := RuleInfo{
		ID:       "setup_steps",
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

// reuse helper from ports rule test
func containsViolationSubstring(list []Violation, needle string) bool {
	for _, v := range list {
		if strings.Contains(v.Message, needle) || strings.Contains(v.Description, needle) {
			return true
		}
	}
	return false
}
