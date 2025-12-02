package evidence

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/requirements/types"
)

// phaseResultFile represents the structure of a phase result JSON file.
type phaseResultFile struct {
	Phase           string                 `json:"phase"`
	Status          string                 `json:"status"`
	ExecutedAt      string                 `json:"executed_at,omitempty"`
	DurationSeconds float64                `json:"duration_seconds,omitempty"`
	Results         []phaseTestResult      `json:"results,omitempty"`
	Summary         phaseResultSummary     `json:"summary,omitempty"`
	Metadata        map[string]any         `json:"metadata,omitempty"`
}

// phaseTestResult represents a single test result within a phase.
type phaseTestResult struct {
	RequirementID   string         `json:"requirement_id,omitempty"`
	TestName        string         `json:"test_name,omitempty"`
	TestFile        string         `json:"test_file,omitempty"`
	Status          string         `json:"status"`
	DurationSeconds float64        `json:"duration_seconds,omitempty"`
	Error           string         `json:"error,omitempty"`
	Output          string         `json:"output,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// phaseResultSummary contains aggregate statistics.
type phaseResultSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// loadPhaseResultsFromDir loads all phase result files from a directory.
func loadPhaseResultsFromDir(ctx context.Context, reader Reader, dir string) (types.EvidenceMap, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	evidenceMap := make(types.EvidenceMap)

	entries, err := reader.ReadDir(dir)
	if err != nil {
		return evidenceMap, nil // Return empty map, not error
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		filePath := filepath.Join(dir, name)
		results, err := loadPhaseResultFile(ctx, reader, filePath)
		if err != nil {
			continue // Skip invalid files
		}

		evidenceMap.Merge(results)
	}

	return evidenceMap, nil
}

// loadPhaseResultFile loads a single phase result file.
func loadPhaseResultFile(ctx context.Context, reader Reader, filePath string) (types.EvidenceMap, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data, err := reader.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var resultFile phaseResultFile
	if err := json.Unmarshal(data, &resultFile); err != nil {
		return nil, err
	}

	evidenceMap := make(types.EvidenceMap)
	phase := strings.ToLower(strings.TrimSpace(resultFile.Phase))

	// Parse execution time
	var executedAt time.Time
	if resultFile.ExecutedAt != "" {
		executedAt, _ = time.Parse(time.RFC3339, resultFile.ExecutedAt)
	}

	// Process individual test results
	for _, result := range resultFile.Results {
		reqID := result.RequirementID
		if reqID == "" {
			// Try to extract from test file path or name
			reqID = extractRequirementID(result.TestFile, result.TestName)
		}
		if reqID == "" {
			continue
		}

		record := types.EvidenceRecord{
			RequirementID:   reqID,
			ValidationRef:   result.TestFile,
			Status:          types.NormalizeLiveStatus(result.Status),
			Phase:           phase,
			Evidence:        result.Output,
			UpdatedAt:       executedAt,
			DurationSeconds: result.DurationSeconds,
			SourcePath:      filePath,
			Metadata:        result.Metadata,
		}

		evidenceMap.Add(record)
	}

	// If no individual results, create a summary record for the phase
	if len(resultFile.Results) == 0 && resultFile.Status != "" {
		// This is a phase-level result without individual test breakdown
		record := types.EvidenceRecord{
			RequirementID:   "", // Will be matched later by phase
			Status:          types.NormalizeLiveStatus(resultFile.Status),
			Phase:           phase,
			UpdatedAt:       executedAt,
			DurationSeconds: resultFile.DurationSeconds,
			SourcePath:      filePath,
			Metadata:        resultFile.Metadata,
		}

		// Store with empty key to indicate phase-level result
		evidenceMap["__phase__"+phase] = append(evidenceMap["__phase__"+phase], record)
	}

	return evidenceMap, nil
}

// extractRequirementID attempts to extract a requirement ID from test file/name.
func extractRequirementID(testFile, testName string) string {
	// Common patterns:
	// - test file: "TESTGENIE-ORCH-P0.test.ts"
	// - test name: "[TESTGENIE-ORCH-P0] should validate..."
	// - comment in file: "@requirement TESTGENIE-ORCH-P0"

	// Try to find ID pattern in test name first
	if id := findRequirementPattern(testName); id != "" {
		return id
	}

	// Try test file name
	if id := findRequirementPattern(testFile); id != "" {
		return id
	}

	return ""
}

// findRequirementPattern looks for requirement ID patterns in a string.
func findRequirementPattern(s string) string {
	// Common patterns:
	// - [REQ-ID] or (REQ-ID)
	// - REQ-ID at word boundary
	// - Uppercase letters followed by dash and alphanumeric

	// Look for bracketed patterns first
	if idx := strings.Index(s, "["); idx >= 0 {
		endIdx := strings.Index(s[idx:], "]")
		if endIdx > 0 {
			candidate := s[idx+1 : idx+endIdx]
			if isValidRequirementID(candidate) {
				return candidate
			}
		}
	}

	if idx := strings.Index(s, "("); idx >= 0 {
		endIdx := strings.Index(s[idx:], ")")
		if endIdx > 0 {
			candidate := s[idx+1 : idx+endIdx]
			if isValidRequirementID(candidate) {
				return candidate
			}
		}
	}

	// Look for pattern at start of string
	parts := strings.Fields(s)
	for _, part := range parts {
		if isValidRequirementID(part) {
			return part
		}
	}

	return ""
}

// isValidRequirementID checks if a string looks like a requirement ID.
func isValidRequirementID(s string) bool {
	// Must contain at least one hyphen or underscore
	if !strings.Contains(s, "-") && !strings.Contains(s, "_") {
		return false
	}

	// Should start with letters
	if len(s) < 3 {
		return false
	}

	first := s[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z')) {
		return false
	}

	// Should contain alphanumeric and hyphens/underscores
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') ||
			(c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_') {
			return false
		}
	}

	return true
}

// GetPhaseStatus returns the overall status for a phase from evidence.
func GetPhaseStatus(evidenceMap types.EvidenceMap, phase string) types.LiveStatus {
	key := "__phase__" + strings.ToLower(phase)
	records := evidenceMap.Get(key)
	if len(records) == 0 {
		return types.LiveUnknown
	}

	// Return the most recent status
	var latest types.EvidenceRecord
	for _, r := range records {
		if r.UpdatedAt.After(latest.UpdatedAt) {
			latest = r
		}
	}

	return latest.Status
}
