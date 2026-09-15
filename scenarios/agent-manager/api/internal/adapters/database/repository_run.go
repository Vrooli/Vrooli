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
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
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
	LifecycleVersion               int64              `db:"lifecycle_version"`
	OwnerIdentity                  string             `db:"owner_identity"`
	OwnerEpoch                     int64              `db:"owner_epoch"`
	ID                             uuid.UUID          `db:"id"`
	TaskID                         sql.NullString     `db:"task_id"`
	AgentProfileID                 NullableUUID       `db:"agent_profile_id"`
	Tag                            string             `db:"tag"`
	Label                          string             `db:"label"`
	LabelSource                    string             `db:"label_source"`
	Subject                        sql.NullString     `db:"subject"`
	OwnerSubject                   sql.NullString     `db:"owner_subject"`
	OwnerExpiresAt                 NullableTime       `db:"owner_expires_at"`
	DispatchBinding                sql.NullString     `db:"dispatch_binding"`
	OwnerScopes                    sql.NullString     `db:"owner_scopes"`
	RequestedScopes                sql.NullString     `db:"requested_scopes"`
	WorkReferences                 sql.NullString     `db:"work_references"`
	WorkloadKind                   string             `db:"workload_kind"`
	WorkloadKey                    string             `db:"workload_key"`
	WorkloadInstance               string             `db:"workload_instance"`
	BillingSnapshot                sql.NullString     `db:"billing_snapshot"`
	SandboxID                      NullableUUID       `db:"sandbox_id"`
	RunMode                        string             `db:"run_mode"`
	ExecutionMode                  sql.NullString     `db:"execution_mode"`
	HarnessKind                    sql.NullString     `db:"harness_kind"`
	HarnessSessionID               sql.NullString     `db:"harness_session_id"`
	GoalDelivery                   sql.NullString     `db:"goal_delivery"`
	TerminalClass                  sql.NullString     `db:"terminal_class"`
	StopReason                     sql.NullString     `db:"stop_reason"`
	LastHandoff                    sql.NullString     `db:"last_handoff"`
	WebConsoleSessionID            sql.NullString     `db:"web_console_session_id"`
	Status                         string             `db:"status"`
	StartedAt                      NullableTime       `db:"started_at"`
	InteractiveInvocationStartedAt NullableTime       `db:"interactive_invocation_started_at"`
	EndedAt                        NullableTime       `db:"ended_at"`
	CancelRequestedAt              NullableTime       `db:"cancel_requested_at"`
	GoalID                         sql.NullString     `db:"goal_id"`
	Phase                          string             `db:"phase"`
	LastCheckpointID               NullableUUID       `db:"last_checkpoint_id"`
	LastHeartbeat                  NullableTime       `db:"last_heartbeat"`
	ProgressPercent                int                `db:"progress_percent"`
	IdempotencyKey                 sql.NullString     `db:"idempotency_key"`
	Summary                        NullableRunSummary `db:"summary"`
	RunResult                      NullableRunResult  `db:"run_result"`
	ErrorMsg                       string             `db:"error_msg"`
	ExitCode                       sql.NullInt32      `db:"exit_code"`
	ApprovalState                  string             `db:"approval_state"`
	ApprovedBy                     string             `db:"approved_by"`
	ApprovedAt                     NullableTime       `db:"approved_at"`
	FinalizationStatus             string             `db:"finalization_status"`
	FinalizationError              string             `db:"finalization_error"`
	FinalizedAt                    NullableTime       `db:"finalized_at"`
	ResolvedConfig                 NullableRunConfig  `db:"resolved_config"`
	DiffPath                       string             `db:"diff_path"`
	LogPath                        string             `db:"log_path"`
	ChangedFiles                   int                `db:"changed_files"`
	TotalSizeBytes                 int64              `db:"total_size_bytes"`
	// CommitHash became additive after the original runs table shipped. SQLite
	// leaves pre-migration rows NULL even though the migration declares a
	// default, so it must remain nullable at the repository boundary.
	CommitHash               sql.NullString        `db:"commit_hash"`
	SandboxConfig            NullableSandboxConfig `db:"sandbox_config"`
	SessionID                sql.NullString        `db:"session_id"`
	RunnerPID                int                   `db:"runner_pid"`
	RunnerPGID               int                   `db:"runner_pgid"`
	TranscriptPath           sql.NullString        `db:"transcript_path"`
	TranscriptCursor         int64                 `db:"transcript_cursor"`
	TranscriptLastSeq        int64                 `db:"transcript_last_seq"`
	ImportSourceHarness      sql.NullString        `db:"import_source_harness"`
	ImportSourceSessionID    sql.NullString        `db:"import_source_session_id"`
	ImportedAt               NullableTime          `db:"imported_at"`
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
	CanaryArm      sql.NullString `db:"canary_arm"`
	CreatedAt      SQLiteTime     `db:"created_at"`
	UpdatedAt      SQLiteTime     `db:"updated_at"`
}

func (row *runRow) toDomain() *domain.Run {
	sourceRunIDs := parseUUIDSliceJSON(row.SourceRunIDs)
	run := &domain.Run{
		LifecycleVersion:               row.LifecycleVersion,
		OwnerIdentity:                  row.OwnerIdentity,
		OwnerEpoch:                     row.OwnerEpoch,
		ID:                             row.ID,
		TaskID:                         parseNullableUUID(row.TaskID),
		AgentProfileID:                 row.AgentProfileID.ToPtr(),
		Tag:                            row.Tag,
		Label:                          row.Label,
		LabelSource:                    domain.RunLabelSource(row.LabelSource),
		Subject:                        parseStringSliceJSON(row.Subject),
		OwnerSubject:                   row.OwnerSubject.String,
		OwnerExpiresAt:                 row.OwnerExpiresAt.ToPtr(),
		DispatchBinding:                parseDispatchBinding(row.DispatchBinding),
		OwnerScopes:                    parseStringSliceJSON(row.OwnerScopes),
		RequestedScopes:                parseStringSliceJSON(row.RequestedScopes),
		WorkReferences:                 parseWorkReferencesJSON(row.WorkReferences),
		Workload:                       domain.WorkloadRef{Kind: domain.WorkloadKind(row.WorkloadKind), Key: row.WorkloadKey, Instance: row.WorkloadInstance},
		Billing:                        decodeBillingSnapshot(row.BillingSnapshot),
		SandboxID:                      row.SandboxID.ToPtr(),
		RunMode:                        domain.RunMode(row.RunMode),
		ExecutionMode:                  domain.ExecutionMode(row.ExecutionMode.String).Normalized(),
		HarnessKind:                    row.HarnessKind.String,
		HarnessSessionID:               row.HarnessSessionID.String,
		GoalDelivery:                   row.GoalDelivery.String,
		TerminalClass:                  domain.RunTerminalClass(row.TerminalClass.String),
		StopReason:                     domain.RunStopReason(row.StopReason.String),
		LastHandoff:                    row.LastHandoff.String,
		WebConsoleSessionID:            row.WebConsoleSessionID.String,
		Status:                         domain.RunStatus(row.Status),
		StartedAt:                      row.StartedAt.ToPtr(),
		InteractiveInvocationStartedAt: row.InteractiveInvocationStartedAt.ToPtr(),
		EndedAt:                        row.EndedAt.ToPtr(),
		CancelRequestedAt:              row.CancelRequestedAt.ToPtr(),
		GoalID:                         row.GoalID.String,
		Phase:                          domain.RunPhase(row.Phase),
		LastCheckpointID:               row.LastCheckpointID.ToPtr(),
		LastHeartbeat:                  row.LastHeartbeat.ToPtr(),
		ProgressPercent:                row.ProgressPercent,
		IdempotencyKey:                 row.IdempotencyKey.String, // Empty string if NULL
		Summary:                        row.Summary.V,
		Result:                         row.RunResult.V,
		ErrorMsg:                       row.ErrorMsg,
		ApprovalState:                  domain.ApprovalState(row.ApprovalState),
		ApprovedBy:                     row.ApprovedBy,
		ApprovedAt:                     row.ApprovedAt.ToPtr(),
		FinalizationStatus:             domain.RunFinalizationStatus(row.FinalizationStatus),
		FinalizationError:              row.FinalizationError,
		FinalizedAt:                    row.FinalizedAt.ToPtr(),
		ResolvedConfig:                 row.ResolvedConfig.V,
		DiffPath:                       row.DiffPath,
		LogPath:                        row.LogPath,
		ChangedFiles:                   row.ChangedFiles,
		TotalSizeBytes:                 row.TotalSizeBytes,
		CommitHash:                     row.CommitHash.String,
		SandboxConfig:                  row.SandboxConfig.V,
		SessionID:                      row.SessionID.String,
		RunnerPID:                      row.RunnerPID,
		RunnerPGID:                     row.RunnerPGID,
		TranscriptPath:                 row.TranscriptPath.String,
		TranscriptCursor:               row.TranscriptCursor,
		TranscriptLastSeq:              row.TranscriptLastSeq,
		ImportSourceHarness:            row.ImportSourceHarness.String,
		ImportSourceSessionID:          row.ImportSourceSessionID.String,
		ImportedAt:                     row.ImportedAt.ToPtr(),
		SourceRunIDs:                   sourceRunIDs,
		SourceInvestigationRunID:       row.SourceInvestigationRunID.ToPtr(),
		ParentRunID:                    row.ParentRunID.ToPtr(),
		ConversationID:                 row.ConversationID.String,
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
		CanaryArm:      row.CanaryArm.String,
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
		ID:                             r.ID,
		TaskID:                         nullableUUID(r.TaskID),
		AgentProfileID:                 NewNullableUUID(r.AgentProfileID),
		Tag:                            r.Tag,
		Label:                          r.Label,
		LabelSource:                    string(r.LabelSource),
		Subject:                        sql.NullString{String: marshalStringSliceJSON(r.Subject), Valid: true},
		OwnerSubject:                   sql.NullString{String: r.OwnerSubject, Valid: true},
		OwnerExpiresAt:                 NewNullableTime(r.OwnerExpiresAt),
		DispatchBinding:                marshalDispatchBinding(r.DispatchBinding),
		OwnerScopes:                    sql.NullString{String: marshalStringSliceJSON(r.OwnerScopes), Valid: true},
		RequestedScopes:                sql.NullString{String: marshalRequestedScopesJSON(r.RequestedScopes), Valid: true},
		WorkReferences:                 sql.NullString{String: marshalWorkReferencesJSON(r.WorkReferences), Valid: true},
		WorkloadKind:                   string(r.Workload.Kind),
		WorkloadKey:                    r.Workload.Key,
		WorkloadInstance:               r.Workload.Instance,
		BillingSnapshot:                sql.NullString{String: marshalBillingSnapshot(r.Billing), Valid: true},
		SandboxID:                      NewNullableUUID(r.SandboxID),
		RunMode:                        string(r.RunMode),
		ExecutionMode:                  sql.NullString{String: string(r.ExecutionMode.Normalized()), Valid: true},
		HarnessKind:                    sql.NullString{String: r.HarnessKind, Valid: r.HarnessKind != ""},
		HarnessSessionID:               sql.NullString{String: r.HarnessSessionID, Valid: r.HarnessSessionID != ""},
		GoalDelivery:                   sql.NullString{String: r.GoalDelivery, Valid: r.GoalDelivery != ""},
		TerminalClass:                  sql.NullString{String: string(r.TerminalClass), Valid: r.TerminalClass != ""},
		StopReason:                     sql.NullString{String: string(r.StopReason), Valid: r.StopReason != ""},
		LastHandoff:                    sql.NullString{String: r.LastHandoff, Valid: r.LastHandoff != ""},
		WebConsoleSessionID:            sql.NullString{String: r.WebConsoleSessionID, Valid: r.WebConsoleSessionID != ""},
		Status:                         string(r.Status),
		LifecycleVersion:               r.LifecycleVersion,
		OwnerIdentity:                  r.OwnerIdentity,
		OwnerEpoch:                     r.OwnerEpoch,
		StartedAt:                      NewNullableTime(r.StartedAt),
		InteractiveInvocationStartedAt: NewNullableTime(r.InteractiveInvocationStartedAt),
		EndedAt:                        NewNullableTime(r.EndedAt),
		CancelRequestedAt:              NewNullableTime(r.CancelRequestedAt),
		GoalID:                         sql.NullString{String: r.GoalID, Valid: true},
		Phase:                          string(r.Phase),
		LastCheckpointID:               NewNullableUUID(r.LastCheckpointID),
		LastHeartbeat:                  NewNullableTime(r.LastHeartbeat),
		ProgressPercent:                r.ProgressPercent,
		IdempotencyKey:                 sql.NullString{String: r.IdempotencyKey, Valid: r.IdempotencyKey != ""},
		Summary:                        NullableRunSummary{V: r.Summary},
		RunResult:                      NullableRunResult{V: r.Result},
		ErrorMsg:                       r.ErrorMsg,
		ApprovalState:                  string(r.ApprovalState),
		ApprovedBy:                     r.ApprovedBy,
		ApprovedAt:                     NewNullableTime(r.ApprovedAt),
		FinalizationStatus:             string(finalizationStatus),
		FinalizationError:              r.FinalizationError,
		FinalizedAt:                    NewNullableTime(r.FinalizedAt),
		ResolvedConfig:                 NullableRunConfig{V: r.ResolvedConfig},
		DiffPath:                       r.DiffPath,
		LogPath:                        r.LogPath,
		ChangedFiles:                   r.ChangedFiles,
		TotalSizeBytes:                 r.TotalSizeBytes,
		CommitHash:                     sql.NullString{String: r.CommitHash, Valid: r.CommitHash != ""},
		SandboxConfig:                  NullableSandboxConfig{V: r.SandboxConfig},
		SessionID:                      sql.NullString{String: r.SessionID, Valid: r.SessionID != ""},
		RunnerPID:                      r.RunnerPID,
		RunnerPGID:                     r.RunnerPGID,
		TranscriptPath:                 sql.NullString{String: r.TranscriptPath, Valid: r.TranscriptPath != ""},
		TranscriptCursor:               r.TranscriptCursor,
		TranscriptLastSeq:              r.TranscriptLastSeq,
		ImportSourceHarness:            sql.NullString{String: r.ImportSourceHarness, Valid: r.ImportSourceHarness != ""},
		ImportSourceSessionID:          sql.NullString{String: r.ImportSourceSessionID, Valid: r.ImportSourceSessionID != ""},
		ImportedAt:                     NewNullableTime(r.ImportedAt),
		SourceRunIDs:                   sql.NullString{String: sourceRunIDs, Valid: sourceRunIDs != ""},
		SourceInvestigationRunID:       NewNullableUUID(r.SourceInvestigationRunID),
		ParentRunID:                    NewNullableUUID(r.ParentRunID),
		ConversationID:                 sql.NullString{String: r.ConversationID, Valid: r.ConversationID != ""},
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
		// The additive migration makes canary_arm NOT NULL. Bind the empty
		// value explicitly for historical/imported runs instead of sending
		// SQL NULL and making every external transcript adoption fail.
		CanaryArm: sql.NullString{String: r.CanaryArm, Valid: true},
		CreatedAt: SQLiteTime(r.CreatedAt),
		UpdatedAt: SQLiteTime(r.UpdatedAt),
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

func parseNullableUUID(raw sql.NullString) uuid.UUID {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(raw.String)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func nullableUUID(id uuid.UUID) sql.NullString {
	if id == uuid.Nil {
		return sql.NullString{}
	}
	return sql.NullString{String: id.String(), Valid: true}
}

func parseStringSliceJSON(raw sql.NullString) []string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw.String), &values); err != nil {
		return nil
	}
	return values
}

func parseWorkReferencesJSON(raw sql.NullString) []*eventdomain.WorkReference {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var values []*eventdomain.WorkReference
	if err := json.Unmarshal([]byte(raw.String), &values); err != nil {
		return nil
	}
	return values
}

func marshalWorkReferencesJSON(values []*eventdomain.WorkReference) string {
	if len(values) == 0 {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func marshalStringSliceJSON(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// Preserve absent versus explicitly empty narrowing across durable run reads.
func marshalRequestedScopesJSON(values []string) string {
	if values == nil {
		return "null"
	}
	return marshalStringSliceJSON(values)
}

func parseDispatchBinding(raw sql.NullString) *domain.DispatchBinding {
	if !raw.Valid || raw.String == "" || raw.String == "null" {
		return nil
	}
	var binding domain.DispatchBinding
	if json.Unmarshal([]byte(raw.String), &binding) != nil {
		return &domain.DispatchBinding{}
	}
	return &binding
}

func marshalDispatchBinding(binding *domain.DispatchBinding) sql.NullString {
	if binding == nil {
		return sql.NullString{}
	}
	raw, _ := json.Marshal(binding)
	return sql.NullString{String: string(raw), Valid: true}
}

func marshalBillingSnapshot(value domain.BillingSnapshot) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func decodeBillingSnapshot(raw sql.NullString) domain.BillingSnapshot {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return domain.BillingSnapshot{Mode: domain.BillingModeUnknown}
	}
	var value domain.BillingSnapshot
	if err := json.Unmarshal([]byte(raw.String), &value); err != nil || value.Mode == "" {
		value.Mode = domain.BillingModeUnknown
	}
	return value
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

const runColumns = `id, lifecycle_version, owner_identity, owner_epoch, task_id, agent_profile_id, tag, label, label_source, subject, owner_subject, owner_expires_at, dispatch_binding, owner_scopes, requested_scopes, work_references, workload_kind, workload_key, workload_instance, billing_snapshot, sandbox_id, run_mode,
	execution_mode, harness_kind, harness_session_id, goal_delivery, terminal_class, stop_reason, last_handoff, web_console_session_id, status,
	started_at, interactive_invocation_started_at, ended_at, cancel_requested_at, goal_id, phase, last_checkpoint_id, last_heartbeat, progress_percent,
	idempotency_key, summary, run_result, error_msg, exit_code, approval_state, approved_by, approved_at,
	finalization_status, finalization_error, finalized_at,
	resolved_config, diff_path, log_path, changed_files, total_size_bytes, commit_hash, sandbox_config, session_id,
	runner_pid, runner_pgid, transcript_path, transcript_cursor, transcript_last_seq, import_source_harness, import_source_session_id, imported_at,
	source_run_ids, source_investigation_run_id, parent_run_id, conversation_id,
	identity_token_hash, identity_token_revoked_at, custom_env, await_handle,
	last_await_key, last_await_result, last_await_resolved_at, last_wake_seq, same_key_park_streak,
	requested_model, actual_model, canary_arm,
	created_at, updated_at`

// listRunColumns contains the pruned column set for List() queries.
// Omits heavy fields: summary, resolved_config, sandbox_config, sandbox_id,
// idempotency_key, last_checkpoint_id, diff_path, log_path,
// approved_by, approved_at.
// NOTE: last_heartbeat MUST be included — the reconciler depends on it
// to detect stale runs. Without it, every run appears stale after creation.
const listRunColumns = `id, lifecycle_version, owner_identity, owner_epoch, task_id, agent_profile_id, tag, label, label_source, subject, owner_subject, owner_expires_at, dispatch_binding, owner_scopes, requested_scopes, work_references, workload_kind, workload_key, workload_instance, billing_snapshot, run_mode,
	execution_mode, harness_kind, harness_session_id, goal_delivery, terminal_class, stop_reason, last_handoff, web_console_session_id, status,
	started_at, interactive_invocation_started_at, ended_at, phase, last_heartbeat, progress_percent,
	error_msg, exit_code, approval_state, finalization_status, finalization_error, finalized_at,
	changed_files, total_size_bytes, commit_hash, session_id,
	runner_pid, runner_pgid, transcript_path, transcript_cursor, transcript_last_seq, import_source_harness, import_source_session_id, imported_at,
	source_run_ids, source_investigation_run_id, parent_run_id, conversation_id,
	requested_model, actual_model, canary_arm,
	created_at, updated_at`

// listRunLiteRow is the database row representation for the pruned list query.
type listRunLiteRow struct {
	LifecycleVersion               int64          `db:"lifecycle_version"`
	OwnerIdentity                  string         `db:"owner_identity"`
	OwnerEpoch                     int64          `db:"owner_epoch"`
	ID                             uuid.UUID      `db:"id"`
	TaskID                         sql.NullString `db:"task_id"`
	AgentProfileID                 NullableUUID   `db:"agent_profile_id"`
	Tag                            string         `db:"tag"`
	Label                          string         `db:"label"`
	LabelSource                    string         `db:"label_source"`
	Subject                        sql.NullString `db:"subject"`
	OwnerSubject                   sql.NullString `db:"owner_subject"`
	OwnerExpiresAt                 NullableTime   `db:"owner_expires_at"`
	DispatchBinding                sql.NullString `db:"dispatch_binding"`
	OwnerScopes                    sql.NullString `db:"owner_scopes"`
	RequestedScopes                sql.NullString `db:"requested_scopes"`
	WorkReferences                 sql.NullString `db:"work_references"`
	WorkloadKind                   string         `db:"workload_kind"`
	WorkloadKey                    string         `db:"workload_key"`
	WorkloadInstance               string         `db:"workload_instance"`
	BillingSnapshot                sql.NullString `db:"billing_snapshot"`
	RunMode                        string         `db:"run_mode"`
	ExecutionMode                  sql.NullString `db:"execution_mode"`
	HarnessKind                    sql.NullString `db:"harness_kind"`
	HarnessSessionID               sql.NullString `db:"harness_session_id"`
	GoalDelivery                   sql.NullString `db:"goal_delivery"`
	TerminalClass                  sql.NullString `db:"terminal_class"`
	StopReason                     sql.NullString `db:"stop_reason"`
	LastHandoff                    sql.NullString `db:"last_handoff"`
	WebConsoleSessionID            sql.NullString `db:"web_console_session_id"`
	Status                         string         `db:"status"`
	StartedAt                      NullableTime   `db:"started_at"`
	InteractiveInvocationStartedAt NullableTime   `db:"interactive_invocation_started_at"`
	EndedAt                        NullableTime   `db:"ended_at"`
	Phase                          string         `db:"phase"`
	LastHeartbeat                  NullableTime   `db:"last_heartbeat"`
	ProgressPercent                int            `db:"progress_percent"`
	ErrorMsg                       string         `db:"error_msg"`
	ExitCode                       sql.NullInt32  `db:"exit_code"`
	ApprovalState                  string         `db:"approval_state"`
	FinalizationStatus             string         `db:"finalization_status"`
	FinalizationError              string         `db:"finalization_error"`
	FinalizedAt                    NullableTime   `db:"finalized_at"`
	ChangedFiles                   int            `db:"changed_files"`
	TotalSizeBytes                 int64          `db:"total_size_bytes"`
	// See runRow.CommitHash: legacy rows can contain NULL after the additive
	// migration, while the domain contract represents no commit as an empty
	// string.
	CommitHash               sql.NullString `db:"commit_hash"`
	SessionID                sql.NullString `db:"session_id"`
	RunnerPID                int            `db:"runner_pid"`
	RunnerPGID               int            `db:"runner_pgid"`
	TranscriptPath           sql.NullString `db:"transcript_path"`
	TranscriptCursor         int64          `db:"transcript_cursor"`
	TranscriptLastSeq        int64          `db:"transcript_last_seq"`
	ImportSourceHarness      sql.NullString `db:"import_source_harness"`
	ImportSourceSessionID    sql.NullString `db:"import_source_session_id"`
	ImportedAt               NullableTime   `db:"imported_at"`
	SourceRunIDs             sql.NullString `db:"source_run_ids"`
	SourceInvestigationRunID NullableUUID   `db:"source_investigation_run_id"`
	ParentRunID              NullableUUID   `db:"parent_run_id"`
	ConversationID           sql.NullString `db:"conversation_id"`
	RequestedModel           sql.NullString `db:"requested_model"`
	ActualModel              sql.NullString `db:"actual_model"`
	CanaryArm                sql.NullString `db:"canary_arm"`
	CreatedAt                SQLiteTime     `db:"created_at"`
	UpdatedAt                SQLiteTime     `db:"updated_at"`
	// Computed field from JOIN
	PromptPreview sql.NullString `db:"prompt_preview"`
}

func (row *listRunLiteRow) toDomain() *domain.Run {
	sourceRunIDs := parseUUIDSliceJSON(row.SourceRunIDs)
	run := &domain.Run{
		LifecycleVersion:               row.LifecycleVersion,
		OwnerIdentity:                  row.OwnerIdentity,
		OwnerEpoch:                     row.OwnerEpoch,
		ID:                             row.ID,
		TaskID:                         parseNullableUUID(row.TaskID),
		AgentProfileID:                 row.AgentProfileID.ToPtr(),
		Tag:                            row.Tag,
		Label:                          row.Label,
		LabelSource:                    domain.RunLabelSource(row.LabelSource),
		Subject:                        parseStringSliceJSON(row.Subject),
		OwnerSubject:                   row.OwnerSubject.String,
		OwnerScopes:                    parseStringSliceJSON(row.OwnerScopes),
		OwnerExpiresAt:                 row.OwnerExpiresAt.ToPtr(),
		DispatchBinding:                parseDispatchBinding(row.DispatchBinding),
		RequestedScopes:                parseStringSliceJSON(row.RequestedScopes),
		WorkReferences:                 parseWorkReferencesJSON(row.WorkReferences),
		Workload:                       domain.WorkloadRef{Kind: domain.WorkloadKind(row.WorkloadKind), Key: row.WorkloadKey, Instance: row.WorkloadInstance},
		Billing:                        decodeBillingSnapshot(row.BillingSnapshot),
		RunMode:                        domain.RunMode(row.RunMode),
		ExecutionMode:                  domain.ExecutionMode(row.ExecutionMode.String).Normalized(),
		HarnessKind:                    row.HarnessKind.String,
		HarnessSessionID:               row.HarnessSessionID.String,
		GoalDelivery:                   row.GoalDelivery.String,
		TerminalClass:                  domain.RunTerminalClass(row.TerminalClass.String),
		StopReason:                     domain.RunStopReason(row.StopReason.String),
		LastHandoff:                    row.LastHandoff.String,
		WebConsoleSessionID:            row.WebConsoleSessionID.String,
		Status:                         domain.RunStatus(row.Status),
		StartedAt:                      row.StartedAt.ToPtr(),
		InteractiveInvocationStartedAt: row.InteractiveInvocationStartedAt.ToPtr(),
		EndedAt:                        row.EndedAt.ToPtr(),
		Phase:                          domain.RunPhase(row.Phase),
		LastHeartbeat:                  row.LastHeartbeat.ToPtr(),
		ProgressPercent:                row.ProgressPercent,
		ErrorMsg:                       row.ErrorMsg,
		ApprovalState:                  domain.ApprovalState(row.ApprovalState),
		FinalizationStatus:             domain.RunFinalizationStatus(row.FinalizationStatus),
		FinalizationError:              row.FinalizationError,
		FinalizedAt:                    row.FinalizedAt.ToPtr(),
		ChangedFiles:                   row.ChangedFiles,
		TotalSizeBytes:                 row.TotalSizeBytes,
		CommitHash:                     row.CommitHash.String,
		SessionID:                      row.SessionID.String,
		RunnerPID:                      row.RunnerPID,
		RunnerPGID:                     row.RunnerPGID,
		TranscriptPath:                 row.TranscriptPath.String,
		TranscriptCursor:               row.TranscriptCursor,
		TranscriptLastSeq:              row.TranscriptLastSeq,
		ImportSourceHarness:            row.ImportSourceHarness.String,
		ImportSourceSessionID:          row.ImportSourceSessionID.String,
		ImportedAt:                     row.ImportedAt.ToPtr(),
		SourceRunIDs:                   sourceRunIDs,
		SourceInvestigationRunID:       row.SourceInvestigationRunID.ToPtr(),
		ParentRunID:                    row.ParentRunID.ToPtr(),
		ConversationID:                 row.ConversationID.String,
		RequestedModel:                 row.RequestedModel.String,
		ActualModel:                    row.ActualModel.String,
		CanaryArm:                      row.CanaryArm.String,
		PromptPreview:                  row.PromptPreview.String,
		CreatedAt:                      row.CreatedAt.Time(),
		UpdatedAt:                      row.UpdatedAt.Time(),
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
	query := `INSERT INTO runs (id, lifecycle_version, owner_identity, owner_epoch, task_id, agent_profile_id, tag, label, label_source, subject, owner_subject, owner_expires_at, dispatch_binding, owner_scopes, requested_scopes, work_references, workload_kind, workload_key, workload_instance, billing_snapshot, sandbox_id, run_mode,
			execution_mode, harness_kind, harness_session_id, goal_delivery, terminal_class, stop_reason, last_handoff, web_console_session_id, status,
			started_at, interactive_invocation_started_at, ended_at, cancel_requested_at, goal_id, phase, last_checkpoint_id, last_heartbeat, progress_percent,
			idempotency_key, summary, run_result, error_msg, exit_code, approval_state, approved_by, approved_at,
			finalization_status, finalization_error, finalized_at,
			resolved_config, diff_path, log_path, changed_files, total_size_bytes, commit_hash, sandbox_config, session_id,
			runner_pid, runner_pgid, transcript_path, transcript_cursor, transcript_last_seq, import_source_harness, import_source_session_id, imported_at,
			source_run_ids, source_investigation_run_id, parent_run_id, conversation_id,
			identity_token_hash, identity_token_revoked_at, custom_env, await_handle,
			last_await_key, last_await_result, last_await_resolved_at, last_wake_seq, same_key_park_streak,
			requested_model, actual_model, canary_arm,
			created_at, updated_at)
			VALUES (:id, :lifecycle_version, :owner_identity, :owner_epoch, :task_id, :agent_profile_id, :tag, :label, :label_source, :subject, :owner_subject, :owner_expires_at, :dispatch_binding, :owner_scopes, :requested_scopes, :work_references, :workload_kind, :workload_key, :workload_instance, :billing_snapshot, :sandbox_id, :run_mode,
			:execution_mode, :harness_kind, :harness_session_id, :goal_delivery, :terminal_class, :stop_reason, :last_handoff, :web_console_session_id, :status,
			:started_at, :interactive_invocation_started_at, :ended_at, :cancel_requested_at, :goal_id, :phase, :last_checkpoint_id, :last_heartbeat, :progress_percent,
			:idempotency_key, :summary, :run_result, :error_msg, :exit_code, :approval_state, :approved_by, :approved_at,
			:finalization_status, :finalization_error, :finalized_at,
			:resolved_config, :diff_path, :log_path, :changed_files, :total_size_bytes, :commit_hash, :sandbox_config, :session_id,
			:runner_pid, :runner_pgid, :transcript_path, :transcript_cursor, :transcript_last_seq, :import_source_harness, :import_source_session_id, :imported_at,
			:source_run_ids, :source_investigation_run_id, :parent_run_id, :conversation_id,
			:identity_token_hash, :identity_token_revoked_at, :custom_env, :await_handle,
			:last_await_key, :last_await_result, :last_await_resolved_at, :last_wake_seq, :same_key_park_streak,
			:requested_model, :actual_model, :canary_arm,
			:created_at, :updated_at)`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("create", "Run", run.ID.String(), err)
	}
	defer tx.Rollback()
	_, err = tx.NamedExecContext(ctx, query, row)
	if err != nil {
		r.log.WithError(err).Error("Failed to create run")
		return wrapDBError("create", "Run", run.ID.String(), err)
	}
	if run.IdempotencyKey != "" {
		_, err = tx.ExecContext(ctx, `INSERT INTO run_creation_receipts (idempotency_key, run_id, task_id, owner_subject, created_at) VALUES (?, ?, ?, ?, ?)`, run.IdempotencyKey, run.ID, row.TaskID, row.OwnerSubject, row.CreatedAt)
		if err != nil {
			return wrapDBError("record_creation", "Run", run.ID.String(), err)
		}
	}
	return tx.Commit()
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
	if filter.EndedFrom != nil {
		conditions = append(conditions, "runs.ended_at >= ?")
		args = append(args, SQLiteTime(*filter.EndedFrom))
	}
	if filter.EndedTo != nil {
		conditions = append(conditions, "runs.ended_at < ?")
		args = append(args, SQLiteTime(*filter.EndedTo))
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

	query := `UPDATE runs SET lifecycle_version = lifecycle_version + CASE WHEN status = :status THEN 0 ELSE 1 END,
			owner_identity = CASE WHEN :owner_identity <> '' THEN :owner_identity ELSE owner_identity END,
			owner_epoch = CASE WHEN :owner_epoch > 0 THEN :owner_epoch ELSE owner_epoch END,
			task_id = :task_id, agent_profile_id = :agent_profile_id,
			tag = :tag, label = :label, label_source = :label_source, subject = :subject, owner_subject = :owner_subject, owner_expires_at = :owner_expires_at, dispatch_binding = :dispatch_binding, owner_scopes = :owner_scopes, requested_scopes = :requested_scopes, work_references = :work_references, sandbox_id = :sandbox_id, run_mode = :run_mode,
			execution_mode = :execution_mode, harness_kind = :harness_kind, harness_session_id = :harness_session_id, goal_delivery = :goal_delivery, terminal_class = :terminal_class, stop_reason = :stop_reason, last_handoff = :last_handoff, web_console_session_id = :web_console_session_id, status = :status,
		started_at = :started_at, interactive_invocation_started_at = :interactive_invocation_started_at, ended_at = :ended_at, cancel_requested_at = COALESCE(cancel_requested_at, :cancel_requested_at), phase = :phase,
		last_checkpoint_id = :last_checkpoint_id, last_heartbeat = CASE WHEN last_heartbeat IS NULL OR last_heartbeat < :last_heartbeat THEN :last_heartbeat ELSE last_heartbeat END,
		progress_percent = :progress_percent, idempotency_key = :idempotency_key,
		summary = :summary, run_result = :run_result, error_msg = :error_msg, exit_code = :exit_code,
		approval_state = :approval_state, approved_by = :approved_by, approved_at = :approved_at,
		finalization_status = :finalization_status, finalization_error = :finalization_error, finalized_at = :finalized_at,
		resolved_config = :resolved_config, diff_path = :diff_path, log_path = :log_path,
			changed_files = :changed_files, total_size_bytes = :total_size_bytes, commit_hash = :commit_hash, sandbox_config = :sandbox_config,
			session_id = :session_id, runner_pid = :runner_pid, runner_pgid = :runner_pgid,
			transcript_path = :transcript_path, transcript_cursor = :transcript_cursor, transcript_last_seq = :transcript_last_seq,
			import_source_harness = :import_source_harness, import_source_session_id = :import_source_session_id, imported_at = :imported_at,
			source_run_ids = :source_run_ids,
			source_investigation_run_id = :source_investigation_run_id,
			parent_run_id = :parent_run_id, conversation_id = :conversation_id,
		identity_token_hash = :identity_token_hash, identity_token_revoked_at = :identity_token_revoked_at,
		custom_env = :custom_env, await_handle = :await_handle,
		last_await_key = :last_await_key, last_await_result = :last_await_result,
		last_await_resolved_at = :last_await_resolved_at, last_wake_seq = :last_wake_seq,
		same_key_park_streak = :same_key_park_streak,
		requested_model = :requested_model, actual_model = :actual_model, canary_arm = :canary_arm,
		updated_at = :updated_at
		WHERE id = :id AND lifecycle_version = :lifecycle_version
		  AND (owner_epoch = :owner_epoch OR (owner_epoch = 0 AND :owner_epoch = 1))
		  AND (fresh_recovery_request_hash = '' OR status = :status)
		RETURNING lifecycle_version`

	bound, args, err := sqlx.Named(query, row)
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	err = r.db.QueryRowContext(ctx, bound, args...).Scan(&run.LifecycleVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.NewStateErrorWithID("Run", run.ID.String(), "superseded", "update", "run lifecycle changed or was consumed by fresh recovery; writer refused")
	}
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	return nil
}

// UpdateRunLabel updates only the two label columns. It is used by the
// historical transcript backfill and deliberately cannot mutate run identity,
// attribution, status, or timestamps.
func (r *runRepository) UpdateRunLabel(ctx context.Context, id uuid.UUID, label string, source domain.RunLabelSource) error {
	label = strings.TrimSpace(label)
	if label == "" {
		return fmt.Errorf("run label is required")
	}
	if source == "" {
		return fmt.Errorf("run label source is required")
	}
	const query = `UPDATE runs SET label = ?, label_source = ? WHERE id = ?`
	if _, err := r.db.ExecContext(ctx, query, label, string(source), id); err != nil {
		return wrapDBError("update_label", "Run", id.String(), err)
	}
	return nil
}

// UpdateRunSubject updates only the derived subject projection.
func (r *runRepository) UpdateRunSubject(ctx context.Context, id uuid.UUID, subject []string) error {
	encoded, err := json.Marshal(subject)
	if err != nil {
		return fmt.Errorf("marshal run subject: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE runs SET subject = ? WHERE id = ?`, string(encoded), id); err != nil {
		return wrapDBError("update_subject", "Run", id.String(), err)
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

// RequestCancellation atomically stamps the durable cancellation intent
// (cancel_requested_at) on a run that is still actively executing. It returns
// true when the intent was newly recorded and false when it was already stamped
// or the run is no longer active. The status guard is applied in SQL using the
// same race discipline as TouchHeartbeat: a stale caller cannot stamp a run
// that has already reached a terminal state. The intent is monotonic — a repeat
// stop request is a no-op rather than a second terminal write.
func (r *runRepository) RequestCancellation(ctx context.Context, id uuid.UUID, at time.Time) (bool, error) {
	const query = `UPDATE runs SET cancel_requested_at = ?, updated_at = ?
		WHERE id = ? AND cancel_requested_at IS NULL AND status IN (?, ?)`
	stamp, err := NullableTime{Time: at, Valid: true}.Value()
	if err != nil {
		return false, wrapDBError("request_cancellation", "Run", id.String(), err)
	}
	now, err := NullableTime{Time: time.Now(), Valid: true}.Value()
	if err != nil {
		return false, wrapDBError("request_cancellation", "Run", id.String(), err)
	}
	res, err := r.db.ExecContext(ctx, query, stamp, now, id,
		string(domain.RunStatusRunning), string(domain.RunStatusStarting))
	if err != nil {
		return false, wrapDBError("request_cancellation", "Run", id.String(), err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, wrapDBError("request_cancellation", "Run", id.String(), err)
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
		transcript_path = ?, transcript_cursor = MAX(transcript_cursor, ?), transcript_last_seq = MAX(transcript_last_seq, ?), updated_at = ?
		WHERE id = ? AND status IN (?, ?) AND lifecycle_version = ?
		  AND owner_epoch = ?`
	res, err := r.db.ExecContext(ctx, query,
		row.SessionID, row.RunnerPID, row.RunnerPGID,
		row.TranscriptPath, row.TranscriptCursor, row.TranscriptLastSeq, row.UpdatedAt,
		run.ID, string(domain.RunStatusRunning), string(domain.RunStatusStarting), run.LifecycleVersion, run.OwnerEpoch)
	if err != nil {
		return false, wrapDBError("update_runner_stream_state", "Run", run.ID.String(), err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, wrapDBError("update_runner_stream_state", "Run", run.ID.String(), err)
	}
	return affected > 0, nil
}

// AttachRecovery repairs legacy current-turn terminal fields and records the
// actual attachment heartbeat. Neither liveness nor attachment proves progress:
// phase, result, accounting, start time and transcript columns stay untouched.
func (r *runRepository) AttachRecovery(ctx context.Context, id uuid.UUID, lifecycleVersion int64, at time.Time) (bool, error) {
	const query = `UPDATE runs SET ended_at = NULL, error_msg = '', exit_code = NULL,
		terminal_class = '', stop_reason = '',
		last_heartbeat = CASE WHEN last_heartbeat IS NULL OR last_heartbeat < ? THEN ? ELSE last_heartbeat END,
		updated_at = ?
		WHERE id = ? AND lifecycle_version = ? AND status IN (?, ?) AND cancel_requested_at IS NULL`
	stamp := NewNullableTime(&at)
	res, err := r.db.ExecContext(ctx, query, stamp, stamp, SQLiteTime(time.Now().UTC()), id, lifecycleVersion,
		string(domain.RunStatusRunning), string(domain.RunStatusStarting))
	if err != nil {
		return false, wrapDBError("attach_recovery", "Run", id.String(), err)
	}
	count, err := res.RowsAffected()
	if err != nil {
		return false, wrapDBError("attach_recovery", "Run", id.String(), err)
	}
	return count == 1, nil
}

// ClearTerminalRunnerProcessIdentity retires only the process identity from a
// terminal run. Transcript/session identity and all outcome evidence remain
// intact. The exact PID/PGID and owner epoch fence a stale observation from
// clearing a replacement executor that raced the recovery check.
func (r *runRepository) ClearTerminalRunnerProcessIdentity(ctx context.Context, id uuid.UUID, lifecycleVersion, ownerEpoch, runnerPID, runnerPGID int64) (bool, error) {
	const query = `UPDATE runs SET runner_pid = 0, runner_pgid = 0, updated_at = ?
		WHERE id = ? AND lifecycle_version = ? AND owner_epoch = ?
		  AND runner_pid = ? AND runner_pgid = ?
		  AND status IN (?, ?, ?)`
	res, err := r.db.ExecContext(ctx, query, SQLiteTime(time.Now().UTC()), id, lifecycleVersion, ownerEpoch,
		runnerPID, runnerPGID, string(domain.RunStatusComplete), string(domain.RunStatusFailed), string(domain.RunStatusCancelled))
	if err != nil {
		return false, wrapDBError("clear_terminal_runner_process_identity", "Run", id.String(), err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, wrapDBError("clear_terminal_runner_process_identity", "Run", id.String(), err)
	}
	return affected == 1, nil
}

// ClaimRecoveryOwnership advances the durable owner epoch before recovery
// attaches a transcript tailer. A previous owner may still have a stale
// in-memory snapshot, but every guarded writer using its old epoch will now
// be rejected by the database.
func (r *runRepository) ClaimRecoveryOwnership(ctx context.Context, id uuid.UUID, lifecycleVersion int64, ownerIdentity string, at time.Time) (int64, bool, error) {
	ownerIdentity = strings.TrimSpace(ownerIdentity)
	if ownerIdentity == "" {
		return 0, false, domain.NewValidationError("ownerIdentity", "owner identity is required")
	}
	const query = `UPDATE runs SET owner_identity = ?, owner_epoch = owner_epoch + 1,
		ended_at = NULL, error_msg = '', exit_code = NULL, terminal_class = '', stop_reason = '',
		last_heartbeat = CASE WHEN last_heartbeat IS NULL OR last_heartbeat < ? THEN ? ELSE last_heartbeat END,
		updated_at = ?
		WHERE id = ? AND lifecycle_version = ? AND status IN (?, ?) AND cancel_requested_at IS NULL
		RETURNING owner_epoch`
	stamp := NewNullableTime(&at)
	var epoch int64
	err := r.db.QueryRowContext(ctx, query, ownerIdentity, stamp, stamp, SQLiteTime(time.Now().UTC()), id, lifecycleVersion,
		string(domain.RunStatusRunning), string(domain.RunStatusStarting)).Scan(&epoch)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, wrapDBError("claim_recovery_ownership", "Run", id.String(), err)
	}
	return epoch, true, nil
}

// ClaimFreshRecovery fences both stale continuation writers and continuations
// that read after the claim. Update cannot reactivate a consumed source even
// when its caller holds the new lifecycle version. No expiring cache is involved.
func (r *runRepository) ClaimFreshRecovery(ctx context.Context, sourceID uuid.UUID, lifecycleVersion int64, requestHash string, allowCancelled bool) (bool, error) {
	if sourceID == uuid.Nil || len(requestHash) != 64 {
		return false, domain.NewValidationError("freshRecovery", "source identity and SHA-256 request hash are required")
	}
	const query = `UPDATE runs SET fresh_recovery_request_hash = ?, lifecycle_version = lifecycle_version + 1, updated_at = ?
		WHERE id = ? AND lifecycle_version = ? AND fresh_recovery_request_hash = ''
		AND (status = 'failed' OR (? AND status = 'cancelled'))
		AND (? OR cancel_requested_at IS NULL)
		AND execution_mode <> 'imported' AND (? OR execution_mode <> 'interactive')
		AND NOT EXISTS (SELECT 1 FROM runs AS replacement WHERE replacement.idempotency_key = ?)
		AND NOT EXISTS (SELECT 1 FROM run_creation_receipts AS receipt WHERE receipt.idempotency_key = ?)`
	result, err := r.db.ExecContext(ctx, query, requestHash, SQLiteTime(time.Now().UTC()), sourceID, lifecycleVersion,
		allowCancelled, allowCancelled, allowCancelled, "resume-from-failed:"+sourceID.String(), "resume-from-failed:"+sourceID.String())
	if err != nil {
		return false, wrapDBError("claim_fresh_recovery", "Run", sourceID.String(), err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, wrapDBError("claim_fresh_recovery", "Run", sourceID.String(), err)
	}
	return count == 1, nil
}

func (r *runRepository) GetFreshRecoveryClaim(ctx context.Context, sourceID uuid.UUID) (string, error) {
	var hash string
	err := r.db.GetContext(ctx, &hash, `SELECT fresh_recovery_request_hash FROM runs WHERE id = ?`, sourceID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", domain.NewNotFoundErrorWithID("Run", sourceID.String())
	}
	if err != nil {
		return "", wrapDBError("get_fresh_recovery_claim", "Run", sourceID.String(), err)
	}
	return hash, nil
}

// ReleaseFreshRecoveryClaim is only for an owner-proven pre-effect refusal.
// Neither a missing result nor an expired cache establishes that precondition.
// Advance the version again so an old claimant cannot regain writer authority.
func (r *runRepository) ReleaseFreshRecoveryClaim(ctx context.Context, sourceID uuid.UUID, claimedLifecycleVersion int64, requestHash string) (bool, error) {
	if sourceID == uuid.Nil || len(requestHash) != 64 {
		return false, domain.NewValidationError("freshRecovery", "source identity and SHA-256 request hash are required")
	}
	const query = `UPDATE runs SET fresh_recovery_request_hash = '', lifecycle_version = lifecycle_version + 1, updated_at = ?
		WHERE id = ? AND lifecycle_version = ? AND fresh_recovery_request_hash = ? AND status IN ('failed', 'cancelled')
		AND NOT EXISTS (SELECT 1 FROM runs AS replacement WHERE replacement.idempotency_key = ?)
		AND NOT EXISTS (SELECT 1 FROM run_creation_receipts AS receipt WHERE receipt.idempotency_key = ?)`
	key := "resume-from-failed:" + sourceID.String()
	result, err := r.db.ExecContext(ctx, query, SQLiteTime(time.Now().UTC()), sourceID, claimedLifecycleVersion, requestHash, key, key)
	if err != nil {
		return false, wrapDBError("release_fresh_recovery_claim", "Run", sourceID.String(), err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, wrapDBError("release_fresh_recovery_claim", "Run", sourceID.String(), err)
	}
	return count == 1, nil
}

// GetByIdempotencyKey resolves the single run durably created under a creation
// idempotency key. The runs table keeps the key UNIQUE, so at most one row can
// match. An empty key identifies nothing and resolves to no match.
func (r *runRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Run, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}
	query := fmt.Sprintf("SELECT %s FROM runs WHERE idempotency_key = ?", runColumns)
	var row runRow
	if err := r.db.GetContext(ctx, &row, query, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			receipt, err := r.GetCreationReceipt(ctx, key)
			if err != nil {
				return nil, err
			}
			if receipt != nil {
				return nil, domain.NewStateError("Run", "accepted-result-unavailable", "replay", "original creation was accepted but its run is unavailable; inspect the retained creation receipt; replacement is forbidden")
			}
			return nil, nil
		}
		return nil, wrapDBError("get_by_idempotency_key", "Run", key, err)
	}
	return row.toDomain(), nil
}

func (r *runRepository) GetCreationReceipt(ctx context.Context, key string) (*domain.RunCreationReceipt, error) {
	if strings.TrimSpace(key) == "" {
		return nil, nil
	}
	var row struct {
		Key          string         `db:"idempotency_key"`
		RunID        uuid.UUID      `db:"run_id"`
		TaskID       NullableUUID   `db:"task_id"`
		OwnerSubject sql.NullString `db:"owner_subject"`
		CreatedAt    SQLiteTime     `db:"created_at"`
	}
	if err := r.db.GetContext(ctx, &row, `SELECT idempotency_key, run_id, task_id, owner_subject, created_at FROM run_creation_receipts WHERE idempotency_key = ?`, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_creation_receipt", "Run", key, err)
	}
	return &domain.RunCreationReceipt{IdempotencyKey: row.Key, RunID: row.RunID, TaskID: row.TaskID.UUID, OwnerSubject: row.OwnerSubject.String, CreatedAt: row.CreatedAt.Time()}, nil
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

// GetByImportProvenance resolves the run that already adopted a harness
// session. The harness is matched through domain.NormalizeImportHarness rather
// than by raw string: import paths label the same store differently, and
// treating the label as part of the identity made each path adopt sessions the
// other had already imported, duplicating runs on every sweep.
//
// An empty session id identifies nothing — every non-imported run shares it —
// so it resolves to no match instead of an arbitrary row.
func (r *runRepository) GetByImportProvenance(ctx context.Context, sourceHarness, sourceSessionID string) (*domain.Run, error) {
	if strings.TrimSpace(sourceSessionID) == "" {
		return nil, nil
	}
	query := fmt.Sprintf("SELECT %s FROM runs WHERE import_source_session_id = ? ORDER BY created_at ASC, id ASC", runColumns)
	var rows []runRow
	if err := r.db.SelectContext(ctx, &rows, query, sourceSessionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, wrapDBError("get_by_import_provenance", "Run", sourceHarness+":"+sourceSessionID, err)
	}
	wanted := domain.NormalizeImportHarness(sourceHarness)
	for index := range rows {
		run := rows[index].toDomain()
		if run == nil {
			continue
		}
		if domain.NormalizeImportHarness(run.ImportSourceHarness) == wanted {
			return run, nil
		}
	}
	return nil, nil
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
