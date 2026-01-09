package artifacts

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/performance/lighthouse"
	sharedartifacts "test-genie/internal/shared/artifacts"
)

// Note: LighthouseDir and PhaseResultsLighthouse are defined in sharedartifacts.paths.go

// Writer defines the interface for writing Lighthouse artifacts.
type Writer interface {
	// WritePageReport writes the JSON report for a single page audit.
	WritePageReport(pageID string, rawResponse []byte) (string, error)
	// WriteHTMLReport writes the HTML report for a single page audit.
	WriteHTMLReport(pageID string, htmlContent []byte) (string, error)
	// WritePhaseResults writes the phase results JSON.
	WritePhaseResults(result *lighthouse.AuditResult) error
	// WriteSummary writes a summary JSON with all page results.
	WriteSummary(result *lighthouse.AuditResult) (string, error)
}

// FileWriter writes artifacts to the filesystem.
type FileWriter struct {
	*sharedartifacts.BaseWriter
}

// NewWriter creates a new artifact writer.
func NewWriter(scenarioDir, scenarioName string, opts ...sharedartifacts.BaseWriterOption) *FileWriter {
	return &FileWriter{
		BaseWriter: sharedartifacts.NewBaseWriter(scenarioDir, scenarioName, opts...),
	}
}

// lighthouseDir returns the lighthouse artifacts directory path.
func (w *FileWriter) lighthouseDir() string {
	return filepath.Join(w.ScenarioDir, sharedartifacts.LighthouseDir)
}

// WritePageReport writes the raw Lighthouse JSON report for a single page.
// Returns the relative path to the artifact.
func (w *FileWriter) WritePageReport(pageID string, rawResponse []byte) (string, error) {
	if len(rawResponse) == 0 {
		return "", nil // Nothing to write
	}

	targetDir := w.lighthouseDir()
	if err := w.EnsureDir(targetDir); err != nil {
		return "", fmt.Errorf("failed to create lighthouse dir: %w", err)
	}

	// Sanitize page ID for filename
	filename := sharedartifacts.SanitizeFilename(pageID) + ".json"
	path := filepath.Join(targetDir, filename)

	// Pretty-print the JSON if it's valid
	prettyData := sharedartifacts.PrettyPrintJSON(rawResponse)
	if err := w.WriteFile(path, prettyData); err != nil {
		return "", fmt.Errorf("failed to write page report: %w", err)
	}

	return sharedartifacts.RelativeLighthouseArtifactPath(filename), nil
}

// WriteHTMLReport writes the Lighthouse HTML report for a single page.
// Returns the relative path to the artifact.
func (w *FileWriter) WriteHTMLReport(pageID string, htmlContent []byte) (string, error) {
	if len(htmlContent) == 0 {
		return "", nil // Nothing to write
	}

	targetDir := w.lighthouseDir()
	if err := w.EnsureDir(targetDir); err != nil {
		return "", fmt.Errorf("failed to create lighthouse dir: %w", err)
	}

	// Sanitize page ID for filename
	filename := sharedartifacts.SanitizeFilename(pageID) + ".html"
	path := filepath.Join(targetDir, filename)

	if err := w.WriteFile(path, htmlContent); err != nil {
		return "", fmt.Errorf("failed to write HTML report: %w", err)
	}

	return sharedartifacts.RelativeLighthouseArtifactPath(filename), nil
}

// WritePhaseResults writes the phase results JSON for integration with the business phase.
func (w *FileWriter) WritePhaseResults(result *lighthouse.AuditResult) error {
	if result == nil || result.Skipped {
		return nil // Nothing to write
	}
	return sharedartifacts.WritePhaseResults(w.BaseWriter, sharedartifacts.PhaseResultsLighthouse, result, buildPhaseOutput)
}

// WriteSummary writes a summary JSON containing all page results.
// Returns the relative path to the summary file.
func (w *FileWriter) WriteSummary(result *lighthouse.AuditResult) (string, error) {
	if result == nil || result.Skipped || len(result.PageResults) == 0 {
		return "", nil // Nothing to write
	}

	targetDir := w.lighthouseDir()
	if err := w.EnsureDir(targetDir); err != nil {
		return "", fmt.Errorf("failed to create lighthouse dir: %w", err)
	}

	summary := buildSummary(w.ScenarioName, result)

	path := filepath.Join(targetDir, sharedartifacts.LighthouseSummary)
	if err := w.WriteJSON(path, summary); err != nil {
		return "", fmt.Errorf("failed to write summary: %w", err)
	}

	return sharedartifacts.RelativeLighthouseArtifactPath(sharedartifacts.LighthouseSummary), nil
}

// buildPhaseOutput constructs the phase results structure for requirements integration.
func buildPhaseOutput(scenarioName string, result *lighthouse.AuditResult) map[string]any {
	output := map[string]any{
		"phase":      "performance",
		"subphase":   "lighthouse",
		"scenario":   scenarioName,
		"pages":      len(result.PageResults),
		"errors":     0,
		"status":     "passed",
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	var requirementEntries []map[string]any
	errorCount := 0

	for _, pr := range result.PageResults {
		status := "passed"
		evidence := fmt.Sprintf("page %s: %s", pr.PageID, formatScores(pr.Scores))

		if !pr.Success {
			status = "failed"
			if pr.Error != nil {
				evidence = fmt.Sprintf("page %s: error - %v", pr.PageID, pr.Error)
			} else if len(pr.Violations) > 0 {
				evidence = fmt.Sprintf("page %s: %s (violations: %s)",
					pr.PageID, formatScores(pr.Scores), formatViolations(pr.Violations))
			}
			errorCount++
		}

		// Create requirement entries for each linked requirement
		for _, reqID := range pr.Requirements {
			requirementEntries = append(requirementEntries, map[string]any{
				"id":         reqID,
				"status":     status,
				"phase":      "performance",
				"evidence":   evidence,
				"page_id":    pr.PageID,
				"scores":     pr.Scores,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	output["errors"] = errorCount
	if errorCount > 0 {
		output["status"] = "failed"
	}
	output["requirements"] = requirementEntries

	return output
}

// buildSummary constructs a summary of all page results.
func buildSummary(scenarioName string, result *lighthouse.AuditResult) map[string]any {
	pages := make([]map[string]any, 0, len(result.PageResults))

	for _, pr := range result.PageResults {
		page := map[string]any{
			"id":          pr.PageID,
			"url":         pr.URL,
			"success":     pr.Success,
			"scores":      pr.Scores,
			"duration_ms": pr.DurationMs,
		}

		if pr.Error != nil {
			page["error"] = pr.Error.Error()
		}
		if len(pr.Violations) > 0 {
			page["violations"] = formatViolationsDetail(pr.Violations)
		}
		if len(pr.Warnings) > 0 {
			page["warnings"] = formatViolationsDetail(pr.Warnings)
		}
		if len(pr.Requirements) > 0 {
			page["requirements"] = pr.Requirements
		}
		if pr.RetryCount > 0 {
			page["retry_count"] = pr.RetryCount
		}

		pages = append(pages, page)
	}

	passed := 0
	failed := 0
	for _, pr := range result.PageResults {
		if pr.Success {
			passed++
		} else {
			failed++
		}
	}

	return map[string]any{
		"scenario":   scenarioName,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"summary":    result.Summary(),
		"passed":     passed,
		"failed":     failed,
		"total":      len(result.PageResults),
		"pages":      pages,
		"success":    result.Success,
		"skipped":    result.Skipped,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
}

// formatScores formats scores as a human-readable string.
func formatScores(scores map[string]float64) string {
	if len(scores) == 0 {
		return "no scores"
	}

	parts := make([]string, 0, len(scores))
	// Order categories consistently
	categories := []string{"performance", "accessibility", "best-practices", "seo"}
	for _, cat := range categories {
		if score, ok := scores[cat]; ok {
			parts = append(parts, fmt.Sprintf("%s: %.0f%%", cat, score*100))
		}
	}
	// Add any other categories
	for cat, score := range scores {
		found := false
		for _, c := range categories {
			if c == cat {
				found = true
				break
			}
		}
		if !found {
			parts = append(parts, fmt.Sprintf("%s: %.0f%%", cat, score*100))
		}
	}

	return strings.Join(parts, ", ")
}

// formatViolations formats violations as a human-readable string.
func formatViolations(violations []lighthouse.CategoryViolation) string {
	parts := make([]string, len(violations))
	for i, v := range violations {
		parts[i] = v.String()
	}
	return strings.Join(parts, ", ")
}

// formatViolationsDetail formats violations as detailed maps.
func formatViolationsDetail(violations []lighthouse.CategoryViolation) []map[string]any {
	result := make([]map[string]any, len(violations))
	for i, v := range violations {
		result[i] = map[string]any{
			"category":  v.Category,
			"score":     v.Score,
			"threshold": v.Threshold,
			"level":     v.Level,
		}
	}
	return result
}

// Ensure FileWriter implements Writer.
var _ Writer = (*FileWriter)(nil)
