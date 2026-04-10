// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SEAMS.md#workspace-store

package main

import "time"

// WorkspaceStore abstracts workspace layout persistence. Implementations may be
// in-memory (for tests) or SQLite-backed (for cross-device sync).
type WorkspaceStore interface {
	// GetLayout returns the full workspace state: ordered panes and tab groups.
	GetLayout() (*WorkspaceLayout, error)
	// SavePaneOrder persists pane ordering and the active pane selection.
	SavePaneOrder(activePaneID string, paneOrder []string) error
	// UpsertPane creates or updates a single pane's metadata. Replay-safe.
	UpsertPane(pane *WorkspacePane) error
	// DeletePane removes pane metadata. Idempotent.
	DeletePane(sessionID string) error
	// CreateGroup adds a new tab group.
	CreateGroup(name, color string) (*TabGroup, error)
	// UpdateGroup modifies a tab group. Nil pointer fields are left unchanged.
	UpdateGroup(id string, name *string, color *string, collapsed *bool) (*TabGroup, error)
	// DeleteGroup removes a tab group. Panes in the group get group_id = NULL.
	// Idempotent: returns true if a group was actually removed.
	DeleteGroup(id string) (bool, error)
}

// WorkspaceLayout is the full workspace state returned by GetLayout.
type WorkspaceLayout struct {
	ActivePane string           `json:"active_pane"`
	Panes      []*WorkspacePane `json:"panes"`
	Groups     []*TabGroup      `json:"groups"`
}

// WorkspacePane is the persisted metadata for a single terminal pane.
type WorkspacePane struct {
	SessionID            string `json:"session_id"`
	Name                 string `json:"name"`
	HeaderColor          string `json:"header_color"`
	ThemeID              string `json:"theme_id"`
	FontSize             int    `json:"font_size"`
	SortOrder            int    `json:"sort_order"`
	GroupID              string `json:"group_id,omitempty"` // empty string = no group
	IsActive             bool   `json:"-"`                  // internal; serialized via WorkspaceLayout.ActivePane
	SupportsMessagesView bool   `json:"supports_messages_view"`
	CreatedAt            string `json:"created_at,omitempty"`
	UpdatedAt            string `json:"updated_at,omitempty"`
}

// TabGroup is a named group of terminal tabs with a display color.
type TabGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sort_order"`
	IsCollapsed bool   `json:"is_collapsed"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// WorkspacePane defaults used when creating new panes.
const (
	defaultPaneName        = "terminal"
	defaultPaneHeaderColor = "transparent"
	defaultPaneThemeID     = "default"
	defaultPaneFontSize    = 14
)

// formatTime formats a time.Time as UTC RFC3339Nano for JSON responses.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
