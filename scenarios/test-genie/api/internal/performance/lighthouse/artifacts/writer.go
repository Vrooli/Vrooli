package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/performance/lighthouse"
)

const (
	// LighthouseDir is the directory for Lighthouse artifacts.
	LighthouseDir = "coverage/lighthouse"
	// PhaseResultsDir is the directory for phase results.
	PhaseResultsDir = "coverage/phase-results"
	// PhaseResultsFile is the filename for lighthouse phase results.
	PhaseResultsFile = "lighthouse.json"
)

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

// FileSystem abstracts filesystem operations for testing.
type FileSystem interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// OSFileSystem is the default filesystem implementation using os package.
type OSFileSystem struct{}

// WriteFile writes data to a file.
func (OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// MkdirAll creates a directory and all parents.
func (OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// FileWriter writes artifacts to the filesystem.
type FileWriter struct {
	scenarioDir  string
	scenarioName string
	fs           FileSystem
}

// WriterOption configures a FileWriter.
type WriterOption func(*FileWriter)

// NewWriter creates a new artifact writer.
func NewWriter(scenarioDir, scenarioName string, opts ...WriterOption) *FileWriter {
	w := &FileWriter{
		scenarioDir:  scenarioDir,
		scenarioName: scenarioName,
		fs:           OSFileSystem{},
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// WithFileSystem sets a custom filesystem implementation.
func WithFileSystem(fs FileSystem) WriterOption {
	return func(w *FileWriter) {
		w.fs = fs
	}
}

// lighthouseDir returns the lighthouse artifacts directory path.
func (w *FileWriter) lighthouseDir() string {
	return filepath.Join(w.scenarioDir, LighthouseDir)
}

// phaseResultsDir returns the phase results directory path.
func (w *FileWriter) phaseResultsDir() string {
	return filepath.Join(w.scenarioDir, PhaseResultsDir)
}

// WritePageReport writes the raw Lighthouse JSON report for a single page.
// Returns the relative path to the artifact.
func (w *FileWriter) WritePageReport(pageID string, rawResponse []byte) (string, error) {
	if len(rawResponse) == 0 {
		return "", nil // Nothing to write
	}

	targetDir := w.lighthouseDir()
	if err := w.fs.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create lighthouse dir: %w", err)
	}

	// Sanitize page ID for filename
	filename := sanitizeFilename(pageID) + ".json"
	path := filepath.Join(targetDir, filename)

	// Pretty-print the JSON if it's valid
	var obj interface{}
	if err := json.Unmarshal(rawResponse, &obj); err == nil {
		prettyData, _ := json.MarshalIndent(obj, "", "  ")
		if err := w.fs.WriteFile(path, prettyData, 0o644); err != nil {
			return "", fmt.Errorf("failed to write page report: %w", err)
		}
	} else {
		// Write raw if not valid JSON
		if err := w.fs.WriteFile(path, rawResponse, 0o644); err != nil {
			return "", fmt.Errorf("failed to write page report: %w", err)
		}
	}

	return filepath.Join(LighthouseDir, filename), nil
}

// WriteHTMLReport writes the Lighthouse HTML report for a single page.
// Returns the relative path to the artifact.
func (w *FileWriter) WriteHTMLReport(pageID string, htmlContent []byte) (string, error) {
	if len(htmlContent) == 0 {
		return "", nil // Nothing to write
	}

	targetDir := w.lighthouseDir()
	if err := w.fs.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create lighthouse dir: %w", err)
	}

	// Sanitize page ID for filename
	filename := sanitizeFilename(pageID) + ".html"
	path := filepath.Join(targetDir, filename)

	if err := w.fs.WriteFile(path, htmlContent, 0o644); err != nil {
		return "", fmt.Errorf("failed to write HTML report: %w", err)
	}

	return filepath.Join(LighthouseDir, filename), nil
}

// WritePhaseResults writes the phase results JSON for integration with the business phase.
func (w *FileWriter) WritePhaseResults(result *lighthouse.AuditResult) error {
	if result == nil || result.Skipped {
		return nil // Nothing to write
	}

	phaseDir := w.phaseResultsDir()
	if err := w.fs.MkdirAll(phaseDir, 0o755); err != nil {
		return fmt.Errorf("failed to create phase results dir: %w", err)
	}

	output := buildPhaseOutput(w.scenarioName, result)

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal phase results: %w", err)
	}

	path := filepath.Join(phaseDir, PhaseResultsFile)
	if err := w.fs.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write phase results: %w", err)
	}

	return nil
}

// WriteSummary writes a summary JSON containing all page results.
// Returns the relative path to the summary file.
func (w *FileWriter) WriteSummary(result *lighthouse.AuditResult) (string, error) {
	if result == nil || result.Skipped || len(result.PageResults) == 0 {
		return "", nil // Nothing to write
	}

	targetDir := w.lighthouseDir()
	if err := w.fs.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create lighthouse dir: %w", err)
	}

	summary := buildSummary(w.scenarioName, result)

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %w", err)
	}

	path := filepath.Join(targetDir, "summary.json")
	if err := w.fs.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write summary: %w", err)
	}

	return filepath.Join(LighthouseDir, "summary.json"), nil
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

// sanitizeFilename converts a string to a safe filename.
func sanitizeFilename(name string) string {
	// Replace path separators and spaces with dashes
	name = strings.ReplaceAll(name, string(filepath.Separator), "-")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, " ", "-")

	// Keep only alphanumeric, dash, and underscore
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, name)

	// Collapse multiple dashes
	for strings.Contains(name, "--") {
		name = strings.ReplaceAll(name, "--", "-")
	}

	return strings.Trim(name, "-")
}

// Ensure FileWriter implements Writer.
var _ Writer = (*FileWriter)(nil)
