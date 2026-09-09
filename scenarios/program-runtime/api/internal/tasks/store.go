// Package tasks owns execution checkpoints and delivery intent. Memory owns
// learning interpretation and Source Ledger owns the resulting memory entries.
package tasks

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

// EnsureCompatibility upgrades databases created before step-tree learning
// existed. SchemaProvider owns greenfield creation; this seam owns only the
// additive columns and indexes needed for an existing runtime database.
func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	columns := map[string]string{}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(learning_tasks)")
	if err != nil {
		return fmt.Errorf("inspect task schema: %w", err)
	}
	for rows.Next() {
		var cid, notNull, primary int
		var name, kind string
		var def any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &def, &primary); err != nil {
			rows.Close()
			return err
		}
		columns[name] = kind
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for name, definition := range map[string]string{"parent_attempt_id": "TEXT", "step_name": "TEXT", "step_path": "TEXT NOT NULL DEFAULT ''"} {
		if _, ok := columns[name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE learning_tasks ADD COLUMN "+name+" "+definition); err != nil {
			return fmt.Errorf("add learning_tasks.%s: %w", name, err)
		}
	}
	if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS learning_tasks_parent ON learning_tasks(parent_attempt_id)"); err != nil {
		return fmt.Errorf("index task parents: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS learning_tasks_task_path ON learning_tasks(task_id,attempt_number,step_path)"); err != nil {
		return fmt.Errorf("index task paths: %w", err)
	}
	return nil
}

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
	ParentAttemptID  string         `json:"parent_attempt_id,omitempty"`
	StepName         string         `json:"step_name,omitempty"`
	StepPath         string         `json:"step_path,omitempty"`
	DeliveryResult   map[string]any `json:"delivery_result,omitempty"`
	LastError        string         `json:"last_error,omitempty"`
	DeliveryAttempts int            `json:"delivery_attempts"`
	NextAttemptAt    string         `json:"next_attempt_at"`
}

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// Fragment is the durable cache record used by learn.act and the improve
// cycle. Trace fields are bounded JSON projections from a verified run.
type Fragment struct {
	CacheHit              bool           `json:"cache_hit,omitempty"`
	AttemptID             string         `json:"attempt_id,omitempty"`
	InputDigest           string         `json:"input_digest,omitempty"`
	Compatibility         map[string]any `json:"compatibility,omitempty"`
	Evidence              []string       `json:"evidence,omitempty"`
	Contexts              int            `json:"contexts"`
	LastVerifiedAt        string         `json:"last_verified_at"`
	StepKey               string         `json:"step_key"`
	FragmentHash          string         `json:"fragment_hash"`
	Fragment              string         `json:"fragment"`
	Verified              int            `json:"verified"`
	CachedRuns            int            `json:"cached_runs"`
	ContradictedSinceEdit int            `json:"contradicted_since_edit"`
	SourceProgramID       string         `json:"source_program_id"`
	Source                string         `json:"source"`
	StepName              string         `json:"step_name"`
	TraceInputs           map[string]any `json:"trace_inputs,omitempty"`
	TraceOutput           map[string]any `json:"trace_output,omitempty"`
	LastUsedAt            string         `json:"last_used_at"`
	CreatedAt             string         `json:"created_at"`
}

// GetFragment is an observation. Only verified execution records count as reuse.
func (s *Store) GetFragment(ctx context.Context, stepKey string) (*Fragment, error) {
	return s.BestFragment(ctx, stepKey)
}

// BestFragment returns the current verified fragment without recording a use.
// Promotion and setpoint reads use this view so observation does not alter
// delivery telemetry.
func (s *Store) BestFragment(ctx context.Context, stepKey string) (*Fragment, error) {
	stepKey = strings.TrimSpace(stepKey)
	if stepKey == "" || len(stepKey) > 512 {
		return nil, errors.New("bounded fragment step_key required")
	}
	return s.bestFragment(ctx, stepKey)
}

func (s *Store) bestFragment(ctx context.Context, stepKey string) (*Fragment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var f Fragment
	var inputs, output string
	err := s.db.QueryRowContext(ctx, `SELECT step_key,fragment_hash,fragment,verified,cached_runs,contradicted_since_edit,source_program_id,source,step_name,trace_inputs,trace_output,last_used_at,created_at FROM learning_fragments WHERE step_key=? AND verified>0 AND contradicted_since_edit=0 ORDER BY verified DESC,last_used_at DESC,created_at DESC LIMIT 1`, stepKey).Scan(&f.StepKey, &f.FragmentHash, &f.Fragment, &f.Verified, &f.CachedRuns, &f.ContradictedSinceEdit, &f.SourceProgramID, &f.Source, &f.StepName, &inputs, &output, &f.LastUsedAt, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(inputs), &f.TraceInputs)
	_ = json.Unmarshal([]byte(output), &f.TraceOutput)
	if err := s.fragmentEvidence(ctx, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) PutFragment(ctx context.Context, f Fragment, verifiedDelta, contradictedDelta int) (*Fragment, error) {
	if verifiedDelta < 0 || verifiedDelta > 1 || contradictedDelta < 0 || contradictedDelta > 1 {
		return nil, errors.New("fragment deltas must be 0 or 1")
	}
	if strings.HasPrefix(f.StepKey, "fragment-v2:") && (f.AttemptID == "" || len(f.AttemptID) > 128 || len(f.Compatibility) == 0 || (verifiedDelta > 0 && (f.InputDigest == "" || len(f.Evidence) == 0))) {
		return nil, errors.New("v2 fragment requires attempt identity, compatibility and verified evidence")
	}
	f.StepKey = strings.TrimSpace(f.StepKey)
	f.Fragment = strings.TrimSpace(f.Fragment)
	if f.StepKey == "" || len(f.StepKey) > 512 || f.Fragment == "" || len(f.Fragment) > 64*1024 {
		return nil, errors.New("bounded fragment step_key and source required")
	}
	hash := sha256.Sum256([]byte(f.Fragment))
	f.FragmentHash = fmt.Sprintf("sha256:%x", hash[:])
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if f.CreatedAt == "" {
		f.CreatedAt = now
	}
	inputs, _ := json.Marshal(f.TraceInputs)
	if len(inputs) == 0 {
		inputs = []byte("{}")
	}
	output, _ := json.Marshal(f.TraceOutput)
	if len(output) == 0 {
		output = []byte("{}")
	}
	if len(inputs) > 16*1024 || len(output) > 16*1024 {
		return nil, errors.New("fragment trace exceeds 16 KiB")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if f.CacheHit && verifiedDelta > 0 {
		f.CachedRuns = 1
	} else {
		f.CachedRuns = 0
	}
	if f.AttemptID != "" {
		compatibility, err := json.Marshal(f.Compatibility)
		if err != nil || len(compatibility) > 32768 {
			return nil, errors.New("bounded fragment compatibility required")
		}
		evidence, err := json.Marshal(f.Evidence)
		if err != nil || len(evidence) > 8192 {
			return nil, errors.New("bounded fragment evidence required")
		}
		outcome := "verified_success"
		if contradictedDelta > 0 {
			outcome = "failed"
		}
		result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO learning_fragment_evidence(step_key,fragment_hash,attempt_id,outcome,input_digest,compatibility,evidence,created_at) VALUES(?,?,?,?,?,?,?,?)`, f.StepKey, f.FragmentHash, f.AttemptID, outcome, f.InputDigest, string(compatibility), string(evidence), now)
		if err != nil {
			return nil, err
		}
		inserted, err := result.RowsAffected()
		if err != nil {
			return nil, err
		}
		if inserted == 0 {
			verifiedDelta = 0
			f.CachedRuns = 0
			contradictedDelta = 0
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_fragments(step_key,fragment_hash,fragment,verified,cached_runs,contradicted_since_edit,source_program_id,source,step_name,trace_inputs,trace_output,last_used_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(step_key,fragment_hash) DO UPDATE SET verified=learning_fragments.verified+excluded.verified,cached_runs=learning_fragments.cached_runs+excluded.cached_runs,contradicted_since_edit=learning_fragments.contradicted_since_edit+excluded.contradicted_since_edit,source_program_id=CASE WHEN excluded.source_program_id<>'' THEN excluded.source_program_id ELSE learning_fragments.source_program_id END,source=CASE WHEN excluded.source<>'' THEN excluded.source ELSE learning_fragments.source END,step_name=CASE WHEN excluded.step_name<>'' THEN excluded.step_name ELSE learning_fragments.step_name END,trace_inputs=CASE WHEN excluded.trace_inputs<>'{}' THEN excluded.trace_inputs ELSE learning_fragments.trace_inputs END,trace_output=CASE WHEN excluded.trace_output<>'{}' THEN excluded.trace_output ELSE learning_fragments.trace_output END,last_used_at=CASE WHEN excluded.last_used_at<>'' THEN excluded.last_used_at ELSE learning_fragments.last_used_at END`, f.StepKey, f.FragmentHash, f.Fragment, verifiedDelta, f.CachedRuns, contradictedDelta, f.SourceProgramID, f.Source, f.StepName, string(inputs), string(output), f.LastUsedAt, f.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.fragmentByHash(ctx, f.StepKey, f.FragmentHash)
}

func (s *Store) fragmentByHash(ctx context.Context, stepKey, hash string) (*Fragment, error) {
	var f Fragment
	var inputs, output string
	err := s.db.QueryRowContext(ctx, `SELECT step_key,fragment_hash,fragment,verified,cached_runs,contradicted_since_edit,source_program_id,source,step_name,trace_inputs,trace_output,last_used_at,created_at FROM learning_fragments WHERE step_key=? AND fragment_hash=?`, stepKey, hash).Scan(&f.StepKey, &f.FragmentHash, &f.Fragment, &f.Verified, &f.CachedRuns, &f.ContradictedSinceEdit, &f.SourceProgramID, &f.Source, &f.StepName, &inputs, &output, &f.LastUsedAt, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(inputs), &f.TraceInputs)
	_ = json.Unmarshal([]byte(output), &f.TraceOutput)
	if err := s.fragmentEvidence(ctx, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) ListFragments(ctx context.Context) ([]Fragment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT step_key,fragment_hash,fragment,verified,cached_runs,contradicted_since_edit,source_program_id,source,step_name,trace_inputs,trace_output,last_used_at,created_at FROM learning_fragments ORDER BY step_key,verified DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Fragment
	for rows.Next() {
		var f Fragment
		var inputs, output string
		if err := rows.Scan(&f.StepKey, &f.FragmentHash, &f.Fragment, &f.Verified, &f.CachedRuns, &f.ContradictedSinceEdit, &f.SourceProgramID, &f.Source, &f.StepName, &inputs, &output, &f.LastUsedAt, &f.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(inputs), &f.TraceInputs)
		_ = json.Unmarshal([]byte(output), &f.TraceOutput)
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	for i := range out {
		if err := s.fragmentEvidence(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Store) AgedBlocked(ctx context.Context, before time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT record FROM learning_tasks WHERE delivery='blocked'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			return 0, err
		}
		var record Record
		if err := json.Unmarshal([]byte(body), &record); err != nil {
			return 0, err
		}
		stamp, err := time.Parse(time.RFC3339Nano, record.FinishedAt)
		if err == nil && stamp.Before(before) {
			count++
		}
	}
	return count, rows.Err()
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
	err = tx.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE task_id=? AND step_path=? ORDER BY attempt_number DESC LIMIT 1", r.TaskID, r.StepPath).Scan(&lastBody)
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
	_, err = tx.ExecContext(ctx, "INSERT INTO learning_tasks (attempt_id,task_id,attempt_number,session_id,program_id,state,delivery,next_attempt_at,record,parent_attempt_id,step_name,step_path) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", r.AttemptID, r.TaskID, r.AttemptNumber, r.SessionID, r.ProgramID, r.State, r.Delivery, r.NextAttemptAt, body, r.ParentAttemptID, r.StepName, r.StepPath)
	if err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	return &r, true, nil
}

// InsertChildren persists step attempts as members of an already completed
// root checkpoint. The root remains the only delivery/outbox row; child rows
// are ledger detail and therefore are marked delivered with the parent's
// immutable finish intent.
func (s *Store) InsertChildren(ctx context.Context, parent *Record, nodes []map[string]any) error {
	if parent == nil || len(nodes) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for index, node := range nodes {
		id, _ := node["attempt_id"].(string)
		if id == "" {
			return errors.New("child attempt_id is required")
		}
		number, ok := node["attempt_number"].(float64)
		if !ok || int(number) != parent.AttemptNumber {
			return errors.New("child attempt ordinal must match parent")
		}
		stepName, _ := node["step_name"].(string)
		contextKey, _ := node["context_key"].(string)
		started, _ := node["started_at"].(string)
		finished, _ := node["finished_at"].(string)
		taskStarted, _ := node["task_started_at"].(string)
		operation, _ := node["operation"].(string)
		outcome, _ := node["outcome"].(string)
		provenance, _ := node["provenance"].(string)
		child := Record{ResumeHash: parent.ResumeHash, InputHash: parent.InputHash, TaskID: parent.TaskID, AttemptID: id, AttemptNumber: parent.AttemptNumber, SessionID: parent.SessionID, ProgramID: parent.ProgramID, Operation: operation, Digest: parent.Digest, Scope: parent.Scope, ContextKey: contextKey, Provenance: provenance, TaskStartedAt: taskStarted, StartedAt: started, FinishedAt: finished, State: "completed", Delivery: "delivered", Outcome: outcome, FinishDigest: parent.FinishDigest, ParentAttemptID: parent.AttemptID, StepName: stepName, StepPath: fmt.Sprintf("%s:%d", stepName, index), NextAttemptAt: finished}
		body, err := encode(child)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT OR IGNORE INTO learning_tasks (attempt_id,task_id,attempt_number,session_id,program_id,state,delivery,next_attempt_at,record,parent_attempt_id,step_name,step_path) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)", child.AttemptID, child.TaskID, child.AttemptNumber, child.SessionID, child.ProgramID, child.State, child.Delivery, child.NextAttemptAt, body, child.ParentAttemptID, child.StepName, child.StepPath)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
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

// fragmentEvidence separates input diversity and verification freshness from cache reads.
func (s *Store) fragmentEvidence(ctx context.Context, f *Fragment) error {
	var compatibility string
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT CASE WHEN outcome='verified_success' THEN input_digest END), COALESCE(MAX(CASE WHEN outcome='verified_success' THEN created_at END),''), COALESCE(MAX(compatibility),'{}') FROM learning_fragment_evidence WHERE step_key=? AND fragment_hash=?`, f.StepKey, f.FragmentHash).Scan(&f.Contexts, &f.LastVerifiedAt, &compatibility)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(compatibility), &f.Compatibility)
}
