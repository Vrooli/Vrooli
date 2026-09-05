// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package workspace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"web-console/internal/dbx"
)

// SQLStore persists workspace layout in SQLite.
type SQLStore struct {
	db dbx.Handle
}

// NewSQLStore wires a SQLite-backed store.
func NewSQLStore(db dbx.Handle) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) GetLayout(ctx context.Context) (Layout, error) {
	paneRows, err := s.db.QueryContext(ctx, `
		SELECT session_id, name, header_color, theme_id, font_size,
		       sort_order, COALESCE(group_id, ''), is_active,
		       supports_messages_view, manually_unread, created_at, updated_at
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

	groupRows, err := s.db.QueryContext(ctx, `
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

	// Roles ride along in the same call: adding a second round trip on load
	// would make every client pay for a feature most workspaces do not use.
	roles, err := s.ListRoles(ctx, "")
	if err != nil {
		return Layout{}, err
	}

	return Layout{
		ActivePane: activePaneID,
		Panes:      panes,
		Groups:     groups,
		Roles:      roles,
	}, nil
}

func (s *SQLStore) SavePaneOrder(ctx context.Context, activePaneID string, paneOrder []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	now := FormatTime(time.Now())

	if _, err := tx.ExecContext(ctx, `UPDATE workspace_panes SET is_active = 0, updated_at = ?`, now); err != nil {
		return fmt.Errorf("reset active: %w", err)
	}

	for i, sid := range paneOrder {
		isActive := 0
		if sid == activePaneID {
			isActive = 1
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE workspace_panes
			SET sort_order = ?, is_active = ?, updated_at = ?
			WHERE session_id = ?`,
			i, isActive, now, sid); err != nil {
			return fmt.Errorf("update pane %s: %w", sid, err)
		}
	}

	return tx.Commit()
}

func (s *SQLStore) UpsertPane(ctx context.Context, pane Pane) error {
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
	manuallyUnread := 0
	if pane.ManuallyUnread {
		manuallyUnread = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspace_panes (session_id, name, header_color, theme_id, font_size, sort_order, group_id, is_active, supports_messages_view, manually_unread, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (session_id) DO UPDATE SET
			name = excluded.name,
			header_color = excluded.header_color,
			theme_id = excluded.theme_id,
			font_size = excluded.font_size,
			sort_order = excluded.sort_order,
			group_id = excluded.group_id,
			supports_messages_view = excluded.supports_messages_view,
			manually_unread = excluded.manually_unread,
			updated_at = excluded.updated_at`,
		pane.SessionID, name, headerColor, themeID, fontSize,
		pane.SortOrder, groupID, isActive, supportsMessagesView, manuallyUnread, now, now)
	if err != nil {
		return fmt.Errorf("upsert pane %s: %w", pane.SessionID, err)
	}
	return nil
}

func (s *SQLStore) DeletePane(ctx context.Context, sessionID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM workspace_panes WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete pane %s: %w", sessionID, err)
	}
	return nil
}

func (s *SQLStore) ReassignPane(ctx context.Context, oldSessionID, newSessionID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin pane reassignment: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	// There is no customized state to migrate when the source pane is absent;
	// retain the replacement's ordinary default pane in that case.
	var sourceExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM workspace_panes WHERE session_id = ?)`, oldSessionID).Scan(&sourceExists); err != nil {
		return fmt.Errorf("check source pane: %w", err)
	}
	if !sourceExists {
		return tx.Commit()
	}

	// The generic create flow may already have inserted a default pane. Delete
	// only that replacement row, then move every original field unchanged.
	if _, err := tx.ExecContext(ctx, `DELETE FROM workspace_panes WHERE session_id = ?`, newSessionID); err != nil {
		return fmt.Errorf("delete replacement pane: %w", err)
	}
	result, err := tx.ExecContext(ctx, `UPDATE workspace_panes SET session_id = ?, updated_at = ? WHERE session_id = ?`, newSessionID, FormatTime(time.Now()), oldSessionID)
	if err != nil {
		return fmt.Errorf("move original pane: %w", err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("count moved pane: %w", err)
	} else if count == 0 {
		return nil
	}
	return tx.Commit()
}

func (s *SQLStore) CreateGroup(ctx context.Context, name, color string) (Group, error) {
	if name == "" {
		name = "Group"
	}
	if color == "" {
		color = "#3b82f6"
	}

	var maxOrder sql.NullInt32
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(sort_order) FROM tab_groups`).Scan(&maxOrder); err != nil {
		return Group{}, fmt.Errorf("query max sort_order: %w", err)
	}
	nextOrder := 0
	if maxOrder.Valid {
		nextOrder = int(maxOrder.Int32) + 1
	}

	now := FormatTime(time.Now())
	id := uuid.New().String()

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO tab_groups (id, name, color, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, name, color, sort_order, is_collapsed, created_at, updated_at`,
		id, name, color, nextOrder, now, now)

	return scanGroupRow(row)
}

func (s *SQLStore) UpdateGroup(ctx context.Context, id string, name *string, color *string, collapsed *bool) (Group, error) {
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

	row := s.db.QueryRowContext(ctx, query, args...)
	g, err := scanGroupRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Group{}, ErrGroupNotFound
		}
		return Group{}, fmt.Errorf("update group: %w", err)
	}
	return g, nil
}

func (s *SQLStore) DeleteGroup(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM tab_groups WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete group %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func scanPane(rows *sql.Rows) (Pane, error) {
	var p Pane
	var isActive, supportsMessagesView, manuallyUnread int
	if err := rows.Scan(
		&p.SessionID, &p.Name, &p.HeaderColor, &p.ThemeID, &p.FontSize,
		&p.SortOrder, &p.GroupID, &isActive, &supportsMessagesView, &manuallyUnread,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Pane{}, err
	}
	p.IsActive = isActive != 0
	p.SupportsMessagesView = supportsMessagesView != 0
	p.ManuallyUnread = manuallyUnread != 0
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

// ---------------------------------------------------------------------------
// Roles
// ---------------------------------------------------------------------------

// roleColumns is the single column list every role read uses, so scanRole and
// scanRoleRow can never disagree with a query about column order.
const roleColumns = `id, group_id, label, command, working_dir, incoming_prompt,
	backend, target_id, COALESCE(session_id, ''), sort_order, created_at, updated_at`

func (s *SQLStore) ListRoles(ctx context.Context, groupID string) ([]Role, error) {
	query := `SELECT ` + roleColumns + ` FROM workspace_roles`
	args := []interface{}{}
	if groupID != "" {
		query += ` WHERE group_id = ?`
		args = append(args, groupID)
	}
	// id breaks ties so the sequence is total: two roles sharing a sort_order
	// must not swap between reads, or the sidebar reorders on every refresh.
	query += ` ORDER BY group_id, sort_order, id`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	roles := make([]Role, 0)
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			log.Printf("workspace.SQLStore.ListRoles: scan role: %v", err)
			continue
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func (s *SQLStore) CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error) {
	if req.GroupID == "" {
		return Role{}, ErrInvalidRole
	}

	sortOrder := req.SortOrder
	if !req.HasSortOrder {
		var maxOrder sql.NullInt32
		if err := s.db.QueryRowContext(ctx,
			`SELECT MAX(sort_order) FROM workspace_roles WHERE group_id = ?`, req.GroupID,
		).Scan(&maxOrder); err != nil {
			return Role{}, fmt.Errorf("query max role sort_order: %w", err)
		}
		if maxOrder.Valid {
			sortOrder = int(maxOrder.Int32) + 1
		} else {
			sortOrder = 0
		}
	}

	label := req.Label
	if label == "" {
		label = DefaultRoleLabel
	}
	now := FormatTime(time.Now())

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO workspace_roles
			(id, group_id, label, command, working_dir, incoming_prompt,
			 backend, target_id, session_id, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING `+roleColumns,
		uuid.New().String(), req.GroupID, label, req.Command, req.WorkingDir,
		req.IncomingPrompt, req.Backend, req.TargetID, nullableSession(req.SessionID),
		sortOrder, now, now)

	r, err := scanRoleRow(row)
	if err != nil {
		// A blank group id is caught above; anything else that fails the
		// group foreign key or the one-session-one-role index is a caller
		// error, not an internal fault.
		if isRoleConstraintError(err) {
			return Role{}, ErrInvalidRole
		}
		return Role{}, fmt.Errorf("create role: %w", err)
	}
	return r, nil
}

func (s *SQLStore) UpdateRole(ctx context.Context, req UpdateRoleRequest) (Role, error) {
	setClauses := []string{"updated_at = ?"}
	args := []interface{}{FormatTime(time.Now())}

	if req.HasLabel {
		setClauses = append(setClauses, "label = ?")
		args = append(args, req.Label)
	}
	if req.HasCommand {
		setClauses = append(setClauses, "command = ?")
		args = append(args, req.Command)
	}
	if req.HasWorkingDir {
		setClauses = append(setClauses, "working_dir = ?")
		args = append(args, req.WorkingDir)
	}
	if req.HasIncomingPrompt {
		setClauses = append(setClauses, "incoming_prompt = ?")
		args = append(args, req.IncomingPrompt)
	}
	if req.HasSessionID {
		// An empty session id stores NULL, which is what returns the role to
		// waiting and what the partial unique index needs to allow duplicates.
		setClauses = append(setClauses, "session_id = ?")
		args = append(args, nullableSession(req.SessionID))
	}
	if req.HasSortOrder {
		setClauses = append(setClauses, "sort_order = ?")
		args = append(args, req.SortOrder)
	}
	if req.HasBackend {
		setClauses = append(setClauses, "backend = ?")
		args = append(args, req.Backend)
	}
	if req.HasTargetID {
		setClauses = append(setClauses, "target_id = ?")
		args = append(args, req.TargetID)
	}
	if req.HasGroupID {
		if req.GroupID == "" {
			return Role{}, ErrInvalidRole
		}
		setClauses = append(setClauses, "group_id = ?")
		args = append(args, req.GroupID)
	}

	args = append(args, req.ID)
	query := fmt.Sprintf(`UPDATE workspace_roles SET %s WHERE id = ? RETURNING %s`,
		strings.Join(setClauses, ", "), roleColumns)

	row := s.db.QueryRowContext(ctx, query, args...)
	r, err := scanRoleRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Role{}, ErrRoleNotFound
		}
		if isRoleConstraintError(err) {
			return Role{}, ErrInvalidRole
		}
		return Role{}, fmt.Errorf("update role: %w", err)
	}
	return r, nil
}

func (s *SQLStore) DeleteRole(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM workspace_roles WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete role %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (s *SQLStore) ReassignRoleSession(ctx context.Context, oldSessionID, newSessionID string) error {
	if oldSessionID == "" || newSessionID == "" {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin role reassignment: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is a no-op

	// Release any role already holding the replacement id first. Without this
	// the partial unique index rejects the move, and the role would be left
	// pointing at a session that no longer exists.
	if _, err := tx.ExecContext(ctx,
		`UPDATE workspace_roles SET session_id = NULL, updated_at = ? WHERE session_id = ?`,
		FormatTime(time.Now()), newSessionID); err != nil {
		return fmt.Errorf("release replacement role: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE workspace_roles SET session_id = ?, updated_at = ? WHERE session_id = ?`,
		newSessionID, FormatTime(time.Now()), oldSessionID); err != nil {
		return fmt.Errorf("move role session: %w", err)
	}
	return tx.Commit()
}

// nullableSession maps the empty session id to SQL NULL. The column is
// nullable precisely so the partial unique index can let many roles wait at
// once while still rejecting two roles for one running session.
func nullableSession(sessionID string) interface{} {
	if sessionID == "" {
		return nil
	}
	return sessionID
}

// isRoleConstraintError reports whether err is SQLite refusing a role write
// on the group foreign key or the one-session-one-role index. Both are caller
// errors; neither is an internal fault.
func isRoleConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "constraint")
}

func scanRole(rows *sql.Rows) (Role, error) {
	var r Role
	if err := rows.Scan(
		&r.ID, &r.GroupID, &r.Label, &r.Command, &r.WorkingDir, &r.IncomingPrompt,
		&r.Backend, &r.TargetID, &r.SessionID, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return Role{}, err
	}
	return r, nil
}

func scanRoleRow(row *sql.Row) (Role, error) {
	var r Role
	if err := row.Scan(
		&r.ID, &r.GroupID, &r.Label, &r.Command, &r.WorkingDir, &r.IncomingPrompt,
		&r.Backend, &r.TargetID, &r.SessionID, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return Role{}, err
	}
	return r, nil
}
