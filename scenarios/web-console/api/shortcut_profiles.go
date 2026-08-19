// DOC: docs/concepts/ARCHITECTURE.md#file-map
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/concepts/GLOSSARY.md#shortcut-profile

package main

import (
	"context"
	"sync"
	"time"
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
		Command:     "vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
		Description: "AI coding assistant with full permissions",
	},
	{
		Label:       "Codex",
		Command:     "codex --yolo",
		Description: "OpenAI Codex CLI in auto-approve mode",
	},
	{
		Label:       "OpenCode",
		Command:     "opencode",
		Description: "OpenCode TUI — conversation captured via its local server API",
	},
	{
		Label:       "Grok",
		Command:     "grok",
		Description: "xAI Grok CLI — conversation captured from its session transcript",
	},
	{
		Label:       "Claude Code (attributed)",
		Command:     "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent claude -- --dangerously-skip-permissions; fi; exec vrooli agent launch --runner claude --arg=--dangerously-skip-permissions",
		Description: "Claude Code with best-effort Agent Manager attribution; direct fallback stays available",
	},
	{
		Label:       "Codex (attributed)",
		Command:     "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent codex -- --yolo; fi; exec codex --yolo",
		Description: "Codex with best-effort Agent Manager attribution; direct fallback stays available",
	},
	{
		Label:       "OpenCode (attributed)",
		Command:     "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent opencode --; fi; exec opencode",
		Description: "OpenCode with best-effort Agent Manager attribution; direct fallback stays available",
	},
	{
		Label:       "Grok (attributed)",
		Command:     "if command -v vrooli-agent-launcher >/dev/null 2>&1; then exec vrooli-agent-launcher --agent grok --; fi; exec grok",
		Description: "Grok with best-effort Agent Manager attribution; direct fallback stays available",
	},
}

// legacyDefaultShortcutSets are prior built-in default shortcut lists that
// shipped before OpenCode/Grok were added. A persisted "default" service
// profile (seeded by older builds that wrote it to SQLite at boot) is only
// reconciled to the current defaults when its content exactly matches one of
// these — i.e. it is provably the unmodified seed, never a user customization.
// Append a snapshot here whenever defaultShortcuts changes, so the prior shape
// stays recognizable as "untouched seed".
var legacyDefaultShortcutSets = [][]ShortcutEntry{
	{
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
	},
	{
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
		{
			Label:       "OpenCode",
			Command:     "opencode",
			Description: "OpenCode TUI — conversation captured via its local server API",
		},
		{
			Label:       "Grok",
			Command:     "grok",
			Description: "xAI Grok CLI — conversation captured from its session transcript",
		},
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
func (s *ShortcutProfileStore) List(_ context.Context) []*ShortcutProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ShortcutProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		result = append(result, p)
	}
	return result
}

// Get returns a profile by ID.
func (s *ShortcutProfileStore) Get(_ context.Context, id string) (*ShortcutProfile, bool) {
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
func (s *ShortcutProfileStore) Upsert(_ context.Context, id, scope, name string, shortcuts []ShortcutEntry) *ShortcutProfile {
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
func (s *ShortcutProfileStore) Delete(_ context.Context, id string) bool {
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
func (s *ShortcutProfileStore) Effective(_ context.Context) []ShortcutEntry {
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
