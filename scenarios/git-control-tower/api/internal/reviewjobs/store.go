// Package reviewjobs owns durable lifecycle state for advisory review and
// draft operations. It stores exact input identity and requested evidence
// references; it never substitutes a newer provider run for a requested one.
package reviewjobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/dbschema"
)

// ErrStorage marks durable-store failures so API adapters can return a stable,
// safe internal-storage category without exposing SQL or driver details.
var ErrStorage = errors.New("review-job storage failure")

func storageFailure(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w during %s: %v", ErrStorage, operation, err)
}

type State string

// MaxAdvisoryEvidenceReceipts covers one durable receipt per supported manifest
// file plus bounded inventory/retry overhead without allowing unbounded job rows.
const MaxAdvisoryEvidenceReceipts = 8192

const (
	Queued           State = "queued"
	AwaitingProvider State = "awaiting_provider"
	Running          State = "running"
	Partial          State = "partial"
	Succeeded        State = "succeeded"
	Failed           State = "failed"
	Cancelled        State = "cancelled"
	Interrupted      State = "interrupted"
)

type Check struct {
	Name         string `json:"name"`
	ExecutionID  string `json:"execution_id,omitempty"`
	Availability string `json:"availability"`
	Verdict      string `json:"verdict"`
	Detail       string `json:"detail,omitempty"`
}
type Job struct {
	Operation              string    `json:"operation,omitempty"`
	RepositoryID           string    `json:"repository_id,omitempty"`
	InputJSON              string    `json:"input_json,omitempty"`
	ExecutionID            string    `json:"execution_id,omitempty"`
	CancelRequested        bool      `json:"cancel_requested,omitempty"`
	DispatchAttempted      bool      `json:"dispatch_attempted,omitempty"`
	DispatchAttemptedKnown bool      `json:"dispatch_attempted_known,omitempty"`
	ID                     string    `json:"id"`
	IdempotencyKey         string    `json:"idempotency_key"`
	InputDigest            string    `json:"input_digest"`
	PolicyVersion          string    `json:"policy_version"`
	State                  State     `json:"state"`
	Phase                  string    `json:"phase,omitempty"`
	Checks                 []Check   `json:"checks"`
	ResultRef              string    `json:"result_ref,omitempty"`
	ResultJSON             string    `json:"result_json,omitempty"`
	Error                  string    `json:"error,omitempty"`
	ErrorCode              string    `json:"error_code,omitempty"`
	Retryable              bool      `json:"retryable,omitempty"`
	ExecutionStarted       bool      `json:"execution_started,omitempty"`
	ExecutionStartedKnown  bool      `json:"execution_started_known,omitempty"`
	TokensUsed             int64     `json:"tokens_used,omitempty"`
	TokensKnown            bool      `json:"tokens_known,omitempty"`
	ChargeMicroUSD         int64     `json:"charge_micro_usd,omitempty"`
	ChargeKnown            bool      `json:"charge_known,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	ScenarioName           string    `json:"scenario_name,omitempty"`
	DetailCount            int       `json:"detail_count,omitempty"`
	DuplicateRetries       int64     `json:"duplicate_retries,omitempty"`
	ThresholdsJSON         string    `json:"thresholds_json,omitempty"`
}

type Store struct{ db dbschema.DB }

func New(db dbschema.DB) (*Store, error) {
	if db == nil {
		return nil, fmt.Errorf("review jobs database is required")
	}
	s := &Store{db: db}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s, s.ensureSchema(ctx)
}

func (s *Store) ensureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS gct_review_jobs (
		id TEXT PRIMARY KEY, idempotency_key TEXT NOT NULL UNIQUE, input_digest TEXT NOT NULL,
		scenario_name TEXT NOT NULL DEFAULT '', detail_count INTEGER NOT NULL DEFAULT 0, thresholds_json TEXT NOT NULL DEFAULT '',
		policy_version TEXT NOT NULL, state TEXT NOT NULL, checks_json TEXT NOT NULL,
		result_ref TEXT NOT NULL DEFAULT '', result_json TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_gct_review_jobs_updated ON gct_review_jobs(updated_at)`); err != nil {
		return err
	}
	for _, column := range []string{"operation TEXT NOT NULL DEFAULT ''", "repository_id TEXT NOT NULL DEFAULT ''", "input_json TEXT NOT NULL DEFAULT ''", "execution_id TEXT NOT NULL DEFAULT ''", "cancel_requested INTEGER NOT NULL DEFAULT 0", "dispatch_attempted INTEGER NOT NULL DEFAULT 0", "dispatch_attempted_known INTEGER NOT NULL DEFAULT 0", "scenario_name TEXT NOT NULL DEFAULT ''", "detail_count INTEGER NOT NULL DEFAULT 0", "thresholds_json TEXT NOT NULL DEFAULT ''", "result_json TEXT NOT NULL DEFAULT ''", "error_code TEXT NOT NULL DEFAULT ''", "retryable INTEGER NOT NULL DEFAULT 0", "execution_started INTEGER NOT NULL DEFAULT 0", "execution_started_known INTEGER NOT NULL DEFAULT 0", "tokens_used INTEGER NOT NULL DEFAULT 0", "tokens_known INTEGER NOT NULL DEFAULT 0", "charge_micro_usd INTEGER NOT NULL DEFAULT 0", "charge_known INTEGER NOT NULL DEFAULT 0", "phase TEXT NOT NULL DEFAULT ''", "duplicate_retries INTEGER NOT NULL DEFAULT 0"} {
		if _, alterErr := s.db.ExecContext(ctx, "ALTER TABLE gct_review_jobs ADD COLUMN "+column); alterErr != nil && !strings.Contains(strings.ToLower(alterErr.Error()), "duplicate column") {
			return alterErr
		}
	}
	return nil
}

func (s *Store) Create(ctx context.Context, job Job) (Job, bool, error) {
	if strings.TrimSpace(job.ID) == "" || strings.TrimSpace(job.IdempotencyKey) == "" || strings.TrimSpace(job.InputDigest) == "" {
		return Job{}, false, fmt.Errorf("id, idempotency_key, and input_digest are required")
	}
	if job.State == "" {
		job.State = Queued
	}
	if !job.State.Valid() {
		return Job{}, false, fmt.Errorf("unknown review-job state %q", job.State)
	}
	if job.Operation == "staged-message" && job.State == Queued && !job.DispatchAttemptedKnown {
		job.DispatchAttemptedKnown = true
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = job.CreatedAt
	data, err := json.Marshal(job.Checks)
	if err != nil {
		return Job{}, false, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO gct_review_jobs (id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested,dispatch_attempted,dispatch_attempted_known,error_code,retryable,execution_started,execution_started_known,tokens_used,tokens_known,charge_micro_usd,charge_known,phase,duplicate_retries) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, job.ID, job.IdempotencyKey, job.InputDigest, job.PolicyVersion, job.State, string(data), job.ResultRef, job.ResultJSON, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano), job.ScenarioName, job.DetailCount, job.ThresholdsJSON, job.Operation, job.RepositoryID, job.InputJSON, job.ExecutionID, job.CancelRequested, job.DispatchAttempted, job.DispatchAttemptedKnown, job.ErrorCode, job.Retryable, job.ExecutionStarted, job.ExecutionStartedKnown, job.TokensUsed, job.TokensKnown, job.ChargeMicroUSD, job.ChargeKnown, job.Phase, job.DuplicateRetries)
	if err != nil {
		return Job{}, false, storageFailure("create", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		if _, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET duplicate_retries=duplicate_retries+1 WHERE idempotency_key=?`, job.IdempotencyKey); err != nil {
			return Job{}, false, storageFailure("record duplicate retry", err)
		}
		existing, getErr := s.GetByIdempotency(ctx, job.IdempotencyKey)
		return existing, false, getErr
	}
	return job, true, nil
}

func (s *Store) Get(ctx context.Context, id string) (Job, error) {
	return s.get(ctx, `WHERE id = ?`, id)
}

func (s *Store) GetByIdempotency(ctx context.Context, key string) (Job, error) {
	return s.get(ctx, `WHERE idempotency_key = ?`, key)
}

func (s *Store) get(ctx context.Context, clause string, arg string) (Job, error) {
	var j Job
	var checks, created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested,dispatch_attempted,dispatch_attempted_known,error_code,retryable,execution_started,execution_started_known,tokens_used,tokens_known,charge_micro_usd,charge_known,phase,duplicate_retries FROM gct_review_jobs `+clause, arg).Scan(&j.ID, &j.IdempotencyKey, &j.InputDigest, &j.PolicyVersion, &j.State, &checks, &j.ResultRef, &j.ResultJSON, &j.Error, &created, &updated, &j.ScenarioName, &j.DetailCount, &j.ThresholdsJSON, &j.Operation, &j.RepositoryID, &j.InputJSON, &j.ExecutionID, &j.CancelRequested, &j.DispatchAttempted, &j.DispatchAttemptedKnown, &j.ErrorCode, &j.Retryable, &j.ExecutionStarted, &j.ExecutionStartedKnown, &j.TokensUsed, &j.TokensKnown, &j.ChargeMicroUSD, &j.ChargeKnown, &j.Phase, &j.DuplicateRetries)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, err
		}
		return Job{}, storageFailure("read", err)
	}
	if !j.State.Valid() {
		return Job{}, fmt.Errorf("decode review job %q: unknown state %q", j.ID, j.State)
	}
	if err := json.Unmarshal([]byte(checks), &j.Checks); err != nil {
		return Job{}, fmt.Errorf("decode checks: %w", err)
	}
	j.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Job{}, err
	}
	j.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return j, err
}

func (s *Store) List(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM gct_review_jobs ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(ids))
	for _, id := range ids {
		job, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *Store) Transition(ctx context.Context, id string, from []State, to State, resultRef, message string) error {
	if !to.Valid() {
		return fmt.Errorf("unknown target state %q", to)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	allowed := make([]string, len(from))
	for i, state := range from {
		candidate := Job{State: state}
		if err := candidate.Transition(to); err != nil {
			return fmt.Errorf("transition from %q: %w", state, err)
		}
		allowed[i] = string(state)
	}
	sort.Strings(allowed)
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(allowed)), ",")
	if len(allowed) == 0 {
		return fmt.Errorf("source state is required")
	}
	args := []any{string(to), resultRef, message, now, id}
	for _, state := range allowed {
		args = append(args, state)
	}
	res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET state=?,result_ref=?,error=?,updated_at=? WHERE id=? AND state IN (`+placeholders+`)`, args...)
	if err != nil {
		return err
	}
	count, _ := res.RowsAffected()
	if count != 1 {
		return fmt.Errorf("job %q is not in an allowed source state", id)
	}
	return nil
}

func (s *Store) SetResult(ctx context.Context, id, resultJSON string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET result_json=?,updated_at=? WHERE id=?`, resultJSON, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("job %q not found", id)
	}
	return nil
}

func (s *Store) UpdateCheck(ctx context.Context, id string, check Check) error {
	job, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	found := false
	for i := range job.Checks {
		if job.Checks[i].Name == check.Name {
			job.Checks[i] = check
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("check %q is not requested", check.Name)
	}
	data, err := json.Marshal(job.Checks)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET checks_json=?,updated_at=? WHERE id=? AND state IN (?,?,?)`, data, time.Now().UTC().Format(time.RFC3339Nano), id, string(Queued), string(Dispatching), string(Running))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return fmt.Errorf("job %q is not active", id)
	}
	return nil
}

// UpdateAdvisory atomically changes an advisory lifecycle and its structured
// payload. A stale provider read cannot overwrite a cancellation or terminal row.
func (s *Store) UpdateAdvisory(ctx context.Context, previous Job, next Job) (bool, error) {
	return s.updateAdvisory(ctx, previous, next, time.Now().UTC())
}

// RequeueUnconfirmedDispatch recovers legacy jobs that were incorrectly
// terminalized after a handle-less owner response. It preserves the original
// idempotency key and immutable input, so the owner can only adopt that same
// execution or confirm that it never existed.
func (s *Store) RequeueUnconfirmedDispatch(ctx context.Context, previous Job) (Job, bool, error) {
	if previous.State != Failed || !previous.DispatchAttempted || previous.ExecutionID != "" || previous.CancelRequested ||
		(previous.ErrorCode != "provider_precondition_failed" && previous.ErrorCode != "dispatch_unconfirmed") {
		return previous, false, nil
	}
	next := previous
	// This guarded recovery is the only authority allowed to requeue a failed
	// dispatch. Keep Failed -> Queued out of the generic lifecycle graph.
	next.State = Queued
	next.ErrorCode = ""
	next.Error = ""
	next.Retryable = false
	next.DispatchAttempted = false
	next.DispatchAttemptedKnown = true
	next.ExecutionStarted = false
	next.ExecutionStartedKnown = false
	next.Phase = string(Queued)
	updatedAt := time.Now().UTC()
	if !updatedAt.After(previous.UpdatedAt) {
		updatedAt = previous.UpdatedAt.Add(time.Nanosecond)
	}
	changed, err := s.updateAdvisory(ctx, previous, next, updatedAt)
	if err != nil || !changed {
		return next, changed, err
	}
	next.UpdatedAt = updatedAt
	return next, true, nil
}

// ClaimAdvisoryDispatch advances the row before a handle-less owner call and
// returns its exact revision. A later retry/cancellation invalidates that
// receipt, so an older refusal cannot overwrite uncertain newer admission.
func (s *Store) ClaimAdvisoryDispatch(ctx context.Context, previous Job) (Job, bool, error) {
	next := previous
	if next.State == Queued {
		if err := next.Transition(Dispatching); err != nil {
			return Job{}, false, err
		}
		next.DispatchAttempted = false
		next.DispatchAttemptedKnown = true
	}
	next.UpdatedAt = time.Now().UTC()
	if !next.UpdatedAt.After(previous.UpdatedAt) {
		next.UpdatedAt = previous.UpdatedAt.Add(time.Nanosecond)
	}
	changed, err := s.updateAdvisory(ctx, previous, next, next.UpdatedAt)
	return next, changed, err
}

// MarkAdvisoryDispatchAttempt records the owner-call boundary after a final
// cancellation check. A cancelled row with a known false marker can safely
// settle without starting; a true marker means recovery must use the same
// idempotency key because the owner may have accepted the request.
func (s *Store) MarkAdvisoryDispatchAttempt(ctx context.Context, previous Job) (Job, bool, error) {
	if previous.State.Terminal() || previous.State != Dispatching && previous.State != Running && previous.State != Cancelling || previous.CancelRequested && previous.DispatchAttemptedKnown && !previous.DispatchAttempted {
		return previous, false, nil
	}
	next := previous
	next.DispatchAttempted = true
	next.DispatchAttemptedKnown = true
	updatedAt := time.Now().UTC()
	if !updatedAt.After(previous.UpdatedAt) {
		updatedAt = previous.UpdatedAt.Add(time.Nanosecond)
	}
	changed, err := s.updateAdvisory(ctx, previous, next, updatedAt)
	if err != nil || !changed {
		return next, changed, err
	}
	next.UpdatedAt = updatedAt
	return next, true, nil
}

func (s *Store) updateAdvisory(ctx context.Context, previous Job, next Job, updatedAt time.Time) (bool, error) {
	transition := previous
	if !transition.State.Valid() {
		return false, fmt.Errorf("unknown previous review-job state %q", transition.State)
	}
	if err := transition.Transition(next.State); err != nil {
		return false, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET state=?,execution_id=?,cancel_requested=?,dispatch_attempted=?,dispatch_attempted_known=?,result_json=?,error=?,error_code=?,retryable=?,execution_started=?,execution_started_known=?,tokens_used=?,tokens_known=?,charge_micro_usd=?,charge_known=?,phase=?,updated_at=? WHERE id=? AND operation=? AND repository_id=? AND state=? AND updated_at=?`, next.State, next.ExecutionID, next.CancelRequested, next.DispatchAttempted, next.DispatchAttemptedKnown, next.ResultJSON, next.Error, next.ErrorCode, next.Retryable, next.ExecutionStarted, next.ExecutionStartedKnown, next.TokensUsed, next.TokensKnown, next.ChargeMicroUSD, next.ChargeKnown, next.Phase, updatedAt.Format(time.RFC3339Nano), previous.ID, previous.Operation, previous.RepositoryID, previous.State, previous.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return false, storageFailure("update lifecycle", err)
	}
	n, e := res.RowsAffected()
	if e != nil {
		return false, storageFailure("read update outcome", e)
	}
	return n == 1, nil
}

// RecordAdvisoryEvidence adds a bounded idempotent read receipt without changing
// the lifecycle revision. Agent tool reads can occur while Start owns that
// revision; touching UpdatedAt here would discard its execution acknowledgement.
func (s *Store) RecordAdvisoryEvidence(ctx context.Context, id, repo, execution string, check Check) error {
	for attempt := 0; attempt < 8; attempt++ {
		var raw string
		if err := s.db.QueryRowContext(ctx, `SELECT checks_json FROM gct_review_jobs WHERE id=? AND operation='staged-message' AND repository_id=? AND state IN (?,?,?,?) AND (execution_id='' OR execution_id=?)`, id, repo, string(Dispatching), string(Running), string(AwaitingInput), string(Cancelling), execution).Scan(&raw); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return err
			}
			return storageFailure("read evidence receipts", err)
		}
		var checks []Check
		if err := json.Unmarshal([]byte(raw), &checks); err != nil {
			return err
		}
		for _, old := range checks {
			if old.Name == check.Name {
				return nil
			}
		}
		if len(checks) >= MaxAdvisoryEvidenceReceipts {
			return fmt.Errorf("evidence receipt limit reached")
		}
		checks = append(checks, check)
		data, err := json.Marshal(checks)
		if err != nil {
			return err
		}
		res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET checks_json=? WHERE id=? AND operation='staged-message' AND repository_id=? AND checks_json=? AND state IN (?,?,?,?) AND (execution_id='' OR execution_id=?)`, string(data), id, repo, raw, string(Dispatching), string(Running), string(AwaitingInput), string(Cancelling), execution)
		if err != nil {
			return storageFailure("write evidence receipt", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return storageFailure("read evidence write outcome", err)
		}
		if n == 1 {
			return nil
		}
	}
	return fmt.Errorf("evidence receipt changed concurrently; retry this page")
}

// PruneAdvisoryCollections bounds unreferenced terminal suggestion collections.
// Any collection retained by a durable message job survives, including terminal
// jobs; active executions and their source evidence are never age-pruned here.
func (s *Store) PruneAdvisoryCollections(ctx context.Context, repositoryID string, keep int) error {
	if keep < 1 {
		return fmt.Errorf("positive collection retention bound is required")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM gct_review_jobs WHERE operation='staged-message-collection' AND repository_id=? AND state='succeeded' AND id NOT IN (SELECT result_ref FROM gct_review_jobs WHERE operation='staged-message' AND result_ref!='') AND id NOT IN (SELECT id FROM gct_review_jobs WHERE operation='staged-message-collection' AND repository_id=? ORDER BY created_at DESC,id DESC LIMIT ?)`, repositoryID, repositoryID, keep)
	return err
}

// RequestAdvisoryCancel is monotonic and evaluates the persisted phase, not a
// caller's stale read. Dispatch must move queued->dispatching before calling its
// owner, so a queued cancellation can safely finish without launching work.
func (s *Store) RequestAdvisoryCancel(ctx context.Context, id, repositoryID string) error {
	for attempt := 0; attempt < 8; attempt++ {
		job, err := s.Get(ctx, id)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		if job.Operation != "staged-message" || job.RepositoryID != repositoryID || job.State.Terminal() {
			return nil
		}
		next := job
		target := Cancelling
		if job.State == AwaitingProvider || job.State == Queued && job.ExecutionID == "" || job.ExecutionID == "" && job.DispatchAttemptedKnown && !job.DispatchAttempted {
			target = Cancelled
		}
		if err := next.Transition(target); err != nil {
			return err
		}
		next.CancelRequested = true
		updatedAt := time.Now().UTC()
		if !updatedAt.After(job.UpdatedAt) {
			updatedAt = job.UpdatedAt.Add(time.Nanosecond)
		}
		res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET state=?,cancel_requested=1,updated_at=? WHERE id=? AND operation='staged-message' AND repository_id=? AND state=? AND updated_at=?`, string(next.State), updatedAt.Format(time.RFC3339Nano), id, repositoryID, string(job.State), job.UpdatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return storageFailure("request cancellation", err)
		}
		count, err := res.RowsAffected()
		if err != nil {
			return storageFailure("read cancellation outcome", err)
		}
		if count == 1 {
			return nil
		}
	}
	return fmt.Errorf("job %q changed repeatedly while requesting cancellation", id)
}
