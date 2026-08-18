package scenarioruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Start-operation records (scenario-lifecycle-start-wait-contract plan,
// Phase 3). A start operation is the durable, cross-process view of one
// in-flight `vrooli scenario start|restart`: who is running it, which step it
// is on, and how it ended. It is DERIVED state — progress only, never
// authority. Health and port authority stay on runtime_instances /
// runtime_port_claims; readers treat a dead-initiator record as noise
// (abandoned), never as a reason to refuse a start.

const (
	StartOperationStatusRunning   = "running"
	StartOperationStatusSucceeded = "succeeded"
	StartOperationStatusFailed    = "failed"
	// StartOperationStatusAbandoned marks a record whose initiator died (or
	// was superseded by a newer start) before reaching a terminal state.
	StartOperationStatusAbandoned = "abandoned"

	// StartOperationKeepTerminal bounds terminal record history per
	// (scenario, variant); older terminal records are pruned on begin.
	StartOperationKeepTerminal = 5
	// PhaseDurationKeep bounds the per-(scenario, variant, phase) duration
	// history used for ETA estimates (last-N smoothed).
	PhaseDurationKeep = 10
	// InitiatorTextLimit bounds each recorded provenance string. Command lines
	// are attacker- and agent-shaped input that can run to kilobytes; the
	// leading portion identifies the caller, and the record is forensics, not
	// a transcript.
	InitiatorTextLimit = 512
)

// truncateInitiatorText bounds one provenance string, marking any cut so a
// reader never mistakes a truncated command for the whole one.
func truncateInitiatorText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= InitiatorTextLimit {
		return value
	}
	return value[:InitiatorTextLimit] + "…(truncated)"
}

// Step states within a start operation.
const (
	StartStepRunning = "running"
	StartStepDone    = "done"
	StartStepFailed  = "failed"
)

// StartOperationStep is one recorded step of a start operation, serialized
// into the record's steps_json in execution order.
type StartOperationStep struct {
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

// StartOperation is the durable record of one start/restart invocation.
type StartOperation struct {
	OperationID string
	Scenario    string
	Variant     string
	Operation   string // start|restart
	Status      string // running|succeeded|failed|abandoned
	Verdict     string // health verdict on success (healthy|degraded|running)
	Error       string // failure message on failed
	// InitiatorPID is the orchestrating CLI process. Its liveness is the
	// staleness signal: a running record with a dead initiator is abandoned.
	InitiatorPID *int
	// The remaining initiator fields are provenance, not signal: they answer
	// "who started this?" after the initiating process is gone. A PID alone
	// cannot — it is stale within seconds on a host where short-lived CLIs
	// start work constantly, and PIDs are reused. Argv says what ran,
	// ParentArgv says who ran it, and Scope says where it belonged (on Linux
	// the cgroup, which outlives the process and names the pane, service, or
	// session). All are best-effort and may be empty.
	InitiatorArgv       string
	InitiatorParentPID  *int
	InitiatorParentArgv string
	InitiatorScope      string
	StartedAt           time.Time
	UpdatedAt           time.Time
	FinishedAt          *time.Time
	CurrentStep         string
	DependencyCurrent   string
	DependencyIndex     int
	DependencyTotal     int
	StepsJSON           string
}

// Steps decodes the recorded step list ([] on empty/corrupt JSON — the
// record is best-effort progress, not authority).
func (o StartOperation) Steps() []StartOperationStep {
	if strings.TrimSpace(o.StepsJSON) == "" {
		return nil
	}
	var steps []StartOperationStep
	if err := json.Unmarshal([]byte(o.StepsJSON), &steps); err != nil {
		return nil
	}
	return steps
}

// WithSteps encodes steps onto the record.
func (o *StartOperation) WithSteps(steps []StartOperationStep) {
	if len(steps) == 0 {
		o.StepsJSON = ""
		return
	}
	data, err := json.Marshal(steps)
	if err != nil {
		return
	}
	o.StepsJSON = string(data)
}

// IsTerminal reports whether the operation reached a final state.
func (o StartOperation) IsTerminal() bool {
	switch o.Status {
	case StartOperationStatusSucceeded, StartOperationStatusFailed, StartOperationStatusAbandoned:
		return true
	}
	return false
}

// StartOperationRepository persists start-operation records and the
// per-phase duration history behind ETA estimates.
type StartOperationRepository interface {
	// BeginStartOperation inserts a new running record, marks any prior
	// running records for the same (scenario, variant) abandoned (superseded),
	// and prunes terminal history beyond StartOperationKeepTerminal.
	BeginStartOperation(ctx context.Context, op StartOperation) (StartOperation, error)
	// UpdateStartOperation overwrites the mutable fields of a still-RUNNING
	// record by id. Terminal records are immutable: once a record is
	// abandoned (signal handler, supersede) or finished, a late in-flight
	// flush cannot resurrect it — the update affects zero rows and returns
	// ErrNotFound. The lifecycle is the single writer for a running
	// operation (it holds the scenario lock), so last-write-wins is safe
	// while running.
	UpdateStartOperation(ctx context.Context, op StartOperation) (StartOperation, error)
	// GetLatestStartOperation returns the most recently started record for
	// the instance, ErrNotFound when none exists.
	GetLatestStartOperation(ctx context.Context, scenario, variant string) (StartOperation, error)
	// MarkStartOperationAbandoned transitions a running record to abandoned
	// (dead initiator or explicit Ctrl-C handoff).
	MarkStartOperationAbandoned(ctx context.Context, operationID string, reason string) (StartOperation, error)
	// RecordPhaseDuration appends one successful phase duration and prunes
	// history beyond PhaseDurationKeep.
	RecordPhaseDuration(ctx context.Context, scenario, variant, phase string, duration time.Duration) error
	// PhaseDurationEstimates returns the smoothed (mean of last-N) duration
	// per phase for the instance. Phases with no history are absent — the
	// caller renders "unknown", never a fabricated number.
	PhaseDurationEstimates(ctx context.Context, scenario, variant string) (map[string]time.Duration, error)
}

func (s *SQLiteStore) BeginStartOperation(ctx context.Context, op StartOperation) (StartOperation, error) {
	if strings.TrimSpace(op.Scenario) == "" {
		return StartOperation{}, fmt.Errorf("begin start operation: scenario is required")
	}
	if strings.TrimSpace(op.OperationID) == "" {
		op.OperationID = newID("startop")
	}
	op.Variant = InstanceKey{Scenario: op.Scenario, Variant: op.Variant}.Normalize().Variant
	if op.Operation == "" {
		op.Operation = "start"
	}
	op.Status = StartOperationStatusRunning
	// Bound provenance here rather than trusting callers, so the invariant
	// holds for every writer.
	op.InitiatorArgv = truncateInitiatorText(op.InitiatorArgv)
	op.InitiatorParentArgv = truncateInitiatorText(op.InitiatorParentArgv)
	op.InitiatorScope = truncateInitiatorText(op.InitiatorScope)
	now := s.now()
	if op.StartedAt.IsZero() {
		op.StartedAt = now
	}
	op.UpdatedAt = now

	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		// Supersede any prior running record: only one live orchestration per
		// instance can exist (the scenario lock enforces it); a lingering
		// running row is a crashed predecessor.
		if _, err := tx.ExecContext(ctx, `
UPDATE runtime_start_operations
SET status = ?, updated_at = ?, finished_at = ?, error = CASE WHEN error = '' THEN 'superseded by a newer start' ELSE error END
WHERE scenario = ? AND variant = ? AND status = ?`,
			StartOperationStatusAbandoned, formatTime(now), formatTime(now),
			op.Scenario, op.Variant, StartOperationStatusRunning); err != nil {
			return fmt.Errorf("abandon superseded start operations: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO runtime_start_operations (
  operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
  initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
  started_at, updated_at, finished_at, current_step, dependency_current,
  dependency_index, dependency_total, steps_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			op.OperationID, op.Scenario, op.Variant, op.Operation, op.Status, op.Verdict, op.Error,
			optionalIntValue(op.InitiatorPID), op.InitiatorArgv, optionalIntValue(op.InitiatorParentPID),
			op.InitiatorParentArgv, op.InitiatorScope,
			formatTime(op.StartedAt), formatTime(op.UpdatedAt),
			formatOptionalTime(op.FinishedAt), op.CurrentStep, op.DependencyCurrent,
			op.DependencyIndex, op.DependencyTotal, op.StepsJSON); err != nil {
			return fmt.Errorf("insert start operation: %w", err)
		}
		// Bound history: keep the newest StartOperationKeepTerminal terminal
		// records per instance.
		if _, err := tx.ExecContext(ctx, `
DELETE FROM runtime_start_operations
WHERE scenario = ? AND variant = ? AND status != ?
  AND operation_id NOT IN (
    SELECT operation_id FROM runtime_start_operations
    WHERE scenario = ? AND variant = ? AND status != ?
    ORDER BY started_at DESC, operation_id DESC
    LIMIT ?)`,
			op.Scenario, op.Variant, StartOperationStatusRunning,
			op.Scenario, op.Variant, StartOperationStatusRunning,
			StartOperationKeepTerminal); err != nil {
			return fmt.Errorf("prune start operation history: %w", err)
		}
		return nil
	})
	if err != nil {
		return StartOperation{}, err
	}
	return op, nil
}

func (s *SQLiteStore) UpdateStartOperation(ctx context.Context, op StartOperation) (StartOperation, error) {
	if strings.TrimSpace(op.OperationID) == "" {
		return StartOperation{}, fmt.Errorf("update start operation: operation_id is required")
	}
	op.UpdatedAt = s.now()
	var affected int64
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		// `status = running` guard: terminal records are immutable, so a
		// flush racing the signal-handler abandon (or a takeover's
		// supersede) cannot resurrect an abandoned record to running.
		res, err := tx.ExecContext(ctx, `
UPDATE runtime_start_operations
SET status = ?, verdict = ?, error = ?, updated_at = ?, finished_at = ?,
    current_step = ?, dependency_current = ?, dependency_index = ?,
    dependency_total = ?, steps_json = ?
WHERE operation_id = ? AND status = ?`,
			op.Status, op.Verdict, op.Error, formatTime(op.UpdatedAt), formatOptionalTime(op.FinishedAt),
			op.CurrentStep, op.DependencyCurrent, op.DependencyIndex,
			op.DependencyTotal, op.StepsJSON, op.OperationID, StartOperationStatusRunning)
		if err != nil {
			return fmt.Errorf("update start operation: %w", err)
		}
		affected, err = res.RowsAffected()
		if err != nil {
			return fmt.Errorf("update start operation: %w", err)
		}
		return nil
	})
	if err != nil {
		return StartOperation{}, err
	}
	if affected == 0 {
		return StartOperation{}, ErrNotFound
	}
	return op, nil
}

func (s *SQLiteStore) GetLatestStartOperation(ctx context.Context, scenario, variant string) (StartOperation, error) {
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	row := s.db.QueryRowContext(ctx, `
SELECT operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
       initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
       started_at, updated_at, finished_at, current_step, dependency_current,
       dependency_index, dependency_total, steps_json
FROM runtime_start_operations
WHERE scenario = ? AND variant = ?
ORDER BY started_at DESC, operation_id DESC
LIMIT 1`, scenario, variant)
	return scanStartOperation(row)
}

func (s *SQLiteStore) MarkStartOperationAbandoned(ctx context.Context, operationID string, reason string) (StartOperation, error) {
	if strings.TrimSpace(operationID) == "" {
		return StartOperation{}, fmt.Errorf("abandon start operation: operation_id is required")
	}
	now := s.now()
	var affected int64
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
UPDATE runtime_start_operations
SET status = ?, error = ?, updated_at = ?, finished_at = ?
WHERE operation_id = ? AND status = ?`,
			StartOperationStatusAbandoned, reason, formatTime(now), formatTime(now),
			operationID, StartOperationStatusRunning)
		if err != nil {
			return fmt.Errorf("abandon start operation: %w", err)
		}
		affected, err = res.RowsAffected()
		if err != nil {
			return fmt.Errorf("abandon start operation: %w", err)
		}
		return nil
	})
	if err != nil {
		return StartOperation{}, err
	}
	if affected == 0 {
		return StartOperation{}, ErrNotFound
	}
	return s.getStartOperation(ctx, operationID)
}

func (s *SQLiteStore) getStartOperation(ctx context.Context, operationID string) (StartOperation, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
       initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
       started_at, updated_at, finished_at, current_step, dependency_current,
       dependency_index, dependency_total, steps_json
FROM runtime_start_operations
WHERE operation_id = ?`, operationID)
	return scanStartOperation(row)
}

func (s *SQLiteStore) RecordPhaseDuration(ctx context.Context, scenario, variant, phase string, duration time.Duration) error {
	if strings.TrimSpace(scenario) == "" || strings.TrimSpace(phase) == "" {
		return fmt.Errorf("record phase duration: scenario and phase are required")
	}
	if duration < 0 {
		return fmt.Errorf("record phase duration: negative duration")
	}
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	now := s.now()
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO runtime_phase_durations (scenario, variant, phase, duration_ms, recorded_at)
VALUES (?, ?, ?, ?, ?)`,
			scenario, variant, phase, duration.Milliseconds(), formatTime(now)); err != nil {
			return fmt.Errorf("insert phase duration: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
DELETE FROM runtime_phase_durations
WHERE scenario = ? AND variant = ? AND phase = ?
  AND rowid NOT IN (
    SELECT rowid FROM runtime_phase_durations
    WHERE scenario = ? AND variant = ? AND phase = ?
    ORDER BY recorded_at DESC, rowid DESC
    LIMIT ?)`,
			scenario, variant, phase,
			scenario, variant, phase,
			PhaseDurationKeep); err != nil {
			return fmt.Errorf("prune phase duration history: %w", err)
		}
		return nil
	})
}

func (s *SQLiteStore) PhaseDurationEstimates(ctx context.Context, scenario, variant string) (map[string]time.Duration, error) {
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	rows, err := s.db.QueryContext(ctx, `
SELECT phase, AVG(duration_ms)
FROM runtime_phase_durations
WHERE scenario = ? AND variant = ?
GROUP BY phase`, scenario, variant)
	if err != nil {
		return nil, fmt.Errorf("read phase duration estimates: %w", err)
	}
	defer rows.Close()
	out := map[string]time.Duration{}
	for rows.Next() {
		var phase string
		var avgMillis float64
		if err := rows.Scan(&phase, &avgMillis); err != nil {
			return nil, fmt.Errorf("scan phase duration estimate: %w", err)
		}
		out[phase] = time.Duration(avgMillis) * time.Millisecond
	}
	return out, rows.Err()
}

func scanStartOperation(row *sql.Row) (StartOperation, error) {
	var op StartOperation
	var initiatorPID, initiatorParentPID sql.NullInt64
	var startedAt, updatedAt string
	var finishedAt sql.NullString
	err := row.Scan(&op.OperationID, &op.Scenario, &op.Variant, &op.Operation, &op.Status,
		&op.Verdict, &op.Error, &initiatorPID,
		&op.InitiatorArgv, &initiatorParentPID, &op.InitiatorParentArgv, &op.InitiatorScope,
		&startedAt, &updatedAt, &finishedAt,
		&op.CurrentStep, &op.DependencyCurrent, &op.DependencyIndex, &op.DependencyTotal,
		&op.StepsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return StartOperation{}, ErrNotFound
	}
	if err != nil {
		return StartOperation{}, fmt.Errorf("scan start operation: %w", err)
	}
	if initiatorPID.Valid {
		pid := int(initiatorPID.Int64)
		op.InitiatorPID = &pid
	}
	if initiatorParentPID.Valid {
		pid := int(initiatorParentPID.Int64)
		op.InitiatorParentPID = &pid
	}
	if op.StartedAt, err = parseRequiredTime(startedAt); err != nil {
		return StartOperation{}, fmt.Errorf("parse start operation started_at: %w", err)
	}
	if op.UpdatedAt, err = parseRequiredTime(updatedAt); err != nil {
		return StartOperation{}, fmt.Errorf("parse start operation updated_at: %w", err)
	}
	if op.FinishedAt, err = parseOptionalTime(finishedAt); err != nil {
		return StartOperation{}, fmt.Errorf("parse start operation finished_at: %w", err)
	}
	return op, nil
}
