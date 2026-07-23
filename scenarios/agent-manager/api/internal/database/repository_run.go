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
	ID                       uuid.UUID             `db:"id"`
	TaskID                   uuid.UUID             `db:"task_id"`
	AgentProfileID           NullableUUID          `db:"agent_profile_id"`
	Tag                      string                `db:"tag"`
	SandboxID                NullableUUID          `db:"sandbox_id"`
	RunMode                  string                `db:"run_mode"`
	ExecutionMode            sql.NullString        `db:"execution_mode"`
	WebConsoleSessionID      sql.NullString        `db:"web_console_session_id"`
	Status                   string                `db:"status"`
	StartedAt                NullableTime          `db:"started_at"`
	EndedAt                  NullableTime          `db:"ended_at"`
	Phase                    string                `db:"phase"`
	LastCheckpointID         NullableUUID          `db:"last_checkpoint_id"`
	LastHeartbeat            NullableTime          `db:"last_heartbeat"`
	ProgressPercent          int                   `db:"progress_percent"`
	IdempotencyKey           sql.NullString        `db:"idempotency_key"`
	Summary                  NullableRunSummary    `db:"summary"`
	RunResult                NullableRunResult     `db:"run_result"`
	ErrorMsg                 string                `db:"error_msg"`
	ExitCode                 sql.NullInt32         `db:"exit_code"`
	ApprovalState            string                `db:"approval_state"`
	ApprovedBy               string                `db:"approved_by"`
	ApprovedAt               NullableTime          `db:"approved_at"`
	FinalizationStatus       string                `db:"finalization_status"`
	FinalizationError        string                `db:"finalization_error"`
	FinalizedAt              NullableTime          `db:"finalized_at"`
	ResolvedConfig           NullableRunConfig     `db:"resolved_config"`
	DiffPath                 string                `db:"diff_path"`
	LogPath                  string                `db:"log_path"`
	ChangedFiles             int                   `db:"changed_files"`
	TotalSizeBytes           int64                 `db:"total_size_bytes"`
	SandboxConfig            NullableSandboxConfig `db:"sandbox_config"`
	SessionID                sql.NullString        `db:"session_id"`
	RunnerPID                int                   `db:"runner_pid"`
	RunnerPGID               int                   `db:"runner_pgid"`
	TranscriptPath           sql.NullString        `db:"transcript_path"`
	TranscriptCursor         int64                 `db:"transcript_cursor"`
	TranscriptLastSeq        int64                 `db:"transcript_last_seq"`
	SourceRunIDs             sql.NullString        `db:"source_run_ids"`
	SourceInvestigationRunID NullableUUID          `db:"source_investigation_run_id"`
	ParentRunID              NullableUUID          `db:"parent_run_id"`
	ConversationID           sql.NullString        `db:"conversation_id"`
	// Identity token fields
	IdentityTokenHash      sql.NullString `db:"identity_token_hash"`
	IdentityTokenRevokedAt NullableTime   `db:"identity_token_revoked_at"`
	// Caller-supplied custom env (JSON object), re-injected on continue/wake
	CustomEnv sql.NullString `db:"custom_env"`
	// Await-handle (JSON object) for a parked run; NULL for non-parked runs
	AwaitHandle sql.NullString `db:"await_handle"`
	// Last-resolved await (re-fetch SSOT) + re-park guard bookkeeping
	LastAwaitKey        sql.NullString `db:"last_await_key"`
	LastAwaitResult     sql.NullString `db:"last_await_result"`
	LastAwaitResolvedAt NullableTime   `db:"last_await_resolved_at"`
	LastWakeSeq         int64          `db:"last_wake_seq"`
	SameKeyParkStreak   int            `db:"same_key_park_streak"`
	// Model provenance
	RequestedModel sql.NullString `db:"requested_model"`
	ActualModel    sql.NullString `db:"actual_model"`
	CreatedAt      SQLiteTime     `db:"created_at"`
	UpdatedAt      SQLiteTime     `db:"updated_at"`
}

func (row *runRow) toDomain() *domain.Run {
	sourceRunIDs := parseUUIDSliceJSON(row.SourceRunIDs)
	run := &domain.Run{
		ID:                       row.ID,
		TaskID:                   row.TaskID,
		AgentProfileID:           row.AgentProfileID.ToPtr(),
		Tag:                      row.Tag,
		SandboxID:                row.SandboxID.ToPtr(),
		RunMode:                  domain.RunMode(row.RunMode),
		ExecutionMode:            domain.ExecutionMode(row.ExecutionMode.String).Normalized(),
		WebConsoleSessionID:      row.WebConsoleSessionID.String,
		Status:                   domain.RunStatus(row.Status),
		StartedAt:                row.StartedAt.ToPtr(),
		EndedAt:                  row.EndedAt.ToPtr(),
		Phase:                    domain.RunPhase(row.Phase),
		LastCheckpointID:         row.LastCheckpointID.ToPtr(),
		LastHeartbeat:            row.LastHeartbeat.ToPtr(),
		ProgressPercent:          row.ProgressPercent,
		IdempotencyKey:           row.IdempotencyKey.String, // Empty string if NULL
		Summary:                  row.Summary.V,
		Result:                   row.RunResult.V,
		ErrorMsg:                 row.ErrorMsg,
		ApprovalState:            domain.ApprovalState(row.ApprovalState),
		ApprovedBy:               row.ApprovedBy,
		ApprovedAt:               row.ApprovedAt.ToPtr(),
		FinalizationStatus:       domain.RunFinalizationStatus(row.FinalizationStatus),
		FinalizationError:        row.FinalizationError,
		FinalizedAt:              row.FinalizedAt.ToPtr(),
		ResolvedConfig:           row.ResolvedConfig.V,
		DiffPath:                 row.DiffPath,
		LogPath:                  row.LogPath,
		ChangedFiles:             row.ChangedFiles,
		TotalSizeBytes:           row.TotalSizeBytes,
		SandboxConfig:            row.SandboxConfig.V,
		SessionID:                row.SessionID.String,
		RunnerPID:                row.RunnerPID,
		RunnerPGID:               row.RunnerPGID,
		TranscriptPath:           row.TranscriptPath.String,
		TranscriptCursor:         row.TranscriptCursor,
		TranscriptLastSeq:        row.TranscriptLastSeq,
		SourceRunIDs:             sourceRunIDs,
		SourceInvestigationRunID: row.SourceInvestigationRunID.ToPtr(),
		ParentRunID:              row.ParentRunID.ToPtr(),
		ConversationID:           row.ConversationID.String,
		// Identity token fields
		IdentityTokenHash:      row.IdentityTokenHash.String,
		IdentityTokenRevokedAt: row.IdentityTokenRevokedAt.ToPtr(),
		CustomEnv:              parseStringMapJSON(row.CustomEnv),
		AwaitHandle:            parseAwaitHandleJSON(row.AwaitHandle),
		LastAwaitKey:           row.LastAwaitKey.String,
		LastAwaitResult:        row.LastAwaitResult.String,
		LastAwaitResolvedAt:    row.LastAwaitResolvedAt.ToPtr(),
		LastWakeSeq:            row.LastWakeSeq,
		SameKeyParkStreak:      row.SameKeyParkStreak,
		// Model provenance
		RequestedModel: row.RequestedModel.String,
		ActualModel:    row.ActualModel.String,
		CreatedAt:      row.CreatedAt.Time(),
		UpdatedAt:      row.UpdatedAt.Time(),
	}
	if row.ExitCode.Valid {
		exitCode := int(row.ExitCode.Int32)
		run.ExitCode = &exitCode
	}
	return run
}

func runFromDomain(r *domain.Run) *runRow {
	finalizationStatus := r.FinalizationStatus
	if finalizationStatus == "" {
		finalizationStatus = domain.RunFinalizationStatusNone
	}
	sourceRunIDs := marshalUUIDSliceJSON(r.SourceRunIDs)
	row := &runRow{
		ID:                       r.ID,
		TaskID:                   r.TaskID,
		AgentProfileID:           NewNullableUUID(r.AgentProfileID),
		Tag:                      r.Tag,
		SandboxID:                NewNullableUUID(r.SandboxID),
		RunMode:                  string(r.RunMode),
		ExecutionMode:            sql.NullString{String: string(r.ExecutionMode.Normalized()), Valid: true},
		WebConsoleSessionID:      sql.NullString{String: r.WebConsoleSessionID, Valid: r.WebConsoleSessionID != ""},
		Status:                   string(r.Status),
		StartedAt:                NewNullableTime(r.StartedAt),
		EndedAt:                  NewNullableTime(r.EndedAt),
		Phase:                    string(r.Phase),
		LastCheckpointID:         NewNullableUUID(r.LastCheckpointID),
		LastHeartbeat:            NewNullableTime(r.LastHeartbeat),
		ProgressPercent:          r.ProgressPercent,
		IdempotencyKey:           sql.NullString{String: r.IdempotencyKey, Valid: r.IdempotencyKey != ""},
		Summary:                  NullableRunSummary{V: r.Summary},
		RunResult:                NullableRunResult{V: r.Result},
		ErrorMsg:                 r.ErrorMsg,
		ApprovalState:            string(r.ApprovalState),
		ApprovedBy:               r.ApprovedBy,
		ApprovedAt:               NewNullableTime(r.ApprovedAt),
		FinalizationStatus:       string(finalizationStatus),
		FinalizationError:        r.FinalizationError,
		FinalizedAt:              NewNullableTime(r.FinalizedAt),
		ResolvedConfig:           NullableRunConfig{V: r.ResolvedConfig},
		DiffPath:                 r.DiffPath,
		LogPath:                  r.LogPath,
		ChangedFiles:             r.ChangedFiles,
		TotalSizeBytes:           r.TotalSizeBytes,
		SandboxConfig:            NullableSandboxConfig{V: r.SandboxConfig},
		SessionID:                sql.NullString{String: r.SessionID, Valid: r.SessionID != ""},
		RunnerPID:                r.RunnerPID,
		RunnerPGID:               r.RunnerPGID,
		TranscriptPath:           sql.NullString{String: r.TranscriptPath, Valid: r.TranscriptPath != ""},
		TranscriptCursor:         r.TranscriptCursor,
		TranscriptLastSeq:        r.TranscriptLastSeq,
		SourceRunIDs:             sql.NullString{String: sourceRunIDs, Valid: sourceRunIDs != ""},
		SourceInvestigationRunID: NewNullableUUID(r.SourceInvestigationRunID),
		ParentRunID:              NewNullableUUID(r.ParentRunID),
		ConversationID:           sql.NullString{String: r.ConversationID, Valid: r.ConversationID != ""},
		// Identity token fields
		IdentityTokenHash:      sql.NullString{String: r.IdentityTokenHash, Valid: r.IdentityTokenHash != ""},
		IdentityTokenRevokedAt: NewNullableTime(r.IdentityTokenRevokedAt),
		CustomEnv:              marshalStringMapJSON(r.CustomEnv),
		AwaitHandle:            marshalAwaitHandleJSON(r.AwaitHandle),
		LastAwaitKey:           sql.NullString{String: r.LastAwaitKey, Valid: r.LastAwaitKey != ""},
		LastAwaitResult:        sql.NullString{String: r.LastAwaitResult, Valid: r.LastAwaitResult != ""},
		LastAwaitResolvedAt:    NewNullableTime(r.LastAwaitResolvedAt),
		LastWakeSeq:            r.LastWakeSeq,
		SameKeyParkStreak:      r.SameKeyParkStreak,
		// Model provenance
		RequestedModel: sql.NullString{String: r.RequestedModel, Valid: r.RequestedModel != ""},
		ActualModel:    sql.NullString{String: r.ActualModel, Valid: r.ActualModel != ""},
		CreatedAt:      SQLiteTime(r.CreatedAt),
		UpdatedAt:      SQLiteTime(r.UpdatedAt),
	}
	if r.ExitCode != nil {
		row.ExitCode = sql.NullInt32{Int32: int32(*r.ExitCode), Valid: true}
	}
	return row
}

func parseUUIDSliceJSON(raw sql.NullString) []uuid.UUID {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw.String), &ids); err != nil {
		return nil
	}
	out := make([]uuid.UUID, 0, len(ids))
	for _, s := range ids {
		id, err := uuid.Parse(s)
		if err != nil {
			continue
		}
		out = append(out, id)
	}
	return out
}

// parseStringMapJSON decodes a JSON object column into a string map, returning
// nil for NULL / empty / malformed values so callers can merge unconditionally.
func parseStringMapJSON(raw sql.NullString) map[string]string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw.String), &m); err != nil {
		return nil
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// marshalStringMapJSON encodes a string map as a JSON object column, storing
// NULL (Valid=false) for empty maps so existing rows decode unchanged.
func marshalStringMapJSON(m map[string]string) sql.NullString {
	if len(m) == 0 {
		return sql.NullString{}
	}
	data, err := json.Marshal(m)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(data), Valid: true}
}

// parseAwaitHandleJSON decodes the await_handle JSON column into a domain
// AwaitHandle, returning nil for NULL / empty / malformed values so non-parked
// runs (and rows written before this column existed) decode to a nil handle.
func parseAwaitHandleJSON(raw sql.NullString) *domain.AwaitHandle {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var h domain.AwaitHandle
	if err := json.Unmarshal([]byte(raw.String), &h); err != nil {
		return nil
	}
	return &h
}

// marshalAwaitHandleJSON encodes a domain AwaitHandle as the await_handle JSON
// column, storing NULL (Valid=false) for a nil handle so non-parked runs leave
// the column empty.
func marshalAwaitHandleJSON(h *domain.AwaitHandle) sql.NullString {
	if h == nil {
		return sql.NullString{}
	}
	data, err := json.Marshal(h)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(data), Valid: true}
}

func marshalUUIDSliceJSON(ids []uuid.UUID) string {
	if len(ids) == 0 {
		return "[]"
	}
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, id.String())
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

const runColumns = `id, task_id, agent_profile_id, tag, sandbox_id, run_mode,
	execution_mode, web_console_session_id, status,
	started_at, ended_at, phase, last_checkpoint_id, last_heartbeat, progress_percent,
	idempotency_key, summary, run_result, error_msg, exit_code, approval_state, approved_by, approved_at,
	finalization_status, finalization_error, finalized_at,
	resolved_config, diff_path, log_path, changed_files, total_size_bytes, sandbox_config, session_id,
	runner_pid, runner_pgid, transcript_path, transcript_cursor, transcript_last_seq,
	source_run_ids, source_investigation_run_id, parent_run_id, conversation_id,
	identity_token_hash, identity_token_revoked_at, custom_env, await_handle,
	last_await_key, last_await_result, last_await_resolved_at, last_wake_seq, same_key_park_streak,
	requested_model, actual_model,
	created_at, updated_at`

// listRunColumns contains the pruned column set for List() queries.
// Omits heavy fields: summary, resolved_config, sandbox_config, sandbox_id,
// idempotency_key, last_checkpoint_id, diff_path, log_path,
// approved_by, approved_at.
// NOTE: last_heartbeat MUST be included — the reconciler depends on it
// to detect stale runs. Without it, every run appears stale after creation.
const listRunColumns = `id, task_id, agent_profile_id, tag, run_mode,
	execution_mode, web_console_session_id, status,
	started_at, ended_at, phase, last_heartbeat, progress_percent,
	error_msg, exit_code, approval_state, finalization_status, finalization_error, finalized_at,
	changed_files, total_size_bytes, session_id,
	runner_pid, runner_pgid, transcript_path, transcript_cursor, transcript_last_seq,
	source_run_ids, source_investigation_run_id, parent_run_id, conversation_id,
	requested_model, actual_model,
	created_at, updated_at`

// listRunLiteRow is the database row representation for the pruned list query.
type listRunLiteRow struct {
	ID                       uuid.UUID      `db:"id"`
	TaskID                   uuid.UUID      `db:"task_id"`
	AgentProfileID           NullableUUID   `db:"agent_profile_id"`
	Tag                      string         `db:"tag"`
	RunMode                  string         `db:"run_mode"`
	ExecutionMode            sql.NullString `db:"execution_mode"`
	WebConsoleSessionID      sql.NullString `db:"web_console_session_id"`
	Status                   string         `db:"status"`
	StartedAt                NullableTime   `db:"started_at"`
	EndedAt                  NullableTime   `db:"ended_at"`
	Phase                    string         `db:"phase"`
	LastHeartbeat            NullableTime   `db:"last_heartbeat"`
	ProgressPercent          int            `db:"progress_percent"`
	ErrorMsg                 string         `db:"error_msg"`
	ExitCode                 sql.NullInt32  `db:"exit_code"`
	ApprovalState            string         `db:"approval_state"`
	FinalizationStatus       string         `db:"finalization_status"`
	FinalizationError        string         `db:"finalization_error"`
	FinalizedAt              NullableTime   `db:"finalized_at"`
	ChangedFiles             int            `db:"changed_files"`
	TotalSizeBytes           int64          `db:"total_size_bytes"`
	SessionID                sql.NullString `db:"session_id"`
	RunnerPID                int            `db:"runner_pid"`
	RunnerPGID               int            `db:"runner_pgid"`
	TranscriptPath           sql.NullString `db:"transcript_path"`
	TranscriptCursor         int64          `db:"transcript_cursor"`
	TranscriptLastSeq        int64          `db:"transcript_last_seq"`
	SourceRunIDs             sql.NullString `db:"source_run_ids"`
	SourceInvestigationRunID NullableUUID   `db:"source_investigation_run_id"`
	ParentRunID              NullableUUID   `db:"parent_run_id"`
	ConversationID           sql.NullString `db:"conversation_id"`
	RequestedModel           sql.NullString `db:"requested_model"`
	ActualModel              sql.NullString `db:"actual_model"`
	CreatedAt                SQLiteTime     `db:"created_at"`
	UpdatedAt                SQLiteTime     `db:"updated_at"`
	// Computed field from JOIN
	PromptPreview sql.NullString `db:"prompt_preview"`
}

func (row *listRunLiteRow) toDomain() *domain.Run {
	sourceRunIDs := parseUUIDSliceJSON(row.SourceRunIDs)
	run := &domain.Run{
		ID:                       row.ID,
		TaskID:                   row.TaskID,
		AgentProfileID:           row.AgentProfileID.ToPtr(),
		Tag:                      row.Tag,
		RunMode:                  domain.RunMode(row.RunMode),
		ExecutionMode:            domain.ExecutionMode(row.ExecutionMode.String).Normalized(),
		WebConsoleSessionID:      row.WebConsoleSessionID.String,
		Status:                   domain.RunStatus(row.Status),
		StartedAt:                row.StartedAt.ToPtr(),
		EndedAt:                  row.EndedAt.ToPtr(),
		Phase:                    domain.RunPhase(row.Phase),
		LastHeartbeat:            row.LastHeartbeat.ToPtr(),
		ProgressPercent:          row.ProgressPercent,
		ErrorMsg:                 row.ErrorMsg,
		ApprovalState:            domain.ApprovalState(row.ApprovalState),
		FinalizationStatus:       domain.RunFinalizationStatus(row.FinalizationStatus),
		FinalizationError:        row.FinalizationError,
		FinalizedAt:              row.FinalizedAt.ToPtr(),
		ChangedFiles:             row.ChangedFiles,
		TotalSizeBytes:           row.TotalSizeBytes,
		SessionID:                row.SessionID.String,
		RunnerPID:                row.RunnerPID,
		RunnerPGID:               row.RunnerPGID,
		TranscriptPath:           row.TranscriptPath.String,
		TranscriptCursor:         row.TranscriptCursor,
		TranscriptLastSeq:        row.TranscriptLastSeq,
		SourceRunIDs:             sourceRunIDs,
		SourceInvestigationRunID: row.SourceInvestigationRunID.ToPtr(),
		ParentRunID:              row.ParentRunID.ToPtr(),
		ConversationID:           row.ConversationID.String,
		RequestedModel:           row.RequestedModel.String,
		ActualModel:              row.ActualModel.String,
		PromptPreview:            row.PromptPreview.String,
		CreatedAt:                row.CreatedAt.Time(),
		UpdatedAt:                row.UpdatedAt.Time(),
	}
	if row.ExitCode.Valid {
		exitCode := int(row.ExitCode.Int32)
		run.ExitCode = &exitCode
	}
	return run
}

func (r *runRepository) Create(ctx context.Context, run *domain.Run) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	now := time.Now()
	run.CreatedAt = now
	run.UpdatedAt = now

	row := runFromDomain(run)
	query := `INSERT INTO runs (id, task_id, agent_profile_id, tag, sandbox_id, run_mode,
			execution_mode, web_console_session_id, status,
			started_at, ended_at, phase, last_checkpoint_id, last_heartbeat, progress_percent,
			idempotency_key, summary, run_result, error_msg, exit_code, approval_state, approved_by, approved_at,
			finalization_status, finalization_error, finalized_at,
			resolved_config, diff_path, log_path, changed_files, total_size_bytes, sandbox_config, session_id,
			runner_pid, runner_pgid, transcript_path, transcript_cursor, transcript_last_seq,
			source_run_ids, source_investigation_run_id, parent_run_id, conversation_id,
			identity_token_hash, identity_token_revoked_at, custom_env, await_handle,
			last_await_key, last_await_result, last_await_resolved_at, last_wake_seq, same_key_park_streak,
			requested_model, actual_model,
			created_at, updated_at)
			VALUES (:id, :task_id, :agent_profile_id, :tag, :sandbox_id, :run_mode,
			:execution_mode, :web_console_session_id, :status,
			:started_at, :ended_at, :phase, :last_checkpoint_id, :last_heartbeat, :progress_percent,
			:idempotency_key, :summary, :run_result, :error_msg, :exit_code, :approval_state, :approved_by, :approved_at,
			:finalization_status, :finalization_error, :finalized_at,
			:resolved_config, :diff_path, :log_path, :changed_files, :total_size_bytes, :sandbox_config, :session_id,
			:runner_pid, :runner_pgid, :transcript_path, :transcript_cursor, :transcript_last_seq,
			:source_run_ids, :source_investigation_run_id, :parent_run_id, :conversation_id,
			:identity_token_hash, :identity_token_revoked_at, :custom_env, :await_handle,
			:last_await_key, :last_await_result, :last_await_resolved_at, :last_wake_seq, :same_key_park_streak,
			:requested_model, :actual_model,
			:created_at, :updated_at)`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create run")
		return wrapDBError("create", "Run", run.ID.String(), err)
	}
	return nil
}

func (r *runRepository) Get(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	query := fmt.Sprintf("SELECT %s FROM runs WHERE id = ?", runColumns)
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
		conditions = append(conditions, "runs.task_id = ?")
		args = append(args, *filter.TaskID)
	}
	if filter.AgentProfileID != nil {
		conditions = append(conditions, "runs.agent_profile_id = ?")
		args = append(args, *filter.AgentProfileID)
	}
	if filter.Status != nil {
		conditions = append(conditions, "runs.status = ?")
		args = append(args, string(*filter.Status))
	}
	if filter.TagPrefix != "" {
		conditions = append(conditions, "runs.tag LIKE ?")
		args = append(args, filter.TagPrefix+"%")
	}
	if filter.ScopePrefix != "" {
		// Enumerate by the joined task's scope_path rather than an ad-hoc tag
		// LIKE: this is what the promote-quiesce drain uses to find every run
		// targeting a scenario (scope "scenarios/<name>"). The exact-boundary
		// refinement (scenarios/foo vs scenarios/foo-bar) is the caller's job;
		// the SQL is a cheap prefix narrowing on the existing tasks JOIN.
		conditions = append(conditions, "t.scope_path LIKE ?")
		args = append(args, filter.ScopePrefix+"%")
	}
	if filter.InvestigatesRunID != nil {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM json_each(runs.source_run_ids) WHERE json_each.value = ?)")
		args = append(args, (*filter.InvestigatesRunID).String())
	}
	if filter.AppliesInvestigationRunID != nil {
		conditions = append(conditions, "runs.source_investigation_run_id = ?")
		args = append(args, *filter.AppliesInvestigationRunID)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Use pruned column set — omits heavy fields (summary, resolved_config, etc.)
	colList := strings.ReplaceAll(listRunColumns, "\n", "")
	colList = strings.ReplaceAll(colList, "\t", " ")
	// Prefix each column with "runs." for the JOIN
	var prefixed []string
	for _, col := range strings.Split(colList, ",") {
		col = strings.TrimSpace(col)
		if col != "" {
			prefixed = append(prefixed, "runs."+col)
		}
	}

	base := fmt.Sprintf(
		"SELECT %s, SUBSTR(t.description, 1, 120) AS prompt_preview FROM runs LEFT JOIN tasks t ON runs.task_id = t.id%s ORDER BY runs.created_at DESC",
		strings.Join(prefixed, ", "),
		whereClause,
	)
	queryWithPaging, pagingArgs := appendLimitOffset(base, filter.Limit, filter.Offset)
	args = append(args, pagingArgs...)
	query := queryWithPaging

	var rows []listRunLiteRow
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
			tag = :tag, sandbox_id = :sandbox_id, run_mode = :run_mode,
			execution_mode = :execution_mode, web_console_session_id = :web_console_session_id, status = :status,
		started_at = :started_at, ended_at = :ended_at, phase = :phase,
		last_checkpoint_id = :last_checkpoint_id, last_heartbeat = :last_heartbeat,
		progress_percent = :progress_percent, idempotency_key = :idempotency_key,
		summary = :summary, run_result = :run_result, error_msg = :error_msg, exit_code = :exit_code,
		approval_state = :approval_state, approved_by = :approved_by, approved_at = :approved_at,
		finalization_status = :finalization_status, finalization_error = :finalization_error, finalized_at = :finalized_at,
		resolved_config = :resolved_config, diff_path = :diff_path, log_path = :log_path,
			changed_files = :changed_files, total_size_bytes = :total_size_bytes, sandbox_config = :sandbox_config,
			session_id = :session_id, runner_pid = :runner_pid, runner_pgid = :runner_pgid,
			transcript_path = :transcript_path, transcript_cursor = :transcript_cursor, transcript_last_seq = :transcript_last_seq,
			source_run_ids = :source_run_ids,
			source_investigation_run_id = :source_investigation_run_id,
			parent_run_id = :parent_run_id, conversation_id = :conversation_id,
		identity_token_hash = :identity_token_hash, identity_token_revoked_at = :identity_token_revoked_at,
		custom_env = :custom_env, await_handle = :await_handle,
		last_await_key = :last_await_key, last_await_result = :last_await_result,
		last_await_resolved_at = :last_await_resolved_at, last_wake_seq = :last_wake_seq,
		same_key_park_streak = :same_key_park_streak,
		requested_model = :requested_model, actual_model = :actual_model,
		updated_at = :updated_at
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, row)
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	return nil
}

// TouchHeartbeat atomically updates only last_heartbeat (and updated_at) and
// only while the run is still actively executing (running or starting). The
// status guard in the WHERE clause means a heartbeat that races a park/stop
// transition is a no-op (0 rows) rather than a clobber: it can never rewrite a
// parked/terminal status back to running. Returns true when a row matched.
func (r *runRepository) TouchHeartbeat(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	const query = `UPDATE runs SET last_heartbeat = ?, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`
	hb, err := NullableTime{Time: at, Valid: true}.Value()
	if err != nil {
		return false, wrapDBError("touch_heartbeat", "Run", id.String(), err)
	}
	now, err := NullableTime{Time: time.Now(), Valid: true}.Value()
	if err != nil {
		return false, wrapDBError("touch_heartbeat", "Run", id.String(), err)
	}
	res, err := r.db.ExecContext(ctx, query, hb, now, id,
		string(domain.RunStatusRunning), string(domain.RunStatusStarting))
	if err != nil {
		return false, wrapDBError("touch_heartbeat", "Run", id.String(), err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, wrapDBError("touch_heartbeat", "Run", id.String(), err)
	}
	return affected > 0, nil
}

// UpdateRunnerStreamState persists ONLY the runner streaming columns (session
// id, runner pid/pgid, transcript path/cursor/seq) and ONLY while the run is
// still actively executing (running or starting). The status guard means an
// in-flight transcript callback (OnAdvance/OnProcessStart/OnSessionID) that
// races a park/stop/terminal transition is a no-op rather than a clobber: it
// can never rewrite a parked/terminal status back to running. Returns true when
// a row matched. The status literals mirror domain.RunStatus{Running,Starting}
// (the status column is free-text).
func (r *runRepository) UpdateRunnerStreamState(ctx context.Context, run *domain.Run) (bool, error) {
	run.UpdatedAt = time.Now()
	row := runFromDomain(run)
	const query = `UPDATE runs SET
		session_id = ?, runner_pid = ?, runner_pgid = ?,
		transcript_path = ?, transcript_cursor = ?, transcript_last_seq = ?, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`
	res, err := r.db.ExecContext(ctx, query,
		row.SessionID, row.RunnerPID, row.RunnerPGID,
		row.TranscriptPath, row.TranscriptCursor, row.TranscriptLastSeq, row.UpdatedAt,
		run.ID, string(domain.RunStatusRunning), string(domain.RunStatusStarting))
	if err != nil {
		return false, wrapDBError("update_runner_stream_state", "Run", run.ID.String(), err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, wrapDBError("update_runner_stream_state", "Run", run.ID.String(), err)
	}
	return affected > 0, nil
}

func (r *runRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Run, error) {
	query := fmt.Sprintf("SELECT %s FROM runs WHERE identity_token_hash = ?", runColumns)
	var row runRow
	if err := r.db.GetContext(ctx, &row, query, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_by_token_hash", "Run", tokenHash, err)
	}
	return row.toDomain(), nil
}

func (r *runRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM runs WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return wrapDBError("delete", "Run", id.String(), err)
	}
	return nil
}

func (r *runRepository) CountByStatus(ctx context.Context, status domain.RunStatus) (int, error) {
	query := `SELECT COUNT(*) FROM runs WHERE status = ?`
	var count int
	if err := r.db.GetContext(ctx, &count, query, string(status)); err != nil {
		return 0, wrapDBError("count_by_status", "Run", string(status), err)
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

	if payload, err := domain.DecodeEventPayload(evt.EventType, e.Data); err == nil {
		evt.Data = payload
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
	query := `SELECT COALESCE(MAX(sequence), -1) FROM run_events WHERE run_id = ?`
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

		schemaVersion := evt.SchemaVersion
		if schemaVersion == 0 {
			schemaVersion = 1
		}

		insertQuery := `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data)
			VALUES (:id, :run_id, :sequence, :event_type, :timestamp, :schema_version, :data)`

		row := struct {
			ID            uuid.UUID  `db:"id"`
			RunID         uuid.UUID  `db:"run_id"`
			Sequence      int64      `db:"sequence"`
			EventType     string     `db:"event_type"`
			Timestamp     SQLiteTime `db:"timestamp"`
			SchemaVersion int        `db:"schema_version"`
			Data          []byte     `db:"data"`
		}{
			ID:            evt.ID,
			RunID:         evt.RunID,
			Sequence:      evt.Sequence,
			EventType:     string(evt.EventType),
			Timestamp:     SQLiteTime(evt.Timestamp),
			SchemaVersion: schemaVersion,
			Data:          data,
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
	query := queryWithLimit

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
	query := base

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
	query := `SELECT COUNT(*) FROM run_events WHERE run_id = ?`
	var count int64
	if err := r.db.GetContext(ctx, &count, query, runID); err != nil {
		return 0, wrapDBError("count_events", "RunEvent", runID.String(), err)
	}
	return count, nil
}

func (r *eventRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	query := `DELETE FROM run_events WHERE run_id = ?`
	_, err := r.db.ExecContext(ctx, query, runID)
	if err != nil {
		return wrapDBError("delete_events", "RunEvent", runID.String(), err)
	}
	return nil
}
