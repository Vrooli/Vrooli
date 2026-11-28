package workspace

import (
	"context"
	"errors"
)

// Store abstracts workspace persistence. MemStore is in-memory (tests),
// SQLStore is SQLite-backed (production).
type Store interface {
	// GetLayout returns the full workspace state: ordered panes and tab groups.
	GetLayout(ctx context.Context) (Layout, error)
	// SavePaneOrder persists pane ordering and the active pane selection.
	SavePaneOrder(ctx context.Context, activePaneID string, paneOrder []string) error
	// UpsertPane creates or updates a pane's metadata. Replay-safe.
	UpsertPane(ctx context.Context, p Pane) error
	// DeletePane removes pane metadata. Idempotent.
	DeletePane(ctx context.Context, sessionID string) error
	// ReassignPane moves a pane's complete workspace identity to its recovered
	// replacement session. Any default pane created for newSessionID is removed
	// before the original pane is re-keyed.
	ReassignPane(ctx context.Context, oldSessionID, newSessionID string) error
	// CreateGroup adds a new tab group.
	CreateGroup(ctx context.Context, name, color string) (Group, error)
	// UpdateGroup modifies a tab group; nil-pointer fields are left unchanged.
	// Returns ErrGroupNotFound if id does not exist.
	UpdateGroup(ctx context.Context, id string, name *string, color *string, collapsed *bool) (Group, error)
	// DeleteGroup removes a tab group. Returns true if a group was actually
	// removed. Panes referencing the group have their group cleared, and its
	// roles are removed with it — a role has no meaning outside its group.
	DeleteGroup(ctx context.Context, id string) (bool, error)

	// ListRoles returns roles ordered by group then sort_order. A non-empty
	// groupID filters to that group.
	ListRoles(ctx context.Context, groupID string) ([]Role, error)
	// CreateRole adds a role to a group. Returns ErrInvalidRole when GroupID
	// is blank.
	CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error)
	// UpdateRole modifies a role; only fields with a Has* flag are applied.
	// Returns ErrRoleNotFound if id does not exist.
	UpdateRole(ctx context.Context, req UpdateRoleRequest) (Role, error)
	// DeleteRole removes a role. Idempotent; returns true if a row went away.
	DeleteRole(ctx context.Context, id string) (bool, error)
	// ReassignRoleSession moves a role's session pointer to the recovered
	// replacement session, mirroring ReassignPane. Without this a recovered
	// session would leave its role pointing at an id that no longer exists.
	// Idempotent: a no-op when no role holds oldSessionID.
	ReassignRoleSession(ctx context.Context, oldSessionID, newSessionID string) error
}

// ErrGroupNotFound is returned by UpdateGroup when the target id is unknown.
var ErrGroupNotFound = errors.New("workspace: group not found")

// ErrRoleNotFound is returned by UpdateRole when the target id is unknown.
var ErrRoleNotFound = errors.New("workspace: role not found")

// ErrInvalidRole is returned when a role write omits a required field
// (today: a blank group id). Callers map it to CodeInvalidArgument.
var ErrInvalidRole = errors.New("workspace: invalid role")
