package autosteer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/autosteer/gameguard"
	"github.com/ecosystem-manager/api/pkg/completeness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
	"github.com/vrooli/maturity-go/dimensions"
)

// defaultAuditPreset is used when a profile does not pin one.
const defaultAuditPreset = "comprehensive"

// NewExecutionOrchestratorDefault wires a production ExecutionOrchestrator:
// test-genie as the audit runner, prompt-manager's catalog as the skill→dimension
// source, scenario-completeness-scoring for the operational-targets measurement,
// and SQLite-backed state/trace. Operates in degraded mode if prompt-manager is
// unavailable (no eligible skills → loop terminates early).
func NewExecutionOrchestratorDefault(profileRepo ProfileRepository, db *sql.DB, projectRoot string) *ExecutionOrchestrator {
	promptEnhancer := NewPromptEnhancer()
	catalog := NewPromptLoaderCatalog(promptEnhancer.GetPromptLoader())

	return NewExecutionOrchestrator(
		NewExecutionStateManager(db),
		profileRepo,
		&findings.TestGenieRunner{ProjectRoot: projectRoot},
		catalog,
		promptEnhancer,
		completeness.NewClient(0),
		NewTraceStore(db),
	)
}

// ExecutionOrchestrator is the closed-loop controller. Each entry runs the
// DIAGNOSE → SELECT → (EXECUTE happens out-of-band via agent-manager) → MEASURE
// → TERMINATE loop, persisting a decision trace each iteration. SELECT is greedy:
// the skill that targets the heaviest open finding dimension (see
// docs/concepts/CONTROL-MODEL.md).
type ExecutionOrchestrator struct {
	stateManager   ExecutionStateRepository
	profileService ProfileRepository
	auditRunner    findings.AuditRunner
	catalog        skillmap.CatalogSource
	promptEnhancer PromptEnhancerAPI
	completeness   completeness.Provider
	traceStore     *TraceStore
	terminator     *Terminator

	// runMu guards the per-task pending run IDs.
	runMu sync.Mutex
	// pendingRunID holds the agent-manager run ID of the most recent run per
	// task, recorded by the queue alongside the run and consumed once at the next
	// EvaluateIteration so the anti-gaming classifier can fetch that run's
	// code-level diff.
	pendingRunID map[string]string

	// diffProvider is the anti-gaming diff seam. Nil ⇒ gaming detection disabled
	// (fail-open: the loop runs exactly as before). Production wires an
	// agent-manager-backed provider via SetDiffProvider. A gamed iteration is
	// flagged on the trace and blocks the shadow→live promote (see RunGamed).
	diffProvider RunDiffProvider
}

// NewExecutionOrchestrator creates a controller from its collaborators. All
// dependencies are interfaces (or injectable), enabling unit testing without a
// database, test-genie, or prompt-manager.
func NewExecutionOrchestrator(
	stateManager ExecutionStateRepository,
	profileService ProfileRepository,
	auditRunner findings.AuditRunner,
	catalog skillmap.CatalogSource,
	promptEnhancer PromptEnhancerAPI,
	completenessProvider completeness.Provider,
	traceStore *TraceStore,
) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		stateManager:   stateManager,
		profileService: profileService,
		auditRunner:    auditRunner,
		catalog:        catalog,
		promptEnhancer: promptEnhancer,
		completeness:   completenessProvider,
		traceStore:     traceStore,
		terminator:     NewTerminator(),
		pendingRunID:   make(map[string]string),
	}
}

func (o *ExecutionOrchestrator) resolver() *skillmap.Resolver {
	return skillmap.NewResolver(o.catalog)
}

// runSelect performs one greedy SELECT for an upcoming iteration: bucket the
// open findings by dimension, rank by profile-weighted severity, and pick the
// first eligible skill for the heaviest actionable dimension.
func (o *ExecutionOrchestrator) runSelect(state *ProfileExecutionState, profile *AutoSteerProfile) Selection {
	return NewSelector(o.resolver()).SelectNextSkill(state.Findings, profile)
}

func (o *ExecutionOrchestrator) auditPreset(profile *AutoSteerProfile) string {
	if profile != nil && profile.AuditPreset != "" {
		return profile.AuditPreset
	}
	return defaultAuditPreset
}

// fetchScore reads the scenario's completeness measurement from the
// scenario-completeness-scoring authority — the operational-targets completion
// the controller gates on. Measurement is load-bearing for termination, so
// unlike a best-effort signal it does NOT fail open; the error is returned and
// the caller degrades loudly (abort the start, or halt the iteration with a
// measurement_unavailable stop reason) rather than substituting a home-grown
// collector.
func (o *ExecutionOrchestrator) fetchScore(ctx context.Context, scenarioName string) (completeness.Score, error) {
	if o.completeness == nil {
		return completeness.Score{}, fmt.Errorf("no completeness provider wired")
	}
	return o.completeness.Score(ctx, scenarioName)
}

// StartExecution runs the initial DIAGNOSE + SELECT for a task: full audit,
// build the findings vector, pick the first skill, and persist the opening
// decision-trace entry.
func (o *ExecutionOrchestrator) StartExecution(taskID, profileID, scenarioName string) (*ProfileExecutionState, error) {
	profile, err := o.profileService.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	resolver := o.resolver()
	if err := ReconcileProfile(profile, resolver); err != nil {
		return nil, fmt.Errorf("profile/catalog mismatch: %w", err)
	}
	allowCount, relevantCount, uncovered := coverageSummary(profile, resolver)
	log.Printf("Auto Steer: coverage preflight for profile %s — effective_allow=%d relevant_dimensions=%d uncovered_dimensions=%v",
		profileID, allowCount, relevantCount, uncovered)

	fs, err := o.fullAudit(context.Background(), scenarioName, profile)
	if err != nil {
		return nil, fmt.Errorf("initial audit failed: %w", err)
	}

	// Fetch the completeness score AFTER the audit above wrote fresh
	// phase-results, so operational-targets reflect this iteration. No fallback —
	// if measurement is unavailable the start aborts loudly.
	score, err := o.fetchScore(context.Background(), scenarioName)
	if err != nil {
		return nil, fmt.Errorf("completeness measurement unavailable: %w", err)
	}

	state := o.stateManager.InitializeState(taskID, profileID)
	state.Findings = fs
	state.Completeness = score
	state.ScoreHistory = []float64{fs.TotalScore}

	o.selectInto(state, profile, scenarioName)

	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	log.Printf("Auto Steer: started controller for task %s (profile %s) — %d open findings, first skill %q",
		taskID, profileID, len(fs.Findings), state.CurrentSkill)
	return state, nil
}

// EvaluateStart runs the controller's termination check BEFORE the first agent
// run. StartExecution has already done the initial DIAGNOSE + SELECT; this asks
// whether running an agent at all is warranted. When the objective is already
// met, or there is nothing to steer (empty findings ⇒ no skill selected), it
// finalizes the run and returns proceed=false — launching a blind, unsteered
// agent pass on an already-satisfied scenario can only regress it. Otherwise it
// returns proceed=true and the caller runs the selected skill.
func (o *ExecutionOrchestrator) EvaluateStart(taskID, scenarioName string) (proceed bool, reason string, err error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get execution state: %w", err)
	}
	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return false, "", fmt.Errorf("failed to get profile: %w", err)
	}
	if err := ReconcileProfile(profile, o.resolver()); err != nil {
		return false, "", fmt.Errorf("profile/catalog mismatch: %w", err)
	}

	if met, metReason := objectiveMet(state, profile); met {
		o.recordHalt(state, metReason)
		if ferr := o.stateManager.FinalizeExecution(state, scenarioName); ferr != nil {
			return false, "", fmt.Errorf("failed to finalize execution: %w", ferr)
		}
		log.Printf("Auto Steer: task %s objective already met at start — %s", taskID, metReason)
		return false, metReason, nil
	}

	if state.CurrentSkill == "" {
		o.recordHalt(state, StopNothingActionable)
		if ferr := o.stateManager.FinalizeExecution(state, scenarioName); ferr != nil {
			return false, "", fmt.Errorf("failed to finalize execution: %w", ferr)
		}
		log.Printf("Auto Steer: task %s nothing actionable at start — %s", taskID, StopNothingActionable)
		return false, StopNothingActionable, nil
	}

	return true, "", nil
}

// auditConclusiveAttempts / auditConclusiveBackoff bound the retry of an
// inconclusive audit (no phases executed). DIAGNOSE is non-deterministic: a
// cold/unwarmed test-genie can return an all-skipped report that reads as a
// spurious clean zero, which would make the controller believe the objective is
// already met. Retrying recovers the real findings; if it stays inconclusive we
// proceed with the empty result and let EvaluateStart's guard avoid a blind run.
// Both are vars (not consts) so tests can disable the backoff.
var (
	auditConclusiveAttempts = 3
	auditConclusiveBackoff  = 3 * time.Second
)

// fullAudit runs a complete preset audit and builds the findings vector. An
// audit whose phases all skipped (Conclusive()==false) is retried, since an
// empty-but-inconclusive result must not be mistaken for a clean scenario.
func (o *ExecutionOrchestrator) fullAudit(ctx context.Context, scenarioName string, profile *AutoSteerProfile) (findings.FindingsState, error) {
	preset := o.auditPreset(profile)
	var audit *findings.Audit
	for attempt := 1; attempt <= auditConclusiveAttempts; attempt++ {
		a, err := o.auditRunner.Audit(ctx, findings.AuditRequest{Scenario: scenarioName, Preset: preset})
		if err != nil {
			return findings.FindingsState{}, err
		}
		audit = a
		if audit.Conclusive() {
			break
		}
		if attempt < auditConclusiveAttempts {
			log.Printf("Auto Steer: audit of %q executed no phases (attempt %d/%d) — retrying before trusting an empty result",
				scenarioName, attempt, auditConclusiveAttempts)
			if auditConclusiveBackoff > 0 {
				time.Sleep(auditConclusiveBackoff)
			}
			continue
		}
		log.Printf("Auto Steer: audit of %q still inconclusive after %d attempts — proceeding with no findings; the start guard will avoid a blind run",
			scenarioName, auditConclusiveAttempts)
	}
	return findings.BuildState(findings.ToFindings(audit)), nil
}

// selectInto runs SELECT for the current findings and records the opening trace
// entry for the next iteration (ScoreAfter is filled after the skill runs).
func (o *ExecutionOrchestrator) selectInto(state *ProfileExecutionState, profile *AutoSteerProfile, scenarioName string) {
	sel := o.runSelect(state, profile)
	state.Iteration++
	state.CurrentSkill = sel.SkillID
	state.CurrentRationale = sel.Rationale

	entry := DecisionTraceEntry{
		Iteration:         state.Iteration,
		Timestamp:         time.Now(),
		DimensionScores:   dimensionScoreMap(state.Findings),
		HeaviestDimension: string(sel.Dimension),
		ChosenSkill:       sel.SkillID,
		Rationale:         sel.Rationale,
		Fingerprint:       state.Findings.Fingerprint,
		ScoreBefore:       state.Findings.TotalScore,
	}
	state.Trace = append(state.Trace, entry)
	if err := o.traceStore.Append(state.TaskID, state.ProfileID, scenarioName, entry); err != nil {
		log.Printf("Auto Steer: failed to persist decision trace (non-fatal): %v", err)
	}
}

// EvaluateIteration runs MEASURE + TERMINATE after the selected skill has
// executed: re-audit (targeted or full per cadence), record the realized delta,
// then either finalize (stop) or select the next skill (continue).
func (o *ExecutionOrchestrator) EvaluateIteration(taskID, scenarioName string) (*IterationEvaluation, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}
	if state == nil {
		return &IterationEvaluation{ShouldStop: false}, nil
	}

	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	if err := ReconcileProfile(profile, o.resolver()); err != nil {
		return nil, fmt.Errorf("profile/catalog mismatch: %w", err)
	}

	// MEASURE — re-audit, then record the realized delta against the prior score.
	prevScore := lastScore(state.ScoreHistory, state.Findings.TotalScore)
	newFS, err := o.reaudit(context.Background(), scenarioName, profile, state)
	if err != nil {
		return nil, fmt.Errorf("re-audit failed: %w", err)
	}
	scoreAfter := newFS.TotalScore
	realizedDelta := prevScore - scoreAfter // positive = improvement

	// Anti-gaming: classify the run's code-level diff. A gamed iteration (e.g.
	// weakened [REQ:] tests, deleted ledgers, suppression directives) is flagged
	// on the trace and blocks the shadow→live promote (see RunGamed), so a faked
	// "green" can never be promoted.
	gaming := o.classifyGaming(context.Background(), taskID)
	o.recordRealized(state, scoreAfter, realizedDelta, gaming)
	state.Findings = newFS
	state.ScoreHistory = append(state.ScoreHistory, scoreAfter)

	// Re-audit above wrote fresh phase-results; read the completeness score now so
	// operational-targets reflect this iteration. Measurement is load-bearing for
	// termination — if it is unavailable, halt the run loudly rather than
	// continuing on stale/zero data.
	score, scoreErr := o.fetchScore(context.Background(), scenarioName)
	if scoreErr != nil {
		log.Printf("Auto Steer: task %s halting — %s: %v", taskID, StopMeasurementUnavailable, scoreErr)
		o.recordHalt(state, StopMeasurementUnavailable)
		if err := o.stateManager.FinalizeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to finalize execution: %w", err)
		}
		return &IterationEvaluation{ShouldStop: true, Reason: StopMeasurementUnavailable}, nil
	}
	state.Completeness = score

	// Snapshot the objective-met signal and the completed iteration before SELECT
	// advances the counter — the Baseline Modes checkpoint_on_green cadence reads
	// these from the continue-path evaluation to promote a validated win early.
	objMet, _ := objectiveMet(state, profile)
	completedIter := state.Iteration

	// TERMINATE — global, gradient-based.
	if stop, reason := o.terminator.ShouldStop(state, profile); stop {
		log.Printf("Auto Steer: task %s stopping — %s", taskID, reason)
		o.recordHalt(state, reason)
		if err := o.stateManager.FinalizeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to finalize execution: %w", err)
		}
		return &IterationEvaluation{ShouldStop: true, Reason: reason}, nil
	}

	// SELECT next skill for the upcoming run.
	o.selectInto(state, profile, scenarioName)
	if state.CurrentSkill == "" {
		log.Printf("Auto Steer: task %s stopping — %s", taskID, StopNothingActionable)
		o.recordHalt(state, StopNothingActionable)
		if err := o.stateManager.FinalizeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to finalize execution: %w", err)
		}
		return &IterationEvaluation{ShouldStop: true, Reason: StopNothingActionable}, nil
	}

	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}
	return &IterationEvaluation{
		ShouldStop:   false,
		Reason:       StopReasonContinue,
		ChosenSkill:  state.CurrentSkill,
		ObjectiveMet: objMet,
		Iteration:    completedIter,
	}, nil
}

// reaudit re-measures the target. Cost control: a targeted re-audit (only the
// dimensions the just-run skill targets) drives the inner loop; the full preset
// runs on the configured cadence. Targeted findings are merged over the stale
// ones for untouched dimensions so the score reflects the whole target.
func (o *ExecutionOrchestrator) reaudit(ctx context.Context, scenarioName string, profile *AutoSteerProfile, state *ProfileExecutionState) (findings.FindingsState, error) {
	cadence := profile.Budget.ReauditCadence
	fullPass := cadence > 0 && state.Iteration%cadence == 0
	if fullPass {
		return o.fullAudit(ctx, scenarioName, profile)
	}

	skillDims := o.resolver().DimensionsForSkill(state.CurrentSkill)
	phases := dimensions.PhasesForDimensions(skillDims...)
	if len(phases) == 0 {
		// No targeted phases resolvable; fall back to a full audit.
		return o.fullAudit(ctx, scenarioName, profile)
	}

	audit, err := o.auditRunner.Audit(ctx, findings.AuditRequest{
		Scenario: scenarioName,
		Phases:   phases,
	})
	if err != nil {
		return findings.FindingsState{}, err
	}

	reauditedDims := make(map[dimensions.Dimension]bool, len(skillDims))
	for _, d := range skillDims {
		reauditedDims[d] = true
	}
	fresh := findings.ToFindings(audit)
	merged := mergeFindings(state.Findings.Findings, fresh, reauditedDims)
	return findings.BuildState(merged), nil
}

// mergeFindings replaces findings for re-audited dimensions with the fresh set
// while preserving the (still-valid) findings for dimensions not re-audited.
func mergeFindings(prior, fresh []findings.Finding, reauditedDims map[dimensions.Dimension]bool) []findings.Finding {
	out := make([]findings.Finding, 0, len(prior)+len(fresh))
	for _, f := range prior {
		if reauditedDims[f.Dimension] {
			continue // superseded by the fresh targeted audit
		}
		out = append(out, f)
	}
	for _, f := range fresh {
		if reauditedDims[f.Dimension] {
			out = append(out, f)
		}
	}
	return out
}

// recordRealized fills the realized outcome on the most recent trace entry (the
// iteration whose skill just ran) — score, realized delta, and the anti-gaming
// verdict — in both the live state and the durable store.
func (o *ExecutionOrchestrator) recordRealized(state *ProfileExecutionState, scoreAfter, realizedDelta float64, gaming gameguard.Result) {
	n := len(state.Trace)
	if n == 0 {
		return
	}
	e := &state.Trace[n-1]
	e.ScoreAfter = scoreAfter
	e.RealizedDelta = realizedDelta
	if cause := gamingTrace(gaming); cause != "" {
		e.GamingCause = cause
	}
	if gaming.Gamed {
		log.Printf("Auto Steer: GAMING DETECTED task %s iteration %d — %s; flagged on the trace; the shadow→live promote will be blocked",
			state.TaskID, e.Iteration, gaming.CauseString())
	}
	if err := o.traceStore.SetRealized(state.TaskID, *e); err != nil {
		log.Printf("Auto Steer: failed to persist realized delta (non-fatal): %v", err)
	}
}

// recordHalt stamps the terminal halt reason on the final trace entry so the
// decision trace shows why the controller stopped.
func (o *ExecutionOrchestrator) recordHalt(state *ProfileExecutionState, reason string) {
	n := len(state.Trace)
	if n == 0 {
		return
	}
	e := &state.Trace[n-1]
	e.HaltReason = reason
	if err := o.traceStore.SetHalt(state.TaskID, e.Iteration, reason); err != nil {
		log.Printf("Auto Steer: failed to persist halt reason (non-fatal): %v", err)
	}
}

// GetExecutionState retrieves the current controller state for a task.
func (o *ExecutionOrchestrator) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	return o.stateManager.Get(taskID)
}

// ProfileForTask returns the objective-function profile a task is executing
// under (nil when the task has no controller state). It backs the queue's
// Baseline Modes engagement decisions (reading the profile's BaselinePromote
// block) without exposing the internal profile/state repositories.
func (o *ExecutionOrchestrator) ProfileForTask(taskID string) (*AutoSteerProfile, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	return o.profileService.GetProfile(state.ProfileID)
}

// RunGamed reports whether any iteration of a task's run was flagged as gamed by
// the anti-gaming classifier (weakened tests, deleted ledgers, suppression
// directives). The queue consults it before a shadow→live promote: a gamed run
// must be abandoned, not promoted, because its "green" cannot be trusted.
func (o *ExecutionOrchestrator) RunGamed(taskID string) (bool, error) {
	trace, err := o.GetDecisionTrace(taskID)
	if err != nil {
		return false, err
	}
	for _, e := range trace {
		if strings.HasPrefix(e.GamingCause, "gamed:") {
			return true, nil
		}
	}
	return false, nil
}

// GetCurrentSet returns the currently selected steering skill for a task (the
// controller selects a single skill; the slice shape is retained for the
// steering-provider contract).
func (o *ExecutionOrchestrator) GetCurrentSet(taskID string) ([]string, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution state: %w", err)
	}
	if state == nil || state.CurrentSkill == "" {
		return nil, nil
	}
	return []string{state.CurrentSkill}, nil
}

// DeleteExecutionState removes any active execution state for a task.
func (o *ExecutionOrchestrator) DeleteExecutionState(taskID string) error {
	return o.stateManager.Delete(taskID)
}

// GetEnhancedPrompt generates the controller's prompt section for a task.
func (o *ExecutionOrchestrator) GetEnhancedPrompt(taskID string) (string, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to get execution state: %w", err)
	}
	if state == nil || state.CurrentSkill == "" {
		return "", nil
	}
	profile, err := o.profileService.GetProfile(state.ProfileID)
	if err != nil {
		return "", fmt.Errorf("failed to get profile: %w", err)
	}
	return o.promptEnhancer.GenerateControllerSection(state, profile), nil
}

// GenerateSkillSetSection renders a standalone skill-set block.
func (o *ExecutionOrchestrator) GenerateSkillSetSection(skillIDs []string, withScope bool, scope string) string {
	if o == nil || o.promptEnhancer == nil {
		return ""
	}
	return o.promptEnhancer.GenerateSkillSetSection(skillIDs, withScope, scope)
}

// IsAutoSteerActive reports whether a task has active controller state.
func (o *ExecutionOrchestrator) IsAutoSteerActive(taskID string) (bool, error) {
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return false, fmt.Errorf("failed to get execution state: %w", err)
	}
	return state != nil, nil
}

// GetDecisionTrace returns the durable decision trace for a task. The trace
// store (VARCHAR-keyed, survives finalization) is authoritative; the live
// state's in-memory trace is only consulted when no store is wired.
func (o *ExecutionOrchestrator) GetDecisionTrace(taskID string) ([]DecisionTraceEntry, error) {
	if o.traceStore != nil {
		return o.traceStore.GetTrace(taskID)
	}
	state, err := o.stateManager.Get(taskID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, nil
	}
	return state.Trace, nil
}

// dimensionScoreMap converts a findings state's per-dimension scores into a
// string-keyed map for the decision trace JSON.
func dimensionScoreMap(fs findings.FindingsState) map[string]float64 {
	out := make(map[string]float64, len(fs.DimensionScore))
	for d, s := range fs.DimensionScore {
		out[string(d)] = s
	}
	return out
}

// lastScore returns the last score in the history, or a fallback when empty.
func lastScore(history []float64, fallback float64) float64 {
	if n := len(history); n > 0 {
		return history[n-1]
	}
	return fallback
}
