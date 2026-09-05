package flows

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	execution "device-control/internal/execution"
)

// SavedFlow is one immutable, device/context-scoped revision. A successful
// execution is the provenance; a descriptive name is never execution authority.
type ValidationReceipt struct {
	Disposition string   `json:"disposition"`
	StepIDs     []string `json:"step_ids"`
}
type SavedFlow struct {
	Receipt     ValidationReceipt `json:"receipt"`
	ID          string            `json:"id"`
	Version     int32             `json:"version"`
	DeviceID    string            `json:"device_id"`
	ContextKey  string            `json:"context_key"`
	SourceRunID string            `json:"source_run_id"`
	Flow        execution.Flow    `json:"flow"`
	CreatedAt   string            `json:"created_at"`
}
type Library interface {
	Save(context.Context, SavedFlow, int32) (SavedFlow, error)
	List(context.Context, string, string) ([]SavedFlow, error)
	Get(context.Context, string, int32) (SavedFlow, error)
	FindSource(context.Context, string, string, string) (SavedFlow, bool, error)
}
type SQLiteLibrary struct{ db AnchorDB }

func NewSQLiteLibrary(db AnchorDB) (*SQLiteLibrary, error) {
	if db == nil {
		return nil, fmt.Errorf("flow library requires durable storage")
	}
	_, err := db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS device_control_flow_library (
 id TEXT NOT NULL, version INTEGER NOT NULL, device_id TEXT NOT NULL, context_key TEXT NOT NULL,
 source_run_id TEXT NOT NULL, flow_json TEXT NOT NULL, receipt_json TEXT NOT NULL, created_at TEXT NOT NULL,
 PRIMARY KEY(id, version), UNIQUE(source_run_id, device_id, context_key));
 CREATE INDEX IF NOT EXISTS device_control_flow_library_lookup ON device_control_flow_library(device_id, context_key);`)
	return &SQLiteLibrary{db: db}, err
}

func (r *SQLiteLibrary) Save(ctx context.Context, f SavedFlow, expected int32) (SavedFlow, error) {
	if f.ID == "" || f.DeviceID == "" || f.ContextKey == "" || f.SourceRunID == "" || expected < 0 {
		return SavedFlow{}, fmt.Errorf("flow identity, device, context, source run and nonnegative expected version required")
	}
	receipt, err := json.Marshal(f.Receipt)
	if err != nil {
		return SavedFlow{}, err
	}
	b, err := json.Marshal(f.Flow)
	if err != nil {
		return SavedFlow{}, err
	}
	f.Version = expected + 1
	f.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	// A single conditional insert makes revision selection atomic across writers.
	result, err := r.db.ExecContext(ctx, `INSERT INTO device_control_flow_library
 (id,version,device_id,context_key,source_run_id,flow_json,receipt_json,created_at)
 SELECT ?,?,?,?,?,?,?,? WHERE COALESCE((SELECT MAX(version) FROM device_control_flow_library WHERE id=?),0)=?`,
		f.ID, f.Version, f.DeviceID, f.ContextKey, f.SourceRunID, string(b), string(receipt), f.CreatedAt, f.ID, expected)
	if err != nil {
		return SavedFlow{}, fmt.Errorf("save flow revision: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return SavedFlow{}, err
	}
	if n != 1 {
		return SavedFlow{}, fmt.Errorf("flow version conflict")
	}
	return f, nil
}

func (r *SQLiteLibrary) List(ctx context.Context, device, cohort string) ([]SavedFlow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,version,device_id,context_key,source_run_id,flow_json,receipt_json,created_at
 FROM device_control_flow_library AS f WHERE device_id=? AND context_key=?
 AND version=(SELECT MAX(version) FROM device_control_flow_library WHERE id=f.id)
 ORDER BY id LIMIT 101`, device, cohort)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedFlow{}
	for rows.Next() {
		var f SavedFlow
		var b, receipt string
		if err := rows.Scan(&f.ID, &f.Version, &f.DeviceID, &f.ContextKey, &f.SourceRunID, &b, &receipt, &f.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(b), &f.Flow); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(receipt), &f.Receipt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if len(out) > 100 {
		return nil, fmt.Errorf("flow inventory exceeds 100; narrow the context")
	}
	return out, rows.Err()
}

func (r *SQLiteLibrary) Get(ctx context.Context, id string, version int32) (SavedFlow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,version,device_id,context_key,source_run_id,flow_json,receipt_json,created_at
 FROM device_control_flow_library WHERE id=? AND (?=0 OR version=?) ORDER BY version DESC LIMIT 1`, id, version, version)
	if err != nil {
		return SavedFlow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return SavedFlow{}, err
		}
		return SavedFlow{}, fmt.Errorf("saved flow not found")
	}
	var f SavedFlow
	var b, receipt string
	if err := rows.Scan(&f.ID, &f.Version, &f.DeviceID, &f.ContextKey, &f.SourceRunID, &b, &receipt, &f.CreatedAt); err != nil {
		return SavedFlow{}, err
	}
	if err := json.Unmarshal([]byte(receipt), &f.Receipt); err != nil {
		return SavedFlow{}, err
	}
	err = json.Unmarshal([]byte(b), &f.Flow)
	return f, err
}

// FindSource makes repeated promotion of the same verified run idempotent.
func (r *SQLiteLibrary) FindSource(ctx context.Context, run, device, cohort string) (SavedFlow, bool, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id,version FROM device_control_flow_library WHERE source_run_id=? AND device_id=? AND context_key=?", run, device, cohort)
	if err != nil {
		return SavedFlow{}, false, err
	}
	var id string
	var version int32
	if !rows.Next() {
		err = rows.Err()
		rows.Close()
		return SavedFlow{}, false, err
	}
	err = rows.Scan(&id, &version)
	rows.Close()
	if err != nil {
		return SavedFlow{}, false, err
	}
	f, err := r.Get(ctx, id, version)
	return f, err == nil, err
}
