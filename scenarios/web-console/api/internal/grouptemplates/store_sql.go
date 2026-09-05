// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package grouptemplates

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"web-console/internal/dbx"
)

// SQLStore persists group templates in SQLite.
//
// The role list is a JSON column, mirroring shortcut_profiles.shortcuts, so
// this scenario has one storage idiom for a configuration row that owns an
// ordered child list. The tradeoff is that a role is not independently
// queryable; nothing needs that today.
type SQLStore struct {
	db dbx.Handle
}

// NewSQLStore wires a SQLite-backed store.
func NewSQLStore(db dbx.Handle) *SQLStore { return &SQLStore{db: db} }

const templateColumns = `id, name, color, roles, use_count, created_at, updated_at`

func (s *SQLStore) List(ctx context.Context) ([]Template, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+templateColumns+` FROM group_templates ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("query templates: %w", err)
	}
	defer rows.Close()

	out := make([]Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			log.Printf("grouptemplates.SQLStore.List: scan template: %v", err)
			continue
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *SQLStore) Upsert(ctx context.Context, req UpsertRequest) (Template, error) {
	if err := req.Validate(); err != nil {
		return Template{}, err
	}

	roles := req.Roles
	if roles == nil {
		roles = []TemplateRole{}
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return Template{}, fmt.Errorf("marshal template roles: %w", err)
	}

	id := req.ID
	if id == "" {
		id = uuid.New().String()
	}
	now := FormatTime(time.Now())

	// use_count is preserved unless the caller explicitly sets it, so editing
	// a template's content never resets how often it has been used.
	useCountExpr := "group_templates.use_count"
	if req.HasUseCount {
		useCountExpr = "excluded.use_count"
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO group_templates (id, name, color, roles, use_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name,
			color = excluded.color,
			roles = excluded.roles,
			use_count = `+useCountExpr+`,
			updated_at = excluded.updated_at
		RETURNING `+templateColumns,
		id, req.Name, req.Color, string(rolesJSON), req.UseCount, now, now)

	t, err := scanTemplateRow(row)
	if err != nil {
		return Template{}, fmt.Errorf("upsert template: %w", err)
	}
	return t, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM group_templates WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete template %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func scanTemplate(rows *sql.Rows) (Template, error) {
	var t Template
	var rolesJSON string
	if err := rows.Scan(&t.ID, &t.Name, &t.Color, &rolesJSON, &t.UseCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return Template{}, err
	}
	return t, decodeRoles(rolesJSON, &t)
}

func scanTemplateRow(row *sql.Row) (Template, error) {
	var t Template
	var rolesJSON string
	if err := row.Scan(&t.ID, &t.Name, &t.Color, &rolesJSON, &t.UseCount, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return Template{}, err
	}
	return t, decodeRoles(rolesJSON, &t)
}

// decodeRoles fills t.Roles, normalising a null or absent list to an empty
// slice so callers never have to distinguish "no roles" from "not loaded".
func decodeRoles(rolesJSON string, t *Template) error {
	if rolesJSON == "" {
		t.Roles = []TemplateRole{}
		return nil
	}
	if err := json.Unmarshal([]byte(rolesJSON), &t.Roles); err != nil {
		return fmt.Errorf("decode template roles: %w", err)
	}
	if t.Roles == nil {
		t.Roles = []TemplateRole{}
	}
	return nil
}
