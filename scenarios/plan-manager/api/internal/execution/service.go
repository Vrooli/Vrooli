package execution

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

// runIDEnv is the orchestration-layer attribution key. Start falls back to it
// when the caller does not supply a run id (best-effort when absent).
const runIDEnv = "VROOLI_AGENT_MANAGER_RUN_ID"

// NoFeedbackCheckpointTitle is the durable note title that explicitly records a
// phase feedback review found nothing to capture.
const NoFeedbackCheckpointTitle = "Phase feedback reviewed: none"

const noFeedbackCheckpointDetail = "No decisions, findings, bugs, records, or reusable notes to capture for this phase."

// Service is the execution application surface — the guided runner.
type Service interface {
	Start(ctx context.Context, planID, runID string) (Execution, PhaseContext, GuidedStep, error)
	GetStatus(ctx context.Context, executionID string) (Execution, PhaseContext, GuidedStep, error)
	GetContext(ctx context.Context, executionID, phaseID string) (Execution, PhaseContext, GuidedStep, error)
	Resume(ctx context.Context, planOrExecution, phaseID, runID string) (Execution, PhaseContext, GuidedStep, error)
	ContinueExecution(ctx context.Context, planOrExecution, phaseID, runID string) (Execution, PhaseContext, GuidedStep, error)
	GetNext(ctx context.Context, executionID string) (PhaseContext, bool, GuidedStep, error)
	TransitionPhase(ctx context.Context, executionID, phaseID string, inputs PhaseTransitionInputs) (Execution, planmodel.Plan, GuidedStep, error)

	Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error)
	GetHandoff(ctx context.Context, executionID string) (Handoff, GuidedStep, error)

	GetVelocity(ctx context.Context, planID string) ([]VelocityPoint, GuidedStep, error)
}

type service struct {
	repo      Repository
	plans     PlanStore
	validator Validator
	log       LogLedger
	velocity  VelocitySink
	freshener InputFreshener
	clock     clock.Clock
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
	Freshener InputFreshener
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
		freshener: d.Freshener,
		clock:     clk,
	}
}

var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context, planID, runID string) (Execution, PhaseContext, GuidedStep, error) {
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
	if err := requireExecutionGradePlan(plan); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	return s.startAtPhase(ctx, plan, "", runID, contextModeStart)
}

func (s *service) startAtPhase(ctx context.Context, plan planmodel.Plan, phaseID, runID string, mode contextMode) (Execution, PhaseContext, GuidedStep, error) {
	if runID == "" {
		runID = strings.TrimSpace(os.Getenv(runIDEnv))
	}
	currentPhaseID, err := resolveExecutionPhaseID(plan, phaseID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	now := s.now()
	e := Execution{
		ID:             uuid.NewString(),
		PlanID:         plan.ID,
		RunID:          runID,
		CurrentPhaseID: currentPhaseID,
		StartedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	s.freshenInputs(ctx, &e, plan)
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID, e.ID, mode)
	s.applyFreshenContext(&pctx, e)
	return e, pctx, stepForStarted(e), nil
}

func (s *service) resumeExecution(ctx context.Context, e Execution, phaseID string) (Execution, PhaseContext, GuidedStep, error) {
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if err := requireExecutionGradePlan(plan); err != nil {
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
	s.freshenInputs(ctx, &e, plan)
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
	if err := requireExecutionGradePlan(plan); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	}
	if e, ok, err := s.repo.LatestExecutionForPlan(ctx, plan.ID); err != nil {
		return Execution{}, PhaseContext{}, GuidedStep{}, err
	} else if ok {
		return s.resumeExecutionWithPlan(ctx, e, plan, phaseID)
	}
	// No prior execution exists for this plan: resume/continue is creating a NEW
	// run, so this is a first start — emit once-per-execution setup context exactly
	// once (contextModeStart), not the resume context mode. This is the fix for
	// once-per-execution context being skipped on the continue/resume create path.
	return s.startAtPhase(ctx, plan, strings.TrimSpace(phaseID), runID, contextModeStart)
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
	if to == planmodel.PhaseStatusDone {
		if err := s.requireValidationForDone(ctx, plan, target.ID, inputs.ValidationOverrideReason); err != nil {
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
		RunID:           e.RunID,
		WallTimeSeconds: s.wallTime(e.StartedAt, now),
		Tokens:          inputs.Tokens,
		Iterations:      inputs.Iterations,
		Completeness:    completeness,
		RecordedAt:      now,
	}
	e.Complete = true
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
	return points, GuidedStep{}, err
}

// --- helpers ---

func requireExecutionGradePlan(plan planmodel.Plan) error {
	report := planmodel.AssessPlanQuality(plan, "")
	if report.ExecutionReady() {
		return nil
	}
	return ErrInvalidExecution{Reason: "plan is not execution-grade; repair before starting execution: " + summarizeQualityFailures(report)}
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
func (s *service) freshenInputs(ctx context.Context, e *Execution, plan planmodel.Plan) {
	if e.FreshenStatus == FreshenStatusCaptured {
		return // already captured; never re-run (the "before" is pinned)
	}
	if s.freshener == nil {
		return // not configured; skip silently (no fabricated capture)
	}
	now := s.now()
	res, err := s.freshener.FreshenInputs(ctx, plan.ID)
	e.InputsFreshenedAt = now
	if err != nil {
		e.FreshenStatus = FreshenStatusDegraded
		e.FreshenDetail = err.Error()
	} else if res.BaselineCaptured {
		e.FreshenStatus = FreshenStatusCaptured
		e.FreshenDetail = strings.TrimSpace(strings.Join(nonEmpty(res.Detail, res.StalenessSummary), "; "))
	} else {
		e.FreshenStatus = FreshenStatusDegraded
		e.FreshenDetail = strings.TrimSpace(strings.Join(nonEmpty(res.Detail, res.StalenessSummary), "; "))
	}
	if res.BaselineSet.Name != "" {
		state := res.BaselineSet
		state.CapturedAt = now
		state.Detail = e.FreshenDetail
		e.BaselineSet = state
	}
	// Best-effort persist of the freshen marker — a write failure must not block
	// the start/resume the agent asked for (the freshen retries next resume).
	_ = s.repo.SaveExecution(ctx, *e)
}

// applyFreshenContext surfaces the recorded freshen status into the phase context
// so the agent sees whether the "before" anchor was captured fresh (or why not).
func (s *service) applyFreshenContext(pctx *PhaseContext, e Execution) {
	pctx.InputsFreshened = e.InputsFreshenedAt != ""
	pctx.FreshenStatus = e.FreshenStatus
	pctx.FreshenDetail = e.FreshenDetail
	pctx.BaselineSet = e.BaselineSet
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

func (s *service) requireValidationForDone(ctx context.Context, plan planmodel.Plan, phaseID, overrideReason string) error {
	res, hasVal, staleness := s.lastValidation(ctx, plan, phaseID)
	if validationIsRecentPass(res, hasVal, staleness) {
		return nil
	}
	if strings.TrimSpace(overrideReason) != "" {
		return nil
	}
	return ErrValidationRequired{PhaseID: phaseID, Reason: validationBlockerReason(res, hasVal, staleness)}
}

func validationIsRecentPass(res ValidationResult, hasVal bool, staleness planmodel.StalenessTier) bool {
	return hasVal && res.Verdict == "pass" && staleness == planmodel.StalenessFresh
}

func validationBlockerReason(res ValidationResult, hasVal bool, staleness planmodel.StalenessTier) string {
	if !hasVal {
		return "no stored validation result"
	}
	if res.Verdict != "pass" {
		return "last validation verdict is " + orUnknown(res.Verdict)
	}
	return "last validation staleness is " + orUnknown(string(staleness))
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
