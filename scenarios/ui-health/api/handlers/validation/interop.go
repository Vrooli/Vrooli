package validation

import (
	"path/filepath"

	"ui-health/internal/services/manifestvalidation"
	"ui-health/internal/uiinterop"
	_ "ui-health/internal/uiinterop/checks" // trigger interop rule init() registrations
)

// runInteropFindings runs the static UI-interop rule engine against the scenario
// at root and maps each failing rule's violations into the unified Finding
// model. ui-health is the single authority for these checks (ported from
// app-monitor + structure-health); they form the "static interop clean" (L1)
// check group, composed into the same ValidateScenario report as manifest
// validation. Passing and skipped rules emit no finding — the report only
// carries actionable violations. Running interop here (handler-side) keeps the
// manifestvalidation service manifest-only and lets the fixer's re-validation
// stay focused on slot findings.
func runInteropFindings(root, scenario string) []manifestvalidation.Finding {
	results := uiinterop.RunAll(root, scenario)
	var finds []manifestvalidation.Finding
	for _, r := range results {
		if r.Passed || r.Skipped {
			continue
		}
		if len(r.Violations) == 0 {
			finds = append(finds, manifestvalidation.Finding{
				Severity: manifestvalidation.SeverityWarning,
				Code:     r.RuleID,
				Location: root,
				Message:  r.Message,
			})
			continue
		}
		for _, v := range r.Violations {
			finds = append(finds, manifestvalidation.Finding{
				Severity:   interopSeverity(v.Severity),
				Code:       r.RuleID,
				Location:   interopLocation(root, v.FilePath),
				Message:    interopMessage(v),
				Suggestion: v.Recommendation,
			})
		}
	}
	return finds
}

// interopSeverity maps a rule engine severity string to the validator's
// severity. critical/high are errors; medium is a warning; low (and anything
// unrecognized) is informational.
func interopSeverity(sev string) manifestvalidation.Severity {
	switch sev {
	case "critical", "high", "error":
		return manifestvalidation.SeverityError
	case "medium", "warning":
		return manifestvalidation.SeverityWarning
	default:
		return manifestvalidation.SeverityInfo
	}
}

// interopLocation prefers the violation's file path (made absolute under the
// scenario when relative) and falls back to the scenario directory.
func interopLocation(root, filePath string) string {
	if filePath == "" {
		return root
	}
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(root, filepath.FromSlash(filePath))
}

func interopMessage(v uiinterop.Violation) string {
	if v.Description != "" {
		return v.Description
	}
	if v.Title != "" {
		return v.Title
	}
	return v.RuleID
}
