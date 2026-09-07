// Package tasks owns execution checkpoints and delivery intent. Memory owns
// learning interpretation and Source Ledger owns the resulting memory entries.
package tasks

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

// Record is a bounded execution receipt, not a second memory journal.
type Record struct {
	ResumeHash       string         `json:"resume_hash"`
	InputHash        string         `json:"input_hash"`
	TaskID           string         `json:"task_id"`
	AttemptID        string         `json:"attempt_id"`
	AttemptNumber    int            `json:"attempt_number"`
	SessionID        string         `json:"session_id"`
	ProgramID        string         `json:"program_id"`
	Operation        string         `json:"operation"`
	Digest           string         `json:"digest"`
	Scope            string         `json:"scope"`
	ContextKey       string         `json:"context_key"`
	Provenance       string         `json:"provenance"`
	TaskStartedAt    string         `json:"task_started_at"`
	StartedAt        string         `json:"started_at"`
	FinishedAt       string         `json:"finished_at,omitempty"`
	State            string         `json:"state"`
	Delivery         string         `json:"delivery"`
	Outcome          string         `json:"outcome"`
	Prepare          map[string]any `json:"prepare,omitempty"`
	Result           map[string]any `json:"result,omitempty"`
	FinishInputs     map[string]any `json:"finish_inputs,omitempty"`
	FinishDigest     string         `json:"finish_digest"`
	DeliveryResult   map[string]any `json:"delivery_result,omitempty"`
	LastError        string         `json:"last_error,omitempty"`
	DeliveryAttempts int            `json:"delivery_attempts"`
	NextAttemptAt    string         `json:"next_attempt_at"`
}

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }
func (s *Store) Get(ctx context.Context, id string) (*Record, error) {
	var body string
	if err := s.db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id=?", id).Scan(&body); err != nil {
		return nil, err
	}
	var r Record
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Begin atomically allocates the ordinal. Reusing an attempt ID is inspection,
// never permission to dispatch its domain operation again.
func (s *Store) Begin(ctx context.Context, r Record) (*Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, err := s.Get(ctx, r.AttemptID); err == nil {
		if old.ResumeHash != r.ResumeHash || old.TaskID != r.TaskID || old.Operation != r.Operation || old.Digest != r.Digest || old.InputHash != r.InputHash || old.ContextKey != r.ContextKey || old.Scope != r.Scope || old.Provenance != r.Provenance {
			return nil, false, errors.New("attempt identity conflict or invalid resume receipt")
		}
		return old, false, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	var lastBody string
	err = tx.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE task_id=? ORDER BY attempt_number DESC LIMIT 1", r.TaskID).Scan(&lastBody)
	r.AttemptNumber = 1
	r.TaskStartedAt = r.StartedAt
	if err == nil {
		var last Record
		if err = json.Unmarshal([]byte(lastBody), &last); err != nil {
			return nil, false, err
		}
		if last.Scope != r.Scope || last.ContextKey != r.ContextKey || last.Operation != r.Operation || last.Provenance != r.Provenance || last.ResumeHash != r.ResumeHash {
			return nil, false, errors.New("task identity belongs to a different context or invalid resume receipt")
		}
		if last.State != "completed" {
			return nil, false, errors.New("previous attempt requires inspection; uncertain work is never replayed")
		}
		r.AttemptNumber = last.AttemptNumber + 1
		r.TaskStartedAt = last.TaskStartedAt
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	r.State = "prepared"
	r.Delivery = "waiting"
	r.Outcome = "unknown"
	r.NextAttemptAt = r.StartedAt
	body, err := encode(r)
	if err != nil {
		return nil, false, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO learning_tasks (attempt_id,task_id,attempt_number,session_id,program_id,state,delivery,next_attempt_at,record) VALUES(?,?,?,?,?,?,?,?,?)", r.AttemptID, r.TaskID, r.AttemptNumber, r.SessionID, r.ProgramID, r.State, r.Delivery, r.NextAttemptAt, body)
	if err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return &r, true, nil
}

func encode(r Record) (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	if len(b) > 256*1024 {
		return "", errors.New("task receipt exceeds 256 KiB")
	}
	return string(b), nil
}
func (s *Store) Update(ctx context.Context, id string, mutate func(*Record) error) (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = mutate(r); err != nil {
		return nil, err
	}
	body, err := encode(*r)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, "UPDATE learning_tasks SET state=?,delivery=?,next_attempt_at=?,record=? WHERE attempt_id=?", r.State, r.Delivery, r.NextAttemptAt, body, id)
	return r, err
}
func (s *Store) Pending(ctx context.Context, now time.Time) ([]Record, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT record FROM learning_tasks WHERE delivery='pending' AND next_attempt_at<=? ORDER BY next_attempt_at LIMIT 20", now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var b string
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		var r Record
		if err = json.Unmarshal([]byte(b), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Recover never dispatches domain code. A missing completion checkpoint means
// effects are unknown, even when the containing program printed success.
func (s *Store) Recover(ctx context.Context, programID string) error {
	query := "SELECT attempt_id FROM learning_tasks WHERE state IN ('prepared','running')"
	var args []any
	if programID != "" {
		query += " AND program_id=?"
		args = append(args, programID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		_, err = s.Update(ctx, id, func(r *Record) error {
			if r.State == "prepared" || r.State == "running" {
				r.State = "uncertain"
				r.Outcome = "unknown"
				r.LastError = "runtime interrupted before durable domain completion; inspect effects before a new task"
				r.Delivery = "blocked"
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("recover task %s: %w", id, err)
		}
	}
	return nil
}
