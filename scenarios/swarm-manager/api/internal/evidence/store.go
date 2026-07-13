package evidence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrSourceConflict = errors.New("evidence source observation conflicts with immutable content")

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) InitSchema(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("evidence store database is required")
	}
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS evidence_observations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_system TEXT NOT NULL,
			source_event_id TEXT NOT NULL,
			run_id TEXT NOT NULL,
			subject_kind TEXT NOT NULL,
			subject_id TEXT NOT NULL,
			action TEXT NOT NULL,
			confidence TEXT NOT NULL,
			verification TEXT NOT NULL,
			content_digest TEXT NOT NULL DEFAULT '',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			observed_at TEXT NOT NULL,
			ownership_status TEXT NOT NULL,
			UNIQUE(source_system, source_event_id, subject_kind, subject_id, action)
		);
		CREATE TABLE IF NOT EXISTS evidence_links (
			observation_id INTEGER NOT NULL REFERENCES evidence_observations(id) ON DELETE RESTRICT,
			owner_kind TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			owner_round INTEGER NOT NULL DEFAULT 0,
			linked_at TEXT NOT NULL,
			PRIMARY KEY(observation_id, owner_kind, owner_id, owner_round)
		);
		CREATE INDEX IF NOT EXISTS idx_evidence_observations_run ON evidence_observations(run_id);
		CREATE INDEX IF NOT EXISTS idx_evidence_observations_subject ON evidence_observations(subject_kind, subject_id);
		CREATE INDEX IF NOT EXISTS idx_evidence_links_owner ON evidence_links(owner_kind, owner_id, owner_round);
		CREATE TABLE IF NOT EXISTS evidence_checkpoints (
			producer_id TEXT NOT NULL,
			run_id TEXT NOT NULL,
			fact_kind TEXT NOT NULL,
			cursor TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(producer_id, run_id, fact_kind)
		);
		CREATE TABLE IF NOT EXISTS evidence_watermarks (
			producer_id TEXT NOT NULL,
			run_id TEXT NOT NULL,
			fact_kind TEXT NOT NULL,
			coverage TEXT NOT NULL,
			completed_at TEXT NOT NULL,
			PRIMARY KEY(producer_id, run_id, fact_kind)
		);
		CREATE TABLE IF NOT EXISTS evidence_migration_audits (
			migration_key TEXT PRIMARY KEY,
			source_count INTEGER NOT NULL,
			projected_count INTEGER NOT NULL,
			source_digest TEXT NOT NULL,
			projected_digest TEXT NOT NULL,
			completed_at TEXT NOT NULL
		);
	`)
	return err
}

func (s *Store) SaveMigrationAudit(ctx context.Context, audit MigrationAudit) error {
	if strings.TrimSpace(audit.MigrationKey) == "" || audit.SourceCount < 0 || audit.ProjectedCount < 0 || strings.TrimSpace(audit.SourceDigest) == "" || strings.TrimSpace(audit.ProjectedDigest) == "" {
		return fmt.Errorf("migration audit key, counts, and digests are required")
	}
	if audit.CompletedAt.IsZero() {
		audit.CompletedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO evidence_migration_audits (migration_key, source_count, projected_count, source_digest, projected_digest, completed_at) VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(migration_key) DO UPDATE SET source_count=excluded.source_count, projected_count=excluded.projected_count, source_digest=excluded.source_digest, projected_digest=excluded.projected_digest, completed_at=excluded.completed_at`, audit.MigrationKey, audit.SourceCount, audit.ProjectedCount, audit.SourceDigest, audit.ProjectedDigest, audit.CompletedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save evidence migration audit: %w", err)
	}
	return nil
}

func (s *Store) UpsertObservation(ctx context.Context, observation Observation, status OwnershipStatus) (int64, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, fmt.Errorf("begin evidence observation: %w", err)
	}
	defer tx.Rollback()
	id, duplicate, err := s.upsertObservationTx(ctx, tx, observation, status)
	if err != nil {
		return 0, false, err
	}
	if err := tx.Commit(); err != nil {
		return 0, false, fmt.Errorf("commit evidence observation: %w", err)
	}
	return id, duplicate, nil
}

func (s *Store) upsertObservationTx(ctx context.Context, tx *sql.Tx, observation Observation, status OwnershipStatus) (int64, bool, error) {
	observation = observation.normalized()
	if err := observation.Validate(); err != nil {
		return 0, false, err
	}
	metadata, err := json.Marshal(observation.Metadata)
	if err != nil {
		return 0, false, fmt.Errorf("marshal evidence metadata: %w", err)
	}
	var id int64
	var digest, confidence, verification, runID string
	err = tx.QueryRowContext(ctx, `SELECT id, content_digest, confidence, verification, run_id FROM evidence_observations WHERE source_system=? AND source_event_id=? AND subject_kind=? AND subject_id=? AND action=?`, observation.SourceSystem, observation.SourceEventID, observation.Subject.Kind, observation.Subject.ID, observation.Action).Scan(&id, &digest, &confidence, &verification, &runID)
	if err == nil {
		if digest != observation.ContentDigest || confidence != string(observation.Confidence) || verification != string(observation.Verification) || runID != observation.RunID {
			return 0, false, ErrSourceConflict
		}
		if _, err := tx.ExecContext(ctx, `UPDATE evidence_observations SET ownership_status=? WHERE id=?`, string(status), id); err != nil {
			return 0, false, fmt.Errorf("update evidence ownership status: %w", err)
		}
		return id, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, fmt.Errorf("look up evidence observation: %w", err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO evidence_observations (source_system, source_event_id, run_id, subject_kind, subject_id, action, confidence, verification, content_digest, metadata_json, observed_at, ownership_status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, observation.SourceSystem, observation.SourceEventID, observation.RunID, observation.Subject.Kind, observation.Subject.ID, observation.Action, observation.Confidence, observation.Verification, observation.ContentDigest, string(metadata), observation.ObservedAt.Format(time.RFC3339Nano), status)
	if err != nil {
		return 0, false, fmt.Errorf("insert evidence observation: %w", err)
	}
	id, err = result.LastInsertId()
	if err != nil {
		return 0, false, fmt.Errorf("read evidence observation id: %w", err)
	}
	return id, false, nil
}

// IngestForOwnerBatch stores and links an explicit-owner batch atomically.
// It validates every record before opening the transaction, so a malformed
// record cannot leave earlier records in the batch visible on its own.
func (s *Store) IngestForOwnerBatch(ctx context.Context, owner Owner, observations []Observation) ([]IngestResult, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	for index, observation := range observations {
		if err := observation.normalized().Validate(); err != nil {
			return nil, fmt.Errorf("evidence observations[%d]: %w", index, err)
		}
	}
	if len(observations) == 0 {
		return []IngestResult{}, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin evidence batch: %w", err)
	}
	defer tx.Rollback()
	results := make([]IngestResult, 0, len(observations))
	for _, observation := range observations {
		id, duplicate, err := s.upsertObservationTx(ctx, tx, observation, OwnershipResolved)
		if err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO evidence_links (observation_id, owner_kind, owner_id, owner_round, linked_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(observation_id, owner_kind, owner_id, owner_round) DO NOTHING`, id, owner.Kind, owner.ID, owner.Round, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return nil, fmt.Errorf("link evidence observation: %w", err)
		}
		ownerCopy := owner
		results = append(results, IngestResult{ObservationID: id, Owner: &ownerCopy, OwnershipStatus: OwnershipResolved, Duplicate: duplicate})
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit evidence batch: %w", err)
	}
	return results, nil
}

func (s *Store) Link(ctx context.Context, observationID int64, owner Owner) error {
	if observationID <= 0 {
		return fmt.Errorf("evidence observation id is required")
	}
	if err := owner.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO evidence_links (observation_id, owner_kind, owner_id, owner_round, linked_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(observation_id, owner_kind, owner_id, owner_round) DO NOTHING`, observationID, owner.Kind, owner.ID, owner.Round, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("link evidence observation: %w", err)
	}
	return nil
}

func (s *Store) ListByOwner(ctx context.Context, owner Owner) ([]Record, error) {
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	return s.listRecords(ctx, `SELECT o.source_system, o.source_event_id, o.run_id, o.subject_kind, o.subject_id, o.action, o.confidence, o.verification, o.content_digest, o.metadata_json, o.observed_at, l.linked_at, l.owner_kind, l.owner_id, l.owner_round FROM evidence_links l JOIN evidence_observations o ON o.id=l.observation_id WHERE l.owner_kind=? AND l.owner_id=? AND l.owner_round=? ORDER BY o.observed_at, o.id`, owner.Kind, owner.ID, owner.Round)
}

func (s *Store) listRecords(ctx context.Context, query string, args ...any) ([]Record, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list evidence by owner: %w", err)
	}
	defer rows.Close()
	records := []Record{}
	for rows.Next() {
		var record Record
		var metadata, observedAt, linkedAt string
		if err := rows.Scan(&record.Observation.SourceSystem, &record.Observation.SourceEventID, &record.Observation.RunID, &record.Observation.Subject.Kind, &record.Observation.Subject.ID, &record.Observation.Action, &record.Observation.Confidence, &record.Observation.Verification, &record.Observation.ContentDigest, &metadata, &observedAt, &linkedAt, &record.Owner.Kind, &record.Owner.ID, &record.Owner.Round); err != nil {
			return nil, fmt.Errorf("scan evidence record: %w", err)
		}
		if err := json.Unmarshal([]byte(metadata), &record.Observation.Metadata); err != nil {
			return nil, fmt.Errorf("decode evidence metadata: %w", err)
		}
		var err error
		record.Observation.ObservedAt, err = time.Parse(time.RFC3339Nano, observedAt)
		if err != nil {
			return nil, fmt.Errorf("parse evidence observed time: %w", err)
		}
		record.LinkedAt, err = time.Parse(time.RFC3339Nano, linkedAt)
		if err != nil {
			return nil, fmt.Errorf("parse evidence linked time: %w", err)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) ListByOwnerID(ctx context.Context, kind OwnerKind, id string) ([]Record, error) {
	if (kind != OwnerAgentSession && kind != OwnerOperatingModeExecution) || strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("evidence owner kind and id are required")
	}
	return s.listRecords(ctx, `SELECT o.source_system, o.source_event_id, o.run_id, o.subject_kind, o.subject_id, o.action, o.confidence, o.verification, o.content_digest, o.metadata_json, o.observed_at, l.linked_at, l.owner_kind, l.owner_id, l.owner_round FROM evidence_links l JOIN evidence_observations o ON o.id=l.observation_id WHERE l.owner_kind=? AND l.owner_id=? ORDER BY o.observed_at, o.id`, kind, id)
}

func (s *Store) ListByRun(ctx context.Context, runID string) ([]Record, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("evidence run id is required")
	}
	return s.listRecords(ctx, `SELECT o.source_system, o.source_event_id, o.run_id, o.subject_kind, o.subject_id, o.action, o.confidence, o.verification, o.content_digest, o.metadata_json, o.observed_at, l.linked_at, l.owner_kind, l.owner_id, l.owner_round FROM evidence_links l JOIN evidence_observations o ON o.id=l.observation_id WHERE o.run_id=? ORDER BY o.observed_at, o.id`, strings.TrimSpace(runID))
}

func (s *Store) ListByEntity(ctx context.Context, subject Subject) ([]Record, error) {
	if err := subject.Validate(); err != nil {
		return nil, err
	}
	return s.listRecords(ctx, `SELECT o.source_system, o.source_event_id, o.run_id, o.subject_kind, o.subject_id, o.action, o.confidence, o.verification, o.content_digest, o.metadata_json, o.observed_at, l.linked_at, l.owner_kind, l.owner_id, l.owner_round FROM evidence_links l JOIN evidence_observations o ON o.id=l.observation_id WHERE o.subject_kind=? AND o.subject_id=? ORDER BY o.observed_at, o.id`, subject.Kind, subject.ID)
}

func (s *Store) SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	if checkpoint.ProducerID == "" || checkpoint.RunID == "" || checkpoint.FactKind == "" {
		return fmt.Errorf("producer id, run id, and fact kind are required")
	}
	if checkpoint.UpdatedAt.IsZero() {
		checkpoint.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO evidence_checkpoints (producer_id, run_id, fact_kind, cursor, updated_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(producer_id, run_id, fact_kind) DO UPDATE SET cursor=excluded.cursor, updated_at=excluded.updated_at`, checkpoint.ProducerID, checkpoint.RunID, checkpoint.FactKind, checkpoint.Cursor, checkpoint.UpdatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) SaveWatermark(ctx context.Context, watermark Watermark) error {
	if watermark.ProducerID == "" || watermark.RunID == "" || watermark.FactKind == "" || watermark.Coverage == "" {
		return fmt.Errorf("producer id, run id, fact kind, and coverage are required")
	}
	if watermark.CompletedAt.IsZero() {
		watermark.CompletedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO evidence_watermarks (producer_id, run_id, fact_kind, coverage, completed_at) VALUES (?, ?, ?, ?, ?) ON CONFLICT(producer_id, run_id, fact_kind) DO UPDATE SET coverage=excluded.coverage, completed_at=excluded.completed_at`, watermark.ProducerID, watermark.RunID, watermark.FactKind, watermark.Coverage, watermark.CompletedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) HasTerminalWatermark(ctx context.Context, producerID, runID, factKind string) (bool, error) {
	if strings.TrimSpace(producerID) == "" {
		return false, nil
	}
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM evidence_watermarks WHERE producer_id=? AND run_id=? AND fact_kind=?`, producerID, runID, factKind).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("query evidence watermark: %w", err)
	}
	return count > 0, nil
}
