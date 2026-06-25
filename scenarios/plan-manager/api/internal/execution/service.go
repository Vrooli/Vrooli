package execution

import (
	"context"
	"os"
	"strings"
	"time"

	"plan-manager/internal/clock"
	internalplans "plan-manager/internal/plans"

	"github.com/google/uuid"
)

// runIDEnv is the orchestration-layer attribution key. Start falls back to it
// when the caller does not supply a run id (best-effort when absent).
const runIDEnv = "VROOLI_AGENT_MANAGER_RUN_ID"

// Service is the execution application surface — the guided runner.
type Service interface {
	Start(ctx context.Context, planID, runID string) (Execution, error)
	GetStatus(ctx context.Context, executionID string) (Execution, PhaseContext, error)
	GetNext(ctx context.Context, executionID string) (PhaseContext, bool, error)
	TransitionPhase(ctx context.Context, executionID, phaseID string, to internalplans.PhaseStatus) (Execution, internalplans.Plan, error)

	RecordDecision(ctx context.Context, executionID, phaseID, summary, detail string) (Decision, error)
	RecordFinding(ctx context.Context, executionID, phaseID, title, detail string) (Finding, error)

	Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, error)
	GetHandoff(ctx context.Context, executionID string) (Handoff, error)

	ListCandidateFindings(ctx context.Context, executionID string) ([]Finding, error)
	TriageFinding(ctx context.Context, findingID string, triage FindingTriage) (Finding, error)

	GetVelocity(ctx context.Context, planID string) ([]VelocityPoint, error)
}

type service struct {
	repo      Repository
	plans     PlanStore
	validator Validator
	velocity  VelocitySink
	clock     clock.Clock
}

// Deps wires the execution Service. Repo + Plans are required; Validator is
// optional (nil => last_validation/staleness degrade to UNKNOWN, never a false
// pass). VelocitySink is optional (nil => the no-op default; velocity is still
// persisted locally regardless).
type Deps struct {
	Repo      Repository
	Plans     PlanStore
	Validator Validator
	Velocity  VelocitySink
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
		velocity:  sink,
		clock:     clk,
	}
}

var _ Service = (*service)(nil)

func (s *service) Start(ctx context.Context, planID, runID string) (Execution, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return Execution{}, ErrInvalidExecution{Reason: "plan id is required"}
	}
	// Resolve the plan (id or slug) through the plans SSOT so the linkage stores
	// the canonical plan id, and so a bad plan fails fast with NotFound.
	plan, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Execution{}, err
	}
	if runID == "" {
		runID = strings.TrimSpace(os.Getenv(runIDEnv))
	}
	now := s.now()
	e := Execution{
		ID:             uuid.NewString(),
		PlanID:         plan.ID,
		RunID:          runID,
		CurrentPhaseID: resumePhaseID(plan.Phases),
		StartedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, err
	}
	return e, nil
}

func (s *service) GetStatus(ctx context.Context, executionID string) (Execution, PhaseContext, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, PhaseContext{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, PhaseContext{}, err
	}
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID)
	return e, pctx, nil
}

func (s *service) GetNext(ctx context.Context, executionID string) (PhaseContext, bool, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return PhaseContext{}, false, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return PhaseContext{}, false, err
	}
	// Advance the runner's pointer to the earliest non-done phase (the resume
	// point). When none remains the run is functionally complete.
	next := resumePhaseID(plan.Phases)
	e.CurrentPhaseID = next
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return PhaseContext{}, false, err
	}
	pctx := s.buildContext(ctx, plan, next)
	return pctx, next == "", nil
}

func (s *service) TransitionPhase(ctx context.Context, executionID, phaseID string, to internalplans.PhaseStatus) (Execution, internalplans.Plan, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Execution{}, internalplans.Plan{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Execution{}, internalplans.Plan{}, err
	}
	target, ok := findPhase(plan.Phases, phaseID)
	if !ok {
		return Execution{}, internalplans.Plan{}, internalplans.ErrPhaseNotFound{PlanID: plan.ID, PhaseID: phaseID}
	}
	// Delegate the phase-status change to the plans domain — it stays the single
	// source of truth for the record (plan status is recomputed there).
	target.Status = to
	updated, err := s.plans.UpdatePhase(ctx, plan.ID, target)
	if err != nil {
		return Execution{}, internalplans.Plan{}, err
	}
	// Move the runner's pointer to the next actionable phase and mark the
	// execution complete when every phase is terminal.
	e.CurrentPhaseID = resumePhaseID(updated.Phases)
	e.Complete = e.CurrentPhaseID == ""
	e.UpdatedAt = s.now()
	if err := s.repo.SaveExecution(ctx, e); err != nil {
		return Execution{}, internalplans.Plan{}, err
	}
	return e, updated, nil
}

func (s *service) RecordDecision(ctx context.Context, executionID, phaseID, summary, detail string) (Decision, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Decision{}, err
	}
	d := Decision{
		ID:          uuid.NewString(),
		ExecutionID: e.ID,
		PhaseID:     phaseID,
		Summary:     summary,
		Detail:      detail,
		RecordedAt:  s.now(),
	}
	if err := s.repo.SaveDecision(ctx, d); err != nil {
		return Decision{}, err
	}
	return d, nil
}

func (s *service) RecordFinding(ctx context.Context, executionID, phaseID, title, detail string) (Finding, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Finding{}, err
	}
	// Attribution-keyed dedup: a finding with the same (attribution_run_id, title)
	// is not double-filed. When the run id is absent, dedup by title within the
	// execution (best-effort). Collect the candidate set first (pool=1 safe — no
	// nested query inside the lookup), then return the existing match if any.
	existing, err := s.repo.ListFindings(ctx, e.ID, "")
	if err != nil {
		return Finding{}, err
	}
	for _, f := range existing {
		if !strings.EqualFold(strings.TrimSpace(f.Title), strings.TrimSpace(title)) {
			continue
		}
		if e.RunID != "" {
			if f.AttributionRunID == e.RunID {
				return f, nil
			}
			continue
		}
		// No run id: dedup by title within the execution.
		return f, nil
	}
	f := Finding{
		ID:               uuid.NewString(),
		ExecutionID:      e.ID,
		PhaseID:          phaseID,
		Title:            title,
		Detail:           detail,
		Triage:           TriageCandidate, // always CANDIDATE; never auto-promoted
		AttributionRunID: e.RunID,
		RecordedAt:       s.now(),
	}
	if err := s.repo.SaveFinding(ctx, f); err != nil {
		// The unique dedup index is the cross-process backstop: if a concurrent
		// process recorded the same (run id, title) between our read above and this
		// write, return that winner instead of failing the agent's record call.
		if isUniqueViolation(err) && e.RunID != "" {
			if winner, ok := s.findByRunAndTitle(ctx, e.ID, e.RunID, title); ok {
				return winner, nil
			}
		}
		return Finding{}, err
	}
	return f, nil
}

// findByRunAndTitle re-reads the execution's findings to return the one matching
// (run id, title) — used to recover the concurrent winner after a unique-index
// race on the dedup key.
func (s *service) findByRunAndTitle(ctx context.Context, executionID, runID, title string) (Finding, bool) {
	existing, err := s.repo.ListFindings(ctx, executionID, "")
	if err != nil {
		return Finding{}, false
	}
	for _, f := range existing {
		if f.AttributionRunID == runID && strings.EqualFold(strings.TrimSpace(f.Title), strings.TrimSpace(title)) {
			return f, true
		}
	}
	return Finding{}, false
}

// isUniqueViolation reports whether err is a SQLite UNIQUE-constraint failure
// (the modernc.org/sqlite driver surfaces it in the error text). Matched
// defensively by substring so a driver-version message tweak does not silently
// turn the dedup race back into a hard error.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "constraint failed")
}

func (s *service) Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Handoff{}, nil, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Handoff{}, nil, err
	}
	decisions, err := s.repo.ListDecisions(ctx, e.ID)
	if err != nil {
		return Handoff{}, nil, err
	}
	candidates, err := s.repo.ListFindings(ctx, e.ID, TriageCandidate)
	if err != nil {
		return Handoff{}, nil, err
	}

	// Assemble the canonical handoff from captured state. Completeness + resume
	// point are COMPUTED from the phase-status set; last_validation/staleness come
	// from the validation seam (degrade to UNKNOWN, never a false pass).
	completeness := computeCompleteness(plan.Phases)
	resume := resumePhaseID(plan.Phases)
	lastVal, hasVal, staleness := s.lastValidation(ctx, plan, e.CurrentPhaseID)

	now := s.now()
	handoff := Handoff{
		ID:                uuid.NewString(),
		ExecutionID:       e.ID,
		PlanID:            plan.ID,
		Completeness:      completeness,
		ResumePhaseID:     resume,
		Decisions:         decisions,
		CandidateFindings: candidates,
		LastValidation:    lastVal,
		HasValidation:     hasVal,
		Staleness:         staleness,
		ProseHandoffRef:   "", // pass-through; the orchestration layer fills this by reference
		AssembledAt:       now,
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
		return Handoff{}, nil, err
	}
	_ = s.velocity.Emit(ctx, point) // best-effort; the no-op default never errors

	nudges := s.completionNudges(plan, candidates)
	return handoff, nudges, nil
}

func (s *service) GetHandoff(ctx context.Context, executionID string) (Handoff, error) {
	if _, err := s.getExecution(ctx, executionID); err != nil {
		return Handoff{}, err
	}
	h, ok, err := s.repo.GetHandoff(ctx, executionID)
	if err != nil {
		return Handoff{}, err
	}
	if !ok {
		// No handoff assembled yet — assemble a live view from current captured
		// state so a caller asking before Complete still gets the structured shape
		// rather than an error (the persisted record is written by Complete).
		return s.assembleLiveHandoff(ctx, executionID)
	}
	return h, nil
}

func (s *service) ListCandidateFindings(ctx context.Context, executionID string) ([]Finding, error) {
	return s.repo.ListFindings(ctx, strings.TrimSpace(executionID), TriageCandidate)
}

func (s *service) TriageFinding(ctx context.Context, findingID string, triage FindingTriage) (Finding, error) {
	f, ok, err := s.repo.GetFinding(ctx, strings.TrimSpace(findingID))
	if err != nil {
		return Finding{}, err
	}
	if !ok {
		return Finding{}, ErrFindingNotFound{ID: findingID}
	}
	if triage == "" {
		return Finding{}, ErrInvalidExecution{Reason: "triage state is required"}
	}
	f.Triage = triage
	if err := s.repo.SaveFinding(ctx, f); err != nil {
		return Finding{}, err
	}
	return f, nil
}

func (s *service) GetVelocity(ctx context.Context, planID string) ([]VelocityPoint, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return nil, ErrInvalidExecution{Reason: "plan id is required"}
	}
	// Resolve a slug to the canonical id so velocity recorded under the id is
	// found when the caller passes the slug.
	if plan, err := s.plans.GetPlan(ctx, planID); err == nil {
		planID = plan.ID
	}
	return s.repo.ListVelocity(ctx, planID)
}

// --- helpers ---

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

// buildContext assembles the just-in-time PhaseContext for the named phase.
func (s *service) buildContext(ctx context.Context, plan internalplans.Plan, phaseID string) PhaseContext {
	pctx := PhaseContext{
		ResumePhaseID: resumePhaseID(plan.Phases),
		Completeness:  computeCompleteness(plan.Phases),
	}
	if cur, ok := findPhase(plan.Phases, phaseID); ok {
		pctx.CurrentPhase = cur
		pctx.HasCurrent = true
		pctx.RequiredReading = cur.RequiredReading
		pctx.Reminders = cur.Reminders
	}
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

// lastValidation reads the validation seam for the phase's LAST STORED validation
// result + the staleness captured with it. This is a cheap store read — it never
// shells a subprocess — so status/next stay cheap. Degrades to UNKNOWN/absent
// (never a false pass) when the validator is nil, errors, or has no result yet.
func (s *service) lastValidation(ctx context.Context, plan internalplans.Plan, phaseID string) (ValidationResult, bool, internalplans.StalenessTier) {
	if s.validator == nil {
		return ValidationResult{}, false, internalplans.StalenessUnknown
	}
	res, ok, err := s.validator.LastValidation(ctx, plan.ID, phaseID)
	if err != nil || !ok {
		return ValidationResult{}, false, internalplans.StalenessUnknown
	}
	staleness := res.Staleness
	if staleness == "" {
		staleness = internalplans.StalenessUnknown
	}
	return res, true, staleness
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
	decisions, err := s.repo.ListDecisions(ctx, e.ID)
	if err != nil {
		return Handoff{}, err
	}
	candidates, err := s.repo.ListFindings(ctx, e.ID, TriageCandidate)
	if err != nil {
		return Handoff{}, err
	}
	lastVal, hasVal, staleness := s.lastValidation(ctx, plan, e.CurrentPhaseID)
	return Handoff{
		ExecutionID:       e.ID,
		PlanID:            plan.ID,
		Completeness:      computeCompleteness(plan.Phases),
		ResumePhaseID:     resumePhaseID(plan.Phases),
		Decisions:         decisions,
		CandidateFindings: candidates,
		LastValidation:    lastVal,
		HasValidation:     hasVal,
		Staleness:         staleness,
		AssembledAt:       s.now(),
	}, nil
}

// completionNudges builds the thin guided-completion checklist. Each nudge is
// satisfied=true when captured state already covers it.
func (s *service) completionNudges(plan internalplans.Plan, candidates []Finding) []CompletionNudge {
	allTerminal := computeCompleteness(plan.Phases) == CompletenessFull
	for _, ph := range plan.Phases {
		st := ph.Status
		if st == internalplans.PhaseStatusDone || st == internalplans.PhaseStatusBlocked {
			continue
		}
		allTerminal = false
		break
	}
	return []CompletionNudge{
		{
			Kind:      "record_finding",
			Message:   "Record any in-flight findings (possible bugs) before handing off — they file as candidates for operator triage.",
			Satisfied: len(candidates) > 0,
		},
		{
			Kind:      "file_bugs",
			Message:   "File out-of-scope defects to the issue tracker; candidate findings here stay unvalidated until an operator promotes them.",
			Satisfied: len(candidates) > 0,
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
func resumePhaseID(phases []internalplans.Phase) string {
	for _, ph := range phases {
		if ph.Status != internalplans.PhaseStatusDone {
			return ph.ID
		}
	}
	return ""
}

// computeCompleteness is FULL iff every phase is done; PARTIAL otherwise. An
// empty plan (no phases) is PARTIAL — there is nothing proven done.
func computeCompleteness(phases []internalplans.Phase) Completeness {
	if len(phases) == 0 {
		return CompletenessPartial
	}
	for _, ph := range phases {
		if ph.Status != internalplans.PhaseStatusDone {
			return CompletenessPartial
		}
	}
	return CompletenessFull
}

// findPhase returns the phase with the given id.
func findPhase(phases []internalplans.Phase, phaseID string) (internalplans.Phase, bool) {
	for _, ph := range phases {
		if ph.ID == phaseID {
			return ph, true
		}
	}
	return internalplans.Phase{}, false
}

// nextPhase returns the phase immediately after phaseID in authored order.
func nextPhase(phases []internalplans.Phase, phaseID string) (internalplans.Phase, bool) {
	for i, ph := range phases {
		if ph.ID == phaseID && i+1 < len(phases) {
			return phases[i+1], true
		}
	}
	return internalplans.Phase{}, false
}
