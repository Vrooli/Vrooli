package main

import (
	"testing"
)

// [REQ:PCT-VALIDATE-DISPLAY] Normalize PRD section titles for comparison
func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Technical Architecture",
			expected: "technical architecture",
		},
		{
			name:     "title with emoji",
			input:    "🎯 Overview",
			expected: "overview",
		},
		{
			name:     "title with multiple emojis",
			input:    "🔧 🚀 Setup Guide",
			expected: "setup guide",
		},
		{
			name:     "title with extra whitespace",
			input:    "  Requirements   Specification  ",
			expected: "requirements specification",
		},
		{
			name:     "title with mixed case",
			input:    "API Reference",
			expected: "api reference",
		},
		{
			name:     "emoji only",
			input:    "🎯",
			expected: "",
		},
		{
			name:     "title with tabs and newlines",
			input:    "Section\t\tTitle\nWith\nNewlines",
			expected: "section title with newlines",
		},
		{
			name:     "unicode emojis of different ranges",
			input:    "📊 Data Analysis 📈",
			expected: "data analysis",
		},
		{
			name:     "multiple consecutive spaces",
			input:    "Test     Multiple     Spaces",
			expected: "test multiple spaces",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "    \t\n    ",
			expected: "",
		},
		{
			name:     "special characters preserved",
			input:    "API/SDK Reference",
			expected: "api/sdk reference",
		},
		{
			name:     "numbers preserved",
			input:    "Version 2.0 Release Notes",
			expected: "version 2.0 release notes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTitle(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
