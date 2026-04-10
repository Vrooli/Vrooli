// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/SEAMS.md

package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// ShortcutProfileRequest is the JSON body for creating/updating a profile.
type ShortcutProfileRequest struct {
	ID        string          `json:"id"`
	Scope     string          `json:"scope"`
	Name      string          `json:"name"`
	Shortcuts []ShortcutEntry `json:"shortcuts"`
}

// validateShortcutProfile checks a profile request for validity.
func validateShortcutProfile(req ShortcutProfileRequest) string {
	if req.ID == "" {
		return "Profile ID is required"
	}
	if !validScopes[req.Scope] {
		return "Scope must be 'service', 'workspace', or 'parent'"
	}
	if req.Name == "" {
		return "Profile name is required"
	}
	for i, sc := range req.Shortcuts {
		if sc.Label == "" {
			return fmt.Sprintf("Shortcut label is required (entry %d)", i)
		}
		if sc.Command == "" {
			return fmt.Sprintf("Shortcut command is required (entry %d)", i)
		}
	}
	return ""
}

// handleListShortcutProfiles returns all profiles.
// GET /api/v1/shortcuts/profiles
// [REQ:P1-002a] Shortcut Profile Storage
func (s *Server) handleListShortcutProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.shortcuts.List())
}

// handleGetEffectiveShortcuts returns the resolved shortcut list.
// GET /api/v1/shortcuts
// [REQ:P1-002a] Shortcut Profile Storage
func (s *Server) handleGetEffectiveShortcuts(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.shortcuts.Effective())
}

// handleUpsertShortcutProfile creates or updates a profile.
// PUT /api/v1/shortcuts/profiles
// [REQ:P1-002a] Shortcut Profile Storage
func (s *Server) handleUpsertShortcutProfile(w http.ResponseWriter, r *http.Request) {
	var req ShortcutProfileRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if msg := validateShortcutProfile(req); msg != "" {
		writeCatalogError(w, "invalid_body", msg)
		return
	}

	profile := s.shortcuts.Upsert(req.ID, req.Scope, req.Name, req.Shortcuts)
	writeJSON(w, http.StatusOK, profile)
}

// handleDeleteShortcutProfile deletes a profile by ID.
// DELETE /api/v1/shortcuts/profiles/{id}
// Idempotent: returns 204 whether the profile existed or not, so that retries
// and replays are safe. The post-condition "profile does not exist" is met.
// [REQ:P1-002a] Shortcut Profile Storage
func (s *Server) handleDeleteShortcutProfile(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	s.shortcuts.Delete(id) // result ignored — idempotent
	w.WriteHeader(http.StatusNoContent)
}
