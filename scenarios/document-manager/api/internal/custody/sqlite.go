package custody

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type SQLiteRepository struct{ db queryer }

func NewSQLiteRepository(db queryer) *SQLiteRepository { return &SQLiteRepository{db: db} }

func (r *SQLiteRepository) Append(ctx context.Context, record Record) error {
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO custody_records (document_hash, step, tier, provider, locality, profile, privacy_class, state, reason, remedy, started_at, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.DocumentHash, record.Step, record.Tier, record.Provider, record.Locality, record.Profile, record.PrivacyClass, record.State, record.Reason, record.Remedy, record.StartedAt.Format(time.RFC3339Nano), record.Duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("append custody record: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) List(ctx context.Context, documentHash string) ([]Record, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, document_hash, step, tier, provider, locality, profile, privacy_class, state, reason, remedy, started_at, duration_ms FROM custody_records WHERE document_hash = ? ORDER BY id`, documentHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []Record
	for rows.Next() {
		var record Record
		var started string
		var durationMs int64
		if err := rows.Scan(&record.ID, &record.DocumentHash, &record.Step, &record.Tier, &record.Provider, &record.Locality, &record.Profile, &record.PrivacyClass, &record.State, &record.Reason, &record.Remedy, &started, &durationMs); err != nil {
			return nil, err
		}
		record.StartedAt, err = time.Parse(time.RFC3339Nano, started)
		if err != nil {
			return nil, err
		}
		record.Duration = time.Duration(durationMs) * time.Millisecond
		records = append(records, record)
	}
	return records, rows.Err()
}

func BuildReceipt(ctx context.Context, repo Repository, documentHash string) (Receipt, error) {
	records, err := repo.List(ctx, documentHash)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{DocumentHash: documentHash, SelfAttested: true, Records: records}, nil
}
