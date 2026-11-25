package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

type RecordSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLRecordRepository struct {
	db RecordSQLExecutor
}

func NewSQLRecordRepository(db RecordSQLExecutor) (*SQLRecordRepository, error) {
	if db == nil {
		return nil, errors.New("deployment record database is required")
	}
	return &SQLRecordRepository{db: db}, nil
}

func (r *SQLRecordRepository) Get(ctx context.Context, id string) (Record, error) {
	var payload string
	err := r.db.QueryRowContext(ctx, `SELECT payload_json FROM deployment_records WHERE id = ?`, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrRecordNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("read deployment record: %w", err)
	}
	var record Record
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return Record{}, fmt.Errorf("decode deployment record %q: %w", id, err)
	}
	return record, nil
}

func (r *SQLRecordRepository) Put(ctx context.Context, record Record) error {
	if record.ID == "" {
		return errors.New("deployment record id is required")
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode deployment record: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO deployment_records (id, payload_json, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO UPDATE SET payload_json = excluded.payload_json, updated_at = CURRENT_TIMESTAMP`, record.ID, string(payload))
	if err != nil {
		return fmt.Errorf("write deployment record: %w", err)
	}
	return nil
}
