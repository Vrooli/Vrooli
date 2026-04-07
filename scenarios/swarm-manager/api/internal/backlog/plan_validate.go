// Plan validation logic: validates that plan.md files contain all 13 mandatory
// sections and required structural elements before/after workshop finalization.
package backlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PlanValidationResult holds the outcome of validating a plan.md file.
type PlanValidationResult struct {
	SectionsPresent []string `json:"sections_present"`
	SectionsMissing []string `json:"sections_missing"`
	Warnings        []string `json:"warnings"`
	Passed          bool     `json:"passed"`
	ValidatedAt     string   `json:"validated_at"`
}

// canonicalSections are the 13 mandatory plan sections, keyed by normalized
// keywords used for fuzzy matching.
var canonicalSections = []struct {
	Name     string   // display name
	Keywords []string // any of these in a normalized header = match
}{
	{"Purpose", []string{"purpose"}},
	{"Required Reading", []string{"required reading"}},
	{"Problem Statement", []string{"problem statement"}},
	{"Scope", []string{"scope"}},
	{"Current Technical Context", []string{"current technical context"}},
	{"Target End State", []string{"target end state"}},
	{"Implementation Strategy", []string{"implementation strategy"}},
	{"Contract Decisions", []string{"contract decisions"}},
	{"Testing Plan", []string{"testing plan"}},
	{"Rollout/Validation Checklist", []string{"rollout", "validation checklist"}},
	{"Risks + Mitigations", []string{"risks", "mitigations"}},
	{"Non-goals/Prohibited Patterns", []string{"non-goals", "non goals", "prohibited patterns"}},
	{"Definition of Done", []string{"definition of done"}},
}

// normalizeHeader strips leading numbers, dots, punctuation, and whitespace
// from a header string, then lowercases it.
func normalizeHeader(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	// Strip leading digits, dots, dashes, colons, and spaces (e.g. "2. " or "02: ")
	for len(h) > 0 {
		r := h[0]
		if r >= '0' && r <= '9' || r == '.' || r == '-' || r == ':' || r == ' ' {
			h = h[1:]
		} else {
			break
		}
	}
	// Replace hyphens/underscores with spaces for matching
	h = strings.ReplaceAll(h, "-", " ")
	h = strings.ReplaceAll(h, "_", " ")
	// Normalize "+" and "&" to space for "Risks + Mitigations" style
	h = strings.ReplaceAll(h, "+", " ")
	h = strings.ReplaceAll(h, "&", " ")
	// Collapse multiple spaces
	parts := strings.Fields(h)
	return strings.Join(parts, " ")
}

// extractH2Headers returns all ## header lines from plan content.
// Only h2 headers are matched — h3 (###) and deeper are ignored.
func extractH2Headers(content string) []string {
	var headers []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && !strings.HasPrefix(trimmed, "### ") {
			header := strings.TrimPrefix(trimmed, "## ")
			headers = append(headers, header)
		}
	}
	return headers
}

// ValidatePlanCompleteness checks a plan.md file for the 13 mandatory sections
// and additional required elements. Returns early with Passed=true for research kind.
func ValidatePlanCompleteness(planContent string, kind BacklogKind) PlanValidationResult {
	now := time.Now().UTC().Format(time.RFC3339)

	// Research items use conclusion.md, not plan.md — skip validation.
	if kind == KindResearch {
		return PlanValidationResult{
			Passed:      true,
			ValidatedAt: now,
		}
	}

	headers := extractH2Headers(planContent)
	normalizedHeaders := make([]string, len(headers))
	for i, h := range headers {
		normalizedHeaders[i] = normalizeHeader(h)
	}

	var present, missing []string
	for _, section := range canonicalSections {
		found := false
		for _, nh := range normalizedHeaders {
			for _, kw := range section.Keywords {
				if strings.Contains(nh, kw) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found {
			present = append(present, section.Name)
		} else {
			missing = append(missing, section.Name)
		}
	}

	var warnings []string
	lowerContent := strings.ToLower(planContent)

	// Check for prompt-manager skill read command
	if !strings.Contains(lowerContent, "prompt-manager skill read") {
		warnings = append(warnings, "No `prompt-manager skill read` command found in Required Reading")
	}

	// Check for greenfield declaration
	if !strings.Contains(lowerContent, "greenfield") {
		warnings = append(warnings, "No greenfield declaration found")
	}

	// Check for vrooli scenario restart
	if !strings.Contains(lowerContent, "vrooli scenario restart") {
		warnings = append(warnings, "No `vrooli scenario restart` final cleanup step found")
	}

	// Idea-specific checks
	if kind == KindIdea {
		if !strings.Contains(lowerContent, "scenario-generation") && !strings.Contains(lowerContent, "vrooli scenario create") {
			warnings = append(warnings, "Idea item missing `scenario-generation` skill reference or `vrooli scenario create` template usage")
		}
		if !strings.Contains(lowerContent, " ui") && !strings.Contains(lowerContent, " ui ") &&
			!strings.Contains(lowerContent, "template") {
			warnings = append(warnings, "Idea item missing UI section or UI template mention")
		}
	}

	// Determine pass/fail: no missing sections AND no critical warnings
	// Critical warnings: prompt-manager and greenfield
	criticalWarnings := 0
	for _, w := range warnings {
		if strings.Contains(w, "prompt-manager") || strings.Contains(w, "greenfield") {
			criticalWarnings++
		}
	}
	passed := len(missing) == 0 && criticalWarnings == 0

	return PlanValidationResult{
		SectionsPresent: present,
		SectionsMissing: missing,
		Warnings:        warnings,
		Passed:          passed,
		ValidatedAt:     now,
	}
}

// WriteValidationReport writes a PlanValidationResult as validation-report.json
// to the item directory.
func WriteValidationReport(itemDir string, result PlanValidationResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validation report: %w", err)
	}
	return os.WriteFile(filepath.Join(itemDir, "validation-report.json"), data, 0o644)
}

// LoadValidationReport reads validation-report.json from the item directory.
// Returns nil if the file doesn't exist.
func LoadValidationReport(itemDir string) (*PlanValidationResult, error) {
	data, err := os.ReadFile(filepath.Join(itemDir, "validation-report.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var result PlanValidationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// LoadOrRefreshValidationReport loads the validation report, re-running validation
// if plan.md is newer than the report. Returns nil if no deliverable exists.
func LoadOrRefreshValidationReport(itemDir string, kind BacklogKind) (*PlanValidationResult, error) {
	if kind == KindResearch {
		return nil, nil
	}

	deliverable := DeliverableForKind(kind)
	planPath := filepath.Join(itemDir, deliverable)
	reportPath := filepath.Join(itemDir, "validation-report.json")

	planInfo, err := os.Stat(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	reportInfo, reportStatErr := os.Stat(reportPath)
	if reportStatErr == nil && reportInfo.ModTime().After(planInfo.ModTime()) {
		// Report is fresh — load and return.
		return LoadValidationReport(itemDir)
	}

	// Stale or missing report — re-validate.
	content := LoadPlanContentByName(itemDir, deliverable)
	result := ValidatePlanCompleteness(content, kind)
	if writeErr := WriteValidationReport(itemDir, result); writeErr != nil {
		return &result, writeErr
	}
	return &result, nil
}

// FormatGapReport formats validation gaps as a structured markdown checklist
// for injection into the finalize agent prompt.
func FormatGapReport(result PlanValidationResult) string {
	if result.Passed && len(result.Warnings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Plan Validation Gaps\nThe following issues were found in plan.md. Please fix these during finalization:\n")
	for _, s := range result.SectionsMissing {
		sb.WriteString("- [ ] Missing section: ")
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	for _, w := range result.Warnings {
		sb.WriteString("- [ ] Warning: ")
		sb.WriteString(w)
		sb.WriteString("\n")
	}
	return sb.String()
}
