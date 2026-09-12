package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// PersistRecordingSession snapshots browser storage and tabs into the selected
// session profile. Profile persistence is intentionally separated from browser
// recording lifecycle and action capture.
func (h *Handler) PersistRecordingSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), recordModeTimeout)
	defer cancel()
	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "sessionId"}))
		return
	}
	if err := h.persistSessionProfile(ctx, sessionID); err != nil {
		h.log.WithError(err).WithField("session_id", sessionID).Warn("Failed to persist session profile")
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{"error": err.Error()}))
		return
	}
	h.respondSuccess(w, http.StatusOK, map[string]string{"status": "persisted", "session_id": sessionID})
}

func (h *Handler) resolveSessionProfile(requestedID string) (*sessionprofilepersistence.SessionProfile, *APIError) {
	if h == nil || h.sessionProfileService == nil {
		return nil, nil
	}
	profile, err := h.sessionProfileService.GetOrCreateProfile(sessionprofilepersistence.ProfileID(strings.TrimSpace(requestedID)))
	if err == nil {
		return profile, nil
	}
	if h.log != nil {
		h.log.WithError(err).Error("Failed to resolve session profile")
	}
	if strings.Contains(err.Error(), "not found") {
		return nil, ErrExecutionNotFound.WithMessage("Session profile not found")
	}
	return nil, ErrInternalServer.WithDetails(map[string]string{"error": err.Error()})
}

func (h *Handler) setActiveSessionProfile(sessionID, profileID string) {
	if h.sessionProfileService != nil {
		h.sessionProfileService.SetActiveSession(sessionID, profileID)
	}
}
func (h *Handler) clearActiveSessionProfile(sessionID string) string {
	if h.sessionProfileService != nil {
		return h.sessionProfileService.ClearActiveSession(sessionID)
	}
	return ""
}
func (h *Handler) getActiveSessionProfile(sessionID string) string {
	if h.sessionProfileService != nil {
		return h.sessionProfileService.GetActiveSession(sessionID)
	}
	return ""
}

func (h *Handler) persistSessionProfile(ctx context.Context, sessionID string) error {
	if h.sessionProfileService == nil {
		return nil
	}
	profileID := h.getActiveSessionProfile(sessionID)
	if profileID == "" {
		return nil
	}
	state, err := h.recordModeService.GetStorageState(ctx, sessionID)
	if err != nil {
		return err
	}
	if len(state) > 0 {
		if _, err := h.sessionProfileService.SaveStorageState(sessionprofilepersistence.ProfileID(profileID), state); err != nil {
			h.log.WithError(err).WithField("profile_id", profileID).Warn("Failed to persist session profile storage state")
		}
	}
	pages, activePageID, err := h.recordModeService.GetOpenPages(sessionID)
	if err != nil {
		h.log.WithError(err).WithField("profile_id", profileID).Warn("Failed to capture open tabs during persist")
		return nil
	}
	if len(pages) == 0 {
		return nil
	}
	tabs := make([]sessionprofilepersistence.TabState, 0, len(pages))
	for index, page := range pages {
		tabs = append(tabs, sessionprofilepersistence.TabState{URL: page.URL, Title: page.Title, IsActive: page.ID == activePageID, Order: index})
	}
	if _, err := h.sessionProfileService.SaveOpenTabs(sessionprofilepersistence.ProfileID(profileID), tabs); err != nil {
		h.log.WithError(err).WithField("profile_id", profileID).Warn("Failed to persist session profile open tabs")
	}
	return nil
}
