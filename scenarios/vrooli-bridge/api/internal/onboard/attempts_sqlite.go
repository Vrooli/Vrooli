package onboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var _ AttemptStore = (*sqliteRepository)(nil)

func (s *sqliteRepository) CreateAttempt(ctx context.Context, attempt EnrollmentAttempt) (EnrollmentAttempt, error) {
	if attempt.ID == "" || attempt.CorrelationID == "" || attempt.MachineID == "" {
		return EnrollmentAttempt{}, ErrInvalid{Field: "attempt", Reason: "id, machine_id, and correlation_id are required"}
	}
	if attempt.State == "" {
		attempt.State = AttemptCreated
	}
	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = s.clock.Now().UTC()
	}
	snapshot, err := marshalAttemptSnapshot(attempt.InputSnapshot)
	if err != nil {
		return EnrollmentAttempt{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO enrollment_attempts (id,machine_id,retry_of_attempt_id,correlation_id,state,input_snapshot,terminal_result,diagnostics,created_at,terminal_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		attempt.ID, attempt.MachineID, attempt.RetryOfAttemptID, attempt.CorrelationID, string(attempt.State), snapshot, attempt.TerminalResult, attempt.Diagnostics, attempt.CreatedAt.Format(opTimeFormat), formatNullableTime(attempt.TerminalAt))
	if err != nil {
		return EnrollmentAttempt{}, fmt.Errorf("insert enrollment attempt: %w", err)
	}
	return attempt, nil
}

func (s *sqliteRepository) GetAttempt(ctx context.Context, id string) (EnrollmentAttempt, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,machine_id,retry_of_attempt_id,correlation_id,state,input_snapshot,terminal_result,diagnostics,created_at,terminal_at FROM enrollment_attempts WHERE id=?`, id)
	attempt, err := scanAttempt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return EnrollmentAttempt{}, ErrOpNotFound{ID: id}
	}
	if err != nil {
		return EnrollmentAttempt{}, fmt.Errorf("get enrollment attempt: %w", err)
	}
	return attempt, nil
}

func (s *sqliteRepository) GetAttemptByCorrelation(ctx context.Context, correlationID string) (EnrollmentAttempt, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,machine_id,retry_of_attempt_id,correlation_id,state,input_snapshot,terminal_result,diagnostics,created_at,terminal_at FROM enrollment_attempts WHERE correlation_id=?`, correlationID)
	attempt, err := scanAttempt(row)
	if errors.Is(err, sql.ErrNoRows) {
		return EnrollmentAttempt{}, ErrOpNotFound{ID: correlationID}
	}
	if err != nil {
		return EnrollmentAttempt{}, fmt.Errorf("get enrollment attempt by correlation: %w", err)
	}
	return attempt, nil
}

// ListAttemptsForMachine returns immutable enrollment history newest first. It
// is intentionally keyed by the durable Machine ID rather than a host or a
// mutable onboarding operation, so recovery reads cannot conflate identities.
func (s *sqliteRepository) ListAttemptsForMachine(ctx context.Context, machineID string) ([]EnrollmentAttempt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,machine_id,retry_of_attempt_id,correlation_id,state,input_snapshot,terminal_result,diagnostics,created_at,terminal_at FROM enrollment_attempts WHERE machine_id=? ORDER BY created_at DESC,id DESC`, machineID)
	if err != nil {
		return nil, fmt.Errorf("list enrollment attempts for machine: %w", err)
	}
	defer rows.Close()
	attempts := make([]EnrollmentAttempt, 0)
	for rows.Next() {
		attempt, scanErr := scanAttempt(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enrollment attempts for machine: %w", err)
	}
	return attempts, nil
}

func (s *sqliteRepository) RetryAttempt(ctx context.Context, id string, snapshot map[string]string) (EnrollmentAttempt, error) {
	prior, err := s.GetAttempt(ctx, id)
	if err != nil {
		return EnrollmentAttempt{}, err
	}
	if !prior.State.Terminal() {
		return EnrollmentAttempt{}, ErrInvalid{Field: "attempt", Reason: "only terminal attempts can be retried"}
	}
	if snapshot == nil {
		snapshot = prior.InputSnapshot
	}
	next, err := NewAttempt(prior.MachineID, snapshot)
	if err != nil {
		return EnrollmentAttempt{}, err
	}
	next.RetryOfAttemptID = prior.ID
	return s.CreateAttempt(ctx, next)
}

func (s *sqliteRepository) CompleteAttempt(ctx context.Context, id string, state AttemptState, result, diagnostics string) (EnrollmentAttempt, error) {
	if !state.Terminal() {
		return EnrollmentAttempt{}, ErrInvalid{Field: "state", Reason: "terminal state required"}
	}
	attempt, err := s.GetAttempt(ctx, id)
	if err != nil {
		return EnrollmentAttempt{}, err
	}
	if attempt.State.Terminal() {
		if attempt.State == state && attempt.TerminalResult == result {
			return attempt, nil
		}
		return EnrollmentAttempt{}, ErrInvalid{Field: "attempt", Reason: "terminal attempt is immutable"}
	}
	now := s.clock.Now().UTC()
	resultSQL, err := s.db.ExecContext(ctx, `UPDATE enrollment_attempts SET state=?,terminal_result=?,diagnostics=?,terminal_at=? WHERE id=? AND state IN ('created','running')`, string(state), result, diagnostics, now.Format(opTimeFormat), id)
	if err != nil {
		return EnrollmentAttempt{}, fmt.Errorf("complete enrollment attempt: %w", err)
	}
	changed, _ := resultSQL.RowsAffected()
	if changed == 0 {
		return s.GetAttempt(ctx, id)
	}
	return s.GetAttempt(ctx, id)
}

func (s *sqliteRepository) RecordCheckpoint(ctx context.Context, attemptID, checkpoint, postcondition string) error {
	if checkpoint == "" || postcondition == "" {
		return ErrInvalid{Field: "checkpoint", Reason: "checkpoint and postcondition required"}
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO enrollment_checkpoints (attempt_id,checkpoint,postcondition,recorded_at) VALUES (?,?,?,?)`, attemptID, checkpoint, postcondition, s.clock.Now().UTC().Format(opTimeFormat))
	if err != nil {
		return fmt.Errorf("record enrollment checkpoint: %w", err)
	}
	return nil
}

func scanAttempt(scanner interface{ Scan(...any) error }) (EnrollmentAttempt, error) {
	var attempt EnrollmentAttempt
	var state, snapshot, created, terminal string
	if err := scanner.Scan(&attempt.ID, &attempt.MachineID, &attempt.RetryOfAttemptID, &attempt.CorrelationID, &state, &snapshot, &attempt.TerminalResult, &attempt.Diagnostics, &created, &terminal); err != nil {
		return EnrollmentAttempt{}, err
	}
	attempt.State = AttemptState(state)
	if err := json.Unmarshal([]byte(snapshot), &attempt.InputSnapshot); err != nil {
		return EnrollmentAttempt{}, fmt.Errorf("decode attempt snapshot: %w", err)
	}
	var err error
	if attempt.CreatedAt, err = time.Parse(opTimeFormat, created); err != nil {
		return EnrollmentAttempt{}, err
	}
	if attempt.TerminalAt, err = parseNullableTime(terminal); err != nil {
		return EnrollmentAttempt{}, err
	}
	return attempt, nil
}
