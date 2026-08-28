// Package workspace owns the workspace-layout domain: panes, tab groups,
// and the active-pane selection. Persistence is exposed via the Store
// interface; production wires SQLStore, tests use MemStore.
//
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SEAMS.md#workspace-store-seam-api-ui
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
	// ManuallyUnread is the user's own "come back to this" flag. It is
	// deliberately separate from the conversation read cursor: the cursor
	// only moves forward (it records what was actually displayed) and exists
	// only for message-capable sessions, so it cannot express this.
	ManuallyUnread bool   `json:"manually_unread"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
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

// Role is a named position inside a group.
//
// SessionID is empty while the role is WAITING: the role holds a command and
// no process, so it costs no PTY. That distinction is what lets the console
// tell a finished group (close it) from a half-started one (keep it).
//
// A role is not a pane. A pane is the runtime projection of a live session
// and is keyed BY session id; a role is the durable identity inside a group.
// They are joined by SessionID, and a pane whose session appears in no role
// is an ordinary hand-grouped session. Roles are optional.
//
// Command is a plain string, never an enum of known agents, so any executable
// the operator can type is expressible as a role.
type Role struct {
	ID         string `json:"id"`
	GroupID    string `json:"group_id"`
	Label      string `json:"label"`
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir"`
	// IncomingPrompt may contain at most one {{payload}} placeholder. It lives
	// on the RECEIVING role so any sender that hands off to this role gets
	// this role's framing, which is what makes handoffs compose.
	IncomingPrompt string `json:"incoming_prompt"`
	Backend        string `json:"backend"`
	TargetID       string `json:"target_id"`
	// SessionID is empty while the role is waiting.
	SessionID string `json:"session_id,omitempty"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// IsWaiting reports whether the role holds no session, and so consumes no
// process. The whole auto-close safety argument rests on this predicate.
func (r Role) IsWaiting() bool { return r.SessionID == "" }

// Layout is the full workspace state returned by GetLayout.
type Layout struct {
	ActivePane string  `json:"active_pane"`
	Panes      []Pane  `json:"panes"`
	Groups     []Group `json:"groups"`
	Roles      []Role  `json:"roles"`
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
	ManuallyUnread          bool
	HasManuallyUnread       bool
}

// CreateRoleRequest is the create payload for a role. GroupID is required.
type CreateRoleRequest struct {
	GroupID        string
	Label          string
	Command        string
	WorkingDir     string
	IncomingPrompt string
	Backend        string
	TargetID       string
	SessionID      string
	SortOrder      int
	// HasSortOrder distinguishes "append to the end" (false) from an explicit
	// position, including position zero.
	HasSortOrder bool
}

// UpdateRoleRequest carries optional role-field overrides. Each Has* flag
// indicates whether the paired field should be applied. Setting
// HasSessionID with an empty SessionID returns the role to waiting.
type UpdateRoleRequest struct {
	ID string

	Label             string
	HasLabel          bool
	Command           string
	HasCommand        bool
	WorkingDir        string
	HasWorkingDir     bool
	IncomingPrompt    string
	HasIncomingPrompt bool
	SessionID         string
	HasSessionID      bool
	SortOrder         int
	HasSortOrder      bool
	Backend           string
	HasBackend        bool
	TargetID          string
	HasTargetID       bool
	GroupID           string
	HasGroupID        bool
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
	DefaultRoleLabel       = "Role"
)

// FormatTime returns a UTC RFC3339Nano string suitable for the CreatedAt /
// UpdatedAt columns.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
