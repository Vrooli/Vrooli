// Package reviewjobs owns durable lifecycle state for advisory review and
// draft operations. It stores exact input identity and requested evidence
// references; it never substitutes a newer provider run for a requested one.
package reviewjobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/dbschema"
)

type State string

const (
	Queued      State = "queued"
	Running     State = "running"
	Partial     State = "partial"
	Succeeded   State = "succeeded"
	Failed      State = "failed"
	Cancelled   State = "cancelled"
	Interrupted State = "interrupted"
)

type Check struct {
	Name         string `json:"name"`
	ExecutionID  string `json:"execution_id,omitempty"`
	Availability string `json:"availability"`
	Verdict      string `json:"verdict"`
	Detail       string `json:"detail,omitempty"`
}
type Job struct {
	Operation       string    `json:"operation,omitempty"`
	RepositoryID    string    `json:"repository_id,omitempty"`
	InputJSON       string    `json:"input_json,omitempty"`
	ExecutionID     string    `json:"execution_id,omitempty"`
	CancelRequested bool      `json:"cancel_requested,omitempty"`
	ID              string    `json:"id"`
	IdempotencyKey  string    `json:"idempotency_key"`
	InputDigest     string    `json:"input_digest"`
	PolicyVersion   string    `json:"policy_version"`
	State           State     `json:"state"`
	Checks          []Check   `json:"checks"`
	ResultRef       string    `json:"result_ref,omitempty"`
	ResultJSON      string    `json:"result_json,omitempty"`
	Error           string    `json:"error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ScenarioName    string    `json:"scenario_name,omitempty"`
	DetailCount     int       `json:"detail_count,omitempty"`
	ThresholdsJSON  string    `json:"thresholds_json,omitempty"`
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
	for _, column := range []string{"operation TEXT NOT NULL DEFAULT ''", "repository_id TEXT NOT NULL DEFAULT ''", "input_json TEXT NOT NULL DEFAULT ''", "execution_id TEXT NOT NULL DEFAULT ''", "cancel_requested INTEGER NOT NULL DEFAULT 0", "scenario_name TEXT NOT NULL DEFAULT ''", "detail_count INTEGER NOT NULL DEFAULT 0", "thresholds_json TEXT NOT NULL DEFAULT ''", "result_json TEXT NOT NULL DEFAULT ''"} {
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
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	job.UpdatedAt = job.CreatedAt
	data, err := json.Marshal(job.Checks)
	if err != nil {
		return Job{}, false, err
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO gct_review_jobs (id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, job.ID, job.IdempotencyKey, job.InputDigest, job.PolicyVersion, job.State, string(data), job.ResultRef, job.ResultJSON, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano), job.ScenarioName, job.DetailCount, job.ThresholdsJSON, job.Operation, job.RepositoryID, job.InputJSON, job.ExecutionID, job.CancelRequested)
	if err != nil {
		return Job{}, false, err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
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
	err := s.db.QueryRowContext(ctx, `SELECT id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json,operation,repository_id,input_json,execution_id,cancel_requested FROM gct_review_jobs `+clause, arg).Scan(&j.ID, &j.IdempotencyKey, &j.InputDigest, &j.PolicyVersion, &j.State, &checks, &j.ResultRef, &j.ResultJSON, &j.Error, &created, &updated, &j.ScenarioName, &j.DetailCount, &j.ThresholdsJSON, &j.Operation, &j.RepositoryID, &j.InputJSON, &j.ExecutionID, &j.CancelRequested)
	if err != nil {
		return Job{}, err
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
	if to == "" {
		return fmt.Errorf("target state is required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	allowed := make([]string, len(from))
	for i, state := range from {
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
	res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET checks_json=?,updated_at=? WHERE id=? AND state IN (?,?)`, data, time.Now().UTC().Format(time.RFC3339Nano), id, string(Queued), string(Running))
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

// ClaimAdvisoryDispatch advances the row before a handle-less owner call and
// returns its exact revision. A later retry/cancellation invalidates that
// receipt, so an older refusal cannot overwrite uncertain newer admission.
func (s *Store) ClaimAdvisoryDispatch(ctx context.Context, previous Job) (Job, bool, error) {
	next := previous
	if next.State == Queued {
		next.State = Running
	}
	next.UpdatedAt = time.Now().UTC()
	if !next.UpdatedAt.After(previous.UpdatedAt) {
		next.UpdatedAt = previous.UpdatedAt.Add(time.Nanosecond)
	}
	changed, err := s.updateAdvisory(ctx, previous, next, next.UpdatedAt)
	return next, changed, err
}

func (s *Store) updateAdvisory(ctx context.Context, previous Job, next Job, updatedAt time.Time) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET state=?,execution_id=?,cancel_requested=?,result_json=?,error=?,updated_at=? WHERE id=? AND operation=? AND repository_id=? AND state=? AND updated_at=?`, next.State, next.ExecutionID, next.CancelRequested, next.ResultJSON, next.Error, updatedAt.Format(time.RFC3339Nano), previous.ID, previous.Operation, previous.RepositoryID, previous.State, previous.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	n, e := res.RowsAffected()
	return n == 1, e
}

// RecordAdvisoryEvidence adds a bounded idempotent read receipt without changing
// the lifecycle revision. Agent tool reads can occur while Start owns that
// revision; touching UpdatedAt here would discard its execution acknowledgement.
func (s *Store) RecordAdvisoryEvidence(ctx context.Context, id, repo, execution string, check Check) error {
	for attempt := 0; attempt < 8; attempt++ {
		var raw string
		if err := s.db.QueryRowContext(ctx, `SELECT checks_json FROM gct_review_jobs WHERE id=? AND operation='staged-message' AND repository_id=? AND state IN ('running','cancelling') AND (execution_id='' OR execution_id=?)`, id, repo, execution).Scan(&raw); err != nil {
			return err
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
		if len(checks) >= 256 {
			return fmt.Errorf("evidence receipt limit reached")
		}
		checks = append(checks, check)
		data, err := json.Marshal(checks)
		if err != nil {
			return err
		}
		res, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET checks_json=? WHERE id=? AND operation='staged-message' AND repository_id=? AND checks_json=? AND state IN ('running','cancelling') AND (execution_id='' OR execution_id=?)`, string(data), id, repo, raw, execution)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
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
// caller's stale read. Dispatch must move queued->running before calling its
// owner, so a queued cancellation can safely finish without launching work.
func (s *Store) RequestAdvisoryCancel(ctx context.Context, id, repositoryID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET cancel_requested=1,state=CASE WHEN state='queued' AND execution_id='' THEN 'cancelled' ELSE 'cancelling' END,updated_at=? WHERE id=? AND operation='staged-message' AND repository_id=? AND state NOT IN ('succeeded','failed','cancelled')`, time.Now().UTC().Format(time.RFC3339Nano), id, repositoryID)
	return err
}
