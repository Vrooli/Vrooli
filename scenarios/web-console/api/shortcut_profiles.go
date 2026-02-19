package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// [REQ:P1-002a] Shortcut Profile Storage
//
// Shortcut profiles are stored in-memory with a scope hierarchy:
// service → workspace → parent. The effective shortcut list is resolved
// by selecting the highest-priority scope's profile.

// ShortcutEntry represents a single launch shortcut.
type ShortcutEntry struct {
	Label       string `json:"label"`
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
}

// ShortcutProfile represents a named set of shortcuts at a given scope.
type ShortcutProfile struct {
	ID        string          `json:"id"`
	Scope     string          `json:"scope"`
	Name      string          `json:"name"`
	Shortcuts []ShortcutEntry `json:"shortcuts"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

// ShortcutProfileStore manages shortcut profiles in memory.
type ShortcutProfileStore struct {
	mu       sync.RWMutex
	profiles map[string]*ShortcutProfile
}

// defaultShortcuts are the built-in shortcuts per PRD (OT-P0-006).
var defaultShortcuts = []ShortcutEntry{
	{
		Label:       "Claude Code",
		Command:     "claude --dangerously-skip-permissions",
		Description: "AI coding assistant with full permissions",
	},
	{
		Label:       "Codex",
		Command:     "codex --yolo",
		Description: "OpenAI Codex CLI in auto-approve mode",
	},
}

// NewShortcutProfileStore creates a store pre-populated with the default
// service-scope profile.
func NewShortcutProfileStore() *ShortcutProfileStore {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	store := &ShortcutProfileStore{
		profiles: make(map[string]*ShortcutProfile),
	}
	store.profiles["default"] = &ShortcutProfile{
		ID:        "default",
		Scope:     "service",
		Name:      "Default",
		Shortcuts: defaultShortcuts,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return store
}

// validScopes defines the allowed scope values.
var validScopes = map[string]bool{
	"service":   true,
	"workspace": true,
	"parent":    true,
}

// scopePriority defines merge order: higher number = higher priority.
var scopePriority = map[string]int{
	"service":   1,
	"workspace": 2,
	"parent":    3,
}

// List returns all profiles.
func (s *ShortcutProfileStore) List() []*ShortcutProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ShortcutProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		result = append(result, p)
	}
	return result
}

// Get returns a profile by ID.
func (s *ShortcutProfileStore) Get(id string) (*ShortcutProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[id]
	return p, ok
}

// shortcutsEqual reports whether two shortcut slices are identical.
func shortcutsEqual(a, b []ShortcutEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Label != b[i].Label || a[i].Command != b[i].Command || a[i].Description != b[i].Description {
			return false
		}
	}
	return true
}

// Upsert creates or updates a profile. Returns the saved profile.
// Replay-safe: if the content (scope, name, shortcuts) is identical to the
// existing profile, the UpdatedAt timestamp is preserved — repeated calls
// with the same payload are a no-op.
func (s *ShortcutProfileStore) Upsert(id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if existing, ok := s.profiles[id]; ok {
		// Only bump UpdatedAt when content actually changed
		if existing.Scope != scope || existing.Name != name || !shortcutsEqual(existing.Shortcuts, shortcuts) {
			existing.Scope = scope
			existing.Name = name
			existing.Shortcuts = shortcuts
			existing.UpdatedAt = now
		}
		return existing
	}
	p := &ShortcutProfile{
		ID:        id,
		Scope:     scope,
		Name:      name,
		Shortcuts: shortcuts,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.profiles[id] = p
	return p
}

// Delete removes a profile by ID. Returns false if not found.
func (s *ShortcutProfileStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.profiles[id]; !ok {
		return false
	}
	delete(s.profiles, id)
	return true
}

// Effective returns the resolved shortcut list by selecting the
// highest-priority scope's profile. Falls back to default shortcuts
// if no profiles exist.
func (s *ShortcutProfileStore) Effective() []ShortcutEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var best *ShortcutProfile
	bestPriority := 0
	for _, p := range s.profiles {
		pri := scopePriority[p.Scope]
		if pri > bestPriority {
			bestPriority = pri
			best = p
		}
	}

	if best == nil {
		return defaultShortcuts
	}
	return best.Shortcuts
}

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
