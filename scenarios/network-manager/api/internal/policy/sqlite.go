package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) SaveChange(ctx context.Context, change Change) (Change, error) {
	if change.CreatedAt.IsZero() {
		change.CreatedAt = time.Now().UTC()
	}
	if change.UpdatedAt.IsZero() {
		change.UpdatedAt = change.CreatedAt
	}
	valuesJSON, err := encodeStrings(change.Values)
	if err != nil {
		return Change{}, err
	}
	effectsJSON, err := encodeStrings(change.Effects)
	if err != nil {
		return Change{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO policy_change_plans (
  id, target, action, status, values_json, effects_json, rollback_supported,
  rollback_handle, approval_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, change.ID, change.Target, change.Action, change.Status, valuesJSON, effectsJSON, boolInt(change.RollbackSupported), change.RollbackHandle, change.ApprovalID, formatTime(change.CreatedAt), formatTime(change.UpdatedAt)); err != nil {
		return Change{}, fmt.Errorf("save policy change %q: %w", change.ID, err)
	}
	return change, nil
}

func (r *sqliteRepository) GetChange(ctx context.Context, id string) (Change, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, target, action, status, values_json, effects_json, rollback_supported,
       rollback_handle, approval_id, created_at, updated_at
FROM policy_change_plans
WHERE id = ?
`, id)
	change, err := scanChange(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Change{}, ErrNotFound
	}
	if err != nil {
		return Change{}, err
	}
	return change, nil
}

func (r *sqliteRepository) UpdateChange(ctx context.Context, change Change) (Change, error) {
	if change.UpdatedAt.IsZero() {
		change.UpdatedAt = time.Now().UTC()
	}
	valuesJSON, err := encodeStrings(change.Values)
	if err != nil {
		return Change{}, err
	}
	effectsJSON, err := encodeStrings(change.Effects)
	if err != nil {
		return Change{}, err
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE policy_change_plans
SET target = ?, action = ?, status = ?, values_json = ?, effects_json = ?,
    rollback_supported = ?, rollback_handle = ?, approval_id = ?, updated_at = ?
WHERE id = ?
`, change.Target, change.Action, change.Status, valuesJSON, effectsJSON, boolInt(change.RollbackSupported), change.RollbackHandle, change.ApprovalID, formatTime(change.UpdatedAt), change.ID)
	if err != nil {
		return Change{}, fmt.Errorf("update policy change %q: %w", change.ID, err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return Change{}, ErrNotFound
	}
	return change, nil
}

func (r *sqliteRepository) SaveApproval(ctx context.Context, approval ApprovalRecord) (ApprovalRecord, error) {
	if approval.CreatedAt.IsZero() {
		approval.CreatedAt = time.Now().UTC()
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO approval_records (id, change_id, approved, note, created_at)
VALUES (?, ?, ?, ?, ?)
`, approval.ID, approval.ChangeID, boolInt(approval.Approved), approval.Note, formatTime(approval.CreatedAt)); err != nil {
		return ApprovalRecord{}, fmt.Errorf("save approval %q: %w", approval.ID, err)
	}
	return approval, nil
}

func (r *sqliteRepository) SaveRollback(ctx context.Context, rollback RollbackRecord) (RollbackRecord, error) {
	if rollback.CreatedAt.IsZero() {
		rollback.CreatedAt = time.Now().UTC()
	}
	detailsJSON, err := encodeStrings(rollback.Details)
	if err != nil {
		return RollbackRecord{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO rollback_records (id, change_id, status, details_json, created_at)
VALUES (?, ?, ?, ?, ?)
`, rollback.ID, rollback.ChangeID, rollback.Status, detailsJSON, formatTime(rollback.CreatedAt)); err != nil {
		return RollbackRecord{}, fmt.Errorf("save rollback %q: %w", rollback.ID, err)
	}
	return rollback, nil
}

type changeScanner interface {
	Scan(dest ...any) error
}

func scanChange(row changeScanner) (Change, error) {
	var change Change
	var valuesJSON, effectsJSON, createdAt, updatedAt string
	var rollbackSupported int
	if err := row.Scan(&change.ID, &change.Target, &change.Action, &change.Status, &valuesJSON, &effectsJSON, &rollbackSupported, &change.RollbackHandle, &change.ApprovalID, &createdAt, &updatedAt); err != nil {
		return Change{}, err
	}
	var err error
	change.Values, err = decodeStrings(valuesJSON)
	if err != nil {
		return Change{}, err
	}
	change.Effects, err = decodeStrings(effectsJSON)
	if err != nil {
		return Change{}, err
	}
	change.RollbackSupported = rollbackSupported == 1
	change.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Change{}, fmt.Errorf("parse policy change created_at: %w", err)
	}
	change.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return Change{}, fmt.Errorf("parse policy change updated_at: %w", err)
	}
	return change, nil
}

func encodeStrings(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	b, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode string list: %w", err)
	}
	return string(b), nil
}

func decodeStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode string list: %w", err)
	}
	return values, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatTime(value time.Time) string {
	return value.UTC().Format(TimeFormat)
}
