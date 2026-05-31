package autosteer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
)

// defaultAuditPreset is used when a profile does not pin one.
const defaultAuditPreset = "comprehensive"

// NewExecutionOrchestratorDefault wires a production ExecutionOrchestrator:
// test-genie as the audit runner, prompt-manager's catalog as the skill→dimension
// source, and Postgres-backed state/trace. Operates in degraded mode if
// prompt-manager is unavailable (no eligible skills → loop terminates early).
func NewExecutionOrchestratorDefault(profileRepo ProfileRepository, db *sql.DB, projectRoot string) *ExecutionOrchestrator {
	promptEnhancer := NewPromptEnhancer()
	catalog := NewPromptLoaderCatalog(promptEnhancer.GetPromptLoader())

	return NewExecutionOrchestrator(
		NewExecutionStateManager(db),
		profileRepo,
		&findings.TestGenieRunner{ProjectRoot: projectRoot},
		catalog,
		promptEnhancer,
		NewMetricsCollector(projectRoot),
		NewTraceStore(db),
	)
}

// ExecutionOrchestrator is the closed-loop controller. Each entry runs the
// DIAGNOSE → SELECT → (EXECUTE happens out-of-band via agent-manager) → MEASURE
// → TERMINATE loop, persisting a decision trace each iteration.
type ExecutionOrchestrator struct {
	stateManager     ExecutionStateRepository
	profileService   ProfileRepository
	auditRunner      findings.AuditRunner
	catalog          skillmap.CatalogSource
	promptEnhancer   PromptEnhancerAPI
	metricsCollector MetricsProvider
	traceStore       *TraceStore
	terminator       *Terminator
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
	metricsCollector MetricsProvider,
	traceStore *TraceStore,
) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		stateManager:     stateManager,
		profileService:   profileService,
		auditRunner:      auditRunner,
		catalog:          catalog,
		promptEnhancer:   promptEnhancer,
		metricsCollector: metricsCollector,
		traceStore:       traceStore,
		terminator:       NewTerminator(),
	}
}

func (o *ExecutionOrchestrator) resolver() *skillmap.Resolver {
	return skillmap.NewResolver(o.catalog)
}

func (o *ExecutionOrchestrator) auditPreset(profile *AutoSteerProfile) string {
	if profile != nil && profile.AuditPreset != "" {
		return profile.AuditPreset
	}
	return defaultAuditPreset
}

// collectMetrics best-effort gathers gap metrics for the objective's
// operational-targets target. Failures are non-fatal (returns the zero value).
func (o *ExecutionOrchestrator) collectMetrics(scenarioName string, iteration int) MetricsSnapshot {
	if o.metricsCollector == nil {
		return MetricsSnapshot{Timestamp: time.Now()}
	}
	m, err := o.metricsCollector.CollectMetrics(scenarioName, iteration, iteration)
	if err != nil || m == nil {
		log.Printf("Auto Steer: gap-metric collection failed (non-fatal): %v", err)
		return MetricsSnapshot{Timestamp: time.Now()}
	}
	return *m
}

// StartExecution runs the initial DIAGNOSE + SELECT for a task: full audit,
// build the findings vector, pick the first skill, and persist the opening
// decision-trace entry.
func (o *ExecutionOrchestrator) StartExecution(taskID, profileID, scenarioName string) (*ProfileExecutionState, error) {
	profile, err := o.profileService.GetProfile(profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	fs, err := o.fullAudit(context.Background(), scenarioName, profile)
	if err != nil {
		return nil, fmt.Errorf("initial audit failed: %w", err)
	}

	state := o.stateManager.InitializeState(taskID, profileID)
	state.Findings = fs
	state.Metrics = o.collectMetrics(scenarioName, 0)
	state.ScoreHistory = []float64{fs.TotalScore}

	o.selectInto(state, profile, scenarioName)

	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}

	log.Printf("Auto Steer: started controller for task %s (profile %s) — %d open findings, first skill %q",
		taskID, profileID, len(fs.Findings), state.CurrentSkill)
	return state, nil
}

// fullAudit runs a complete preset audit and builds the findings vector.
func (o *ExecutionOrchestrator) fullAudit(ctx context.Context, scenarioName string, profile *AutoSteerProfile) (findings.FindingsState, error) {
	audit, err := o.auditRunner.Audit(ctx, findings.AuditRequest{
		Scenario: scenarioName,
		Preset:   o.auditPreset(profile),
	})
	if err != nil {
		return findings.FindingsState{}, err
	}
	return findings.BuildState(findings.ToFindings(audit)), nil
}

// selectInto runs SELECT for the current findings and records the opening trace
// entry for the next iteration (ScoreAfter is filled after the skill runs).
func (o *ExecutionOrchestrator) selectInto(state *ProfileExecutionState, profile *AutoSteerProfile, scenarioName string) {
	sel := NewSelector(o.resolver()).SelectNextSkill(state.Findings, profile)
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

	// MEASURE — re-audit and merge into the findings vector.
	prevScore := lastScore(state.ScoreHistory, state.Findings.TotalScore)
	newFS, err := o.reaudit(context.Background(), scenarioName, profile, state)
	if err != nil {
		return nil, fmt.Errorf("re-audit failed: %w", err)
	}
	scoreAfter := newFS.TotalScore
	realizedDelta := prevScore - scoreAfter // positive = improvement

	o.recordRealized(state, scoreAfter, realizedDelta)
	state.Findings = newFS
	state.ScoreHistory = append(state.ScoreHistory, scoreAfter)
	state.Metrics = o.collectMetrics(scenarioName, state.Iteration)

	// TERMINATE — global, gradient-based.
	if stop, reason := o.terminator.ShouldStop(state, profile); stop {
		log.Printf("Auto Steer: task %s stopping — %s", taskID, reason)
		if err := o.stateManager.FinalizeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to finalize execution: %w", err)
		}
		return &IterationEvaluation{ShouldStop: true, Reason: reason}, nil
	}

	// SELECT next skill for the upcoming run.
	o.selectInto(state, profile, scenarioName)
	if state.CurrentSkill == "" {
		log.Printf("Auto Steer: task %s stopping — %s", taskID, StopNothingActionable)
		if err := o.stateManager.FinalizeExecution(state, scenarioName); err != nil {
			return nil, fmt.Errorf("failed to finalize execution: %w", err)
		}
		return &IterationEvaluation{ShouldStop: true, Reason: StopNothingActionable}, nil
	}

	if err := o.stateManager.Save(state); err != nil {
		return nil, fmt.Errorf("failed to save execution state: %w", err)
	}
	return &IterationEvaluation{ShouldStop: false, Reason: StopReasonContinue, ChosenSkill: state.CurrentSkill}, nil
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
// iteration whose skill just ran) in both the live state and the durable store.
func (o *ExecutionOrchestrator) recordRealized(state *ProfileExecutionState, scoreAfter, realizedDelta float64) {
	if n := len(state.Trace); n > 0 {
		state.Trace[n-1].ScoreAfter = scoreAfter
		state.Trace[n-1].RealizedDelta = realizedDelta
	}
	if err := o.traceStore.SetRealized(state.TaskID, state.Iteration, scoreAfter, realizedDelta); err != nil {
		log.Printf("Auto Steer: failed to persist realized delta (non-fatal): %v", err)
	}
}

// GetExecutionState retrieves the current controller state for a task.
func (o *ExecutionOrchestrator) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	return o.stateManager.Get(taskID)
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
