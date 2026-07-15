package agentops

// PolicyControls is the typed projection of user settings into the controls
// that orchestration consumers — the operation runner's transition policies,
// the workshop auto-advance/scheduler-intent path, review-round spawning, and
// fixup/retry handling — are allowed to read. It is the SEAM between "what the
// user configured" (internal/settings, persisted JSON) and "what governs
// operation transitions" (this package's transition-policy vocabulary).
//
// Consumers MUST read these controls through a PolicyControlsProvider instead
// of ad-hoc settings lookups. The projection is intentionally lossless for the
// retained user preferences (auto-advance + delay, retry limits, review
// thresholds, execution posture, agent budgets): for identical settings values
// the derived controls — and therefore runtime behavior — are identical.
//
// Phase 8 migrates persisted legacy settings into system-default bindings and
// transition-policy revisions using the equivalence table in
// docs/operations/migration/LEGACY-MAPPING.md §9; this type is the target
// vocabulary for that migration.
type PolicyControls struct {
	// Execution is the default execution posture applied when a queue
	// request does not specify a mode.
	Execution ExecutionControls `json:"execution"`
	// AutoAdvance governs autonomous workflow progression (workshop
	// auto-initialize, auto-advance + grace delay, dependency cascade).
	AutoAdvance AutoAdvanceControls `json:"auto_advance"`
	// Retry governs automatic fixup re-runs after a failed review.
	Retry RetryControls `json:"retry"`
	// Review governs autonomous review-round spawning and the readiness
	// thresholds handed to the reviewer.
	Review ReviewControls `json:"review"`
	// Budgets caps per-run agent effort (turns / wall clock).
	Budgets AgentBudgetControls `json:"budgets"`
}

// ExecutionControls is the default execution posture.
type ExecutionControls struct {
	// DefaultMode is "manual" or "yolo".
	DefaultMode string `json:"default_mode"`
}

// Execution posture values for ExecutionControls.DefaultMode.
const (
	ExecutionPostureManual = "manual"
	ExecutionPostureYOLO   = "yolo"
)

// AutoAdvanceControls governs autonomous workflow progression.
type AutoAdvanceControls struct {
	// AutoInitialize spawns the first workshop round when an item is created.
	AutoInitialize bool `json:"auto_initialize"`
	// Enabled continues workshop saves into the next round or synthesis.
	Enabled bool `json:"enabled"`
	// Cascade triggers dependent-item workshops when a dependency unblocks.
	Cascade bool `json:"cascade"`
	// DelaySeconds is the grace period before a scheduled advance intent
	// fires (0 = immediate advance, no scheduler intent).
	DelaySeconds int `json:"delay_seconds"`
	// MaxAutoRounds caps autonomous workshop rounds before advancement stops.
	MaxAutoRounds int `json:"max_auto_rounds"`
}

// RetryControls governs automatic fixup re-runs.
type RetryControls struct {
	// AutoFixup re-runs execution automatically when review finds issues.
	AutoFixup bool `json:"auto_fixup"`
	// MaxFixupAttempts caps automatic fixup re-runs per execution.
	MaxFixupAttempts int `json:"max_fixup_attempts"`
}

// ReviewControls governs autonomous review spawning and readiness thresholds.
type ReviewControls struct {
	// AgentEnabled allows autonomous evidence-gathering review rounds.
	AgentEnabled bool `json:"agent_enabled"`
	// CodeQualityMinScore is the minimum tidiness score for green (0-100).
	CodeQualityMinScore float64 `json:"code_quality_min_score"`
	// TestMinPassRate is the minimum test pass rate for green (0-1).
	TestMinPassRate float64 `json:"test_min_pass_rate"`
	// MaxBlockingViolations caps critical/error violations for green.
	MaxBlockingViolations int `json:"max_blocking_violations"`
	// MaxWarnings caps warnings before yellow (-1 = unlimited).
	MaxWarnings int `json:"max_warnings"`
	// RequireScreenshots requires screenshots for green status.
	RequireScreenshots bool `json:"require_screenshots"`
	// RequireTests requires existing, passing tests for green status.
	RequireTests bool `json:"require_tests"`
}

// AgentBudgetControls caps per-run agent effort.
type AgentBudgetControls struct {
	// MaxTurns is the per-run conversation-turn budget (also the cost-cap
	// estimation input).
	MaxTurns int `json:"max_turns"`
	// TimeoutSeconds is the per-run wall-clock budget. Currently DORMANT:
	// persisted and surfaced but no runtime consumer reads it (spawn
	// timeouts come from the agent profile). Kept in the projection so
	// Phase 8 migrates it alongside MaxTurns instead of dropping it.
	TimeoutSeconds int `json:"timeout_seconds"`
}

// PolicyControlsProvider is the read seam orchestration consumers use to
// obtain the current controls. Implementations load the latest persisted
// settings on every call so an operator update is picked up without a
// restart (matching the pre-seam behavior of ad-hoc settings loads).
type PolicyControlsProvider interface {
	LoadPolicyControls() (PolicyControls, error)
}

// PolicyControlsProviderFunc adapts a function to PolicyControlsProvider.
type PolicyControlsProviderFunc func() (PolicyControls, error)

// LoadPolicyControls implements PolicyControlsProvider.
func (f PolicyControlsProviderFunc) LoadPolicyControls() (PolicyControls, error) {
	return f()
}

// DefaultPolicyControls returns the controls derived from default settings.
// It MUST stay equal to the projection of settings.DefaultSettings() — the
// settings package asserts this equivalence in its tests — so a settings-store
// outage degrades to exactly the same behavior as a missing settings file.
func DefaultPolicyControls() PolicyControls {
	return PolicyControls{
		Execution: ExecutionControls{DefaultMode: ExecutionPostureYOLO},
		AutoAdvance: AutoAdvanceControls{
			AutoInitialize: true,
			Enabled:        true,
			Cascade:        true,
			DelaySeconds:   10,
			MaxAutoRounds:  10,
		},
		Retry: RetryControls{
			AutoFixup:        false,
			MaxFixupAttempts: 2,
		},
		Review: ReviewControls{
			AgentEnabled:          true,
			CodeQualityMinScore:   60,
			TestMinPassRate:       1.0,
			MaxBlockingViolations: 0,
			MaxWarnings:           -1,
			RequireScreenshots:    true,
			RequireTests:          true,
		},
		Budgets: AgentBudgetControls{
			MaxTurns:       600,
			TimeoutSeconds: 3600,
		},
	}
}
