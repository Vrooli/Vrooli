// Package settings provides filesystem-backed settings persistence.
//
// Settings are stored at scenarios/swarm-manager/config/settings.json by default.
// This keeps them git-trackable as shared scenario behavior defaults.
//
// DOC: docs/reference/configuration.md
// DOC: docs/reference/operational-targets.md
// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
package settings

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

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
		apierr.MapError(w, "[settings] get", apierr.Internal("failed to load settings"))
		return
	}
	resp := &apipb.SettingsResponse{Settings: settingsToProto(settings)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[settings] get", apierr.Internal("failed to encode response"))
		return
	}
}

// Update applies a partial settings update and persists it.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req apipb.UpdateSettingsRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[settings] update", apierr.BadRequest("invalid request body"))
		return
	}

	if isEmptyUpdateSettingsRequest(&req) {
		apierr.MapError(w, "[settings] update", apierr.BadRequest("no settings provided"))
		return
	}
	if !httputil.ValidateProtoRequest(w, "[settings] update", "invalid request body", &req) {
		return
	}

	current, err := h.store.Load()
	if err != nil {
		apierr.MapError(w, "[settings] update", apierr.Internal("failed to load settings"))
		return
	}

	patch := settingsPatchFromProto(&req)
	updated := applyPatch(current, patch)
	if err := h.store.Save(updated); err != nil {
		if errors.Is(err, errInvalidTheme) || errors.Is(err, errInvalidMode) {
			apierr.MapError(w, "[settings] update", apierr.BadRequest("%s", err.Error()))
			return
		}
		apierr.MapError(w, "[settings] update", apierr.Internal("failed to persist settings"))
		return
	}

	slog.Info("settings updated",
		"theme", updated.Theme,
		"mode", updated.DefaultMode,
		"at", time.Now().UTC().Format(time.RFC3339),
	)

	resp := &apipb.SettingsResponse{Settings: settingsToProto(updated)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[settings] update", apierr.Internal("failed to encode response"))
		return
	}
}
