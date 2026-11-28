package modelstorage

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"image-tools/internal/catalogstate"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the model-storage SQL contribution (keep pins and offload
// receipts), applied through the modules.AllSchemas registry.
func Schema() string { return schemaSQL }

// Receipt outcomes.
const (
	OutcomeRemoved = "removed"
	OutcomeSkipped = "skipped"
	OutcomeFailed  = "failed"
)

// KeepRecord is one operator keep pin.
type KeepRecord struct {
	ModelID   string    `json:"model_id"`
	Reason    string    `json:"reason,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Receipt is the durable record of one offload attempt.
type Receipt struct {
	ID                   string    `json:"id"`
	ModelID              string    `json:"model_id"`
	Trigger              string    `json:"trigger"`
	Reason               string    `json:"reason"`
	Outcome              string    `json:"outcome"`
	Error                string    `json:"error,omitempty"`
	Bytes                int64     `json:"bytes"`
	Checksum             string    `json:"checksum,omitempty"`
	LastUseAt            time.Time `json:"last_use_at,omitempty"`
	LastUseBasis         string    `json:"last_use_basis"`
	IdleSeconds          int64     `json:"idle_seconds"`
	IdleThresholdSeconds int64     `json:"idle_threshold_seconds"`
	Mode                 string    `json:"mode"`
	FreeBeforeBytes      int64     `json:"free_before_bytes"`
	FreeAfterBytes       int64     `json:"free_after_bytes"`
	CreatedAt            time.Time `json:"created_at"`
}

// Store persists keep pins and offload receipts.
type Store struct{ db catalogstate.SQLExecutor }

// NewStore constructs the store over db.
func NewStore(db catalogstate.SQLExecutor) *Store { return &Store{db: db} }

// SetKeep pins (keep=true) or unpins a model.
func (s *Store) SetKeep(ctx context.Context, id string, keep bool, reason string, at time.Time) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("modelstorage: model id is required")
	}
	if !keep {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM model_keep WHERE model_id=?`, id); err != nil {
			return fmt.Errorf("modelstorage: unkeep %q: %w", id, err)
		}
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO model_keep(model_id,reason,created_at) VALUES(?,?,?)
ON CONFLICT(model_id) DO UPDATE SET reason=excluded.reason`, id, strings.TrimSpace(reason), at.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("modelstorage: keep %q: %w", id, err)
	}
	return nil
}

// Kept returns every keep pin keyed by model id.
func (s *Store) Kept(ctx context.Context) (map[string]KeepRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model_id,reason,created_at FROM model_keep`)
	if err != nil {
		return nil, fmt.Errorf("modelstorage: list keep pins: %w", err)
	}
	defer rows.Close()
	out := map[string]KeepRecord{}
	for rows.Next() {
		var r KeepRecord
		var at string
		if err := rows.Scan(&r.ModelID, &r.Reason, &at); err != nil {
			return nil, fmt.Errorf("modelstorage: scan keep pin: %w", err)
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, at)
		out[r.ModelID] = r
	}
	return out, rows.Err()
}

// RecordReceipt persists one offload attempt.
func (s *Store) RecordReceipt(ctx context.Context, r Receipt) error {
	lastUse := ""
	if !r.LastUseAt.IsZero() {
		lastUse = r.LastUseAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO model_offload_receipt(id,model_id,trigger,reason,outcome,error,bytes,checksum,
last_use_at,last_use_basis,idle_seconds,idle_threshold_seconds,mode,free_before_bytes,free_after_bytes,created_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.ModelID, r.Trigger, r.Reason, r.Outcome, r.Error, r.Bytes, r.Checksum,
		lastUse, r.LastUseBasis, r.IdleSeconds, r.IdleThresholdSeconds, r.Mode, r.FreeBeforeBytes, r.FreeAfterBytes,
		r.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("modelstorage: record offload receipt for %q: %w", r.ModelID, err)
	}
	return nil
}

// Receipts returns the newest receipts first.
func (s *Store) Receipts(ctx context.Context, limit int) ([]Receipt, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,model_id,trigger,reason,outcome,error,bytes,checksum,last_use_at,last_use_basis,
idle_seconds,idle_threshold_seconds,mode,free_before_bytes,free_after_bytes,created_at
FROM model_offload_receipt ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("modelstorage: list offload receipts: %w", err)
	}
	defer rows.Close()
	var out []Receipt
	for rows.Next() {
		var r Receipt
		var lastUse, created string
		if err := rows.Scan(&r.ID, &r.ModelID, &r.Trigger, &r.Reason, &r.Outcome, &r.Error, &r.Bytes, &r.Checksum, &lastUse,
			&r.LastUseBasis, &r.IdleSeconds, &r.IdleThresholdSeconds, &r.Mode, &r.FreeBeforeBytes, &r.FreeAfterBytes, &created); err != nil {
			return nil, fmt.Errorf("modelstorage: scan offload receipt: %w", err)
		}
		r.LastUseAt, _ = time.Parse(time.RFC3339Nano, lastUse)
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, r)
	}
	return out, rows.Err()
}
