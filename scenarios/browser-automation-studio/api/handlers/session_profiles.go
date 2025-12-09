package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vrooli/browser-automation-studio/services/recording"
)

type sessionProfileResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	LastUsedAt      string `json:"last_used_at"`
	HasStorageState bool   `json:"has_storage_state"`
}

func toSessionProfileResponse(p *recording.SessionProfile) sessionProfileResponse {
	if p == nil {
		return sessionProfileResponse{}
	}
	return sessionProfileResponse{
		ID:              p.ID,
		Name:            p.Name,
		CreatedAt:       p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       p.UpdatedAt.Format(time.RFC3339),
		LastUsedAt:      p.LastUsedAt.Format(time.RFC3339),
		HasStorageState: len(p.StorageState) > 0,
	}
}

// ListRecordingSessionProfiles returns all persisted session profiles.
func (h *Handler) ListRecordingSessionProfiles(w http.ResponseWriter, _ *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profiles, err := h.sessionProfiles.List()
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	response := make([]sessionProfileResponse, 0, len(profiles))
	for i := range profiles {
		response = append(response, toSessionProfileResponse(&profiles[i]))
	}

	h.respondSuccess(w, http.StatusOK, map[string]any{
		"profiles": response,
	})
}

// CreateRecordingSessionProfile creates a new empty session profile.
func (h *Handler) CreateRecordingSessionProfile(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	profile, err := h.sessionProfiles.Create(req.Name)
	if err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusCreated, toSessionProfileResponse(profile))
}

// UpdateRecordingSessionProfile renames an existing profile.
func (h *Handler) UpdateRecordingSessionProfile(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "name",
		}))
		return
	}

	profile, err := h.sessionProfiles.Rename(profileID, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, toSessionProfileResponse(profile))
}

// DeleteRecordingSessionProfile removes a profile and any in-memory session mapping.
func (h *Handler) DeleteRecordingSessionProfile(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfiles == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	profileID := chi.URLParam(r, "profileId")
	if strings.TrimSpace(profileID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "profileId",
		}))
		return
	}

	if err := h.sessionProfiles.Delete(profileID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.respondError(w, ErrExecutionNotFound.WithMessage("Session profile not found"))
			return
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.removeProfileSessions(profileID)

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status": "deleted",
		"id":     profileID,
	})
}

// removeProfileSessions clears any active session mappings for the given profile.
func (h *Handler) removeProfileSessions(profileID string) {
	h.activeSessionsMu.Lock()
	defer h.activeSessionsMu.Unlock()
	for sessionID, pid := range h.activeSessions {
		if pid == profileID {
			delete(h.activeSessions, sessionID)
		}
	}
}
