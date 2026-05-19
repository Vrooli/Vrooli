package lighthouse

import (
	"fmt"
	"strings"

	"test-genie/internal/structure/types"
)

// Re-export shared types for convenience within this package.
type (
	FailureClass    = types.FailureClass
	ObservationType = types.ObservationType
	Observation     = types.Observation
	Result          = types.Result
)

// Re-export constants.
const (
	FailureClassNone             = types.FailureClassNone
	FailureClassMisconfiguration = types.FailureClassMisconfiguration
	FailureClassSystem           = types.FailureClassSystem
	FailureClassUnavailable      = FailureClass("unavailable")

	ObservationSection = types.ObservationSection
	ObservationSuccess = types.ObservationSuccess
	ObservationWarning = types.ObservationWarning
	ObservationError   = types.ObservationError
	ObservationInfo    = types.ObservationInfo
	ObservationSkip    = types.ObservationSkip
)

// Re-export constructor functions.
var (
	NewSectionObservation = types.NewSectionObservation
	NewSuccessObservation = types.NewSuccessObservation
	NewWarningObservation = types.NewWarningObservation
	NewErrorObservation   = types.NewErrorObservation
	NewInfoObservation    = types.NewInfoObservation
	NewSkipObservation    = types.NewSkipObservation

	OK                   = types.OK
	OKWithCount          = types.OKWithCount
	Fail                 = types.Fail
	FailMisconfiguration = types.FailMisconfiguration
	FailSystem           = types.FailSystem
)

// FailUnavailable creates an unavailable failure (Lighthouse CLI not available).
func FailUnavailable(err error, remediation string) Result {
	return Fail(err, FailureClassUnavailable, remediation)
}

// AuditResult represents the complete outcome of running Lighthouse audits.
type AuditResult struct {
	Result

	// Skipped indicates the audit was skipped (config disabled or no pages).
	Skipped bool

	// PageResults contains results for each audited page.
	PageResults []PageResult
}

// PageResult represents the outcome of auditing a single page.
type PageResult struct {
	// PageID is the identifier from the config.
	PageID string

	// URL is the full URL that was audited.
	URL string

	// Success indicates all thresholds were met.
	Success bool

	// Scores contains the category scores (0-1 scale).
	Scores map[string]float64

	// Violations contains categories that failed the error threshold.
	Violations []CategoryViolation

	// Warnings contains categories that passed error but failed warn threshold.
	Warnings []CategoryViolation

	// Error contains any error that occurred during the audit.
	Error error

	// DurationMs is how long the audit took.
	DurationMs int64

	// Requirements lists the requirement IDs that this page audit validates.
	Requirements []string

	// RawResponse contains the full Lighthouse JSON response for report generation.
	RawResponse []byte

	// RawHTMLResponse contains the Lighthouse HTML report if HTML format was requested.
	RawHTMLResponse []byte

	// RetryCount indicates how many retries were needed (0 = first attempt succeeded).
	RetryCount int
}

// CategoryViolation represents a threshold violation for a category.
type CategoryViolation struct {
	// Category is the Lighthouse category (e.g., "performance").
	Category string

	// Score is the actual score (0-1 scale).
	Score float64

	// Threshold is the threshold that was violated.
	Threshold float64

	// Level indicates whether this was an "error" or "warn" violation.
	Level string
}

// String returns a human-readable description of the violation.
func (v CategoryViolation) String() string {
	return fmt.Sprintf("%s: %.0f%% (threshold: %.0f%%)",
		v.Category, v.Score*100, v.Threshold*100)
}

// Summary returns a human-readable summary of all page results.
func (r *AuditResult) Summary() string {
	if r.Skipped {
		return "Lighthouse audits skipped"
	}

	if len(r.PageResults) == 0 {
		return "No pages audited"
	}

	passed := 0
	failed := 0
	for _, pr := range r.PageResults {
		if pr.Success {
			passed++
		} else {
			failed++
		}
	}

	if failed == 0 {
		return fmt.Sprintf("Lighthouse: %d page(s) passed", passed)
	}
	return fmt.Sprintf("Lighthouse: %d passed, %d failed", passed, failed)
}

// ScoreSummary returns a formatted summary of scores for a page result.
func (pr *PageResult) ScoreSummary() string {
	if pr.Error != nil {
		return fmt.Sprintf("%s: error - %v", pr.PageID, pr.Error)
	}

	parts := make([]string, 0, len(pr.Scores))
	// Order categories consistently
	categories := []string{"performance", "accessibility", "best-practices", "seo"}
	for _, cat := range categories {
		if score, ok := pr.Scores[cat]; ok {
			parts = append(parts, fmt.Sprintf("%s: %.0f%%", cat, score*100))
		}
	}
	// Add any other categories
	for cat, score := range pr.Scores {
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
