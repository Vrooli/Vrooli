package execution_test

import (
	"context"
	"testing"
	"time"

	"plan-manager/internal/execution"
	internalplans "plan-manager/internal/plans"
	"plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

// --- fakes ---

// fakePlanStore is the PlanStore seam: an in-memory plan whose phases mutate on
// UpdatePhase, so transition-driven resume/complete derivation is exercised.
type fakePlanStore struct {
	plan internalplans.Plan
	err  error
}

func (f *fakePlanStore) GetPlan(_ context.Context, _ string) (internalplans.Plan, error) {
	return f.plan, f.err
}

func (f *fakePlanStore) UpdatePhase(_ context.Context, _ string, phase internalplans.Phase) (internalplans.Plan, error) {
	if f.err != nil {
		return internalplans.Plan{}, f.err
	}
	for i := range f.plan.Phases {
		if f.plan.Phases[i].ID == phase.ID {
			f.plan.Phases[i] = phase
			break
		}
	}
	return f.plan, nil
}

// fakeValidator is the Validator seam — the cheap read of the LAST STORED
// validation result. A zero value returns "no result yet" (ok=false); set
// hasResult to surface the canned result.
type fakeValidator struct {
	result    execution.ValidationResult
	hasResult bool
	err       error
}

func (f fakeValidator) LastValidation(_ context.Context, _, _ string) (execution.ValidationResult, bool, error) {
	if f.err != nil {
		return execution.ValidationResult{}, false, f.err
	}
	return f.result, f.hasResult, nil
}

// recordingSink is the VelocitySink seam, capturing emitted points.
type recordingSink struct{ emitted []execution.VelocityPoint }

func (s *recordingSink) Emit(_ context.Context, p execution.VelocityPoint) error {
	s.emitted = append(s.emitted, p)
	return nil
}

// --- harness ---

type harness struct {
	svc   execution.Service
	store *fakePlanStore
	sink  *recordingSink
	clock *mocks.FakeClock
}

func newHarness(t *testing.T, plan internalplans.Plan, validator execution.Validator) harness {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(execution.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	store := &fakePlanStore{plan: plan}
	sink := &recordingSink{}
	svc := execution.NewService(execution.Deps{
		Repo:      execution.NewSQLiteRepository(d, clk),
		Plans:     store,
		Validator: validator,
		Velocity:  sink,
		Clock:     clk,
	})
	return harness{svc: svc, store: store, sink: sink, clock: clk}
}

func threePhasePlan() internalplans.Plan {
	return internalplans.Plan{
		ID:    "plan-1",
		Slug:  "plan-1",
		Title: "Plan One",
		Phases: []internalplans.Phase{
			{ID: "ph-1", Order: 1, Title: "First", Status: internalplans.PhaseStatusTodo, RequiredReading: []string{"docs/a.md"}, Reminders: []string{"think first"}},
			{ID: "ph-2", Order: 2, Title: "Second", Status: internalplans.PhaseStatusTodo},
			{ID: "ph-3", Order: 3, Title: "Third", Status: internalplans.PhaseStatusTodo},
		},
	}
}

// --- tests ---

func TestStartLinksRunAndSetsResumePointer(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "run-abc")
	require.NoError(t, err)
	require.NotEmpty(t, e.ID)
	require.Equal(t, "plan-1", e.PlanID)
	require.Equal(t, "run-abc", e.RunID)
	require.Equal(t, "ph-1", e.CurrentPhaseID, "current pointer starts at the earliest non-done phase")
	require.False(t, e.Complete)
}

func TestStartRequiresPlanID(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	_, err := h.svc.Start(context.Background(), "  ", "")
	require.ErrorAs(t, err, &execution.ErrInvalidExecution{})
}

func TestFindingDedupIndexNormalizesTitleCase(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(execution.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	repo := execution.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	require.NoError(t, repo.SaveFinding(ctx, execution.Finding{
		ID:               "finding-1",
		ExecutionID:      "exec-1",
		Title:            "Config Drift",
		Triage:           execution.TriageCandidate,
		AttributionRunID: "run-1",
	}))
	err := repo.SaveFinding(ctx, execution.Finding{
		ID:               "finding-2",
		ExecutionID:      "exec-1",
		Title:            "  config drift  ",
		Triage:           execution.TriageCandidate,
		AttributionRunID: "run-1",
	})
	require.Error(t, err, "case/space variants must hit the store-level dedup backstop")
}

func TestResumePointDerivationEarliestNonDone(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusDone // ph-1 done
	h := newHarness(t, plan, nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, "ph-2", e.CurrentPhaseID, "resume point skips the done phase to the earliest non-done one")
}

// [REQ:PM-EXEC-001]
func TestGetStatusInjectsPhaseScopedContext(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	_, pctx, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "ph-1", pctx.CurrentPhase.ID)
	require.Equal(t, []string{"docs/a.md"}, pctx.RequiredReading)
	require.Equal(t, []string{"think first"}, pctx.Reminders)
	require.True(t, pctx.HasNext)
	require.Equal(t, "ph-2", pctx.NextPhase.ID)
	require.Equal(t, "ph-1", pctx.ResumePhaseID)
	require.Equal(t, execution.CompletenessPartial, pctx.Completeness)
}

func TestGetStatusDegradesToUnknownWhenValidatorNil(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil) // nil validator
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, pctx, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.False(t, pctx.HasValidation, "nil validator => no validation, never a false pass")
	require.Equal(t, internalplans.StalenessUnknown, pctx.Staleness)
}

func TestTransitionPhaseDelegatesAndAdvancesPointer(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	updatedE, plan, err := h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", internalplans.PhaseStatusDone)
	require.NoError(t, err)
	require.Equal(t, internalplans.PhaseStatusDone, plan.Phases[0].Status, "the plans domain owns the persisted status")
	require.Equal(t, "ph-2", updatedE.CurrentPhaseID, "pointer advances to the next non-done phase")
	require.False(t, updatedE.Complete)
}

func TestTransitionPhaseUnknownPhase(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, _, err = h.svc.TransitionPhase(context.Background(), e.ID, "nope", internalplans.PhaseStatusDone)
	require.ErrorAs(t, err, &internalplans.ErrPhaseNotFound{})
}

func TestGetNextAdvancesAndReportsComplete(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	// Drive every phase to done, then GetNext reports complete.
	for _, id := range []string{"ph-1", "ph-2", "ph-3"} {
		_, _, err = h.svc.TransitionPhase(context.Background(), e.ID, id, internalplans.PhaseStatusDone)
		require.NoError(t, err)
	}
	pctx, complete, err := h.svc.GetNext(context.Background(), e.ID)
	require.NoError(t, err)
	require.True(t, complete, "no actionable phase remains")
	require.Equal(t, "", pctx.ResumePhaseID)
	require.Equal(t, execution.CompletenessFull, pctx.Completeness)
}

// [REQ:PM-EXEC-001]
func TestGetNextAdvancesPastCurrentUnfinishedPhase(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, "ph-1", e.CurrentPhaseID)

	pctx, complete, err := h.svc.GetNext(context.Background(), e.ID)
	require.NoError(t, err)
	require.False(t, complete)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "ph-2", pctx.CurrentPhase.ID, "GetNext advances to a later actionable phase instead of recomputing the resume point")
	require.Equal(t, "ph-1", pctx.ResumePhaseID, "resume point remains the earliest unfinished phase")

	updated, pctx, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, "ph-2", updated.CurrentPhaseID, "advanced pointer is persisted")
	require.Equal(t, "ph-2", pctx.CurrentPhase.ID)
}

func TestGetNextKeepsLastUnfinishedPhaseIncomplete(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusDone
	plan.Phases[1].Status = internalplans.PhaseStatusDone
	h := newHarness(t, plan, nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, "ph-3", e.CurrentPhaseID)

	pctx, complete, err := h.svc.GetNext(context.Background(), e.ID)
	require.NoError(t, err)
	require.False(t, complete, "last unfinished phase is still actionable")
	require.Equal(t, "ph-3", pctx.CurrentPhase.ID)
	require.Equal(t, "ph-3", pctx.ResumePhaseID)
}

func TestRecordDecisionPersistsForHandoff(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	d, err := h.svc.RecordDecision(context.Background(), e.ID, "ph-1", "chose SQLite", "home store")
	require.NoError(t, err)
	require.NotEmpty(t, d.ID)
	require.Equal(t, "chose SQLite", d.Summary)
}

// [REQ:PM-HANDOFF-002]
func TestRecordFindingAlwaysCandidate(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "run-x")
	require.NoError(t, err)
	f, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-1", "maybe a bug", "detail")
	require.NoError(t, err)
	require.Equal(t, execution.TriageCandidate, f.Triage, "findings are filed CANDIDATE, never auto-promoted")
	require.Equal(t, "run-x", f.AttributionRunID)
}

// [REQ:PM-HANDOFF-002]
func TestRecordFindingDedupByRunIDAndTitle(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "run-dup")
	require.NoError(t, err)

	first, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-1", "Same Title", "first detail")
	require.NoError(t, err)
	second, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-2", "  same title ", "second detail")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "same run_id + title is not double-filed (case/space-insensitive)")

	candidates, err := h.svc.ListCandidateFindings(context.Background(), e.ID)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
}

func TestRecordFindingDedupByTitleWhenNoRunID(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	// Ensure no env run id leaks in; Start with empty run id and no env.
	t.Setenv("VROOLI_AGENT_MANAGER_RUN_ID", "")
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Empty(t, e.RunID)

	first, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-1", "Bug A", "d1")
	require.NoError(t, err)
	second, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-2", "Bug A", "d2")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "no run id => dedup by title within the execution")
}

// [REQ:PM-EXEC-002]
func TestCompletenessFullVsPartial(t *testing.T) {
	require := require.New(t)

	partial := newHarness(t, threePhasePlan(), nil)
	e, err := partial.svc.Start(context.Background(), "plan-1", "")
	require.NoError(err)
	_, pctx, err := partial.svc.GetStatus(context.Background(), e.ID)
	require.NoError(err)
	require.Equal(execution.CompletenessPartial, pctx.Completeness)

	allDone := threePhasePlan()
	for i := range allDone.Phases {
		allDone.Phases[i].Status = internalplans.PhaseStatusDone
	}
	full := newHarness(t, allDone, nil)
	e2, err := full.svc.Start(context.Background(), "plan-1", "")
	require.NoError(err)
	_, pctx2, err := full.svc.GetStatus(context.Background(), e2.ID)
	require.NoError(err)
	require.Equal(execution.CompletenessFull, pctx2.Completeness)
}

// [REQ:PM-HANDOFF-001] [REQ:PM-EXEC-002] [REQ:PM-VEL-001]
func TestCompleteAssemblesHandoffAndCapturesVelocity(t *testing.T) {
	validator := fakeValidator{
		hasResult: true,
		result:    execution.ValidationResult{Verdict: "pass", Detail: "ok", Staleness: internalplans.StalenessFresh},
	}
	plan := threePhasePlan()
	for i := range plan.Phases {
		plan.Phases[i].Status = internalplans.PhaseStatusDone
	}
	h := newHarness(t, plan, validator)
	e, err := h.svc.Start(context.Background(), "plan-1", "run-c")
	require.NoError(t, err)
	_, err = h.svc.RecordDecision(context.Background(), e.ID, "ph-1", "a decision", "")
	require.NoError(t, err)
	_, err = h.svc.RecordFinding(context.Background(), e.ID, "ph-1", "a finding", "")
	require.NoError(t, err)

	// Advance the clock so wall-time is non-zero.
	h.clock.Advance(90 * time.Second)

	handoff, nudges, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{Tokens: 1200, Iterations: 4})
	require.NoError(t, err)
	require.Equal(t, execution.CompletenessFull, handoff.Completeness)
	require.Equal(t, "", handoff.ResumePhaseID, "all done => no resume point")
	require.Len(t, handoff.Decisions, 1)
	require.Len(t, handoff.CandidateFindings, 1)
	require.True(t, handoff.HasValidation)
	require.Equal(t, "", handoff.ProseHandoffRef, "prose handoff ref is a pass-through, default empty")

	// Nudges: all should be satisfied (a candidate finding exists; all phases done).
	require.Len(t, nudges, 3)
	for _, n := range nudges {
		require.True(t, n.Satisfied, "nudge %q should be satisfied", n.Kind)
	}

	// Velocity captured LOCAL ONLY and offered to the (stub) sink.
	require.Len(t, h.sink.emitted, 1)
	points, err := h.svc.GetVelocity(context.Background(), "plan-1")
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Equal(t, int64(90), points[0].WallTimeSeconds)
	require.Equal(t, int64(1200), points[0].Tokens)
	require.Equal(t, int32(4), points[0].Iterations)
	require.Equal(t, execution.CompletenessFull, points[0].Completeness)
}

func TestCompleteNudgesUnsatisfiedWhenStatePartial(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil) // no findings, phases still todo
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, nudges, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
	require.NoError(t, err)
	byKind := map[string]bool{}
	for _, n := range nudges {
		byKind[n.Kind] = n.Satisfied
	}
	require.False(t, byKind["record_finding"], "no findings => unsatisfied")
	require.False(t, byKind["confirm_phase_status"], "phases still todo => unsatisfied")
}

func TestGetHandoffReturnsPersistedAfterComplete(t *testing.T) {
	plan := threePhasePlan()
	for i := range plan.Phases {
		plan.Phases[i].Status = internalplans.PhaseStatusDone
	}
	h := newHarness(t, plan, nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	written, _, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
	require.NoError(t, err)

	got, err := h.svc.GetHandoff(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, written.ID, got.ID, "GetHandoff returns the persisted record after Complete")
	require.Equal(t, execution.CompletenessFull, got.Completeness)
}

func TestGetHandoffLiveViewBeforeComplete(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, err = h.svc.RecordDecision(context.Background(), e.ID, "ph-1", "early decision", "")
	require.NoError(t, err)

	got, err := h.svc.GetHandoff(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, e.ID, got.ExecutionID)
	require.Len(t, got.Decisions, 1, "live handoff view assembles from captured state before Complete")
	require.Equal(t, execution.CompletenessPartial, got.Completeness)
}

func TestTriageFindingPromoteAndDismiss(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, err := h.svc.Start(context.Background(), "plan-1", "run-t")
	require.NoError(t, err)
	promote, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-1", "promote me", "")
	require.NoError(t, err)
	dismiss, err := h.svc.RecordFinding(context.Background(), e.ID, "ph-1", "dismiss me", "")
	require.NoError(t, err)

	pr, err := h.svc.TriageFinding(context.Background(), promote.ID, execution.TriagePromoted)
	require.NoError(t, err)
	require.Equal(t, execution.TriagePromoted, pr.Triage)
	ds, err := h.svc.TriageFinding(context.Background(), dismiss.ID, execution.TriageDismissed)
	require.NoError(t, err)
	require.Equal(t, execution.TriageDismissed, ds.Triage)

	// Only candidate findings remain in the candidate list (both triaged away).
	candidates, err := h.svc.ListCandidateFindings(context.Background(), e.ID)
	require.NoError(t, err)
	require.Empty(t, candidates, "promoted/dismissed findings leave the candidate list")
}

func TestTriageFindingNotFound(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	_, err := h.svc.TriageFinding(context.Background(), "missing", execution.TriagePromoted)
	require.ErrorAs(t, err, &execution.ErrFindingNotFound{})
}

func TestListCandidateFindingsAcrossExecutions(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e1, err := h.svc.Start(context.Background(), "plan-1", "run-1")
	require.NoError(t, err)
	e2, err := h.svc.Start(context.Background(), "plan-1", "run-2")
	require.NoError(t, err)
	_, err = h.svc.RecordFinding(context.Background(), e1.ID, "ph-1", "f1", "")
	require.NoError(t, err)
	_, err = h.svc.RecordFinding(context.Background(), e2.ID, "ph-1", "f2", "")
	require.NoError(t, err)

	scoped, err := h.svc.ListCandidateFindings(context.Background(), e1.ID)
	require.NoError(t, err)
	require.Len(t, scoped, 1, "scoped to one execution")

	all, err := h.svc.ListCandidateFindings(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, all, 2, "empty scope lists across executions")
}

func TestExecutionNotFound(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	_, _, err := h.svc.GetStatus(context.Background(), "nope")
	require.ErrorAs(t, err, &execution.ErrExecutionNotFound{})
}
