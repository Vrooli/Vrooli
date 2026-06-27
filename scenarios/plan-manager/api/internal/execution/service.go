package execution

import (
	"context"
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

// Service is the execution application surface — the guided runner.
type Service interface {
	Start(ctx context.Context, planID, runID string) (Execution, GuidedStep, error)
	GetStatus(ctx context.Context, executionID string) (Execution, PhaseContext, GuidedStep, error)
	GetNext(ctx context.Context, executionID string) (PhaseContext, bool, GuidedStep, error)
	TransitionPhase(ctx context.Context, executionID, phaseID string, to planmodel.PhaseStatus) (Execution, planmodel.Plan, GuidedStep, error)

	RecordDecision(ctx context.Context, executionID, phaseID, summary, detail string) (Decision, GuidedStep, error)
	RecordFinding(ctx context.Context, executionID, phaseID, title, detail string) (Finding, GuidedStep, error)

	Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error)
	GetHandoff(ctx context.Context, executionID string) (Handoff, GuidedStep, error)

	ListCandidateFindings(ctx context.Context, executionID string) ([]Finding, GuidedStep, error)
	TriageFinding(ctx context.Context, findingID string, triage FindingTriage) (Finding, GuidedStep, error)

	GetVelocity(ctx context.Context, planID string) ([]VelocityPoint, GuidedStep, error)
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

func (s *service) Start(ctx context.Context, planID, runID string) (Execution, GuidedStep, error) {
	planID = strings.TrimSpace(planID)
	if planID == "" {
		return Execution{}, GuidedStep{}, ErrInvalidExecution{Reason: "plan id is required"}
	}
	// Resolve the plan (id or slug) through the plans SSOT so the linkage stores
	// the canonical plan id, and so a bad plan fails fast with NotFound.
	plan, err := s.plans.GetPlan(ctx, planID)
	if err != nil {
		return Execution{}, GuidedStep{}, err
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
		return Execution{}, GuidedStep{}, err
	}
	return e, stepForStarted(e), nil
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
	pctx := s.buildContext(ctx, plan, e.CurrentPhaseID)
	return e, pctx, stepForContext(e.ID, pctx, e.Complete), nil
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
	pctx := s.buildContext(ctx, plan, next)
	return pctx, e.Complete, stepForContext(e.ID, pctx, e.Complete), nil
}

func (s *service) TransitionPhase(ctx context.Context, executionID, phaseID string, to planmodel.PhaseStatus) (Execution, planmodel.Plan, GuidedStep, error) {
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
	// Delegate the phase-status change to the plans domain — it stays the single
	// source of truth for the record (plan status is recomputed there).
	target.Status = to
	updated, err := s.plans.UpdatePhase(ctx, plan.ID, target)
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

func (s *service) RecordDecision(ctx context.Context, executionID, phaseID, summary, detail string) (Decision, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Decision{}, GuidedStep{}, err
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
		return Decision{}, GuidedStep{}, err
	}
	return d, stepForRecorded(e.ID, phaseID), nil
}

func (s *service) RecordFinding(ctx context.Context, executionID, phaseID, title, detail string) (Finding, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Finding{}, GuidedStep{}, err
	}
	// Attribution-keyed dedup: a finding with the same (attribution_run_id, title)
	// is not double-filed. When the run id is absent, dedup by title within the
	// execution (best-effort). Collect the candidate set first (pool=1 safe — no
	// nested query inside the lookup), then return the existing match if any.
	existing, err := s.repo.ListFindings(ctx, e.ID, "")
	if err != nil {
		return Finding{}, GuidedStep{}, err
	}
	for _, f := range existing {
		if !strings.EqualFold(strings.TrimSpace(f.Title), strings.TrimSpace(title)) {
			continue
		}
		if e.RunID != "" {
			if f.AttributionRunID == e.RunID {
				return f, stepForRecorded(e.ID, phaseID), nil
			}
			continue
		}
		// No run id: dedup by title within the execution.
		return f, stepForRecorded(e.ID, phaseID), nil
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
				return winner, stepForRecorded(e.ID, phaseID), nil
			}
		}
		return Finding{}, GuidedStep{}, err
	}
	return f, stepForRecorded(e.ID, phaseID), nil
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

func (s *service) Complete(ctx context.Context, executionID string, inputs CompletionInputs) (Handoff, []CompletionNudge, GuidedStep, error) {
	e, err := s.getExecution(ctx, executionID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	plan, err := s.plans.GetPlan(ctx, e.PlanID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	decisions, err := s.repo.ListDecisions(ctx, e.ID)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
	}
	candidates, err := s.repo.ListFindings(ctx, e.ID, TriageCandidate)
	if err != nil {
		return Handoff{}, nil, GuidedStep{}, err
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
		return Handoff{}, nil, GuidedStep{}, err
	}
	_ = s.velocity.Emit(ctx, point) // best-effort; the no-op default never errors

	nudges := s.completionNudges(plan, candidates)
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

func (s *service) ListCandidateFindings(ctx context.Context, executionID string) ([]Finding, GuidedStep, error) {
	findings, err := s.repo.ListFindings(ctx, strings.TrimSpace(executionID), TriageCandidate)
	return findings, GuidedStep{
		StepKind: "candidate_findings",
		Title:    "Candidate Findings",
		Summary:  "Candidate findings are awaiting operator triage.",
		NextActions: []NextAction{{
			ID:                 "triage-finding",
			Kind:               NextActionRecommended,
			Label:              "Triage a finding",
			Reason:             "Promote or dismiss a candidate finding after review.",
			Argv:               []string{"exec", "triage", "<finding id>", "--status", "promoted"},
			ContentPlaceholder: "<finding id>",
		}},
	}, err
}

func (s *service) TriageFinding(ctx context.Context, findingID string, triage FindingTriage) (Finding, GuidedStep, error) {
	f, ok, err := s.repo.GetFinding(ctx, strings.TrimSpace(findingID))
	if err != nil {
		return Finding{}, GuidedStep{}, err
	}
	if !ok {
		return Finding{}, GuidedStep{}, ErrFindingNotFound{ID: findingID}
	}
	if triage == "" {
		return Finding{}, GuidedStep{}, ErrInvalidExecution{Reason: "triage state is required"}
	}
	f.Triage = triage
	if err := s.repo.SaveFinding(ctx, f); err != nil {
		return Finding{}, GuidedStep{}, err
	}
	return f, stepForRecorded(f.ExecutionID, f.PhaseID), nil
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
func (s *service) buildContext(ctx context.Context, plan planmodel.Plan, phaseID string) PhaseContext {
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
func (s *service) completionNudges(plan planmodel.Plan, candidates []Finding) []CompletionNudge {
	allTerminal := computeCompleteness(plan.Phases) == CompletenessFull
	for _, ph := range plan.Phases {
		st := ph.Status
		if st == planmodel.PhaseStatusDone || st == planmodel.PhaseStatusBlocked {
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
func resumePhaseID(phases []planmodel.Phase) string {
	for _, ph := range phases {
		if ph.Status != planmodel.PhaseStatusDone {
			return ph.ID
		}
	}
	return ""
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
