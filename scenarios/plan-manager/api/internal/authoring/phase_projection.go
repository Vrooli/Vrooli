package authoring

import (
	"fmt"
	"regexp"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

var authoredListPrefix = regexp.MustCompile(`^\d+\.\s+`)

func splitLines(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSpace(authoredListPrefix.ReplaceAllString(trimmed, ""))
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func applyPhaseField(phase *PhaseDraft, field PhaseField, content string) error {
	content = strings.TrimSpace(content)
	switch field {
	case PhaseFieldTitle:
		phase.Title = content
	case PhaseFieldIntent:
		phase.Intent = content
	case PhaseFieldReferences:
		refs, err := parseReferencesContent(content)
		if err != nil {
			return ErrAuthoredMarkup{SectionKey: SectionPhases, Reason: err.Error()}
		}
		phase.References = refs
		if len(refs) > 0 {
			phase.NoCodeRefsReason = ""
		}
	case PhaseFieldAffectedAreas:
		phase.AffectedAreas = splitLines(content)
	case PhaseFieldSteps:
		phase.Steps = splitLines(content)
	case PhaseFieldExpectedOutputs:
		phase.ExpectedOutputs = splitLines(content)
	case PhaseFieldValidation:
		phase.Validation = content
	case PhaseFieldRisksHazards:
		phase.RisksHazards = splitLines(content)
	case PhaseFieldHandoffNotes:
		phase.HandoffNotes = content
	case PhaseFieldRequiredReading:
		phase.RequiredReading = splitLines(content)
	case PhaseFieldReminders:
		phase.Reminders = splitLines(content)
	case PhaseFieldAcceptance:
		phase.Acceptance = content
	case PhaseFieldNoCodeRefsReason:
		phase.NoCodeRefsReason = content
		if content != "" {
			phase.References = nil
		}
	case PhaseFieldRelevantContext:
		// Free-form phase context lines are classified as notes only — prose must
		// never become an executable command argv. Executable setup context flows
		// through typed context-submit/candidate acceptance.
		phase.RelevantContext = append(phase.RelevantContext, noteContextItemsFromLines(content, phase.ID)...)
	default:
		return ErrInvalidSession{Reason: "unknown phase field " + string(field)}
	}
	return nil
}

func syncPhaseSection(sess Session) Session {
	if len(sess.PhaseDrafts) == 0 {
		return sess
	}
	idx := indexOf(sess.Sections, SectionPhases)
	if idx < 0 {
		return sess
	}
	sess.Sections[idx].Content = renderPhaseDrafts(sess.PhaseDrafts)
	sess.Sections[idx].Filled = strings.TrimSpace(sess.Sections[idx].Content) != ""
	sess.Sections[idx].Autofilled = false
	sess.CurrentSectionKey = firstUnfilledMandatory(sess.Sections)
	return sess
}

func syncContextSection(sess Session) Session {
	idx := indexOf(sess.Sections, SectionRelevantContext)
	if idx < 0 {
		return sess
	}
	sess.Sections[idx].Content = renderContextItems(sess.RelevantContext)
	sess.Sections[idx].Filled = strings.TrimSpace(sess.Sections[idx].Content) != ""
	sess.Sections[idx].Autofilled = false
	return sess
}

func renderPhaseDrafts(phases []PhaseDraft) string {
	var b strings.Builder
	for i, ph := range phases {
		order := ph.Order
		if order <= 0 {
			order = i + 1
		}
		fmt.Fprintf(&b, "### Phase %d — %s\n", order, ph.Title)
		if ph.Intent != "" {
			fmt.Fprintf(&b, "- Intent: %s\n", ph.Intent)
		}
		if ph.Acceptance != "" {
			fmt.Fprintf(&b, "- Acceptance: %s\n", ph.Acceptance)
		}
		context := append([]planmodel.RelevantContextItem(nil), ph.RelevantContext...)
		context = append(context, contextItemsFromRequiredReading(ph.RequiredReading, ph.ID)...)
		if len(context) > 0 {
			b.WriteString("- Relevant context:\n")
			for _, item := range context {
				fmt.Fprintf(&b, "  - %s\n", renderContextItemSummary(item))
			}
		}
		if len(ph.Reminders) > 0 {
			b.WriteString("- Reminders:\n")
			for _, item := range ph.Reminders {
				fmt.Fprintf(&b, "  - %s\n", item)
			}
		}
		if len(ph.References) > 0 {
			b.WriteString("- References:\n")
			for _, ref := range ph.References {
				fmt.Fprintf(&b, "  - [%s: %s]\n", referenceMarker(ref.Kind), ref.Target)
			}
		}
		if ph.NoCodeRefsReason != "" {
			fmt.Fprintf(&b, "- NO_CODE_REFS: %s\n", ph.NoCodeRefsReason)
		}
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func referenceMarker(k planmodel.ReferenceKind) string {
	switch k {
	case planmodel.ReferenceReq:
		return "REQ"
	case planmodel.ReferenceDoc:
		return "DOC"
	default:
		return "CODE"
	}
}

func findDraft(phases []PhaseDraft, id string) (PhaseDraft, bool) {
	if strings.TrimSpace(id) == "" && len(phases) > 0 {
		return phases[0], true
	}
	for _, ph := range phases {
		if ph.ID == id || fmt.Sprint(ph.Order) == id {
			return ph, true
		}
	}
	return PhaseDraft{}, false
}

func indexOfDraft(phases []PhaseDraft, id string) int {
	if strings.TrimSpace(id) == "" && len(phases) > 0 {
		return 0
	}
	for i, ph := range phases {
		if ph.ID == id || fmt.Sprint(ph.Order) == id {
			return i
		}
	}
	return -1
}

func nextIncompletePhaseID(phases []PhaseDraft) string {
	for _, ph := range phases {
		if len(phaseViolations(ph)) > 0 {
			return ph.ID
		}
	}
	return ""
}

func renumberPhaseDrafts(phases []PhaseDraft) {
	for i := range phases {
		phases[i].Order = i + 1
	}
}

// sessionToPlan maps a finalized session's sections into the structured plans
// model. The prose sections map directly to the matching plan fields; the
// references and phases sections carry authored markup (the same [CODE:]/[REQ:]/
// [DOC:] locators and `### Phase N — Title` headings the plans renderer emits),
// so they are parsed through the plans-domain markdown parser — the one SSOT for
// that grammar — by assembling a minimal markdown view and re-reading the
// references/phases it recovers. The prose fields are taken verbatim from the
// sections (not re-extracted) so authored content is never lossily reshaped. The
// regression anchor section is parsed into typed anchor fields when it uses the
// rendered structure; legacy prose remains marked as legacy/degraded.
