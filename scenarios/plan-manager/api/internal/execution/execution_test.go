package execution_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"plan-manager/internal/execution"
	planmodel "plan-manager/internal/planmodel"
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

func (f *fakePlanStore) UpdatePhase(_ context.Context, _, _, _ string, phase internalplans.Phase) (internalplans.Plan, error) {
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
type recordingSink struct {
	emitted []execution.VelocityPoint
	err     error
}

func (s *recordingSink) Emit(_ context.Context, p execution.VelocityPoint) error {
	s.emitted = append(s.emitted, p)
	return s.err
}

// fakeLog is the LogLedger seam: a canned summary + entries so handoff/context
// log roll-ups are exercised without the real log domain. Decisions/findings are
// OWNED by the log domain; execution only reads compact summaries here.
type fakeLog struct {
	summary      planmodel.LogSummary
	entries      []planmodel.LogEntry
	phaseSummary planmodel.LogSummary
	phaseEntries []planmodel.LogEntry
	err          error
}

func (f *fakeLog) Summarize(context.Context, string) (planmodel.LogSummary, []planmodel.LogEntry, error) {
	return f.summary, f.entries, f.err
}

func (f *fakeLog) SummarizePhase(context.Context, string, string) (planmodel.LogSummary, []planmodel.LogEntry, error) {
	return f.phaseSummary, f.phaseEntries, f.err
}

// fakeFreshener is the InputFreshener seam — the execution-start baseline
// capture + staleness recompute. The test dials its result/error and counts how
// many times it is invoked to prove the once-per-start guard and the cheap-poll
// invariant (status/next never freshen).
type fakeFreshener struct {
	result execution.FreshenResult
	err    error
	calls  int
}

func (f *fakeFreshener) FreshenInputs(_ context.Context, _ string) (execution.FreshenResult, error) {
	f.calls++
	return f.result, f.err
}

// --- harness ---

type harness struct {
	svc   execution.Service
	store *fakePlanStore
	sink  *recordingSink
	log   *fakeLog
	clock *mocks.FakeClock
}

func newHarness(t *testing.T, plan internalplans.Plan, validator execution.Validator) harness {
	return newHarnessWithLog(t, plan, validator, &fakeLog{})
}

func newHarnessWithLog(t *testing.T, plan internalplans.Plan, validator execution.Validator, lg *fakeLog) harness {
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
		Log:       lg,
		Velocity:  sink,
		Clock:     clk,
	})
	return harness{svc: svc, store: store, sink: sink, log: lg, clock: clk}
}

func newHarnessWithFreshener(t *testing.T, plan internalplans.Plan, freshener execution.InputFreshener) harness {
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
		Velocity:  sink,
		Freshener: freshener,
		Clock:     clk,
	})
	return harness{svc: svc, store: store, sink: sink, log: &fakeLog{}, clock: clk}
}

func threePhasePlan() internalplans.Plan {
	return internalplans.Plan{
		ID:                 "plan-1",
		Slug:               "plan-1",
		Title:              "Plan One",
		Purpose:            "Exercise the execution runner.",
		ProblemStatement:   "Execution tests need a valid plan fixture.",
		TargetOutcome:      "Runner behavior is tested against execution-grade plans.",
		Scope:              "Execution service unit tests.",
		TechnicalApproach:  "Use an in-memory PlanStore seam.",
		ValidationStrategy: "Run execution unit tests.",
		DefinitionOfDone:   "Execution service behavior is deterministic.",
		Constraints:        "NO_CODE_REFS: execution unit fixture has no plan-level connected refs.",
		ChangeBoundary: internalplans.ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/plan-manager/**"},
		},
		RegressionAnchor: internalplans.RegressionAnchor{
			Strategy: internalplans.AnchorStrategyChangeBoundary,
		},
		RelevantContext: []internalplans.RelevantContextItem{{
			ID:           "ctx-global",
			Kind:         internalplans.RelevantContextNote,
			Scope:        internalplans.RelevantContextScopeGlobal,
			Label:        "NO_CONTEXT: execution unit fixture has no plan-wide setup.",
			Instruction:  "NO_CONTEXT: execution unit fixture has no plan-wide setup.",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextOncePerExecution,
			Source:       internalplans.RelevantContextSourceAuthored,
			Status:       internalplans.RelevantContextStatusReady,
		}},
		Phases: []internalplans.Phase{
			validExecutionPhase("ph-1", 1, "First", []string{"docs/a.md"}, []string{"think first"}),
			validExecutionPhase("ph-2", 2, "Second", nil, nil),
			validExecutionPhase("ph-3", 3, "Third", nil, nil),
		},
	}
}

func validExecutionPhase(id string, order int, title string, requiredReading, reminders []string) internalplans.Phase {
	reminders = append([]string(nil), reminders...)
	reminders = append(reminders, "NO_CODE_REFS: execution unit fixture has no phase refs.")
	return internalplans.Phase{
		ID:              id,
		Order:           order,
		Title:           title,
		Intent:          "Exercise " + title,
		Steps:           []string{"Run the service method under test."},
		Validation:      "go test ./internal/execution",
		Acceptance:      "The service returns the expected state.",
		Status:          internalplans.PhaseStatusTodo,
		RequiredReading: append([]string(nil), requiredReading...),
		Reminders:       reminders,
		RelevantContext: []internalplans.RelevantContextItem{{
			ID:           "ctx-" + id,
			Kind:         internalplans.RelevantContextNote,
			Scope:        internalplans.RelevantContextScopePhase,
			PhaseID:      id,
			Label:        "NO_CONTEXT: execution unit fixture has no phase setup.",
			Instruction:  "NO_CONTEXT: execution unit fixture has no phase setup.",
			Required:     true,
			RepeatPolicy: internalplans.RelevantContextPhaseEntry,
			Source:       internalplans.RelevantContextSourceAuthored,
			Status:       internalplans.RelevantContextStatusReady,
		}},
	}
}

func contextIDs(items []internalplans.RelevantContextItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func doneOverride() execution.PhaseTransitionInputs {
	return execution.PhaseTransitionInputs{
		ToStatus:                 internalplans.PhaseStatusDone,
		ValidationOverrideReason: "test fixture bypasses validation to exercise runner pointer behavior",
		FeedbackOverrideReason:   "test fixture bypasses feedback checkpoint to exercise runner pointer behavior",
	}
}

// --- tests ---

func TestStartLinksRunAndSetsResumePointer(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, pctx, step, err := h.svc.Start(context.Background(), "plan-1", "run-abc")
	require.NoError(t, err)
	require.NotEmpty(t, e.ID)
	require.Equal(t, "plan-1", e.PlanID)
	require.Equal(t, "run-abc", e.RunID)
	require.Equal(t, "ph-1", e.CurrentPhaseID, "current pointer starts at the earliest non-done phase")
	require.False(t, e.Complete)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "ph-1", pctx.CurrentPhase.ID)
	require.Equal(t, []string{"ctx-global", "ctx-ph-1"}, contextIDs(pctx.RelevantContext))
	require.Equal(t, "execution_started", step.StepKind)
	require.Equal(t, []string{"exec", "status", e.ID}, step.NextActions[0].Argv)
}

func TestStartRequiresPlanID(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	_, _, _, err := h.svc.Start(context.Background(), "  ", "")
	require.ErrorAs(t, err, &execution.ErrInvalidExecution{})
}

func TestStartRejectsPlanThatNeedsRepair(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases = nil
	plan.ImportProvenance = &internalplans.ImportProvenance{SourcePath: "docs/plans/legacy.md"}
	h := newHarness(t, plan, nil)

	_, _, _, err := h.svc.Start(context.Background(), "plan-1", "")

	require.ErrorAs(t, err, &execution.ErrInvalidExecution{})
	require.Contains(t, err.Error(), "plan is not execution-grade")
	require.Contains(t, err.Error(), "plan_missing_phases")
}

func TestResumePointDerivationEarliestNonDone(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusDone // ph-1 done
	h := newHarness(t, plan, nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, "ph-2", e.CurrentPhaseID, "resume point skips the done phase to the earliest non-done one")
}

// [REQ:PM-EXEC-001]
func TestGetStatusInjectsPhaseScopedContext(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	_, pctx, step, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "ph-1", pctx.CurrentPhase.ID)
	require.Equal(t, []string{"docs/a.md"}, pctx.RequiredReading)
	require.Contains(t, pctx.Reminders, "think first")
	require.True(t, pctx.HasNext)
	require.Equal(t, "ph-2", pctx.NextPhase.ID)
	require.Equal(t, "ph-1", pctx.ResumePhaseID)
	require.Equal(t, execution.CompletenessPartial, pctx.Completeness)
	require.Equal(t, "phase_context", step.StepKind)
	require.Equal(t, []string{"exec", "transition", e.ID, "ph-1", "--status", "active"}, step.NextActions[0].Argv)
}

func TestContinueExecutionRecommendsValidationForActiveUnvalidatedPhase(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusActive
	h := newHarness(t, plan, nil)

	e, pctx, step, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.Equal(t, "ph-1", e.CurrentPhaseID)
	require.Equal(t, internalplans.PhaseStatusActive, pctx.CurrentPhase.Status)
	require.Len(t, step.NextActions, 1, "continue returns exactly one recommended action")
	require.Equal(t, "run-validation", step.NextActions[0].ID)
	require.Equal(t, []string{"validate", "run", "plan-1", "--phase", "ph-1"}, step.NextActions[0].Argv)
	require.Contains(t, step.NextActions[0].BlockedBy[0], "no stored validation result")
}

func TestContinueExecutionRecommendsFeedbackCheckpointAfterFreshPassingValidation(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusActive
	validator := fakeValidator{
		hasResult: true,
		result: execution.ValidationResult{
			Verdict:   "pass",
			Staleness: internalplans.StalenessFresh,
		},
	}
	h := newHarness(t, plan, validator)

	e, pctx, step, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.Len(t, step.NextActions, 1, "continue returns exactly one recommended action")
	require.Equal(t, "review-phase-feedback", step.NextActions[0].ID)
	require.Equal(t, []string{"log", "note-add", e.ID, "--phase", "ph-1", "--title", execution.NoFeedbackCheckpointTitle, "--detail", "No decisions, findings, bugs, records, or reusable notes to capture for this phase."}, step.NextActions[0].Argv)
	require.False(t, pctx.FeedbackCheckpoint.Satisfied)
}

func TestContinueExecutionRecommendsDoneAfterFeedbackCheckpointSatisfied(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusActive
	validator := fakeValidator{
		hasResult: true,
		result: execution.ValidationResult{
			Verdict:   "pass",
			Staleness: internalplans.StalenessFresh,
		},
	}
	lg := &fakeLog{
		phaseSummary: planmodel.LogSummary{Total: 1, Notes: 1},
		phaseEntries: []planmodel.LogEntry{{
			ID:      "note-1",
			Type:    planmodel.LogEntryNote,
			PhaseID: "ph-1",
			Title:   execution.NoFeedbackCheckpointTitle,
		}},
	}
	h := newHarnessWithLog(t, plan, validator, lg)

	e, pctx, step, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.True(t, pctx.FeedbackCheckpoint.Satisfied)
	require.True(t, pctx.FeedbackCheckpoint.Reviewed)
	require.Len(t, step.NextActions, 1, "continue returns exactly one recommended action")
	require.Equal(t, "transition-done", step.NextActions[0].ID)
	require.Equal(t, []string{"exec", "transition", e.ID, "ph-1", "--status", "done"}, step.NextActions[0].Argv)
}

func TestContinueExecutionRecommendsDoneAfterCapturedPhaseFeedback(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusActive
	validator := fakeValidator{
		hasResult: true,
		result: execution.ValidationResult{
			Verdict:   "pass",
			Staleness: internalplans.StalenessFresh,
		},
	}
	lg := &fakeLog{
		phaseSummary: planmodel.LogSummary{Total: 1, Decisions: 1},
		phaseEntries: []planmodel.LogEntry{{
			ID:      "decision-1",
			Type:    planmodel.LogEntryDecision,
			PhaseID: "ph-1",
			Title:   "Use phase-scoped feedback checkpoint",
		}},
	}
	h := newHarnessWithLog(t, plan, validator, lg)

	e, pctx, step, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.True(t, pctx.FeedbackCheckpoint.Satisfied)
	require.Equal(t, 1, pctx.FeedbackCheckpoint.Decisions)
	require.Len(t, step.NextActions, 1)
	require.Equal(t, "transition-done", step.NextActions[0].ID)
	require.Equal(t, []string{"exec", "transition", e.ID, "ph-1", "--status", "done"}, step.NextActions[0].Argv)
}

func TestGetContextPhaseOverrideDoesNotAdvancePointer(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	got, pctx, _, err := h.svc.GetContext(context.Background(), e.ID, "ph-2")
	require.NoError(t, err)
	require.Equal(t, e.ID, got.ID)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "ph-2", pctx.CurrentPhase.ID)

	updated, pctx, _, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, "ph-1", updated.CurrentPhaseID, "context phase override is read-only")
	require.Equal(t, "ph-1", pctx.CurrentPhase.ID)
}

func TestResumeByPlanReusesLatestExecution(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	first, _, _, err := h.svc.Start(context.Background(), "plan-1", "run-1")
	require.NoError(t, err)
	_, _, _, err = h.svc.TransitionPhase(context.Background(), first.ID, "ph-1", doneOverride())
	require.NoError(t, err)

	resumed, pctx, _, err := h.svc.Resume(context.Background(), "plan-1", "", "ignored-new-run")
	require.NoError(t, err)
	require.Equal(t, first.ID, resumed.ID, "resume by plan should reuse the latest execution")
	require.Equal(t, "ph-2", resumed.CurrentPhaseID)
	require.Equal(t, "ph-2", pctx.CurrentPhase.ID)
	require.Equal(t, "run-1", resumed.RunID, "run id is only used when creating a new execution")
}

func TestResumeExplicitPhasePersistsPointer(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	resumed, pctx, _, err := h.svc.Resume(context.Background(), e.ID, "ph-3", "")
	require.NoError(t, err)
	require.Equal(t, "ph-3", resumed.CurrentPhaseID)
	require.Equal(t, "ph-3", pctx.CurrentPhase.ID)

	updated, pctx, _, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, "ph-3", updated.CurrentPhaseID)
	require.Equal(t, "ph-3", pctx.CurrentPhase.ID)
}

func TestRelevantContextRepeatPolicies(t *testing.T) {
	plan := threePhasePlan()
	plan.RelevantContext = []internalplans.RelevantContextItem{
		{ID: "once", Kind: internalplans.RelevantContextSkill, Label: "once", RepeatPolicy: internalplans.RelevantContextOncePerExecution},
		{ID: "resume", Kind: internalplans.RelevantContextCommand, Label: "resume", RepeatPolicy: internalplans.RelevantContextOnResume},
		{ID: "every", Kind: internalplans.RelevantContextCommand, Label: "every", RepeatPolicy: internalplans.RelevantContextEveryPhase},
	}
	plan.Phases[0].RelevantContext = []internalplans.RelevantContextItem{
		{ID: "phase", Kind: internalplans.RelevantContextDoc, Label: "phase", RepeatPolicy: internalplans.RelevantContextPhaseEntry},
	}
	h := newHarness(t, plan, nil)

	e, pctx, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, []string{"once", "every", "phase"}, contextIDs(pctx.RelevantContext))

	_, pctx, _, err = h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, []string{"resume", "every", "phase"}, contextIDs(pctx.RelevantContext))

	_, pctx, _, err = h.svc.Resume(context.Background(), e.ID, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"resume", "every", "phase"}, contextIDs(pctx.RelevantContext))
}

// [REQ:PM-EXEC-001] First-start via continue/resume must emit once-per-execution
// context exactly once — the bug was that creating a new run through continue used
// the resume context mode and silently skipped once_per_execution items.
func TestContinueFirstStartEmitsOncePerExecutionContext(t *testing.T) {
	plan := threePhasePlan()
	plan.RelevantContext = []internalplans.RelevantContextItem{
		{ID: "once", Kind: internalplans.RelevantContextSkill, Label: "once", RepeatPolicy: internalplans.RelevantContextOncePerExecution},
		{ID: "resume", Kind: internalplans.RelevantContextCommand, Label: "resume", RepeatPolicy: internalplans.RelevantContextOnResume},
	}
	h := newHarness(t, plan, nil)

	// First continue with no prior execution => a brand-new run => first-start.
	_, pctx, _, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.Contains(t, contextIDs(pctx.RelevantContext), "once", "once-per-execution context must emit on first start via continue")
	require.NotContains(t, contextIDs(pctx.RelevantContext), "resume", "on-resume context must not emit on a first start")

	// Second continue resumes the existing run => once-per-execution is NOT re-emitted.
	_, pctx2, _, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.NotContains(t, contextIDs(pctx2.RelevantContext), "once", "once-per-execution context must not re-emit on resume")
	require.Contains(t, contextIDs(pctx2.RelevantContext), "resume", "on-resume context emits when resuming an existing run")
}

func TestGetStatusDegradesToUnknownWhenValidatorNil(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil) // nil validator
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, pctx, _, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.False(t, pctx.HasValidation, "nil validator => no validation, never a false pass")
	require.Equal(t, internalplans.StalenessUnknown, pctx.Staleness)
}

func TestContinueExecutionDegradesValidatorErrorToUnknownGuidance(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusActive
	h := newHarness(t, plan, fakeValidator{err: errors.New("validation store unavailable")})

	_, pctx, step, err := h.svc.ContinueExecution(context.Background(), "plan-1", "", "run-1")
	require.NoError(t, err)
	require.False(t, pctx.HasValidation, "validator errors degrade to absent validation, never a false pass")
	require.Equal(t, internalplans.StalenessUnknown, pctx.Staleness)
	require.Len(t, step.NextActions, 1, "continue still returns one recovery action")
	require.Equal(t, "run-validation", step.NextActions[0].ID)
	require.Contains(t, step.NextActions[0].BlockedBy[0], "no stored validation result")
}

func TestTransitionPhaseRequiresValidationBeforeDone(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", execution.PhaseTransitionInputs{ToStatus: internalplans.PhaseStatusDone})
	require.ErrorAs(t, err, &execution.ErrValidationRequired{})
	require.Equal(t, internalplans.PhaseStatusTodo, h.store.plan.Phases[0].Status, "failed validation gate must not mutate the plan")
}

func TestTransitionPhaseDoneRequiresFeedbackCheckpointAfterFreshPassingValidation(t *testing.T) {
	validator := fakeValidator{
		hasResult: true,
		result: execution.ValidationResult{
			Verdict:   "pass",
			Staleness: internalplans.StalenessFresh,
		},
	}
	h := newHarness(t, threePhasePlan(), validator)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", execution.PhaseTransitionInputs{ToStatus: internalplans.PhaseStatusDone})
	require.Error(t, err)
	require.Contains(t, err.Error(), "phase feedback checkpoint is required")
	require.Equal(t, internalplans.PhaseStatusTodo, h.store.plan.Phases[0].Status)
}

func TestTransitionPhaseDoneAllowsFreshPassingValidationAndFeedbackCheckpoint(t *testing.T) {
	validator := fakeValidator{
		hasResult: true,
		result: execution.ValidationResult{
			Verdict:   "pass",
			Staleness: internalplans.StalenessFresh,
		},
	}
	lg := &fakeLog{
		phaseSummary: planmodel.LogSummary{Total: 1, Notes: 1},
		phaseEntries: []planmodel.LogEntry{{
			ID:      "note-1",
			Type:    planmodel.LogEntryNote,
			PhaseID: "ph-1",
			Title:   execution.NoFeedbackCheckpointTitle,
		}},
	}
	h := newHarnessWithLog(t, threePhasePlan(), validator, lg)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	updatedE, plan, _, err := h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", execution.PhaseTransitionInputs{ToStatus: internalplans.PhaseStatusDone})
	require.NoError(t, err)
	require.Equal(t, internalplans.PhaseStatusDone, plan.Phases[0].Status, "the plans domain owns the persisted status")
	require.Equal(t, "ph-2", updatedE.CurrentPhaseID, "pointer advances to the next non-done phase")
	require.False(t, updatedE.Complete)
}

func TestTransitionPhaseDoneAllowsExplicitValidationAndFeedbackOverrides(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	updatedE, plan, _, err := h.svc.TransitionPhase(context.Background(), e.ID, "ph-1", doneOverride())
	require.NoError(t, err)
	require.Equal(t, internalplans.PhaseStatusDone, plan.Phases[0].Status)
	require.Equal(t, "ph-2", updatedE.CurrentPhaseID)
}

func TestTransitionPhaseUnknownPhase(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, "nope", doneOverride())
	require.ErrorAs(t, err, &internalplans.ErrPhaseNotFound{})
}

func TestGetNextAdvancesAndReportsComplete(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	// Drive every phase to done, then GetNext reports complete.
	for _, id := range []string{"ph-1", "ph-2", "ph-3"} {
		_, _, _, err = h.svc.TransitionPhase(context.Background(), e.ID, id, doneOverride())
		require.NoError(t, err)
	}
	pctx, complete, _, err := h.svc.GetNext(context.Background(), e.ID)
	require.NoError(t, err)
	require.True(t, complete, "no actionable phase remains")
	require.Equal(t, "", pctx.ResumePhaseID)
	require.Equal(t, execution.CompletenessFull, pctx.Completeness)
}

// [REQ:PM-EXEC-001]
func TestGetNextAdvancesPastCurrentUnfinishedPhase(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, "ph-1", e.CurrentPhaseID)

	pctx, complete, _, err := h.svc.GetNext(context.Background(), e.ID)
	require.NoError(t, err)
	require.False(t, complete)
	require.True(t, pctx.HasCurrent)
	require.Equal(t, "ph-2", pctx.CurrentPhase.ID, "GetNext advances to a later actionable phase instead of recomputing the resume point")
	require.Equal(t, "ph-1", pctx.ResumePhaseID, "resume point remains the earliest unfinished phase")

	updated, pctx, _, err := h.svc.GetStatus(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, "ph-2", updated.CurrentPhaseID, "advanced pointer is persisted")
	require.Equal(t, "ph-2", pctx.CurrentPhase.ID)
}

func TestGetNextKeepsLastUnfinishedPhaseIncomplete(t *testing.T) {
	plan := threePhasePlan()
	plan.Phases[0].Status = internalplans.PhaseStatusDone
	plan.Phases[1].Status = internalplans.PhaseStatusDone
	h := newHarness(t, plan, nil)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	require.Equal(t, "ph-3", e.CurrentPhaseID)

	pctx, complete, _, err := h.svc.GetNext(context.Background(), e.ID)
	require.NoError(t, err)
	require.False(t, complete, "last unfinished phase is still actionable")
	require.Equal(t, "ph-3", pctx.CurrentPhase.ID)
	require.Equal(t, "ph-3", pctx.ResumePhaseID)
}

// [REQ:PM-EXEC-002]
func TestCompletenessFullVsPartial(t *testing.T) {
	require := require.New(t)

	partial := newHarness(t, threePhasePlan(), nil)
	e, _, _, err := partial.svc.Start(context.Background(), "plan-1", "")
	require.NoError(err)
	_, pctx, _, err := partial.svc.GetStatus(context.Background(), e.ID)
	require.NoError(err)
	require.Equal(execution.CompletenessPartial, pctx.Completeness)

	allDone := threePhasePlan()
	for i := range allDone.Phases {
		allDone.Phases[i].Status = internalplans.PhaseStatusDone
	}
	full := newHarness(t, allDone, nil)
	e2, _, _, err := full.svc.Start(context.Background(), "plan-1", "")
	require.NoError(err)
	_, pctx2, _, err := full.svc.GetStatus(context.Background(), e2.ID)
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
	lg := &fakeLog{
		summary: planmodel.LogSummary{Total: 4, Decisions: 1, Findings: 1, BugReports: 1, Records: 1},
		entries: []planmodel.LogEntry{
			{ID: "le-1", Type: planmodel.LogEntryDecision, Title: "a decision"},
			{ID: "le-2", Type: planmodel.LogEntryFinding, Title: "a finding"},
			{ID: "le-3", Type: planmodel.LogEntryBugReport, Title: "a bug"},
			{ID: "le-4", Type: planmodel.LogEntryRecord, Title: "a record"},
		},
	}
	h := newHarnessWithLog(t, plan, validator, lg)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "run-c")
	require.NoError(t, err)

	// Advance the clock so wall-time is non-zero.
	h.clock.Advance(90 * time.Second)

	handoff, nudges, _, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{Tokens: 1200, Iterations: 4})
	require.NoError(t, err)
	require.Equal(t, execution.CompletenessFull, handoff.Completeness)
	require.Equal(t, "", handoff.ResumePhaseID, "all done => no resume point")
	require.Equal(t, 1, handoff.LogSummary.Decisions)
	require.Equal(t, 1, handoff.LogSummary.Findings)
	require.Equal(t, 1, handoff.LogSummary.BugReports)
	require.Len(t, handoff.LogEntries, 4, "the handoff snapshots the captured log entries")
	require.True(t, handoff.HasValidation)
	require.Equal(t, "", handoff.ProseHandoffRef, "prose handoff ref is a pass-through, default empty")

	// Nudges: all should be satisfied (decisions+findings+bug recorded; all phases done).
	require.Len(t, nudges, 4)
	for _, n := range nudges {
		require.True(t, n.Satisfied, "nudge %q should be satisfied", n.Kind)
	}

	// Velocity captured LOCAL ONLY and offered to the (stub) sink.
	require.Len(t, h.sink.emitted, 1)
	points, _, err := h.svc.GetVelocity(context.Background(), "plan-1")
	require.NoError(t, err)
	require.Len(t, points, 1)
	require.Equal(t, int64(90), points[0].WallTimeSeconds)
	require.Equal(t, int64(1200), points[0].Tokens)
	require.Equal(t, int32(4), points[0].Iterations)
	require.Equal(t, execution.CompletenessFull, points[0].Completeness)
}

func TestCompleteKeepsLocalVelocityWhenSinkFails(t *testing.T) {
	plan := threePhasePlan()
	for i := range plan.Phases {
		plan.Phases[i].Status = internalplans.PhaseStatusDone
	}
	h := newHarness(t, plan, nil)
	h.sink.err = errors.New("meta-optimization sink unavailable")
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "run-c")
	require.NoError(t, err)
	h.clock.Advance(30 * time.Second)

	handoff, _, _, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{Tokens: 10, Iterations: 1})
	require.NoError(t, err, "remote velocity sink failure must not fail local completion")
	require.Equal(t, execution.CompletenessFull, handoff.Completeness)
	require.Len(t, h.sink.emitted, 1, "the point is still offered to the seam")

	points, _, err := h.svc.GetVelocity(context.Background(), "plan-1")
	require.NoError(t, err)
	require.Len(t, points, 1, "velocity is persisted locally before best-effort sink emit")
	require.Equal(t, int64(30), points[0].WallTimeSeconds)
}

func TestCompleteNudgesUnsatisfiedWhenStatePartial(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil) // no findings, phases still todo
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	_, nudges, _, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
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
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)
	written, _, _, err := h.svc.Complete(context.Background(), e.ID, execution.CompletionInputs{})
	require.NoError(t, err)

	got, _, err := h.svc.GetHandoff(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, written.ID, got.ID, "GetHandoff returns the persisted record after Complete")
	require.Equal(t, execution.CompletenessFull, got.Completeness)
}

func TestGetHandoffLiveViewBeforeComplete(t *testing.T) {
	lg := &fakeLog{
		summary: planmodel.LogSummary{Total: 1, Decisions: 1},
		entries: []planmodel.LogEntry{{ID: "le-1", Type: planmodel.LogEntryDecision, Title: "early decision"}},
	}
	h := newHarnessWithLog(t, threePhasePlan(), nil, lg)
	e, _, _, err := h.svc.Start(context.Background(), "plan-1", "")
	require.NoError(t, err)

	got, _, err := h.svc.GetHandoff(context.Background(), e.ID)
	require.NoError(t, err)
	require.Equal(t, e.ID, got.ExecutionID)
	require.Equal(t, 1, got.LogSummary.Decisions, "live handoff view reads the log ledger before Complete")
	require.Len(t, got.LogEntries, 1)
	require.Equal(t, execution.CompletenessPartial, got.Completeness)
}

func TestExecutionNotFound(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	_, _, _, err := h.svc.GetStatus(context.Background(), "nope")
	require.ErrorAs(t, err, &execution.ErrExecutionNotFound{})
}

// --- execution-start freshen inputs (Phase 3) ---

// TestFreshenRunsOnceOnStartNotOnPoll proves the freshen step runs exactly once
// at start and NEVER on the per-poll status/next path (the cheap-context
// invariant).
func TestFreshenRunsOnceOnStartNotOnPoll(t *testing.T) {
	fr := &fakeFreshener{result: execution.FreshenResult{BaselineCaptured: true, BaselineName: "plan-1-baseline", Detail: "captured baseline plan-1-baseline"}}
	h := newHarnessWithFreshener(t, threePhasePlan(), fr)
	ctx := context.Background()

	e, pctx, _, err := h.svc.Start(ctx, "plan-1", "run-1")
	require.NoError(t, err)
	require.Equal(t, 1, fr.calls, "freshen runs once on start")
	require.True(t, pctx.InputsFreshened)
	require.Equal(t, execution.FreshenStatusCaptured, pctx.FreshenStatus)
	require.Contains(t, pctx.FreshenDetail, "plan-1-baseline")

	// The per-poll context server must NOT freshen again.
	_, _, _, err = h.svc.GetStatus(ctx, e.ID)
	require.NoError(t, err)
	_, _, _, err = h.svc.GetNext(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, 1, fr.calls, "status/next must never trigger a freshen")
}

// TestFreshenCapturedNotReRunOnResume proves a successful capture is terminal:
// resuming the same execution does not re-capture the pinned baseline.
func TestFreshenCapturedNotReRunOnResume(t *testing.T) {
	fr := &fakeFreshener{result: execution.FreshenResult{BaselineCaptured: true, BaselineName: "plan-1-baseline"}}
	h := newHarnessWithFreshener(t, threePhasePlan(), fr)
	ctx := context.Background()

	e, _, _, err := h.svc.Start(ctx, "plan-1", "run-1")
	require.NoError(t, err)
	require.Equal(t, 1, fr.calls)

	_, _, _, err = h.svc.Resume(ctx, e.ID, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, fr.calls, "a captured baseline is pinned; resume must not re-capture")
}

// TestFreshenDegradationIsNonBlocking proves a freshener error is recorded as a
// degraded status, surfaced in the phase context, and never blocks the start.
func TestFreshenDegradationIsNonBlocking(t *testing.T) {
	fr := &fakeFreshener{err: errors.New("git-control-tower unavailable")}
	h := newHarnessWithFreshener(t, threePhasePlan(), fr)
	ctx := context.Background()

	e, pctx, _, err := h.svc.Start(ctx, "plan-1", "run-1")
	require.NoError(t, err, "a degraded freshen must never block phase work")
	require.True(t, pctx.HasCurrent, "phase work proceeds despite a degraded freshen")
	require.Equal(t, execution.FreshenStatusDegraded, pctx.FreshenStatus)
	require.Contains(t, pctx.FreshenDetail, "git-control-tower unavailable")

	// A degraded freshen is re-attempted on the next resume (agent can retry).
	_, _, _, err = h.svc.Resume(ctx, e.ID, "", "")
	require.NoError(t, err)
	require.Equal(t, 2, fr.calls, "a degraded freshen is retried on resume")
}

// TestFreshenReportsStalenessSummary proves the staleness recompute is surfaced
// (reported only) in the freshen detail without mutating the authored plan.
func TestFreshenReportsStalenessSummary(t *testing.T) {
	fr := &fakeFreshener{result: execution.FreshenResult{
		BaselineCaptured: true,
		BaselineName:     "plan-1-baseline",
		StalenessSummary: "staleness: 2 reference(s), overall=fresh",
	}}
	h := newHarnessWithFreshener(t, threePhasePlan(), fr)
	ctx := context.Background()

	_, pctx, _, err := h.svc.Start(ctx, "plan-1", "run-1")
	require.NoError(t, err)
	require.Contains(t, pctx.FreshenDetail, "staleness: 2 reference(s)")
	// The authored plan references are untouched by the freshen (report-only).
	require.Empty(t, h.store.plan.References)
}

// TestFreshenSkippedWhenNoSeam proves a nil freshener is a silent skip (no
// fabricated capture, no block).
func TestFreshenSkippedWhenNoSeam(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil) // no freshener wired
	ctx := context.Background()
	_, pctx, _, err := h.svc.Start(ctx, "plan-1", "run-1")
	require.NoError(t, err)
	require.False(t, pctx.InputsFreshened, "no freshener wired => freshen is skipped silently")
	require.Empty(t, pctx.FreshenStatus)
}

// TestChangeBoundarySurfacedInContextAndHandoff proves the plan's change boundary
// is surfaced in the just-in-time phase context (with reminders) and snapshotted
// into the canonical handoff.
func TestChangeBoundarySurfacedInContextAndHandoff(t *testing.T) {
	plan := threePhasePlan()
	plan.ChangeBoundary = internalplans.ChangeBoundary{
		AcceptanceAllow: []string{"scenarios/plan-manager/**", "packages/proto/**"},
		AcceptanceDeny:  []string{"scenarios/swarm-manager/**"},
	}
	h := newHarness(t, plan, nil)
	ctx := context.Background()

	_, pctx, step, err := h.svc.Start(ctx, "plan-1", "run-boundary")
	require.NoError(t, err)
	require.Equal(t, []string{"scenarios/plan-manager/**", "packages/proto/**"}, pctx.ChangeBoundary.AcceptanceAllow)

	// Status surfaces the boundary reminders in the phase-context step.
	_, _, statusStep, err := h.svc.GetStatus(ctx, executionIDFromStep(t, step))
	require.NoError(t, err)
	require.Equal(t, "phase_context", statusStep.StepKind)
	joined := strings.Join(statusStep.Instructions, " | ")
	require.Contains(t, joined, "only edit within")
	require.Contains(t, joined, "Forbidden paths")
	require.Contains(t, joined, "informational only", "repo paths must be flagged as non-oracle")

	// Drive every phase done and complete: the handoff snapshots the boundary.
	for _, ph := range plan.Phases {
		exec, _, _, err := h.svc.GetStatus(ctx, executionIDFromStep(t, step))
		require.NoError(t, err)
		_, _, _, err = h.svc.TransitionPhase(ctx, exec.ID, ph.ID, doneOverride())
		require.NoError(t, err)
	}
	exec, _, _, err := h.svc.GetStatus(ctx, executionIDFromStep(t, step))
	require.NoError(t, err)
	handoff, _, _, err := h.svc.Complete(ctx, exec.ID, execution.CompletionInputs{})
	require.NoError(t, err)
	require.Equal(t, plan.ChangeBoundary.AcceptanceAllow, handoff.ChangeBoundary.AcceptanceAllow)
	require.Equal(t, plan.ChangeBoundary.AcceptanceDeny, handoff.ChangeBoundary.AcceptanceDeny)
}

func TestPhaseContextStepSurfacesFeedbackCaptureActions(t *testing.T) {
	h := newHarness(t, threePhasePlan(), nil)
	ctx := context.Background()

	e, _, _, err := h.svc.Start(ctx, "plan-1", "run-feedback")
	require.NoError(t, err)
	_, _, step, err := h.svc.GetStatus(ctx, e.ID)
	require.NoError(t, err)
	require.Equal(t, "phase_context", step.StepKind)

	joinedInstructions := strings.Join(step.Instructions, " | ")
	require.Contains(t, joinedInstructions, "Capture feedback in the log ledger")

	actionIDs := map[string]bool{}
	for _, action := range step.NextActions {
		actionIDs[action.ID] = true
	}
	require.True(t, actionIDs["log-decision"], "phase context should expose decision capture")
	require.True(t, actionIDs["log-finding"], "phase context should expose candidate finding capture")
	require.True(t, actionIDs["log-record"], "phase context should expose reusable record capture")
	require.True(t, actionIDs["log-note"], "phase context should expose progress note capture")
}

func executionIDFromStep(t *testing.T, step execution.GuidedStep) string {
	t.Helper()
	for _, a := range step.NextActions {
		if len(a.Argv) >= 3 && a.Argv[0] == "exec" {
			return a.Argv[2]
		}
	}
	t.Fatalf("no execution id in step actions: %+v", step.NextActions)
	return ""
}
