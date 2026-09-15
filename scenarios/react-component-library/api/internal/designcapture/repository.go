package designcapture

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
func (r *SQLiteRepository) Create(ctx context.Context, key string, request Request) (Operation, error) {
	if len(key) < 1 || len(key) > 200 {
		return Operation{}, fmt.Errorf("idempotency key must contain 1 to 200 bytes")
	}
	hash, raw, err := requestIdentity(request)
	if err != nil {
		return Operation{}, err
	}
	digest := sha256.Sum256([]byte(key))
	id := "capture_" + hex.EncodeToString(digest[:])
	_, err = r.db.ExecContext(ctx, `INSERT INTO design_capture_operations(id,request_hash,request_json,state) VALUES(?,?,?,?) ON CONFLICT(id) DO NOTHING`, id, hash, string(raw), Prepared)
	if err != nil {
		return Operation{}, err
	}
	op, err := r.Get(ctx, id)
	if err != nil {
		return op, err
	}
	if op.RequestHash != hash {
		return Operation{}, ErrConflict
	}
	return op, nil
}
func (r *SQLiteRepository) Get(ctx context.Context, id string) (Operation, error) {
	var op Operation
	var raw, artifacts string
	err := r.db.QueryRowContext(ctx, `SELECT id,request_hash,request_json,state,producer_id,artifacts_json,detail,version FROM design_capture_operations WHERE id=?`, id).Scan(&op.ID, &op.RequestHash, &raw, &op.State, &op.ProducerID, &artifacts, &op.Detail, &op.Version)
	if err != nil {
		return op, err
	}
	if err = json.Unmarshal([]byte(raw), &op.Request); err != nil {
		return Operation{}, err
	}
	hash, _, err := requestIdentity(op.Request)
	if err != nil {
		return Operation{}, err
	}
	if hash != op.RequestHash {
		return Operation{}, fmt.Errorf("persisted capture request identity mismatch")
	}
	if err = json.Unmarshal([]byte(artifacts), &op.Artifacts); err != nil {
		return Operation{}, err
	}
	if err := validateOperation(op); err != nil {
		return Operation{}, err
	}
	return op, nil
}
func (r *SQLiteRepository) Transition(ctx context.Context, id string, version int64, next State, producer string, artifacts []Artifact, detail string) (Operation, error) {
	op, err := r.Get(ctx, id)
	if err != nil {
		return op, err
	}
	if op.Version != version || !allowed(op.State, next) {
		return Operation{}, ErrConflict
	}
	if op.ProducerID != "" && producer != op.ProducerID {
		return Operation{}, fmt.Errorf("producer identity cannot change")
	}
	nextOperation := op
	nextOperation.State, nextOperation.ProducerID, nextOperation.Artifacts, nextOperation.Detail = next, producer, artifacts, detail
	if err := validateOperation(nextOperation); err != nil {
		return Operation{}, err
	}
	raw, err := json.Marshal(artifacts)
	if err != nil {
		return Operation{}, err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE design_capture_operations SET state=?,producer_id=?,artifacts_json=?,detail=?,version=version+1 WHERE id=? AND version=?`, next, producer, string(raw), detail, id, version)
	if err != nil {
		return Operation{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Operation{}, err
	}
	if changed != 1 {
		return Operation{}, ErrConflict
	}
	return r.Get(ctx, id)
}
