package playbooks

import (
	"fmt"

	"test-genie/internal/evidence"
	"test-genie/internal/playbooks/execution"
)

// evidenceObservations runs the shared browser-evidence analyzer over a
// playbooks workflow timeline and renders its findings as additive
// observations.
//
// The findings are additive: a workflow's own assertion pass/fail remains the
// primary verdict, so these observations never flip a passing workflow to
// failed. They exist to surface browser-level problems the workflow assertions
// did not assert on — console errors, failed network requests, uncaught page
// exceptions, and a blank final DOM — using the same single analyzer
// (internal/evidence) that the smoke phase uses, so the severity rules are
// identical across phases:
//
//   - failed network requests / uncaught page exceptions → error observations;
//   - console errors → an error observation (counted, surfaced, non-fatal);
//   - console warnings → a warning observation.
//
// A clean timeline (no console errors/warnings, no network failures, no page
// errors) yields no observations.
func evidenceObservations(workflowFile string, parsed *execution.ParsedTimeline) []Observation {
	ev := execution.ToEvidence(parsed, execution.ToEvidenceOptions{Label: workflowFile})
	verdict := evidence.Analyze(ev)

	var obs []Observation

	if verdict.NetworkFailureCount > 0 {
		obs = append(obs, NewErrorObservation(fmt.Sprintf(
			"%s: %d failed network request(s) observed: %s",
			workflowFile, verdict.NetworkFailureCount, summarizeNetwork(ev.Network))))
	}
	for _, pe := range ev.PageErrors {
		obs = append(obs, NewErrorObservation(fmt.Sprintf(
			"%s: page error: %s", workflowFile, pe.Message)))
	}
	if verdict.ConsoleErrorCount > 0 {
		obs = append(obs, NewErrorObservation(fmt.Sprintf(
			"%s: %d console error(s) observed", workflowFile, verdict.ConsoleErrorCount)))
	}
	if verdict.ConsoleWarningCount > 0 {
		obs = append(obs, NewWarningObservation(fmt.Sprintf(
			"%s: %d console warning(s) observed", workflowFile, verdict.ConsoleWarningCount)))
	}

	return obs
}

// summarizeNetwork renders a short, capped description of failed requests.
func summarizeNetwork(failures []evidence.NetworkEntry) string {
	const cap = 3
	out := ""
	for i, f := range failures {
		if i >= cap {
			out += fmt.Sprintf(", ... and %d more", len(failures)-cap)
			break
		}
		if i > 0 {
			out += ", "
		}
		switch {
		case f.Status != nil:
			out += fmt.Sprintf("HTTP %d %s", *f.Status, f.URL)
		case f.ErrorText != "":
			out += fmt.Sprintf("%s %s", f.ErrorText, f.URL)
		default:
			out += f.URL
		}
	}
	return out
}
