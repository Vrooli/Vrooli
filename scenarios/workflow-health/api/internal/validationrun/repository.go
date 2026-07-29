// Package validationrun owns Workflow Health's durable validation-run ledger.
package validationrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	core "github.com/vrooli/api-core/validationrun"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type DB interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Record struct {
	Run         core.Run
	ETA         time.Duration
	Preliminary *scenariovalidationv1.ValidateScenarioResponse
	Terminal    *scenariovalidationv1.ValidateScenarioResponse
	ErrorCode   string
	Error       string
	Artifacts   []string
}

type Repository struct{ DB DB }

// ListInterrupted returns provider work that was durable but not terminal at
// process shutdown. Startup policy turns these into explicit recovery failures;
// it never blindly reruns BAS work and risks duplicate side effects.
func (r Repository) ListInterrupted(ctx context.Context) ([]Record, error) {
	rows, err := r.DB.QueryContext(ctx, selectRun+" WHERE state IN (?, ?)", string(core.StateQueued), string(core.StateRunning))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		record, err := r.scanRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (r Repository) FindByIdempotency(ctx context.Context, key string) (Record, error) {
	return r.scan(r.DB.QueryRowContext(ctx, selectRun+" WHERE idempotency_key = ?", key))
}

func (r Repository) Get(ctx context.Context, id string) (Record, error) {
	return r.scan(r.DB.QueryRowContext(ctx, selectRun+" WHERE id = ?", id))
}

func (r Repository) Create(ctx context.Context, record Record) error {
	if err := record.Run.Validate(); err != nil {
		return err
	}
	preliminary, err := marshal(record.Preliminary)
	if err != nil {
		return fmt.Errorf("marshal preliminary result: %w", err)
	}
	_, err = r.DB.ExecContext(ctx, `INSERT INTO validation_runs (id, scenario, target_path, idempotency_key, parent_run_id, state, created_at, eta_seconds, preliminary_result, artifact_refs, cancellation_requested, version) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.Run.ID, record.Run.Target.Scenario, record.Run.Target.Path, record.Run.IdempotencyKey, record.Run.ParentRunID, string(record.Run.State), stamp(record.Run.CreatedAt), int64(record.ETA.Seconds()), preliminary, []byte("[]"), boolInt(record.Run.CancellationRequested), record.Run.Version)
	return err
}

func (r Repository) Update(ctx context.Context, record Record, expectedVersion int64) error {
	terminal, err := marshal(record.Terminal)
	if err != nil {
		return fmt.Errorf("marshal terminal result: %w", err)
	}
	artifacts, err := json.Marshal(record.Artifacts)
	if err != nil {
		return fmt.Errorf("marshal artifact references: %w", err)
	}
	result, err := r.DB.ExecContext(ctx, `UPDATE validation_runs SET state=?, started_at=?, completed_at=?, terminal_result=?, error_code=?, error_message=?, artifact_refs=?, cancellation_requested=?, version=? WHERE id=? AND version=?`, string(record.Run.State), stamp(record.Run.StartedAt), stamp(record.Run.CompletedAt), terminal, record.ErrorCode, record.Error, artifacts, boolInt(record.Run.CancellationRequested), record.Run.Version, record.Run.ID, expectedVersion)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("stale validation-run version")
	}
	return nil
}

const selectRun = `SELECT id, scenario, target_path, idempotency_key, parent_run_id, state, created_at, started_at, completed_at, eta_seconds, preliminary_result, terminal_result, error_code, error_message, artifact_refs, cancellation_requested, version FROM validation_runs`

func (r Repository) scan(row *sql.Row) (Record, error) {
	var out Record
	var state, created, started, completed string
	var preliminary, terminal, artifacts []byte
	var cancelled int
	err := row.Scan(&out.Run.ID, &out.Run.Target.Scenario, &out.Run.Target.Path, &out.Run.IdempotencyKey, &out.Run.ParentRunID, &state, &created, &started, &completed, &out.ETA, &preliminary, &terminal, &out.ErrorCode, &out.Error, &artifacts, &cancelled, &out.Run.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, &core.LifecycleError{Code: core.ErrorNotFound, Operation: "get", Cause: err}
	}
	if err != nil {
		return Record{}, err
	}
	out.Run.State = core.State(state)
	out.Run.CreatedAt = parseStamp(created)
	out.Run.StartedAt = parseStamp(started)
	out.Run.CompletedAt = parseStamp(completed)
	out.Run.CancellationRequested = cancelled != 0
	if err := unmarshal(preliminary, &out.Preliminary); err != nil {
		return Record{}, err
	}
	if len(terminal) > 0 {
		if err := unmarshal(terminal, &out.Terminal); err != nil {
			return Record{}, err
		}
	}
	_ = json.Unmarshal(artifacts, &out.Artifacts)
	return out, nil
}

func (r Repository) scanRows(rows *sql.Rows) (Record, error) {
	var out Record
	var state, created, started, completed string
	var preliminary, terminal, artifacts []byte
	var cancelled int
	err := rows.Scan(&out.Run.ID, &out.Run.Target.Scenario, &out.Run.Target.Path, &out.Run.IdempotencyKey, &out.Run.ParentRunID, &state, &created, &started, &completed, &out.ETA, &preliminary, &terminal, &out.ErrorCode, &out.Error, &artifacts, &cancelled, &out.Run.Version)
	if err != nil {
		return Record{}, err
	}
	out.Run.State = core.State(state)
	out.Run.CreatedAt = parseStamp(created)
	out.Run.StartedAt = parseStamp(started)
	out.Run.CompletedAt = parseStamp(completed)
	out.Run.CancellationRequested = cancelled != 0
	if err := unmarshal(preliminary, &out.Preliminary); err != nil {
		return Record{}, err
	}
	if len(terminal) > 0 {
		if err := unmarshal(terminal, &out.Terminal); err != nil {
			return Record{}, err
		}
	}
	_ = json.Unmarshal(artifacts, &out.Artifacts)
	return out, nil
}

func marshal(value *scenariovalidationv1.ValidateScenarioResponse) ([]byte, error) {
	if value == nil {
		return []byte{}, nil
	}
	return protojson.MarshalOptions{UseProtoNames: true}.Marshal(value)
}

func unmarshal(data []byte, into **scenariovalidationv1.ValidateScenarioResponse) error {
	if len(data) == 0 {
		return nil
	}
	value := &scenariovalidationv1.ValidateScenarioResponse{}
	if err := protojson.Unmarshal(data, value); err != nil {
		return err
	}
	*into = value
	return nil
}

func stamp(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseStamp(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
