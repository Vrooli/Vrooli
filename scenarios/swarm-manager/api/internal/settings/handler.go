// Package settings provides filesystem-backed settings persistence.
//
// Settings are stored at scenarios/swarm-manager/.vrooli/settings.json by default.
// This keeps the scenario fully local and git-trackable without DB dependencies.
//
// DOC: docs/reference/configuration.md
// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
package settings

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/storage"
)

// Settings represents persisted configuration for the scenario.
type Settings struct {
	Theme string `json:"theme"`

	// Execution defaults.
	DefaultMode      string `json:"default_mode"`
	AutoFixup        bool   `json:"auto_fixup"`
	MaxFixupAttempts int    `json:"max_fixup_attempts"`

	// Workshop.
	MaxAutoRounds          int  `json:"max_auto_rounds"`
	AutoInitializeWorkshop bool `json:"auto_initialize_workshop"`
	AutoAdvanceWorkshop    bool `json:"auto_advance_workshop"`
	AutoCascadeWorkshop    bool `json:"auto_cascade_workshop"`

	// Agent behavior.
	AgentMaxTurns         int  `json:"agent_max_turns"`
	AgentTimeoutSeconds   int  `json:"agent_timeout_seconds"`
	AgentRequiresApproval bool `json:"agent_requires_approval"`

	// UI preferences.
	SearchDebounceMs          int  `json:"search_debounce_ms"`
	ToastDurationMs           int  `json:"toast_duration_ms"`
	ConfirmDestructiveActions bool `json:"confirm_destructive_actions"`

	// Review thresholds.
	ReviewCodeQualityMinScore   float64 `json:"review_code_quality_min_score"`
	ReviewTestMinPassRate       float64 `json:"review_test_min_pass_rate"`
	ReviewMaxBlockingViolations int     `json:"review_max_blocking_violations"`
	ReviewMaxWarnings           int     `json:"review_max_warnings"`
	ReviewRequireScreenshots    bool    `json:"review_require_screenshots"`
	ReviewRequireTests          bool    `json:"review_require_tests"`
}

// SettingsPatch allows partial updates.
type SettingsPatch struct {
	Theme *string `json:"theme,omitempty"`

	DefaultMode      *string `json:"default_mode,omitempty"`
	AutoFixup        *bool   `json:"auto_fixup,omitempty"`
	MaxFixupAttempts *int    `json:"max_fixup_attempts,omitempty"`

	MaxAutoRounds          *int  `json:"max_auto_rounds,omitempty"`
	AutoInitializeWorkshop *bool `json:"auto_initialize_workshop,omitempty"`
	AutoAdvanceWorkshop    *bool `json:"auto_advance_workshop,omitempty"`
	AutoCascadeWorkshop    *bool `json:"auto_cascade_workshop,omitempty"`

	AgentMaxTurns         *int  `json:"agent_max_turns,omitempty"`
	AgentTimeoutSeconds   *int  `json:"agent_timeout_seconds,omitempty"`
	AgentRequiresApproval *bool `json:"agent_requires_approval,omitempty"`

	SearchDebounceMs          *int  `json:"search_debounce_ms,omitempty"`
	ToastDurationMs           *int  `json:"toast_duration_ms,omitempty"`
	ConfirmDestructiveActions *bool `json:"confirm_destructive_actions,omitempty"`

	ReviewCodeQualityMinScore   *float64 `json:"review_code_quality_min_score,omitempty"`
	ReviewTestMinPassRate       *float64 `json:"review_test_min_pass_rate,omitempty"`
	ReviewMaxBlockingViolations *int     `json:"review_max_blocking_violations,omitempty"`
	ReviewMaxWarnings           *int     `json:"review_max_warnings,omitempty"`
	ReviewRequireScreenshots    *bool    `json:"review_require_screenshots,omitempty"`
	ReviewRequireTests          *bool    `json:"review_require_tests,omitempty"`
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
		path = filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "settings.json")
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
		Theme:                     "dark",
		DefaultMode:               "yolo",
		AutoFixup:                 false,
		MaxFixupAttempts:          2,
		MaxAutoRounds:             10,
		AutoInitializeWorkshop:    true,
		AutoAdvanceWorkshop:       true,
		AutoCascadeWorkshop:       true,
		AgentMaxTurns:             60,
		AgentTimeoutSeconds:       900,
		AgentRequiresApproval:     true,
		SearchDebounceMs:          300,
		ToastDurationMs:           5000,
		ConfirmDestructiveActions: true,

		ReviewCodeQualityMinScore:   60,
		ReviewTestMinPassRate:       1.0,
		ReviewMaxBlockingViolations: 0,
		ReviewMaxWarnings:           -1,
		ReviewRequireScreenshots:    true,
		ReviewRequireTests:          true,
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

	// Agent behavior.
	settings.AgentMaxTurns = clampInt(settings.AgentMaxTurns, 5, 200)
	settings.AgentTimeoutSeconds = clampInt(settings.AgentTimeoutSeconds, 60, 3600)

	// UI preferences.
	settings.SearchDebounceMs = clampInt(settings.SearchDebounceMs, 100, 2000)
	settings.ToastDurationMs = clampInt(settings.ToastDurationMs, 1000, 30000)

	// Review thresholds.
	settings.ReviewCodeQualityMinScore = clampFloat(settings.ReviewCodeQualityMinScore, 0, 100)
	settings.ReviewTestMinPassRate = clampFloat(settings.ReviewTestMinPassRate, 0, 1)
	if settings.ReviewMaxBlockingViolations < 0 {
		settings.ReviewMaxBlockingViolations = 0
	}
	if settings.ReviewMaxWarnings < -1 {
		settings.ReviewMaxWarnings = -1
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

// Handler exposes HTTP endpoints for settings persistence.
// [REQ:REQ-P1-010-API] Settings persistence API
type Handler struct {
	store *Store
}

// NewHandler creates a new settings handler.
func NewHandler(path string) *Handler {
	return &Handler{store: NewStore(path)}
}

// GetStore returns the underlying store (for dependency injection into other handlers).
func (h *Handler) GetStore() *Store {
	return h.store
}

// RegisterRoutes registers settings endpoints.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/settings", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/settings", h.Update).Methods("PUT")
}

// Get returns the current settings.
func (h *Handler) Get(w http.ResponseWriter, _ *http.Request) {
	settings, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[settings] get", "failed to load settings")
		return
	}
	resp := &apipb.SettingsResponse{Settings: settingsToProto(settings)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[settings] get", "failed to encode response")
		return
	}
}

// Update applies a partial settings update and persists it.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req apipb.UpdateSettingsRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[settings] update", "invalid request body")
		return
	}

	if isEmptyUpdateSettingsRequest(&req) {
		httputil.BadRequest(w, "[settings] update", "no settings provided")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[settings] update", "invalid request body", &req) {
		return
	}

	current, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[settings] update", "failed to load settings")
		return
	}

	patch := settingsPatchFromProto(&req)
	updated := applyPatch(current, patch)
	if err := h.store.Save(updated); err != nil {
		if errors.Is(err, errInvalidTheme) || errors.Is(err, errInvalidMode) {
			httputil.BadRequest(w, "[settings] update", err.Error())
			return
		}
		httputil.InternalError(w, "[settings] update", "failed to persist settings")
		return
	}

	log.Printf("[settings] updated: theme=%s mode=%s at=%s",
		updated.Theme,
		updated.DefaultMode,
		time.Now().UTC().Format(time.RFC3339),
	)

	resp := &apipb.SettingsResponse{Settings: settingsToProto(updated)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[settings] update", "failed to encode response")
		return
	}
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
	if patch.AgentMaxTurns != nil {
		current.AgentMaxTurns = *patch.AgentMaxTurns
	}
	if patch.AgentTimeoutSeconds != nil {
		current.AgentTimeoutSeconds = *patch.AgentTimeoutSeconds
	}
	if patch.AgentRequiresApproval != nil {
		current.AgentRequiresApproval = *patch.AgentRequiresApproval
	}
	if patch.SearchDebounceMs != nil {
		current.SearchDebounceMs = *patch.SearchDebounceMs
	}
	if patch.ToastDurationMs != nil {
		current.ToastDurationMs = *patch.ToastDurationMs
	}
	if patch.ConfirmDestructiveActions != nil {
		current.ConfirmDestructiveActions = *patch.ConfirmDestructiveActions
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
	return current
}

func settingsToProto(s Settings) *domainpb.Settings {
	return &domainpb.Settings{
		Theme:                       s.Theme,
		DefaultMode:                 s.DefaultMode,
		AutoFixup:                   s.AutoFixup,
		MaxFixupAttempts:            int32(s.MaxFixupAttempts),
		MaxAutoRounds:               int32(s.MaxAutoRounds),
		AutoInitializeWorkshop:      s.AutoInitializeWorkshop,
		AutoAdvanceWorkshop:         s.AutoAdvanceWorkshop,
		AutoCascadeWorkshop:         s.AutoCascadeWorkshop,
		AgentMaxTurns:               int32(s.AgentMaxTurns),
		AgentTimeoutSeconds:         int32(s.AgentTimeoutSeconds),
		AgentRequiresApproval:       s.AgentRequiresApproval,
		SearchDebounceMs:            int32(s.SearchDebounceMs),
		ToastDurationMs:             int32(s.ToastDurationMs),
		ConfirmDestructiveActions:   s.ConfirmDestructiveActions,
		ReviewCodeQualityMinScore:   s.ReviewCodeQualityMinScore,
		ReviewTestMinPassRate:       s.ReviewTestMinPassRate,
		ReviewMaxBlockingViolations: int32(s.ReviewMaxBlockingViolations),
		ReviewMaxWarnings:           int32(s.ReviewMaxWarnings),
		ReviewRequireScreenshots:    s.ReviewRequireScreenshots,
		ReviewRequireTests:          s.ReviewRequireTests,
	}
}

func settingsPatchFromProto(req *apipb.UpdateSettingsRequest) SettingsPatch {
	patch := SettingsPatch{}
	if req == nil {
		return patch
	}
	if req.Theme != nil {
		patch.Theme = req.Theme
	}
	if req.DefaultMode != nil {
		s := *req.DefaultMode
		patch.DefaultMode = &s
	}
	if req.AutoFixup != nil {
		v := *req.AutoFixup
		patch.AutoFixup = &v
	}
	if req.MaxFixupAttempts != nil {
		v := int(*req.MaxFixupAttempts)
		patch.MaxFixupAttempts = &v
	}
	if req.MaxAutoRounds != nil {
		v := int(*req.MaxAutoRounds)
		patch.MaxAutoRounds = &v
	}
	if req.AutoInitializeWorkshop != nil {
		v := *req.AutoInitializeWorkshop
		patch.AutoInitializeWorkshop = &v
	}
	if req.AutoAdvanceWorkshop != nil {
		v := *req.AutoAdvanceWorkshop
		patch.AutoAdvanceWorkshop = &v
	}
	if req.AutoCascadeWorkshop != nil {
		v := *req.AutoCascadeWorkshop
		patch.AutoCascadeWorkshop = &v
	}
	if req.AgentMaxTurns != nil {
		v := int(*req.AgentMaxTurns)
		patch.AgentMaxTurns = &v
	}
	if req.AgentTimeoutSeconds != nil {
		v := int(*req.AgentTimeoutSeconds)
		patch.AgentTimeoutSeconds = &v
	}
	if req.AgentRequiresApproval != nil {
		v := *req.AgentRequiresApproval
		patch.AgentRequiresApproval = &v
	}
	if req.SearchDebounceMs != nil {
		v := int(*req.SearchDebounceMs)
		patch.SearchDebounceMs = &v
	}
	if req.ToastDurationMs != nil {
		v := int(*req.ToastDurationMs)
		patch.ToastDurationMs = &v
	}
	if req.ConfirmDestructiveActions != nil {
		v := *req.ConfirmDestructiveActions
		patch.ConfirmDestructiveActions = &v
	}
	if req.ReviewCodeQualityMinScore != nil {
		v := *req.ReviewCodeQualityMinScore
		patch.ReviewCodeQualityMinScore = &v
	}
	if req.ReviewTestMinPassRate != nil {
		v := *req.ReviewTestMinPassRate
		patch.ReviewTestMinPassRate = &v
	}
	if req.ReviewMaxBlockingViolations != nil {
		v := int(*req.ReviewMaxBlockingViolations)
		patch.ReviewMaxBlockingViolations = &v
	}
	if req.ReviewMaxWarnings != nil {
		v := int(*req.ReviewMaxWarnings)
		patch.ReviewMaxWarnings = &v
	}
	if req.ReviewRequireScreenshots != nil {
		v := *req.ReviewRequireScreenshots
		patch.ReviewRequireScreenshots = &v
	}
	if req.ReviewRequireTests != nil {
		v := *req.ReviewRequireTests
		patch.ReviewRequireTests = &v
	}
	return patch
}

func isEmptyUpdateSettingsRequest(req *apipb.UpdateSettingsRequest) bool {
	if req == nil {
		return true
	}
	return req.Theme == nil &&
		req.DefaultMode == nil &&
		req.AutoFixup == nil &&
		req.MaxFixupAttempts == nil &&
		req.MaxAutoRounds == nil &&
		req.AutoInitializeWorkshop == nil &&
		req.AutoAdvanceWorkshop == nil &&
		req.AutoCascadeWorkshop == nil &&
		req.AgentMaxTurns == nil &&
		req.AgentTimeoutSeconds == nil &&
		req.AgentRequiresApproval == nil &&
		req.SearchDebounceMs == nil &&
		req.ToastDurationMs == nil &&
		req.ConfirmDestructiveActions == nil &&
		req.ReviewCodeQualityMinScore == nil &&
		req.ReviewTestMinPassRate == nil &&
		req.ReviewMaxBlockingViolations == nil &&
		req.ReviewMaxWarnings == nil &&
		req.ReviewRequireScreenshots == nil &&
		req.ReviewRequireTests == nil
}
