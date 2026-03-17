// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// PGWorkspaceStore persists workspace layout in PostgreSQL.
// It implements WorkspaceStore.
type PGWorkspaceStore struct {
	db *sql.DB
}

// NewPGWorkspaceStore creates a PostgreSQL-backed workspace store.
func NewPGWorkspaceStore(db *sql.DB) *PGWorkspaceStore {
	return &PGWorkspaceStore{db: db}
}

// GetLayout returns the full workspace layout: ordered panes and tab groups.
func (s *PGWorkspaceStore) GetLayout() (*WorkspaceLayout, error) {
	// Fetch panes ordered by sort_order
	paneRows, err := s.db.Query(`
		SELECT session_id, name, header_color, theme_id, font_size,
		       sort_order, COALESCE(group_id::text, ''), is_active,
		       created_at, updated_at
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
			log.Printf("PGWorkspaceStore.GetLayout: scan pane: %v", err)
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
			log.Printf("PGWorkspaceStore.GetLayout: scan group: %v", err)
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
func (s *PGWorkspaceStore) SavePaneOrder(activePaneID string, paneOrder []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	now := time.Now().UTC()

	// Reset all active flags
	if _, err := tx.Exec(`UPDATE workspace_panes SET is_active = false, updated_at = $1`, now); err != nil {
		return fmt.Errorf("reset active: %w", err)
	}

	// Update sort_order and is_active per pane
	for i, sid := range paneOrder {
		isActive := sid == activePaneID
		_, err := tx.Exec(`
			UPDATE workspace_panes
			SET sort_order = $1, is_active = $2, updated_at = $3
			WHERE session_id = $4`,
			i, isActive, now, sid)
		if err != nil {
			return fmt.Errorf("update pane %s: %w", sid, err)
		}
	}

	return tx.Commit()
}

// UpsertPane creates or updates a pane's metadata. Replay-safe via ON CONFLICT.
func (s *PGWorkspaceStore) UpsertPane(pane *WorkspacePane) error {
	now := time.Now().UTC()

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

	_, err := s.db.Exec(`
		INSERT INTO workspace_panes (session_id, name, header_color, theme_id, font_size, sort_order, group_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		ON CONFLICT (session_id) DO UPDATE SET
			name = EXCLUDED.name,
			header_color = EXCLUDED.header_color,
			theme_id = EXCLUDED.theme_id,
			font_size = EXCLUDED.font_size,
			sort_order = EXCLUDED.sort_order,
			group_id = EXCLUDED.group_id,
			updated_at = EXCLUDED.updated_at`,
		pane.SessionID, name, headerColor, themeID, fontSize,
		pane.SortOrder, groupID, pane.IsActive, now)
	if err != nil {
		return fmt.Errorf("upsert pane %s: %w", pane.SessionID, err)
	}
	return nil
}

// DeletePane removes pane metadata. Idempotent.
func (s *PGWorkspaceStore) DeletePane(sessionID string) error {
	_, err := s.db.Exec(`DELETE FROM workspace_panes WHERE session_id = $1`, sessionID)
	if err != nil {
		return fmt.Errorf("delete pane %s: %w", sessionID, err)
	}
	return nil
}

// CreateGroup adds a new tab group.
func (s *PGWorkspaceStore) CreateGroup(name, color string) (*TabGroup, error) {
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

	now := time.Now().UTC()
	row := s.db.QueryRow(`
		INSERT INTO tab_groups (name, color, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		RETURNING id, name, color, sort_order, is_collapsed, created_at, updated_at`,
		name, color, nextOrder, now)

	g, err := scanTabGroupRow(row)
	if err != nil {
		return nil, fmt.Errorf("create group: %w", err)
	}
	return g, nil
}

// UpdateGroup modifies a tab group. Nil pointer fields are left unchanged.
func (s *PGWorkspaceStore) UpdateGroup(id string, name *string, color *string, collapsed *bool) (*TabGroup, error) {
	// Build dynamic SET clause
	setClauses := []string{"updated_at = $1"}
	args := []interface{}{time.Now().UTC()}
	argIdx := 2

	if name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *name)
		argIdx++
	}
	if color != nil {
		setClauses = append(setClauses, fmt.Sprintf("color = $%d", argIdx))
		args = append(args, *color)
		argIdx++
	}
	if collapsed != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_collapsed = $%d", argIdx))
		args = append(args, *collapsed)
		argIdx++
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE tab_groups SET %s
		WHERE id = $%d
		RETURNING id, name, color, sort_order, is_collapsed, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx)

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
func (s *PGWorkspaceStore) DeleteGroup(id string) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM tab_groups WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete group %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// scanWorkspacePane reads a WorkspacePane from a *sql.Rows cursor.
func scanWorkspacePane(rows *sql.Rows) (*WorkspacePane, error) {
	var p WorkspacePane
	var createdAt, updatedAt time.Time
	if err := rows.Scan(
		&p.SessionID, &p.Name, &p.HeaderColor, &p.ThemeID, &p.FontSize,
		&p.SortOrder, &p.GroupID, &p.IsActive, &createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}
	p.CreatedAt = formatTime(createdAt)
	p.UpdatedAt = formatTime(updatedAt)
	return &p, nil
}

// scanTabGroup reads a TabGroup from a *sql.Rows cursor.
func scanTabGroup(rows *sql.Rows) (*TabGroup, error) {
	var g TabGroup
	var createdAt, updatedAt time.Time
	if err := rows.Scan(&g.ID, &g.Name, &g.Color, &g.SortOrder, &g.IsCollapsed, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	g.CreatedAt = formatTime(createdAt)
	g.UpdatedAt = formatTime(updatedAt)
	return &g, nil
}

// scanTabGroupRow reads a TabGroup from a *sql.Row.
func scanTabGroupRow(row *sql.Row) (*TabGroup, error) {
	var g TabGroup
	var createdAt, updatedAt time.Time
	if err := row.Scan(&g.ID, &g.Name, &g.Color, &g.SortOrder, &g.IsCollapsed, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	g.CreatedAt = formatTime(createdAt)
	g.UpdatedAt = formatTime(updatedAt)
	return &g, nil
}
