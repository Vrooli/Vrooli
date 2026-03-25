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
	DefaultMode         string `json:"default_mode"`
	DefaultDelaySeconds int64  `json:"default_delay_seconds"`
	AutoFixup           bool   `json:"auto_fixup"`
	MaxFixupAttempts    int    `json:"max_fixup_attempts"`

	// Workshop.
	MaxAutoRounds int `json:"max_auto_rounds"`

	// Agent behavior.
	AgentMaxTurns         int  `json:"agent_max_turns"`
	AgentTimeoutSeconds   int  `json:"agent_timeout_seconds"`
	AgentRequiresApproval bool `json:"agent_requires_approval"`

	// UI preferences.
	SearchDebounceMs          int  `json:"search_debounce_ms"`
	ToastDurationMs           int  `json:"toast_duration_ms"`
	ConfirmDestructiveActions bool `json:"confirm_destructive_actions"`
}

// SettingsPatch allows partial updates.
type SettingsPatch struct {
	Theme *string `json:"theme,omitempty"`

	DefaultMode         *string `json:"default_mode,omitempty"`
	DefaultDelaySeconds *int64  `json:"default_delay_seconds,omitempty"`
	AutoFixup           *bool   `json:"auto_fixup,omitempty"`
	MaxFixupAttempts    *int    `json:"max_fixup_attempts,omitempty"`

	MaxAutoRounds *int `json:"max_auto_rounds,omitempty"`

	AgentMaxTurns         *int  `json:"agent_max_turns,omitempty"`
	AgentTimeoutSeconds   *int  `json:"agent_timeout_seconds,omitempty"`
	AgentRequiresApproval *bool `json:"agent_requires_approval,omitempty"`

	SearchDebounceMs          *int  `json:"search_debounce_ms,omitempty"`
	ToastDurationMs           *int  `json:"toast_duration_ms,omitempty"`
	ConfirmDestructiveActions *bool `json:"confirm_destructive_actions,omitempty"`
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
		DefaultMode:               "manual",
		DefaultDelaySeconds:       300,
		AutoFixup:                 false,
		MaxFixupAttempts:          2,
		MaxAutoRounds:             10,
		AgentMaxTurns:             60,
		AgentTimeoutSeconds:       900,
		AgentRequiresApproval:     true,
		SearchDebounceMs:          300,
		ToastDurationMs:           5000,
		ConfirmDestructiveActions: true,
	}
}

// isUninitialized detects whether settings were loaded from an old file
// that lacks the new fields (all zero values). MaxAutoRounds must be >= 1
// in valid settings, so 0 is our sentinel for uninitialized.
func isUninitialized(s Settings) bool {
	return s.MaxAutoRounds == 0
}

// Load retrieves settings from disk, returning defaults when missing.
func (s *Store) Load() (Settings, error) {
	data, err := storage.ReadJSONBytes(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defaults := DefaultSettings()
			// Attempt migration from legacy execution-policy.json.
			migrateExecutionPolicy(&defaults, s.path)
			return defaults, nil
		}
		return Settings{}, err
	}
	if len(data) == 0 {
		defaults := DefaultSettings()
		migrateExecutionPolicy(&defaults, s.path)
		return defaults, nil
	}

	settings := DefaultSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	// If loaded from an old file with only theme, fill in defaults.
	if isUninitialized(settings) {
		defaults := DefaultSettings()
		settings.DefaultMode = defaults.DefaultMode
		settings.DefaultDelaySeconds = defaults.DefaultDelaySeconds
		settings.AutoFixup = defaults.AutoFixup
		settings.MaxFixupAttempts = defaults.MaxFixupAttempts
		settings.MaxAutoRounds = defaults.MaxAutoRounds
		settings.AgentMaxTurns = defaults.AgentMaxTurns
		settings.AgentTimeoutSeconds = defaults.AgentTimeoutSeconds
		settings.AgentRequiresApproval = defaults.AgentRequiresApproval
		settings.SearchDebounceMs = defaults.SearchDebounceMs
		settings.ToastDurationMs = defaults.ToastDurationMs
		settings.ConfirmDestructiveActions = defaults.ConfirmDestructiveActions
		// Try to absorb execution-policy.json values.
		migrateExecutionPolicy(&settings, s.path)
	}

	normalized := normalizeSettings(settings)
	if err := validateSettings(normalized); err != nil {
		return Settings{}, err
	}
	return normalized, nil
}

// legacyPolicy is the shape of the old execution-policy.json.
type legacyPolicy struct {
	DefaultMode         string `json:"default_mode"`
	DefaultDelaySeconds int64  `json:"default_delay_seconds"`
	AutoFixup           bool   `json:"auto_fixup"`
	MaxFixupAttempts    int    `json:"max_fixup_attempts"`
}

// migrateExecutionPolicy reads legacy execution-policy.json, merges values
// into settings, and removes the old file.
func migrateExecutionPolicy(settings *Settings, settingsPath string) {
	policyPath := filepath.Join(filepath.Dir(settingsPath), "execution-policy.json")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return // no legacy file
	}
	var policy legacyPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return
	}
	if policy.DefaultMode != "" {
		settings.DefaultMode = policy.DefaultMode
	}
	if policy.DefaultDelaySeconds > 0 {
		settings.DefaultDelaySeconds = policy.DefaultDelaySeconds
	}
	settings.AutoFixup = policy.AutoFixup
	if policy.MaxFixupAttempts > 0 {
		settings.MaxFixupAttempts = policy.MaxFixupAttempts
	}
	// Remove legacy file after migration.
	_ = os.Remove(policyPath)
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

func normalizeSettings(settings Settings) Settings {
	if strings.TrimSpace(settings.Theme) == "" {
		settings.Theme = "dark"
	}

	// Execution defaults.
	mode := strings.TrimSpace(settings.DefaultMode)
	switch mode {
	case "manual", "scheduled", "yolo":
		settings.DefaultMode = mode
	default:
		settings.DefaultMode = "manual"
	}
	if settings.DefaultDelaySeconds < 0 {
		settings.DefaultDelaySeconds = 0
	}
	settings.MaxFixupAttempts = clampInt(settings.MaxFixupAttempts, 0, 5)

	// Workshop.
	settings.MaxAutoRounds = clampInt(settings.MaxAutoRounds, 1, 50)

	// Agent behavior.
	settings.AgentMaxTurns = clampInt(settings.AgentMaxTurns, 5, 200)
	settings.AgentTimeoutSeconds = clampInt(settings.AgentTimeoutSeconds, 60, 3600)

	// UI preferences.
	settings.SearchDebounceMs = clampInt(settings.SearchDebounceMs, 100, 2000)
	settings.ToastDurationMs = clampInt(settings.ToastDurationMs, 1000, 30000)

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
	case "manual", "scheduled", "yolo":
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
	if patch.DefaultDelaySeconds != nil {
		current.DefaultDelaySeconds = *patch.DefaultDelaySeconds
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
	return current
}

func settingsToProto(s Settings) *domainpb.Settings {
	return &domainpb.Settings{
		Theme:                     s.Theme,
		DefaultMode:               s.DefaultMode,
		DefaultDelaySeconds:       s.DefaultDelaySeconds,
		AutoFixup:                 s.AutoFixup,
		MaxFixupAttempts:          int32(s.MaxFixupAttempts),
		MaxAutoRounds:             int32(s.MaxAutoRounds),
		AgentMaxTurns:             int32(s.AgentMaxTurns),
		AgentTimeoutSeconds:       int32(s.AgentTimeoutSeconds),
		AgentRequiresApproval:     s.AgentRequiresApproval,
		SearchDebounceMs:          int32(s.SearchDebounceMs),
		ToastDurationMs:           int32(s.ToastDurationMs),
		ConfirmDestructiveActions: s.ConfirmDestructiveActions,
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
	if req.DefaultDelaySeconds != nil {
		v := *req.DefaultDelaySeconds
		patch.DefaultDelaySeconds = &v
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
	return patch
}

func isEmptyUpdateSettingsRequest(req *apipb.UpdateSettingsRequest) bool {
	if req == nil {
		return true
	}
	return req.Theme == nil &&
		req.DefaultMode == nil &&
		req.DefaultDelaySeconds == nil &&
		req.AutoFixup == nil &&
		req.MaxFixupAttempts == nil &&
		req.MaxAutoRounds == nil &&
		req.AgentMaxTurns == nil &&
		req.AgentTimeoutSeconds == nil &&
		req.AgentRequiresApproval == nil &&
		req.SearchDebounceMs == nil &&
		req.ToastDurationMs == nil &&
		req.ConfirmDestructiveActions == nil
}
