// Package validationrun owns Workflow Health's durable validation-run ledger.
package validationrun

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
	"workflow-health/internal/execution"

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
	// nil is the compatible omitted/default selection; non-nil is explicit.
	BrowserCasePaths []string
	Run              core.Run
	ETA              time.Duration
	Preliminary      *scenariovalidationv1.ValidateScenarioResponse
	Terminal         *scenariovalidationv1.ValidateScenarioResponse
	ErrorCode        string
	Error            string
	Artifacts        []string
	ExecutionBinding *scenariovalidationv1.DesktopValidationBinding
}

// ActiveRun is the bounded durable inventory item used by lifecycle drain.
type ActiveRun struct {
	ID    string
	State core.State
}

// LifecycleFence is the provider-owned admission state retained across a
// managed process replacement.
type LifecycleFence struct {
	Closed    bool
	Operation string
	Revision  int64
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

func (r Repository) ListActive(ctx context.Context) ([]ActiveRun, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, state FROM validation_runs WHERE state IN (?, ?) ORDER BY created_at, id`, string(core.StateQueued), string(core.StateRunning))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ActiveRun
	for rows.Next() {
		var item ActiveRun
		var state string
		if err := rows.Scan(&item.ID, &state); err != nil {
			return nil, err
		}
		item.State = core.State(state)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) GetLifecycleFence(ctx context.Context) (LifecycleFence, error) {
	var fence LifecycleFence
	var closed int
	err := r.DB.QueryRowContext(ctx, `SELECT closed, operation_id, revision FROM validation_lifecycle_fence WHERE singleton = 1`).Scan(&closed, &fence.Operation, &fence.Revision)
	if err != nil {
		return LifecycleFence{}, err
	}
	fence.Closed = closed != 0
	return fence, nil
}

// UpdateLifecycleFence uses a single compare-and-swap statement so a stale
// lifecycle request cannot overwrite a newer revision.
func (r Repository) UpdateLifecycleFence(ctx context.Context, expectedRevision int64, next LifecycleFence) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE validation_lifecycle_fence SET closed = ?, operation_id = ?, revision = ? WHERE singleton = 1 AND revision = ?`, boolInt(next.Closed), next.Operation, next.Revision, expectedRevision)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("stale lifecycle fence revision %d", expectedRevision)
	}
	return nil
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
	var selection []byte
	if record.BrowserCasePaths != nil {
		paths, err := execution.NormalizeCasePaths(record.BrowserCasePaths)
		if err != nil {
			return err
		}
		selection, err = json.Marshal(paths)
		if err != nil {
			return err
		}
	}
	preliminary, err := marshal(record.Preliminary)
	if err != nil {
		return fmt.Errorf("marshal preliminary result: %w", err)
	}
	binding, err := marshalBinding(record.ExecutionBinding)
	if err != nil {
		return fmt.Errorf("marshal execution binding: %w", err)
	}
	_, err = r.DB.ExecContext(ctx, `INSERT INTO validation_runs (id, scenario, target_path, idempotency_key, parent_run_id, state, created_at, eta_seconds, preliminary_result, artifact_refs, execution_binding, browser_case_paths, cancellation_requested, version) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, record.Run.ID, record.Run.Target.Scenario, record.Run.Target.Path, record.Run.IdempotencyKey, record.Run.ParentRunID, string(record.Run.State), stamp(record.Run.CreatedAt), int64(record.ETA.Seconds()), preliminary, []byte("[]"), binding, selection, boolInt(record.Run.CancellationRequested), record.Run.Version)
	return err
}

func (r Repository) Update(ctx context.Context, record Record, expectedVersion int64) error {
	current, err := r.Get(ctx, record.Run.ID)
	if err != nil {
		return err
	}
	if (current.BrowserCasePaths == nil) != (record.BrowserCasePaths == nil) || !slices.Equal(current.BrowserCasePaths, record.BrowserCasePaths) {
		return errors.New("validation-run selection is immutable")
	}
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

const selectRun = `SELECT id, scenario, target_path, idempotency_key, parent_run_id, state, created_at, started_at, completed_at, eta_seconds, preliminary_result, terminal_result, error_code, error_message, artifact_refs, execution_binding, browser_case_paths, cancellation_requested, version FROM validation_runs`

func (r Repository) scan(row *sql.Row) (Record, error) {
	var out Record
	var state, created, started, completed string
	var preliminary, terminal, artifacts []byte
	var cancelled int
	var binding, selection []byte
	err := row.Scan(&out.Run.ID, &out.Run.Target.Scenario, &out.Run.Target.Path, &out.Run.IdempotencyKey, &out.Run.ParentRunID, &state, &created, &started, &completed, &out.ETA, &preliminary, &terminal, &out.ErrorCode, &out.Error, &artifacts, &binding, &selection, &cancelled, &out.Run.Version)
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
	if err := unmarshalBinding(binding, &out.ExecutionBinding); err != nil {
		return Record{}, err
	}
	if len(selection) > 0 {
		if err := json.Unmarshal(selection, &out.BrowserCasePaths); err != nil {
			return Record{}, fmt.Errorf("decode browser case selection: %w", err)
		}
		if _, err := execution.NormalizeCasePaths(out.BrowserCasePaths); err != nil {
			return Record{}, err
		}
	}
	return out, nil
}

func (r Repository) scanRows(rows *sql.Rows) (Record, error) {
	var out Record
	var state, created, started, completed string
	var preliminary, terminal, artifacts []byte
	var cancelled int
	var binding, selection []byte
	err := rows.Scan(&out.Run.ID, &out.Run.Target.Scenario, &out.Run.Target.Path, &out.Run.IdempotencyKey, &out.Run.ParentRunID, &state, &created, &started, &completed, &out.ETA, &preliminary, &terminal, &out.ErrorCode, &out.Error, &artifacts, &binding, &selection, &cancelled, &out.Run.Version)
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
	if err := unmarshalBinding(binding, &out.ExecutionBinding); err != nil {
		return Record{}, err
	}
	if len(selection) > 0 {
		if err := json.Unmarshal(selection, &out.BrowserCasePaths); err != nil {
			return Record{}, fmt.Errorf("decode browser case selection: %w", err)
		}
		if _, err := execution.NormalizeCasePaths(out.BrowserCasePaths); err != nil {
			return Record{}, err
		}
	}
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

func marshalBinding(binding *scenariovalidationv1.DesktopValidationBinding) ([]byte, error) {
	if binding == nil {
		return nil, nil
	}
	return protojson.MarshalOptions{UseProtoNames: true}.Marshal(binding)
}

func unmarshalBinding(data []byte, into **scenariovalidationv1.DesktopValidationBinding) error {
	if len(data) == 0 {
		return nil
	}
	binding := &scenariovalidationv1.DesktopValidationBinding{}
	if err := protojson.Unmarshal(data, binding); err != nil {
		return err
	}
	*into = binding
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
