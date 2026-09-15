package profiles

import (
	"encoding/json"
	"fmt"
	"net/http"

	"deployment-manager/shared"

	"github.com/gorilla/mux"
)

// LPBSConfigHandler serves LPBS release-config routes on a profile.
type LPBSConfigHandler struct {
	profiles Repository
	repo     LPBSReleaseConfigRepository
	log      func(string, map[string]interface{})
}

// NewLPBSConfigHandler creates a new handler.
func NewLPBSConfigHandler(profiles Repository, repo LPBSReleaseConfigRepository, log func(string, map[string]interface{})) *LPBSConfigHandler {
	return &LPBSConfigHandler{profiles: profiles, repo: repo, log: log}
}

// Get handles GET /api/v1/profiles/{id}/lpbs-config.
func (h *LPBSConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]

	cfg, err := h.repo.Get(r.Context(), profileID)
	if err != nil {
		h.log("failed to get lpbs config", map[string]interface{}{"error": err.Error()})
		shared.JSONError(w, "failed to get lpbs config", http.StatusInternalServerError)
		return
	}
	if cfg == nil {
		// Return an empty config with the profile id so the UI can render a form.
		shared.JSONOK(w, &LPBSReleaseConfig{ProfileID: profileID, DefaultChannel: "stable"})
		return
	}
	shared.JSONOK(w, cfg)
}

// Upsert handles PUT /api/v1/profiles/{id}/lpbs-config.
func (h *LPBSConfigHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]

	// Confirm the profile exists before creating a child row so FK errors are
	// reported as a clear 404 instead of a 500.
	profile, err := h.profiles.Get(r.Context(), profileID)
	if err != nil {
		shared.JSONError(w, fmt.Sprintf("failed to load profile: %v", err), http.StatusInternalServerError)
		return
	}
	if profile == nil {
		shared.JSONError(w, fmt.Sprintf("profile '%s' not found", profileID), http.StatusNotFound)
		return
	}

	var body LPBSReleaseConfig
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		shared.JSONError(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}
	body.ProfileID = profile.ID

	if err := h.repo.Upsert(r.Context(), &body); err != nil {
		h.log("failed to upsert lpbs config", map[string]interface{}{"error": err.Error()})
		shared.JSONError(w, fmt.Sprintf("failed to save lpbs config: %v", err), http.StatusInternalServerError)
		return
	}

	saved, err := h.repo.Get(r.Context(), profile.ID)
	if err != nil || saved == nil {
		shared.JSONError(w, "saved but failed to re-read lpbs config", http.StatusInternalServerError)
		return
	}
	shared.JSONOK(w, saved)
}

// Delete handles DELETE /api/v1/profiles/{id}/lpbs-config.
func (h *LPBSConfigHandler) Delete(w http.ResponseWriter, r *http.Request) {
	profileID := mux.Vars(r)["id"]
	if err := h.repo.Delete(r.Context(), profileID); err != nil {
		shared.JSONError(w, fmt.Sprintf("failed to delete lpbs config: %v", err), http.StatusInternalServerError)
		return
	}
	shared.JSONOK(w, map[string]interface{}{"profile_id": profileID, "deleted": true})
}
