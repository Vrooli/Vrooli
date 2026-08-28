// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/internal/SECURITY-POSTURE.md#sql-injection-prevention

package handoffrules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"web-console/internal/dbx"
)

// SQLStore persists capture rules in SQLite.
type SQLStore struct {
	db dbx.Handle
}

// NewSQLStore wires a SQLite-backed store.
func NewSQLStore(db dbx.Handle) *SQLStore { return &SQLStore{db: db} }

const ruleColumns = `id, name, enabled, source, pattern, surfaces, sort_order, created_at, updated_at`

func (s *SQLStore) List(ctx context.Context) ([]Rule, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+ruleColumns+` FROM handoff_rules ORDER BY sort_order, id`)
	if err != nil {
		return nil, fmt.Errorf("query rules: %w", err)
	}
	defer rows.Close()

	out := make([]Rule, 0)
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			log.Printf("handoffrules.SQLStore.List: scan rule: %v", err)
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLStore) Upsert(ctx context.Context, req UpsertRequest) (Rule, error) {
	if err := req.Validate(); err != nil {
		return Rule{}, err
	}

	surfaces := req.Surfaces
	if surfaces == nil {
		surfaces = []string{}
	}
	surfacesJSON, err := json.Marshal(surfaces)
	if err != nil {
		return Rule{}, fmt.Errorf("marshal rule surfaces: %w", err)
	}

	id := req.ID
	if id == "" {
		id = uuid.New().String()
	}
	enabled := 0
	if req.Enabled {
		enabled = 1
	}
	now := FormatTime(time.Now())

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO handoff_rules (id, name, enabled, source, pattern, surfaces, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			name = excluded.name,
			enabled = excluded.enabled,
			source = excluded.source,
			pattern = excluded.pattern,
			surfaces = excluded.surfaces,
			sort_order = excluded.sort_order,
			updated_at = excluded.updated_at
		RETURNING `+ruleColumns,
		id, req.Name, enabled, req.Source, req.Pattern, string(surfacesJSON), req.SortOrder, now, now)

	r, err := scanRuleRow(row)
	if err != nil {
		// The source CHECK constraint is the schema's own guard. Validate
		// catches the same case first, so reaching here means the column list
		// and the validator disagreed — still a caller error, not a fault.
		if strings.Contains(strings.ToLower(err.Error()), "constraint") {
			return Rule{}, ErrInvalidRule
		}
		return Rule{}, fmt.Errorf("upsert rule: %w", err)
	}
	return r, nil
}

func (s *SQLStore) Delete(ctx context.Context, id string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM handoff_rules WHERE id = ?`, id)
	if err != nil {
		return false, fmt.Errorf("delete rule %s: %w", id, err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func scanRule(rows *sql.Rows) (Rule, error) {
	var r Rule
	var enabled int
	var surfacesJSON string
	if err := rows.Scan(&r.ID, &r.Name, &enabled, &r.Source, &r.Pattern, &surfacesJSON, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return Rule{}, err
	}
	r.Enabled = enabled != 0
	return r, decodeSurfaces(surfacesJSON, &r)
}

func scanRuleRow(row *sql.Row) (Rule, error) {
	var r Rule
	var enabled int
	var surfacesJSON string
	if err := row.Scan(&r.ID, &r.Name, &enabled, &r.Source, &r.Pattern, &surfacesJSON, &r.SortOrder, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return Rule{}, err
	}
	r.Enabled = enabled != 0
	return r, decodeSurfaces(surfacesJSON, &r)
}

// decodeSurfaces normalises a null or absent list to an empty slice so callers
// never have to distinguish "no surfaces" from "not loaded".
func decodeSurfaces(surfacesJSON string, r *Rule) error {
	if surfacesJSON == "" {
		r.Surfaces = []string{}
		return nil
	}
	if err := json.Unmarshal([]byte(surfacesJSON), &r.Surfaces); err != nil {
		return fmt.Errorf("decode rule surfaces: %w", err)
	}
	if r.Surfaces == nil {
		r.Surfaces = []string{}
	}
	return nil
}
