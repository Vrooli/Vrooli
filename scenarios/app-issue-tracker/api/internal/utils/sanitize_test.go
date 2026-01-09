package utils

import "testing"

func TestForFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple-name", "simple-name"},
		{"with spaces", "with_spaces"},
		{"with/slash", "with_slash"},
		{"with\\backslash", "with_backslash"},
		{"app-issue-tracker-123", "app-issue-tracker-123"},
		{"MixedCase123", "MixedCase123"},
		{"special!@#$%chars", "special_____chars"},
		{"", "default"},
		{"!!!", "___"},
	}

	for _, tt := range tests {
		got := ForFilename(tt.input)
		if got != tt.want {
			t.Errorf("ForFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestForArtifact(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple Name", "simple-name"},
		{"with  spaces", "with-spaces"},
		{"with/slash", "with-slash"},
		{"with\\backslash", "with-backslash"},
		{"MixedCase", "mixedcase"},
		{"file.txt", "file.txt"},
		{"file_name.txt", "file_name.txt"},
		{"special!@#chars", "special-chars"},
		{"", ""},
		{"  trimmed  ", "trimmed"},
		{"multiple---hyphens", "multiple-hyphens"},
	}

	for _, tt := range tests {
		got := ForArtifact(tt.input)
		if got != tt.want {
			t.Errorf("ForArtifact(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValueOrFallback(t *testing.T) {
	tests := []struct {
		value    string
		fallback string
		want     string
	}{
		{"actual", "fallback", "actual"},
		{"", "fallback", "fallback"},
		{"  ", "fallback", "fallback"},
		{"  actual  ", "fallback", "actual"},
		{"value", "(not provided)", "value"},
		{"", "(not provided)", "(not provided)"},
	}

	for _, tt := range tests {
		got := ValueOrFallback(tt.value, tt.fallback)
		if got != tt.want {
			t.Errorf("ValueOrFallback(%q, %q) = %q, want %q", tt.value, tt.fallback, got, tt.want)
		}
	}
}
