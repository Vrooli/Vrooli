package execution_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"plan-manager/internal/execution"
	"plan-manager/internal/planmodel"
	"testing"
)

func TestAdvisoryCompletionRetainsFailedEvidenceAndAssessment(t *testing.T) { // [REQ:PM-VALID-002]
	ctx := context.Background()
	plan := threePhasePlan()
	plan.Phases = plan.Phases[:1]
	plan.BaselineSet = planmodel.BaselineSetIntent{}
	validator := fakeValidator{hasResult: true, result: execution.ValidationResult{Verdict: "fail", Staleness: planmodel.StalenessDefinitelyStale, Detail: "unrelated validator finding; concurrent drift"}}
	h := newHarness(t, plan, validator)
	e, _, _, err := h.svc.Start(ctx, plan.ID, "")
	require.NoError(t, err)
	require.Empty(t, e.BaselineSet.ReceiptID, "ordinary start must not admit baseline work")
	a := execution.OutcomeAssessment{Summary: "Changed interaction observed and implementation reviewed", Evidence: []string{"review:changed-handler", "observation:interaction"}, Limitations: []string{"prior state unknown", "broad validator finding retained; attribution uncertain"}}
	_, _, _, err = h.svc.TransitionPhase(ctx, e.ID, plan.Phases[0].ID, execution.PhaseTransitionInputs{ToStatus: planmodel.PhaseStatusDone, Assessment: a})
	require.NoError(t, err)
	stored, _, _, err := h.svc.GetStatus(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, a, stored.PhaseAssessments[plan.Phases[0].ID])
	handoff, _, _, err := h.svc.Complete(ctx, e.ID, execution.CompletionInputs{})
	require.NoError(t, err)
	require.Equal(t, "fail", handoff.LastValidation.Verdict, "acceptance must not rewrite producer observations")
	require.Equal(t, a, handoff.PhaseAssessments[plan.Phases[0].ID])
	replayed, _, _, err := h.svc.Complete(ctx, e.ID, execution.CompletionInputs{})
	require.NoError(t, err)
	require.Equal(t, handoff.PhaseAssessments, replayed.PhaseAssessments)
}

func TestAdvisoryAcceptanceRequiresSupportedOutcome(t *testing.T) { // [REQ:PM-VALID-002]
	for _, a := range []execution.OutcomeAssessment{
		{},
		{Summary: "done"},
		{Summary: "feature incomplete", Evidence: []string{"review:handler"}, UnmetOutcomes: []string{"required interaction fails"}},
	} {
		plan := threePhasePlan()
		h := newHarness(t, plan, nil)
		e, _, _, err := h.svc.Start(context.Background(), plan.ID, "")
		require.NoError(t, err)
		_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, plan.Phases[0].ID, execution.PhaseTransitionInputs{ToStatus: planmodel.PhaseStatusDone, Assessment: a, ValidationOverrideReason: "do not bypass assessment"})
		require.Error(t, err)
	}
}

func TestCertificationCannotUseAssessmentOrOverrideToBypassEvidence(t *testing.T) {
	h := newHarness(t, certificationPlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", doneOverride())
	require.Error(t, err)
}

func TestAdvisoryLegacyDonePhasesRequestAssessmentBeforeCompletion(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases = plan.Phases[:1]
	plan.Phases[0].Status = planmodel.PhaseStatusDone
	h := newHarness(t, plan, nil)
	e, _, step, err := h.svc.Start(context.Background(), plan.ID, "")
	require.NoError(t, err)
	require.Equal(t, "assess-outcome", step.NextActions[0].ID)
	_, _, _, err = h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
	require.Error(t, err)
	_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", doneOverride())
	require.NoError(t, err)
	_, _, _, err = h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
	require.NoError(t, err)
}

func TestExecutionKeepsCapturedCompletionPolicyAcrossPlanEdits(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases = plan.Phases[:1]
	plan.CompletionPolicy = planmodel.CompletionPolicy{Mode: "advisory"}
	h := newHarness(t, plan, nil)
	e, started, _, err := h.svc.Start(context.Background(), plan.ID, "")
	require.NoError(t, err)
	require.Equal(t, "advisory", started.CompletionPolicy.Mode)
	require.True(t, e.CompletionPolicyCaptured)

	// A concurrent plan edit must not turn an already-running ordinary
	// execution into a certification run. The shared worktree remains editable;
	// the execution's interpretation is the stable part.
	h.store.plan.CompletionPolicy = planmodel.CompletionPolicy{Mode: "certification", Reason: "edited after execution start"}
	a := execution.OutcomeAssessment{
		Summary:  "The promised outcome was reviewed after the edit.",
		Evidence: []string{"review:captured-policy"},
	}
	_, updated, _, err := h.svc.TransitionPhase(context.Background(), e.ID, plan.Phases[0].ID, execution.PhaseTransitionInputs{
		ToStatus:   planmodel.PhaseStatusDone,
		Assessment: a,
	})
	require.NoError(t, err)
	require.Equal(t, "advisory", updated.CompletionPolicy.Mode)

	stored, statusContext, _, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, "advisory", statusContext.CompletionPolicy.Mode)
	require.Equal(t, a, stored.PhaseAssessments[plan.Phases[0].ID])
	_, _, _, err = h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
	require.NoError(t, err)
}
