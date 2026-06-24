package autosteer

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ecosystem-manager/api/pkg/completeness"
	"github.com/ecosystem-manager/api/pkg/findings"
)

// SteerMode defines the different improvement dimensions agents can focus on.
// These remain prompt-routing labels (which skill family to render); the
// controller's dimension vocabulary lives in the shared maturity-go/dimensions
// package (packages/maturity-go).
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

// Budget bounds the controller loop (backstop + cost control).
type Budget struct {
	// MaxIterations is the hard iteration cap (the loop's backstop / "bounded
	// timeout" — the controller never runs more agent passes than this).
	MaxIterations int `json:"max_iterations"`
	// DiminishingReturnsFloor is the minimum mean weighted-score improvement per
	// iteration (over the trailing window) below which the loop stops.
	DiminishingReturnsFloor float64 `json:"diminishing_returns_floor"`
	// ReauditCadence controls re-audit cost. 0 = targeted re-audit each
	// iteration (only the chosen skill's dimensions); N>0 = run the full preset
	// every N iterations and a targeted audit otherwise.
	ReauditCadence int `json:"reaudit_cadence,omitempty"`
}

// BaselinePromote cadence modes (Baseline Modes P6). The value selects WHEN an
// engagement's shadow is promoted to live during an autosteer run.
const (
	// BaselinePromoteEndOfEngagement promotes once, when the controller's loop
	// terminates with the objective met — the safe default. A non-objective-met
	// stop (budget/diminishing/thrashing) abandons the shadow instead.
	BaselinePromoteEndOfEngagement = "end_of_engagement"
	// BaselinePromoteCheckpointOnGreen promotes early — the first time a cadence
	// checkpoint observes an already-met objective — banking the validated win
	// rather than risking a later regression.
	BaselinePromoteCheckpointOnGreen = "checkpoint_on_green"
)

// BaselinePromoteObjective is the optional Baseline Modes block of a profile
// (P6). Absent (nil) or Enabled=false ⇒ the autosteer loop edits the scenario in
// place exactly as before — no engagement, no shadow, no promote — so existing
// profiles are unperturbed until they opt in. When enabled the orchestrator
// fronts the run with `git-control-tower baseline start` (which runs the mode
// decision tree and takes a git-free restore point), routes the coding agent's
// nested CLI calls to the shadow instance via VROOLI_SHADOW_SCENARIOS, and
// promotes (shadow→live, green) or abandons (tear down the shadow, not green)
// per Mode at the controller's terminal decision.
type BaselinePromoteObjective struct {
	// Enabled turns Baseline Modes engagement on for this profile. Default-off.
	Enabled bool `json:"enabled"`
	// Mode selects the promote cadence. Empty ⇒ end_of_engagement.
	Mode string `json:"mode,omitempty"`
	// CadenceIter throttles checkpoint_on_green: only consider an early promote
	// every CadenceIter controller iterations. <=0 ⇒ consider every iteration.
	// Ignored by end_of_engagement.
	CadenceIter int `json:"cadence_iter,omitempty"`
}

// BaselinePromoteEnabled reports whether Baseline Modes engagement is on. The
// queue's engagement layer (a separate package) consumes it, so it is exported.
func (p *AutoSteerProfile) BaselinePromoteEnabled() bool {
	return p != nil && p.BaselinePromote != nil && p.BaselinePromote.Enabled
}

// BaselinePromoteMode returns the effective promote cadence (default
// end_of_engagement when unset).
func (p *AutoSteerProfile) BaselinePromoteMode() string {
	if p == nil || p.BaselinePromote == nil || strings.TrimSpace(p.BaselinePromote.Mode) == "" {
		return BaselinePromoteEndOfEngagement
	}
	return p.BaselinePromote.Mode
}

// BaselinePromoteCadence returns the effective checkpoint cadence (<=0 ⇒ every
// iteration).
func (p *AutoSteerProfile) BaselinePromoteCadence() int {
	if p == nil || p.BaselinePromote == nil {
		return 0
	}
	return p.BaselinePromote.CadenceIter
}

// AutoSteerProfile is the controller's objective function for an improvement
// run. (The type name is retained across the API/CLI/UI surfaces; its shape is
// the greenfield objective model.)
type AutoSteerProfile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Objective   Objective `json:"objective"`
	// AllowedSkills is an optional restriction mask over catalog-derived
	// eligibility. Empty means "derive from the objective's weighted dimensions."
	AllowedSkills []string `json:"allowed_skills,omitempty"`
	// DeniedSkills subtracts specific catalog-derived skills from eligibility.
	DeniedSkills []string `json:"denied_skills,omitempty"`
	Budget       Budget   `json:"budget"`
	// BaselinePromote is the optional Baseline Modes engagement block (P6).
	// Absent/disabled ⇒ the loop edits the scenario in place (no shadow/promote).
	BaselinePromote *BaselinePromoteObjective `json:"baseline_promote,omitempty"`
	// AuditPreset is the test-genie preset used for full audits (the initial
	// diagnose and the termination gate). Defaults to "comprehensive".
	AuditPreset string    `json:"audit_preset,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags"`
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
	// GamingCause records the anti-gaming classifier verdict for this iteration
	// when it detected gaming-shaped work (e.g. "gamed:test-weakening,suppression")
	// or flagged it for review ("flagged-for-review"). Empty ⇒ clean. A "gamed:"
	// verdict blocks the shadow→live promote (see ExecutionOrchestrator.RunGamed).
	GamingCause string `json:"gaming_cause,omitempty"`
	// HaltReason is set on the final iteration when the controller stopped,
	// capturing why (objective_met, budget_exhausted, diminishing_returns, …).
	HaltReason string `json:"halt_reason,omitempty"`
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
	// Completeness is the latest measurement fetched from the
	// scenario-completeness-scoring authority (rung, build, operational targets).
	// It replaces the deleted self-collected MetricsSnapshot.
	Completeness completeness.Score `json:"completeness"`
	StartedAt    time.Time          `json:"started_at"`
	LastUpdated  time.Time          `json:"last_updated"`
}

// IterationEvaluation is the result of a controller MEASURE+TERMINATE step.
type IterationEvaluation struct {
	ShouldStop bool   `json:"should_stop"`
	Reason     string `json:"reason,omitempty"`
	// ChosenSkill is the skill selected for the next iteration when the loop
	// continues (empty when stopping).
	ChosenSkill string `json:"chosen_skill,omitempty"`
	// ObjectiveMet reports whether the profile's objective is satisfied by the
	// state measured this iteration. The Baseline Modes checkpoint_on_green cadence
	// consumes it to promote a validated win early. (Under the greedy terminator,
	// objective-met also triggers a stop, so on the continue path this is normally
	// false; the field is retained for the checkpoint cadence contract.)
	ObjectiveMet bool `json:"objective_met,omitempty"`
	// Iteration is the controller iteration this evaluation reflects, used by the
	// checkpoint cadence to throttle how often an early promote is considered.
	Iteration int `json:"iteration,omitempty"`
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
