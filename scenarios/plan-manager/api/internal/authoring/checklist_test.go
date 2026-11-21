package authoring

import (
	"testing"

	planmodel "plan-manager/internal/planmodel"

	"github.com/stretchr/testify/require"
)

// TestPhaseChecklistAgreesWithDerivedCursor is the form-not-wizard property:
// the first missing checklist item is ALWAYS the derived cursor
// (nextMissingPhaseField), for every partial fill order — the checklist and
// the guided cursor can never disagree.
func TestPhaseChecklistAgreesWithDerivedCursor(t *testing.T) {
	fills := []func(*PhaseDraft){
		func(p *PhaseDraft) { p.Title = "Title" },
		func(p *PhaseDraft) { p.Intent = "Intent" },
		func(p *PhaseDraft) { p.NoCodeRefsReason = "NO_CODE_REFS: fixture" },
		func(p *PhaseDraft) { p.Steps = []string{"step"} },
		func(p *PhaseDraft) { p.Validation = "go test ./..." },
		func(p *PhaseDraft) { p.Acceptance = "All green." },
		func(p *PhaseDraft) {
			p.RelevantContext = []planmodel.RelevantContextItem{{
				Kind: planmodel.RelevantContextNote, Label: "n", Reason: "r", Instruction: "i",
			}}
		},
	}
	// Apply fills cumulatively in order; after each step the first checklist
	// gap must equal the cursor.
	var phase PhaseDraft
	for i := 0; i <= len(fills); i++ {
		checkChecklistCursorAgreement(t, phase)
		if i < len(fills) {
			fills[i](&phase)
		}
	}
	// And in a scrambled order (fields are order-free).
	phase = PhaseDraft{}
	for _, i := range []int{4, 1, 6, 0, 5, 2, 3} {
		fills[i](&phase)
		checkChecklistCursorAgreement(t, phase)
	}
}

func checkChecklistCursorAgreement(t *testing.T, phase PhaseDraft) {
	t.Helper()
	cursor := nextMissingPhaseField(phase)
	var firstGap string
	for _, item := range phaseChecklist(phase) {
		if item.State == planmodel.ChecklistMissing {
			firstGap = item.Key
			break
		}
	}
	require.Equal(t, string(cursor), firstGap,
		"first missing checklist item must equal the derived cursor (phase=%+v)", phase)
}

// TestPhaseChecklistDisclosesAllSevenFields: the complete field set is visible
// the moment a phase exists — nothing is revealed only after a submission.
func TestPhaseChecklistDisclosesAllSevenFields(t *testing.T) {
	items := phaseChecklist(PhaseDraft{Title: "T", Intent: "I"})
	require.Len(t, items, 7)
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	require.Equal(t, []string{
		"title", "intent", "references", "steps", "validation", "acceptance", "relevant_context",
	}, keys)
	require.Equal(t, planmodel.ChecklistFilled, items[0].State)
	require.Equal(t, planmodel.ChecklistFilled, items[1].State)
	for _, item := range items[2:] {
		require.Equal(t, planmodel.ChecklistMissing, item.State, "field %s must be disclosed as missing", item.Key)
	}
}

// TestPhaseChecklistFlagsAcceptanceDuplicatingValidation: quality violations
// surface as state=violation with the reason, not as silently "filled".
func TestPhaseChecklistFlagsAcceptanceDuplicatingValidation(t *testing.T) {
	items := phaseChecklist(PhaseDraft{Validation: "go test ./...", Acceptance: "go test ./..."})
	var acceptance planmodel.ChecklistItem
	for _, item := range items {
		if item.Key == "acceptance" {
			acceptance = item
		}
	}
	require.Equal(t, planmodel.ChecklistViolation, acceptance.State)
	require.Contains(t, acceptance.Detail, "duplicates validation")
}

// TestSessionChecklistCoversSectionsGatesAndPhases: the session-wide map
// discloses mandatory sections, the gated inputs, and one rollup per phase.
func TestSessionChecklistCoversSectionsGatesAndPhases(t *testing.T) {
	sess := Session{
		Sections: newSkeleton(),
		PhaseDrafts: []PhaseDraft{
			{Order: 1, Title: "Done", Intent: "i", Steps: []string{"s"}, Validation: "v", Acceptance: "a", NoCodeRefsReason: "NO_CODE_REFS: x", RelevantContext: []planmodel.RelevantContextItem{{Kind: planmodel.RelevantContextNote, Label: "l", Reason: "r", Instruction: "i"}}},
			{Order: 2, Title: "Partial"},
		},
	}
	items := sessionChecklist(sess)
	byKey := map[string]planmodel.ChecklistItem{}
	for _, item := range items {
		byKey[item.Key] = item
	}
	require.Contains(t, byKey, "purpose")
	require.Equal(t, planmodel.ChecklistMissing, byKey["purpose"].State)
	require.Contains(t, byKey, "references")
	require.Contains(t, byKey, "acceptance_boundary")
	require.Contains(t, byKey, "relevant_context")
	require.Contains(t, byKey, "skill_context")
	require.Contains(t, byKey, "phase:1")
	require.Equal(t, planmodel.ChecklistFilled, byKey["phase:1"].State)
	require.Contains(t, byKey, "phase:2")
	require.Equal(t, planmodel.ChecklistMissing, byKey["phase:2"].State)
	require.Contains(t, byKey["phase:2"].Detail, "steps")
}

// TestSessionChecklistDisclosesMissingPhases: a session with no phases says so.
func TestSessionChecklistDisclosesMissingPhases(t *testing.T) {
	items := sessionChecklist(Session{Sections: newSkeleton()})
	var phases planmodel.ChecklistItem
	for _, item := range items {
		if item.Key == "phases" {
			phases = item
		}
	}
	require.Equal(t, planmodel.ChecklistMissing, phases.State)
}
