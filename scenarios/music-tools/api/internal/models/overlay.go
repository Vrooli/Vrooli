package models

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Overlay is the durable state layer over the committed seed registry. Empty
// overlay tables intentionally mean that seed defaults remain authoritative.
type Overlay struct {
	db   SQLExecutor
	base *Registry
}

func NewOverlay(db SQLExecutor, base *Registry) *Overlay { return &Overlay{db: db, base: base} }

func (o *Overlay) SetInstalled(ctx context.Context, modelID, checksum string, bytes uint64) error {
	if modelID == "" || checksum == "" {
		return fmt.Errorf("models: model id and checksum are required")
	}
	_, err := o.db.ExecContext(ctx, `INSERT INTO model_install(model_id, installed_at, checksum, bytes) VALUES (?, ?, ?, ?) ON CONFLICT(model_id) DO UPDATE SET installed_at=excluded.installed_at, checksum=excluded.checksum, bytes=excluded.bytes`, modelID, time.Now().UTC().Format(time.RFC3339Nano), checksum, bytes)
	return err
}

func (o *Overlay) SetEnabled(ctx context.Context, modelID string, enabled bool) error {
	value := 0
	if enabled {
		value = 1
	}
	_, err := o.db.ExecContext(ctx, `INSERT INTO model_state(model_id, enabled, updated_at) VALUES (?, ?, ?) ON CONFLICT(model_id) DO UPDATE SET enabled=excluded.enabled, updated_at=excluded.updated_at`, modelID, value, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (o *Overlay) SetCustom(ctx context.Context, model Model) error {
	if model.ID == "" {
		return fmt.Errorf("models: custom model id is required")
	}
	b, err := json.Marshal(model)
	if err != nil {
		return err
	}
	_, err = o.db.ExecContext(ctx, `INSERT INTO model_custom(model_id, model_json, updated_at) VALUES (?, ?, ?) ON CONFLICT(model_id) DO UPDATE SET model_json=excluded.model_json, updated_at=excluded.updated_at`, model.ID, string(b), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (o *Overlay) SetOperationDefault(ctx context.Context, operation, modelID string) error {
	if operation == "" || modelID == "" {
		return fmt.Errorf("models: operation and model id are required")
	}
	_, err := o.db.ExecContext(ctx, `INSERT INTO model_operation_default(operation, model_id, updated_at) VALUES (?, ?, ?) ON CONFLICT(operation) DO UPDATE SET model_id=excluded.model_id, updated_at=excluded.updated_at`, operation, modelID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (o *Overlay) ResolveOperation(ctx context.Context, operation string) (Model, error) {
	var modelID string
	if err := o.db.QueryRowContext(ctx, `SELECT model_id FROM model_operation_default WHERE operation=?`, operation).Scan(&modelID); err != nil {
		return Model{}, fmt.Errorf("models: no default for operation %q: %w", operation, err)
	}
	return o.base.Resolve(modelID)
}
