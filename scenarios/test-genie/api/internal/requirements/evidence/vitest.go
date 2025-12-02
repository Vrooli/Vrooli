package evidence

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"test-genie/internal/requirements/types"
)

// vitestCoverageFile represents the structure of vitest-requirements.json.
type vitestCoverageFile struct {
	Files     []vitestFileEntry `json:"files,omitempty"`
	TestFiles []vitestTestEntry `json:"testFiles,omitempty"`
	Summary   vitestSummary     `json:"summary,omitempty"`
}

// vitestFileEntry represents coverage data for a source file.
type vitestFileEntry struct {
	Path         string   `json:"path"`
	Requirements []string `json:"requirements,omitempty"`
	Coverage     float64  `json:"coverage,omitempty"`
	CoveredLines int      `json:"coveredLines,omitempty"`
	TotalLines   int      `json:"totalLines,omitempty"`
}

// vitestTestEntry represents a test file and its results.
type vitestTestEntry struct {
	Path          string           `json:"path"`
	RequirementID string           `json:"requirementId,omitempty"`
	Tests         []vitestTest     `json:"tests,omitempty"`
	Status        string           `json:"status,omitempty"`
	Duration      float64          `json:"duration,omitempty"`
}

// vitestTest represents a single test within a file.
type vitestTest struct {
	Name          string  `json:"name"`
	RequirementID string  `json:"requirementId,omitempty"`
	Status        string  `json:"status"`
	Duration      float64 `json:"duration,omitempty"`
	Error         string  `json:"error,omitempty"`
}

// vitestSummary contains aggregate statistics.
type vitestSummary struct {
	Total   int     `json:"total"`
	Passed  int     `json:"passed"`
	Failed  int     `json:"failed"`
	Skipped int     `json:"skipped"`
	Coverage float64 `json:"coverage,omitempty"`
}

// loadVitestFromFile loads vitest evidence from a JSON file.
func loadVitestFromFile(ctx context.Context, reader Reader, filePath string) (map[string][]types.VitestResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := reader.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var coverageFile vitestCoverageFile
	if err := json.Unmarshal(data, &coverageFile); err != nil {
		return nil, err
	}

	results := make(map[string][]types.VitestResult)

	// Process file coverage entries
	for _, file := range coverageFile.Files {
		for _, reqID := range file.Requirements {
			result := types.VitestResult{
				FilePath:      file.Path,
				RequirementID: reqID,
				Status:        "passed", // Files with coverage are considered passing
				CoveredLines:  file.CoveredLines,
				TotalLines:    file.TotalLines,
			}
			results[reqID] = append(results[reqID], result)
		}
	}

	// Process test file entries
	for _, testFile := range coverageFile.TestFiles {
		// File-level requirement ID
		fileReqID := testFile.RequirementID

		// Process individual tests
		for _, test := range testFile.Tests {
			reqID := test.RequirementID
			if reqID == "" {
				reqID = fileReqID
			}
			if reqID == "" {
				// Try to extract from test name
				reqID = extractRequirementFromTestName(test.Name)
			}
			if reqID == "" {
				continue
			}

			result := types.VitestResult{
				FilePath:      testFile.Path,
				TestNames:     []string{test.Name},
				RequirementID: reqID,
				Status:        test.Status,
				Duration:      test.Duration,
			}
			results[reqID] = append(results[reqID], result)
		}

		// If no individual tests but file has requirement ID
		if len(testFile.Tests) == 0 && fileReqID != "" {
			result := types.VitestResult{
				FilePath:      testFile.Path,
				RequirementID: fileReqID,
				Status:        testFile.Status,
				Duration:      testFile.Duration,
			}
			results[fileReqID] = append(results[fileReqID], result)
		}
	}

	return results, nil
}

// extractRequirementFromTestName attempts to extract a requirement ID from a test name.
func extractRequirementFromTestName(testName string) string {
	// Common patterns:
	// - "TESTGENIE-ORCH-P0: should do something"
	// - "[TESTGENIE-ORCH-P0] should do something"
	// - "should do something (TESTGENIE-ORCH-P0)"

	// Look for bracketed pattern
	if idx := strings.Index(testName, "["); idx >= 0 {
		endIdx := strings.Index(testName[idx:], "]")
		if endIdx > 0 {
			candidate := strings.TrimSpace(testName[idx+1 : idx+endIdx])
			if isValidRequirementID(candidate) {
				return candidate
			}
		}
	}

	// Look for parenthesized pattern
	if idx := strings.LastIndex(testName, "("); idx >= 0 {
		endIdx := strings.Index(testName[idx:], ")")
		if endIdx > 0 {
			candidate := strings.TrimSpace(testName[idx+1 : idx+endIdx])
			if isValidRequirementID(candidate) {
				return candidate
			}
		}
	}

	// Look for colon-separated pattern
	if idx := strings.Index(testName, ":"); idx > 0 {
		candidate := strings.TrimSpace(testName[:idx])
		if isValidRequirementID(candidate) {
			return candidate
		}
	}

	return ""
}

// DiscoverVitestFiles finds test files that might contain requirement validations.
func DiscoverVitestFiles(ctx context.Context, reader Reader, uiDir string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	var testFiles []string

	srcDir := filepath.Join(uiDir, "src")
	if !reader.Exists(srcDir) {
		return testFiles, nil
	}

	// Walk the src directory looking for test files
	err := walkDirForTests(ctx, reader, srcDir, &testFiles)
	if err != nil {
		return testFiles, err
	}

	return testFiles, nil
}

// walkDirForTests recursively finds test files.
func walkDirForTests(ctx context.Context, reader Reader, dir string, testFiles *[]string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entries, err := reader.ReadDir(dir)
	if err != nil {
		return nil // Ignore read errors
	}

	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)

		if entry.IsDir() {
			if name == "node_modules" || name == "dist" || name == "build" {
				continue
			}
			walkDirForTests(ctx, reader, path, testFiles)
			continue
		}

		// Check for test file patterns
		if isTestFile(name) {
			*testFiles = append(*testFiles, path)
		}
	}

	return nil
}

// isTestFile checks if a filename matches test file patterns.
func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.tsx") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".test.jsx") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.tsx") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasSuffix(lower, ".spec.jsx")
}

// AggregateVitestResults combines multiple vitest results into a summary.
func AggregateVitestResults(results []types.VitestResult) types.LiveStatus {
	if len(results) == 0 {
		return types.LiveNotRun
	}

	var statuses []types.LiveStatus
	for _, r := range results {
		statuses = append(statuses, r.ToLiveStatus())
	}

	return types.DeriveLiveRollup(statuses)
}
