package execution

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/provenance"
	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// NoFeedbackCheckpointTitle is the durable note title that explicitly records a
// phase feedback review found nothing to capture.
const NoFeedbackCheckpointTitle = "Phase feedback reviewed: none"

const noFeedbackCheckpointDetail = "No decisions, findings, bugs, records, or reusable notes to capture for this phase."

// Service is the execution application surface — the guided runner.
type Service interface {
	Start(ctx context.Context, planID, runID string) (Execution, PhaseContext, GuidedStep, error)
	GetStatus(ctx context.Context, executionID string) (Execution, PhaseContext, GuidedStep, error)
	GetContext(ctx context.Context, executionID, phaseID string) (Execution, PhaseContext, GuidedStep, error)
	SyncBaseline(ctx context.Context, executionID string) (Execution, PhaseContext, GuidedStep, error)
	AmendScope(ctx context.Context, executionID string, req ScopeAmendmentRequest) (Execution, PhaseContext, GuidedStep, error)
	AdoptBaseline(ctx context.Context, executionID string, req BaselineAdoptionRequest) (Execution, PhaseContext, GuidedStep, error)
	RepairSourceScope(ctx context.Context, executionID string, req SourceScopeRepairRequest) (Execution, PhaseContext, GuidedStep, error)
	// ExtendBoundary returns the globs it actually added. An empty slice with a
	// nil error is a successful no-op (every requested glob was already inside
	// the boundary) — callers must not infer the outcome from the execution's
	// extension history, which may hold an earlier entry.
	ExtendBoundary(ctx context.Context, executionID string, req BoundaryExtensionRequest) (Execution, PhaseContext, GuidedStep, []string, error)
	Resume(ctx context.Context, planOrExecution, phaseID, runID string) (Execution, PhaseContext, GuidedStep, error)
	ContinueExecution(ctx context.Context, planOrExecution, phaseID, runID string) (Execution, PhaseContext, GuidedStep, error)
	AbandonExecution(ctx context.Context, executionID, reason, actor string) (Execution, bool, GuidedStep, error)
	GetNext(ctx context.Context, executionID string) (PhaseContext, bool, GuidedStep, error)
	TransitionPhase(ctx context.Context, executionID, phaseID string, inputs PhaseTransitionInputs) (Execution, planmodel.Plan, GuidedStep, error)

	Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error)
	PartialHandoff(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error)
	GetHandoff(ctx context.Context, executionID string) (Handoff, GuidedStep, error)

	GetVelocity(ctx context.Context, planID string) ([]VelocityPoint, GuidedStep, error)
}

type service struct {
	repo      Repository
	plans     PlanStore
	validator Validator
	log       LogLedger
	velocity  VelocitySink
	baseline  BaselineSynchronizer
	preflight SourceEvidencePreflighter
	clock     clock.Clock
	startMu   sync.Mutex
}

// Deps wires the execution Service. Repo + Plans are required; Validator is
// optional (nil => last_validation/staleness degrade to UNKNOWN, never a false
// pass). LogLedger is optional (nil => empty log summaries; the handoff still
// assembles). VelocitySink is optional (nil => the no-op default; velocity is
// still persisted locally regardless). InputFreshener is optional (nil => the
// execution-start freshen step is skipped silently; phase work never blocks).
type Deps struct {
	Repo      Repository
	Plans     PlanStore
	Validator Validator
	Log       LogLedger
	Velocity  VelocitySink
	Baseline  BaselineSynchronizer
	Preflight SourceEvidencePreflighter
	Clock     clock.Clock
}

// NewService constructs the execution Service.
func NewService(d Deps) Service {
	clk := d.Clock
	if clk == nil {
		clk = clock.System{}
	}
	sink := d.Velocity
	if sink == nil {
		sink = DefaultVelocitySink()
	}
	return &service{
		repo:      d.Repo,
		plans:     d.Plans,
		validator: d.Validator,
		log:       d.Log,
		velocity:  sink,
		baseline:  d.Baseline,
		preflight: d.Preflight,
		clock:     clk,
	}
}

var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context, planID, runID string) (Execution, PhaseContext, GuidedStep, error) {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "plan id is required"}
	}
	// Resolve the plan (id or slug) through the plans SSOT so the linkage stores
	// the canonical plan id, and so a bad plan fails fast with NotFound.
	plan, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if err := requireExecutionGradePlan(plan, false); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	active, err := s.activeExecutionsForPlan(ctx, plan.ID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if len(active) > 0 {
		ids := make([]string, 0, len(active))
		for _, existing := range active {
			ids = append(ids, existing.ID)
		}
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrActiveExecutionConflict{PlanID: plan.ID, ExecutionIDs: ids}
	}
	return s.startAtPhase(ctx, plan, "", runID, contextModeStart)
}

func (s *service) startAtPhase(ctx context.Context, plan planmodel.Plan, phaseID, runID string, mode contextMode) (Execution, PhaseContext, GuidedStep, error) {
	prov := provenance.FromContext(ctx)
	_, _, _, verificationStatus, verifiedRunID, _ := prov.WriteFields()
	harnessSessionID, harnessKind := prov.ObservationFields()
	runID = verifiedRunID
	currentPhaseID, err := resolveExecutionPhaseID(plan, phaseID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	now := s.now()
	e := Execution{
		ID:                 uuid.NewString(),
		PlanID:             plan.ID,
		RunID:              runID,
		VerificationStatus: verificationStatus,
		HarnessSessionID:   harnessSessionID,
		HarnessKind:        harnessKind,
		CurrentPhaseID:     currentPhaseID,
		StartedAt:          now,
		UpdatedAt:          now,
		LifecycleState:     ExecutionLifecycleActive,
	}
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	s.ensureBaselineTicket(ctx, &e, plan)
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, mode)
	s.applyFreshenContext(&pctx, e)
	if e.BaselineSet.Status == BaselineSetStatusScopeRepairRequired || e.BaselineSet.PreflightUnavailable {
		return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
	}
	return e, pctx, stepForStarted(e), nil
}

func (s *service) resumeExecution(ctx context.Context, e Execution, phaseID string) (Execution, PhaseContext, GuidedStep, error) {
	if e.EffectiveLifecycleState() == ExecutionLifecycleAbandoned {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "abandoned execution cannot be resumed: " + e.ID}
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if err := requireExecutionGradePlan(plan, hasExplicitBaselineRecovery(e)); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	return s.resumeExecutionWithPlan(ctx, e, plan, phaseID)
}

func (s *service) resumeExecutionWithPlan(ctx context.Context, e Execution, plan planmodel.Plan, phaseID string) (Execution, PhaseContext, GuidedStep, error) {
	currentPhaseID, err := resolveExecutionPhaseID(plan, phaseID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	e.CurrentPhaseID = currentPhaseID
	e.Complete = currentPhaseID == ""
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	s.ensureBaselineTicket(ctx, &e, plan)
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeResume)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

func (s *service) GetStatus(ctx context.Context, executionID string) (Execution, PhaseContext, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeStatus)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

func (s *service) GetContext(ctx context.Context, executionID, phaseID string) (Execution, PhaseContext, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	targetPhaseID := strings.TrimSpace(phaseID)
	if targetPhaseID == "" {
		targetPhaseID = e.CurrentPhaseID
	}
	pctx := s.buildContext(ctx, plan, targetPhaseID, e.ID, contextModeContext)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

func (s *service) Resume(ctx context.Context, planOrExecution, phaseID, runID string) (Execution, PhaseContext, GuidedStep, error) {
	planOrExecution = strings.TrimSpace(planOrExecution)
	if planOrExecution == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "plan or execution id is required"}
	}
	if e, ok, err := s.repo.GetExecution(ctx, planOrExecution); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	} else if ok {
		return s.resumeExecution(ctx, e, phaseID)
	}
	plan, err := s.plans.GetPlan(ctx, planOrExecution)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if e, ok, err := s.latestActiveExecutionForPlan(ctx, plan.ID); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	} else if ok {
		if err := requireExecutionGradePlan(plan, hasExplicitBaselineRecovery(e)); err != nil {
			return Execution{}, PhaseContext{}, GuidedStep{}, err
		}
		return s.resumeExecutionWithPlan(ctx, e, plan, phaseID)
	}
	if err := requireExecutionGradePlan(plan, false); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	// No prior execution exists for this plan: resume/continue is creating a NEW
	// run, so this is a first start — emit once-per-execution setup context exactly
	// once (contextModeStart), not the resume context mode. This is the fix for
	// once-per-execution context being skipped on the continue/resume create path.
	return s.startAtPhase(ctx, plan, strings.TrimSpace(phaseID), runID, contextModeStart)
}

func (s *service) AbandonExecution(ctx context.Context, executionID, reason, actor string) (Execution, bool, GuidedStep, error) {
	executionID = strings.TrimSpace(executionID)
	reason = strings.TrimSpace(reason)
	actor = strings.TrimSpace(actor)
	if reason == "" {
		return Execution{}, false, GuidedStep{}, ErrInvalidExecution{Reason: "abandon reason is required"}
	}
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, false, GuidedStep{}, err
	}
	switch e.EffectiveLifecycleState() {
	case ExecutionLifecycleAbandoned:
		return e, true, stepForAbandoned(e), nil
	case ExecutionLifecycleCompleted:
		return Execution{}, false, GuidedStep{}, ErrInvalidExecution{Reason: "completed execution cannot be abandoned: " + e.ID}
	}
	if actor == "" {
		actor = strings.TrimSpace(e.RunID)
	}
	if actor == "" {
		actor = "operator"
	}
	now := s.now()
	e.LifecycleState = ExecutionLifecycleAbandoned
	e.AbandonedReason = reason
	e.AbandonedAt = now
	e.AbandonedBy = actor
	e.UpdatedAt = now
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, false, GuidedStep{}, err
	}
	return e, false, stepForAbandoned(e), nil
}

func (s *service) activeExecutionsForPlan(ctx context.Context, planID string) ([]Execution, error) {
	if lister, ok := s.repo.(activeExecutionLister); ok {
		all, err := lister.ListExecutionsForPlan(ctx, planID)
		if err != nil {
			return nil, err
		}
		active := make([]Execution, 0, len(all))
		for _, e := range all {
			if e.EffectiveLifecycleState() == ExecutionLifecycleActive {
				active = append(active, e)
			}
		}
		return active, nil
	}
	e, ok, err := s.repo.LatestExecutionForPlan(ctx, planID)
	if err != nil || !ok || e.EffectiveLifecycleState() != ExecutionLifecycleActive {
		return nil, err
	}
	return []Execution{e}, nil
}

func (s *service) latestActiveExecutionForPlan(ctx context.Context, planID string) (Execution, bool, error) {
	active, err := s.activeExecutionsForPlan(ctx, planID)
	if err != nil || len(active) == 0 {
		return Execution{}, false, err
	}
	return active[0], true, nil
}

func (s *service) ContinueExecution(ctx context.Context, planOrExecution, phaseID, runID string) (Execution, PhaseContext, GuidedStep, error) {
	e, pctx, _, err := s.Resume(ctx, planOrExecution, phaseID, runID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	step := stepForContext(e.ID, e.PlanID, pctx, e.Complete)
	return e, pctx, onlyRecommendedExecutionAction(step), nil
}

func (s *service) GetNext(ctx context.Context, executionID string) (PhaseContext, bool, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return PhaseContext{}, false, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return PhaseContext{}, false, GuidedStep{}, err
	}
	// Advance means "move past the current pointer" when a later non-done phase
	// exists. The resume point remains the earliest non-done phase and is exposed
	// by GetStatus/buildContext; using it here would repeat the current phase.
	next := nextActionablePhaseID(plan.Phases, e.CurrentPhaseID)
	resume := resumePhaseID(plan.Phases)
	if next == "" {
		next = resume
	}
	e.CurrentPhaseID = next
	e.Complete = resume == ""
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return PhaseContext{}, false, GuidedStep{}, err
	}
	pctx := s.buildContext(ctx, plan, next, e.ID, contextModePhaseEntry)
	s.applyFreshenContext(&pctx, e)
	return pctx, e.Complete, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

func (s *service) TransitionPhase(ctx context.Context, executionID, phaseID string, inputs PhaseTransitionInputs) (Execution, planmodel.Plan, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, planmodel.Plan{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, planmodel.Plan{}, GuidedStep{}, err
	}
	target, ok := findPhase(plan.Phases, phaseID)
	if !ok {
		return Execution{}, planmodel.Plan{}, GuidedStep{}, planmodel.ErrPhaseNotFound{PlanID: plan.ID, PhaseID: phaseID}
	}
	to := inputs.ToStatus
	if to == "" {
		return Execution{}, planmodel.Plan{}, GuidedStep{}, ErrInvalidExecution{Reason: "target phase status is required"}
	}
	if to == planmodel.PhaseStatusActive || to == planmodel.PhaseStatusDone {
		if err := requireBaselineReady(e.BaselineSet); err != nil {
			return Execution{}, planmodel.Plan{}, GuidedStep{}, err
		}
	}
	if to == planmodel.PhaseStatusDone {
		if err := s.requireValidationForDone(ctx, e, plan, target.ID, inputs.ValidationOverrideReason); err != nil {
			return Execution{}, planmodel.Plan{}, GuidedStep{}, err
		}
		if err := s.requireFeedbackForDone(ctx, e.ID, target.ID, inputs.FeedbackOverrideReason); err != nil {
			return Execution{}, planmodel.Plan{}, GuidedStep{}, err
		}
	}
	// Delegate the phase-status change to the plans domain — it stays the single
	// source of truth for the record (plan status is recomputed there).
	target.Status = to
	updated, err := s.plans.UpdatePhase(ctx, plan.ID, plan.WorkspaceID, plan.WorkspaceRoot, target)
	if err != nil {
		return Execution{}, planmodel.Plan{}, GuidedStep{}, err
	}
	// Move the runner's pointer to the next actionable phase and mark the
	// execution complete when every phase is terminal.
	e.CurrentPhaseID = resumePhaseID(updated.Phases)
	e.Complete = e.CurrentPhaseID == ""
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, planmodel.Plan{}, GuidedStep{}, err
	}
	return e, updated, stepForTransition(e), nil
}

func (s *service) Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	// Idempotent completion: an execution that already produced a full handoff
	// returns it again instead of re-running the completion gates. Without this a
	// second `exec complete` (e.g. after a transient client error, or an operator
	// retry) re-validated and could fail with a validation error even though the
	// completion had already landed (knw-1784053356805823492). The last phase
	// transitioning to done sets e.Complete=true but writes NO handoff, so the
	// FIRST real completion still runs every gate below.
	if e.Complete {
		if h, ok, herr := s.repo.GetHandoff(ctx, e.ID); herr != nil {
			return Handoff{}, nil, GuidedStep{}, herr
		} else if ok && h.Completeness == CompletenessFull {
			logSummary, _ := s.logLedger(ctx, e.ID)
			nudges := s.completionNudges(plan, logSummary)
			return h, nudges, stepForComplete(e.ID, nudges), nil
		}
	}
	if computeCompleteness(plan.Phases) != CompletenessFull {
		return Handoff{}, nil, GuidedStep{}, ErrInvalidExecution{Reason: "normal completion requires every phase done; use partial-handoff for an honest incomplete handoff"}
	}
	if e.DegradedReason != "" {
		return Handoff{}, nil, GuidedStep{}, ErrInvalidExecution{Reason: "normal completion is disabled: " + e.DegradedReason}
	}
	if e.BaselineSet.Name != "" {
		if err := requireBaselineReady(e.BaselineSet); err != nil {
			return Handoff{}, nil, GuidedStep{}, err
		}
		if err := s.requireFinalValidation(ctx, e, plan); err != nil {
			return Handoff{}, nil, GuidedStep{}, err
		}
	}
	logSummary, logEntries := s.logLedger(ctx, e.ID)

	// Assemble the canonical handoff from captured state. Completeness + resume
	// point are COMPUTED from the phase-status set; last_validation/staleness come
	// from the validation seam (degrade to UNKNOWN, never a false pass); the log
	// summary/entries come from the log domain through the LogLedger seam.
	completeness := computeCompleteness(plan.Phases)
	resume := resumePhaseID(plan.Phases)
	lastVal, hasVal, staleness := s.lastValidation(ctx, plan, e.CurrentPhaseID)

	now := s.now()
	handoff := Handoff{
		ID:              uuid.NewString(),
		ExecutionID:     e.ID,
		PlanID:          plan.ID,
		Completeness:    completeness,
		ResumePhaseID:   resume,
		LogSummary:      logSummary,
		LogEntries:      logEntries,
		LastValidation:  lastVal,
		HasValidation:   hasVal,
		Staleness:       staleness,
		ProseHandoffRef: "", // pass-through; the orchestration layer fills this by reference
		AssembledAt:     now,
		ChangeBoundary:  plan.ChangeBoundary,
	}
	// Capture the velocity point LOCAL ONLY — persisted regardless, then offered
	// to the (stubbed) MoM emit seam after the durable write commits.
	point := VelocityPoint{
		ID:              uuid.NewString(),
		PlanID:          plan.ID,
		RunID:           velocityRunKey(e),
		WallTimeSeconds: s.wallTime(e.StartedAt, now),
		Tokens:          inputs.Tokens,
		Iterations:      inputs.Iterations,
		Completeness:    completeness,
		RecordedAt:      now,
	}
	e.Complete = true
	e.LifecycleState = ExecutionLifecycleCompleted
	e.UpdatedAt = now

	// Persist the handoff, the velocity point, and the completed-execution state as
	// ONE transaction so a mid-sequence failure cannot leave a half-written
	// completion (a handoff with no velocity, or an execution marked complete with
	// no handoff).
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.SaveHandoff(ctx, handoff); err != nil {
			return err
		}
		if err := repo.SaveVelocity(ctx, point); err != nil {
			return err
		}
		return repo.SaveExecution(ctx, e)
	}); err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	_ = s.velocity.Emit(ctx, point) // best-effort; the no-op default never errors

	nudges := s.completionNudges(plan, logSummary)
	return handoff, nudges, stepForComplete(e.ID, nudges), nil
}

// PartialHandoff persists a truthful checkpoint without changing the execution
// to complete. It is intentionally separate from Complete so automation cannot
// turn missing full-inventory evidence into a normal completion.
func (s *service) PartialHandoff(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	logSummary, logEntries := s.logLedger(ctx, e.ID)
	lastVal, hasVal, staleness := s.lastValidation(ctx, plan, e.CurrentPhaseID)
	now := s.now()
	handoff := Handoff{
		ID: uuid.NewString(), ExecutionID: e.ID, PlanID: plan.ID, Completeness: CompletenessPartial,
		ResumePhaseID: resumePhaseID(plan.Phases), LogSummary: logSummary, LogEntries: logEntries, LastValidation: lastVal,
		HasValidation: hasVal, Staleness: staleness, AssembledAt: now, ChangeBoundary: plan.ChangeBoundary,
	}
	point := VelocityPoint{ID: uuid.NewString(), PlanID: plan.ID, RunID: velocityRunKey(e), WallTimeSeconds: s.wallTime(e.StartedAt, now), Tokens: inputs.Tokens, Iterations: inputs.Iterations, Completeness: CompletenessPartial, RecordedAt: now}
	e.Complete, e.UpdatedAt = false, now
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.SaveHandoff(ctx, handoff); err != nil {
			return err
		}
		if err := repo.SaveVelocity(ctx, point); err != nil {
			return err
		}
		return repo.SaveExecution(ctx, e)
	}); err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	_ = s.velocity.Emit(ctx, point)
	return handoff, s.completionNudges(plan, logSummary), stepForHandoff(e.ID), nil
}

func (s *service) GetHandoff(ctx context.Context, executionID string) (Handoff, GuidedStep, error) {
	if _, err := s.getExecution(ctx, executionID); err != nil {
		return Handoff{}, GuidedStep{}, err
	}
	h, ok, err := s.repo.GetHandoff(ctx, executionID)
	if err != nil {
		return Handoff{}, GuidedStep{}, err
	}
	if !ok {
		// No handoff assembled yet — assemble a live view from current captured
		// state so a caller asking before Complete still gets the structured shape
		// rather than an error (the persisted record is written by Complete).
		h, err := s.assembleLiveHandoff(ctx, executionID)
		return h, stepForHandoff(executionID), err
	}
	return h, stepForHandoff(executionID), nil
}

func (s *service) GetVelocity(ctx context.Context, planID string) ([]VelocityPoint, GuidedStep, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return nil, GuidedStep{}, ErrInvalidExecution{Reason: "plan id is required"}
	}
	// Resolve a slug to the canonical id so velocity recorded under the id is
	// found when the caller passes the slug.
	if plan, err := s.plans.GetPlan(ctx, planID); err == nil {
		planID = plan.ID
	}
	points, err := s.repo.ListVelocity(ctx, planID)
	if err != nil {
		return nil, GuidedStep{}, err
	}
	if lister, ok := s.repo.(activeExecutionLister); ok {
		executions, listErr := lister.ListExecutionsForPlan(ctx, planID)
		if listErr != nil {
			return nil, GuidedStep{}, listErr
		}
		abandonedRuns := make(map[string]struct{})
		for _, execution := range executions {
			if execution.EffectiveLifecycleState() == ExecutionLifecycleAbandoned {
				abandonedRuns[velocityRunKey(execution)] = struct{}{}
			}
		}
		kept := points[:0]
		for _, point := range points {
			if _, abandoned := abandonedRuns[point.RunID]; !abandoned {
				kept = append(kept, point)
			}
		}
		points = kept
	}
	return points, GuidedStep{}, nil
}

func velocityRunKey(e Execution) string {
	if strings.TrimSpace(e.RunID) != "" {
		return e.RunID
	}
	return e.ID
}

// --- helpers ---

func requireExecutionGradePlan(plan planmodel.Plan, allowRecoveredLegacy bool) error {
	report := planmodel.AssessPlanQuality(plan, "")
	if allowRecoveredLegacy && onlyBaselineSetFailures(report) {
		return nil
	}
	if report.ExecutionReady() {
		return nil
	}
	return ErrInvalidExecution{Reason: "plan is not execution-grade; repair before starting execution: " + summarizeQualityFailures(report)}
}

func hasExplicitBaselineRecovery(e Execution) bool {
	return e.BaselineSet.Name != "" || e.BaselineSet.LegacyAdoptionRequired || e.DegradedReason != ""
}

func onlyBaselineSetFailures(report planmodel.QualityReport) bool {
	for _, finding := range report.Findings {
		if finding.Severity != planmodel.QualitySeverityFail {
			continue
		}
		if finding.Code != "plan_missing_baseline_set" && finding.Code != "plan_invalid_baseline_set" && finding.Code != "plan_baseline_set_no_scenarios" {
			return false
		}
	}
	return true
}

func summarizeQualityFailures(report planmodel.QualityReport) string {
	var parts []string
	for _, finding := range report.Findings {
		if finding.Severity != planmodel.QualitySeverityFail {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s at %s", finding.Code, finding.Location))
		if len(parts) == 4 {
			break
		}
	}
	if len(parts) == 0 {
		return report.Status
	}
	remaining := 0
	for _, finding := range report.Findings {
		if finding.Severity == planmodel.QualitySeverityFail {
			remaining++
		}
	}
	if remaining > len(parts) {
		parts = append(parts, fmt.Sprintf("and %d more", remaining-len(parts)))
	}
	return strings.Join(parts, "; ")
}

// freshenInputs runs the one-time execution-start freshen step: it captures the
// regression-anchor's baseline snapshot fresh and recomputes reference staleness,
// delegated to the validation domain through the InputFreshener seam. It runs at
// most once successfully per execution (recorded on the Execution record) and is
// re-attempted on a later start/resume only while it has not yet succeeded — so a
// transient git-control-tower outage is retryable but a captured baseline is not
// re-captured. It NEVER blocks phase work: a nil seam or an error is recorded as a
// degraded freshen status and surfaced, not returned.
// ensureBaselineTicket records policy and exact producer argv only. Starting an
// execution must never start or wait for a producer operation behind the agent's
// back: Git Control Tower owns both parts of that lifecycle.
func (s *service) ensureBaselineTicket(ctx context.Context, e *Execution, plan planmodel.Plan) {
	if e.BaselineSet.Name != "" || e.BaselineSet.LegacyAdoptionRequired {
		return
	}
	if plan.BaselineSet.IsLegacy() {
		e.BaselineSet = BaselineSetState{
			Version: BaselineSetStateSchemaVersion, Status: BaselineSetStatusRequired, LegacyAdoptionRequired: true,
			Detail: "historical plan requires explicit baseline adoption before normal phase work; recapture only when the current worktree is a trustworthy before-state, otherwise record a degraded partial-handoff path",
		}
		e.FreshenStatus, e.FreshenDetail, e.UpdatedAt = "baseline_adoption_required", e.BaselineSet.Detail, s.now()
		_ = s.repo.SaveExecution(ctx, *e)
		return
	}
	if strings.TrimSpace(plan.BaselineSet.Name) == "" {
		return
	}
	name := strings.TrimSpace(plan.BaselineSet.Name)
	state := BaselineSetState{
		Version: BaselineSetStateSchemaVersion, Name: name,
		ScenarioTargets: append([]string(nil), plan.BaselineSet.ScenarioTargets...),
		RepoPaths:       append([]string(nil), plan.BaselineSet.RepoPaths...),
		Status:          BaselineSetStatusRequired,
		Detail:          "baseline capture has not been started; run the producer-owned capture action, use GCT's printed wait command, then synchronize this ticket",
		WaitArgv:        []string{"git-control-tower", "baseline", "collection", "show", "--name", name, "--wait", "--json"},
		SyncArgv:        []string{"plan-manager", "exec", "baseline-sync", e.ID},
	}
	if len(plan.BaselineSet.RepoPaths) > 0 && s.preflight != nil {
		preflight, preflightErr := s.preflight.EstimateSourceEvidence(ctx, plan.BaselineSet.RepoPaths)
		if preflightErr != nil {
			state.PreflightUnavailable = true
			state.Detail = "Git Control Tower source-evidence preflight is unavailable: " + preflightErr.Error()
			state.Status = BaselineSetStatusDegraded
		} else {
			state.SourcePreflight = preflight
			if preflight.RepairRequired {
				state.Status = BaselineSetStatusScopeRepairRequired
				state.Detail = sourcePreflightDetail(preflight)
			}
		}
	}
	capture := []string{"git-control-tower", "baseline", "collection", "capture", "--name", name}
	for _, scenario := range plan.BaselineSet.ScenarioTargets {
		capture = append(capture, "--member", scenario)
	}
	for _, path := range plan.BaselineSet.RepoPaths {
		capture = append(capture, "--path", path)
	}
	if state.Status == BaselineSetStatusRequired {
		state.CaptureArgv = capture
	}
	e.BaselineSet = state
	e.FreshenStatus, e.FreshenDetail, e.UpdatedAt = "baseline_required", e.BaselineSet.Detail, s.now()
	_ = s.repo.SaveExecution(ctx, *e)
}

func sourcePreflightDetail(preflight SourceEvidencePreflight) string {
	for _, issue := range preflight.Issues {
		if issue.Severity == "repair-required" || issue.Severity == "error" {
			return "source evidence scope repair required: " + issue.Detail
		}
	}
	return "source evidence scope repair required by Git Control Tower"
}

// RepairSourceScope is the sole execution-time repair lane for informational
// source evidence. It never changes GCT collection members or plan boundary;
// replacements must be strictly contained by the authored allow globs and are
// immediately re-estimated by the authoritative provider.
func (s *service) RepairSourceScope(ctx context.Context, executionID string, req SourceScopeRepairRequest) (Execution, PhaseContext, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if e.BaselineSet.Status != BaselineSetStatusScopeRepairRequired {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "source-scope repair requires a repair-required baseline checkpoint"}
	}
	if len(req.Paths) == 0 || strings.TrimSpace(req.Reason) == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "replacement source paths and repair reason are required"}
	}
	for _, path := range req.Paths {
		if !sourcePathWithinBoundary(path, plan.ChangeBoundary.AcceptanceAllow) {
			return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "replacement source path is outside the plan change boundary: " + path}
		}
	}
	if s.preflight == nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "Git Control Tower source-evidence preflight is unavailable"}
	}
	preflight, err := s.preflight.EstimateSourceEvidence(ctx, req.Paths)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "source-evidence preflight failed: " + err.Error()}
	}
	e.BaselineSet.RepoPaths = append([]string(nil), req.Paths...)
	e.BaselineSet.SourcePreflight, e.BaselineSet.PreflightUnavailable = preflight, false
	if preflight.RepairRequired {
		e.BaselineSet.Detail = sourcePreflightDetail(preflight)
		e.UpdatedAt = s.now()
		_ = s.repo.SaveExecution(ctx, e)
		pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeStatus)
		s.applyFreshenContext(&pctx, e)
		return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
	}
	capture := []string{"git-control-tower", "baseline", "collection", "capture", "--name", e.BaselineSet.Name}
	for _, scenario := range e.BaselineSet.ScenarioTargets {
		capture = append(capture, "--member", scenario)
	}
	for _, path := range e.BaselineSet.RepoPaths {
		capture = append(capture, "--path", path)
	}
	e.BaselineSet.Status, e.BaselineSet.Detail, e.BaselineSet.CaptureArgv = BaselineSetStatusRequired, "source scope repaired and re-estimated; run the producer-owned capture action", capture
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeStatus)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

func sourcePathWithinBoundary(path string, allows []string) bool {
	path = strings.Trim(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"), "/")
	for _, allow := range allows {
		allow = strings.Trim(strings.ReplaceAll(strings.TrimSpace(allow), "\\", "/"), "/")
		base := strings.TrimSuffix(allow, "**")
		if path == allow || (base != "" && strings.HasPrefix(path, base)) {
			return true
		}
	}
	return false
}

// SyncBaseline is a one-shot nonblocking typed read of GCT's durable collection
// state. It deliberately offers no wait option or local retry loop.
func (s *service) SyncBaseline(ctx context.Context, executionID string) (Execution, PhaseContext, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if e.BaselineSet.Name == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "execution has no baseline collection ticket; repair or adopt the legacy execution before normal completion"}
	}
	e.BaselineSet.LastSyncedAt = s.now()
	if s.baseline == nil {
		e.BaselineSet.Status, e.BaselineSet.Detail = BaselineSetStatusDegraded, "Git Control Tower baseline synchronization is unavailable"
	} else if result, syncErr := s.baseline.SyncBaseline(ctx, plan.ID, e.BaselineSet.Name); syncErr != nil {
		e.BaselineSet.Status, e.BaselineSet.Detail = BaselineSetStatusDegraded, "baseline sync failed: "+syncErr.Error()
	} else {
		state := result.BaselineSet
		state.CaptureArgv, state.WaitArgv, state.SyncArgv = e.BaselineSet.CaptureArgv, e.BaselineSet.WaitArgv, e.BaselineSet.SyncArgv
		state.LastSyncedAt = e.BaselineSet.LastSyncedAt
		if state.CapturedAt == "" && state.Complete() {
			state.CapturedAt = e.BaselineSet.LastSyncedAt
		}
		e.BaselineSet = state
		e.FreshenStatus, e.FreshenDetail = "baseline_synced", state.Detail
	}
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeStatus)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

// AmendScope appends an execution-local scope decision. It only accepts members
// already present in the synchronized immutable collection; callers must use
// GCT collection extend + its native wait + baseline-sync before naming a new
// member here. Any amendment invalidates the phase's prior evidence generation.
func (s *service) AmendScope(ctx context.Context, executionID string, req ScopeAmendmentRequest) (Execution, PhaseContext, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	phaseID := strings.TrimSpace(req.PhaseID)
	if _, ok := findPhase(plan.Phases, phaseID); !ok {
		return Execution{}, PhaseContext{}, GuidedStep{}, planmodel.ErrPhaseNotFound{PlanID: plan.ID, PhaseID: phaseID}
	}
	if strings.TrimSpace(req.Reason) == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "scope amendment reason is required"}
	}
	available := make(map[string]struct{}, len(e.BaselineSet.Members))
	for _, member := range e.BaselineSet.Members {
		if member.Scenario != "" {
			available[member.Scenario] = struct{}{}
		}
	}
	if len(available) == 0 || !e.BaselineSet.Complete() {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "scope amendment requires a synchronized complete baseline collection"}
	}
	oldMinimum := effectivePhaseMinimum(e, plan, phaseID)
	newMinimum := append([]string(nil), oldMinimum...)
	seen := make(map[string]struct{}, len(newMinimum))
	for _, member := range newMinimum {
		seen[member] = struct{}{}
	}
	for _, member := range req.Members {
		member = strings.TrimSpace(member)
		if member == "" {
			continue
		}
		if _, ok := available[member]; !ok {
			return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "scenario " + member + " is not in the captured baseline inventory; extend in Git Control Tower, use its native wait, and baseline-sync before amending scope"}
		}
		if _, ok := seen[member]; !ok {
			newMinimum, seen[member] = append(newMinimum, member), struct{}{}
		}
	}
	if len(newMinimum) == len(oldMinimum) {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "scope amendment adds no captured validation members"}
	}
	if e.PhaseValidationGenerations == nil {
		e.PhaseValidationGenerations = map[string]int{}
	}
	e.PhaseValidationGenerations[phaseID]++
	now := s.now()
	e.ScopeAmendments = append(e.ScopeAmendments, ScopeAmendment{ID: uuid.NewString(), PhaseID: phaseID, Author: strings.TrimSpace(req.Author), Reason: strings.TrimSpace(req.Reason), OldMinimum: oldMinimum, NewMinimum: newMinimum, InvalidatedAt: now, CreatedAt: now})
	e.UpdatedAt = now
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeStatus)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

// ExtendBoundary widens the plan's acceptance_allow mid-execution so validation
// scope follows a sanctioned scope expansion.
//
// The authored change boundary is an estimate made before the work started. Its
// inaccuracy is only discoverable during execution — precisely when, before this
// lane existed, the plan was frozen. That left an executing agent two bad
// options when a phase's intent genuinely needed an edit outside the boundary:
// write a workaround that stays inside a stale estimate, or stop and report a
// blocker. This is the third option, and it is the one that keeps the code clean
// AND keeps the evidence honest.
//
// The widening is deliberately constrained:
//   - Append-only, delegated to the plans domain, which re-checks acceptance_deny
//     and placeholder rules. A forbidden path is still forbidden.
//   - The current phase's validation generation is advanced, so evidence gathered
//     before the widening cannot certify work done after it. Widening the blast
//     radius always costs a re-validation; that is the price of the honesty.
//   - Recorded as an append-only BoundaryExtension with a mandatory reason.
//
// It is NOT a general plan editor. Phases, acceptance, and validation strategy
// stay immutable during execution — those still require a candidate revision
// against a terminal execution.
func (s *service) ExtendBoundary(ctx context.Context, executionID string, req BoundaryExtensionRequest) (Execution, PhaseContext, GuidedStep, []string, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, err
	}
	if e.EffectiveLifecycleState() != ExecutionLifecycleActive {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, ErrInvalidExecution{Reason: "boundary extension requires an active execution"}
	}
	if len(req.Paths) == 0 {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, ErrInvalidExecution{Reason: "at least one path glob is required"}
	}
	if strings.TrimSpace(req.Reason) == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, ErrInvalidExecution{Reason: "boundary extension reason is required (state which phase intent needs the edit and why the authored boundary did not cover it)"}
	}
	oldAllow := append([]string(nil), plan.ChangeBoundary.AcceptanceAllow...)
	updated, added, err := s.plans.ExtendChangeBoundary(ctx, plan.ID, plan.WorkspaceID, plan.WorkspaceRoot, req.Paths)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, err
	}
	if len(added) == 0 {
		// Every glob was already inside the boundary. This is a no-op, not an
		// error: the agent was right to check, and the edit needs no widening.
		pctx := s.buildContext(ctx, updated, e.CurrentPhaseID, e.ID, contextModeStatus)
		s.applyFreshenContext(&pctx, e)
		return e, pctx, stepForContext(e.ID, updated.ID, pctx, e.Complete), added, nil
	}
	phaseID := strings.TrimSpace(e.CurrentPhaseID)
	now := s.now()
	if phaseID != "" {
		if e.PhaseValidationGenerations == nil {
			e.PhaseValidationGenerations = map[string]int{}
		}
		e.PhaseValidationGenerations[phaseID]++
	}
	e.BoundaryExtensions = append(e.BoundaryExtensions, BoundaryExtension{
		ID:            uuid.NewString(),
		PhaseID:       phaseID,
		Author:        strings.TrimSpace(req.Author),
		Reason:        strings.TrimSpace(req.Reason),
		AddedAllow:    added,
		OldAllow:      oldAllow,
		NewAllow:      append([]string(nil), updated.ChangeBoundary.AcceptanceAllow...),
		InvalidatedAt: now,
		CreatedAt:     now,
	})
	e.UpdatedAt = now
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, nil, err
	}
	pctx := s.buildContext(ctx, updated, e.CurrentPhaseID, e.ID, contextModeStatus)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, updated.ID, pctx, e.Complete), added, nil
}

// AdoptBaseline makes legacy handling explicit. It cannot infer whether edits
// already happened: callers either create a fresh producer ticket before more
// work or record a degraded reason that blocks normal completion.
func (s *service) AdoptBaseline(ctx context.Context, executionID string, req BaselineAdoptionRequest) (Execution, PhaseContext, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if e.BaselineSet.Name != "" && !e.BaselineSet.LegacyAdoptionRequired && req.Mode != BaselineAdoptionRecapture {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "execution already has a baseline ticket; use baseline-sync or record a partial handoff"}
	}
	if strings.TrimSpace(req.Reason) == "" {
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "legacy adoption reason is required"}
	}
	switch req.Mode {
	case BaselineAdoptionDegraded:
		e.DegradedReason, e.UpdatedAt = "legacy execution degraded: "+strings.TrimSpace(req.Reason), s.now()
		e.BaselineSet = BaselineSetState{Version: BaselineSetStateSchemaVersion, Status: BaselineSetStatusDegraded, Detail: e.DegradedReason}
	case BaselineAdoptionRecapture:
		name := strings.TrimSpace(req.Name)
		if name == "" {
			name = strings.TrimSpace(plan.Slug) + "-baseline"
		}
		members := uniqueStrings(req.Members)
		if name == "-baseline" || len(members) == 0 {
			return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "recapture adoption requires collection name and at least one member"}
		}
		priorName := strings.TrimSpace(e.BaselineSet.Name)
		if priorName != "" && name == priorName {
			return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "recapture adoption must use a new collection name to preserve the prior ticket"}
		}
		detail := "legacy adoption recapture: " + strings.TrimSpace(req.Reason)
		if priorName != "" {
			detail = "recapture supersedes " + priorName + ": " + strings.TrimSpace(req.Reason)
		}
		e.BaselineSet = baselineTicket(e.ID, name, members, uniqueStrings(req.RepoPaths), detail)
		e.FreshenStatus, e.FreshenDetail, e.UpdatedAt = "baseline_required", e.BaselineSet.Detail, s.now()
	default:
		return Execution{}, PhaseContext{}, GuidedStep{}, ErrInvalidExecution{Reason: "baseline adoption mode must be recapture or degraded"}
	}
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, contextModeStatus)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForContext(e.ID, plan.ID, pctx, e.Complete), nil
}

func baselineTicket(executionID, name string, scenarios, paths []string, detail string) BaselineSetState {
	capture := []string{"git-control-tower", "baseline", "collection", "capture", "--name", name}
	for _, scenario := range scenarios {
		capture = append(capture, "--member", scenario)
	}
	for _, path := range paths {
		capture = append(capture, "--path", path)
	}
	return BaselineSetState{Version: BaselineSetStateSchemaVersion, Name: name, ScenarioTargets: scenarios, RepoPaths: paths, Status: BaselineSetStatusRequired, Detail: detail, CaptureArgv: capture, WaitArgv: []string{"git-control-tower", "baseline", "collection", "show", "--name", name, "--wait", "--json"}, SyncArgv: []string{"plan-manager", "exec", "baseline-sync", executionID}}
}

func phaseMinimum(plan planmodel.Plan, phaseID string) []string {
	phase, ok := findPhase(plan.Phases, phaseID)
	if !ok {
		return nil
	}
	boundary := plan.ChangeBoundary
	if !phase.ChangeBoundary.IsZero() {
		boundary = phase.ChangeBoundary
	}
	return uniqueStrings(boundary.AffectedScenarios())
}

func uniqueStrings(values []string) []string {
	seen, out := map[string]struct{}{}, make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			if _, ok := seen[value]; !ok {
				seen[value] = struct{}{}
				out = append(out, value)
			}
		}
	}
	return out
}

func requireBaselineReady(state BaselineSetState) error {
	if state.LegacyAdoptionRequired {
		return ErrInvalidExecution{Reason: "historical plan requires plan-manager exec baseline-adopt before normal phase work"}
	}
	if state.Name == "" {
		// Historical executions are readable until the explicit adoption/degraded
		// workflow lands. New executions always carry a ticket and therefore take
		// the strict branch below.
		return nil
	}
	if state.Complete() {
		return nil
	}
	return ErrInvalidExecution{Reason: "baseline collection " + state.Name + " is " + string(state.Status) + "; run its producer action/wait and plan-manager exec baseline-sync before normal phase work"}
}

// applyFreshenContext surfaces the recorded freshen status into the phase context
// so the agent sees whether the "before" anchor was captured fresh (or why not).
func (s *service) applyFreshenContext(pctx *PhaseContext, e Execution) {
	pctx.InputsFreshened = e.InputsFreshenedAt != ""
	pctx.FreshenStatus = e.FreshenStatus
	pctx.FreshenDetail = e.FreshenDetail
	pctx.BaselineSet = e.BaselineSet
	pctx.ScopeGeneration = e.PhaseValidationGenerations[pctx.CurrentPhase.ID]
	pctx.ValidationMembers = amendedPhaseMembers(e, pctx.CurrentPhase.ID)
}

// nonEmpty returns the non-blank values among its args, preserving order.
func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, strings.TrimSpace(v))
		}
	}
	return out
}

func (s *service) getExecution(ctx context.Context, executionID string) (Execution, error) {
	e, ok, err := s.repo.GetExecution(ctx, strings.TrimSpace(executionID))
	if err != nil {
		return Execution{}, err
	}
	if !ok {
		return Execution{}, ErrExecutionNotFound{ID: executionID}
	}
	return e, nil
}

type contextMode string

const (
	contextModeStart      contextMode = "start"
	contextModeStatus     contextMode = "status"
	contextModeContext    contextMode = "context"
	contextModeResume     contextMode = "resume"
	contextModePhaseEntry contextMode = "phase_entry"
)

// buildContext assembles the just-in-time PhaseContext for the named phase.
func (s *service) buildContext(ctx context.Context, plan planmodel.Plan, phaseID, executionID string, mode contextMode) PhaseContext {
	pctx := PhaseContext{
		ResumePhaseID: resumePhaseID(plan.Phases),
		Completeness:  computeCompleteness(plan.Phases),
		LogSummary:    s.logSummary(ctx, executionID),
	}
	pctx.ChangeBoundary = plan.ChangeBoundary
	if cur, ok := findPhase(plan.Phases, phaseID); ok {
		pctx.CurrentPhase = cur
		pctx.HasCurrent = true
		pctx.RequiredReading = cur.RequiredReading
		pctx.Reminders = cur.Reminders
		// A phase that declares its own boundary narrows the surfaced contract.
		if !cur.ChangeBoundary.IsZero() {
			pctx.ChangeBoundary = cur.ChangeBoundary
		}
		pctx.FeedbackCheckpoint = s.feedbackCheckpoint(ctx, executionID, cur.ID)
	}
	pctx.RelevantContext = contextForPhase(plan, pctx.CurrentPhase, pctx.HasCurrent, mode)
	if next, ok := nextPhase(plan.Phases, phaseID); ok {
		pctx.NextPhase = next
		pctx.HasNext = true
	}
	lastVal, hasVal, staleness := s.lastValidation(ctx, plan, phaseID)
	pctx.LastValidation = lastVal
	pctx.HasValidation = hasVal
	pctx.Staleness = staleness
	return pctx
}

// logLedger reads the execution's compact log summary + captured entries through
// the LogLedger seam. Degrades to an empty summary when the seam is nil or
// errors (never a fabricated count).
func (s *service) logLedger(ctx context.Context, executionID string) (planmodel.LogSummary, []planmodel.LogEntry) {
	if s.log == nil {
		return planmodel.LogSummary{}, nil
	}
	summary, entries, err := s.log.Summarize(ctx, executionID)
	if err != nil {
		return planmodel.LogSummary{}, nil
	}
	return summary, entries
}

func (s *service) phaseLogLedger(ctx context.Context, executionID, phaseID string) (planmodel.LogSummary, []planmodel.LogEntry) {
	if s.log == nil {
		return planmodel.LogSummary{}, nil
	}
	summary, entries, err := s.log.SummarizePhase(ctx, executionID, phaseID)
	if err != nil {
		return planmodel.LogSummary{}, nil
	}
	return summary, entries
}

// logSummary reads only the compact summary for the just-in-time context.
func (s *service) logSummary(ctx context.Context, executionID string) planmodel.LogSummary {
	summary, _ := s.logLedger(ctx, executionID)
	return summary
}

func (s *service) feedbackCheckpoint(ctx context.Context, executionID, phaseID string) PhaseFeedbackCheckpoint {
	summary, entries := s.phaseLogLedger(ctx, executionID, phaseID)
	cp := PhaseFeedbackCheckpoint{
		PhaseID:          phaseID,
		Decisions:        summary.Decisions,
		Findings:         summary.Findings,
		BugReports:       summary.BugReports,
		Records:          summary.Records,
		Notes:            summary.Notes,
		PendingSync:      summary.PendingSync,
		FailedSync:       summary.FailedSync,
		NoFeedbackTitle:  NoFeedbackCheckpointTitle,
		NoFeedbackDetail: noFeedbackCheckpointDetail,
	}
	explicitNone := false
	for _, entry := range entries {
		if entry.Type == planmodel.LogEntryNote && strings.EqualFold(strings.TrimSpace(entry.Title), NoFeedbackCheckpointTitle) {
			explicitNone = true
			break
		}
	}
	capturedWork := summary.Decisions + summary.Findings + summary.BugReports + summary.Records
	cp.Reviewed = explicitNone || capturedWork > 0
	cp.Satisfied = cp.Reviewed && summary.FailedSync == 0 && summary.PendingSync == 0
	switch {
	case !cp.Reviewed:
		cp.Summary = "Review phase feedback before marking done: capture decisions/findings/bugs/records, or record an explicit no-feedback note."
	case summary.FailedSync > 0:
		cp.Summary = "Feedback was reviewed, but some bug/record forwarding failed; retry sync before marking done."
	case summary.PendingSync > 0:
		cp.Summary = "Feedback was reviewed, but some bug/record forwarding is pending; retry or acknowledge before marking done."
	case explicitNone:
		cp.Summary = "Feedback reviewed: no durable work products needed for this phase."
	default:
		cp.Summary = "Feedback reviewed with durable phase log entries."
	}
	return cp
}

func (s *service) requireFeedbackForDone(ctx context.Context, executionID, phaseID, overrideReason string) error {
	if strings.TrimSpace(overrideReason) != "" {
		return nil
	}
	cp := s.feedbackCheckpoint(ctx, executionID, phaseID)
	if cp.Satisfied {
		return nil
	}
	return ErrInvalidExecution{Reason: "phase feedback checkpoint is required before marking done; capture phase feedback or record a no-feedback note, or provide feedback override reason"}
}

func contextForPhase(plan planmodel.Plan, cur planmodel.Phase, hasCurrent bool, mode contextMode) []planmodel.RelevantContextItem {
	out := make([]planmodel.RelevantContextItem, 0, len(plan.RelevantContext)+len(cur.RelevantContext)+len(cur.RequiredReading))
	for _, item := range plan.RelevantContext {
		if item.Scope == "" {
			item.Scope = planmodel.RelevantContextScopeGlobal
		}
		if !includeGlobalContext(item, mode) {
			continue
		}
		out = append(out, item)
	}
	if !hasCurrent {
		return out
	}
	for _, item := range cur.RelevantContext {
		if item.Scope == "" {
			item.Scope = planmodel.RelevantContextScopePhase
		}
		if item.PhaseID == "" {
			item.PhaseID = cur.ID
		}
		if !includePhaseContext(item, mode) {
			continue
		}
		out = append(out, item)
	}
	if len(cur.RelevantContext) == 0 {
		out = append(out, legacyRequiredReadingContext(cur)...)
	}
	return out
}

func includeGlobalContext(item planmodel.RelevantContextItem, mode contextMode) bool {
	switch item.RepeatPolicy {
	case planmodel.RelevantContextOncePerExecution:
		return mode == contextModeStart
	case planmodel.RelevantContextOnResume:
		return mode == contextModeResume || mode == contextModeContext || mode == contextModeStatus
	case planmodel.RelevantContextEveryPhase, "":
		return true
	case planmodel.RelevantContextPhaseEntry:
		return mode == contextModeStart || mode == contextModeResume || mode == contextModePhaseEntry
	case planmodel.RelevantContextAsNeeded:
		return true
	default:
		return true
	}
}

func includePhaseContext(item planmodel.RelevantContextItem, mode contextMode) bool {
	switch item.RepeatPolicy {
	case planmodel.RelevantContextOncePerExecution:
		return mode == contextModeStart
	case planmodel.RelevantContextOnResume:
		return mode == contextModeResume || mode == contextModeContext || mode == contextModeStatus
	case planmodel.RelevantContextPhaseEntry, "":
		return true
	case planmodel.RelevantContextEveryPhase:
		return true
	case planmodel.RelevantContextAsNeeded:
		return true
	default:
		return true
	}
}

func legacyRequiredReadingContext(ph planmodel.Phase) []planmodel.RelevantContextItem {
	items := make([]planmodel.RelevantContextItem, 0, len(ph.RequiredReading))
	for _, raw := range ph.RequiredReading {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		items = append(items, planmodel.RelevantContextItem{
			Kind:         legacyContextKind(raw),
			Scope:        planmodel.RelevantContextScopePhase,
			PhaseID:      ph.ID,
			Label:        raw,
			Instruction:  raw,
			Target:       raw,
			Required:     true,
			RepeatPolicy: planmodel.RelevantContextPhaseEntry,
			Source:       planmodel.RelevantContextSourceMigrated,
			Status:       planmodel.RelevantContextStatusReady,
		})
	}
	return items
}

func legacyContextKind(raw string) planmodel.RelevantContextKind {
	switch {
	case strings.HasPrefix(raw, "prompt-manager skill read"):
		return planmodel.RelevantContextSkill
	case strings.HasPrefix(raw, "search-hub query"):
		return planmodel.RelevantContextSearch
	case strings.HasPrefix(raw, "cli:"):
		return planmodel.RelevantContextCommand
	case strings.Contains(raw, ".md") || strings.HasPrefix(raw, "docs/"):
		return planmodel.RelevantContextDoc
	default:
		return planmodel.RelevantContextNote
	}
}

// lastValidation reads the validation seam for the phase's LAST STORED validation
// result + the staleness captured with it. This is a cheap store read — it never
// shells a subprocess — so status/next stay cheap. Degrades to UNKNOWN/absent
// (never a false pass) when the validator is nil, errors, or has no result yet.
func (s *service) lastValidation(ctx context.Context, plan planmodel.Plan, phaseID string) (ValidationResult, bool, planmodel.StalenessTier) {
	if s.validator == nil {
		return ValidationResult{}, false, planmodel.StalenessUnknown
	}
	res, ok, err := s.validator.LastValidation(ctx, plan.ID, phaseID)
	if err != nil || !ok {
		return ValidationResult{}, false, planmodel.StalenessUnknown
	}
	staleness := res.Staleness
	if staleness == "" {
		staleness = planmodel.StalenessUnknown
	}
	return res, true, staleness
}

func (s *service) requireValidationForDone(ctx context.Context, e Execution, plan planmodel.Plan, phaseID, overrideReason string) error {
	res, hasVal, staleness := s.lastValidation(ctx, plan, phaseID)
	// Legacy executions remain readable and can finish their already-authored
	// phase workflow. New collection-ticketed executions require execution-bound
	// provenance and scenario coverage — but a PHASE transition is satisfied by a
	// fresh phase-scoped pass whose scope covers the phase. Full-inventory is a
	// plan-COMPLETION concern, so it is intentionally NOT demanded here.
	var reason string
	if e.BaselineSet.Name != "" {
		reason = validationGateFailure(res, hasVal, staleness, e.ID, e.PhaseValidationGenerations[phaseID], false, effectivePhaseMinimum(e, plan, phaseID))
	} else {
		reason = validationBlockerReason(res, hasVal, staleness)
	}
	if reason == "" {
		return nil
	}
	if strings.TrimSpace(overrideReason) != "" {
		return nil
	}
	return ErrValidationRequired{PhaseID: phaseID, Reason: reason}
}

// effectivePhaseMinimum is the actual execution-owned selector for a phase.
// Plan policy supplies the initial minimum; each append-only amendment replaces
// it with a previously audited superset. This is intentionally calculated from
// the execution record, never inferred from a ticket supplied by the caller.
func effectivePhaseMinimum(e Execution, plan planmodel.Plan, phaseID string) []string {
	minimum := normalizedMembers(phaseMinimum(plan, phaseID))
	for _, amendment := range e.ScopeAmendments {
		if amendment.PhaseID == phaseID && len(amendment.NewMinimum) > 0 {
			minimum = normalizedMembers(amendment.NewMinimum)
		}
	}
	return minimum
}

// amendedPhaseMembers returns the last audited selector only when it differs
// from authored policy. An empty result lets ticket creation retain its normal
// plan-derived default scope.
func amendedPhaseMembers(e Execution, phaseID string) []string {
	for i := len(e.ScopeAmendments) - 1; i >= 0; i-- {
		if e.ScopeAmendments[i].PhaseID == phaseID {
			return normalizedMembers(e.ScopeAmendments[i].NewMinimum)
		}
	}
	return nil
}

func normalizedMembers(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; !exists {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func containsMembers(selected, required []string) bool {
	set := make(map[string]struct{}, len(selected))
	for _, member := range selected {
		set[member] = struct{}{}
	}
	for _, member := range required {
		if _, exists := set[member]; !exists {
			return false
		}
	}
	return true
}

func (s *service) requireFinalValidation(ctx context.Context, e Execution, plan planmodel.Plan) error {
	res, hasVal, staleness := s.lastValidation(ctx, plan, "")
	// Completion legitimately demands a fresh, execution-bound, full-inventory
	// pass — full-inventory is a plan-completion concern (never a per-phase one).
	reason := validationGateFailure(res, hasVal, staleness, e.ID, e.PhaseValidationGenerations[""], true, nil)
	if reason == "" {
		return nil
	}
	return ErrInvalidExecution{Reason: "full collection Definition-of-Done validation is required before completion: " + reason}
}

func validationIsRecentPass(res ValidationResult, hasVal bool, staleness planmodel.StalenessTier) bool {
	return hasVal && res.Verdict == "pass" && staleness == planmodel.StalenessFresh
}

// validationGateFailure returns an empty string when the stored validation
// satisfies the execution-bound done/completion gate, otherwise a precise reason
// naming the FIRST failing condition. The gate predicate checks more than
// verdict+staleness (execution binding, scope generation, full-inventory,
// scenario coverage); reporting only the staleness here produced the
// contradictory "staleness is fresh" rejection (knw-1784053356805823492) when the
// real blocker was one of the provenance conditions. Every rejection message must
// name the condition that actually failed, never one that passed.
func validationGateFailure(res ValidationResult, hasVal bool, staleness planmodel.StalenessTier, executionID string, generation int, requireFullInventory bool, requiredMembers []string) string {
	if reason := validationBlockerReason(res, hasVal, staleness); reason != "" {
		return reason
	}
	if res.ExecutionID != executionID {
		return "last validation was bound to a different execution (" + orUnknown(res.ExecutionID) + "); run validation for this execution"
	}
	if res.ScopeGeneration != generation {
		return fmt.Sprintf("last validation covered scope generation %d but this phase is at generation %d; re-run validation after the scope change", res.ScopeGeneration, generation)
	}
	if requireFullInventory && !res.FullInventory {
		return "last validation was phase-scoped, but full-inventory Definition-of-Done validation is required for completion"
	}
	if len(requiredMembers) > 0 && !containsMembers(res.SelectedMembers, requiredMembers) {
		return "last validation did not cover required scenario(s): " + strings.Join(missingMembers(res.SelectedMembers, requiredMembers), ", ")
	}
	return ""
}

// validationBlockerReason reports why the LAST STORED validation is not a fresh
// pass. It returns "" exactly when validationIsRecentPass is true, so a fresh
// passing validation never yields a (self-contradictory) blocker message.
func validationBlockerReason(res ValidationResult, hasVal bool, staleness planmodel.StalenessTier) string {
	if !hasVal {
		return "no stored validation result"
	}
	if res.Verdict != "pass" {
		return "last validation verdict is " + orUnknown(res.Verdict)
	}
	if staleness != planmodel.StalenessFresh {
		return "last validation staleness is " + orUnknown(string(staleness))
	}
	return ""
}

// missingMembers lists required members not present in selected, preserving the
// required order so the rejection message is deterministic.
func missingMembers(selected, required []string) []string {
	set := make(map[string]struct{}, len(selected))
	for _, member := range selected {
		set[member] = struct{}{}
	}
	var missing []string
	for _, member := range required {
		if _, ok := set[member]; !ok {
			missing = append(missing, member)
		}
	}
	return missing
}

func orUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

// assembleLiveHandoff builds a handoff view from current captured state without
// persisting it — used by GetHandoff before Complete has run.
func (s *service) assembleLiveHandoff(ctx context.Context, executionID string) (Handoff, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Handoff{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Handoff{}, err
	}
	logSummary, logEntries := s.logLedger(ctx, e.ID)
	lastVal, hasVal, staleness := s.lastValidation(ctx, plan, e.CurrentPhaseID)
	return Handoff{
		ExecutionID:    e.ID,
		PlanID:         plan.ID,
		Completeness:   computeCompleteness(plan.Phases),
		ResumePhaseID:  resumePhaseID(plan.Phases),
		LogSummary:     logSummary,
		LogEntries:     logEntries,
		LastValidation: lastVal,
		HasValidation:  hasVal,
		Staleness:      staleness,
		AssembledAt:    s.now(),
	}, nil
}

// completionNudges builds the thin guided-completion checklist. Each nudge is
// typed and Plan Manager-local, and satisfied=true when the log ledger already
// covers it. Missing bug reports only matter if defects were encountered;
// missing records only matter for non-trivial reusable work — so those nudges
// are conditional on findings present rather than always-on checklist items.
func (s *service) completionNudges(plan planmodel.Plan, summary planmodel.LogSummary) []CompletionNudge {
	allTerminal := computeCompleteness(plan.Phases) == CompletenessFull
	for _, ph := range plan.Phases {
		st := ph.Status
		if st == planmodel.PhaseStatusDone || st == planmodel.PhaseStatusBlocked {
			continue
		}
		allTerminal = false
		break
	}
	hasFindings := summary.Findings > 0
	return []CompletionNudge{
		{
			Kind:      "record_finding",
			Message:   "Record any in-flight findings (possible bugs) with `plan-manager log finding-add` before handing off.",
			Satisfied: summary.Findings > 0 || summary.Decisions > 0,
		},
		{
			Kind: "file_bug",
			// Only matters if a defect was encountered: a candidate finding that
			// looks like a real bug should be filed with `plan-manager log bug-add`
			// (or promoted from the finding) — Plan Manager forwards it downstream.
			Message:   "If you confirmed a defect, file it with `plan-manager log bug-add` or `plan-manager log promote <finding>` — Plan Manager forwards it for you.",
			Satisfied: !hasFindings || summary.BugReports > 0,
		},
		{
			Kind: "capture_record",
			// Only matters for non-trivial reusable learning/completed work.
			Message:   "Capture reusable learning or completed work with `plan-manager log record-add` so the learning loop benefits.",
			Satisfied: summary.Records > 0 || !allTerminal,
		},
		{
			Kind:      "confirm_phase_status",
			Message:   "Confirm every phase's status is terminal (done/blocked) so the resume point and completeness are accurate.",
			Satisfied: allTerminal,
		},
	}
}

func (s *service) now() string { return s.clock.Now().UTC().Format(execTimeFormat) }

// wallTime computes the wall-clock seconds between startedAt and now, parsing
// the RFC3339Nano timestamps. A parse failure yields 0 (honest, never negative).
func (s *service) wallTime(startedAt, now string) int64 {
	start, err := time.Parse(execTimeFormat, startedAt)
	if err != nil {
		return 0
	}
	end, err := time.Parse(execTimeFormat, now)
	if err != nil {
		return 0
	}
	d := end.Sub(start)
	if d < 0 {
		return 0
	}
	return int64(d.Seconds())
}

// resumePhaseID returns the earliest non-done phase id (the resume point), or ""
// when every phase is done. "Earliest" is by the authored phase order. A blocked
// phase is non-done and so is a candidate resume point.
func resumePhaseID(phases []planmodel.Phase) string {
	for _, ph := range phases {
		if ph.Status != planmodel.PhaseStatusDone {
			return ph.ID
		}
	}
	return ""
}

func resolveExecutionPhaseID(plan planmodel.Plan, phaseID string) (string, error) {
	phaseID = strings.TrimSpace(phaseID)
	if phaseID == "" {
		return resumePhaseID(plan.Phases), nil
	}
	if _, ok := findPhase(plan.Phases, phaseID); !ok {
		return "", planmodel.ErrPhaseNotFound{PlanID: plan.ID, PhaseID: phaseID}
	}
	return phaseID, nil
}

// computeCompleteness is FULL iff every phase is done; PARTIAL otherwise. An
// empty plan (no phases) is PARTIAL — there is nothing proven done.
func computeCompleteness(phases []planmodel.Phase) Completeness {
	if len(phases) == 0 {
		return CompletenessPartial
	}
	for _, ph := range phases {
		if ph.Status != planmodel.PhaseStatusDone {
			return CompletenessPartial
		}
	}
	return CompletenessFull
}

// findPhase returns the phase with the given id.
func findPhase(phases []planmodel.Phase, phaseID string) (planmodel.Phase, bool) {
	for _, ph := range phases {
		if ph.ID == phaseID {
			return ph, true
		}
	}
	return planmodel.Phase{}, false
}

// nextPhase returns the phase immediately after phaseID in authored order.
func nextPhase(phases []planmodel.Phase, phaseID string) (planmodel.Phase, bool) {
	for i, ph := range phases {
		if ph.ID == phaseID && i+1 < len(phases) {
			return phases[i+1], true
		}
	}
	return planmodel.Phase{}, false
}

func nextActionablePhaseID(phases []planmodel.Phase, currentID string) string {
	if strings.TrimSpace(currentID) == "" {
		return resumePhaseID(phases)
	}
	for i, ph := range phases {
		if ph.ID != currentID {
			continue
		}
		for _, candidate := range phases[i+1:] {
			if candidate.Status != planmodel.PhaseStatusDone {
				return candidate.ID
			}
		}
		return ""
	}
	return resumePhaseID(phases)
}
