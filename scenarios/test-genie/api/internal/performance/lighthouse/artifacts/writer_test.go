package artifacts

import (
	"encoding/json"
	"os"
	"testing"

	"test-genie/internal/performance/lighthouse"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// mockFileSystem is a test double for FileSystem.
type mockFileSystem struct {
	files    map[string][]byte
	dirs     map[string]bool
	writeErr error
	mkdirErr error
}

func newMockFileSystem() *mockFileSystem {
	return &mockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.files[path] = data
	return nil
}

func (m *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.mkdirErr != nil {
		return m.mkdirErr
	}
	m.dirs[path] = true
	return nil
}

func TestNewWriter(t *testing.T) {
	w := NewWriter("/scenario/dir", "test-scenario")
	if w == nil {
		t.Fatal("expected non-nil writer")
	}
	if w.ScenarioDir != "/scenario/dir" {
		t.Errorf("expected ScenarioDir '/scenario/dir', got %q", w.ScenarioDir)
	}
	if w.ScenarioName != "test-scenario" {
		t.Errorf("expected ScenarioName 'test-scenario', got %q", w.ScenarioName)
	}
}

func TestFileWriter_WritePageReport(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	rawResponse := []byte(`{"categories": {"performance": {"score": 0.85}}}`)
	path, err := w.WritePageReport("home", rawResponse)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "coverage/lighthouse/home.json" {
		t.Errorf("expected path 'coverage/lighthouse/home.json', got %q", path)
	}

	// Check that directory was created
	if !fs.dirs["/scenario/dir/coverage/lighthouse"] {
		t.Error("expected lighthouse directory to be created")
	}

	// Check that file was written
	data, ok := fs.files["/scenario/dir/coverage/lighthouse/home.json"]
	if !ok {
		t.Fatal("expected file to be written")
	}

	// Verify it's valid JSON
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Errorf("expected valid JSON, got error: %v", err)
	}
}

func TestFileWriter_WritePageReport_EmptyResponse(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	path, err := w.WritePageReport("home", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path for nil response, got %q", path)
	}
}

func TestFileWriter_WritePageReport_SanitizesFilename(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	rawResponse := []byte(`{}`)
	path, err := w.WritePageReport("home/with spaces & special!", rawResponse)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be sanitized to a valid filename
	expected := "coverage/lighthouse/home-with-spaces-special.json"
	if path != expected {
		t.Errorf("expected path %q, got %q", expected, path)
	}
}

func TestFileWriter_WritePhaseResults(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	result := &lighthouse.AuditResult{
		PageResults: []lighthouse.PageResult{
			{
				PageID:       "home",
				URL:          "http://localhost:3000/",
				Success:      true,
				Scores:       map[string]float64{"performance": 0.85},
				Requirements: []string{"PERF-001"},
			},
		},
	}
	result.Success = true

	err := w.WritePhaseResults(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check directory was created
	if !fs.dirs["/scenario/dir/coverage/phase-results"] {
		t.Error("expected phase-results directory to be created")
	}

	// Check file was written
	data, ok := fs.files["/scenario/dir/coverage/phase-results/lighthouse.json"]
	if !ok {
		t.Fatal("expected phase results file to be written")
	}

	// Verify contents
	var output map[string]interface{}
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("failed to unmarshal phase results: %v", err)
	}

	if output["phase"] != "performance" {
		t.Errorf("expected phase 'performance', got %v", output["phase"])
	}
	if output["subphase"] != "lighthouse" {
		t.Errorf("expected subphase 'lighthouse', got %v", output["subphase"])
	}
	if output["status"] != "passed" {
		t.Errorf("expected status 'passed', got %v", output["status"])
	}
}

func TestFileWriter_WritePhaseResults_Skipped(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	result := &lighthouse.AuditResult{Skipped: true}

	err := w.WritePhaseResults(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not create any files for skipped results
	if len(fs.files) > 0 {
		t.Errorf("expected no files for skipped result, got %d", len(fs.files))
	}
}

func TestFileWriter_WriteSummary(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	result := &lighthouse.AuditResult{
		PageResults: []lighthouse.PageResult{
			{
				PageID:  "home",
				URL:     "http://localhost:3000/",
				Success: true,
				Scores:  map[string]float64{"performance": 0.85, "accessibility": 0.90},
			},
			{
				PageID:  "about",
				URL:     "http://localhost:3000/about",
				Success: false,
				Scores:  map[string]float64{"performance": 0.60},
				Violations: []lighthouse.CategoryViolation{
					{Category: "performance", Score: 0.60, Threshold: 0.75, Level: "error"},
				},
			},
		},
	}

	path, err := w.WriteSummary(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != "coverage/lighthouse/summary.json" {
		t.Errorf("expected path 'coverage/lighthouse/summary.json', got %q", path)
	}

	// Verify contents
	data := fs.files["/scenario/dir/coverage/lighthouse/summary.json"]
	var summary map[string]interface{}
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("failed to unmarshal summary: %v", err)
	}

	if summary["scenario"] != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got %v", summary["scenario"])
	}
	if summary["passed"].(float64) != 1 {
		t.Errorf("expected 1 passed, got %v", summary["passed"])
	}
	if summary["failed"].(float64) != 1 {
		t.Errorf("expected 1 failed, got %v", summary["failed"])
	}
	if summary["total"].(float64) != 2 {
		t.Errorf("expected 2 total, got %v", summary["total"])
	}
}

func TestFileWriter_WriteSummary_Skipped(t *testing.T) {
	fs := newMockFileSystem()
	w := NewWriter("/scenario/dir", "test-scenario", sharedartifacts.WithFileSystem(fs))

	result := &lighthouse.AuditResult{Skipped: true}

	path, err := w.WriteSummary(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path for skipped result, got %q", path)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"home", "home"},
		{"home-page", "home-page"},
		{"home_page", "home_page"},
		{"home page", "home-page"},
		{"home/subpage", "home-subpage"},
		{"home & about", "home-about"},
		{"home!", "home"},
		{"---home---", "home"},
		{"UPPERCASE", "UPPERCASE"},
		{"mixed123", "mixed123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sharedartifacts.SanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatScores(t *testing.T) {
	tests := []struct {
		name     string
		scores   map[string]float64
		contains []string
	}{
		{
			name:     "empty scores",
			scores:   nil,
			contains: []string{"no scores"},
		},
		{
			name:     "performance only",
			scores:   map[string]float64{"performance": 0.85},
			contains: []string{"performance: 85%"},
		},
		{
			name:     "multiple scores",
			scores:   map[string]float64{"performance": 0.85, "accessibility": 0.90},
			contains: []string{"performance: 85%", "accessibility: 90%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatScores(tt.scores)
			for _, substr := range tt.contains {
				if !containsSubstring(result, substr) {
					t.Errorf("expected %q to contain %q", result, substr)
				}
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || containsSubstring(s[1:], substr)))
}

func TestFormatViolations(t *testing.T) {
	violations := []lighthouse.CategoryViolation{
		{Category: "performance", Score: 0.60, Threshold: 0.75, Level: "error"},
		{Category: "accessibility", Score: 0.80, Threshold: 0.90, Level: "warn"},
	}

	result := formatViolations(violations)

	if result == "" {
		t.Error("expected non-empty result")
	}
	// Should contain both violations
	if !containsSubstring(result, "performance") {
		t.Error("expected result to contain 'performance'")
	}
	if !containsSubstring(result, "accessibility") {
		t.Error("expected result to contain 'accessibility'")
	}
}
