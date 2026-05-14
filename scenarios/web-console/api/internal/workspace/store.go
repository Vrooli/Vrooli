package workspace

import "errors"

// Store abstracts workspace persistence. MemStore is in-memory (tests),
// SQLStore is SQLite-backed (production).
type Store interface {
	// GetLayout returns the full workspace state: ordered panes and tab groups.
	GetLayout() (Layout, error)
	// SavePaneOrder persists pane ordering and the active pane selection.
	SavePaneOrder(activePaneID string, paneOrder []string) error
	// UpsertPane creates or updates a pane's metadata. Replay-safe.
	UpsertPane(p Pane) error
	// DeletePane removes pane metadata. Idempotent.
	DeletePane(sessionID string) error
	// CreateGroup adds a new tab group.
	CreateGroup(name, color string) (Group, error)
	// UpdateGroup modifies a tab group; nil-pointer fields are left unchanged.
	// Returns ErrGroupNotFound if id does not exist.
	UpdateGroup(id string, name *string, color *string, collapsed *bool) (Group, error)
	// DeleteGroup removes a tab group. Returns true if a group was actually
	// removed. Panes referencing the group have their group cleared.
	DeleteGroup(id string) (bool, error)
}

// ErrGroupNotFound is returned by UpdateGroup when the target id is unknown.
var ErrGroupNotFound = errors.New("workspace: group not found")
