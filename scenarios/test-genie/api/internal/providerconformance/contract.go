package providerconformance

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/maturity-go/assessment"

	"test-genie/internal/orchestrator/providerdescriptor"
)

// RequiredDocHeadings is the fixed remediation-doc skeleton every phase's
// docs.path target must contain (Phase Capability Contract). The headings are
// the remediation question-space an agent walks when a finding fires; keeping
// them uniform lets a doc-search topic resolve to the exact section with no
// per-phase glue. See scenarios/test-genie/docs/concepts/phase-capability-contract.md.
var RequiredDocHeadings = []string{
	"North Star",
	"The rungs and their gates",
	"What each finding means",
	"The canonical fix",
	"How to verify",
}

// validateDocsSkeleton checks that a resolved docs.path target contains all of
// the required H2 headings. Advisory (WARNING) during rollout; graduates to
// gating for compliant phases in a later wave. No-op when the file is unreadable
// (validateDocs already reported it absent).
func validateDocsSkeleton(report *Report, resolvedPath, docsPath string) {
	file, err := os.Open(resolvedPath)
	if err != nil {
		return
	}
	defer file.Close()

	present := map[string]bool{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		heading, ok := h2Heading(scanner.Text())
		if !ok {
			continue
		}
		for _, required := range RequiredDocHeadings {
			if strings.EqualFold(heading, required) {
				present[required] = true
			}
		}
	}

	var missing []string
	for _, required := range RequiredDocHeadings {
		if !present[required] {
			missing = append(missing, required)
		}
	}
	if len(missing) == 0 {
		return
	}
	report.add(Finding{
		Code:     CodeDocsSkeletonIncomplete,
		Severity: SeverityError,
		Title:    "Remediation doc does not follow the required skeleton",
		Message: fmt.Sprintf(
			"docs.path %q is missing required H2 heading(s): %s. The remediation-doc skeleton is what lets a run's doc-search topics resolve to the right section.",
			docsPath, strings.Join(missing, ", ")),
		Location:    docsPath,
		Remediation: "Add the missing H2 heading(s), or seed the skeleton with `test-genie phase scaffold " + report.Phase + "`. See the Phase Capability Contract.",
	})
}

// h2Heading returns the text of a level-2 Markdown heading ("## Foo"), if the
// line is one. It ignores deeper headings (### ...).
func h2Heading(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "##")), true
}

// validateMaturityContract enforces the first-class North Star and the gated
// ladder shape (Phase Capability Contract) on the embedded maturity spec.
// Advisory (WARNING) during rollout. No-op when no maturity spec is declared
// (the descriptor loader already reports a truly malformed one).
func validateMaturityContract(report *Report, descriptor providerdescriptor.Descriptor) {
	spec := descriptor.MaturitySpec
	if spec == nil {
		return
	}
	loc := providerdescriptor.RelPath + ":maturity"
	// The per-capability ladders are the North Star SSOT — the standing surfaces
	// the focus capability's ladder, not the aggregate phase ladder. Enforce the
	// contract on the capability ladders when present; only fall back to the
	// phase-level ladder for single-ladder providers.
	if len(spec.Capabilities) == 0 {
		validateLadder(report, "phase", loc+".levels", spec.Levels)
		return
	}
	for _, capability := range spec.Capabilities {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			id = strings.TrimSpace(capability.Label)
		}
		validateLadder(report, "capability "+id, fmt.Sprintf("%s.capabilities[%s].levels", loc, id), capability.Levels)
	}
}

// validateLadder checks one ladder for the North Star aspiration on its top rung
// and for the single next-unlock on every non-top rung. The load-bearing fields
// are capability_summary (what the rung/ceiling means — the North Star at the
// top) and next_unlock (the single next move the standing surfaces); entry/exit
// criteria are optional prose a provider may add but are not required.
func validateLadder(report *Report, label, location string, levels []assessment.Level) {
	if len(levels) == 0 {
		return
	}
	top := levels[len(levels)-1]
	if strings.TrimSpace(top.CapabilitySummary) == "" {
		report.add(Finding{
			Code:     CodeNorthStarMissing,
			Severity: SeverityError,
			Title:    "Ladder top rung has no North Star",
			Message: fmt.Sprintf(
				"The %s ladder's top rung %q declares no aspiration (capability_summary). The top rung must state what maximum maturity looks like — the first-class North Star.",
				label, firstNonEmpty(top.ID, top.Name)),
			Location:    location,
			Remediation: "Set capability_summary on the top ladder rung to the North Star aspiration for this capability.",
		})
	}
	for i, level := range levels {
		isTop := i == len(levels)-1
		// Every non-top rung must name the single next unlock — the field the
		// per-phase standing surfaces as the next move.
		if !isTop && strings.TrimSpace(level.NextUnlock) == "" {
			report.add(Finding{
				Code:     CodeLadderIncomplete,
				Severity: SeverityError,
				Title:    "Ladder rung has no next unlock",
				Message: fmt.Sprintf(
					"The %s ladder rung %q declares no next_unlock, so the standing cannot direct the single next move toward the rung above it.",
					label, firstNonEmpty(level.ID, level.Name)),
				Location:    location,
				Remediation: "Set next_unlock on every non-top ladder rung to the single highest-unlock next move.",
			})
		}
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
