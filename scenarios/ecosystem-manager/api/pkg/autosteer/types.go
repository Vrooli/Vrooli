package autosteer

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/findings"
)

// SteerMode defines the different improvement dimensions agents can focus on.
// These remain prompt-routing labels (which skill family to render); the
// controller's dimension vocabulary lives in pkg/dimensions.
type SteerMode string

const (
	ModeProgress    SteerMode = "progress"    // Default: operational target completion
	ModeUX          SteerMode = "ux"          // Accessibility, user flows, design, responsiveness
	ModeRefactor    SteerMode = "refactor"    // Code quality, standards, complexity reduction
	ModeTest        SteerMode = "test"        // Coverage, edge cases, test quality
	ModeExplore     SteerMode = "explore"     // Experimentation, novel approaches
	ModePolish      SteerMode = "polish"      // Final touches, error messages, loading states
	ModePerformance SteerMode = "performance" // Profiling, optimization, caching
	ModeSecurity    SteerMode = "security"    // Vulnerability scanning, input validation
)

var (
	builtInSteerModes = []SteerMode{
		ModeProgress,
		ModeUX,
		ModeRefactor,
		ModeTest,
		ModeExplore,
		ModePolish,
		ModePerformance,
		ModeSecurity,
	}

	builtInSteerModeSet = map[SteerMode]struct{}{
		ModeProgress:    {},
		ModeUX:          {},
		ModeRefactor:    {},
		ModeTest:        {},
		ModeExplore:     {},
		ModePolish:      {},
		ModePerformance: {},
		ModeSecurity:    {},
	}

	steerModeRegistry = &modeRegistry{
		custom: make(map[SteerMode]struct{}),
	}
)

type modeRegistry struct {
	mu     sync.RWMutex
	custom map[SteerMode]struct{}
}

func normalizeSteerMode(mode SteerMode) SteerMode {
	return SteerMode(strings.TrimSpace(strings.ToLower(string(mode))))
}

// Normalized returns a lowercase, trimmed representation of the mode.
func (m SteerMode) Normalized() SteerMode {
	return normalizeSteerMode(m)
}

func (r *modeRegistry) has(mode SteerMode) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.custom[mode]
	return ok
}

func (r *modeRegistry) add(mode SteerMode) {
	if mode == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.custom[mode] = struct{}{}
}

func (r *modeRegistry) listCustom() []SteerMode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	modes := make([]SteerMode, 0, len(r.custom))
	for mode := range r.custom {
		modes = append(modes, mode)
	}
	return modes
}

// RegisterSteerModes allows custom modes to be registered at runtime (e.g., from prompt-manager skills).
func RegisterSteerModes(modes ...SteerMode) {
	for _, mode := range modes {
		normalized := normalizeSteerMode(mode)
		if normalized == "" {
			continue
		}
		steerModeRegistry.add(normalized)
	}
}

// AllowedSteerModes returns all built-in and registered custom modes, sorted for stability.
func AllowedSteerModes() []SteerMode {
	customModes := steerModeRegistry.listCustom()

	modes := make([]SteerMode, 0, len(builtInSteerModes)+len(customModes))
	modes = append(modes, builtInSteerModes...)

	for _, mode := range customModes {
		if _, exists := builtInSteerModeSet[mode]; !exists {
			modes = append(modes, mode)
		}
	}

	sort.Slice(modes, func(i, j int) bool {
		return modes[i] < modes[j]
	})

	return modes
}

// IsValid checks if a SteerMode is valid. Custom modes come from prompt-manager skills.
func (m SteerMode) IsValid() bool {
	normalized := normalizeSteerMode(m)
	if normalized == "" {
		return false
	}

	if _, ok := builtInSteerModeSet[normalized]; ok {
		return true
	}

	return steerModeRegistry.has(normalized)
}

// ─────────────────────────────────────────────────────────────────────────────
// Objective-function profiles (the closed-loop controller model).
//
// A profile is no longer a script (ordered phase list). It is an OBJECTIVE
// FUNCTION: how to weight the open-findings vector, what "done" means, which
// skills are eligible, and the iteration budget. The controller derives the
// path from the target's measured state — see docs/concepts/CONTROL-MODEL.md.
// ─────────────────────────────────────────────────────────────────────────────

// Objective is the weighting + target definition the controller optimizes
// against. Dimension weights bias selection toward what the profile cares about
// most; targets define when the objective is met.
type Objective struct {
	// DimensionWeights biases the weighted-score of each finding dimension.
	// A dimension absent from the map defaults to weight 1.0.
	DimensionWeights map[string]float64 `json:"dimension_weights"`
	Targets          ObjectiveTargets   `json:"targets"`
}

// ObjectiveTargets define when the controller may declare the objective met.
type ObjectiveTargets struct {
	// MaxOpenSeverity is the highest finding severity tolerated at completion.
	// Findings strictly above this severity keep the loop running. One of
	// "info", "warning", "error", "blocker" (case-insensitive). Empty means
	// "no open findings of any severity".
	MaxOpenSeverity string `json:"max_open_severity,omitempty"`
	// OperationalTargetsPct is the minimum operational-target completion (0-100)
	// required at completion, measured from the gap-metric collectors. Zero
	// disables the gate.
	OperationalTargetsPct float64 `json:"operational_targets_pct,omitempty"`
}

// Budget bounds the controller loop (Layer-3 backstop + cost control).
type Budget struct {
	// MaxIterations is the hard iteration cap (Layer-3 thrash backstop).
	MaxIterations int `json:"max_iterations"`
	// DiminishingReturnsFloor is the minimum mean weighted-score improvement per
	// iteration (over the trailing window) below which the loop stops.
	DiminishingReturnsFloor float64 `json:"diminishing_returns_floor"`
	// ReauditCadence controls re-audit cost. 0 = targeted re-audit each
	// iteration (only the chosen skill's dimensions); N>0 = run the full preset
	// every N iterations and a targeted audit otherwise.
	ReauditCadence int `json:"reaudit_cadence,omitempty"`

	// Layer-2 runtime thrashing-defense knobs (all default to conservative
	// non-zero values when left at 0; see CONTROL-MODEL.md "Termination").
	//
	// CycleWindow (K) is how many prior iterations the fingerprint cycle detector
	// scans for a recurrence of the current open-findings set.
	CycleWindow int `json:"cycle_window,omitempty"`
	// NetProgressWindow (W) is the trailing iteration count over which net
	// findings flow (closed − introduced) must clear NetProgressFloor.
	NetProgressWindow int `json:"net_progress_window,omitempty"`
	// NetProgressFloor is the minimum |net findings flow| over the window below
	// which the loop is judged to be churning without net gain.
	NetProgressFloor float64 `json:"net_progress_floor,omitempty"`
	// SkillCooldown (C) is how many iterations a skill that regressed its own
	// target dimension is deprioritized before it is eligible again.
	SkillCooldown int `json:"skill_cooldown,omitempty"`
	// RegressionVeto, when true, prominently flags (and applies cooldown to) any
	// iteration whose net weighted score went up. P1 records the veto decision;
	// speculative per-iteration rollback is out of scope.
	RegressionVeto bool `json:"regression_veto,omitempty"`
}

// Layer-2 defaults: conservative so the cycle detector needs an exact repeat and
// the net-progress window is wide. Applied when a profile leaves a knob at 0.
const (
	defaultCycleWindow       = 4
	defaultNetProgressWindow = 3
	defaultNetProgressFloor  = 0.0
	defaultSkillCooldown     = 2
)

// cycleWindow returns the effective K.
func (b Budget) cycleWindow() int {
	if b.CycleWindow <= 0 {
		return defaultCycleWindow
	}
	return b.CycleWindow
}

// netProgressWindow returns the effective W.
func (b Budget) netProgressWindow() int {
	if b.NetProgressWindow <= 0 {
		return defaultNetProgressWindow
	}
	return b.NetProgressWindow
}

// netProgressFloor returns the effective net-progress floor.
func (b Budget) netProgressFloor() float64 {
	if b.NetProgressFloor < 0 {
		return defaultNetProgressFloor
	}
	return b.NetProgressFloor
}

// skillCooldown returns the effective C.
func (b Budget) skillCooldown() int {
	if b.SkillCooldown <= 0 {
		return defaultSkillCooldown
	}
	return b.SkillCooldown
}

// DTVObjective is the optional development-toolchain-validator block of a
// profile's objective function (P2). It tunes how DTV fitness gates and seeds
// selection. Absent (nil) means "DTV defaults": gate on red, seed priors — which
// still degrades to exact P1 behavior whenever DTV has no data (fail-open).
type DTVObjective struct {
	// GateEnabled toggles the Layer-1 eligibility gate (deny DTV-red skills).
	// nil ⇒ true (gate on). Set false to keep priors but never hard-gate.
	GateEnabled *bool `json:"gate_enabled,omitempty"`
	// PriorWeight scales the DTV trust/cost prior. 0/omitted ⇒ default weight;
	// the bandit blend washes the prior out with live evidence regardless.
	PriorWeight float64 `json:"prior_weight,omitempty"`
	// TrustFloor is a pass-rate floor (0–1): below it the prior is 0 (a
	// low-trust skill gets no cold-start head start). 0 ⇒ no floor.
	TrustFloor float64 `json:"trust_floor,omitempty"`
	// RefreshIters is the snapshot TTL in controller iterations. <=0 ⇒ default.
	RefreshIters int `json:"refresh_iters,omitempty"`
}

// AutoSteerProfile is the controller's objective function for an improvement
// run. (The type name is retained across the API/CLI/UI surfaces; its shape is
// the greenfield objective model.)
type AutoSteerProfile struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Objective     Objective `json:"objective"`
	AllowedSkills []string  `json:"allowed_skills"`
	Budget        Budget    `json:"budget"`
	// DTV is the optional development-toolchain-validator objective-block (P2).
	DTV *DTVObjective `json:"dtv,omitempty"`
	// AuditPreset is the test-genie preset used for full audits (the initial
	// diagnose and the termination gate). Defaults to "comprehensive".
	AuditPreset string    `json:"audit_preset,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags"`
}

// defaultDTVRefreshIters is the snapshot TTL (in controller iterations) when a
// profile leaves dtv.refresh_iters unset: cheap enough to track skill changes,
// infrequent enough to keep DTV off the hot SELECT path.
const defaultDTVRefreshIters = 5

// dtvGateEnabled reports whether the Layer-1 eligibility gate is on (default
// true when the dtv block or its flag is absent).
func (p *AutoSteerProfile) dtvGateEnabled() bool {
	if p == nil || p.DTV == nil || p.DTV.GateEnabled == nil {
		return true
	}
	return *p.DTV.GateEnabled
}

// dtvRefreshIters returns the effective snapshot TTL in iterations.
func (p *AutoSteerProfile) dtvRefreshIters() int {
	if p == nil || p.DTV == nil || p.DTV.RefreshIters <= 0 {
		return defaultDTVRefreshIters
	}
	return p.DTV.RefreshIters
}

// dtvPriorConfig maps the profile's dtv block onto the prior-mapping config
// (zero fields fall back to package defaults via withDefaults).
func (p *AutoSteerProfile) dtvPriorConfig() DTVPriorConfig {
	cfg := DTVPriorConfig{}
	if p != nil && p.DTV != nil {
		cfg.Weight = p.DTV.PriorWeight
		cfg.TrustFloor = p.DTV.TrustFloor
	}
	return cfg
}

// MetricsSnapshot captures gap-metric measurements at a point in time. In the
// controller model these are NOT the primary state (that is the findings
// vector); they are retained as gap measurements for the objective's
// operational-targets target and for history/analytics.
type MetricsSnapshot struct {
	Timestamp time.Time `json:"timestamp"`

	// TotalLoops is the global controller iteration counter.
	PhaseLoops                   int     `json:"phase_loops"`
	TotalLoops                   int     `json:"total_loops"`
	BuildStatus                  int     `json:"build_status"` // 0 = failing, 1 = passing
	OperationalTargetsTotal      int     `json:"operational_targets_total"`
	OperationalTargetsPassing    int     `json:"operational_targets_passing"`
	OperationalTargetsPercentage float64 `json:"operational_targets_percentage"`

	// Mode-specific metrics (optional, populated based on scenario capabilities)
	UX          *UXMetrics          `json:"ux,omitempty"`
	Refactor    *RefactorMetrics    `json:"refactor,omitempty"`
	Test        *TestMetrics        `json:"test,omitempty"`
	Performance *PerformanceMetrics `json:"performance,omitempty"`
	Security    *SecurityMetrics    `json:"security,omitempty"`
}

// UXMetrics captures user experience quality metrics
type UXMetrics struct {
	AccessibilityScore    float64 `json:"accessibility_score"`     // 0-100
	UITestCoverage        float64 `json:"ui_test_coverage"`        // 0-100
	ResponsiveBreakpoints int     `json:"responsive_breakpoints"`  // Count of breakpoints
	UserFlowsImplemented  int     `json:"user_flows_implemented"`  // Count of complete user flows
	LoadingStatesCount    int     `json:"loading_states_count"`    // Count of loading states
	ErrorHandlingCoverage float64 `json:"error_handling_coverage"` // 0-100
}

// RefactorMetrics captures code quality metrics
type RefactorMetrics struct {
	CyclomaticComplexityAvg float64 `json:"cyclomatic_complexity_avg"` // Average complexity
	DuplicationPercentage   float64 `json:"duplication_percentage"`    // 0-100
	StandardsViolations     int     `json:"standards_violations"`      // Count of violations
	TidinessScore           float64 `json:"tidiness_score"`            // 0-100 from tidiness-manager
	TechDebtItems           int     `json:"tech_debt_items"`           // Count of tech debt items
}

// TestMetrics captures testing quality metrics
type TestMetrics struct {
	UnitTestCoverage        float64 `json:"unit_test_coverage"`        // 0-100
	IntegrationTestCoverage float64 `json:"integration_test_coverage"` // 0-100
	UITestCoverage          float64 `json:"ui_test_coverage"`          // 0-100
	EdgeCasesCovered        int     `json:"edge_cases_covered"`        // Count of edge cases
	FlakyTests              int     `json:"flaky_tests"`               // Count of flaky tests
	TestQualityScore        float64 `json:"test_quality_score"`        // 0-100
}

// PerformanceMetrics captures performance metrics
type PerformanceMetrics struct {
	BundleSizeKB      float64 `json:"bundle_size_kb"`       // Bundle size in KB
	InitialLoadTimeMS int     `json:"initial_load_time_ms"` // Initial load time in ms
	LCPMS             int     `json:"lcp_ms"`               // Largest Contentful Paint in ms
	FIDMS             int     `json:"fid_ms"`               // First Input Delay in ms
	CLSScore          float64 `json:"cls_score"`            // Cumulative Layout Shift score
}

// SecurityMetrics captures security quality metrics
type SecurityMetrics struct {
	VulnerabilityCount      int     `json:"vulnerability_count"`       // Count of vulnerabilities
	InputValidationCoverage float64 `json:"input_validation_coverage"` // 0-100
	AuthImplementationScore float64 `json:"auth_implementation_score"` // 0-100
	SecurityScanScore       float64 `json:"security_scan_score"`       // 0-100
}

// DecisionTraceEntry records one controller iteration's reasoning so the loop is
// a glass box (see CONTROL-MODEL.md "Transparency"). One entry is appended when
// a skill is selected; ScoreAfter / RealizedDelta are filled after the skill
// runs and the target is re-audited.
type DecisionTraceEntry struct {
	Iteration         int                `json:"iteration"`
	Timestamp         time.Time          `json:"timestamp"`
	DimensionScores   map[string]float64 `json:"dimension_scores"`
	HeaviestDimension string             `json:"heaviest_dimension"`
	ChosenSkill       string             `json:"chosen_skill"`
	Rationale         string             `json:"rationale"`
	Fingerprint       string             `json:"fingerprint"`
	ScoreBefore       float64            `json:"score_before"`
	ScoreAfter        float64            `json:"score_after"`
	RealizedDelta     float64            `json:"realized_delta"`
	// TokensUsed is the agent run's token cost for this iteration (0 = unknown).
	TokensUsed int64 `json:"tokens_used"`
	// ClosedByDimension / IntroducedByDimension are the per-dimension findings
	// flow this iteration produced (filled after MEASURE, by stable finding ID).
	ClosedByDimension     map[string]int `json:"closed_by_dimension,omitempty"`
	IntroducedByDimension map[string]int `json:"introduced_by_dimension,omitempty"`
	// Regressed is true when this iteration's net weighted score went up.
	Regressed bool `json:"regressed,omitempty"`
	// VetoApplied is true when the profile's regression veto fired this iteration.
	VetoApplied bool `json:"veto_applied,omitempty"`
	// HaltReason is set on the final iteration when the controller stopped,
	// capturing why (objective_met, thrashing_cycle, no_net_progress, …).
	HaltReason string `json:"halt_reason,omitempty"`

	// DTV transparency (P2). Populated only when the DTV seam is active.
	// DTVVerdict is the chosen skill's DTV fitness verdict
	// (unknown/green/yellow/red).
	DTVVerdict string `json:"dtv_verdict,omitempty"`
	// DTVPrior is the cold-start trust/cost prior DTV seeded for the chosen skill
	// (0 when DTV had no usable data — i.e. P1 uniform).
	DTVPrior float64 `json:"dtv_prior,omitempty"`
	// DTVExcluded maps each skill the Layer-1 gate denied for the chosen
	// dimension to its reason (e.g. "dtv:red").
	DTVExcluded map[string]string `json:"dtv_excluded,omitempty"`
	// DTVGateOverride is true when the gate would have emptied the chosen
	// dimension and the selector fell back to allow-all to avoid stalling.
	DTVGateOverride bool `json:"dtv_gate_override,omitempty"`
	// DTVDegraded is true when this selection's fitness snapshot was captured
	// while DTV was unreachable (fail-open ⇒ P1 behavior).
	DTVDegraded bool `json:"dtv_degraded,omitempty"`
	// PredictedReduction is the controller's forward estimate (at SELECT time) of
	// the weighted-score reduction the chosen skill will realize this iteration:
	// the bandit's expected reduction-per-token × estimated run tokens (EM-P4).
	// Compared against RealizedDelta in the trace to surface bandit calibration.
	// 0 when no estimate was computable (e.g. greedy cold start, no token model).
	PredictedReduction float64 `json:"predicted_reduction,omitempty"`
	// GateDegradedCause is non-empty when the Layer-1 DTV gate ran in a degraded
	// mode for this iteration under the proceed-cap-flag policy (EM-P2): the
	// controller did not stall, it proceeded with the least-bad skill and halved
	// the remaining iteration budget once. One of GateCauseDTVUnavailable (DTV
	// unreachable ⇒ no fitness data) or GateCauseAllRed (every eligible skill in
	// the chosen dimension was DTV-red). Empty ⇒ healthy gate.
	GateDegradedCause string `json:"gate_degraded_cause,omitempty"`
}

// ProfileExecutionState tracks the live state of an active controller run.
type ProfileExecutionState struct {
	TaskID    string `json:"task_id"`
	ProfileID string `json:"profile_id"`
	// Iteration is the global controller iteration counter (1-based once the
	// first skill has been selected).
	Iteration int `json:"iteration"`
	// CurrentSkill is the skill the controller selected for the next/active run.
	CurrentSkill string `json:"current_skill"`
	// CurrentRationale explains why CurrentSkill was selected.
	CurrentRationale string `json:"current_rationale"`
	// Findings is the latest diagnosed findings vector (the primary state).
	Findings findings.FindingsState `json:"findings"`
	// ScoreHistory is the trailing total weighted-score per iteration, oldest
	// first; drives diminishing-returns termination.
	ScoreHistory []float64 `json:"score_history"`
	// Trace is the per-iteration decision trace.
	Trace []DecisionTraceEntry `json:"trace"`
	// Metrics is the latest gap-metric snapshot (operational targets, build).
	Metrics     MetricsSnapshot `json:"metrics"`
	StartedAt   time.Time       `json:"started_at"`
	LastUpdated time.Time       `json:"last_updated"`
}

// IterationEvaluation is the result of a controller MEASURE+TERMINATE step.
type IterationEvaluation struct {
	ShouldStop bool   `json:"should_stop"`
	Reason     string `json:"reason,omitempty"`
	// ChosenSkill is the skill selected for the next iteration when the loop
	// continues (empty when stopping).
	ChosenSkill string `json:"chosen_skill,omitempty"`
}

// SkillPerformance summarizes one skill's contribution within a completed run
// (derived from the decision trace), used for history/analytics.
type SkillPerformance struct {
	SkillName     string  `json:"skill_name"`
	Iterations    int     `json:"iterations"`
	WeightedDelta float64 `json:"weighted_delta"` // total realized weighted-score reduction
}

// ProfilePerformance represents historical performance data for a completed execution
type ProfilePerformance struct {
	ID              string                   `json:"id"`
	ProfileID       string                   `json:"profile_id"`
	ScenarioName    string                   `json:"scenario_name"`
	ExecutionID     string                   `json:"execution_id"`
	StartMetrics    MetricsSnapshot          `json:"start_metrics"`
	EndMetrics      MetricsSnapshot          `json:"end_metrics"`
	PhaseBreakdown  []SkillPerformance       `json:"phase_breakdown"`
	TotalIterations int                      `json:"total_iterations"`
	TotalDuration   int64                    `json:"total_duration"` // milliseconds
	UserFeedback    *UserFeedback            `json:"user_feedback,omitempty"`
	FeedbackEntries []ExecutionFeedbackEntry `json:"feedback_entries,omitempty"`
	ExecutedAt      time.Time                `json:"executed_at"`
}

// UserFeedback represents user rating and comments for an execution
type UserFeedback struct {
	Rating      int       `json:"rating"` // 1-5
	Comments    string    `json:"comments"`
	SubmittedAt time.Time `json:"submitted_at"`
}

// ExecutionFeedbackEntry represents structured feedback attached to an Auto Steer execution.
type ExecutionFeedbackEntry struct {
	ID              string         `json:"id"`
	Category        string         `json:"category"`
	Severity        string         `json:"severity"`
	SuggestedAction string         `json:"suggested_action,omitempty"`
	Comments        string         `json:"comments,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

// ExecutionFeedbackRequest defines the payload used when submitting structured execution feedback.
type ExecutionFeedbackRequest struct {
	Category        string         `json:"category"`
	Severity        string         `json:"severity"`
	SuggestedAction string         `json:"suggested_action,omitempty"`
	Comments        string         `json:"comments,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}
