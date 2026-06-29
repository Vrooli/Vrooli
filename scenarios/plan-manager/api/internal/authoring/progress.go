package authoring

import (
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

// AuthoringProgress is the compact navigation snapshot every normal mutation
// returns so a small-context agent can decide its next action WITHOUT the full
// session graph. It is computed purely from the saved session; the full state is
// available only through GetSession/PreviewPlan. RemainingRequiredInputs and
// ReadyToFinalize mirror the structure gate minus the command-reference seam
// (that runs only at Finalize), so ReadyToFinalize is a "structurally ready"
// hint, never a guarantee.
type AuthoringProgress struct {
	SessionID               string
	CurrentSectionKey       string
	CurrentPhaseID          string
	MandatorySectionsTotal  int
	MandatorySectionsFilled int
	PhasesTotal             int
	PhasesComplete          int
	RemainingRequiredInputs []string
	ReadyToFinalize         bool
}

// ComputeProgress derives the compact navigation snapshot from a saved session.
func ComputeProgress(sess Session) AuthoringProgress {
	p := AuthoringProgress{
		SessionID:         sess.ID,
		CurrentSectionKey: string(sess.CurrentSectionKey),
		CurrentPhaseID:    sess.CurrentPhaseID,
		PhasesTotal:       len(sess.PhaseDrafts),
	}
	for _, sec := range sess.Sections {
		if sec.Mandatory {
			p.MandatorySectionsTotal++
			if strings.TrimSpace(sec.Content) != "" {
				p.MandatorySectionsFilled++
			}
		}
	}
	for _, phase := range sess.PhaseDrafts {
		if len(phaseViolations(phase)) == 0 {
			p.PhasesComplete++
		}
	}
	p.RemainingRequiredInputs = remainingRequiredInputs(sess)
	p.ReadyToFinalize = !sess.Finalized && len(p.RemainingRequiredInputs) == 0
	return p
}

// remainingRequiredInputs lists, in the order the wizard would resolve them, the
// inputs still blocking a structurally valid finalize. It deliberately omits the
// command-reference seam (validated at Finalize) so it stays a cheap pure
// function suitable for every mutation response.
func remainingRequiredInputs(sess Session) []string {
	var out []string
	for _, sec := range sess.Sections {
		// References and the change boundary are mandatory but own dedicated gates
		// (below) that encode their NO_CODE_REFS / OPERATOR_ONLY escapes, so they
		// are not listed via the generic path.
		if sec.Key == SectionReferences || sec.Key == SectionAcceptanceBoundary {
			continue
		}
		if sec.Mandatory && strings.TrimSpace(sec.Content) == "" {
			out = append(out, sec.Label+" (mandatory section)")
		}
	}
	if len(boundaryGateViolations(contentOf(sess.Sections, SectionAcceptanceBoundary))) > 0 {
		out = append(out, "Change boundary: acceptance_allow path globs or an OPERATOR_ONLY: reason")
	}
	if !hasReferencesOrNoCodeReason(contentOf(sess.Sections, SectionReferences)) {
		out = append(out, "References: at least one [CODE:]/[REQ:]/[DOC:] reference or a NO_CODE_REFS: reason")
	}
	if !globalContextResolved(sess) {
		out = append(out, "Relevant context: accept/add a global item or record a NO_CONTEXT: reason")
	}
	if len(sess.PhaseDrafts) == 0 {
		out = append(out, "Phases: add at least one structured phase")
	}
	for i, phase := range sess.PhaseDrafts {
		if missing := phaseMissingFields(phase); len(missing) > 0 {
			order := phase.Order
			if order <= 0 {
				order = i + 1
			}
			out = append(out, fmt.Sprintf("Phase %d: %s", order, strings.Join(missing, ", ")))
		}
	}
	return out
}

// phaseMissingFields returns the short field tokens a phase still needs for the
// structure gate. It mirrors phaseViolations' required-field checks (it omits the
// acceptance≠validation quality check, which is surfaced as a violation, not a
// missing input).
func phaseMissingFields(phase PhaseDraft) []string {
	var out []string
	if strings.TrimSpace(phase.Title) == "" {
		out = append(out, "title")
	}
	if strings.TrimSpace(phase.Intent) == "" {
		out = append(out, "intent")
	}
	if len(phase.Steps) == 0 {
		out = append(out, "steps")
	}
	if strings.TrimSpace(phase.Validation) == "" {
		out = append(out, "validation")
	}
	if strings.TrimSpace(phase.Acceptance) == "" {
		out = append(out, "acceptance")
	}
	if len(phase.References) == 0 && strings.TrimSpace(phase.NoCodeRefsReason) == "" {
		out = append(out, "references|no_code_refs_reason")
	}
	if !hasPhaseContextOrNoContextReason(phase) {
		out = append(out, "relevant_context|NO_CONTEXT")
	}
	return out
}

// summary builders — these produce the short, honest acknowledgement string for
// a mutation. They operate on the domain object that changed so the wire summary
// echoes exactly what was parsed, never accumulated session state.

// SectionSummary describes a section submission.
func SectionSummary(sec Section) string {
	if strings.TrimSpace(sec.Content) == "" {
		return fmt.Sprintf("cleared section %q", sec.Key)
	}
	return fmt.Sprintf("submitted section %q", sec.Key)
}

// PhaseFieldSummary describes one phase-field submission, counting parsed list
// items where the field is a list.
func PhaseFieldSummary(field PhaseField, phase PhaseDraft) string {
	switch field {
	case PhaseFieldSteps:
		return fmt.Sprintf("parsed %d ordered step(s)", len(phase.Steps))
	case PhaseFieldAffectedAreas:
		return fmt.Sprintf("parsed %d affected area(s)", len(phase.AffectedAreas))
	case PhaseFieldExpectedOutputs:
		return fmt.Sprintf("parsed %d expected output(s)", len(phase.ExpectedOutputs))
	case PhaseFieldRisksHazards:
		return fmt.Sprintf("parsed %d risk(s)/hazard(s)", len(phase.RisksHazards))
	case PhaseFieldReferences:
		return fmt.Sprintf("parsed %d reference(s)", len(phase.References))
	case PhaseFieldRequiredReading:
		return fmt.Sprintf("parsed %d required-reading line(s)", len(phase.RequiredReading))
	case PhaseFieldReminders:
		return fmt.Sprintf("parsed %d reminder(s)", len(phase.Reminders))
	case PhaseFieldRelevantContext:
		return fmt.Sprintf("recorded phase context (%d item(s) total)", len(phase.RelevantContext))
	default:
		return fmt.Sprintf("submitted phase field %q", field)
	}
}

// PhaseAddSummary describes an add-phase mutation. AddPhase submits both the
// title and the intent, so the summary names both material fields instead of
// reporting only the title (the underreported-summary friction).
func PhaseAddSummary(phase PhaseDraft) string {
	title := strings.TrimSpace(phase.Title)
	if title == "" {
		title = "(untitled)"
	}
	intent := strings.TrimSpace(phase.Intent)
	if intent == "" {
		return fmt.Sprintf("added phase %d titled %q (intent pending)", phase.Order, title)
	}
	return fmt.Sprintf("added phase %d titled %q with intent %q", phase.Order, title, intent)
}

// FindPhaseDraft resolves a phase draft by id or authored order number. It lets
// the handler edge echo the single changed phase in a mutation acknowledgement
// without re-loading or exposing the unexported lookup.
func FindPhaseDraft(sess Session, phaseID string) (PhaseDraft, bool) {
	return findDraft(sess.PhaseDrafts, phaseID)
}

// ContextItemSummary describes a context-item mutation.
func ContextItemSummary(item planmodel.RelevantContextItem) string {
	label := firstNonEmpty(item.Label, item.Target, item.Command, item.Instruction, string(item.Kind))
	return fmt.Sprintf("%s context %q", item.Kind, label)
}

// ReferenceCandidateSummary describes a reference-candidate mutation.
func ReferenceCandidateSummary(candidate ReferenceCandidate) string {
	return fmt.Sprintf("[%s: %s]", referenceMarker(candidate.Reference.Kind), candidate.Reference.Target)
}
