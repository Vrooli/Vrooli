package development

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/vrooli/api-core/database"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

var (
	ErrNotFound = errors.New("development engagement not found")
	ErrConflict = errors.New("development engagement changed; reload before deciding")
	ErrInvalid  = errors.New("invalid development operation")
	ErrDenied   = errors.New("development operation is not authorized")
)

// Repository commits an immutable snapshot and its aggregate atomically. Version
// zero creates; other versions are compare-and-swap, including across processes.
// There is deliberately no operation to overwrite or delete a retained snapshot.
type Repository interface {
	Get(context.Context, string) (Engagement, error)
	Snapshot(context.Context, string) (Snapshot, error)
	Commit(context.Context, int64, Engagement, *Snapshot) error
}

type SQLiteRepository struct{ db *database.RoutedDB }

func NewSQLiteRepository(db *database.RoutedDB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Get(ctx context.Context, item string) (Engagement, error) {
	var value Engagement
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM development_engagements WHERE work_item = ?`, item).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	if err != nil {
		return value, err
	}
	err = json.Unmarshal(body, &value)
	return value, err
}

func (r *SQLiteRepository) Snapshot(ctx context.Context, digest string) (Snapshot, error) {
	var value Snapshot
	var body []byte
	err := r.db.QueryRowContext(ctx, `SELECT body FROM development_snapshots WHERE digest = ?`, digest).Scan(&body)
	if errors.Is(err, sql.ErrNoRows) {
		return value, ErrNotFound
	}
	if err != nil {
		return value, err
	}
	err = json.Unmarshal(body, &value)
	return value, err
}

func (r *SQLiteRepository) Commit(ctx context.Context, expected int64, value Engagement, snapshot *Snapshot) error {
	if expected < 0 || value.Version != expected+1 {
		return ErrConflict
	}
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if snapshot != nil {
		encoded, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO development_snapshots(digest, body) VALUES (?, ?) ON CONFLICT(digest) DO NOTHING`, snapshot.Digest, encoded); err != nil {
			return err
		}
		var retained []byte
		if err = tx.QueryRowContext(ctx, `SELECT body FROM development_snapshots WHERE digest = ?`, snapshot.Digest).Scan(&retained); err != nil {
			return err
		}
		if string(retained) != string(encoded) {
			return fmt.Errorf("snapshot digest collision: %w", ErrConflict)
		}
	}
	var result sql.Result
	if expected == 0 {
		result, err = tx.ExecContext(ctx, `INSERT INTO development_engagements(work_item, version, body) VALUES (?, ?, ?) ON CONFLICT(work_item) DO NOTHING`, value.WorkItem, value.Version, body)
	} else {
		result, err = tx.ExecContext(ctx, `UPDATE development_engagements SET version = ?, body = ? WHERE work_item = ? AND version = ?`, value.Version, body, value.WorkItem, expected)
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrConflict
	}
	return tx.Commit()
}
