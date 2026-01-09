package nodejs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectCoverage_FromSummaryJSON(t *testing.T) {
	dir := t.TempDir()
	coverageDir := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("failed to create coverage dir: %v", err)
	}

	summary := map[string]interface{}{
		"total": map[string]interface{}{
			"statements": map[string]interface{}{
				"pct": 85.5,
			},
		},
	}
	data, _ := json.Marshal(summary)
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-summary.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write coverage-summary.json: %v", err)
	}

	got := DetectCoverage(dir, "")
	if got != "85.50" {
		t.Errorf("DetectCoverage() = %q, want %q", got, "85.50")
	}
}

func TestDetectCoverage_FromOutput(t *testing.T) {
	dir := t.TempDir()

	output := `
Test Files  10 passed (10)
Tests       42 passed (42)
Coverage    75.25% statements
`
	got := DetectCoverage(dir, output)
	if got != "75.25" {
		t.Errorf("DetectCoverage() = %q, want %q", got, "75.25")
	}
}

func TestDetectCoverage_NoSource(t *testing.T) {
	dir := t.TempDir()

	got := DetectCoverage(dir, "All tests passed")
	if got != "" {
		t.Errorf("DetectCoverage() = %q, want empty string", got)
	}
}

func TestDetectCoverage_SummaryTakesPriority(t *testing.T) {
	dir := t.TempDir()
	coverageDir := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("failed to create coverage dir: %v", err)
	}

	summary := map[string]interface{}{
		"total": map[string]interface{}{
			"statements": map[string]interface{}{
				"pct": 90.0,
			},
		},
	}
	data, _ := json.Marshal(summary)
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-summary.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write coverage-summary.json: %v", err)
	}

	// Output has different percentage - should be ignored
	output := "Coverage 50% statements"
	got := DetectCoverage(dir, output)
	if got != "90.00" {
		t.Errorf("DetectCoverage() = %q, want %q (summary should take priority)", got, "90.00")
	}
}

func TestExtractFromSummaryJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	coverageDir := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("failed to create coverage dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-summary.json"), []byte(`{invalid`), 0o644); err != nil {
		t.Fatalf("failed to write coverage-summary.json: %v", err)
	}

	got := extractFromSummaryJSON(dir)
	if got != "" {
		t.Errorf("extractFromSummaryJSON() = %q, want empty string for invalid JSON", got)
	}
}

func TestExtractFromSummaryJSON_ZeroCoverage(t *testing.T) {
	dir := t.TempDir()
	coverageDir := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("failed to create coverage dir: %v", err)
	}

	summary := map[string]interface{}{
		"total": map[string]interface{}{
			"statements": map[string]interface{}{
				"pct": 0.0,
			},
		},
	}
	data, _ := json.Marshal(summary)
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-summary.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write coverage-summary.json: %v", err)
	}

	got := extractFromSummaryJSON(dir)
	if got != "" {
		t.Errorf("extractFromSummaryJSON() = %q, want empty string for zero coverage", got)
	}
}

func TestExtractFromOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "percentage in line",
			output: "Coverage: 85.5% of statements",
			want:   "85.5",
		},
		{
			name:   "multiple lines",
			output: "Tests passed\nCoverage 75%\nDone",
			want:   "75",
		},
		{
			name:   "no percentage",
			output: "All tests passed",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFromOutput(tt.output); got != tt.want {
				t.Errorf("extractFromOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractPercentage(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"Coverage: 85.5% of statements", "85.5"},
		{"75% coverage", "75"},
		{"100%", "100"},
		{"No percentage here", ""},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := extractPercentage(tt.line); got != tt.want {
				t.Errorf("extractPercentage(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestExtractFromSummaryJSON_FileNotExists(t *testing.T) {
	dir := t.TempDir()
	// Don't create the coverage directory or file

	got := extractFromSummaryJSON(dir)
	if got != "" {
		t.Errorf("extractFromSummaryJSON() = %q, want empty string for missing file", got)
	}
}

func TestExtractFromSummaryJSON_MissingTotalField(t *testing.T) {
	dir := t.TempDir()
	coverageDir := filepath.Join(dir, "coverage")
	if err := os.MkdirAll(coverageDir, 0o755); err != nil {
		t.Fatalf("failed to create coverage dir: %v", err)
	}

	// JSON without the expected structure
	summary := map[string]interface{}{
		"other": "data",
	}
	data, _ := json.Marshal(summary)
	if err := os.WriteFile(filepath.Join(coverageDir, "coverage-summary.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write coverage-summary.json: %v", err)
	}

	got := extractFromSummaryJSON(dir)
	if got != "" {
		t.Errorf("extractFromSummaryJSON() = %q, want empty string for missing total field", got)
	}
}

func TestExtractFromOutput_MultiplePercentages(t *testing.T) {
	// When there are multiple percentages, should return the first one
	output := `
Test Suites: 50% complete
Coverage: 85% of statements
Lines: 90%
`
	got := extractFromOutput(output)
	if got != "50" {
		t.Errorf("extractFromOutput() = %q, want %q (first percentage)", got, "50")
	}
}

func TestExtractFromOutput_OnlyWhitespaceLines(t *testing.T) {
	output := "\n   \n\t\n"
	got := extractFromOutput(output)
	if got != "" {
		t.Errorf("extractFromOutput() = %q, want empty string for whitespace-only output", got)
	}
}

func TestDetectCoverage_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	// No summary JSON, no output with percentage
	got := DetectCoverage(dir, "")
	if got != "" {
		t.Errorf("DetectCoverage() = %q, want empty string for empty sources", got)
	}
}

func TestExtractPercentage_PercentInMiddle(t *testing.T) {
	// Edge case: percentage sign in middle of token (should not match)
	tests := []struct {
		line string
		want string
	}{
		{"file%name.txt 100", ""},      // % in middle of word, no valid percentage token
		{"85.5% coverage 90%", "85.5"}, // Multiple percentages, takes first
		{"test: 0% (empty)", "0"},      // Zero percent
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			if got := extractPercentage(tt.line); got != tt.want {
				t.Errorf("extractPercentage(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
