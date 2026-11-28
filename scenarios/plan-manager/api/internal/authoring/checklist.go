package authoring

import (
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

// Full-disclosure checklists (contract: form-not-wizard). Every guided step
// carries the COMPLETE requirement set for its scope with live status, so an
// agent never has to submit a field just to learn the next one. The derived
// cursor (nextMissingPhaseField / firstUnfilledMandatory) stays the novice
// spine; the checklist is the whole form.

// phaseFieldChecklistOrder is the canonical phase requirement order — it MUST
// match nextMissingPhaseField's resolution order (the property test enforces
// checklist-missing == cursor agreement).
var phaseFieldChecklistOrder = []PhaseField{
	PhaseFieldTitle,
	PhaseFieldIntent,
	PhaseFieldReferences,
	PhaseFieldSteps,
	PhaseFieldValidation,
	PhaseFieldAcceptance,
	PhaseFieldRelevantContext,
}

// phaseChecklist is the full 7-field requirement set for one phase draft with
// live filled/missing/violation status.
func phaseChecklist(phase PhaseDraft) []planmodel.ChecklistItem {
	items := make([]planmodel.ChecklistItem, 0, len(phaseFieldChecklistOrder))
	for _, field := range phaseFieldChecklistOrder {
		items = append(items, phaseFieldChecklistItem(phase, field))
	}
	return items
}

func phaseFieldChecklistItem(phase PhaseDraft, field PhaseField) planmodel.ChecklistItem {
	item := planmodel.ChecklistItem{Key: string(field), Label: string(field)}
	switch field {
	case PhaseFieldTitle:
		item.State = filledOrMissing(strings.TrimSpace(phase.Title) != "")
	case PhaseFieldIntent:
		item.State = filledOrMissing(strings.TrimSpace(phase.Intent) != "")
	case PhaseFieldReferences:
		item.Label = "references (or no_code_refs_reason)"
		switch {
		case len(phase.References) > 0:
			item.State = planmodel.ChecklistFilled
			item.Detail = fmt.Sprintf("%d reference(s)", len(phase.References))
		case strings.TrimSpace(phase.NoCodeRefsReason) != "":
			item.State = planmodel.ChecklistFilled
			item.Detail = "NO_CODE_REFS recorded"
		default:
			item.State = planmodel.ChecklistMissing
		}
	case PhaseFieldSteps:
		item.State = filledOrMissing(len(phase.Steps) > 0)
		if len(phase.Steps) > 0 {
			item.Detail = fmt.Sprintf("%d step(s)", len(phase.Steps))
		}
	case PhaseFieldValidation:
		item.State = filledOrMissing(strings.TrimSpace(phase.Validation) != "")
	case PhaseFieldAcceptance:
		item.State = filledOrMissing(strings.TrimSpace(phase.Acceptance) != "")
		if a, v := normalizeForCompare(phase.Acceptance), normalizeForCompare(phase.Validation); a != "" && a == v {
			item.State = planmodel.ChecklistViolation
			item.Detail = "duplicates validation — acceptance is the outcome gate, validation the checking method"
		}
	case PhaseFieldRelevantContext:
		item.Label = "relevant_context (or NO_CONTEXT: reason)"
		switch {
		case len(phase.RelevantContext) > 0:
			item.State = planmodel.ChecklistFilled
			item.Detail = fmt.Sprintf("%d item(s)", len(phase.RelevantContext))
		case hasPhaseContextOrNoContextReason(phase):
			item.State = planmodel.ChecklistFilled
			item.Detail = "NO_CONTEXT recorded"
		default:
			item.State = planmodel.ChecklistMissing
		}
	}
	return item
}

// sessionChecklist is the session-wide requirement map: every mandatory
// section, the gated inputs (boundary, references, global/skill context), and
// a rollup entry per phase — filled and missing alike (full disclosure).
func sessionChecklist(sess Session) []planmodel.ChecklistItem {
	var items []planmodel.ChecklistItem
	for _, sec := range sess.Sections {
		// References, the change boundary, and relevant context are enforced by
		// dedicated gates rather than the generic mandatory flag — disclose them
		// regardless of that flag.
		gated := sec.Key == SectionReferences || sec.Key == SectionAcceptanceBoundary || sec.Key == SectionRelevantContext
		if !sec.Mandatory && !gated {
			continue
		}
		switch sec.Key {
		case SectionAcceptanceBoundary:
			items = append(items, gatedSectionItem(sec, boundaryGateViolations(sec.Content), "acceptance_allow globs or OPERATOR_ONLY: reason"))
		case SectionReferences:
			item := planmodel.ChecklistItem{Key: string(sec.Key), Label: sec.Label}
			if hasReferencesOrNoCodeReason(sec.Content) {
				item.State = planmodel.ChecklistFilled
			} else {
				item.State = planmodel.ChecklistMissing
				item.Detail = "[CODE:]/[DOC:]/[REQ:] locators or NO_CODE_REFS: reason"
			}
			items = append(items, item)
		case SectionRelevantContext:
			items = append(items, globalContextChecklistItems(sess, sec)...)
		case SectionPhases:
			// Phase completeness is disclosed per phase draft below, not via the
			// synced markdown blob.
		default:
			items = append(items, planmodel.ChecklistItem{
				Key:   string(sec.Key),
				Label: sec.Label,
				State: filledOrMissing(strings.TrimSpace(sec.Content) != ""),
			})
		}
	}
	if len(sess.PhaseDrafts) == 0 {
		items = append(items, planmodel.ChecklistItem{
			Key:    "phases",
			Label:  "Phases",
			State:  planmodel.ChecklistMissing,
			Detail: "add at least one structured phase",
		})
		return items
	}
	for i, phase := range sess.PhaseDrafts {
		order := phase.Order
		if order <= 0 {
			order = i + 1
		}
		item := planmodel.ChecklistItem{
			Key:   fmt.Sprintf("phase:%d", order),
			Label: fmt.Sprintf("Phase %d — %s", order, firstNonEmpty(strings.TrimSpace(phase.Title), "(untitled)")),
		}
		missing := phaseMissingFields(phase)
		switch {
		case len(missing) > 0:
			item.State = planmodel.ChecklistMissing
			item.Detail = "needs " + strings.Join(missing, ", ")
		case len(phaseViolations(phase)) > 0:
			item.State = planmodel.ChecklistViolation
			item.Detail = phaseViolations(phase)[0].Message
		default:
			item.State = planmodel.ChecklistFilled
		}
		items = append(items, item)
	}
	return items
}

// globalContextChecklistItems discloses advisory context state. Missing context
// no longer blocks finalize; DiscoverSkillPack is the low-friction path for
// adding professional skill setup.
func globalContextChecklistItems(sess Session, sec Section) []planmodel.ChecklistItem {
	global := planmodel.ChecklistItem{Key: string(SectionRelevantContext), Label: sec.Label}
	if len(sess.RelevantContext) > 0 || noContextReason(sec.Content) != "" {
		global.State = planmodel.ChecklistFilled
	} else {
		global.State = planmodel.ChecklistFilled
		global.Detail = "optional: add durable setup context if it will help execution"
	}
	skill := planmodel.ChecklistItem{Key: "skill_context", Label: "Skill context decision"}
	if globalSkillContextResolved(sess) {
		skill.State = planmodel.ChecklistFilled
	} else {
		skill.State = planmodel.ChecklistMissing
		skill.Detail = "run author skill-pack with 2-5 concepts (recommended; keep most returned skills) or record NO_SKILL_CONTEXT: <reason>"
	}
	return []planmodel.ChecklistItem{global, skill}
}

func gatedSectionItem(sec Section, violations []StructureViolation, needHint string) planmodel.ChecklistItem {
	item := planmodel.ChecklistItem{Key: string(sec.Key), Label: sec.Label}
	switch {
	case len(violations) == 0:
		item.State = planmodel.ChecklistFilled
	case strings.TrimSpace(sec.Content) == "":
		item.State = planmodel.ChecklistMissing
		item.Detail = needHint
	default:
		item.State = planmodel.ChecklistViolation
		item.Detail = violations[0].Message
	}
	return item
}

func filledOrMissing(filled bool) planmodel.ChecklistState {
	if filled {
		return planmodel.ChecklistFilled
	}
	return planmodel.ChecklistMissing
}

// missingPhaseChecklistFields lists the phase fields the checklist reports as
// missing, in canonical order — the input for the batched-submit hint.
func missingPhaseChecklistFields(phase PhaseDraft) []PhaseField {
	var out []PhaseField
	for _, item := range phaseChecklist(phase) {
		if item.State == planmodel.ChecklistMissing {
			out = append(out, PhaseField(item.Key))
		}
	}
	return out
}
