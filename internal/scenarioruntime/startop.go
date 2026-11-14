package scenarioruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	startopParameterA = 0.50
	startopParameterB = 0.90
	startopParameterC = 1000
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
	// FenceOperationID links this start attempt to the provider's independent
	// maintenance-fence identity. The start-attempt ID remains unique and
	// immutable; a later retry may reuse this fence ID after a replacement
	// failed, without reusing or mutating the old attempt record.
	FenceOperationID string
	Scenario         string
	Variant          string
	Operation        string // start|restart
	Status           string // running|succeeded|failed|abandoned
	Verdict          string // health verdict on success (healthy|degraded|running)
	Error            string // failure message on failed
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
	// SetStartOperationFenceOperationID links a live attempt to its independent
	// provider maintenance fence before the lifecycle crosses a stop boundary.
	// The link is write-once and terminal attempts are immutable.
	SetStartOperationFenceOperationID(ctx context.Context, operationID, fenceOperationID string) error
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
	// GetStartOperation reads one exact admitted attempt for an attached waiter.
	GetStartOperation(ctx context.Context, operationID string) (StartOperation, error)
	// GetStartOperationFenceRecoveryOwner returns the newest linked terminal
	// attempt only when it failed or was abandoned. Unlinked preparation
	// failures must not erase the durable owner of an already closed fence.
	GetStartOperationFenceRecoveryOwner(ctx context.Context, scenario, variant string) (StartOperation, error)
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
	// StartTimingSummaries returns step timing aggregates over retained terminal
	// start-operation records. An empty scenario includes both per-scenario and
	// fleet rows; a named scenario only includes that scenario.
	StartTimingSummaries(ctx context.Context, scenario string) ([]StartTimingSummary, error)
}

func (s *SQLiteStore) SetStartOperationFenceOperationID(ctx context.Context, operationID, fenceOperationID string) error {
	operationID = strings.TrimSpace(operationID)
	fenceOperationID = strings.TrimSpace(fenceOperationID)
	if operationID == "" || fenceOperationID == "" {
		return fmt.Errorf("link start operation fence: both operation IDs are required")
	}
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var status, current string
		if err := tx.QueryRowContext(ctx, `SELECT status, fence_operation_id FROM runtime_start_operations WHERE operation_id = ?`, operationID).Scan(&status, &current); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("read start operation before fence link: %w", err)
		}
		if status != StartOperationStatusRunning {
			return fmt.Errorf("start operation %q is %s; provider fence link requires a running attempt", operationID, status)
		}
		if current == fenceOperationID {
			return nil
		}
		if current != "" {
			return fmt.Errorf("start operation %q already links provider fence %q", operationID, current)
		}
		result, err := tx.ExecContext(ctx, `UPDATE runtime_start_operations SET fence_operation_id = ? WHERE operation_id = ? AND status = ? AND fence_operation_id = ''`, fenceOperationID, operationID, StartOperationStatusRunning)
		if err != nil {
			return fmt.Errorf("write provider fence link: %w", err)
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("confirm provider fence link: %w", err)
		}
		if updated != 1 {
			return fmt.Errorf("start operation %q changed while linking provider fence", operationID)
		}
		return nil
	})
}

// StartTimingSummary is the typed read model behind `vrooli scenario
// timings`. Durations are milliseconds so the JSON surface remains stable and
// easy for non-Go consumers to decode.
type StartTimingSummary struct {
	Scenario  string  `json:"scenario"`
	Operation string  `json:"operation"`
	Step      string  `json:"step"`
	Count     int     `json:"count"`
	MeanMS    float64 `json:"mean_ms"`
	P50MS     float64 `json:"p50_ms"`
	P90MS     float64 `json:"p90_ms"`
	TotalMS   float64 `json:"total_ms"`
	Share     float64 `json:"share"`
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
  operation_id, fence_operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
  initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
  started_at, updated_at, finished_at, current_step, dependency_current,
  dependency_index, dependency_total, steps_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			op.OperationID, op.FenceOperationID, op.Scenario, op.Variant, op.Operation, op.Status, op.Verdict, op.Error,
			optionalIntValue(op.InitiatorPID), op.InitiatorArgv, optionalIntValue(op.InitiatorParentPID),
			op.InitiatorParentArgv, op.InitiatorScope,
			formatTime(op.StartedAt), formatTime(op.UpdatedAt),
			formatOptionalTime(op.FinishedAt), op.CurrentStep, op.DependencyCurrent,
			op.DependencyIndex, op.DependencyTotal, op.StepsJSON); err != nil {
			return fmt.Errorf("insert start operation: %w", err)
		}
		// Keep bounded ordinary history plus at most one latest linked terminal
		// owner. Repeated failed preparations must not prune the recovery proof
		// for a fence still closed in a replacement provider.
		if _, err := tx.ExecContext(ctx, `
DELETE FROM runtime_start_operations
WHERE scenario = ? AND variant = ? AND status != ?
  AND operation_id NOT IN (
    SELECT operation_id FROM runtime_start_operations
    WHERE scenario = ? AND variant = ? AND status != ?
    ORDER BY rowid DESC
    LIMIT ?)
  AND operation_id NOT IN (
    SELECT operation_id FROM runtime_start_operations
    WHERE scenario = ? AND variant = ? AND status != ? AND fence_operation_id != ''
    ORDER BY rowid DESC LIMIT 1)`,
			op.Scenario, op.Variant, StartOperationStatusRunning,
			op.Scenario, op.Variant, StartOperationStatusRunning,
			StartOperationKeepTerminal,
			op.Scenario, op.Variant, StartOperationStatusRunning); err != nil {
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
SELECT operation_id, fence_operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
       initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
       started_at, updated_at, finished_at, current_step, dependency_current,
       dependency_index, dependency_total, steps_json
FROM runtime_start_operations
WHERE scenario = ? AND variant = ?
ORDER BY rowid DESC
LIMIT 1`, scenario, variant)
	return scanStartOperation(row)
}

func (s *SQLiteStore) GetStartOperationFenceRecoveryOwner(ctx context.Context, scenario, variant string) (StartOperation, error) {
	variant = InstanceKey{Scenario: scenario, Variant: variant}.Normalize().Variant
	row := s.db.QueryRowContext(ctx, `
SELECT operation_id, fence_operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
       initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
       started_at, updated_at, finished_at, current_step, dependency_current,
       dependency_index, dependency_total, steps_json
FROM runtime_start_operations
WHERE scenario = ? AND variant = ? AND fence_operation_id != '' AND status != ?
ORDER BY rowid DESC LIMIT 1`, scenario, variant, StartOperationStatusRunning)
	op, err := scanStartOperation(row)
	if err != nil {
		return StartOperation{}, err
	}
	if op.Status != StartOperationStatusFailed && op.Status != StartOperationStatusAbandoned {
		return StartOperation{}, ErrNotFound
	}
	return op, nil
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
SELECT operation_id, fence_operation_id, scenario, variant, operation, status, verdict, error, initiator_pid,
       initiator_argv, initiator_parent_pid, initiator_parent_argv, initiator_scope,
       started_at, updated_at, finished_at, current_step, dependency_current,
       dependency_index, dependency_total, steps_json
FROM runtime_start_operations
WHERE operation_id = ?`, operationID)
	return scanStartOperation(row)
}

func (s *SQLiteStore) GetStartOperation(ctx context.Context, operationID string) (StartOperation, error) {
	return s.getStartOperation(ctx, operationID)
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

type timingSamples struct {
	scenario  string
	operation string
	step      string
	values    []float64
}

// StartTimingSummaries reads the durable operation rows and derives timing
// statistics in Go. The operation record is intentionally the source of truth
// for this view: its JSON step list preserves the operation dimension that the
// phase-duration ETA table does not retain.
func (s *SQLiteStore) StartTimingSummaries(ctx context.Context, scenario string) ([]StartTimingSummary, error) {
	return s.startTimingSummariesSince(ctx, scenario, time.Time{})
}

// StartTimingSummariesSince limits the retained timing view to records at or
// after since. A zero time preserves the all-retained-history behavior.
func (s *SQLiteStore) StartTimingSummariesSince(ctx context.Context, scenario string, since time.Time) ([]StartTimingSummary, error) {
	return s.startTimingSummariesSince(ctx, scenario, since)
}

func (s *SQLiteStore) startTimingSummariesSince(ctx context.Context, scenario string, since time.Time) ([]StartTimingSummary, error) {
	scenario = strings.TrimSpace(scenario)
	query := `
SELECT scenario, operation, steps_json
FROM runtime_start_operations
WHERE status != ?`
	args := []any{StartOperationStatusRunning}
	if !since.IsZero() {
		// runtime_start_operations records operation time as started_at. The
		// phase-duration table has recorded_at, but it does not retain the
		// operation step JSON needed by this view.
		query += " AND started_at >= ?"
		args = append(args, since.UTC().Format(time.RFC3339Nano))
	}
	if scenario != "" {
		query += " AND scenario = ?"
		args = append(args, scenario)
	}
	query += " ORDER BY scenario, started_at, operation_id"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read start timing records: %w", err)
	}
	defer rows.Close()

	byScenario := map[string]map[string]*timingSamples{}
	byFleet := map[string]*timingSamples{}
	for rows.Next() {
		var recordScenario, operation, stepsJSON string
		if err := rows.Scan(&recordScenario, &operation, &stepsJSON); err != nil {
			return nil, fmt.Errorf("scan start timing record: %w", err)
		}
		var steps []StartOperationStep
		if strings.TrimSpace(stepsJSON) == "" || json.Unmarshal([]byte(stepsJSON), &steps) != nil {
			continue
		}
		for _, step := range steps {
			if step.EndedAt == nil || step.StartedAt.IsZero() || step.EndedAt.Before(step.StartedAt) {
				continue
			}
			key := operation + "\x00" + step.Name
			if byScenario[recordScenario] == nil {
				byScenario[recordScenario] = map[string]*timingSamples{}
			}
			if byScenario[recordScenario][key] == nil {
				byScenario[recordScenario][key] = &timingSamples{scenario: recordScenario, operation: operation, step: step.Name}
			}
			byScenario[recordScenario][key].values = append(byScenario[recordScenario][key].values, float64(step.EndedAt.Sub(step.StartedAt).Microseconds())/startopParameterC)
			if scenario == "" {
				if byFleet[key] == nil {
					byFleet[key] = &timingSamples{scenario: "fleet", operation: operation, step: step.Name}
				}
				byFleet[key].values = append(byFleet[key].values, float64(step.EndedAt.Sub(step.StartedAt).Microseconds())/startopParameterC)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read start timing records: %w", err)
	}

	return buildTimingSummaries(scenario, byScenario, byFleet), nil
}

func buildTimingSummaries(scenario string, byScenario map[string]map[string]*timingSamples, byFleet map[string]*timingSamples) []StartTimingSummary {
	result := make([]StartTimingSummary, 0)
	if scenario == "" {
		scenarios := make([]string, 0, len(byScenario))
		for name := range byScenario {
			scenarios = append(scenarios, name)
		}
		slices.Sort(scenarios)
		for _, name := range scenarios {
			result = append(result, summarizeTimingMap(byScenario[name])...)
		}
		fleet := make([]*timingSamples, 0, len(byFleet))
		for _, sample := range byFleet {
			fleet = append(fleet, sample)
		}
		result = append(result, summarizeTimingSamples(fleet)...)
		return result
	}

	return summarizeTimingMap(byScenario[scenario])
}

func summarizeTimingMap(samples map[string]*timingSamples) []StartTimingSummary {
	values := make([]*timingSamples, 0, len(samples))
	for _, sample := range samples {
		values = append(values, sample)
	}
	return summarizeTimingSamples(values)
}

func summarizeTimingSamples(values []*timingSamples) []StartTimingSummary {
	sort.Slice(values, func(i, j int) bool {
		if values[i].scenario != values[j].scenario {
			return values[i].scenario < values[j].scenario
		}
		if values[i].operation != values[j].operation {
			return values[i].operation < values[j].operation
		}
		return values[i].step < values[j].step
	})
	var scopeTotal float64
	for _, sample := range values {
		for _, value := range sample.values {
			scopeTotal += value
		}
	}
	result := make([]StartTimingSummary, 0, len(values))
	for _, sample := range values {
		if len(sample.values) == 0 {
			continue
		}
		sorted := append([]float64(nil), sample.values...)
		sort.Float64s(sorted)
		var total float64
		for _, value := range sorted {
			total += value
		}
		result = append(result, StartTimingSummary{
			Scenario:  sample.scenario,
			Operation: sample.operation,
			Step:      sample.step,
			Count:     len(sorted),
			MeanMS:    total / float64(len(sorted)),
			P50MS:     percentile(sorted, startopParameterA),
			P90MS:     percentile(sorted, startopParameterB),
			TotalMS:   total,
			Share:     total / scopeTotal,
		})
	}
	return result
}

// percentile uses linear interpolation for startup telemetry. This is a
// descriptive timing summary, not a scheduler admission input, so it does not
// adopt Test Genie's observed-value nearest-rank convention.
func percentile(sorted []float64, fraction float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := fraction * float64(len(sorted)-1)
	lower := int(position)
	upper := lower + 1
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	weight := position - float64(lower)
	return sorted[lower] + weight*(sorted[upper]-sorted[lower])
}

func scanStartOperation(row *sql.Row) (StartOperation, error) {
	var op StartOperation
	var initiatorPID, initiatorParentPID sql.NullInt64
	var startedAt, updatedAt string
	var finishedAt sql.NullString
	err := row.Scan(&op.OperationID, &op.FenceOperationID, &op.Scenario, &op.Variant, &op.Operation, &op.Status,
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
