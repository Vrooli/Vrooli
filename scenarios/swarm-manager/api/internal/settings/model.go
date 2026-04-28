package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/storage"
)

// DeleteConfirmLevel controls the confirmation UI for delete operations.
type DeleteConfirmLevel string

const (
	DeleteConfirmNone   DeleteConfirmLevel = "none"
	DeleteConfirmSimple DeleteConfirmLevel = "simple"
	DeleteConfirmStrong DeleteConfirmLevel = "strong"
)

// DeleteConfirmationSettings holds per-entity-type confirmation levels.
type DeleteConfirmationSettings struct {
	Backlog    DeleteConfirmLevel `json:"backlog"`
	Initiative DeleteConfirmLevel `json:"initiative"`
	Capture    DeleteConfirmLevel `json:"capture"`
}

// DeleteConfirmationSettingsPatch allows partial updates to delete confirmation.
type DeleteConfirmationSettingsPatch struct {
	Backlog    *DeleteConfirmLevel `json:"backlog,omitempty"`
	Initiative *DeleteConfirmLevel `json:"initiative,omitempty"`
	Capture    *DeleteConfirmLevel `json:"capture,omitempty"`
}

// Settings represents persisted configuration for the scenario.
type Settings struct {
	Theme string `json:"theme"`

	// Execution defaults.
	DefaultMode        string `json:"default_mode"`
	AutoFixup          bool   `json:"auto_fixup"`
	MaxFixupAttempts   int    `json:"max_fixup_attempts"`
	ReviewAgentEnabled bool   `json:"review_agent_enabled"`

	// Workshop.
	MaxAutoRounds           int  `json:"max_auto_rounds"`
	AutoInitializeWorkshop  bool `json:"auto_initialize_workshop"`
	AutoAdvanceWorkshop     bool `json:"auto_advance_workshop"`
	AutoCascadeWorkshop     bool `json:"auto_cascade_workshop"`
	AutoAdvanceDelaySeconds int  `json:"auto_advance_delay_seconds"`

	// Agent behavior.
	AgentMaxTurns       int `json:"agent_max_turns"`
	AgentTimeoutSeconds int `json:"agent_timeout_seconds"`

	// UI preferences.
	SearchDebounceMs   int                        `json:"search_debounce_ms"`
	ToastDurationMs    int                        `json:"toast_duration_ms"`
	DeleteConfirmation DeleteConfirmationSettings `json:"delete_confirmation"`

	// Review thresholds.
	ReviewCodeQualityMinScore   float64 `json:"review_code_quality_min_score"`
	ReviewTestMinPassRate       float64 `json:"review_test_min_pass_rate"`
	ReviewMaxBlockingViolations int     `json:"review_max_blocking_violations"`
	ReviewMaxWarnings           int     `json:"review_max_warnings"`
	ReviewRequireScreenshots    bool    `json:"review_require_screenshots"`
	ReviewRequireTests          bool    `json:"review_require_tests"`

	// Concurrency and governance.
	MaxConcurrentExecutions       int     `json:"max_concurrent_executions"`
	MaxQueueDepth                 int     `json:"max_queue_depth"`
	CircuitBreakerThreshold       int     `json:"circuit_breaker_threshold"`
	CircuitBreakerCooldownMinutes int     `json:"circuit_breaker_cooldown_minutes"`
	ExecutionCostCapPerRun        float64 `json:"execution_cost_cap_per_run"`
	CostPerTurnEstimate           float64 `json:"cost_per_turn_estimate"`
}

// SettingsPatch allows partial updates.
type SettingsPatch struct {
	Theme *string `json:"theme,omitempty"`

	DefaultMode        *string `json:"default_mode,omitempty"`
	AutoFixup          *bool   `json:"auto_fixup,omitempty"`
	MaxFixupAttempts   *int    `json:"max_fixup_attempts,omitempty"`
	ReviewAgentEnabled *bool   `json:"review_agent_enabled,omitempty"`

	MaxAutoRounds           *int  `json:"max_auto_rounds,omitempty"`
	AutoInitializeWorkshop  *bool `json:"auto_initialize_workshop,omitempty"`
	AutoAdvanceWorkshop     *bool `json:"auto_advance_workshop,omitempty"`
	AutoCascadeWorkshop     *bool `json:"auto_cascade_workshop,omitempty"`
	AutoAdvanceDelaySeconds *int  `json:"auto_advance_delay_seconds,omitempty"`

	AgentMaxTurns       *int `json:"agent_max_turns,omitempty"`
	AgentTimeoutSeconds *int `json:"agent_timeout_seconds,omitempty"`

	SearchDebounceMs   *int                             `json:"search_debounce_ms,omitempty"`
	ToastDurationMs    *int                             `json:"toast_duration_ms,omitempty"`
	DeleteConfirmation *DeleteConfirmationSettingsPatch `json:"delete_confirmation,omitempty"`

	ReviewCodeQualityMinScore   *float64 `json:"review_code_quality_min_score,omitempty"`
	ReviewTestMinPassRate       *float64 `json:"review_test_min_pass_rate,omitempty"`
	ReviewMaxBlockingViolations *int     `json:"review_max_blocking_violations,omitempty"`
	ReviewMaxWarnings           *int     `json:"review_max_warnings,omitempty"`
	ReviewRequireScreenshots    *bool    `json:"review_require_screenshots,omitempty"`
	ReviewRequireTests          *bool    `json:"review_require_tests,omitempty"`

	MaxConcurrentExecutions       *int     `json:"max_concurrent_executions,omitempty"`
	MaxQueueDepth                 *int     `json:"max_queue_depth,omitempty"`
	CircuitBreakerThreshold       *int     `json:"circuit_breaker_threshold,omitempty"`
	CircuitBreakerCooldownMinutes *int     `json:"circuit_breaker_cooldown_minutes,omitempty"`
	ExecutionCostCapPerRun        *float64 `json:"execution_cost_cap_per_run,omitempty"`
	CostPerTurnEstimate           *float64 `json:"cost_per_turn_estimate,omitempty"`
}

// Store persists settings on disk.
type Store struct {
	path string
}

var (
	errInvalidTheme = errors.New("invalid theme")
	errInvalidMode  = errors.New("invalid default_mode")
)

// NewStore creates a settings store. If path is empty, uses the scenario default.
func NewStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), "config", "settings.json")
	}
	return &Store{path: path}
}

// StoreForPath creates a settings store at the given path (for use by other packages).
func StoreForPath(path string) *Store {
	return &Store{path: path}
}

// DefaultSettings returns the baseline settings.
func DefaultSettings() Settings {
	return Settings{
		Theme:                   "dark",
		DefaultMode:             "yolo",
		AutoFixup:               false,
		MaxFixupAttempts:        2,
		ReviewAgentEnabled:      true,
		MaxAutoRounds:           10,
		AutoInitializeWorkshop:  true,
		AutoAdvanceWorkshop:     true,
		AutoCascadeWorkshop:     true,
		AutoAdvanceDelaySeconds: 10,
		// Keep in sync with agentmanager.DefaultAgentMaxTurns (600).
		AgentMaxTurns:       600,
		AgentTimeoutSeconds: 3600,
		SearchDebounceMs:    300,
		ToastDurationMs:     5000,
		DeleteConfirmation: DeleteConfirmationSettings{
			Backlog:    DeleteConfirmSimple,
			Initiative: DeleteConfirmStrong,
			Capture:    DeleteConfirmNone,
		},

		ReviewCodeQualityMinScore:   60,
		ReviewTestMinPassRate:       1.0,
		ReviewMaxBlockingViolations: 0,
		ReviewMaxWarnings:           -1,
		ReviewRequireScreenshots:    true,
		ReviewRequireTests:          true,

		MaxConcurrentExecutions:       3,
		MaxQueueDepth:                 50,
		CircuitBreakerThreshold:       3,
		CircuitBreakerCooldownMinutes: 60,
		ExecutionCostCapPerRun:        0,
		CostPerTurnEstimate:           0.10,
	}
}

// Load retrieves settings from disk, returning defaults when missing.
func (s *Store) Load() (Settings, error) {
	data, err := storage.ReadJSONBytes(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultSettings(), nil
		}
		return Settings{}, err
	}
	if len(data) == 0 {
		return DefaultSettings(), nil
	}

	// Unmarshal into defaults so missing fields get sane values.
	settings := DefaultSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	normalized := normalizeSettings(settings)
	if err := validateSettings(normalized); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}

// Save persists settings to disk with validation and atomic writes.
func (s *Store) Save(settings Settings) error {
	normalized := normalizeSettings(settings)
	if err := validateSettings(normalized); err != nil {
		return err
	}
	return storage.WriteJSONAtomic(s.path, normalized)
}

// Path returns the underlying file path (for callers that need to pass it on).
func (s *Store) Path() string {
	return s.path
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func normalizeDeleteConfirmLevel(level, fallback DeleteConfirmLevel) DeleteConfirmLevel {
	switch level {
	case DeleteConfirmNone, DeleteConfirmSimple, DeleteConfirmStrong:
		return level
	default:
		return fallback
	}
}

func normalizeSettings(settings Settings) Settings {
	if strings.TrimSpace(settings.Theme) == "" {
		settings.Theme = "dark"
	}

	// Execution defaults.
	mode := strings.TrimSpace(settings.DefaultMode)
	switch mode {
	case "manual", "yolo":
		settings.DefaultMode = mode
	default:
		settings.DefaultMode = "yolo"
	}
	settings.MaxFixupAttempts = clampInt(settings.MaxFixupAttempts, 0, 5)

	// Workshop.
	settings.MaxAutoRounds = clampInt(settings.MaxAutoRounds, 0, 50)
	settings.AutoAdvanceDelaySeconds = clampInt(settings.AutoAdvanceDelaySeconds, 0, 120)

	// Agent behavior.
	settings.AgentMaxTurns = clampInt(settings.AgentMaxTurns, 5, 1000)
	settings.AgentTimeoutSeconds = clampInt(settings.AgentTimeoutSeconds, 60, 3600)

	// UI preferences.
	settings.SearchDebounceMs = clampInt(settings.SearchDebounceMs, 100, 2000)
	settings.ToastDurationMs = clampInt(settings.ToastDurationMs, 1000, 30000)
	defaults := DefaultSettings()
	settings.DeleteConfirmation.Backlog = normalizeDeleteConfirmLevel(settings.DeleteConfirmation.Backlog, defaults.DeleteConfirmation.Backlog)
	settings.DeleteConfirmation.Initiative = normalizeDeleteConfirmLevel(settings.DeleteConfirmation.Initiative, defaults.DeleteConfirmation.Initiative)
	settings.DeleteConfirmation.Capture = normalizeDeleteConfirmLevel(settings.DeleteConfirmation.Capture, defaults.DeleteConfirmation.Capture)

	// Review thresholds.
	settings.ReviewCodeQualityMinScore = clampFloat(settings.ReviewCodeQualityMinScore, 0, 100)
	settings.ReviewTestMinPassRate = clampFloat(settings.ReviewTestMinPassRate, 0, 1)
	if settings.ReviewMaxBlockingViolations < 0 {
		settings.ReviewMaxBlockingViolations = 0
	}
	if settings.ReviewMaxWarnings < -1 {
		settings.ReviewMaxWarnings = -1
	}

	// Concurrency and governance.
	settings.MaxConcurrentExecutions = clampInt(settings.MaxConcurrentExecutions, 1, 20)
	settings.MaxQueueDepth = clampInt(settings.MaxQueueDepth, 0, 100)
	settings.CircuitBreakerThreshold = clampInt(settings.CircuitBreakerThreshold, 1, 10)
	settings.CircuitBreakerCooldownMinutes = clampInt(settings.CircuitBreakerCooldownMinutes, 5, 1440)
	if settings.ExecutionCostCapPerRun < 0 {
		settings.ExecutionCostCapPerRun = 0
	}
	settings.CostPerTurnEstimate = clampFloat(settings.CostPerTurnEstimate, 0, 5)

	return settings
}

func validateSettings(settings Settings) error {
	switch settings.Theme {
	case "dark", "light", "system":
		// ok
	default:
		return errInvalidTheme
	}
	switch settings.DefaultMode {
	case "manual", "yolo":
		// ok
	default:
		return errInvalidMode
	}
	return nil
}

func applyPatch(current Settings, patch SettingsPatch) Settings {
	if patch.Theme != nil {
		current.Theme = strings.TrimSpace(*patch.Theme)
	}
	if patch.DefaultMode != nil {
		current.DefaultMode = strings.TrimSpace(*patch.DefaultMode)
	}
	if patch.AutoFixup != nil {
		current.AutoFixup = *patch.AutoFixup
	}
	if patch.MaxFixupAttempts != nil {
		current.MaxFixupAttempts = *patch.MaxFixupAttempts
	}
	if patch.ReviewAgentEnabled != nil {
		current.ReviewAgentEnabled = *patch.ReviewAgentEnabled
	}
	if patch.MaxAutoRounds != nil {
		current.MaxAutoRounds = *patch.MaxAutoRounds
	}
	if patch.AutoInitializeWorkshop != nil {
		current.AutoInitializeWorkshop = *patch.AutoInitializeWorkshop
	}
	if patch.AutoAdvanceWorkshop != nil {
		current.AutoAdvanceWorkshop = *patch.AutoAdvanceWorkshop
	}
	if patch.AutoCascadeWorkshop != nil {
		current.AutoCascadeWorkshop = *patch.AutoCascadeWorkshop
	}
	if patch.AutoAdvanceDelaySeconds != nil {
		current.AutoAdvanceDelaySeconds = *patch.AutoAdvanceDelaySeconds
	}
	if patch.AgentMaxTurns != nil {
		current.AgentMaxTurns = *patch.AgentMaxTurns
	}
	if patch.AgentTimeoutSeconds != nil {
		current.AgentTimeoutSeconds = *patch.AgentTimeoutSeconds
	}
	if patch.SearchDebounceMs != nil {
		current.SearchDebounceMs = *patch.SearchDebounceMs
	}
	if patch.ToastDurationMs != nil {
		current.ToastDurationMs = *patch.ToastDurationMs
	}
	if patch.DeleteConfirmation != nil {
		if patch.DeleteConfirmation.Backlog != nil {
			current.DeleteConfirmation.Backlog = *patch.DeleteConfirmation.Backlog
		}
		if patch.DeleteConfirmation.Initiative != nil {
			current.DeleteConfirmation.Initiative = *patch.DeleteConfirmation.Initiative
		}
		if patch.DeleteConfirmation.Capture != nil {
			current.DeleteConfirmation.Capture = *patch.DeleteConfirmation.Capture
		}
	}
	if patch.ReviewCodeQualityMinScore != nil {
		current.ReviewCodeQualityMinScore = *patch.ReviewCodeQualityMinScore
	}
	if patch.ReviewTestMinPassRate != nil {
		current.ReviewTestMinPassRate = *patch.ReviewTestMinPassRate
	}
	if patch.ReviewMaxBlockingViolations != nil {
		current.ReviewMaxBlockingViolations = *patch.ReviewMaxBlockingViolations
	}
	if patch.ReviewMaxWarnings != nil {
		current.ReviewMaxWarnings = *patch.ReviewMaxWarnings
	}
	if patch.ReviewRequireScreenshots != nil {
		current.ReviewRequireScreenshots = *patch.ReviewRequireScreenshots
	}
	if patch.ReviewRequireTests != nil {
		current.ReviewRequireTests = *patch.ReviewRequireTests
	}
	if patch.MaxConcurrentExecutions != nil {
		current.MaxConcurrentExecutions = *patch.MaxConcurrentExecutions
	}
	if patch.MaxQueueDepth != nil {
		current.MaxQueueDepth = *patch.MaxQueueDepth
	}
	if patch.CircuitBreakerThreshold != nil {
		current.CircuitBreakerThreshold = *patch.CircuitBreakerThreshold
	}
	if patch.CircuitBreakerCooldownMinutes != nil {
		current.CircuitBreakerCooldownMinutes = *patch.CircuitBreakerCooldownMinutes
	}
	if patch.ExecutionCostCapPerRun != nil {
		current.ExecutionCostCapPerRun = *patch.ExecutionCostCapPerRun
	}
	if patch.CostPerTurnEstimate != nil {
		current.CostPerTurnEstimate = *patch.CostPerTurnEstimate
	}
	return current
}
