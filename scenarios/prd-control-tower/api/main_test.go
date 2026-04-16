package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSplitOrigins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single origin",
			input:    "http://localhost:3000",
			expected: []string{"http://localhost:3000"},
		},
		{
			name:     "multiple origins",
			input:    "http://localhost:3000,http://localhost:4000",
			expected: []string{"http://localhost:3000", "http://localhost:4000"},
		},
		{
			name:     "origins with spaces",
			input:    "http://localhost:3000, http://localhost:4000 , http://localhost:5000",
			expected: []string{"http://localhost:3000", "http://localhost:4000", "http://localhost:5000"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single origin with trailing comma",
			input:    "http://localhost:3000,",
			expected: []string{"http://localhost:3000"},
		},
		{
			name:     "whitespace only",
			input:    "   ,   ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitOrigins(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("splitOrigins(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}

			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitOrigins(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestGetVrooliRoot(t *testing.T) {
	tests := []struct {
		name        string
		vrooliRoot  string
		expectError bool
		expected    string
	}{
		{
			name:        "VROOLI_ROOT is set to repo root",
			vrooliRoot:  repoRootForTest(t),
			expectError: false,
			expected:    repoRootForTest(t),
		},
		{
			name:        "missing repo env returns error",
			vrooliRoot:  "",
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VROOLI_SOURCE_ROOT", "")
			t.Setenv("VROOLI_ROOT", tt.vrooliRoot)

			result, err := getVrooliRoot()

			if tt.expectError {
				if err == nil {
					t.Errorf("getVrooliRoot() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("getVrooliRoot() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("getVrooliRoot() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

func newContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	for _, dir := range []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(repoRootForTest(t), ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo-contract.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), data, 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	return root
}
