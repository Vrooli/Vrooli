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
	ID             string    `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	InputDigest    string    `json:"input_digest"`
	PolicyVersion  string    `json:"policy_version"`
	State          State     `json:"state"`
	Checks         []Check   `json:"checks"`
	ResultRef      string    `json:"result_ref,omitempty"`
	ResultJSON     string    `json:"result_json,omitempty"`
	Error          string    `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ScenarioName   string    `json:"scenario_name,omitempty"`
	DetailCount    int       `json:"detail_count,omitempty"`
	ThresholdsJSON string    `json:"thresholds_json,omitempty"`
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
	for _, column := range []string{"scenario_name TEXT NOT NULL DEFAULT ''", "detail_count INTEGER NOT NULL DEFAULT 0", "thresholds_json TEXT NOT NULL DEFAULT ''", "result_json TEXT NOT NULL DEFAULT ''"} {
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
	result, err := s.db.ExecContext(ctx, `INSERT INTO gct_review_jobs (id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, job.ID, job.IdempotencyKey, job.InputDigest, job.PolicyVersion, job.State, string(data), job.ResultRef, job.ResultJSON, job.Error, job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano), job.ScenarioName, job.DetailCount, job.ThresholdsJSON)
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
	err := s.db.QueryRowContext(ctx, `SELECT id,idempotency_key,input_digest,policy_version,state,checks_json,result_ref,result_json,error,created_at,updated_at,scenario_name,detail_count,thresholds_json FROM gct_review_jobs `+clause, arg).Scan(&j.ID, &j.IdempotencyKey, &j.InputDigest, &j.PolicyVersion, &j.State, &checks, &j.ResultRef, &j.ResultJSON, &j.Error, &created, &updated, &j.ScenarioName, &j.DetailCount, &j.ThresholdsJSON)
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
