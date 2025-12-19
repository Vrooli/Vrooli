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
	ID               uuid.UUID          `db:"id"`
	TaskID           uuid.UUID          `db:"task_id"`
	AgentProfileID   NullableUUID       `db:"agent_profile_id"`
	Tag              string             `db:"tag"`
	SandboxID        NullableUUID       `db:"sandbox_id"`
	RunMode          string             `db:"run_mode"`
	Status           string             `db:"status"`
	StartedAt        NullableTime       `db:"started_at"`
	EndedAt          NullableTime       `db:"ended_at"`
	Phase            string             `db:"phase"`
	LastCheckpointID NullableUUID       `db:"last_checkpoint_id"`
	LastHeartbeat    NullableTime       `db:"last_heartbeat"`
	ProgressPercent  int                `db:"progress_percent"`
	IdempotencyKey   sql.NullString     `db:"idempotency_key"`
	Summary          NullableRunSummary `db:"summary"`
	ErrorMsg         string             `db:"error_msg"`
	ExitCode         sql.NullInt32      `db:"exit_code"`
	ApprovalState    string             `db:"approval_state"`
	ApprovedBy       string             `db:"approved_by"`
	ApprovedAt       NullableTime       `db:"approved_at"`
	ResolvedConfig   NullableRunConfig  `db:"resolved_config"`
	DiffPath         string             `db:"diff_path"`
	LogPath          string             `db:"log_path"`
	ChangedFiles     int                `db:"changed_files"`
	TotalSizeBytes   int64              `db:"total_size_bytes"`
	CreatedAt        SQLiteTime         `db:"created_at"`
	UpdatedAt        SQLiteTime         `db:"updated_at"`
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
		CreatedAt:        row.CreatedAt.Time(),
		UpdatedAt:        row.UpdatedAt.Time(),
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
		CreatedAt:        SQLiteTime(r.CreatedAt),
		UpdatedAt:        SQLiteTime(r.UpdatedAt),
	}
	if r.ExitCode != nil {
		row.ExitCode = sql.NullInt32{Int32: int32(*r.ExitCode), Valid: true}
	}
	return row
}

const runColumns = `id, task_id, agent_profile_id, tag, sandbox_id, run_mode, status,
	started_at, ended_at, phase, last_checkpoint_id, last_heartbeat, progress_percent,
	idempotency_key, summary, error_msg, exit_code, approval_state, approved_by, approved_at,
	resolved_config, diff_path, log_path, changed_files, total_size_bytes, created_at, updated_at`

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
		resolved_config, diff_path, log_path, changed_files, total_size_bytes, created_at, updated_at)
		VALUES (:id, :task_id, :agent_profile_id, :tag, :sandbox_id, :run_mode, :status,
		:started_at, :ended_at, :phase, :last_checkpoint_id, :last_heartbeat, :progress_percent,
		:idempotency_key, :summary, :error_msg, :exit_code, :approval_state, :approved_by, :approved_at,
		:resolved_config, :diff_path, :log_path, :changed_files, :total_size_bytes, :created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create run")
		return fmt.Errorf("failed to create run: %w", err)
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
		return nil, fmt.Errorf("failed to get run: %w", err)
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
		return nil, fmt.Errorf("failed to list runs: %w", err)
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
		changed_files = :changed_files, total_size_bytes = :total_size_bytes, updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return fmt.Errorf("failed to update run: %w", err)
	}
	return nil
}

func (r *runRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM runs WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete run: %w", err)
	}
	return nil
}

func (r *runRepository) CountByStatus(ctx context.Context, status domain.RunStatus) (int, error) {
	query := r.db.Rebind(`SELECT COUNT(*) FROM runs WHERE status = ?`)
	var count int
	if err := r.db.GetContext(ctx, &count, query, string(status)); err != nil {
		return 0, fmt.Errorf("failed to count runs by status: %w", err)
	}
	return count, nil
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
		return fmt.Errorf("failed to get max sequence: %w", err)
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
			return fmt.Errorf("failed to marshal event data: %w", err)
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
			return fmt.Errorf("failed to insert event: %w", err)
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
		return nil, fmt.Errorf("failed to get events: %w", err)
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
		return nil, fmt.Errorf("failed to get events by type: %w", err)
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
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

func (r *eventRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM run_events WHERE run_id = ?`)
	_, err := r.db.ExecContext(ctx, query, runID)
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}
	return nil
}
