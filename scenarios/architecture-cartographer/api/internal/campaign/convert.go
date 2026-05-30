package campaign

import (
	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

// sourceToken maps the shared FindingSource enum to its lower-case token.
func sourceToken(s architecturev1.FindingSource) string {
	switch s {
	case architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE:
		return "structure"
	case architecturev1.FindingSource_FINDING_SOURCE_CLI:
		return "cli"
	case architecturev1.FindingSource_FINDING_SOURCE_UI:
		return "ui"
	case architecturev1.FindingSource_FINDING_SOURCE_DOCS:
		return "docs"
	case architecturev1.FindingSource_FINDING_SOURCE_STANDARDS:
		return "standards"
	case architecturev1.FindingSource_FINDING_SOURCE_ARCHITECTURE:
		return "architecture"
	case architecturev1.FindingSource_FINDING_SOURCE_TIDINESS:
		return "tidiness"
	default:
		return "unspecified"
	}
}

// severityToken maps the shared FindingSeverity enum to its lower-case
// token (matching the cartographer's severity vocabulary).
func severityToken(s architecturev1.FindingSeverity) string {
	switch s {
	case architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER:
		return "blocker"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR:
		return "error"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING:
		return "warn"
	case architecturev1.FindingSeverity_FINDING_SEVERITY_INFO:
		return "info"
	default:
		return "unspecified"
	}
}

// effortToken maps the shared EffortHint enum to its lower-case token.
func effortToken(e architecturev1.EffortHint) string {
	switch e {
	case architecturev1.EffortHint_EFFORT_HINT_TRIVIAL:
		return EffortTrivial
	case architecturev1.EffortHint_EFFORT_HINT_SMALL:
		return EffortSmall
	case architecturev1.EffortHint_EFFORT_HINT_MEDIUM:
		return EffortMedium
	case architecturev1.EffortHint_EFFORT_HINT_LARGE:
		return EffortLarge
	default:
		return EffortUnspecified
	}
}

// fromProto converts an ingested ArchitectureFinding into the tracker's
// Finding model. The stable ID is RECOMPUTED via the shared findingid
// helper (never trusting the caller's stamp) so reconciliation is anchored
// to the canonical afid for the (scenario, source, code, locations) tuple.
// Status/note are NOT carried from the wire — lifecycle is the tracker's
// own state, always starting at detected on first ingest. Effort IS carried
// (advisory ranking input, never part of the stable-id identity).
func fromProto(scenario string, pf *architecturev1.ArchitectureFinding) Finding {
	if pf == nil {
		return Finding{}
	}
	// The campaign's scenario is authoritative; fall back to the finding's
	// own scenario only when the campaign scenario is blank.
	sc := scenario
	if sc == "" {
		sc = pf.GetScenario()
	}
	f := Finding{
		Scenario:   sc,
		Source:     sourceToken(pf.GetSource()),
		Code:       pf.GetCode(),
		Severity:   severityToken(pf.GetSeverity()),
		Locations:  pf.GetLocations(),
		Domains:    pf.GetDomains(),
		Message:    pf.GetMessage(),
		Suggestion: pf.GetSuggestion(),
		Status:     StatusDetected,
		Effort:     effortToken(pf.GetEffort()),
	}
	// Recompute the afid against the campaign scenario so a finding whose
	// wire `scenario` differs still keys consistently within this campaign.
	f.StableID = findingid.Compute(findingid.Inputs{
		Scenario:  sc,
		Source:    pf.GetSource(),
		Code:      pf.GetCode(),
		Locations: pf.GetLocations(),
	})
	return f
}
