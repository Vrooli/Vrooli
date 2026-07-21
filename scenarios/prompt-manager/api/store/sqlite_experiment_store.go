package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteExperimentStore is the durable evidence store for skill experiments.
// The JSON payload keeps experiment evolution additive while event rows have a
// database identity and idempotency constraint instead of append-only logs.
type SQLiteExperimentStore struct{ db *sql.DB }

func NewSQLiteExperimentStore(runtimeDataDir string) (*SQLiteExperimentStore, error) {
	db, err := sql.Open("sqlite", filepath.Join(runtimeDataDir, "experiments.sqlite"))
	if err != nil {
		return nil, fmt.Errorf("open experiment sqlite: %w", err)
	}
	s := &SQLiteExperimentStore{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteExperimentStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS experiments (id TEXT PRIMARY KEY, skill_id TEXT NOT NULL, payload BLOB NOT NULL);
CREATE INDEX IF NOT EXISTS experiments_skill_idx ON experiments(skill_id);
CREATE TABLE IF NOT EXISTS experiment_outcomes (
  id INTEGER PRIMARY KEY, experiment_id TEXT NOT NULL, variant_id TEXT NOT NULL,
  source TEXT NOT NULL, schema_version INTEGER NOT NULL, recorded_at TEXT NOT NULL,
  data BLOB NOT NULL, idempotency_key TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS experiment_serves (
  id INTEGER PRIMARY KEY, experiment_id TEXT NOT NULL, skill_id TEXT NOT NULL,
  variant_id TEXT NOT NULL, source TEXT NOT NULL, served_at TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS experiment_assignments (
  id INTEGER PRIMARY KEY, experiment_id TEXT NOT NULL, skill_id TEXT NOT NULL,
  variant_id TEXT NOT NULL, execution_id TEXT NOT NULL, node_id TEXT NOT NULL,
  attempt_key TEXT NOT NULL, idempotency_key TEXT NOT NULL UNIQUE,
  content TEXT NOT NULL, content_hash TEXT NOT NULL, assigned_at TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS experiment_exposures (
  id INTEGER PRIMARY KEY, experiment_id TEXT NOT NULL, variant_id TEXT NOT NULL,
  run_id TEXT NOT NULL, execution_id TEXT NOT NULL, node_id TEXT NOT NULL,
  attempt_id TEXT NOT NULL, read_skill_id TEXT NOT NULL, provenance TEXT NOT NULL,
  observed_at TEXT NOT NULL, idempotency_key TEXT NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS controlled_experiment_outcomes (
  id INTEGER PRIMARY KEY, experiment_id TEXT NOT NULL, variant_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE, assignment_id TEXT NOT NULL,
  execution_id TEXT NOT NULL, evaluator_attempt_id TEXT NOT NULL,
  evaluator_run_id TEXT NOT NULL, verdict TEXT NOT NULL, success INTEGER,
  outcome_status TEXT NOT NULL, rubric_hash TEXT NOT NULL,
  evaluator_prompt_hash TEXT NOT NULL, structured_schema_hash TEXT NOT NULL,
  evidence_hash TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS experiment_audit_receipts (
  experiment_id TEXT PRIMARY KEY, protocol_hash TEXT NOT NULL,
  sampled_assignment_ids BLOB NOT NULL, findings_hash TEXT NOT NULL,
  challenge_state TEXT NOT NULL, anomaly_count INTEGER NOT NULL,
  gaming_count INTEGER NOT NULL, completed_at TEXT NOT NULL,
  signature TEXT NOT NULL, idempotency_key TEXT NOT NULL UNIQUE);
`)
	return err
}

func (s *SQLiteExperimentStore) RecordAuditReceipt(ctx context.Context, receipt ExperimentAuditReceipt) error {
	if receipt.CompletedAt == "" {
		receipt.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	}
	samples, err := json.Marshal(receipt.SampledAssignmentIDs)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO experiment_audit_receipts(experiment_id,protocol_hash,sampled_assignment_ids,findings_hash,challenge_state,anomaly_count,gaming_count,completed_at,signature,idempotency_key) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(experiment_id) DO NOTHING`, receipt.ExperimentID, receipt.ProtocolHash, samples, receipt.FindingsHash, receipt.ChallengeState, receipt.AnomalyCount, receipt.GamingCount, receipt.CompletedAt, receipt.Signature, receipt.IdempotencyKey)
	return err
}
func (s *SQLiteExperimentStore) GetAuditReceipt(ctx context.Context, experimentID string) (*ExperimentAuditReceipt, error) {
	var receipt ExperimentAuditReceipt
	var samples []byte
	err := s.db.QueryRowContext(ctx, `SELECT experiment_id,protocol_hash,sampled_assignment_ids,findings_hash,challenge_state,anomaly_count,gaming_count,completed_at,signature,idempotency_key FROM experiment_audit_receipts WHERE experiment_id=?`, experimentID).Scan(&receipt.ExperimentID, &receipt.ProtocolHash, &samples, &receipt.FindingsHash, &receipt.ChallengeState, &receipt.AnomalyCount, &receipt.GamingCount, &receipt.CompletedAt, &receipt.Signature, &receipt.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(samples, &receipt.SampledAssignmentIDs); err != nil {
		return nil, err
	}
	return &receipt, nil
}

func (s *SQLiteExperimentStore) GetAssignment(ctx context.Context, experimentID, key string) (*ExperimentAssignment, error) {
	var a ExperimentAssignment
	err := s.db.QueryRowContext(ctx, `SELECT experiment_id,skill_id,variant_id,execution_id,node_id,attempt_key,idempotency_key,content,content_hash,assigned_at FROM experiment_assignments WHERE experiment_id=? AND idempotency_key=?`, experimentID, key).Scan(&a.ExperimentID, &a.SkillID, &a.VariantID, &a.ExecutionID, &a.NodeID, &a.AttemptKey, &a.IdempotencyKey, &a.Content, &a.ContentHash, &a.AssignedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *SQLiteExperimentStore) CreateAssignment(ctx context.Context, a ExperimentAssignment) error {
	if a.AssignedAt == "" {
		a.AssignedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO experiment_assignments(experiment_id,skill_id,variant_id,execution_id,node_id,attempt_key,idempotency_key,content,content_hash,assigned_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, a.ExperimentID, a.SkillID, a.VariantID, a.ExecutionID, a.NodeID, a.AttemptKey, a.IdempotencyKey, a.Content, a.ContentHash, a.AssignedAt)
	return err
}

func (s *SQLiteExperimentStore) ListAssignments(ctx context.Context, experimentID string) ([]ExperimentAssignment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT experiment_id,skill_id,variant_id,execution_id,node_id,attempt_key,idempotency_key,content,content_hash,assigned_at FROM experiment_assignments WHERE experiment_id=? ORDER BY id`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ExperimentAssignment
	for rows.Next() {
		var a ExperimentAssignment
		if err := rows.Scan(&a.ExperimentID, &a.SkillID, &a.VariantID, &a.ExecutionID, &a.NodeID, &a.AttemptKey, &a.IdempotencyKey, &a.Content, &a.ContentHash, &a.AssignedAt); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *SQLiteExperimentStore) RecordExposure(ctx context.Context, e ExperimentExposure) error {
	if e.ObservedAt == "" {
		e.ObservedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO experiment_exposures(experiment_id,variant_id,run_id,execution_id,node_id,attempt_id,read_skill_id,provenance,observed_at,idempotency_key) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, e.ExperimentID, e.VariantID, e.RunID, e.ExecutionID, e.NodeID, e.AttemptID, e.ReadSkillID, e.Provenance, e.ObservedAt, e.IdempotencyKey)
	return err
}

func (s *SQLiteExperimentStore) ListExposures(ctx context.Context, experimentID string) ([]ExperimentExposure, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT experiment_id,variant_id,run_id,execution_id,node_id,attempt_id,read_skill_id,provenance,observed_at,idempotency_key FROM experiment_exposures WHERE experiment_id=? ORDER BY id`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ExperimentExposure
	for rows.Next() {
		var e ExperimentExposure
		if err := rows.Scan(&e.ExperimentID, &e.VariantID, &e.RunID, &e.ExecutionID, &e.NodeID, &e.AttemptID, &e.ReadSkillID, &e.Provenance, &e.ObservedAt, &e.IdempotencyKey); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *SQLiteExperimentStore) List(ctx context.Context) ([]Experiment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM experiments ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Experiment
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var e Experiment
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, fmt.Errorf("decode experiment: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
func (s *SQLiteExperimentStore) ListBySkill(ctx context.Context, skillID string) ([]Experiment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM experiments WHERE skill_id=? ORDER BY id`, skillID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []Experiment
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var e Experiment
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
func (s *SQLiteExperimentStore) Get(ctx context.Context, id string) (*Experiment, error) {
	var payload []byte
	if err := s.db.QueryRowContext(ctx, `SELECT payload FROM experiments WHERE id=?`, id).Scan(&payload); err != nil {
		return nil, fmt.Errorf("experiment not found: %s", id)
	}
	var e Experiment
	if err := json.Unmarshal(payload, &e); err != nil {
		return nil, fmt.Errorf("decode experiment: %w", err)
	}
	return &e, nil
}
func (s *SQLiteExperimentStore) Create(ctx context.Context, e *Experiment) error {
	e.Kind = KindExperiment
	e.SchemaVersion = CurrentSchemaVersion
	if e.Status == "" {
		e.Status = ExperimentStatusDraft
	}
	e.Timestamps = NewTimestamps()
	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO experiments(id,skill_id,payload) VALUES(?,?,?)`, e.ID, e.SkillID, payload); err != nil {
		return fmt.Errorf("create experiment: %w", err)
	}
	return nil
}
func (s *SQLiteExperimentStore) Update(ctx context.Context, id string, e *Experiment) error {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	e.Kind = existing.Kind
	e.SchemaVersion = existing.SchemaVersion
	e.ID = existing.ID
	e.Timestamps = existing.Timestamps
	e.UpdateTimestamp()
	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}
	r, err := s.db.ExecContext(ctx, `UPDATE experiments SET skill_id=?, payload=? WHERE id=?`, e.SkillID, payload, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return fmt.Errorf("experiment not found: %s", id)
	}
	return nil
}
func (s *SQLiteExperimentStore) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, `DELETE FROM experiments WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return fmt.Errorf("experiment not found: %s", id)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM experiment_outcomes WHERE experiment_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM experiment_serves WHERE experiment_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM experiment_assignments WHERE experiment_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM experiment_exposures WHERE experiment_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM controlled_experiment_outcomes WHERE experiment_id=?`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM experiment_audit_receipts WHERE experiment_id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *SQLiteExperimentStore) RecordOutcome(ctx context.Context, id string, o ExperimentOutcome) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if o.RecordedAt == "" {
		o.RecordedAt = time.Now().UTC().Format(time.RFC3339)
	}
	key := o.IdempotencyKey
	if key == "" {
		key = outcomeKey(id, o)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO experiment_outcomes(experiment_id,variant_id,source,schema_version,recorded_at,data,idempotency_key) VALUES(?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, id, o.VariantID, o.Source, o.SchemaVersion, o.RecordedAt, []byte(o.Data), key); err != nil {
		return err
	}
	if o.Controlled != nil {
		c := o.Controlled
		var success any
		if c.Success != nil {
			success = *c.Success
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO controlled_experiment_outcomes(experiment_id,variant_id,idempotency_key,assignment_id,execution_id,evaluator_attempt_id,evaluator_run_id,verdict,success,outcome_status,rubric_hash,evaluator_prompt_hash,structured_schema_hash,evidence_hash) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, id, o.VariantID, key, c.AssignmentID, c.ExecutionID, c.EvaluatorAttemptID, c.EvaluatorRunID, c.Verdict, success, c.OutcomeStatus, c.RubricHash, c.EvaluatorPromptHash, c.StructuredSchemaHash, c.EvidenceHash); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *SQLiteExperimentStore) RecordServe(ctx context.Context, e ExperimentServe) error {
	if _, err := s.Get(ctx, e.ExperimentID); err != nil {
		return err
	}
	if e.ServedAt == "" {
		e.ServedAt = time.Now().UTC().Format(time.RFC3339)
	}
	key := serveKey(e)
	_, err := s.db.ExecContext(ctx, `INSERT INTO experiment_serves(experiment_id,skill_id,variant_id,source,served_at,idempotency_key) VALUES(?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, e.ExperimentID, e.SkillID, e.VariantID, e.Source, e.ServedAt, key)
	return err
}
func (s *SQLiteExperimentStore) ListServes(ctx context.Context, id string) ([]ExperimentServe, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT experiment_id,skill_id,variant_id,source,served_at FROM experiment_serves WHERE experiment_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ExperimentServe
	for rows.Next() {
		var e ExperimentServe
		if err := rows.Scan(&e.ExperimentID, &e.SkillID, &e.VariantID, &e.Source, &e.ServedAt); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
func (s *SQLiteExperimentStore) CountServesByVariant(ctx context.Context, id string) (map[string]int, error) {
	rows, err := s.ListServes(ctx, id)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.VariantID]++
	}
	return out, nil
}
func (s *SQLiteExperimentStore) ListOutcomes(ctx context.Context, id string) ([]ExperimentOutcome, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT o.variant_id,o.source,o.schema_version,o.recorded_at,o.data,o.idempotency_key,COALESCE(c.assignment_id,''),COALESCE(c.execution_id,''),COALESCE(c.evaluator_attempt_id,''),COALESCE(c.evaluator_run_id,''),COALESCE(c.verdict,''),c.success,COALESCE(c.outcome_status,''),COALESCE(c.rubric_hash,''),COALESCE(c.evaluator_prompt_hash,''),COALESCE(c.structured_schema_hash,''),COALESCE(c.evidence_hash,'') FROM experiment_outcomes o LEFT JOIN controlled_experiment_outcomes c ON c.idempotency_key=o.idempotency_key WHERE o.experiment_id=? ORDER BY o.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ExperimentOutcome
	for rows.Next() {
		var o ExperimentOutcome
		var data []byte
		var c ControlledExperimentOutcome
		var success sql.NullBool
		var assignmentID string
		if err := rows.Scan(&o.VariantID, &o.Source, &o.SchemaVersion, &o.RecordedAt, &data, &o.IdempotencyKey, &assignmentID, &c.ExecutionID, &c.EvaluatorAttemptID, &c.EvaluatorRunID, &c.Verdict, &success, &c.OutcomeStatus, &c.RubricHash, &c.EvaluatorPromptHash, &c.StructuredSchemaHash, &c.EvidenceHash); err != nil {
			return nil, err
		}
		o.Data = json.RawMessage(data)
		if assignmentID != "" {
			c.AssignmentID = assignmentID
			if success.Valid {
				value := success.Bool
				c.Success = &value
			}
			o.Controlled = &c
		}
		result = append(result, o)
	}
	return result, rows.Err()
}
func (s *SQLiteExperimentStore) CountOutcomesByVariant(ctx context.Context, id string) (map[string]int, error) {
	rows, err := s.ListOutcomes(ctx, id)
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, r := range rows {
		out[r.VariantID]++
	}
	return out, nil
}
func outcomeKey(id string, o ExperimentOutcome) string {
	return fmt.Sprintf("%s|%s|%s|%d|%s|%s", id, o.VariantID, o.Source, o.SchemaVersion, o.RecordedAt, string(o.Data))
}
func serveKey(e ExperimentServe) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", e.ExperimentID, e.SkillID, e.VariantID, e.Source, e.ServedAt)
}
