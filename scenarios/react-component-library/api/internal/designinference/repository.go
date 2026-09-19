package designinference

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

type SQLiteRepository struct{ db *sql.DB }

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository { return &SQLiteRepository{db: db} }
func OperationID(key string) (string, error) {
	if len(key) < 1 || len(key) > 200 {
		return "", fmt.Errorf("idempotency key must contain 1 to 200 bytes")
	}
	digest := sha256.Sum256([]byte(key))
	return "inference_" + hex.EncodeToString(digest[:]), nil
}
func (r *SQLiteRepository) Create(ctx context.Context, key string, request Request) (Operation, error) {
	id, err := OperationID(key)
	if err != nil {
		return Operation{}, err
	}
	hash, raw, err := identity(request)
	if err != nil {
		return Operation{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO design_inference_operations(id,request_hash,request_json,state) VALUES(?,?,?,'prepared') ON CONFLICT(id) DO NOTHING`, id, hash, string(raw))
	if err != nil {
		return Operation{}, err
	}
	op, err := r.Get(ctx, id)
	if err == nil && op.RequestHash != hash {
		return Operation{}, ErrConflict
	}
	return op, err
}
func (r *SQLiteRepository) Get(ctx context.Context, id string) (Operation, error) {
	var op Operation
	var raw string
	err := r.db.QueryRowContext(ctx, `SELECT id,request_hash,request_json,state,response_json,detail FROM design_inference_operations WHERE id=?`, id).Scan(&op.ID, &op.RequestHash, &raw, &op.State, &op.ResponseJSON, &op.Detail)
	if err != nil {
		return op, err
	}
	if err := json.Unmarshal([]byte(raw), &op.Request); err != nil {
		return Operation{}, err
	}
	hash, _, err := identity(op.Request)
	if err != nil || hash != op.RequestHash {
		return Operation{}, fmt.Errorf("persisted inference input identity mismatch")
	}
	if (op.State == "completed" || op.State == "failed") && !json.Valid([]byte(op.ResponseJSON)) {
		return Operation{}, fmt.Errorf("terminal inference response is missing or invalid")
	}
	return op, nil
}

// Claim records uncertainty before any external call. A crash at any later
// instruction must never cause automatic redispatch, even if no call was sent.
func (r *SQLiteRepository) Claim(ctx context.Context, id string) (bool, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE design_inference_operations SET state='dispatch_unknown',detail='Dispatch reserved; completion is not yet known.' WHERE id=? AND state='prepared'`, id)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}
func (r *SQLiteRepository) Finish(ctx context.Context, id, state, response, detail string) (Operation, error) {
	if (state != "completed" && state != "failed") || len(response) > 2*1024*1024 || !json.Valid([]byte(response)) || len(detail) > 4000 {
		return Operation{}, fmt.Errorf("invalid inference result")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE design_inference_operations SET state=?,response_json=?,detail=? WHERE id=? AND state='dispatch_unknown'`, state, response, detail, id)
	if err != nil {
		return Operation{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return Operation{}, err
	}
	if n != 1 {
		return Operation{}, ErrConflict
	}
	return r.Get(ctx, id)
}
