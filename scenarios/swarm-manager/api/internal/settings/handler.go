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
}

// SettingsPatch allows partial updates.
type SettingsPatch struct {
	Theme *string `json:"theme,omitempty"`
}

// Store persists settings on disk.
type Store struct {
	path string
}

var errInvalidTheme = errors.New("invalid theme")

// NewStore creates a settings store. If path is empty, uses the scenario default.
func NewStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(pathutil.ResolveScenarioRoot("swarm-manager"), ".vrooli", "settings.json")
	}
	return &Store{path: path}
}

// DefaultSettings returns the baseline settings.
func DefaultSettings() Settings {
	return Settings{
		Theme: "dark",
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

func normalizeSettings(settings Settings) Settings {
	if strings.TrimSpace(settings.Theme) == "" {
		settings.Theme = "dark"
	}
	return settings
}

func validateSettings(settings Settings) error {
	switch settings.Theme {
	case "dark", "light", "system":
		return nil
	default:
		return errInvalidTheme
	}
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
		if errors.Is(err, errInvalidTheme) {
			httputil.BadRequest(w, "[settings] update", err.Error())
			return
		}
		httputil.InternalError(w, "[settings] update", "failed to persist settings")
		return
	}

	log.Printf("[settings] updated: theme=%s at=%s",
		updated.Theme,
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
	return current
}

func settingsToProto(settings Settings) *domainpb.Settings {
	return &domainpb.Settings{
		Theme: settings.Theme,
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
	return patch
}

func isEmptyUpdateSettingsRequest(req *apipb.UpdateSettingsRequest) bool {
	if req == nil {
		return true
	}
	return req.Theme == nil
}
