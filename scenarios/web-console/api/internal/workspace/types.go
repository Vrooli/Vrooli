// Package workspace owns the workspace-layout domain: panes, tab groups,
// and the active-pane selection. Persistence is exposed via the Store
// interface; production wires SQLStore, tests use MemStore.
//
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SEAMS.md#workspace-store
package workspace

import "time"

// Pane is the canonical workspace pane shape. IsActive is a storage detail
// (which pane is currently selected); the wire surface exposes it via
// Layout.ActivePane rather than per-pane.
type Pane struct {
	SessionID            string `json:"session_id"`
	Name                 string `json:"name"`
	HeaderColor          string `json:"header_color"`
	ThemeID              string `json:"theme_id"`
	FontSize             int    `json:"font_size"`
	SortOrder            int    `json:"sort_order"`
	GroupID              string `json:"group_id,omitempty"`
	IsActive             bool   `json:"-"`
	SupportsMessagesView bool   `json:"supports_messages_view"`
	CreatedAt            string `json:"created_at,omitempty"`
	UpdatedAt            string `json:"updated_at,omitempty"`
}

// Group is a named group of tabs with a display color.
type Group struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	SortOrder   int    `json:"sort_order"`
	IsCollapsed bool   `json:"is_collapsed"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// Layout is the full workspace state returned by GetLayout.
type Layout struct {
	ActivePane string  `json:"active_pane"`
	Panes      []Pane  `json:"panes"`
	Groups     []Group `json:"groups"`
}

// UpdatePaneRequest carries optional pane-field overrides. Each Has* flag
// indicates whether the paired field should be applied.
type UpdatePaneRequest struct {
	SessionID string

	Name                    string
	HasName                 bool
	HeaderColor             string
	HasHeaderColor          bool
	ThemeID                 string
	HasThemeID              bool
	FontSize                int
	HasFontSize             bool
	SortOrder               int
	HasSortOrder            bool
	GroupID                 string
	HasGroupID              bool
	SupportsMessagesView    bool
	HasSupportsMessagesView bool
}

// UpdateGroupRequest carries optional group-field overrides.
type UpdateGroupRequest struct {
	ID             string
	Name           string
	HasName        bool
	Color          string
	HasColor       bool
	IsCollapsed    bool
	HasIsCollapsed bool
}

// Defaults applied when creating panes without explicit values.
const (
	DefaultPaneName        = "terminal"
	DefaultPaneHeaderColor = "transparent"
	DefaultPaneThemeID     = "default"
	DefaultPaneFontSize    = 14
)

// FormatTime returns a UTC RFC3339Nano string suitable for the CreatedAt /
// UpdatedAt columns.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
