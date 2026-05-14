// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package workspace

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SQLStore persists workspace layout in SQLite.
type SQLStore struct {
	db *sql.DB
}

// NewSQLStore wires a SQLite-backed store.
func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) GetLayout() (Layout, error) {
	paneRows, err := s.db.Query(`
		SELECT session_id, name, header_color, theme_id, font_size,
		       sort_order, COALESCE(group_id, ''), is_active,
		       supports_messages_view, created_at, updated_at
		FROM workspace_panes
		ORDER BY sort_order, created_at`)
	if err != nil {
		return Layout{}, fmt.Errorf("query panes: %w", err)
	}
	defer paneRows.Close()

	panes := make([]Pane, 0)
	var activePaneID string
	for paneRows.Next() {
		p, err := scanPane(paneRows)
		if err != nil {
			log.Printf("workspace.SQLStore.GetLayout: scan pane: %v", err)
			continue
		}
		if p.IsActive {
			activePaneID = p.SessionID
		}
		panes = append(panes, p)
	}

	groupRows, err := s.db.Query(`
		SELECT id, name, color, sort_order, is_collapsed, created_at, updated_at
		FROM tab_groups
		ORDER BY sort_order, created_at`)
	if err != nil {
		return Layout{}, fmt.Errorf("query groups: %w", err)
	}
	defer groupRows.Close()

	groups := make([]Group, 0)
	for groupRows.Next() {
		g, err := scanGroup(groupRows)
		if err != nil {
			log.Printf("workspace.SQLStore.GetLayout: scan group: %v", err)
			continue
		}
		groups = append(groups, g)
	}

	return Layout{
		ActivePane: activePaneID,
		Panes:      panes,
		Groups:     groups,
	}, nil
}

func (s *SQLStore) SavePaneOrder(activePaneID string, paneOrder []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	now := FormatTime(time.Now())

	if _, err := tx.Exec(`UPDATE workspace_panes SET is_active = 0, updated_at = ?`, now); err != nil {
		return fmt.Errorf("reset active: %w", err)
	}

	for i, sid := range paneOrder {
		isActive := 0
		if sid == activePaneID {
			isActive = 1
		}
		if _, err := tx.Exec(`
			UPDATE workspace_panes
			SET sort_order = ?, is_active = ?, updated_at = ?
			WHERE session_id = ?`,
			i, isActive, now, sid); err != nil {
			return fmt.Errorf("update pane %s: %w", sid, err)
		}
	}

	return tx.Commit()
}

func (s *SQLStore) UpsertPane(pane Pane) error {
	now := FormatTime(time.Now())

	var groupID interface{}
	if pane.GroupID != "" {
		groupID = pane.GroupID
	}

	name := pane.Name
	if name == "" {
		name = DefaultPaneName
	}
	headerColor := pane.HeaderColor
	if headerColor == "" {
		headerColor = DefaultPaneHeaderColor
	}
	themeID := pane.ThemeID
	if themeID == "" {
		themeID = DefaultPaneThemeID
	}
	fontSize := pane.FontSize
	if fontSize == 0 {
		fontSize = DefaultPaneFontSize
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

func (s *SQLStore) DeletePane(sessionID string) error {
	if _, err := s.db.Exec(`DELETE FROM workspace_panes WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete pane %s: %w", sessionID, err)
	}
	return nil
}

func (s *SQLStore) CreateGroup(name, color string) (Group, error) {
	if name == "" {
		name = "Group"
	}
	if color == "" {
		color = "#3b82f6"
	}

	var maxOrder sql.NullInt32
	if err := s.db.QueryRow(`SELECT MAX(sort_order) FROM tab_groups`).Scan(&maxOrder); err != nil {
		return Group{}, fmt.Errorf("query max sort_order: %w", err)
	}
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int32) + 1
	}

	now := FormatTime(time.Now())
	id := uuid.New().String()

	row := s.db.QueryRow(`
		INSERT INTO tab_groups (id, name, color, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, name, color, sort_order, is_collapsed, created_at, updated_at`,
		id, name, color, nextOrder, now, now)

	return scanGroupRow(row)
}

func (s *SQLStore) UpdateGroup(id string, name *string, color *string, collapsed *bool) (Group, error) {
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{FormatTime(time.Now())}

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
	g, err := scanGroupRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("update group: %w", err)
	}
	return g, nil
}

func (s *SQLStore) DeleteGroup(id string) (bool, error) {
	result, err := s.db.Exec(`DELETE FROM tab_groups WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete group %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func scanPane(rows *sql.Rows) (Pane, error) {
	var p Pane
	var isActive, supportsMessagesView int
	if err := rows.Scan(
		&p.SessionID, &p.Name, &p.HeaderColor, &p.ThemeID, &p.FontSize,
		&p.SortOrder, &p.GroupID, &isActive, &supportsMessagesView,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Pane{}, err
	}
	p.IsActive = isActive != 0
	p.SupportsMessagesView = supportsMessagesView != 0
	return p, nil
}

func scanGroup(rows *sql.Rows) (Group, error) {
	var g Group
	var isCollapsed int
	if err := rows.Scan(&g.ID, &g.Name, &g.Color, &g.SortOrder, &isCollapsed, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return Group{}, err
	}
	g.IsCollapsed = isCollapsed != 0
	return g, nil
}

func scanGroupRow(row *sql.Row) (Group, error) {
	var g Group
	var isCollapsed int
	if err := row.Scan(&g.ID, &g.Name, &g.Color, &g.SortOrder, &isCollapsed, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return Group{}, err
	}
	g.IsCollapsed = isCollapsed != 0
	return g, nil
}
