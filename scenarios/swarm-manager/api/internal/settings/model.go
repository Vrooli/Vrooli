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

// Fix-before-feature gate modes.
const (
	FixBeforeFeatureOff     = "off"
	FixBeforeFeatureSuggest = "suggest"
	FixBeforeFeatureBlock   = "block"
)

// deletableEntityDefaults mirrors the UI registry in
// ui/src/lib/deletable-entities.ts. Keys are the entity-type strings shared
// with the UI as the delete-confirmation map keys. Keep the two in sync when
// adding a deletable entity type.
var deletableEntityDefaults = map[string]DeleteConfirmLevel{
	"session":     DeleteConfirmSimple,
	"scenario":    DeleteConfirmStrong,
	"backlog":     DeleteConfirmSimple,
	"initiative":  DeleteConfirmStrong,
	"capture":     DeleteConfirmNone,
	"backlogFile": DeleteConfirmSimple,
}

// defaultDeleteConfirmationLevels returns a fresh copy of the registry
// defaults so callers never mutate the shared map.
func defaultDeleteConfirmationLevels() map[string]DeleteConfirmLevel {
	out := make(map[string]DeleteConfirmLevel, len(deletableEntityDefaults))
	for k, v := range deletableEntityDefaults {
		out[k] = v
	}
	return out
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
	SearchDebounceMs int `json:"search_debounce_ms"`
	ToastDurationMs  int `json:"toast_duration_ms"`
	// DeleteConfirmationLevels maps a deletable entity-type string (mirroring
	// the UI registry: "session", "scenario", "backlog", "initiative",
	// "capture", "backlogFile", ...) to its confirmation level. Missing known
	// keys are filled from registry defaults on normalize; unknown keys are
	// preserved for forward-compat with newer UIs.
	DeleteConfirmationLevels map[string]DeleteConfirmLevel `json:"delete_confirmation_levels"`

	// Review thresholds.
	ReviewCodeQualityMinScore   float64 `json:"review_code_quality_min_score"`
	ReviewTestMinPassRate       float64 `json:"review_test_min_pass_rate"`
	ReviewMaxBlockingViolations int     `json:"review_max_blocking_violations"`
	ReviewMaxWarnings           int     `json:"review_max_warnings"`
	ReviewRequireScreenshots    bool    `json:"review_require_screenshots"`
	ReviewRequireTests          bool    `json:"review_require_tests"`

	// Concurrency and governance.
	// LaneConcurrencyLimits caps simultaneous tracked agent activities by
	// phase-kind lane. Keys are lane names matching agentactivity.Lane /
	// operatingmode.PhaseKind values: "investigate", "execute", "review",
	// "reconcile". Values <= 0 are clamped to 1 by normalize. Replaces the
	// pre-P2 single global cap.
	LaneConcurrencyLimits         map[string]int `json:"lane_concurrency_limits"`
	MaxQueueDepth                 int            `json:"max_queue_depth"`
	CircuitBreakerThreshold       int            `json:"circuit_breaker_threshold"`
	CircuitBreakerCooldownMinutes int            `json:"circuit_breaker_cooldown_minutes"`
	ExecutionCostCapPerRun        float64        `json:"execution_cost_cap_per_run"`
	CostPerTurnEstimate           float64        `json:"cost_per_turn_estimate"`

	// Fix-before-feature gate. When a feature item (kind=execute) is queued
	// onto a scenario that already has open fix/chore items, the gate reacts
	// per this mode: "off" (silent), "suggest" (advisory in the queue
	// response), or "block" (forceable BlockingReason). Default "suggest".
	FixBeforeFeature string `json:"fix_before_feature"`
	// FixBeforeFeatureDiscovery, when true, lets the gate trigger an async
	// on-demand readiness review for a scenario that has *no* known open fix
	// items (and no recent readiness signal), auto-filing fix items it finds.
	// Default false.
	FixBeforeFeatureDiscovery bool `json:"fix_before_feature_discovery"`
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

	SearchDebounceMs *int `json:"search_debounce_ms,omitempty"`
	ToastDurationMs  *int `json:"toast_duration_ms,omitempty"`
	// DeleteConfirmationLevels, when non-nil, merges the given entity→level
	// pairs over the current map (provided keys overwrite; omitted keys and
	// unknown existing keys are left intact). Pass nil to leave untouched.
	DeleteConfirmationLevels map[string]DeleteConfirmLevel `json:"delete_confirmation_levels,omitempty"`

	ReviewCodeQualityMinScore   *float64 `json:"review_code_quality_min_score,omitempty"`
	ReviewTestMinPassRate       *float64 `json:"review_test_min_pass_rate,omitempty"`
	ReviewMaxBlockingViolations *int     `json:"review_max_blocking_violations,omitempty"`
	ReviewMaxWarnings           *int     `json:"review_max_warnings,omitempty"`
	ReviewRequireScreenshots    *bool    `json:"review_require_screenshots,omitempty"`
	ReviewRequireTests          *bool    `json:"review_require_tests,omitempty"`

	// LaneConcurrencyLimits, when non-nil, replaces the corresponding lane
	// caps wholesale. Keys are lane names; values must be >= 1. Pass nil to
	// leave the existing map untouched. Pass an empty map to reset to
	// defaults (DefaultSettings()).
	LaneConcurrencyLimits         map[string]int `json:"lane_concurrency_limits,omitempty"`
	MaxQueueDepth                 *int           `json:"max_queue_depth,omitempty"`
	CircuitBreakerThreshold       *int           `json:"circuit_breaker_threshold,omitempty"`
	CircuitBreakerCooldownMinutes *int           `json:"circuit_breaker_cooldown_minutes,omitempty"`
	ExecutionCostCapPerRun        *float64       `json:"execution_cost_cap_per_run,omitempty"`
	CostPerTurnEstimate           *float64       `json:"cost_per_turn_estimate,omitempty"`

	FixBeforeFeature          *string `json:"fix_before_feature,omitempty"`
	FixBeforeFeatureDiscovery *bool   `json:"fix_before_feature_discovery,omitempty"`
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
		// Per-run turn budget mirrored by the swarm-manager agent profile
		// in .vrooli/agent-profiles (the canonical source for spawns).
		AgentMaxTurns:            600,
		AgentTimeoutSeconds:      3600,
		SearchDebounceMs:         300,
		ToastDurationMs:          5000,
		DeleteConfirmationLevels: defaultDeleteConfirmationLevels(),

		ReviewCodeQualityMinScore:   60,
		ReviewTestMinPassRate:       1.0,
		ReviewMaxBlockingViolations: 0,
		ReviewMaxWarnings:           -1,
		ReviewRequireScreenshots:    true,
		ReviewRequireTests:          true,

		// Lane caps: investigate (workshop / clarify / classify / research /
		// feedback) is the most parallelizable; execute matches today's
		// global default of 3 to preserve backlog-process semantics; review
		// is read-mostly so we allow more headroom; reconcile is bounded
		// since it follows execute and writes to the backlog.
		LaneConcurrencyLimits: map[string]int{
			"investigate": 6,
			"execute":     3,
			"review":      8,
			"reconcile":   2,
		},
		MaxQueueDepth:                 50,
		CircuitBreakerThreshold:       3,
		CircuitBreakerCooldownMinutes: 60,
		ExecutionCostCapPerRun:        0,
		CostPerTurnEstimate:           0.10,

		FixBeforeFeature:          FixBeforeFeatureSuggest,
		FixBeforeFeatureDiscovery: false,
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

// laneKeys is the canonical set of LaneConcurrencyLimits keys. Mirrors
// agentactivity.Lanes() but lives here as plain strings to keep the
// settings package free of agentactivity imports (settings is a leaf used
// by many adapters).
var laneKeys = []string{"investigate", "execute", "review", "reconcile"}

// normalizeLaneConcurrencyLimits canonicalizes the lane cap map: every
// canonical key is present, missing keys are filled from DefaultSettings,
// values are clamped to [1, 50] (50 is generous; the validator stops well
// before any realistic system limit), and unknown keys are dropped to
// keep the wire surface tight.
func normalizeLaneConcurrencyLimits(raw map[string]int) map[string]int {
	defaults := defaultLaneConcurrencyLimits()
	out := make(map[string]int, len(laneKeys))
	for _, key := range laneKeys {
		val, ok := raw[key]
		if !ok || val <= 0 {
			out[key] = defaults[key]
			continue
		}
		out[key] = clampInt(val, 1, 50)
	}
	return out
}

// defaultLaneConcurrencyLimits returns a fresh copy of the canonical
// defaults so callers never accidentally mutate the DefaultSettings
// shared map.
func defaultLaneConcurrencyLimits() map[string]int {
	return map[string]int{
		"investigate": 6,
		"execute":     3,
		"review":      8,
		"reconcile":   2,
	}
}

func normalizeDeleteConfirmLevel(level, fallback DeleteConfirmLevel) DeleteConfirmLevel {
	switch level {
	case DeleteConfirmNone, DeleteConfirmSimple, DeleteConfirmStrong:
		return level
	default:
		return fallback
	}
}

// normalizeDeleteConfirmationLevels canonicalizes the per-entity level map:
// every known registry key is present (missing ones filled from defaults),
// and any unknown key is preserved (forward-compat with newer UIs) but its
// value is coerced to a valid level. Known keys with invalid values fall back
// to the registry default; unknown keys with invalid values fall back to
// simple confirmation (the safe default).
func normalizeDeleteConfirmationLevels(raw map[string]DeleteConfirmLevel) map[string]DeleteConfirmLevel {
	out := make(map[string]DeleteConfirmLevel, len(deletableEntityDefaults))
	// Fill known keys from defaults, overridden by any valid provided value.
	for key, def := range deletableEntityDefaults {
		out[key] = normalizeDeleteConfirmLevel(raw[key], def)
	}
	// Preserve unknown keys so an older API does not clobber a newer UI's
	// entity types; coerce their values to a valid level.
	for key, val := range raw {
		if _, known := deletableEntityDefaults[key]; known {
			continue
		}
		out[key] = normalizeDeleteConfirmLevel(val, DeleteConfirmSimple)
	}
	return out
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
	settings.DeleteConfirmationLevels = normalizeDeleteConfirmationLevels(settings.DeleteConfirmationLevels)

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
	settings.LaneConcurrencyLimits = normalizeLaneConcurrencyLimits(settings.LaneConcurrencyLimits)
	settings.MaxQueueDepth = clampInt(settings.MaxQueueDepth, 0, 100)
	settings.CircuitBreakerThreshold = clampInt(settings.CircuitBreakerThreshold, 1, 10)
	settings.CircuitBreakerCooldownMinutes = clampInt(settings.CircuitBreakerCooldownMinutes, 5, 1440)
	if settings.ExecutionCostCapPerRun < 0 {
		settings.ExecutionCostCapPerRun = 0
	}
	settings.CostPerTurnEstimate = clampFloat(settings.CostPerTurnEstimate, 0, 5)

	// Fix-before-feature gate.
	switch settings.FixBeforeFeature {
	case FixBeforeFeatureOff, FixBeforeFeatureSuggest, FixBeforeFeatureBlock:
		// ok
	default:
		settings.FixBeforeFeature = FixBeforeFeatureSuggest
	}

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
	applyExecutionPatch(&current, patch)
	applyWorkshopPatch(&current, patch)
	applyAgentPatch(&current, patch)
	applyUIPatch(&current, patch)
	applyReviewPatch(&current, patch)
	applyGovernancePatch(&current, patch)
	return current
}

// applyExecutionPatch overlays the execution-default fields.
func applyExecutionPatch(current *Settings, patch SettingsPatch) {
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
}

// applyWorkshopPatch overlays the workshop fields.
func applyWorkshopPatch(current *Settings, patch SettingsPatch) {
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
}

// applyAgentPatch overlays the agent-behavior fields.
func applyAgentPatch(current *Settings, patch SettingsPatch) {
	if patch.AgentMaxTurns != nil {
		current.AgentMaxTurns = *patch.AgentMaxTurns
	}
	if patch.AgentTimeoutSeconds != nil {
		current.AgentTimeoutSeconds = *patch.AgentTimeoutSeconds
	}
}

// applyUIPatch overlays the UI-preference fields, including the
// delete-confirmation map merge.
func applyUIPatch(current *Settings, patch SettingsPatch) {
	if patch.SearchDebounceMs != nil {
		current.SearchDebounceMs = *patch.SearchDebounceMs
	}
	if patch.ToastDurationMs != nil {
		current.ToastDurationMs = *patch.ToastDurationMs
	}
	if patch.DeleteConfirmationLevels != nil {
		if current.DeleteConfirmationLevels == nil {
			current.DeleteConfirmationLevels = make(map[string]DeleteConfirmLevel, len(patch.DeleteConfirmationLevels))
		}
		// Merge provided keys over current; omitted and unknown existing keys
		// are left intact. normalize fills/validates afterwards.
		for k, v := range patch.DeleteConfirmationLevels {
			current.DeleteConfirmationLevels[k] = v
		}
	}
}

// applyReviewPatch overlays the review-threshold fields.
func applyReviewPatch(current *Settings, patch SettingsPatch) {
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
}

// applyGovernancePatch overlays the concurrency/governance and
// fix-before-feature fields, including the lane-cap map replacement.
func applyGovernancePatch(current *Settings, patch SettingsPatch) {
	if patch.LaneConcurrencyLimits != nil {
		// Non-nil patch replaces wholesale (after normalize fills any
		// missing canonical keys from defaults). Empty map → reset to
		// defaults via normalize.
		merged := make(map[string]int, len(patch.LaneConcurrencyLimits))
		for k, v := range patch.LaneConcurrencyLimits {
			merged[k] = v
		}
		current.LaneConcurrencyLimits = merged
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
	if patch.FixBeforeFeature != nil {
		current.FixBeforeFeature = strings.TrimSpace(*patch.FixBeforeFeature)
	}
	if patch.FixBeforeFeatureDiscovery != nil {
		current.FixBeforeFeatureDiscovery = *patch.FixBeforeFeatureDiscovery
	}
}
