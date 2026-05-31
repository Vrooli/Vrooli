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
	// AuditPreset is the test-genie preset used for full audits (the initial
	// diagnose and the termination gate). Defaults to "comprehensive".
	AuditPreset string    `json:"audit_preset,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags"`
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
