package authoring_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"plan-manager/internal/authoring"

	"github.com/stretchr/testify/require"
)

func phaseWrite(ref, field, content string) authoring.FieldWrite {
	return authoring.FieldWrite{PhaseRef: ref, PhaseField: authoring.PhaseField(field), Content: content}
}

func sectionWrite(key authoring.SectionKey, content string) authoring.FieldWrite {
	return authoring.FieldWrite{SectionKey: key, Content: content}
}

// TestSubmitFieldsMixedBatch: one call carries sections AND multiple phases;
// everything lands with a legible per-item summary.
func TestSubmitFieldsMixedBatch(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Mixed batch", "", "")
	require.NoError(t, err)
	_, p1, _, _, err := svc.AddPhase(ctx, sess.ID, "One", "First.")
	require.NoError(t, err)
	_, p2, _, _, err := svc.AddPhase(ctx, sess.ID, "Two", "Second.")
	require.NoError(t, err)

	_, results, step, err := svc.SubmitFields(ctx, sess.ID, []authoring.FieldWrite{
		sectionWrite(authoring.SectionPurpose, "Batch everything."),
		sectionWrite(authoring.SectionProblemStatement, "Drip-feed is slow."),
		phaseWrite(p1.ID, "steps", "step a\nstep b"),
		phaseWrite(p1.ID, "validation", "go test ./..."),
		phaseWrite("2", "steps", "step c"), // by order number
		phaseWrite(p2.ID, "acceptance", "Batch lands in one call."),
	})
	require.NoError(t, err)
	require.Len(t, results, 6)
	for i, result := range results {
		require.True(t, result.Accepted, "item %d must be accepted: %s", i, result.Summary)
		require.Equal(t, i, result.Index)
		require.NotEmpty(t, result.Summary)
	}
	require.Contains(t, results[2].Summary, "2 ordered step(s)")
	require.NotEmpty(t, step.Checklist, "the batch response step must carry the full-disclosure checklist")

	final, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	requireSectionContent(t, final, authoring.SectionPurpose, "Batch everything.")
	require.Equal(t, []string{"step a", "step b"}, final.PhaseDrafts[0].Steps)
	require.Equal(t, "go test ./...", final.PhaseDrafts[0].Validation)
	require.Equal(t, []string{"step c"}, final.PhaseDrafts[1].Steps)
	require.Equal(t, "Batch lands in one call.", final.PhaseDrafts[1].Acceptance)
}

func TestSubmitFieldsNormalizesNumberedPhaseSteps(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Numbered steps", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Implement", "Normalize authored step input.")
	require.NoError(t, err)

	updated, violations, _, err := svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldSteps, "1. Add proto fields\n2. Wire the CLI\n- Run focused tests")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "the phase is still incomplete; this assertion keeps the test focused on step parsing")
	require.Equal(t, []string{"Add proto fields", "Wire the CLI", "Run focused tests"}, updated.PhaseDrafts[0].Steps)
}

// TestSubmitFieldsPartialRejection: invalid items are rejected with correct
// indices while the rest of the batch lands (never all-or-nothing).
func TestSubmitFieldsPartialRejection(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Partial rejection", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "One", "First.")
	require.NoError(t, err)

	_, results, _, err := svc.SubmitFields(ctx, sess.ID, []authoring.FieldWrite{
		sectionWrite(authoring.SectionPurpose, "Valid purpose."),           // 0 accepted
		phaseWrite(phase.ID, "steps", "step a"),                            // 1 accepted
		phaseWrite(phase.ID, "bogus_field", "content"),                     // 2 rejected: unknown field
		phaseWrite(phase.ID, "validation", "go test ./..."),                // 3 accepted
		phaseWrite(phase.ID, "acceptance", "go test ./..."),                // 4 rejected: duplicates validation
		sectionWrite("nonexistent_section", "content"),                     // 5 rejected: unknown section
		phaseWrite("99", "steps", "content"),                               // 6 rejected: unknown phase
		phaseWrite(phase.ID, "acceptance", "Distinct outcome gate happy."), // 7 accepted
	})
	require.NoError(t, err)
	require.Len(t, results, 8)
	accepted := map[int]bool{0: true, 1: true, 3: true, 7: true}
	for i, result := range results {
		require.Equal(t, i, result.Index)
		require.Equal(t, accepted[i], result.Accepted, "item %d: %s", i, result.Summary)
	}
	require.Contains(t, results[2].Summary, "unknown phase field")
	require.Contains(t, results[4].Summary, "must not be identical to its validation")
	require.NotEmpty(t, results[4].Violations)

	final, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, "Distinct outcome gate happy.", final.PhaseDrafts[0].Acceptance,
		"the rejected duplicate acceptance must not have been applied; the later valid one must")
	require.Equal(t, "go test ./...", final.PhaseDrafts[0].Validation)
}

// TestSubmitFieldsEmptyBatchIsCallError: a malformed request (no items) is a
// whole-call error, not an empty success.
func TestSubmitFieldsEmptyBatchIsCallError(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Empty batch", "", "")
	require.NoError(t, err)
	_, _, _, err = svc.SubmitFields(ctx, sess.ID, nil)
	require.Error(t, err)
}

// TestSubmitFieldsIdempotentResend: re-sending the same batch yields the same
// end state (writes are absolute, not accumulative) — except relevant_context
// note lines, which are additive by design and excluded here.
func TestSubmitFieldsIdempotentResend(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Idempotent resend", "", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "One", "First.")
	require.NoError(t, err)

	batch := []authoring.FieldWrite{
		sectionWrite(authoring.SectionPurpose, "Same purpose."),
		phaseWrite(phase.ID, "steps", "step a\nstep b"),
		phaseWrite(phase.ID, "validation", "go test ./..."),
	}
	_, _, _, err = svc.SubmitFields(ctx, sess.ID, batch)
	require.NoError(t, err)
	first, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)

	_, results, _, err := svc.SubmitFields(ctx, sess.ID, batch)
	require.NoError(t, err)
	for _, result := range results {
		require.True(t, result.Accepted)
	}
	second, _, err := svc.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, first.PhaseDrafts, second.PhaseDrafts)
	require.Equal(t, first.Sections, second.Sections)
}

// TestSubmitFieldsEquivalentToSequentialSingleWrites is the equivalence
// property: one batch of N writes produces the same end state as N sequential
// SubmitSection/SubmitPhaseField calls (single-field RPCs are wrappers over
// the same apply path — no drift possible).
func TestSubmitFieldsEquivalentToSequentialSingleWrites(t *testing.T) {
	ctx := context.Background()
	writes := func(phaseID string) []authoring.FieldWrite {
		return []authoring.FieldWrite{
			sectionWrite(authoring.SectionPurpose, "Equivalence."),
			sectionWrite(authoring.SectionScope, "In: everything."),
			phaseWrite(phaseID, "steps", "one\ntwo\nthree"),
			phaseWrite(phaseID, "validation", "go test ./..."),
			phaseWrite(phaseID, "acceptance", "All assertions hold."),
			phaseWrite(phaseID, "no_code_refs_reason", "NO_CODE_REFS: fixture"),
			phaseWrite(phaseID, "relevant_context", "NO_CONTEXT: fixture."),
		}
	}

	batchSvc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	batchSess, _, err := batchSvc.StartSession(ctx, "Equivalence", "equivalence-batch", "")
	require.NoError(t, err)
	_, batchPhase, _, _, err := batchSvc.AddPhase(ctx, batchSess.ID, "P", "Intent.")
	require.NoError(t, err)
	_, results, _, err := batchSvc.SubmitFields(ctx, batchSess.ID, writes(batchPhase.ID))
	require.NoError(t, err)
	for _, result := range results {
		require.True(t, result.Accepted, result.Summary)
	}

	seqSvc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	seqSess, _, err := seqSvc.StartSession(ctx, "Equivalence", "equivalence-seq", "")
	require.NoError(t, err)
	_, seqPhase, _, _, err := seqSvc.AddPhase(ctx, seqSess.ID, "P", "Intent.")
	require.NoError(t, err)
	for _, write := range writes(seqPhase.ID) {
		if write.IsSection() {
			_, _, _, err = seqSvc.SubmitSection(ctx, seqSess.ID, write.SectionKey, write.Content)
		} else {
			_, _, _, err = seqSvc.SubmitPhaseField(ctx, seqSess.ID, write.PhaseRef, write.PhaseField, write.Content)
		}
		require.NoError(t, err)
	}

	batchFinal, _, err := batchSvc.GetSession(ctx, batchSess.ID)
	require.NoError(t, err)
	seqFinal, _, err := seqSvc.GetSession(ctx, seqSess.ID)
	require.NoError(t, err)

	normalizePhase := func(p authoring.PhaseDraft) authoring.PhaseDraft {
		p.ID = ""
		for i := range p.RelevantContext {
			p.RelevantContext[i].ID = ""
			p.RelevantContext[i].PhaseID = ""
		}
		return p
	}
	require.Len(t, batchFinal.PhaseDrafts, 1)
	require.Len(t, seqFinal.PhaseDrafts, 1)
	require.Equal(t, normalizePhase(seqFinal.PhaseDrafts[0]), normalizePhase(batchFinal.PhaseDrafts[0]))
	for _, key := range []authoring.SectionKey{authoring.SectionPurpose, authoring.SectionScope} {
		batchSec, _ := sectionOf(batchFinal, key)
		seqSec, _ := sectionOf(seqFinal, key)
		require.Equal(t, seqSec.Content, batchSec.Content)
	}
}

func sectionOf(sess authoring.Session, key authoring.SectionKey) (authoring.Section, bool) {
	for _, sec := range sess.Sections {
		if sec.Key == key {
			return sec, true
		}
	}
	return authoring.Section{}, false
}

// TestSubmitFieldsConcurrentWithSingleWrites: a batch racing single-field
// writes on the same session loses nothing (-race).
func TestSubmitFieldsConcurrentWithSingleWrites(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})

	for i := 0; i < 10; i++ {
		sess, _, err := svc.StartSession(ctx, fmt.Sprintf("Batch race %d", i), "", "")
		require.NoError(t, err)
		_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "P", "Intent.")
		require.NoError(t, err)

		var wg sync.WaitGroup
		errs := make([]error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, _, errs[0] = svc.SubmitFields(ctx, sess.ID, []authoring.FieldWrite{
				sectionWrite(authoring.SectionPurpose, "From the batch."),
				phaseWrite(phase.ID, "steps", "batch step"),
			})
		}()
		go func() {
			defer wg.Done()
			_, _, _, errs[1] = svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldValidation, "go test ./...")
		}()
		wg.Wait()
		require.NoError(t, errs[0])
		require.NoError(t, errs[1])

		final, _, err := svc.GetSession(ctx, sess.ID)
		require.NoError(t, err)
		requireSectionContent(t, final, authoring.SectionPurpose, "From the batch.")
		require.Equal(t, []string{"batch step"}, final.PhaseDrafts[0].Steps)
		require.Equal(t, "go test ./...", final.PhaseDrafts[0].Validation)
	}
}

// TestSubmitFieldsAuthorsFullPlanInThreeMutations proves the ≤ 3+N budget at
// the service level: start (1), one sections batch (2), one per-phase batch
// (3), finalize (4) — a complete 1-phase plan in 3+N mutation calls.
func TestSubmitFieldsAuthorsFullPlanInThreeMutations(t *testing.T) {
	ctx := context.Background()
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})

	sess, _, err := svc.StartSession(ctx, "Batch authored plan", "batch-authored-plan", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Only phase", "Do the work.")
	require.NoError(t, err)

	_, results, _, err := svc.SubmitFields(ctx, sess.ID, []authoring.FieldWrite{
		sectionWrite(authoring.SectionPurpose, "Prove batch authoring."),
		sectionWrite(authoring.SectionProblemStatement, "Drip-feed costs 40 round trips."),
		sectionWrite(authoring.SectionTargetOutcome, "A full plan lands in 3+N calls."),
		sectionWrite(authoring.SectionScope, "In: authoring service."),
		sectionWrite(authoring.SectionTechnicalApproach, "One batch RPC over the shared apply path."),
		sectionWrite(authoring.SectionAcceptanceBoundary, "acceptance_allow:\n- scenarios/plan-manager/**"),
		sectionWrite(authoring.SectionReferences, "NO_CODE_REFS: unit fixture with no connected production code"),
		sectionWrite(authoring.SectionRegressionAnchor, "Strategy: change_boundary"),
		sectionWrite(authoring.SectionValidationStrategy, "Run the authoring unit suite."),
		sectionWrite(authoring.SectionDefinitionOfDone, "Finalize succeeds; plan retrievable."),
		sectionWrite(authoring.SectionRelevantContext, "NO_CONTEXT: unit fixture needs no plan-wide setup.\nNO_SKILL_CONTEXT: unit fixture has no skill setup."),
	})
	require.NoError(t, err)
	for _, result := range results {
		require.True(t, result.Accepted, result.Summary)
	}

	_, results, _, err = svc.SubmitFields(ctx, sess.ID, []authoring.FieldWrite{
		phaseWrite(phase.ID, "steps", "implement\nvalidate"),
		phaseWrite(phase.ID, "validation", "go test ./internal/authoring"),
		phaseWrite(phase.ID, "acceptance", "The full suite is green."),
		phaseWrite(phase.ID, "no_code_refs_reason", "NO_CODE_REFS: fixture"),
		phaseWrite(phase.ID, "relevant_context", "NO_CONTEXT: fixture."),
	})
	require.NoError(t, err)
	for _, result := range results {
		require.True(t, result.Accepted, result.Summary)
	}

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, result.Plan.ID)
	require.Len(t, writer.created.Phases, 1)
}
