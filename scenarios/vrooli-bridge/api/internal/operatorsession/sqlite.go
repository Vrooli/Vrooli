package operatorsession

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	shared "github.com/vrooli/api-core/operatorsession"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Store { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Enroll(ctx context.Context, record Record) (Record, error) {
	if err := Validate(record); err != nil {
		return Record{}, err
	}
	scopes, err := json.Marshal(record.Scopes)
	if err != nil {
		return Record{}, fmt.Errorf("encode operator enrollment scopes: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO operator_session_enrollments (reference, operator_id, mode, public_key, scopes_json, enrolled_at, revoked) VALUES (?, ?, ?, ?, ?, ?, ?)`, record.Reference, record.OperatorID, string(record.Mode), record.PublicKey, scopes, record.EnrolledAt.UTC().Format(time.RFC3339Nano), boolInt(record.Revoked))
	if err != nil {
		return Record{}, fmt.Errorf("store operator enrollment: %w", err)
	}
	record.PublicKey = append([]byte(nil), record.PublicKey...)
	return record, nil
}

func (r *sqliteRepository) Lookup(ctx context.Context, reference string) (Record, error) {
	var record Record
	var mode, scopes, enrolled string
	var revoked int
	err := r.db.QueryRowContext(ctx, `SELECT reference, operator_id, mode, public_key, scopes_json, enrolled_at, revoked FROM operator_session_enrollments WHERE reference = ?`, reference).Scan(&record.Reference, &record.OperatorID, &mode, &record.PublicKey, &scopes, &enrolled, &revoked)
	if err == sql.ErrNoRows {
		return Record{}, ErrNotFound{Reference: reference}
	}
	if err != nil {
		return Record{}, fmt.Errorf("lookup operator enrollment %q: %w", reference, err)
	}
	record.Mode = shared.Mode(mode)
	if err := json.Unmarshal([]byte(scopes), &record.Scopes); err != nil {
		return Record{}, fmt.Errorf("decode operator enrollment scopes: %w", err)
	}
	record.EnrolledAt, err = time.Parse(time.RFC3339Nano, enrolled)
	if err != nil {
		return Record{}, fmt.Errorf("parse operator enrollment time: %w", err)
	}
	record.Revoked = revoked != 0
	return record, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
