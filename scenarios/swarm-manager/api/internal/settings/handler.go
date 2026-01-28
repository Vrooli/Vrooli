// Package settings provides filesystem-backed settings persistence.
//
// Settings are stored at scenarios/swarm-manager/.vrooli/settings.json by default.
// This keeps the scenario fully local and git-trackable without DB dependencies.
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
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/storage"
)

// Settings represents persisted configuration for the scenario.
type Settings struct {
	Theme                  string                 `json:"theme"`
	RecommendationMode     string                 `json:"recommendationMode"`
	CustomFocus            string                 `json:"customFocus,omitempty"`
	InsightsEnabled        bool                   `json:"insightsEnabled"`
	InsightsAutoAnalyze    bool                   `json:"insightsAutoAnalyze"`
	RecommendationSources  RecommendationSources  `json:"recommendationSources"`
	RecommendationAutoSync RecommendationAutoSync `json:"recommendationAutoSync"`
}

// RecommendationSources controls which inputs are used to generate recommendations.
type RecommendationSources struct {
	Problems      bool `json:"problems"`
	Completeness  bool `json:"completeness"`
	Tests         bool `json:"tests"`
	Coverage      bool `json:"coverage"`
	CustomFocus   bool `json:"customFocus"`
	ScenarioNotes bool `json:"scenarioNotes"`
}

// RecommendationAutoSync controls automatic refresh behavior.
type RecommendationAutoSync struct {
	Enabled      bool   `json:"enabled"`
	Interval     string `json:"interval"`     // e.g. "15m", "1h"
	LastRefresh  string `json:"lastRefresh"`  // RFC3339 timestamp
	NextRefresh  string `json:"nextRefresh"`  // RFC3339 timestamp
	RefreshScope string `json:"refreshScope"` // "manual" | "scheduled"
}

// SettingsPatch allows partial updates.
type SettingsPatch struct {
	Theme                  *string                      `json:"theme,omitempty"`
	RecommendationMode     *string                      `json:"recommendationMode,omitempty"`
	CustomFocus            *string                      `json:"customFocus,omitempty"`
	InsightsEnabled        *bool                        `json:"insightsEnabled,omitempty"`
	InsightsAutoAnalyze    *bool                        `json:"insightsAutoAnalyze,omitempty"`
	RecommendationSources  *RecommendationSourcesPatch  `json:"recommendationSources,omitempty"`
	RecommendationAutoSync *RecommendationAutoSyncPatch `json:"recommendationAutoSync,omitempty"`
}

// RecommendationSourcesPatch allows partial updates of recommendation sources.
type RecommendationSourcesPatch struct {
	Problems      *bool `json:"problems,omitempty"`
	Completeness  *bool `json:"completeness,omitempty"`
	Tests         *bool `json:"tests,omitempty"`
	Coverage      *bool `json:"coverage,omitempty"`
	CustomFocus   *bool `json:"customFocus,omitempty"`
	ScenarioNotes *bool `json:"scenarioNotes,omitempty"`
}

// RecommendationAutoSyncPatch allows partial updates of auto sync settings.
type RecommendationAutoSyncPatch struct {
	Enabled      *bool   `json:"enabled,omitempty"`
	Interval     *string `json:"interval,omitempty"`
	LastRefresh  *string `json:"lastRefresh,omitempty"`
	NextRefresh  *string `json:"nextRefresh,omitempty"`
	RefreshScope *string `json:"refreshScope,omitempty"`
}

// SettingsResponse wraps settings responses for consistency.
type SettingsResponse struct {
	Settings Settings `json:"settings"`
}

// Store persists settings on disk.
type Store struct {
	path string
}

var (
	errInvalidTheme              = errors.New("invalid theme")
	errInvalidRecommendationMode = errors.New("invalid recommendationMode")
)

// NewStore creates a settings store. If path is empty, uses the scenario default.
func NewStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		path = filepath.Join("scenarios", "swarm-manager", ".vrooli", "settings.json")
	}
	return &Store{path: path}
}

// DefaultSettings returns the baseline settings.
func DefaultSettings() Settings {
	return Settings{
		Theme:                 "dark",
		RecommendationMode:    "off",
		CustomFocus:           "",
		InsightsEnabled:       false,
		InsightsAutoAnalyze:   false,
		RecommendationSources: DefaultRecommendationSources(),
		RecommendationAutoSync: RecommendationAutoSync{
			Enabled:      false,
			Interval:     "1h",
			LastRefresh:  "",
			NextRefresh:  "",
			RefreshScope: "manual",
		},
	}
}

// DefaultRecommendationSources returns the baseline recommendation sources.
func DefaultRecommendationSources() RecommendationSources {
	return RecommendationSources{
		Problems:      true,
		Completeness:  true,
		Tests:         true,
		Coverage:      true,
		CustomFocus:   true,
		ScenarioNotes: true,
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

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}

	// Detect missing recommendationSources to backfill defaults for older settings files.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if _, ok := raw["recommendationSources"]; !ok {
			settings.RecommendationSources = DefaultRecommendationSources()
		}
		if _, ok := raw["recommendationAutoSync"]; !ok {
			settings.RecommendationAutoSync = DefaultSettings().RecommendationAutoSync
		}
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

func normalizeSettings(settings Settings) Settings {
	if strings.TrimSpace(settings.Theme) == "" {
		settings.Theme = "dark"
	}
	if strings.TrimSpace(settings.RecommendationMode) == "" {
		settings.RecommendationMode = "off"
	}
	settings.CustomFocus = strings.TrimSpace(settings.CustomFocus)
	settings.RecommendationSources = normalizeRecommendationSources(settings.RecommendationSources)
	settings.RecommendationAutoSync = normalizeAutoSync(settings.RecommendationAutoSync)
	return settings
}

func normalizeRecommendationSources(sources RecommendationSources) RecommendationSources {
	if sources == (RecommendationSources{}) {
		return DefaultRecommendationSources()
	}
	return sources
}

func normalizeAutoSync(sync RecommendationAutoSync) RecommendationAutoSync {
	if strings.TrimSpace(sync.Interval) == "" {
		sync.Interval = "1h"
	}
	if strings.TrimSpace(sync.RefreshScope) == "" {
		sync.RefreshScope = "manual"
	}
	return sync
}

func validateSettings(settings Settings) error {
	switch settings.Theme {
	case "dark", "light", "system":
		// valid
	default:
		return errInvalidTheme
	}
	switch settings.RecommendationMode {
	case "off", "suggestions", "yolo":
		// valid
	default:
		return errInvalidRecommendationMode
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
	if err := httputil.JSON(w, SettingsResponse{Settings: settings}); err != nil {
		httputil.InternalError(w, "[settings] get", "failed to encode response")
		return
	}
}

// Update applies a partial settings update and persists it.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var patch SettingsPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httputil.BadRequest(w, "[settings] update", "invalid request body")
		return
	}

	if patch == (SettingsPatch{}) {
		httputil.BadRequest(w, "[settings] update", "no settings provided")
		return
	}

	current, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[settings] update", "failed to load settings")
		return
	}

	updated := applyPatch(current, patch)
	if err := h.store.Save(updated); err != nil {
		if errors.Is(err, errInvalidTheme) || errors.Is(err, errInvalidRecommendationMode) {
			httputil.BadRequest(w, "[settings] update", err.Error())
			return
		}
		httputil.InternalError(w, "[settings] update", "failed to persist settings")
		return
	}

	log.Printf("[settings] updated: theme=%s mode=%s focus=%q insights=%v auto=%v at=%s",
		updated.Theme,
		updated.RecommendationMode,
		updated.CustomFocus,
		updated.InsightsEnabled,
		updated.InsightsAutoAnalyze,
		time.Now().UTC().Format(time.RFC3339),
	)

	if err := httputil.JSON(w, SettingsResponse{Settings: updated}); err != nil {
		httputil.InternalError(w, "[settings] update", "failed to encode response")
		return
	}
}

func applyPatch(current Settings, patch SettingsPatch) Settings {
	if patch.Theme != nil {
		current.Theme = strings.TrimSpace(*patch.Theme)
	}
	if patch.RecommendationMode != nil {
		current.RecommendationMode = strings.TrimSpace(*patch.RecommendationMode)
	}
	if patch.CustomFocus != nil {
		current.CustomFocus = strings.TrimSpace(*patch.CustomFocus)
	}
	if patch.InsightsEnabled != nil {
		current.InsightsEnabled = *patch.InsightsEnabled
	}
	if patch.InsightsAutoAnalyze != nil {
		current.InsightsAutoAnalyze = *patch.InsightsAutoAnalyze
	}
	if patch.RecommendationSources != nil {
		applyRecommendationSources(&current.RecommendationSources, patch.RecommendationSources)
	}
	if patch.RecommendationAutoSync != nil {
		applyAutoSyncPatch(&current.RecommendationAutoSync, patch.RecommendationAutoSync)
	}
	return current
}

func applyRecommendationSources(current *RecommendationSources, patch *RecommendationSourcesPatch) {
	if patch.Problems != nil {
		current.Problems = *patch.Problems
	}
	if patch.Completeness != nil {
		current.Completeness = *patch.Completeness
	}
	if patch.Tests != nil {
		current.Tests = *patch.Tests
	}
	if patch.Coverage != nil {
		current.Coverage = *patch.Coverage
	}
	if patch.CustomFocus != nil {
		current.CustomFocus = *patch.CustomFocus
	}
	if patch.ScenarioNotes != nil {
		current.ScenarioNotes = *patch.ScenarioNotes
	}
}

func applyAutoSyncPatch(current *RecommendationAutoSync, patch *RecommendationAutoSyncPatch) {
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.Interval != nil {
		current.Interval = strings.TrimSpace(*patch.Interval)
	}
	if patch.LastRefresh != nil {
		current.LastRefresh = strings.TrimSpace(*patch.LastRefresh)
	}
	if patch.NextRefresh != nil {
		current.NextRefresh = strings.TrimSpace(*patch.NextRefresh)
	}
	if patch.RefreshScope != nil {
		current.RefreshScope = strings.TrimSpace(*patch.RefreshScope)
	}
}
