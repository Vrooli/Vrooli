package autosteer

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"log"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/dimensions"
	"github.com/ecosystem-manager/api/pkg/dtv"
	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/ecosystem-manager/api/pkg/findings"
	"github.com/ecosystem-manager/api/pkg/skillmap"
)

// defaultAuditPreset is used when a profile does not pin one.
const defaultAuditPreset = "comprehensive"

// Exploration policy for the v1 bandit. epsilon decays as base/(1+iteration):
// early iterations explore a little to gather efficacy evidence; later ones
// exploit. The prior is uniform/neutral in P1 (so a fresh target reproduces v0
// greedy ordering); P2 swaps in DTV trust/cost priors here.
const (
	explorationBaseEpsilon = 0.15
	coldStartPrior         = 0.0
)

// NewExecutionOrchestratorDefault wires a production ExecutionOrchestrator:
// test-genie as the audit runner, prompt-manager's catalog as the skill→dimension
// source, and Postgres-backed state/trace. Operates in degraded mode if
// prompt-manager is unavailable (no eligible skills → loop terminates early).
func NewExecutionOrchestratorDefault(profileRepo ProfileRepository, db *sql.DB, projectRoot string) *ExecutionOrchestrator {
	promptEnhancer := NewPromptEnhancer()
	catalog := NewPromptLoaderCatalog(promptEnhancer.GetPromptLoader())

	o := NewExecutionOrchestrator(
		NewExecutionStateManager(db),
		profileRepo,
		&findings.TestGenieRunner{ProjectRoot: projectRoot},
		catalog,
		promptEnhancer,
		NewMetricsCollector(projectRoot),
		NewTraceStore(db),
		effectiveness.NewPostgresStore(db),
	)
	// Wire the P2 DTV read seam. The client fails open (a DTV outage yields
	// UNKNOWN fitness ⇒ allow-all + uniform prior), so this never risks the loop.
	o.SetFitnessProvider(dtv.NewClient(0))
	return o
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
	effectiveness    effectiveness.Store
	terminator       *Terminator

	// pendingCost holds the token cost of the most recent agent run per task,
	// recorded by the queue via RecordRunCost and consumed once at the next
	// EvaluateIteration (the run executes out-of-band, so its cost arrives via
	// this seam rather than as a MEASURE return value).
	costMu      sync.Mutex
	pendingCost map[string]RunCost

	// fitnessProvider is the P2 DTV read seam. Nil ⇒ DTV disabled (pure P1:
	// uniform prior, allow-all). Production wires dtv.Client; everything below it
	// fails open, so a DTV outage degrades to P1 rather than blocking the loop.
	fitnessProvider dtv.SkillFitnessProvider
	// snapMu guards the per-task fitness snapshots and the degraded-log set.
	snapMu         sync.Mutex
	fitnessSnaps   map[string]*taskFitness
	degradedLogged map[string]bool
}

// taskFitness is a task's cached DTV fitness snapshot plus the iteration it was
// refreshed at (for TTL) and whether that capture was degraded (fail-open).
type taskFitness struct {
	snap            FitnessSnapshot
	refreshedAtIter int
	degraded        bool
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
	effectivenessStore effectiveness.Store,
) *ExecutionOrchestrator {
	return &ExecutionOrchestrator{
		stateManager:     stateManager,
		profileService:   profileService,
		auditRunner:      auditRunner,
		catalog:          catalog,
		promptEnhancer:   promptEnhancer,
		metricsCollector: metricsCollector,
		traceStore:       traceStore,
		effectiveness:    effectivenessStore,
		terminator:       NewTerminator(),
		pendingCost:      make(map[string]RunCost),
		fitnessSnaps:     make(map[string]*taskFitness),
		degradedLogged:   make(map[string]bool),
	}
}

// SetFitnessProvider wires (or clears) the P2 DTV fitness seam. nil disables DTV
// (the controller runs pure P1). Returns the orchestrator for chaining.
func (o *ExecutionOrchestrator) SetFitnessProvider(p dtv.SkillFitnessProvider) *ExecutionOrchestrator {
	o.fitnessProvider = p
	return o
}

// RecordRunCost stashes the token cost of the agent run that just completed for a
// task. The next EvaluateIteration consumes it for credit assignment. Recording a
// zero/unknown cost is a no-op (the run's cost stays unknown rather than being
// recorded as free).
func (o *ExecutionOrchestrator) RecordRunCost(taskID string, cost RunCost) {
	if o == nil || taskID == "" || !cost.Known() {
		return
	}
	o.costMu.Lock()
	defer o.costMu.Unlock()
	o.pendingCost[taskID] = cost
}

// takeRunCost pops the recorded run cost for a task (zero/unknown if none).
func (o *ExecutionOrchestrator) takeRunCost(taskID string) RunCost {
	o.costMu.Lock()
	defer o.costMu.Unlock()
	cost := o.pendingCost[taskID]
	delete(o.pendingCost, taskID)
	return cost
}

func (o *ExecutionOrchestrator) resolver() *skillmap.Resolver {
	return skillmap.NewResolver(o.catalog)
}

// dtvSelectionInfo carries the DTV context a single SELECT used, so the trace can
// surface the chosen skill's verdict, the prior provenance, and exclusions.
type dtvSelectionInfo struct {
	active   bool
	degraded bool
	snapshot FitnessSnapshot
	prior    PriorProvider
}

// runSelect performs one SELECT for an upcoming iteration and returns both the
// decision and the DTV context behind it. With an effectiveness ledger wired it
// is the v1 reduction-per-token bandit (deterministic exploration seeded by
// task+iteration, the Layer-1 DTV eligibility gate, the DTV trust/cost prior, and
// the hysteresis cooldown); otherwise pure greedy. DTV is consulted only via the
// per-task snapshot — never synchronously in the candidate-ranking hot path.
func (o *ExecutionOrchestrator) runSelect(state *ProfileExecutionState, profile *AutoSteerProfile, iteration int) (Selection, dtvSelectionInfo) {
	if o.effectiveness == nil {
		return NewSelector(o.resolver()).SelectNextSkill(state.Findings, profile), dtvSelectionInfo{}
	}
	prior, filter, info := o.dtvSeams(state, profile)
	sel := NewSelectorWithConfig(SelectorConfig{
		Resolver:      o.resolver(),
		Effectiveness: o.effectiveness,
		Prior:         prior,
		ShrinkageK:    effectiveness.DefaultShrinkageK,
		Epsilon:       explorationEpsilon(iteration),
		Seed:          explorationSeed(state.TaskID, iteration),
		Filter:        filter,
		Cooldown:      cooldownSkills(state, profile.Budget.skillCooldown(), iteration),
	}).SelectNextSkill(state.Findings, profile)
	return sel, info
}

// dtvSeams returns the prior provider and eligibility filter for a selection. No
// fitness provider (or DTV unreachable, handled inside fitnessSnapshot) ⇒ exact
// P1 behavior: uniform prior + allow-all.
func (o *ExecutionOrchestrator) dtvSeams(state *ProfileExecutionState, profile *AutoSteerProfile) (PriorProvider, EligibilityFilter, dtvSelectionInfo) {
	if o.fitnessProvider == nil {
		return UniformPrior{Value: coldStartPrior}, AllowAllFilter{}, dtvSelectionInfo{}
	}
	snap, degraded := o.fitnessSnapshot(state, profile)
	prior := NewDTVPriorProvider(snap, profile.dtvPriorConfig())
	var filter EligibilityFilter = AllowAllFilter{}
	if profile.dtvGateEnabled() {
		filter = NewDTVEligibilityFilter(snap)
	}
	return prior, filter, dtvSelectionInfo{active: true, degraded: degraded, snapshot: snap, prior: prior}
}

// fitnessSnapshot returns the task's DTV fitness snapshot, fetching it on first
// use and refreshing it every dtv.refresh_iters iterations (TTL). All DTV I/O is
// here, off the candidate-ranking hot path. A degraded fetch (any provider
// error) is logged once per task and still returns a usable fail-open snapshot.
func (o *ExecutionOrchestrator) fitnessSnapshot(state *ProfileExecutionState, profile *AutoSteerProfile) (FitnessSnapshot, bool) {
	o.snapMu.Lock()
	defer o.snapMu.Unlock()

	tf := o.fitnessSnaps[state.TaskID]
	ttl := profile.dtvRefreshIters()
	if tf == nil || state.Iteration-tf.refreshedAtIter >= ttl {
		snap, degraded := o.fetchFitness(context.Background(), profile)
		tf = &taskFitness{snap: snap, refreshedAtIter: state.Iteration, degraded: degraded}
		o.fitnessSnaps[state.TaskID] = tf
		if degraded && !o.degradedLogged[state.TaskID] {
			o.degradedLogged[state.TaskID] = true
			log.Printf("Auto Steer: DTV fitness unavailable for task %s — degrading to P1 selection (uniform prior, allow-all) until DTV recovers", state.TaskID)
		}
	}
	return tf.snap, tf.degraded
}

// fetchFitness pulls fitness for every allowed skill. Each read fails open: a
// per-skill error yields an UNKNOWN fitness and flips the degraded flag.
func (o *ExecutionOrchestrator) fetchFitness(ctx context.Context, profile *AutoSteerProfile) (FitnessSnapshot, bool) {
	fits := make(map[string]dtv.Fitness, len(profile.AllowedSkills))
	degraded := false
	for _, skill := range profile.AllowedSkills {
		f, err := o.fitnessProvider.Fitness(ctx, skill)
		if err != nil {
			degraded = true
		}
		fits[skill] = f
	}
	return NewFitnessSnapshot(fits), degraded
}

// cooldownSkills returns skills deprioritized for the upcoming iteration because
// their most recent run regressed their own target dimension within the last C
// iterations (hysteresis). Returns nil when nothing is cooling.
func cooldownSkills(state *ProfileExecutionState, c, upcoming int) map[string]bool {
	if c <= 0 || len(state.Trace) == 0 {
		return nil
	}
	lastRun := make(map[string]DecisionTraceEntry)
	for _, e := range state.Trace {
		if e.ChosenSkill == "" {
			continue
		}
		if prev, ok := lastRun[e.ChosenSkill]; !ok || e.Iteration > prev.Iteration {
			lastRun[e.ChosenSkill] = e
		}
	}
	cooling := make(map[string]bool)
	for skill, e := range lastRun {
		if regressedTarget(e) && upcoming-e.Iteration <= c {
			cooling[skill] = true
		}
	}
	if len(cooling) == 0 {
		return nil
	}
	return cooling
}

// regressedTarget reports whether an iteration regressed (or failed to net-close)
// its own target dimension: it introduced at least one finding there and did not
// close more than it introduced.
func regressedTarget(e DecisionTraceEntry) bool {
	intro := e.IntroducedByDimension[e.HeaviestDimension]
	closed := e.ClosedByDimension[e.HeaviestDimension]
	return intro > 0 && intro >= closed
}

// explorationEpsilon decays the exploration probability by iteration.
func explorationEpsilon(iteration int) float64 {
	if iteration < 0 {
		iteration = 0
	}
	return explorationBaseEpsilon / (1.0 + float64(iteration))
}

// explorationSeed derives a deterministic exploration seed from the task and
// iteration, so a selection is reproducible (required for tests and replay).
func explorationSeed(taskID string, iteration int) uint64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%s:%d", taskID, iteration)
	return h.Sum64()
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
	sel, dtvInfo := o.runSelect(state, profile, state.Iteration+1)
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
	annotateDTVTrace(&entry, sel, dtvInfo)
	state.Trace = append(state.Trace, entry)
	if err := o.traceStore.Append(state.TaskID, state.ProfileID, scenarioName, entry); err != nil {
		log.Printf("Auto Steer: failed to persist decision trace (non-fatal): %v", err)
	}
}

// annotateDTVTrace records the DTV provenance of a selection on its trace entry:
// the chosen skill's fitness verdict, the seeded prior, any Layer-1 exclusions
// (skill → reason), the all-red gate-override flag, and whether the snapshot was
// captured in degraded (fail-open) mode. No-op when DTV is inactive (pure P1).
func annotateDTVTrace(entry *DecisionTraceEntry, sel Selection, info dtvSelectionInfo) {
	if !info.active {
		return
	}
	entry.DTVDegraded = info.degraded
	entry.DTVGateOverride = sel.GateOverride
	if sel.SkillID != "" {
		entry.DTVVerdict = info.snapshot.Get(sel.SkillID).Verdict.String()
		if info.prior != nil {
			entry.DTVPrior = info.prior.Prior(sel.SkillID, sel.Dimension)
		}
	}
	if len(sel.ExcludedSkills) > 0 {
		entry.DTVExcluded = make(map[string]string, len(sel.ExcludedSkills))
		for _, id := range sel.ExcludedSkills {
			entry.DTVExcluded[id] = "dtv:" + info.snapshot.Get(id).Verdict.String()
		}
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

	// MEASURE — re-audit, diff against the prior open set, and merge.
	cost := o.takeRunCost(taskID)
	prevFindings := state.Findings
	prevScore := lastScore(state.ScoreHistory, prevFindings.TotalScore)
	newFS, err := o.reaudit(context.Background(), scenarioName, profile, state)
	if err != nil {
		return nil, fmt.Errorf("re-audit failed: %w", err)
	}
	scoreAfter := newFS.TotalScore
	realizedDelta := prevScore - scoreAfter // positive = improvement

	// LEARN — attribute the per-dimension findings flow to the skill that ran.
	diff := findings.DiffStates(prevFindings, newFS)
	o.recordRealized(state, scoreAfter, realizedDelta, cost, diff, profile.Budget.RegressionVeto)
	o.assignCredit(state, diff, cost)
	state.Findings = newFS
	state.ScoreHistory = append(state.ScoreHistory, scoreAfter)
	state.Metrics = o.collectMetrics(scenarioName, state.Iteration)

	// TERMINATE — global, gradient-based + Layer-2 thrashing defenses.
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
// iteration whose skill just ran) — score, realized delta, token cost, and the
// per-dimension findings flow — in both the live state and the durable store.
func (o *ExecutionOrchestrator) recordRealized(state *ProfileExecutionState, scoreAfter, realizedDelta float64, cost RunCost, diff findings.Diff, regressionVeto bool) {
	n := len(state.Trace)
	if n == 0 {
		return
	}
	e := &state.Trace[n-1]
	e.ScoreAfter = scoreAfter
	e.RealizedDelta = realizedDelta
	e.TokensUsed = cost.TotalTokens
	e.ClosedByDimension = dimCountsToStringMap(diff.ClosedByDimension)
	e.IntroducedByDimension = dimCountsToStringMap(diff.IntroducedByDimension)
	e.Regressed = realizedDelta < 0 // net weighted score went up
	if regressionVeto && e.Regressed {
		e.VetoApplied = true
		log.Printf("Auto Steer: REGRESSION VETO task %s iteration %d — net weighted score rose by %.2f; flagged and cooled (no auto-rollback in P1)",
			state.TaskID, e.Iteration, -realizedDelta)
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

// assignCredit attributes the iteration's findings flow to the skill that ran,
// updating the effectiveness ledger. The run's target dimension is the heaviest
// dimension chosen at SELECT (recorded on the current trace entry); collateral
// closed/introduced in other dimensions are recorded too, so a skill that fixes
// one dimension while breaking another earns the debt.
func (o *ExecutionOrchestrator) assignCredit(state *ProfileExecutionState, diff findings.Diff, cost RunCost) {
	if o.effectiveness == nil || state.CurrentSkill == "" {
		return
	}
	n := len(state.Trace)
	if n == 0 {
		return
	}
	ev := effectiveness.CreditEvent{
		SkillID:               state.CurrentSkill,
		TargetDimension:       dimensions.Dimension(state.Trace[n-1].HeaviestDimension),
		ClosedByDimension:     diff.ClosedByDimension,
		IntroducedByDimension: diff.IntroducedByDimension,
		Tokens:                cost.TotalTokens,
	}
	if err := o.effectiveness.Record(ev); err != nil {
		log.Printf("Auto Steer: failed to record effectiveness credit (non-fatal): %v", err)
	}
}

// dimCountsToStringMap converts a per-dimension count map into the string-keyed
// shape stored in the decision trace JSON (nil when empty).
func dimCountsToStringMap(m map[dimensions.Dimension]int) map[string]int {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]int, len(m))
	for d, n := range m {
		out[string(d)] = n
	}
	return out
}

// GetExecutionState retrieves the current controller state for a task.
func (o *ExecutionOrchestrator) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	return o.stateManager.Get(taskID)
}

// Effectiveness returns the effectiveness-ledger rows, optionally filtered by
// skill and/or dimension (empty = no filter). It is the read side of the
// operator's "which skills actually work" view. Returns nil when no ledger is
// wired.
func (o *ExecutionOrchestrator) Effectiveness(skillID string, dim dimensions.Dimension) ([]effectiveness.Stat, error) {
	if o.effectiveness == nil {
		return nil, nil
	}
	return o.effectiveness.List(skillID, dim)
}

// EffectivenessPrior exposes the cold-start prior and shrinkage constant the
// production selector uses, so read surfaces can render the same expected
// efficacy the bandit would compute.
func (o *ExecutionOrchestrator) EffectivenessPrior() (prior, k float64) {
	return coldStartPrior, effectiveness.DefaultShrinkageK
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

// DeleteExecutionState removes any active execution state for a task, including
// its cached DTV fitness snapshot.
func (o *ExecutionOrchestrator) DeleteExecutionState(taskID string) error {
	o.snapMu.Lock()
	delete(o.fitnessSnaps, taskID)
	delete(o.degradedLogged, taskID)
	o.snapMu.Unlock()
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
