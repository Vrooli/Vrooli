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

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	eventdomain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
)

// parseAwaitHandleJSON decodes the persisted await owner, returning nil for
// absent or malformed JSON so rows without an active park remain unowned.
func parseAwaitHandleJSON(raw sql.NullString) *domain.AwaitHandle {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var handle domain.AwaitHandle
	if err := json.Unmarshal([]byte(raw.String), &handle); err != nil {
		return nil
	}
	return &handle
}

// marshalAwaitHandleJSON stores a nil handle as SQL NULL.
func marshalAwaitHandleJSON(handle *domain.AwaitHandle) sql.NullString {
	if handle == nil {
		return sql.NullString{}
	}
	data, err := json.Marshal(handle)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(data), Valid: true}
}

func parseEffortSignalCursorsJSON(raw sql.NullString) map[string]domain.EffortSignalCursor {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var cursors map[string]domain.EffortSignalCursor
	if err := json.Unmarshal([]byte(raw.String), &cursors); err != nil {
		return nil
	}
	return cursors
}

func decodeEffortSignalCursors(raw string) (map[string]domain.EffortSignalCursor, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "{}"
	}
	var cursors map[string]domain.EffortSignalCursor
	if err := json.Unmarshal([]byte(raw), &cursors); err != nil {
		return nil, fmt.Errorf("decode effort signal cursors: %w", err)
	}
	if cursors == nil {
		cursors = map[string]domain.EffortSignalCursor{}
	}
	return cursors, nil
}

func (r *runRepository) QueueEffortSignalCursor(ctx context.Context, runID uuid.UUID, effortRef string, through int64) (domain.EffortSignalCursor, bool, error) {
	return r.mutateEffortSignalCursor(ctx, runID, effortRef, through)
}

func (r *runRepository) mutateEffortSignalCursor(ctx context.Context, runID uuid.UUID, effortRef string, through int64) (domain.EffortSignalCursor, bool, error) {
	if runID == uuid.Nil || strings.TrimSpace(effortRef) == "" || through < 0 {
		return domain.EffortSignalCursor{}, false, fmt.Errorf("exact run, effort reference and nonnegative signal sequence required")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.EffortSignalCursor{}, false, err
	}
	defer tx.Rollback()
	var status string
	var references, rawCursors sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT status, work_references, effort_signal_cursors FROM runs WHERE id=?`, runID).Scan(&status, &references, &rawCursors)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EffortSignalCursor{}, false, nil
	}
	if err != nil {
		return domain.EffortSignalCursor{}, false, wrapDBError("read_effort_signal_cursor", "Run", runID.String(), err)
	}
	if domain.RunStatus(status).IsTerminal() || !domain.RunStatus(status).IsValid() || !carriesActiveEffortOrchestrator(parseWorkReferencesJSON(references), effortRef) {
		return domain.EffortSignalCursor{}, false, nil
	}
	oldRaw := rawCursors.String
	cursors, err := decodeEffortSignalCursors(oldRaw)
	if err != nil {
		return domain.EffortSignalCursor{}, false, wrapDBError("decode_effort_signal_cursor", "Run", runID.String(), err)
	}
	cursor, exists := cursors[effortRef]
	changed := false
	if !exists {
		// No prior read boundary is recorded for an unseen owner. Queue the
		// coalesced catch-up without claiming any retained row was delivered.
		cursor = domain.EffortSignalCursor{QueuedSeq: through, DeliveredSeq: 0}
		cursors[effortRef] = cursor
		changed = true
	} else if through > cursor.QueuedSeq && through > cursor.DeliveredSeq {
		cursor.QueuedSeq = through
		cursors[effortRef] = cursor
		changed = true
	}
	if !changed {
		return cursor, true, nil
	}
	encoded, err := json.Marshal(cursors)
	if err != nil {
		return domain.EffortSignalCursor{}, false, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE runs SET effort_signal_cursors=? WHERE id=? AND status=? AND COALESCE(work_references,'[]')=? AND COALESCE(effort_signal_cursors,'')=?`, string(encoded), runID, status, references.String, oldRaw)
	if err != nil {
		return domain.EffortSignalCursor{}, false, wrapDBError("write_effort_signal_cursor", "Run", runID.String(), err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return domain.EffortSignalCursor{}, false, err
	}
	if updated != 1 {
		return domain.EffortSignalCursor{}, false, domain.NewStateErrorWithID("Run", runID.String(), "superseded", "queue effort signal", "run identity or cursor changed")
	}
	if err := tx.Commit(); err != nil {
		return domain.EffortSignalCursor{}, false, wrapDBError("write_effort_signal_cursor", "Run", runID.String(), err)
	}
	return cursor, true, nil
}

func carriesActiveEffortOrchestrator(refs []*eventdomain.WorkReference, effortRef string) bool {
	for _, ref := range refs {
		if domain.IsPublicActiveEffortOrchestratorReference(ref) && ref.GetId() == effortRef {
			return true
		}
	}
	return false
}

func (r *runRepository) commitEffortSignalReceipt(ctx context.Context, tx *Tx, runID uuid.UUID, receipt *domain.EffortSignalReceipt) error {
	if receipt == nil || receipt.EffortRef == "" || receipt.Through < 0 {
		return fmt.Errorf("valid exact effort signal receipt required")
	}
	var raw sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT effort_signal_cursors FROM runs WHERE id=?`, runID).Scan(&raw); err != nil {
		return err
	}
	cursors, err := decodeEffortSignalCursors(raw.String)
	if err != nil {
		return err
	}
	cursor, ok := cursors[receipt.EffortRef]
	if !ok || cursor.QueuedSeq < receipt.Through {
		return fmt.Errorf("effort %s sequence %d was not durably queued", receipt.EffortRef, receipt.Through)
	}
	if receipt.Through > cursor.DeliveredSeq {
		cursor.DeliveredSeq = receipt.Through
		cursors[receipt.EffortRef] = cursor
		encoded, err := json.Marshal(cursors)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE runs SET effort_signal_cursors=? WHERE id=?`, string(encoded), runID); err != nil {
			return err
		}
	}
	return nil
}

func (r *runRepository) Update(ctx context.Context, run *domain.Run) error {
	return r.update(ctx, run, nil)
}

type expectedInteractiveContinuation struct {
	status       domain.RunStatus
	invocationAt NullableTime
}

func (r *runRepository) UpdateInteractiveContinuation(ctx context.Context, run *domain.Run, expectedStatus domain.RunStatus, expectedInvocationStartedAt *time.Time) error {
	if run == nil {
		return domain.NewValidationError("run", "run is required")
	}
	if (!expectedStatus.IsTerminal() && expectedStatus != domain.RunStatusParked && expectedStatus != domain.RunStatusRunning) || run.Status != domain.RunStatusRunning || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.InteractiveInvocationStartedAt == nil {
		return domain.NewValidationError("interactiveContinuation", "expected a running, parked, or terminal interactive run continuing with a new invocation")
	}
	if expectedInvocationStartedAt != nil && !run.InteractiveInvocationStartedAt.After(*expectedInvocationStartedAt) {
		return domain.NewValidationError("interactiveContinuation", "new invocation must be later than the expected prior invocation")
	}
	expectedInvocation := NullableTime{}
	if expectedInvocationStartedAt != nil {
		expectedInvocation = NullableTime{Time: *expectedInvocationStartedAt, Valid: true}
	}
	return r.update(ctx, run, &expectedInteractiveContinuation{
		status:       expectedStatus,
		invocationAt: expectedInvocation,
	})
}

func (r *runRepository) update(ctx context.Context, run *domain.Run, continuation *expectedInteractiveContinuation) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	defer tx.Rollback()
	var nextLifecycleVersion int64
	if err := r.updateTx(ctx, tx, run, continuation, &nextLifecycleVersion); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	run.LifecycleVersion = nextLifecycleVersion
	run.EffortSignalReceipt = nil
	return nil
}

func (r *runRepository) updateTx(ctx context.Context, tx *Tx, run *domain.Run, continuation *expectedInteractiveContinuation, nextVersion *int64) error {
	if run.EffortSignalReceipt != nil && run.Status != domain.RunStatusRunning {
		return fmt.Errorf("effort signal receipt is valid only for a parked-to-running wake")
	}
	run.UpdatedAt = time.Now()
	row := runFromDomain(run)
	if continuation != nil {
		row.ExpectedInteractiveStatus = string(continuation.status)
		row.ExpectedInteractiveInvocationAt = continuation.invocationAt
	}

	// A renewal anchor marks an attached run whose process binding participates
	// in the broker's proof. Preserve that binding across stale full-row writes;
	// legacy attached rows without a renewal anchor retain the prior update path.
	// Renewal-generation columns themselves are intentionally absent here.
	query := `UPDATE runs SET lifecycle_version = lifecycle_version + CASE WHEN status = :status THEN 0 ELSE 1 END,
			owner_identity = CASE WHEN :owner_identity <> '' THEN :owner_identity ELSE owner_identity END,
			owner_epoch = CASE WHEN :owner_epoch > 0 THEN :owner_epoch ELSE owner_epoch END,
			task_id = CASE WHEN identity_renewal_anchor_hash <> '' THEN task_id ELSE :task_id END,
			agent_profile_id = :agent_profile_id,
			tag = :tag, label = :label, label_source = :label_source, subject = :subject,
			owner_subject = CASE WHEN identity_renewal_anchor_hash <> '' THEN owner_subject ELSE :owner_subject END,
			owner_expires_at = CASE WHEN identity_renewal_anchor_hash <> '' THEN owner_expires_at ELSE :owner_expires_at END,
			owner_scopes = CASE WHEN identity_renewal_anchor_hash <> '' THEN owner_scopes ELSE :owner_scopes END,
			requested_scopes = :requested_scopes, sandbox_id = :sandbox_id, run_mode = :run_mode,
		execution_mode = CASE WHEN identity_renewal_anchor_hash <> '' THEN execution_mode ELSE :execution_mode END,
		harness_kind = CASE WHEN identity_renewal_anchor_hash <> '' THEN harness_kind ELSE :harness_kind END,
		harness_session_id = CASE WHEN identity_renewal_anchor_hash <> '' THEN harness_session_id ELSE :harness_session_id END,
		goal_delivery = :goal_delivery, terminal_class = :terminal_class, stop_reason = :stop_reason, last_handoff = :last_handoff,
		web_console_session_id = CASE WHEN identity_renewal_anchor_hash <> '' THEN web_console_session_id ELSE :web_console_session_id END,
		status = :status,
		started_at = :started_at, interactive_invocation_started_at = :interactive_invocation_started_at, ended_at = :ended_at, cancel_requested_at = COALESCE(cancel_requested_at, :cancel_requested_at), phase = :phase,
		last_checkpoint_id = :last_checkpoint_id, last_heartbeat = CASE WHEN last_heartbeat IS NULL OR last_heartbeat < :last_heartbeat THEN :last_heartbeat ELSE last_heartbeat END,
		progress_percent = :progress_percent, idempotency_key = :idempotency_key,
		summary = :summary, run_result = :run_result, error_msg = :error_msg, exit_code = :exit_code,
		approval_state = :approval_state, approved_by = :approved_by, approved_at = :approved_at,
		finalization_status = :finalization_status, finalization_error = :finalization_error, finalized_at = :finalized_at,
		resolved_config = :resolved_config, diff_path = :diff_path, log_path = :log_path,
			changed_files = :changed_files, total_size_bytes = :total_size_bytes, commit_hash = :commit_hash, sandbox_config = :sandbox_config,
			 session_id = :session_id,
			runner_pid = CASE WHEN identity_renewal_anchor_hash <> '' THEN runner_pid ELSE :runner_pid END,
			runner_pgid = CASE WHEN identity_renewal_anchor_hash <> '' THEN runner_pgid ELSE :runner_pgid END,
			transcript_path = :transcript_path, transcript_cursor = :transcript_cursor, transcript_last_seq = :transcript_last_seq,
			import_source_harness = :import_source_harness, import_source_session_id = :import_source_session_id, imported_at = :imported_at,
			source_run_ids = :source_run_ids,
			source_investigation_run_id = :source_investigation_run_id,
			parent_run_id = CASE WHEN identity_renewal_anchor_hash <> '' THEN parent_run_id ELSE :parent_run_id END,
		conversation_id = :conversation_id,
		identity_token_hash = CASE WHEN identity_renewal_anchor_hash <> '' THEN identity_token_hash ELSE :identity_token_hash END,
		identity_token_revoked_at = CASE WHEN identity_renewal_anchor_hash <> '' THEN COALESCE(identity_token_revoked_at, :identity_token_revoked_at) ELSE :identity_token_revoked_at END,
		custom_env = :custom_env, await_handle = :await_handle,
		last_await_key = :last_await_key, last_await_result = :last_await_result,
		last_await_resolved_at = :last_await_resolved_at, last_wake_seq = :last_wake_seq,
		same_key_park_streak = :same_key_park_streak,
		requested_model = :requested_model, actual_model = :actual_model, canary_arm = :canary_arm,
		updated_at = :updated_at
	WHERE id = :id AND lifecycle_version = :lifecycle_version`
	if continuation == nil {
		query += ` AND (execution_mode IS NULL OR execution_mode <> 'interactive' OR interactive_invocation_started_at IS :interactive_invocation_started_at)`
	} else {
		query += ` AND execution_mode = 'interactive' AND status = :expected_interactive_status AND interactive_invocation_started_at IS :expected_interactive_invocation_at AND task_id IS :task_id AND session_id IS :session_id AND web_console_session_id IS :web_console_session_id AND cancel_requested_at IS NULL AND fresh_recovery_request_hash = ''
			AND (task_id IS NULL OR EXISTS (SELECT 1 FROM tasks WHERE tasks.id = runs.task_id AND tasks.current_run_id = runs.id AND tasks.status NOT IN ('cancelled', 'approved', 'rejected')))`
	}
	query += ` AND (owner_epoch = :owner_epoch OR (owner_epoch = 0 AND :owner_epoch = 1))
		  AND (fresh_recovery_request_hash = '' OR status = :status)
		  AND COALESCE(work_references, '[]') = :work_references`
	effortIDs := []string(nil)
	guard, guardArgs := "", []any(nil)
	if run.Status.IsActive() {
		effortIDs = activeEffortOrchestratorIDs(run)
		var err error
		guard, guardArgs, err = effortOrchestratorUpdateGuard(effortIDs)
		if err != nil {
			return fmt.Errorf("prepare effort orchestrator continuation: %w", err)
		}
	}
	query += guard
	if run.EffortSignalReceipt != nil {
		query += ` AND status = ?`
	}
	query += ` RETURNING lifecycle_version`

	bound, args, err := sqlx.Named(query, row)
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	args = append(args, guardArgs...)
	if run.EffortSignalReceipt != nil {
		args = append(args, string(domain.RunStatusParked))
	}
	var nextLifecycleVersion int64
	err = tx.QueryRowContext(ctx, bound, args...).Scan(&nextLifecycleVersion)
	if errors.Is(err, sql.ErrNoRows) {
		var previousStatus domain.RunStatus
		var storedWorkReferences string
		var storedInvocation NullableTime
		lookupErr := tx.QueryRowContext(ctx, `SELECT status, COALESCE(work_references, '[]'), interactive_invocation_started_at FROM runs WHERE id = ? AND lifecycle_version = ?`, run.ID, run.LifecycleVersion).Scan(&previousStatus, &storedWorkReferences, &storedInvocation)
		if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
			return wrapDBError("read_update_conflict", "Run", run.ID.String(), lookupErr)
		}
		if lookupErr == nil && storedWorkReferences != row.WorkReferences.String {
			return domain.NewStateErrorWithID("Run", run.ID.String(), "immutable_identity", "update", "work references are immutable after run creation")
		}
		if lookupErr == nil && run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive && !nullableTimesEqual(storedInvocation, row.InteractiveInvocationStartedAt) {
			return domain.NewStateErrorWithID("Run", run.ID.String(), "invocation_changed", "update", "interactive invocation changed; stale observer refused")
		}
		if len(effortIDs) > 0 && lookupErr == nil && !previousStatus.IsActive() {
			conflict, checkErr := activeEffortOrchestratorConflictForIDs(ctx, tx, run.ID.String(), effortIDs)
			if checkErr != nil {
				return wrapDBError("check_effort_orchestrator_owner", "Run", run.ID.String(), checkErr)
			}
			if conflict {
				return domain.NewStateErrorWithID("Run", run.ID.String(), "active_effort_owner", "continue", "another nonterminal orchestrator run already owns this effort")
			}
		}
		return domain.NewStateErrorWithID("Run", run.ID.String(), "superseded", "update", "run lifecycle changed or was consumed by fresh recovery; writer refused")
	}
	if err != nil {
		return wrapDBError("update", "Run", run.ID.String(), err)
	}
	if receipt := run.EffortSignalReceipt; receipt != nil {
		if err := r.commitEffortSignalReceipt(ctx, tx, run.ID, receipt); err != nil {
			return wrapDBError("commit_effort_signal_receipt", "Run", run.ID.String(), err)
		}
	}
	if run.TaskID != uuid.Nil {
		if status, ok := taskProjectionForRunStatus(run.Status); ok {
			_, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ?,
				updated_at = CASE WHEN status <> ? THEN ? ELSE updated_at END
				WHERE id = ? AND current_run_id = ? AND status IN (?, ?, ?, ?, ?)`,
				string(status), string(status), SQLiteTime(run.UpdatedAt), run.TaskID, run.ID,
				string(domain.TaskStatusQueued), string(domain.TaskStatusRunning), string(domain.TaskStatusNeedsReview), string(domain.TaskStatusCompleted), string(domain.TaskStatusFailed))
			if err != nil {
				return wrapDBError("project_run_status", "Task", run.TaskID.String(), err)
			}
		}
	}
	*nextVersion = nextLifecycleVersion
	return nil
}

func taskProjectionForRunStatus(status domain.RunStatus) (domain.TaskStatus, bool) {
	switch status {
	case domain.RunStatusPending, domain.RunStatusStarting, domain.RunStatusRunning, domain.RunStatusParked:
		return domain.TaskStatusRunning, true
	case domain.RunStatusNeedsReview:
		return domain.TaskStatusNeedsReview, true
	case domain.RunStatusComplete:
		return domain.TaskStatusCompleted, true
	case domain.RunStatusFailed, domain.RunStatusCancelled:
		// Run cancellation is an execution outcome, not the explicit Task cancel
		// operation. Keep TaskStatusCancelled reserved for that operator action.
		return domain.TaskStatusFailed, true
	default:
		return "", false
	}
}

// UpdateRunLabel updates only the two label columns. It is used by the
// historical transcript backfill and deliberately cannot mutate run identity,
// attribution, status, or timestamps.
