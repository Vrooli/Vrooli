// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SQLWorkspaceStore persists workspace layout in SQLite.
// It implements WorkspaceStore.
type SQLWorkspaceStore struct {
	db *sql.DB
}

// NewSQLWorkspaceStore creates a SQLite-backed workspace store.
func NewSQLWorkspaceStore(db *sql.DB) *SQLWorkspaceStore {
	return &SQLWorkspaceStore{db: db}
}

// GetLayout returns the full workspace layout: ordered panes and tab groups.
func (s *SQLWorkspaceStore) GetLayout() (*WorkspaceLayout, error) {
	// Fetch panes ordered by sort_order
	paneRows, err := s.db.Query(`
		SELECT session_id, name, header_color, theme_id, font_size,
		       sort_order, COALESCE(group_id, ''), is_active,
		       supports_messages_view, created_at, updated_at
		FROM workspace_panes
		ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, fmt.Errorf("query panes: %w", err)
	}
	defer paneRows.Close()

	var panes []*WorkspacePane
	var activePaneID string
	for paneRows.Next() {
		p, err := scanWorkspacePane(paneRows)
		if err != nil {
			log.Printf("SQLWorkspaceStore.GetLayout: scan pane: %v", err)
			continue
		}
		if p.IsActive {
			activePaneID = p.SessionID
		}
		panes = append(panes, p)
	}
	if panes == nil {
		panes = make([]*WorkspacePane, 0)
	}

	// Fetch groups ordered by sort_order
	groupRows, err := s.db.Query(`
		SELECT id, name, color, sort_order, is_collapsed, created_at, updated_at
		FROM tab_groups
		ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, fmt.Errorf("query groups: %w", err)
	}
	defer groupRows.Close()

	var groups []*TabGroup
	for groupRows.Next() {
		g, err := scanTabGroup(groupRows)
		if err != nil {
			log.Printf("SQLWorkspaceStore.GetLayout: scan group: %v", err)
			continue
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = make([]*TabGroup, 0)
	}

	return &WorkspaceLayout{
		ActivePane: activePaneID,
		Panes:      panes,
		Groups:     groups,
	}, nil
}

// SavePaneOrder updates sort_order and is_active for the given pane ordering.
// Uses a single transaction for atomicity.
func (s *SQLWorkspaceStore) SavePaneOrder(activePaneID string, paneOrder []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	now := formatTime(time.Now())

	// Reset all active flags
	if _, err := tx.Exec(`UPDATE workspace_panes SET is_active = 0, updated_at = ?`, now); err != nil {
		return fmt.Errorf("reset active: %w", err)
	}

	// Update sort_order and is_active per pane
	for i, sid := range paneOrder {
		isActive := 0
		if sid == activePaneID {
			isActive = 1
		}
		_, err := tx.Exec(`
			UPDATE workspace_panes
			SET sort_order = ?, is_active = ?, updated_at = ?
			WHERE session_id = ?`,
			i, isActive, now, sid)
		if err != nil {
			return fmt.Errorf("update pane %s: %w", sid, err)
		}
	}

	return tx.Commit()
}

// UpsertPane creates or updates a pane's metadata. Replay-safe via ON CONFLICT.
func (s *SQLWorkspaceStore) UpsertPane(pane *WorkspacePane) error {
	now := formatTime(time.Now())

	// Convert empty group_id to NULL
	var groupID interface{}
	if pane.GroupID != "" {
		groupID = pane.GroupID
	}

	name := pane.Name
	if name == "" {
		name = defaultPaneName
	}
	headerColor := pane.HeaderColor
	if headerColor == "" {
		headerColor = defaultPaneHeaderColor
	}
	themeID := pane.ThemeID
	if themeID == "" {
		themeID = defaultPaneThemeID
	}
	fontSize := pane.FontSize
	if fontSize == 0 {
		fontSize = defaultPaneFontSize
	}

	isActive := 0
	if pane.IsActive {
		isActive = 1
	}
	supportsMessagesView := 0
	if pane.SupportsMessagesView {
		supportsMessagesView = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO workspace_panes (session_id, name, header_color, theme_id, font_size, sort_order, group_id, is_active, supports_messages_view, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (session_id) DO UPDATE SET
			name = excluded.name,
			header_color = excluded.header_color,
			theme_id = excluded.theme_id,
			font_size = excluded.font_size,
			sort_order = excluded.sort_order,
			group_id = excluded.group_id,
			supports_messages_view = excluded.supports_messages_view,
			updated_at = excluded.updated_at`,
		pane.SessionID, name, headerColor, themeID, fontSize,
		pane.SortOrder, groupID, isActive, supportsMessagesView, now, now)
	if err != nil {
		return fmt.Errorf("upsert pane %s: %w", pane.SessionID, err)
	}
	return nil
}

// DeletePane removes pane metadata. Idempotent.
func (s *SQLWorkspaceStore) DeletePane(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM workspace_panes WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("delete pane %s: %w", sessionID, err)
	}
	return nil
}

// CreateGroup adds a new tab group.
func (s *SQLWorkspaceStore) CreateGroup(name, color string) (*TabGroup, error) {
	if name == "" {
		name = "Group"
	}
	if color == "" {
		color = "#3b82f6"
	}

	// Determine next sort_order
	var maxOrder sql.NullInt32
	if err := s.db.QueryRow(`SELECT MAX(sort_order) FROM tab_groups`).Scan(&maxOrder); err != nil {
		return nil, fmt.Errorf("query max sort_order: %w", err)
	}
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int32) + 1
	}

	now := formatTime(time.Now())
	id := uuid.New().String()

	row := s.db.QueryRow(`
		INSERT INTO tab_groups (id, name, color, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, name, color, sort_order, is_collapsed, created_at, updated_at`,
		id, name, color, nextOrder, now, now)

	g, err := scanTabGroupRow(row)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

// UpdateGroup modifies a tab group. Nil pointer fields are left unchanged.
func (s *SQLWorkspaceStore) UpdateGroup(id string, name *string, color *string, collapsed *bool) (*TabGroup, error) {
	// Build dynamic SET clause
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{formatTime(time.Now())}

	if name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *name)
	}
	if color != nil {
		setClauses = append(setClauses, "color = ?")
		args = append(args, *color)
	}
	if collapsed != nil {
		val := 0
		if *collapsed {
			val = 1
		}
		setClauses = append(setClauses, "is_collapsed = ?")
		args = append(args, val)
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE tab_groups SET %s
		WHERE id = ?
		RETURNING id, name, color, sort_order, is_collapsed, created_at, updated_at`,
		strings.Join(setClauses, ", "))

	row := s.db.QueryRow(query, args...)
	g, err := scanTabGroupRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("group not found")
		}
		return nil, fmt.Errorf("update group: %w", err)
	}
	return g, nil
}

// DeleteGroup removes a tab group. ON DELETE SET NULL handles pane cleanup.
// Returns true if a group was actually removed.
func (s *SQLWorkspaceStore) DeleteGroup(id string) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM tab_groups WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete group %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// scanWorkspacePane reads a WorkspacePane from a *sql.Rows cursor.
func scanWorkspacePane(rows *sql.Rows) (*WorkspacePane, error) {
	var p WorkspacePane
	var isActive, supportsMessagesView int
	if err := rows.Scan(
		&p.SessionID, &p.Name, &p.HeaderColor, &p.ThemeID, &p.FontSize,
		&p.SortOrder, &p.GroupID, &isActive, &supportsMessagesView,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	p.IsActive = isActive != 0
	p.SupportsMessagesView = supportsMessagesView != 0
	return &p, nil
}

// scanTabGroup reads a TabGroup from a *sql.Rows cursor.
func scanTabGroup(rows *sql.Rows) (*TabGroup, error) {
	var g TabGroup
	var isCollapsed int
	if err := rows.Scan(&g.ID, &g.Name, &g.Color, &g.SortOrder, &isCollapsed, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return nil, err
	}
	g.IsCollapsed = isCollapsed != 0
	return &g, nil
}

// scanTabGroupRow reads a TabGroup from a *sql.Row.
func scanTabGroupRow(row *sql.Row) (*TabGroup, error) {
	var g TabGroup
	var isCollapsed int
	if err := row.Scan(&g.ID, &g.Name, &g.Color, &g.SortOrder, &isCollapsed, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return nil, err
	}
	g.IsCollapsed = isCollapsed != 0
	return &g, nil
}
