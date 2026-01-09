package autosteer

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// SteerMode defines the different improvement dimensions agents can focus on
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
	mu        sync.RWMutex
	custom    map[SteerMode]struct{}
	phasesDir string
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

func (r *modeRegistry) setPhasesDir(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.phasesDir = dir
}

func (r *modeRegistry) getPhasesDir() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.phasesDir
}

// RegisterSteerModes allows custom modes to be registered at runtime (e.g., from prompt files).
func RegisterSteerModes(modes ...SteerMode) {
	for _, mode := range modes {
		normalized := normalizeSteerMode(mode)
		if normalized == "" {
			continue
		}
		steerModeRegistry.add(normalized)
	}
}

// RegisterSteerModesFromDir scans prompts/phases for markdown files and registers them as modes.
// It also records the directory for future lookups when new prompts are added at runtime.
func RegisterSteerModesFromDir(phasesDir string) error {
	if strings.TrimSpace(phasesDir) == "" {
		return nil
	}

	steerModeRegistry.setPhasesDir(phasesDir)

	var modes []SteerMode

	err := filepath.WalkDir(phasesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		name := normalizeSteerMode(SteerMode(strings.TrimSuffix(d.Name(), ".md")))
		if name != "" {
			modes = append(modes, name)
		}
		return nil
	})
	if err != nil {
		return err
	}

	RegisterSteerModes(modes...)
	return nil
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

// IsValid checks if a SteerMode is valid. Custom modes are lazily registered from phase prompts.
func (m SteerMode) IsValid() bool {
	normalized := normalizeSteerMode(m)
	if normalized == "" {
		return false
	}

	if _, ok := builtInSteerModeSet[normalized]; ok {
		return true
	}

	if steerModeRegistry.has(normalized) {
		return true
	}

	if phasesDir := steerModeRegistry.getPhasesDir(); phasesDir != "" {
		promptPath := filepath.Join(phasesDir, string(normalized)+".md")
		if _, err := os.Stat(promptPath); err == nil {
			steerModeRegistry.add(normalized)
			return true
		}
	}

	return false
}

// ConditionType defines the type of condition (simple or compound)
type ConditionType string

const (
	ConditionTypeSimple   ConditionType = "simple"
	ConditionTypeCompound ConditionType = "compound"
)

// ConditionOperator defines comparison operators
type ConditionOperator string

const (
	OpGreaterThan       ConditionOperator = ">"
	OpLessThan          ConditionOperator = "<"
	OpGreaterThanEquals ConditionOperator = ">="
	OpLessThanEquals    ConditionOperator = "<="
	OpEquals            ConditionOperator = "=="
	OpNotEquals         ConditionOperator = "!="
)

// LogicalOperator defines how conditions are combined
type LogicalOperator string

const (
	LogicalAND LogicalOperator = "AND"
	LogicalOR  LogicalOperator = "OR"
)

// StopCondition represents a condition for stopping a phase
// Can be either simple or compound (recursive)
type StopCondition struct {
	Type       ConditionType   `json:"type"`
	Operator   LogicalOperator `json:"operator,omitempty"`   // For compound conditions
	Conditions []StopCondition `json:"conditions,omitempty"` // For compound conditions

	// For simple conditions
	Metric          string            `json:"metric,omitempty"`
	CompareOperator ConditionOperator `json:"compare_operator,omitempty"`
	Value           float64           `json:"value,omitempty"`
}

// SteerPhase represents a single phase in an Auto Steer profile
type SteerPhase struct {
	ID             string          `json:"id"`
	Mode           SteerMode       `json:"mode"`
	StopConditions []StopCondition `json:"stop_conditions"`
	MaxIterations  int             `json:"max_iterations"`
	Description    string          `json:"description,omitempty"`
}

// QualityGateAction defines what happens when a quality gate fails
type QualityGateAction string

const (
	ActionHalt      QualityGateAction = "halt"       // Stop execution completely
	ActionSkipPhase QualityGateAction = "skip_phase" // Skip to next phase
	ActionWarn      QualityGateAction = "warn"       // Log warning but continue
)

// QualityGate represents a validation check between phases
type QualityGate struct {
	Name          string            `json:"name"`
	Condition     StopCondition     `json:"condition"`
	FailureAction QualityGateAction `json:"failure_action"`
	Message       string            `json:"message"`
}

// AutoSteerProfile represents a reusable improvement profile
type AutoSteerProfile struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	Phases       []SteerPhase  `json:"phases"`
	QualityGates []QualityGate `json:"quality_gates"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Tags         []string      `json:"tags"`
}

// MetricsSnapshot captures all metrics at a point in time
type MetricsSnapshot struct {
	Timestamp time.Time `json:"timestamp"`

	// Universal metrics
	// PhaseLoops counts iterations within the current phase (resets when a phase advances)
	PhaseLoops int `json:"phase_loops"`
	// TotalLoops is the global iteration counter across all phases
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

// PhaseExecution represents the execution of a single phase
type PhaseExecution struct {
	PhaseID      string          `json:"phase_id"`
	Mode         SteerMode       `json:"mode"`
	Iterations   int             `json:"iterations"`
	StartMetrics MetricsSnapshot `json:"start_metrics"`
	EndMetrics   MetricsSnapshot `json:"end_metrics"`
	Commits      []string        `json:"commits"` // Git commits made during phase
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	StopReason   string          `json:"stop_reason"` // max_iterations, condition_met, quality_gate_failed
}

// ProfileExecutionState tracks the current state of an active profile execution
type ProfileExecutionState struct {
	TaskID                string           `json:"task_id"`
	ProfileID             string           `json:"profile_id"`
	CurrentPhaseIndex     int              `json:"current_phase_index"`
	CurrentPhaseIteration int              `json:"current_phase_iteration"`
	AutoSteerIteration    int              `json:"auto_steer_iteration"`
	PhaseStartedAt        time.Time        `json:"phase_started_at"`
	PhaseHistory          []PhaseExecution `json:"phase_history"`
	Metrics               MetricsSnapshot  `json:"metrics"`             // Current metrics
	PhaseStartMetrics     MetricsSnapshot  `json:"phase_start_metrics"` // Metrics at start of current phase
	StartedAt             time.Time        `json:"started_at"`
	LastUpdated           time.Time        `json:"last_updated"`
}

// PhasePerformance tracks metrics for a completed phase
type PhasePerformance struct {
	Mode          SteerMode          `json:"mode"`
	Iterations    int                `json:"iterations"`
	MetricDeltas  map[string]float64 `json:"metric_deltas"` // metric_name -> change
	Duration      int64              `json:"duration"`      // milliseconds
	Effectiveness float64            `json:"effectiveness"` // Calculated score
}

// ProfilePerformance represents historical performance data for a completed execution
type ProfilePerformance struct {
	ID              string                   `json:"id"`
	ProfileID       string                   `json:"profile_id"`
	ScenarioName    string                   `json:"scenario_name"`
	ExecutionID     string                   `json:"execution_id"`
	StartMetrics    MetricsSnapshot          `json:"start_metrics"`
	EndMetrics      MetricsSnapshot          `json:"end_metrics"`
	PhaseBreakdown  []PhasePerformance       `json:"phase_breakdown"`
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

// IterationEvaluation represents the result of evaluating iteration conditions
type IterationEvaluation struct {
	ShouldStop bool   `json:"should_stop"`
	Reason     string `json:"reason,omitempty"`
	NextPhase  int    `json:"next_phase,omitempty"`
}

// PhaseAdvanceResult represents the result of advancing to the next phase
type PhaseAdvanceResult struct {
	Success        bool   `json:"success"`
	NextPhaseIndex int    `json:"next_phase_index"`
	Completed      bool   `json:"completed"` // True if all phases are done
	Message        string `json:"message"`
}

// QualityGateResult represents the result of evaluating a quality gate
type QualityGateResult struct {
	GateName string            `json:"gate_name"`
	Passed   bool              `json:"passed"`
	Message  string            `json:"message"`
	Action   QualityGateAction `json:"action"`
}
