package flows

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"device-control/internal/execution"
	"github.com/vrooli/api-core/database"
)

// RetainedRun is explicitly selected by the owner. Flow definitions can contain
// private targets and arguments; they must not be retained automatically.
type RetainedRun struct {
	DeviceID  string              `json:"device_id"`
	ExpiresAt *time.Time          `json:"expires_at"`
	Flow      execution.Flow      `json:"flow"`
	Result    execution.RunResult `json:"result"`
}

type RetainedRunSummary struct {
	RunID     string     `json:"run_id"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type RetainedRuns interface {
	Put(context.Context, RetainedRun) error
	Get(context.Context, string) (RetainedRun, error)
	List(context.Context) ([]RetainedRunSummary, error)
	Delete(context.Context, string) error
}

//go:embed retained_runs.sql
var retainedRunsSchema string

type SQLiteRetainedRuns struct {
	db  AnchorDB
	now func() time.Time
}

var ErrClipboardRetention = errors.New("clipboard flows cannot be retained")

func NewSQLiteRetainedRuns(ctx context.Context, db AnchorDB) (*SQLiteRetainedRuns, error) {
	if db == nil {
		return nil, ErrInvalidRequest
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(func() string { return retainedRunsSchema })); err != nil {
		return nil, err
	}
	return &SQLiteRetainedRuns{db: db, now: time.Now}, nil
}

func (r *SQLiteRetainedRuns) Put(ctx context.Context, run RetainedRun) error {
	if run.Result.RunID == "" || run.DeviceID == "" || run.ExpiresAt == nil || !run.ExpiresAt.After(r.now()) {
		return ErrInvalidRequest
	}
	for _, step := range run.Flow.Steps {
		kind := strings.ToLower(strings.TrimSpace(step.Kind))
		if kind == "clipboard-read" || kind == "clipboard-write" {
			return ErrClipboardRetention
		}
	}
	// Clear an expired existing record on access before a fresh explicit capture.
	if _, err := r.read(ctx, run.Result.RunID); err != nil && !errors.Is(err, ErrInvalidRequest) {
		return err
	}
	data, err := json.Marshal(run)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO device_control_retained_runs(run_id,device_id,payload) VALUES(?,?,?) ON CONFLICT(run_id) DO NOTHING`, run.Result.RunID, run.DeviceID, string(data))
	if err != nil {
		return err
	}
	stored, err := r.read(ctx, run.Result.RunID)
	if err != nil {
		return err
	}
	if stored != string(data) {
		return errors.New("retained run evidence conflict")
	}
	return nil
}

func (r *SQLiteRetainedRuns) Get(ctx context.Context, id string) (RetainedRun, error) {
	raw, err := r.read(ctx, id)
	if err != nil {
		return RetainedRun{}, err
	}
	var run RetainedRun
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		return RetainedRun{}, fmt.Errorf("decode retained run: %w", err)
	}
	if run.Result.RunID != id {
		return RetainedRun{}, errors.New("retained run identity mismatch")
	}
	return run, nil
}

func (r *SQLiteRetainedRuns) read(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", ErrInvalidRequest
	}
	rows, err := r.db.QueryContext(ctx, `SELECT payload FROM device_control_retained_runs WHERE run_id=?`, id)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", ErrInvalidRequest
	}
	var raw string
	if err := rows.Scan(&raw); err != nil {
		return "", err
	}
	if err := rows.Close(); err != nil {
		return "", err
	}
	var run RetainedRun
	if err := json.Unmarshal([]byte(raw), &run); err != nil {
		return "", fmt.Errorf("decode retained run: %w", err)
	}
	if run.ExpiresAt != nil && !r.now().Before(*run.ExpiresAt) {
		// Conditional deletion cannot erase a concurrently changed record.
		if _, err := r.db.ExecContext(ctx, `DELETE FROM device_control_retained_runs WHERE run_id=? AND payload=?`, id, raw); err != nil {
			return "", err
		}
		return "", ErrInvalidRequest
	}
	return raw, nil
}

func (r *SQLiteRetainedRuns) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidRequest
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM device_control_retained_runs WHERE run_id=?`, id)
	return err
}

func (r *SQLiteRetainedRuns) List(ctx context.Context) ([]RetainedRunSummary, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT run_id,payload FROM device_control_retained_runs ORDER BY run_id`)
	if err != nil {
		return nil, err
	}
	type entry struct{ id, raw string }
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.raw); err != nil {
			rows.Close()
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	out := []RetainedRunSummary{}
	for _, e := range entries {
		var run RetainedRun
		if err := json.Unmarshal([]byte(e.raw), &run); err != nil {
			return nil, fmt.Errorf("decode retained run: %w", err)
		}
		if run.Result.RunID != e.id {
			return nil, errors.New("retained run identity mismatch")
		}
		if run.ExpiresAt != nil && !r.now().Before(*run.ExpiresAt) {
			if _, err := r.db.ExecContext(ctx, `DELETE FROM device_control_retained_runs WHERE run_id=? AND payload=?`, e.id, e.raw); err != nil {
				return nil, err
			}
			continue
		}
		out = append(out, RetainedRunSummary{RunID: e.id, ExpiresAt: run.ExpiresAt})
	}
	return out, nil
}
