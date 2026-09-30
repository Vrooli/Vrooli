package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
)

// HistoryCallback receives history entries from the playwright driver.
// POST /internal/history-callback
//
// RESTException: webhook_receiver — fire-and-forget history-entry sink posted
// by the out-of-process playwright-driver. Bound to the driver protocol, not
// RPC-shaped. The user-facing history list/delete/navigate surface lives on
// RecordingsService over Connect-RPC.
func (h *Handler) HistoryCallback(w http.ResponseWriter, r *http.Request) {
	if h.sessionProfileService == nil {
		h.respondError(w, ErrServiceUnavailable.WithMessage("Session profiles are not configured"))
		return
	}

	var req struct {
		SessionID      string                                 `json:"session_id"`
		Entry          sessionprofilepersistence.HistoryEntry `json:"entry"`
		NavigationType string                                 `json:"navigation_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"error": "Invalid JSON body: " + err.Error(),
		}))
		return
	}

	if strings.TrimSpace(req.SessionID) == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{
			"field": "session_id",
		}))
		return
	}

	// Look up profile for this session
	profileID := h.sessionProfileService.GetActiveSession(req.SessionID)
	if profileID == "" {
		// No profile associated with this session - might be execution mode
		h.respondSuccess(w, http.StatusOK, map[string]string{
			"status":  "ignored",
			"message": "No profile associated with session",
		})
		return
	}

	// Add the history entry
	if _, err := h.sessionProfileService.AddHistoryEntry(sessionprofilepersistence.ProfileID(profileID), req.Entry); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": err.Error(),
		}))
		return
	}

	h.respondSuccess(w, http.StatusOK, map[string]string{
		"status":     "added",
		"profile_id": profileID,
	})
}
