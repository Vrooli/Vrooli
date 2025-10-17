package main

import (
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
		home        string
		expectError bool
		expected    string
	}{
		{
			name:        "VROOLI_ROOT is set",
			vrooliRoot:  "/custom/vrooli",
			home:        "/home/user",
			expectError: false,
			expected:    "/custom/vrooli",
		},
		{
			name:        "VROOLI_ROOT not set, HOME is set",
			vrooliRoot:  "",
			home:        "/home/user",
			expectError: false,
			expected:    "/home/user/Vrooli",
		},
		{
			name:        "neither VROOLI_ROOT nor HOME is set",
			vrooliRoot:  "",
			home:        "",
			expectError: true,
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origVrooliRoot := getEnv("VROOLI_ROOT")
			origHome := getEnv("HOME")

			// Set test environment
			if tt.vrooliRoot != "" {
				t.Setenv("VROOLI_ROOT", tt.vrooliRoot)
			} else {
				t.Setenv("VROOLI_ROOT", "")
			}

			if tt.home != "" {
				t.Setenv("HOME", tt.home)
			} else {
				t.Setenv("HOME", "")
			}

			result, err := getVrooliRoot()

			// Restore original environment
			if origVrooliRoot != "" {
				t.Setenv("VROOLI_ROOT", origVrooliRoot)
			}
			if origHome != "" {
				t.Setenv("HOME", origHome)
			}

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

// Helper function to safely get environment variable
func getEnv(key string) string {
	// This is a test helper, actual implementation would use os.Getenv
	return ""
}
