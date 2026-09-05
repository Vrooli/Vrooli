package conversationsearch

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"agent-manager/internal/sqlcompat"
)

var ErrNotFound = errors.New("conversation search projection not found")

type SQLiteRepository struct {
	db             sqlcompat.DB
	coverageMu     sync.Mutex
	coverageCached [3]uint64
	coverageUntil  time.Time
}

func NewSQLiteRepository(db sqlcompat.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) UpsertDocument(ctx context.Context, document Document) error {
	if err := validateDocument(document); err != nil {
		return err
	}
	tags, err := marshalStringSlice(document.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	workloads, err := marshalStringSlice(document.Workloads)
	if err != nil {
		return fmt.Errorf("marshal workloads: %w", err)
	}
	visible := 0
	if document.Visible {
		visible = 1
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO conversation_search_documents (
        document_id, source_run_id, source_event_id, source_message_id,
        chunk_index, chunk_total, start_byte, end_byte, event_sequence, role, occurred_at, content,
        content_class, source_hash, content_hash, recipe_version, harness,
        source_session_id, provider_origin, importer, project_scope, cwd_scope,
        runner, model, profile, run_status, run_label, tags_json, workloads_json,
        evidence_ref, visible, indexed_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(document_id) DO UPDATE SET
        source_run_id=excluded.source_run_id, source_event_id=excluded.source_event_id,
        source_message_id=excluded.source_message_id, chunk_index=excluded.chunk_index,
        chunk_total=excluded.chunk_total, start_byte=excluded.start_byte,
        end_byte=excluded.end_byte, event_sequence=excluded.event_sequence,
        role=excluded.role, occurred_at=excluded.occurred_at, content=excluded.content,
        content_class=excluded.content_class, source_hash=excluded.source_hash,
        content_hash=excluded.content_hash, recipe_version=excluded.recipe_version,
        harness=excluded.harness, source_session_id=excluded.source_session_id,
        provider_origin=excluded.provider_origin, importer=excluded.importer,
        project_scope=excluded.project_scope, cwd_scope=excluded.cwd_scope,
        runner=excluded.runner, model=excluded.model, profile=excluded.profile,
        run_status=excluded.run_status, run_label=excluded.run_label,
        tags_json=excluded.tags_json, workloads_json=excluded.workloads_json,
        evidence_ref=excluded.evidence_ref, visible=excluded.visible,
        indexed_at=excluded.indexed_at`,
		document.DocumentID, document.SourceRunID, document.SourceEventID, document.SourceMessageID,
		document.ChunkIndex, document.ChunkTotal, document.StartByte, document.EndByte, document.EventSequence, document.Role,
		formatTime(document.OccurredAt), document.Content, document.ContentClass, document.SourceHash,
		document.ContentHash, document.RecipeVersion, document.Harness, document.SourceSessionID,
		document.ProviderOrigin, document.Importer, document.ProjectScope, document.CWDScope,
		document.Runner, document.Model, document.Profile, document.RunStatus, document.RunLabel,
		string(tags), string(workloads), document.EvidenceRef, visible, formatTime(document.IndexedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert conversation search document %q: %w", document.DocumentID, err)
	}
	r.invalidateCoverage()
	return nil
}

func (r *SQLiteRepository) DeleteDocument(ctx context.Context, documentID string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_documents WHERE document_id = ?`, documentID)
	if err != nil {
		return fmt.Errorf("delete conversation search document %q: %w", documentID, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted conversation search document %q: %w", documentID, err)
	}
	if count == 0 {
		return ErrNotFound
	}
	r.invalidateCoverage()
	return nil
}

func (r *SQLiteRepository) DeleteSourceEvent(ctx context.Context, runID, eventID string) (int64, error) {
	return r.archiveAndDelete(ctx, `source_run_id = ? AND source_event_id = ?`, runID, eventID)
}

func (r *SQLiteRepository) DeleteRun(ctx context.Context, runID string) (int64, error) {
	return r.archiveAndDelete(ctx, `source_run_id = ?`, runID)
}

func (r *SQLiteRepository) archiveAndDelete(ctx context.Context, predicate string, args ...any) (int64, error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	archive := `INSERT OR REPLACE INTO conversation_search_deleted_sources (document_id, source_run_id, source_event_id, deleted_at)
SELECT document_id, source_run_id, source_event_id, ? FROM conversation_search_documents WHERE ` + predicate
	archiveArgs := append([]any{formatTime(time.Now().UTC())}, args...)
	if _, err := tx.ExecContext(ctx, archive, archiveArgs...); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM conversation_search_documents WHERE `+predicate, args...)
	if err != nil {
		return 0, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	r.invalidateCoverage()
	return count, nil
}

func (r *SQLiteRepository) deleteWhere(ctx context.Context, predicate string, args ...any) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_documents WHERE `+predicate, args...)
	if err != nil {
		return 0, fmt.Errorf("delete conversation search projection: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count deleted conversation search projection: %w", err)
	}
	r.invalidateCoverage()
	return count, nil
}

func (r *SQLiteRepository) invalidateCoverage() {
	r.coverageMu.Lock()
	r.coverageUntil = time.Time{}
	r.coverageMu.Unlock()
}

func (r *SQLiteRepository) GetDocument(ctx context.Context, documentID string) (Document, error) {
	var row documentRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM conversation_search_documents WHERE document_id = ?`, documentID)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("get conversation search document %q: %w", documentID, err)
	}
	return row.document()
}

func (r *SQLiteRepository) VisibleDocument(ctx context.Context, documentID string) (bool, error) {
	var visible int
	err := r.db.GetContext(ctx, &visible, `SELECT visible FROM conversation_search_documents WHERE document_id = ?`, documentID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, fmt.Errorf("resolve conversation search visibility %q: %w", documentID, err)
	}
	return visible == 1, nil
}

func (r *SQLiteRepository) CountCoverage(ctx context.Context) (visibleMessages, catalogDocuments, lexicalDocuments uint64, err error) {
	r.coverageMu.Lock()
	defer r.coverageMu.Unlock()
	if time.Now().Before(r.coverageUntil) {
		return r.coverageCached[0], r.coverageCached[1], r.coverageCached[2], nil
	}
	if err := r.db.GetContext(ctx, &visibleMessages, `SELECT COUNT(*) FROM (
        SELECT source_run_id, source_message_id FROM conversation_search_documents
        WHERE visible = 1 GROUP BY source_run_id, source_message_id
    )`); err != nil {
		return 0, 0, 0, fmt.Errorf("count visible conversation messages: %w", err)
	}
	if err := r.db.GetContext(ctx, &catalogDocuments, `SELECT COUNT(*) FROM conversation_search_documents WHERE visible = 1`); err != nil {
		return 0, 0, 0, fmt.Errorf("count visible conversation documents: %w", err)
	}
	if err := r.db.GetContext(ctx, &lexicalDocuments, `SELECT COUNT(*) FROM conversation_search_fts f
        JOIN conversation_search_documents d ON d.rowid = f.rowid WHERE d.visible = 1`); err != nil {
		return 0, 0, 0, fmt.Errorf("count lexical conversation documents: %w", err)
	}
	r.coverageCached = [3]uint64{visibleMessages, catalogDocuments, lexicalDocuments}
	r.coverageUntil = time.Now().Add(5 * time.Second)
	return visibleMessages, catalogDocuments, lexicalDocuments, nil
}

func (r *SQLiteRepository) SaveCheckpoint(ctx context.Context, checkpoint Checkpoint) error {
	if checkpoint.SourceName == "" || checkpoint.UpdatedAt.IsZero() {
		return errors.New("checkpoint source name and updated time are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO conversation_search_checkpoints
        (source_name, source_cursor, source_fingerprint, updated_at, last_error_code)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(source_name) DO UPDATE SET source_cursor=excluded.source_cursor,
        source_fingerprint=excluded.source_fingerprint, updated_at=excluded.updated_at,
        last_error_code=excluded.last_error_code`, checkpoint.SourceName, checkpoint.SourceCursor,
		checkpoint.SourceFingerprint, formatTime(checkpoint.UpdatedAt), checkpoint.LastErrorCode)
	if err != nil {
		return fmt.Errorf("save conversation search checkpoint %q: %w", checkpoint.SourceName, err)
	}
	return nil
}

func (r *SQLiteRepository) LoadCheckpoint(ctx context.Context, sourceName string) (Checkpoint, error) {
	var row struct {
		SourceName        string `db:"source_name"`
		SourceCursor      string `db:"source_cursor"`
		SourceFingerprint string `db:"source_fingerprint"`
		UpdatedAt         string `db:"updated_at"`
		LastErrorCode     string `db:"last_error_code"`
	}
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM conversation_search_checkpoints WHERE source_name = ?`, sourceName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Checkpoint{}, ErrNotFound
		}
		return Checkpoint{}, fmt.Errorf("load conversation search checkpoint %q: %w", sourceName, err)
	}
	updatedAt, err := parseTime(row.UpdatedAt)
	if err != nil {
		return Checkpoint{}, err
	}
	return Checkpoint{SourceName: row.SourceName, SourceCursor: row.SourceCursor, SourceFingerprint: row.SourceFingerprint, UpdatedAt: updatedAt, LastErrorCode: row.LastErrorCode}, nil
}

func (r *SQLiteRepository) SaveGeneration(ctx context.Context, generation Generation) error {
	if generation.GenerationID == "" || generation.State == "" || generation.RecipeVersion == "" || generation.CreatedAt.IsZero() || generation.UpdatedAt.IsZero() {
		return errors.New("generation id, state, recipe version, and timestamps are required")
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO conversation_search_generations
        (generation_id, state, recipe_version, source_checkpoint, planned_documents,
         processed_documents, failed_documents, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(generation_id) DO UPDATE SET state=excluded.state,
        recipe_version=excluded.recipe_version, source_checkpoint=excluded.source_checkpoint,
        planned_documents=excluded.planned_documents, processed_documents=excluded.processed_documents,
        failed_documents=excluded.failed_documents, updated_at=excluded.updated_at`,
		generation.GenerationID, generation.State, generation.RecipeVersion, generation.SourceCheckpoint,
		generation.PlannedDocuments, generation.ProcessedDocuments, generation.FailedDocuments,
		formatTime(generation.CreatedAt), formatTime(generation.UpdatedAt))
	if err != nil {
		return fmt.Errorf("save conversation search generation %q: %w", generation.GenerationID, err)
	}
	return nil
}

func (r *SQLiteRepository) LoadGeneration(ctx context.Context, generationID string) (Generation, error) {
	var row generationRow
	if err := r.db.GetContext(ctx, &row, `SELECT * FROM conversation_search_generations WHERE generation_id = ?`, generationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Generation{}, ErrNotFound
		}
		return Generation{}, fmt.Errorf("load conversation search generation %q: %w", generationID, err)
	}
	return row.generation()
}

func (r *SQLiteRepository) BuildingGenerations(ctx context.Context) ([]Generation, error) {
	var rows []generationRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT * FROM conversation_search_generations WHERE state IN ('building','ready') ORDER BY created_at`); err != nil {
		return nil, err
	}
	out := make([]Generation, 0, len(rows))
	for _, row := range rows {
		generation, err := row.generation()
		if err != nil {
			return nil, err
		}
		out = append(out, generation)
	}
	return out, nil
}

const projectionDocumentColumns = `document_id, source_run_id, source_event_id, source_message_id,
chunk_index, chunk_total, start_byte, end_byte, event_sequence, role, occurred_at, content,
content_class, source_hash, content_hash, recipe_version, harness, source_session_id,
provider_origin, importer, project_scope, cwd_scope, runner, model, profile, run_status,
run_label, tags_json, workloads_json, evidence_ref, visible, indexed_at`

// BeginStagedGeneration clears only the named shadow. The active projection is
// never touched until PromoteStagedGeneration commits.
func (r *SQLiteRepository) BeginStagedGeneration(ctx context.Context, generationID string) error {
	if generationID == "" {
		return errors.New("generation id is required")
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_generation_documents WHERE generation_id = ?`, generationID)
	return err
}

func (r *SQLiteRepository) BeginIncrementalGeneration(ctx context.Context, generationID string) error {
	if err := r.BeginStagedGeneration(ctx, generationID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO conversation_search_generation_documents (generation_id, `+projectionDocumentColumns+`)
SELECT ?, `+projectionDocumentColumns+` FROM conversation_search_documents`, generationID)
	return err
}

func (r *SQLiteRepository) DeleteStagedRun(ctx context.Context, generationID, runID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_generation_documents WHERE generation_id=? AND source_run_id=?`, generationID, runID)
	return err
}

func (r *SQLiteRepository) DeleteStagedEvent(ctx context.Context, generationID, runID, eventID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_generation_documents WHERE generation_id=? AND source_run_id=? AND source_event_id=?`, generationID, runID, eventID)
	return err
}

func (r *SQLiteRepository) RunDocumentIDs(ctx context.Context, runID string) ([]string, error) {
	var ids []string
	err := r.db.SelectContext(ctx, &ids, `SELECT document_id FROM conversation_search_documents WHERE source_run_id=? ORDER BY document_id`, runID)
	return ids, err
}

func (r *SQLiteRepository) EventDocumentIDs(ctx context.Context, runID, eventID string) ([]string, error) {
	var ids []string
	err := r.db.SelectContext(ctx, &ids, `SELECT document_id FROM conversation_search_documents WHERE source_run_id=? AND source_event_id=? ORDER BY document_id`, runID, eventID)
	return ids, err
}

func (r *SQLiteRepository) TombstonedDocumentIDs(ctx context.Context, runID, eventID string) ([]string, error) {
	query := `SELECT document_id FROM conversation_search_deleted_sources WHERE source_run_id=?`
	args := []any{runID}
	if eventID != "" {
		query += ` AND source_event_id=?`
		args = append(args, eventID)
	}
	query += ` ORDER BY document_id`
	var ids []string
	err := r.db.SelectContext(ctx, &ids, query, args...)
	return ids, err
}

func (r *SQLiteRepository) ClearTombstones(ctx context.Context, changes []ProjectionChange) error {
	for _, change := range changes {
		if change.Operation == ChangeDeleteEvent {
			if _, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_deleted_sources WHERE source_run_id=? AND source_event_id=?`, change.SourceRunID, change.SourceEventID); err != nil {
				return err
			}
		} else if change.Operation == ChangeDeleteRun {
			if _, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_deleted_sources WHERE source_run_id=?`, change.SourceRunID); err != nil {
				return err
			}
		}
	}
	return nil
}

// ApplyStagedChanges atomically replaces only the runs/events named by a
// bounded change batch. It avoids copying and re-publishing the full serving
// corpus for routine transcript imports.
func (r *SQLiteRepository) ApplyStagedChanges(ctx context.Context, generationID string, changes []ProjectionChange) error {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, change := range changes {
		switch change.Operation {
		case ChangeUpsertRun:
			if _, err = tx.ExecContext(ctx, `DELETE FROM conversation_search_documents WHERE source_run_id=?`, change.SourceRunID); err == nil {
				_, err = tx.ExecContext(ctx, `INSERT INTO conversation_search_documents (`+projectionDocumentColumns+`)
SELECT `+projectionDocumentColumns+` FROM conversation_search_generation_documents WHERE generation_id=? AND source_run_id=?`, generationID, change.SourceRunID)
			}
		case ChangeDeleteEvent:
			_, err = tx.ExecContext(ctx, `DELETE FROM conversation_search_documents WHERE source_run_id=? AND source_event_id=?`, change.SourceRunID, change.SourceEventID)
		case ChangeDeleteRun:
			_, err = tx.ExecContext(ctx, `DELETE FROM conversation_search_documents WHERE source_run_id=?`, change.SourceRunID)
		default:
			err = fmt.Errorf("unsupported staged change %q", change.Operation)
		}
		if err != nil {
			return fmt.Errorf("apply staged conversation change %d: %w", change.Sequence, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	r.invalidateCoverage()
	return nil
}

func (r *SQLiteRepository) StageDocument(ctx context.Context, generationID string, document Document) error {
	if generationID == "" {
		return errors.New("generation id is required")
	}
	if err := validateDocument(document); err != nil {
		return err
	}
	tags, err := marshalStringSlice(document.Tags)
	if err != nil {
		return err
	}
	workloads, err := marshalStringSlice(document.Workloads)
	if err != nil {
		return err
	}
	visible := 0
	if document.Visible {
		visible = 1
	}
	query := `INSERT INTO conversation_search_generation_documents (generation_id, ` + projectionDocumentColumns + `)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(generation_id, document_id) DO UPDATE SET
source_run_id=excluded.source_run_id, source_event_id=excluded.source_event_id,
source_message_id=excluded.source_message_id, chunk_index=excluded.chunk_index,
chunk_total=excluded.chunk_total, start_byte=excluded.start_byte, end_byte=excluded.end_byte,
event_sequence=excluded.event_sequence, role=excluded.role, occurred_at=excluded.occurred_at,
content=excluded.content, content_class=excluded.content_class, source_hash=excluded.source_hash,
content_hash=excluded.content_hash, recipe_version=excluded.recipe_version, harness=excluded.harness,
source_session_id=excluded.source_session_id, provider_origin=excluded.provider_origin,
importer=excluded.importer, project_scope=excluded.project_scope, cwd_scope=excluded.cwd_scope,
runner=excluded.runner, model=excluded.model, profile=excluded.profile, run_status=excluded.run_status,
run_label=excluded.run_label, tags_json=excluded.tags_json, workloads_json=excluded.workloads_json,
evidence_ref=excluded.evidence_ref, visible=excluded.visible, indexed_at=excluded.indexed_at`
	_, err = r.db.ExecContext(ctx, query, generationID, document.DocumentID, document.SourceRunID,
		document.SourceEventID, document.SourceMessageID, document.ChunkIndex, document.ChunkTotal,
		document.StartByte, document.EndByte, document.EventSequence, document.Role,
		formatTime(document.OccurredAt), document.Content, document.ContentClass, document.SourceHash,
		document.ContentHash, document.RecipeVersion, document.Harness, document.SourceSessionID,
		document.ProviderOrigin, document.Importer, document.ProjectScope, document.CWDScope,
		document.Runner, document.Model, document.Profile, document.RunStatus, document.RunLabel,
		string(tags), string(workloads), document.EvidenceRef, visible, formatTime(document.IndexedAt))
	if err != nil {
		return fmt.Errorf("stage conversation document %q: %w", document.DocumentID, err)
	}
	return nil
}

func (r *SQLiteRepository) CountStagedGeneration(ctx context.Context, generationID string) (uint64, error) {
	var count uint64
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id = ? AND visible = 1`, generationID)
	return count, err
}

// PublishStagedGeneration makes the validated catalog + FTS shadow available
// in bounded transactions while leaving generation state as building. Stable
// document identities keep unchanged rows intact, and yielding between batches
// keeps the single SQLite connection available to health and serving traffic.
func (r *SQLiteRepository) PublishStagedGeneration(ctx context.Context, generationID string, expected uint64) error {
	defer r.invalidateCoverage()
	return r.publishGenerationBatches(ctx, generationID, expected)
}

func (r *SQLiteRepository) publishGenerationBatches(ctx context.Context, generationID string, expected uint64) error {
	generation, err := r.LoadGeneration(ctx, generationID)
	if err != nil {
		return err
	}
	if generation.State == "building" {
		conn, connErr := r.db.Conn(ctx)
		if connErr != nil {
			return connErr
		}
		tx, beginErr := conn.BeginTx(ctx, nil)
		if beginErr != nil {
			conn.Close()
			return beginErr
		}
		_, err = tx.ExecContext(ctx, `UPDATE conversation_search_generations SET state='ready', updated_at=? WHERE generation_id=?`, formatTime(time.Now().UTC()), generationID)
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		_ = conn.Close()
		if err != nil {
			return fmt.Errorf("initialize lexical bootstrap: %w", err)
		}
	}
	const batchSize = 1000
	after := ""
	for {
		var ids []string
		if err := r.db.SelectContext(ctx, &ids, `SELECT document_id FROM conversation_search_generation_documents
WHERE generation_id=? AND document_id>? ORDER BY document_id LIMIT ?`, generationID, after, batchSize); err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		last := ids[len(ids)-1]
		conn, connErr := r.db.Conn(ctx)
		if connErr != nil {
			return connErr
		}
		tx, beginErr := conn.BeginTx(ctx, nil)
		if beginErr != nil {
			_ = conn.Close()
			return beginErr
		}
		// Do not use INSERT OR REPLACE here. SQLite implements REPLACE as an
		// implicit delete whose DELETE triggers are disabled unless recursive
		// triggers are enabled, leaving stale rows in the standalone FTS table.
		// The explicit delete keeps catalog and FTS identity synchronized.
		if _, err := tx.ExecContext(ctx, `DELETE FROM conversation_search_documents
WHERE document_id IN (
  SELECT document_id FROM conversation_search_generation_documents
  WHERE generation_id=? AND document_id>? AND document_id<=?
)
AND NOT EXISTS (
  SELECT 1 FROM conversation_search_generation_documents staged
  WHERE staged.generation_id=?
    AND staged.document_id=conversation_search_documents.document_id
    AND staged.source_run_id IS conversation_search_documents.source_run_id
    AND staged.source_event_id IS conversation_search_documents.source_event_id
    AND staged.source_message_id IS conversation_search_documents.source_message_id
    AND staged.chunk_index IS conversation_search_documents.chunk_index
    AND staged.chunk_total IS conversation_search_documents.chunk_total
    AND staged.start_byte IS conversation_search_documents.start_byte
    AND staged.end_byte IS conversation_search_documents.end_byte
    AND staged.event_sequence IS conversation_search_documents.event_sequence
    AND staged.role IS conversation_search_documents.role
    AND staged.occurred_at IS conversation_search_documents.occurred_at
    AND staged.content IS conversation_search_documents.content
    AND staged.content_class IS conversation_search_documents.content_class
    AND staged.source_hash IS conversation_search_documents.source_hash
    AND staged.content_hash IS conversation_search_documents.content_hash
    AND staged.recipe_version IS conversation_search_documents.recipe_version
    AND staged.harness IS conversation_search_documents.harness
    AND staged.source_session_id IS conversation_search_documents.source_session_id
    AND staged.provider_origin IS conversation_search_documents.provider_origin
    AND staged.importer IS conversation_search_documents.importer
    AND staged.project_scope IS conversation_search_documents.project_scope
    AND staged.cwd_scope IS conversation_search_documents.cwd_scope
    AND staged.runner IS conversation_search_documents.runner
    AND staged.model IS conversation_search_documents.model
    AND staged.profile IS conversation_search_documents.profile
    AND staged.run_status IS conversation_search_documents.run_status
    AND staged.run_label IS conversation_search_documents.run_label
    AND staged.tags_json IS conversation_search_documents.tags_json
    AND staged.workloads_json IS conversation_search_documents.workloads_json
    AND staged.evidence_ref IS conversation_search_documents.evidence_ref
    AND staged.visible IS conversation_search_documents.visible
)`, generationID, after, last, generationID); err != nil {
			_ = tx.Rollback()
			_ = conn.Close()
			return fmt.Errorf("replace lexical bootstrap batch through %q: %w", last, err)
		}
		query := `INSERT INTO conversation_search_documents (` + projectionDocumentColumns + `)
SELECT ` + projectionDocumentColumns + ` FROM conversation_search_generation_documents staged
WHERE staged.generation_id=? AND staged.document_id>? AND staged.document_id<=?
AND NOT EXISTS (
  SELECT 1 FROM conversation_search_changes c
  WHERE c.processed_at IS NULL
    AND ((c.operation='delete_run' AND c.source_run_id=staged.source_run_id)
      OR (c.operation='delete_event' AND c.source_run_id=staged.source_run_id AND c.source_event_id=staged.source_event_id))
)
AND NOT EXISTS (
  SELECT 1 FROM conversation_search_documents serving
  WHERE serving.document_id=staged.document_id
)
ORDER BY staged.document_id`
		if _, err := tx.ExecContext(ctx, query, generationID, after, last); err != nil {
			_ = tx.Rollback()
			_ = conn.Close()
			return fmt.Errorf("publish lexical bootstrap through %q: %w", last, err)
		}
		if err := tx.Commit(); err != nil {
			_ = conn.Close()
			return fmt.Errorf("commit lexical bootstrap through %q: %w", last, err)
		}
		_ = conn.Close()
		after = last
		// Yield the single SQLite writer/connection between bounded batches so
		// health, capture, and search requests are not starved by bootstrap.
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	var staged uint64
	if err := r.db.GetContext(ctx, &staged, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id=? AND visible=1`, generationID); err != nil {
		return err
	}
	if staged != expected {
		return fmt.Errorf("lexical publication source changed: want %d documents, got %d", expected, staged)
	}
	var afterServingID string
	for {
		var orphanIDs []string
		if err := r.db.SelectContext(ctx, &orphanIDs, `SELECT serving.document_id
FROM conversation_search_documents serving
WHERE serving.document_id>?
  AND NOT EXISTS (
    SELECT 1 FROM conversation_search_generation_documents staged
    WHERE staged.generation_id=? AND staged.document_id=serving.document_id
  )
ORDER BY serving.document_id LIMIT ?`, afterServingID, generationID, batchSize); err != nil {
			return fmt.Errorf("select lexical publication orphans: %w", err)
		}
		if len(orphanIDs) == 0 {
			break
		}
		last := orphanIDs[len(orphanIDs)-1]
		result, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_documents
WHERE document_id>? AND document_id<=?
  AND NOT EXISTS (
    SELECT 1 FROM conversation_search_generation_documents staged
    WHERE staged.generation_id=? AND staged.document_id=conversation_search_documents.document_id
  )`, afterServingID, last, generationID)
		if err != nil {
			return fmt.Errorf("remove lexical publication orphans through %q: %w", last, err)
		}
		removed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("count removed lexical publication orphans: %w", err)
		}
		if removed != int64(len(orphanIDs)) {
			return fmt.Errorf("lexical publication orphan set changed: selected %d, removed %d", len(orphanIDs), removed)
		}
		afterServingID = last
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	// Repair residue from historical INSERT OR REPLACE publication. Bound each
	// writer transaction so search and health requests retain a scheduling turn.
	var afterFTSRowID int64
	for {
		var orphanRowIDs []int64
		if err := r.db.SelectContext(ctx, &orphanRowIDs, `SELECT f.rowid
FROM conversation_search_fts f
LEFT JOIN conversation_search_documents d ON d.rowid=f.rowid
WHERE f.rowid>? AND d.rowid IS NULL
ORDER BY f.rowid LIMIT ?`, afterFTSRowID, batchSize); err != nil {
			return fmt.Errorf("select lexical index orphans: %w", err)
		}
		if len(orphanRowIDs) == 0 {
			break
		}
		last := orphanRowIDs[len(orphanRowIDs)-1]
		result, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_fts
WHERE rowid>? AND rowid<=?
  AND NOT EXISTS (SELECT 1 FROM conversation_search_documents d WHERE d.rowid=conversation_search_fts.rowid)`, afterFTSRowID, last)
		if err != nil {
			return fmt.Errorf("remove lexical index orphans: %w", err)
		}
		removed, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("count removed lexical index orphans: %w", err)
		}
		if removed != int64(len(orphanRowIDs)) {
			return fmt.Errorf("lexical index orphan set changed: selected %d, removed %d", len(orphanRowIDs), removed)
		}
		afterFTSRowID = last
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return nil
}

// ActivateGeneration records the generation whose lexical and semantic
// projections both reached their atomic serving aliases.
func (r *SQLiteRepository) ActivateGeneration(ctx context.Context, generationID string, now time.Time) error {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire generation activation connection: %w", err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin generation activation: %w", err)
	}
	defer tx.Rollback()
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `UPDATE conversation_search_generations SET state='retired', updated_at=? WHERE state='active'`, stamp); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE conversation_search_generations SET state='active', updated_at=? WHERE generation_id=?`, stamp, generationID)
	if err != nil {
		return err
	}
	if changed, rowsErr := result.RowsAffected(); rowsErr != nil {
		return rowsErr
	} else if changed != 1 {
		return fmt.Errorf("activate generation %q: expected one row, changed %d", generationID, changed)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit generation activation: %w", err)
	}
	return nil
}

// PromoteStagedGeneration preserves the original one-call contract for tests
// and non-semantic callers.
func (r *SQLiteRepository) PromoteStagedGeneration(ctx context.Context, generationID string, expected uint64, now time.Time) error {
	if err := r.PublishStagedGeneration(ctx, generationID, expected); err != nil {
		return err
	}
	return r.ActivateGeneration(ctx, generationID, now)
}

func (r *SQLiteRepository) RollbackStagedGeneration(ctx context.Context, generationID, state string, now time.Time) error {
	if state != "failed" && state != "cancelled" {
		return errors.New("rollback state must be failed or cancelled")
	}
	if _, err := r.db.ExecContext(ctx, `DELETE FROM conversation_search_generation_documents WHERE generation_id = ?`, generationID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE conversation_search_generations SET state=?, updated_at=? WHERE generation_id=?`, state, formatTime(now), generationID)
	return err
}

func (r *SQLiteRepository) EnqueueChange(ctx context.Context, operation ChangeOperation, runID, eventID string, now time.Time) error {
	switch operation {
	case ChangeUpsertRun, ChangeDeleteEvent, ChangeDeleteRun, ChangeRepair:
	default:
		return fmt.Errorf("unsupported conversation projection change %q", operation)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO conversation_search_changes (operation, source_run_id, source_event_id, created_at) VALUES (?, ?, ?, ?)`, operation, runID, eventID, formatTime(now))
	return err
}

func (r *SQLiteRepository) PendingChanges(ctx context.Context, limit int) ([]ProjectionChange, error) {
	if limit <= 0 || limit > maxSourcePageSize {
		limit = maxSourcePageSize
	}
	var rows []struct {
		Sequence      int64           `db:"sequence"`
		Operation     ChangeOperation `db:"operation"`
		SourceRunID   string          `db:"source_run_id"`
		SourceEventID string          `db:"source_event_id"`
		CreatedAt     string          `db:"created_at"`
	}
	if err := r.db.SelectContext(ctx, &rows, `SELECT sequence, operation, source_run_id, source_event_id, created_at FROM conversation_search_changes WHERE processed_at IS NULL ORDER BY sequence LIMIT ?`, limit); err != nil {
		return nil, err
	}
	out := make([]ProjectionChange, 0, len(rows))
	for _, row := range rows {
		created, err := parseTime(row.CreatedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, ProjectionChange{Sequence: row.Sequence, Operation: row.Operation, SourceRunID: row.SourceRunID, SourceEventID: row.SourceEventID, CreatedAt: created})
	}
	return out, nil
}

func (r *SQLiteRepository) MarkChangesProcessed(ctx context.Context, through int64, now time.Time) error {
	if through <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `UPDATE conversation_search_changes SET processed_at=? WHERE processed_at IS NULL AND sequence <= ?`, formatTime(now), through)
	return err
}

func (r *SQLiteRepository) MaxPendingChangeSequence(ctx context.Context) (int64, error) {
	var sequence int64
	err := r.db.GetContext(ctx, &sequence, `SELECT COALESCE(MAX(sequence), 0) FROM conversation_search_changes WHERE processed_at IS NULL`)
	return sequence, err
}

func (r *SQLiteRepository) HasPendingDeletionAfter(ctx context.Context, sequence int64) (bool, error) {
	var exists int
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(
SELECT 1 FROM conversation_search_changes
WHERE processed_at IS NULL AND sequence > ? AND operation IN ('delete_event','delete_run')
)`, sequence)
	return exists != 0, err
}

func (r *SQLiteRepository) ProjectionStatus(ctx context.Context) (ProjectionStatus, error) {
	visible, catalog, lexical, err := r.CountCoverage(ctx)
	if err != nil {
		return ProjectionStatus{}, err
	}
	status := ProjectionStatus{CanonicalMessages: visible, CatalogDocuments: catalog, LexicalDocuments: lexical}
	_ = r.db.GetContext(ctx, &status.PendingChanges, `SELECT COUNT(*) FROM conversation_search_changes WHERE processed_at IS NULL`)
	_ = r.db.GetContext(ctx, &status.DeletedDocuments, `SELECT COUNT(*) FROM conversation_search_changes WHERE operation IN ('delete_event','delete_run')`)
	_ = r.db.GetContext(ctx, &status.ActiveGeneration, `SELECT generation_id FROM conversation_search_generations WHERE state='active' LIMIT 1`)
	_ = r.db.GetContext(ctx, &status.CandidateGeneration, `SELECT generation_id FROM conversation_search_generations WHERE state IN ('building','ready') ORDER BY updated_at DESC LIMIT 1`)
	var lastSuccess, lastIndexed, lastError sql.NullString
	_ = r.db.GetContext(ctx, &lastSuccess, `SELECT MAX(updated_at) FROM conversation_search_generations WHERE state='active'`)
	_ = r.db.GetContext(ctx, &lastIndexed, `SELECT MAX(indexed_at) FROM conversation_search_documents`)
	_ = r.db.GetContext(ctx, &lastError, `SELECT last_error_code FROM conversation_search_checkpoints WHERE source_name='canonical'`)
	if lastSuccess.Valid {
		status.LastSuccessAt, _ = parseTime(lastSuccess.String)
	}
	if lastIndexed.Valid {
		status.LastIndexedAt, _ = parseTime(lastIndexed.String)
	}
	if lastError.Valid {
		status.LastErrorCode = lastError.String
	}
	return status, nil
}

func (r *SQLiteRepository) ProjectionDocumentIDs(ctx context.Context) ([]string, error) {
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, `SELECT document_id FROM conversation_search_documents WHERE visible = 1 ORDER BY document_id`); err != nil {
		return nil, err
	}
	return ids, nil
}

func validateDocument(document Document) error {
	if document.DocumentID == "" || document.SourceRunID == "" || document.SourceEventID == "" || document.SourceMessageID == "" {
		return errors.New("document and source identities are required")
	}
	if document.ChunkIndex < 0 || document.ChunkTotal <= 0 || document.ChunkIndex >= document.ChunkTotal {
		return errors.New("chunk index must identify a member of chunk total")
	}
	if document.EventSequence < 0 || document.Role == "" || document.OccurredAt.IsZero() || document.IndexedAt.IsZero() {
		return errors.New("event sequence, role, occurred time, and indexed time are required")
	}
	if document.SourceHash == "" || document.ContentHash == "" || document.RecipeVersion == "" {
		return errors.New("source hash, content hash, and recipe version are required")
	}
	return nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func marshalStringSlice(values []string) ([]byte, error) {
	if values == nil {
		values = []string{}
	}
	return json.Marshal(values)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse conversation search timestamp %q: %w", value, err)
	}
	return parsed, nil
}

type documentRow struct {
	DocumentID      string       `db:"document_id"`
	SourceRunID     string       `db:"source_run_id"`
	SourceEventID   string       `db:"source_event_id"`
	SourceMessageID string       `db:"source_message_id"`
	ChunkIndex      int          `db:"chunk_index"`
	ChunkTotal      int          `db:"chunk_total"`
	StartByte       int          `db:"start_byte"`
	EndByte         int          `db:"end_byte"`
	EventSequence   int64        `db:"event_sequence"`
	Role            string       `db:"role"`
	OccurredAt      string       `db:"occurred_at"`
	Content         string       `db:"content"`
	ContentClass    ContentClass `db:"content_class"`
	SourceHash      string       `db:"source_hash"`
	ContentHash     string       `db:"content_hash"`
	RecipeVersion   string       `db:"recipe_version"`
	Harness         string       `db:"harness"`
	SourceSessionID string       `db:"source_session_id"`
	ProviderOrigin  string       `db:"provider_origin"`
	Importer        string       `db:"importer"`
	ProjectScope    string       `db:"project_scope"`
	CWDScope        string       `db:"cwd_scope"`
	Runner          string       `db:"runner"`
	Model           string       `db:"model"`
	Profile         string       `db:"profile"`
	RunStatus       string       `db:"run_status"`
	RunLabel        string       `db:"run_label"`
	TagsJSON        string       `db:"tags_json"`
	WorkloadsJSON   string       `db:"workloads_json"`
	EvidenceRef     string       `db:"evidence_ref"`
	Visible         int          `db:"visible"`
	IndexedAt       string       `db:"indexed_at"`
}

func (row documentRow) document() (Document, error) {
	var tags, workloads []string
	if err := json.Unmarshal([]byte(row.TagsJSON), &tags); err != nil {
		return Document{}, fmt.Errorf("decode conversation search tags: %w", err)
	}
	if err := json.Unmarshal([]byte(row.WorkloadsJSON), &workloads); err != nil {
		return Document{}, fmt.Errorf("decode conversation search workloads: %w", err)
	}
	occurredAt, err := parseTime(row.OccurredAt)
	if err != nil {
		return Document{}, err
	}
	indexedAt, err := parseTime(row.IndexedAt)
	if err != nil {
		return Document{}, err
	}
	return Document{DocumentID: row.DocumentID, SourceRunID: row.SourceRunID, SourceEventID: row.SourceEventID, SourceMessageID: row.SourceMessageID, ChunkIndex: row.ChunkIndex, ChunkTotal: row.ChunkTotal, StartByte: row.StartByte, EndByte: row.EndByte, EventSequence: row.EventSequence, Role: row.Role, OccurredAt: occurredAt, Content: row.Content, ContentClass: row.ContentClass, SourceHash: row.SourceHash, ContentHash: row.ContentHash, RecipeVersion: row.RecipeVersion, Harness: row.Harness, SourceSessionID: row.SourceSessionID, ProviderOrigin: row.ProviderOrigin, Importer: row.Importer, ProjectScope: row.ProjectScope, CWDScope: row.CWDScope, Runner: row.Runner, Model: row.Model, Profile: row.Profile, RunStatus: row.RunStatus, RunLabel: row.RunLabel, Tags: tags, Workloads: workloads, EvidenceRef: row.EvidenceRef, Visible: row.Visible == 1, IndexedAt: indexedAt}, nil
}

type generationRow struct {
	GenerationID       string `db:"generation_id"`
	State              string `db:"state"`
	RecipeVersion      string `db:"recipe_version"`
	SourceCheckpoint   string `db:"source_checkpoint"`
	PlannedDocuments   uint64 `db:"planned_documents"`
	ProcessedDocuments uint64 `db:"processed_documents"`
	FailedDocuments    uint64 `db:"failed_documents"`
	CreatedAt          string `db:"created_at"`
	UpdatedAt          string `db:"updated_at"`
}

func (row generationRow) generation() (Generation, error) {
	createdAt, err := parseTime(row.CreatedAt)
	if err != nil {
		return Generation{}, err
	}
	updatedAt, err := parseTime(row.UpdatedAt)
	if err != nil {
		return Generation{}, err
	}
	return Generation{GenerationID: row.GenerationID, State: row.State, RecipeVersion: row.RecipeVersion, SourceCheckpoint: row.SourceCheckpoint, PlannedDocuments: row.PlannedDocuments, ProcessedDocuments: row.ProcessedDocuments, FailedDocuments: row.FailedDocuments, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

var (
	_ ProjectionRepository = (*SQLiteRepository)(nil)
	_ VisibilityRepository = (*SQLiteRepository)(nil)
	_ StatusRepository     = (*SQLiteRepository)(nil)
)
