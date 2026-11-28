package sceneprep

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"

	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqlTransactor interface {
	SQLExecutor
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type SQLRepository struct{ db SQLExecutor }

func NewSQLRepository(db SQLExecutor) *SQLRepository { return &SQLRepository{db: db} }
func (r *SQLRepository) Put(ctx context.Context, p *preparationv1.StoredPhotograph) error {
	raw, err := protojson.Marshal(p)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO prepared_photograph (id,json,created_at_unix_ms) VALUES (?,?,?) ON CONFLICT(id) DO NOTHING`, p.Id, string(raw), p.CreatedAtUnixMs)
	return err
}
func (r *SQLRepository) Get(ctx context.Context, id string) (*preparationv1.StoredPhotograph, error) {
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT json FROM prepared_photograph WHERE id=?`, id).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return decodeRecord(raw)
}
func (r *SQLRepository) List(ctx context.Context, limit int) ([]*preparationv1.StoredPhotograph, error) {
	// Row insertion order is the owner's durable admission order. Timestamps
	// stay truthful wall-clock observations and may legitimately tie.
	rows, err := r.db.QueryContext(ctx, `SELECT json FROM prepared_photograph ORDER BY rowid DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*preparationv1.StoredPhotograph{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		p, err := decodeRecord(raw)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *SQLRepository) ReviewOperation(ctx context.Context, operationID string) (string, *preparationv1.StoredPhotograph, error) {
	var fingerprint, id string
	err := r.db.QueryRowContext(ctx, `SELECT request_sha256,photograph_id FROM photograph_review_operation WHERE operation_id=?`, operationID).Scan(&fingerprint, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	p, err := r.Get(ctx, id)
	return fingerprint, p, err
}

// PutReviewOperation atomically claims an operation key and stores its immutable
// child. Concurrent retries with the same fingerprint resolve to the winner.
func (r *SQLRepository) PutReviewOperation(ctx context.Context, operationID, fingerprint string, p *preparationv1.StoredPhotograph) (*preparationv1.StoredPhotograph, error) {
	db, ok := r.db.(sqlTransactor)
	if !ok {
		return nil, errors.New("review repository does not support atomic operation commits")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO photograph_review_operation(operation_id,request_sha256,photograph_id) VALUES(?,?,?) ON CONFLICT(operation_id) DO NOTHING`, operationID, fingerprint, p.Id)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		var existingFingerprint, existingID string
		if err := tx.QueryRowContext(ctx, `SELECT request_sha256,photograph_id FROM photograph_review_operation WHERE operation_id=?`, operationID).Scan(&existingFingerprint, &existingID); err != nil {
			return nil, err
		}
		if existingFingerprint != fingerprint {
			return nil, errors.New("operation_id was already used for a different review request")
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return r.Get(ctx, existingID)
	}
	raw, err := protojson.Marshal(p)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO prepared_photograph (id,json,created_at_unix_ms) VALUES (?,?,?) ON CONFLICT(id) DO NOTHING`, p.Id, string(raw), p.CreatedAtUnixMs); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, p.Id)
}
func decodeRecord(raw string) (*preparationv1.StoredPhotograph, error) {
	p := &preparationv1.StoredPhotograph{}
	if err := protojson.Unmarshal([]byte(raw), p); err != nil {
		return nil, err
	}
	return p, nil
}
