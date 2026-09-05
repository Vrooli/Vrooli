package checks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"experience-manager/internal/spec"
	"experience-manager/internal/statevocab"
)

// StateCoverageCheck compares page states with an explicit DESIGN.md UX-state
// contract seed when a scenario provides one.
type StateCoverageCheck struct{}

func (StateCoverageCheck) Name() string { return "state.coverage" }

func (StateCoverageCheck) Run(_ context.Context, report spec.Report) []spec.Finding {
	if report.Spec == nil || report.DegradedReason != "" {
		return nil
	}
	required := requiredDesignStates(report.TargetPath)
	if len(required) == 0 {
		return nil
	}
	var findings []spec.Finding
	refs := pageRefs(report.Spec.Index.Pages)
	for pageID, page := range report.Spec.Pages {
		declared := map[string]bool{}
		for _, state := range page.States {
			declared[state.ID] = true
		}
		loc := "experience/pages/" + pageID + ".json"
		if ref := refs[pageID]; ref.Path != "" {
			loc = "experience/" + ref.Path
		}
		for _, state := range required {
			if declared[state] {
				continue
			}
			findings = append(findings, spec.Finding{
				Code:       spec.CodeStateMissing,
				Severity:   spec.SeverityWarning,
				Message:    fmt.Sprintf("page %q does not declare DESIGN.md-required UX state %q", pageID, state),
				Locations:  []string{loc},
				Suggestion: "Declare the required UX state on the page or update the DESIGN.md UX-state contract.",
			})
		}
	}
	sortFindings(findings)
	return findings
}

func requiredDesignStates(scenarioDir string) []string {
	repoRoot := filepath.Dir(filepath.Dir(scenarioDir))
	allowed := statevocab.View(repoRoot, "design-required")
	if len(allowed) == 0 {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(scenarioDir, "DESIGN.md"))
	if err != nil {
		return allowed
	}
	seen := map[string]bool{}
	inSection := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "## ") {
			inSection = strings.Contains(lower, "ux-state contract") || strings.Contains(lower, "ux state contract")
			continue
		}
		if !inSection {
			continue
		}
		for _, state := range allowed {
			if strings.Contains(lower, "`"+state+"`") || strings.Contains(lower, "- "+state) || strings.Contains(lower, state+",") || strings.Contains(lower, state+"/") {
				seen[state] = true
			}
		}
	}
	var out []string
	for _, state := range allowed {
		if seen[state] {
			out = append(out, state)
		}
	}
	return out
}

func pageRefs(refs []spec.DocumentRef) map[string]spec.DocumentRef {
	out := map[string]spec.DocumentRef{}
	for _, ref := range refs {
		out[ref.ID] = ref
	}
	return out
}
