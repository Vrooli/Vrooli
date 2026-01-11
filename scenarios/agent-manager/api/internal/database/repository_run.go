package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// RunRepository Implementation
// ============================================================================

type runRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.RunRepository = (*runRepository)(nil)

// runRow is the database row representation for runs.
type runRow struct {
	ID               uuid.UUID             `db:"id"`
	TaskID           uuid.UUID             `db:"task_id"`
	AgentProfileID   NullableUUID          `db:"agent_profile_id"`
	Tag              string                `db:"tag"`
	SandboxID        NullableUUID          `db:"sandbox_id"`
	RunMode          string                `db:"run_mode"`
	Status           string                `db:"status"`
	StartedAt        NullableTime          `db:"started_at"`
	EndedAt          NullableTime          `db:"ended_at"`
	Phase            string                `db:"phase"`
	LastCheckpointID NullableUUID          `db:"last_checkpoint_id"`
	LastHeartbeat    NullableTime          `db:"last_heartbeat"`
	ProgressPercent  int                   `db:"progress_percent"`
	IdempotencyKey   sql.NullString        `db:"idempotency_key"`
	Summary          NullableRunSummary    `db:"summary"`
	ErrorMsg         string                `db:"error_msg"`
	ExitCode         sql.NullInt32         `db:"exit_code"`
	ApprovalState    string                `db:"approval_state"`
	ApprovedBy       string                `db:"approved_by"`
	ApprovedAt       NullableTime          `db:"approved_at"`
	ResolvedConfig   NullableRunConfig     `db:"resolved_config"`
	DiffPath         string                `db:"diff_path"`
	LogPath          string                `db:"log_path"`
	ChangedFiles     int                   `db:"changed_files"`
	TotalSizeBytes   int64                 `db:"total_size_bytes"`
	SandboxConfig    NullableSandboxConfig `db:"sandbox_config"`
	SessionID        sql.NullString        `db:"session_id"`
	// Recommendation extraction fields (for investigation runs)
	RecommendationStatus   sql.NullString           `db:"recommendation_status"`
	RecommendationResult   NullableExtractionResult `db:"recommendation_result"`
	RecommendationAttempts int                      `db:"recommendation_attempts"`
	RecommendationError    sql.NullString           `db:"recommendation_error"`
	RecommendationQueuedAt NullableTime             `db:"recommendation_queued_at"`
	CreatedAt              SQLiteTime               `db:"created_at"`
	UpdatedAt              SQLiteTime               `db:"updated_at"`
}

func (row *runRow) toDomain() *domain.Run {
	run := &domain.Run{
		ID:               row.ID,
		TaskID:           row.TaskID,
		AgentProfileID:   row.AgentProfileID.ToPtr(),
		Tag:              row.Tag,
		SandboxID:        row.SandboxID.ToPtr(),
		RunMode:          domain.RunMode(row.RunMode),
		Status:           domain.RunStatus(row.Status),
		StartedAt:        row.StartedAt.ToPtr(),
		EndedAt:          row.EndedAt.ToPtr(),
		Phase:            domain.RunPhase(row.Phase),
		LastCheckpointID: row.LastCheckpointID.ToPtr(),
		LastHeartbeat:    row.LastHeartbeat.ToPtr(),
		ProgressPercent:  row.ProgressPercent,
		IdempotencyKey:   row.IdempotencyKey.String, // Empty string if NULL
		Summary:          row.Summary.V,
		ErrorMsg:         row.ErrorMsg,
		ApprovalState:    domain.ApprovalState(row.ApprovalState),
		ApprovedBy:       row.ApprovedBy,
		ApprovedAt:       row.ApprovedAt.ToPtr(),
		ResolvedConfig:   row.ResolvedConfig.V,
		DiffPath:         row.DiffPath,
		LogPath:          row.LogPath,
		ChangedFiles:     row.ChangedFiles,
		TotalSizeBytes:   row.TotalSizeBytes,
		SandboxConfig:    row.SandboxConfig.V,
		SessionID:        row.SessionID.String,
		// Recommendation extraction fields
		RecommendationStatus:   domain.RecommendationStatus(row.RecommendationStatus.String),
		RecommendationResult:   row.RecommendationResult.V,
		RecommendationAttempts: row.RecommendationAttempts,
		RecommendationError:    row.RecommendationError.String,
		RecommendationQueuedAt: row.RecommendationQueuedAt.ToPtr(),
		CreatedAt:              row.CreatedAt.Time(),
		UpdatedAt:              row.UpdatedAt.Time(),
	}
	if row.ExitCode.Valid {
		exitCode := int(row.ExitCode.Int32)
		run.ExitCode = &exitCode
	}
	return run
}

func runFromDomain(r *domain.Run) *runRow {
	row := &runRow{
		ID:               r.ID,
		TaskID:           r.TaskID,
		AgentProfileID:   NewNullableUUID(r.AgentProfileID),
		Tag:              r.Tag,
		SandboxID:        NewNullableUUID(r.SandboxID),
		RunMode:          string(r.RunMode),
		Status:           string(r.Status),
		StartedAt:        NewNullableTime(r.StartedAt),
		EndedAt:          NewNullableTime(r.EndedAt),
		Phase:            string(r.Phase),
		LastCheckpointID: NewNullableUUID(r.LastCheckpointID),
		LastHeartbeat:    NewNullableTime(r.LastHeartbeat),
		ProgressPercent:  r.ProgressPercent,
		IdempotencyKey:   sql.NullString{String: r.IdempotencyKey, Valid: r.IdempotencyKey != ""},
		Summary:          NullableRunSummary{V: r.Summary},
		ErrorMsg:         r.ErrorMsg,
		ApprovalState:    string(r.ApprovalState),
		ApprovedBy:       r.ApprovedBy,
		ApprovedAt:       NewNullableTime(r.ApprovedAt),
		ResolvedConfig:   NullableRunConfig{V: r.ResolvedConfig},
		DiffPath:         r.DiffPath,
		LogPath:          r.LogPath,
		ChangedFiles:     r.ChangedFiles,
		TotalSizeBytes:   r.TotalSizeBytes,
		SandboxConfig:    NullableSandboxConfig{V: r.SandboxConfig},
		SessionID:        sql.NullString{String: r.SessionID, Valid: r.SessionID != ""},
		// Recommendation extraction fields
		RecommendationStatus:   sql.NullString{String: string(r.RecommendationStatus), Valid: r.RecommendationStatus != ""},
		RecommendationResult:   NullableExtractionResult{V: r.RecommendationResult},
		RecommendationAttempts: r.RecommendationAttempts,
		RecommendationError:    sql.NullString{String: r.RecommendationError, Valid: r.RecommendationError != ""},
		RecommendationQueuedAt: NewNullableTime(r.RecommendationQueuedAt),
		CreatedAt:              SQLiteTime(r.CreatedAt),
		UpdatedAt:              SQLiteTime(r.UpdatedAt),
	}
	if r.ExitCode != nil {
		row.ExitCode = sql.NullInt32{Int32: int32(*r.ExitCode), Valid: true}
	}
	return row
}

const runColumns = `id, task_id, agent_profile_id, tag, sandbox_id, run_mode, status,
	started_at, ended_at, phase, last_checkpoint_id, last_heartbeat, progress_percent,
	idempotency_key, summary, error_msg, exit_code, approval_state, approved_by, approved_at,
	resolved_config, diff_path, log_path, changed_files, total_size_bytes, sandbox_config, session_id,
	recommendation_status, recommendation_result, recommendation_attempts, recommendation_error, recommendation_queued_at,
	created_at, updated_at`

func (r *runRepository) Create(ctx context.Context, run *domain.Run) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	now := time.Now()
	run.CreatedAt = now
	run.UpdatedAt = now

	row := runFromDomain(run)
	query := `INSERT INTO runs (id, task_id, agent_profile_id, tag, sandbox_id, run_mode, status,
		started_at, ended_at, phase, last_checkpoint_id, last_heartbeat, progress_percent,
		idempotency_key, summary, error_msg, exit_code, approval_state, approved_by, approved_at,
		resolved_config, diff_path, log_path, changed_files, total_size_bytes, sandbox_config, session_id,
		recommendation_status, recommendation_result, recommendation_attempts, recommendation_error, recommendation_queued_at,
		created_at, updated_at)
		VALUES (:id, :task_id, :agent_profile_id, :tag, :sandbox_id, :run_mode, :status,
		:started_at, :ended_at, :phase, :last_checkpoint_id, :last_heartbeat, :progress_percent,
		:idempotency_key, :summary, :error_msg, :exit_code, :approval_state, :approved_by, :approved_at,
		:resolved_config, :diff_path, :log_path, :changed_files, :total_size_bytes, :sandbox_config, :session_id,
		:recommendation_status, :recommendation_result, :recommendation_attempts, :recommendation_error, :recommendation_queued_at,
		:created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create run")
		return wrapDBError("create", "Run", run.ID.String(), err)
	}
	return nil
}

func (r *runRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	query := r.db.Rebind(fmt.Sprintf("SELECT %s FROM runs WHERE id = ?", runColumns))
	var row runRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get", "Run", id.String(), err)
	}
	return row.toDomain(), nil
}

func (r *runRepository) List(ctx context.Context, filter repository.RunListFilter) ([]*domain.Run, error) {
	var conditions []string
	var args []interface{}

	if filter.TaskID != nil {
		conditions = append(conditions, "task_id = ?")
		args = append(args, *filter.TaskID)
	}
	if filter.AgentProfileID != nil {
		conditions = append(conditions, "agent_profile_id = ?")
		args = append(args, *filter.AgentProfileID)
	}
	if filter.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.TagPrefix != "" {
		conditions = append(conditions, "tag LIKE ?")
		args = append(args, filter.TagPrefix+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	base := fmt.Sprintf("SELECT %s FROM runs%s ORDER BY created_at DESC", runColumns, whereClause)
	queryWithPaging, pagingArgs := appendLimitOffset(base, filter.Limit, filter.Offset)
	args = append(args, pagingArgs...)
	query := r.db.Rebind(queryWithPaging)

	var rows []runRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list", "Run", "", err)
	}

	result := make([]*domain.Run, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *runRepository) ListByTask(ctx context.Context, taskID uuid.UUID, filter repository.ListFilter) ([]*domain.Run, error) {
	return r.List(ctx, repository.RunListFilter{
		ListFilter: filter,
		TaskID:     &taskID,
	})
}

func (r *runRepository) Update(ctx context.Context, run *domain.Run) error {
	run.UpdatedAt = time.Now()
	row := runFromDomain(run)

	query := `UPDATE runs SET task_id = :task_id, agent_profile_id = :agent_profile_id,
		tag = :tag, sandbox_id = :sandbox_id, run_mode = :run_mode, status = :status,
		started_at = :started_at, ended_at = :ended_at, phase = :phase,
		last_checkpoint_id = :last_checkpoint_id, last_heartbeat = :last_heartbeat,
		progress_percent = :progress_percent, idempotency_key = :idempotency_key,
		summary = :summary, error_msg = :error_msg, exit_code = :exit_code,
		approval_state = :approval_state, approved_by = :approved_by, approved_at = :approved_at,
		resolved_config = :resolved_config, diff_path = :diff_path, log_path = :log_path,
		changed_files = :changed_files, total_size_bytes = :total_size_bytes, sandbox_config = :sandbox_config,
		session_id = :session_id,
		recommendation_status = :recommendation_status, recommendation_result = :recommendation_result,
		recommendation_attempts = :recommendation_attempts, recommendation_error = :recommendation_error,
		recommendation_queued_at = :recommendation_queued_at,
		updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	return nil
}

func (r *runRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM runs WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return wrapDBError("delete", "Run", id.String(), err)
	}
	return nil
}

func (r *runRepository) CountByStatus(ctx context.Context, status domain.RunStatus) (int, error) {
	query := r.db.Rebind(`SELECT COUNT(*) FROM runs WHERE status = ?`)
	var count int
	if err := r.db.GetContext(ctx, &count, query, string(status)); err != nil {
		return 0, wrapDBError("count_by_status", "Run", string(status), err)
	}
	return count, nil
}

// ListPendingRecommendationExtractions returns runs that need recommendation extraction.
// Returns runs with status=pending or status=failed (with attempts < maxRetries),
// ordered by queued_at ascending (oldest first).
// NOTE: This uses a broad filter (tag contains 'investigation' and not 'apply').
// The caller should apply additional filtering using the configurable allowlist.
func (r *runRepository) ListPendingRecommendationExtractions(ctx context.Context, maxRetries, limit int) ([]*domain.Run, error) {
	// Query for runs that:
	// 1. Tag contains 'investigation' but not 'apply' (broad filter, caller filters precisely)
	// 2. Have recommendation_status = 'pending' OR (status = 'failed' AND attempts < maxRetries)
	// Ordered by queued_at ascending (oldest first)
	query := r.db.Rebind(fmt.Sprintf(`
		SELECT %s FROM runs
		WHERE tag LIKE '%%investigation%%'
		  AND tag NOT LIKE '%%apply'
		  AND (
		      recommendation_status = ?
		      OR (recommendation_status = ? AND recommendation_attempts < ?)
		  )
		ORDER BY recommendation_queued_at ASC NULLS LAST
		LIMIT ?
	`, runColumns))

	args := []interface{}{
		string(domain.RecommendationStatusPending),
		string(domain.RecommendationStatusFailed),
		maxRetries,
		limit,
	}

	var rows []runRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list_pending_recommendation_extractions", "Run", "", err)
	}

	result := make([]*domain.Run, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// ClaimRecommendationExtraction atomically marks a run as "extracting".
// Returns true if claim succeeded (no concurrent extractor got it first).
// Uses optimistic locking via WHERE clause to prevent race conditions.
func (r *runRepository) ClaimRecommendationExtraction(ctx context.Context, runID uuid.UUID) (bool, error) {
	// Atomic UPDATE: only succeeds if status is still pending or failed
	query := r.db.Rebind(`
		UPDATE runs
		SET recommendation_status = ?, updated_at = ?
		WHERE id = ?
		  AND (recommendation_status = ? OR recommendation_status = ?)
	`)

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		string(domain.RecommendationStatusExtracting),
		now,
		runID,
		string(domain.RecommendationStatusPending),
		string(domain.RecommendationStatusFailed),
	)
	if err != nil {
		return false, wrapDBError("claim_recommendation_extraction", "Run", runID.String(), err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, wrapDBError("claim_recommendation_extraction_rows", "Run", runID.String(), err)
	}

	return rowsAffected > 0, nil
}

// ListUnextractedInvestigationRuns returns complete investigation runs that haven't had
// recommendations extracted yet (status is empty, NULL, or "none").
// Used on startup to seed the extraction queue with existing runs.
// Limited to most recent runs (by created_at desc) to avoid overwhelming the queue.
// NOTE: If tagPrefix is empty, uses a broad filter (tag contains 'investigation').
// The caller should apply additional filtering using the configurable allowlist.
func (r *runRepository) ListUnextractedInvestigationRuns(ctx context.Context, tagPrefix string, limit int) ([]*domain.Run, error) {
	// Query for runs that:
	// 1. Tag matches pattern (broad filter, caller filters precisely)
	// 2. Are complete (status = 'complete')
	// 3. Have recommendation_status = '' OR NULL OR 'none'
	// Ordered by created_at DESC (most recent first)

	// Use broad filter if no prefix specified, otherwise use prefix
	tagPattern := "%investigation%"
	if tagPrefix != "" {
		tagPattern = tagPrefix + "%"
	}

	query := r.db.Rebind(fmt.Sprintf(`
		SELECT %s FROM runs
		WHERE tag LIKE ?
		  AND tag NOT LIKE '%%apply'
		  AND status = ?
		  AND (recommendation_status = '' OR recommendation_status IS NULL OR recommendation_status = ?)
		ORDER BY created_at DESC
		LIMIT ?
	`, runColumns))

	args := []interface{}{
		tagPattern,
		string(domain.RunStatusComplete),
		string(domain.RecommendationStatusNone),
		limit,
	}

	var rows []runRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list_unextracted_investigation_runs", "Run", "", err)
	}

	result := make([]*domain.Run, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// ListStaleExtractions returns runs that have been stuck in "extracting" status
// for longer than the stale timeout. These are likely from crashed workers.
func (r *runRepository) ListStaleExtractions(ctx context.Context, staleTimeout time.Duration, limit int) ([]*domain.Run, error) {
	// Query for runs that:
	// 1. Have recommendation_status = 'extracting'
	// 2. Were updated more than staleTimeout ago (indicating a stuck worker)
	cutoff := time.Now().Add(-staleTimeout)

	query := r.db.Rebind(fmt.Sprintf(`
		SELECT %s FROM runs
		WHERE recommendation_status = ?
		  AND updated_at < ?
		ORDER BY updated_at ASC
		LIMIT ?
	`, runColumns))

	args := []interface{}{
		string(domain.RecommendationStatusExtracting),
		cutoff,
		limit,
	}

	var rows []runRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("list_stale_extractions", "Run", "", err)
	}

	result := make([]*domain.Run, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// ============================================================================
// EventRepository Implementation
// ============================================================================

type eventRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.EventRepository = (*eventRepository)(nil)

// eventRow is the database row representation for run_events.
type eventRow struct {
	ID        uuid.UUID  `db:"id"`
	RunID     uuid.UUID  `db:"run_id"`
	Sequence  int64      `db:"sequence"`
	EventType string     `db:"event_type"`
	Timestamp SQLiteTime `db:"timestamp"`
	Data      []byte     `db:"data"`
}

func (e *eventRow) toDomain() *domain.RunEvent {
	evt := &domain.RunEvent{
		ID:        e.ID,
		RunID:     e.RunID,
		Sequence:  e.Sequence,
		EventType: domain.RunEventType(e.EventType),
		Timestamp: e.Timestamp.Time(),
	}

	// Parse legacy data format
	var legacy domain.RunEventData
	if err := json.Unmarshal(e.Data, &legacy); err == nil {
		evt.Data = legacy.ToTypedPayload()
	}
	return evt
}

const eventColumns = `id, run_id, sequence, event_type, timestamp, data`

func (r *eventRepository) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Get the next sequence number
	var maxSeq int64
	query := r.db.Rebind(`SELECT COALESCE(MAX(sequence), -1) FROM run_events WHERE run_id = ?`)
	if err := r.db.GetContext(ctx, &maxSeq, query, runID); err != nil {
		return wrapDBError("get_max_sequence", "RunEvent", runID.String(), err)
	}

	for _, evt := range events {
		maxSeq++
		evt.RunID = runID
		evt.Sequence = maxSeq
		if evt.ID == uuid.Nil {
			evt.ID = uuid.New()
		}
		if evt.Timestamp.IsZero() {
			evt.Timestamp = time.Now()
		}

		data, err := json.Marshal(evt.Data)
		if err != nil {
			return wrapDBError("marshal_event", "RunEvent", runID.String(), err)
		}

		insertQuery := `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data)
			VALUES (:id, :run_id, :sequence, :event_type, :timestamp, :data)`

		row := struct {
			ID        uuid.UUID  `db:"id"`
			RunID     uuid.UUID  `db:"run_id"`
			Sequence  int64      `db:"sequence"`
			EventType string     `db:"event_type"`
			Timestamp SQLiteTime `db:"timestamp"`
			Data      []byte     `db:"data"`
		}{
			ID:        evt.ID,
			RunID:     evt.RunID,
			Sequence:  evt.Sequence,
			EventType: string(evt.EventType),
			Timestamp: SQLiteTime(evt.Timestamp),
			Data:      data,
		}

		if _, err := r.db.NamedExecContext(ctx, insertQuery, row); err != nil {
			return wrapDBError("insert_event", "RunEvent", runID.String(), err)
		}
	}

	return nil
}

func (r *eventRepository) Get(ctx context.Context, runID uuid.UUID, afterSequence int64, limit int) ([]*domain.RunEvent, error) {
	base := fmt.Sprintf("SELECT %s FROM run_events WHERE run_id = ? AND sequence > ? ORDER BY sequence ASC", eventColumns)
	queryWithLimit := base
	args := []interface{}{runID, afterSequence}
	if limit > 0 {
		queryWithLimit += " LIMIT ?"
		args = append(args, limit)
	}
	query := r.db.Rebind(queryWithLimit)

	var rows []eventRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_events", "RunEvent", runID.String(), err)
	}

	result := make([]*domain.RunEvent, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *eventRepository) GetByType(ctx context.Context, runID uuid.UUID, types []domain.RunEventType, limit int) ([]*domain.RunEvent, error) {
	if len(types) == 0 {
		return []*domain.RunEvent{}, nil
	}

	typeStrs := make([]interface{}, len(types))
	placeholders := make([]string, len(types))
	for i, t := range types {
		typeStrs[i] = string(t)
		placeholders[i] = "?"
	}

	base := fmt.Sprintf("SELECT %s FROM run_events WHERE run_id = ? AND event_type IN (%s) ORDER BY sequence ASC",
		eventColumns, strings.Join(placeholders, ","))
	args := append([]interface{}{runID}, typeStrs...)

	if limit > 0 {
		base += " LIMIT ?"
		args = append(args, limit)
	}
	query := r.db.Rebind(base)

	var rows []eventRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_events_by_type", "RunEvent", runID.String(), err)
	}

	result := make([]*domain.RunEvent, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

func (r *eventRepository) Count(ctx context.Context, runID uuid.UUID) (int64, error) {
	query := r.db.Rebind(`SELECT COUNT(*) FROM run_events WHERE run_id = ?`)
	var count int64
	if err := r.db.GetContext(ctx, &count, query, runID); err != nil {
		return 0, wrapDBError("count_events", "RunEvent", runID.String(), err)
	}
	return count, nil
}

func (r *eventRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM run_events WHERE run_id = ?`)
	_, err := r.db.ExecContext(ctx, query, runID)
	if err != nil {
		return wrapDBError("delete_events", "RunEvent", runID.String(), err)
	}
	return nil
}
