package supervision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (r *Repository) RequestAction(ctx context.Context, req *domainpb.RequestCohortWatchActionRequest) (*domainpb.WatchAction, bool, error) {
	if req == nil || strings.TrimSpace(req.GetWatchId()) == "" || strings.TrimSpace(req.GetIdempotencyKey()) == "" || req.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_UNSPECIFIED {
		return nil, false, errors.New("watch id, action kind, and idempotency key are required")
	}
	if existing, err := r.GetActionByIdempotencyKey(ctx, req.GetIdempotencyKey()); err == nil {
		return existing, true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	watch, _, err := r.Get(ctx, req.GetWatchId())
	if err != nil {
		return nil, false, err
	}
	terminalWake := watch.GetStatus() == domainpb.WatchStatus_WATCH_STATUS_TERMINAL && (req.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT || req.GetKind() == domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE)
	if (watch.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_ACTIVE && !terminalWake) || watch.GetRevision() != req.GetExpectedWatchRevision() {
		return nil, false, ErrConflict
	}
	now := r.now().UTC()
	action := &domainpb.WatchAction{ActionId: uuid.NewString(), DecisionId: watch.GetLastDecision().GetDecisionId(), WatchId: req.GetWatchId(), IdempotencyKey: strings.TrimSpace(req.GetIdempotencyKey()), Kind: req.GetKind(), TargetRunId: strings.TrimSpace(req.GetTargetRunId()), Rationale: strings.TrimSpace(req.GetRationale()), State: domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED, RequestedBy: strings.TrimSpace(req.GetRequestedBy()), Authority: req.GetAuthority(), Message: strings.TrimSpace(req.GetMessage()), Cooldown: req.GetCooldown(), MaximumCount: req.GetMaximumCount(), CreatedAt: timestamppb.New(now)}
	encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(action)
	if err != nil {
		return nil, false, err
	}
	var cooldownUntil any
	if action.GetCooldown().IsValid() && action.GetCooldown().AsDuration() > 0 {
		cooldownUntil = formatTime(now.Add(action.GetCooldown().AsDuration()))
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO cohort_watch_actions (action_id,watch_id,decision_id,idempotency_key,kind,target_run_id,state,action_json,cooldown_until,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, action.GetActionId(), action.GetWatchId(), nullableString(action.GetDecisionId()), action.GetIdempotencyKey(), int32(action.GetKind()), action.GetTargetRunId(), int32(action.GetState()), string(encoded), cooldownUntil, formatTime(now))
	if err != nil {
		_ = tx.Rollback()
		if existing, getErr := r.GetActionByIdempotencyKey(ctx, req.GetIdempotencyKey()); getErr == nil {
			return existing, true, nil
		}
		return nil, false, fmt.Errorf("insert watch action: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cohort_watch_action_transitions (action_id,sequence,state,reason,occurred_at) VALUES (?,1,?,'',?)`, action.GetActionId(), int32(action.GetState()), formatTime(now)); err != nil {
		return nil, false, err
	}
	// Every intervention begins as an unassessed outcome linked to the exact
	// decision. Application is not evidence of benefit.
	if action.GetDecisionId() != "" {
		for _, subject := range watch.GetSpec().GetSubjects() {
			if action.GetTargetRunId() != watch.GetSpec().GetParentRunId() && action.GetTargetRunId() != "" && action.GetTargetRunId() != subject.GetRunId() {
				continue
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO supervision_outcomes(outcome_id,idempotency_key,policy_version,family_execution_id,watch_id,decision_id,action_id,child_run_id,evidence_json,predicted_class,observed_class,overridden,counterexample,safety_violation,completion_impact,supersedes_outcome_id,created_at,expires_at)
    SELECT ?,?,policy_version,family_execution_id,watch_id,decision_id,?,child_run_id,evidence_json,predicted_class,'',0,0,0,0,outcome_id,?,? FROM supervision_outcomes WHERE decision_id=? AND child_run_id=? AND action_id='' AND observed_class='' LIMIT 1`, uuid.NewString(), "action-observation:"+action.GetActionId()+":"+subject.GetRunId(), action.GetActionId(), formatTime(now), formatTime(now.Add(180*24*time.Hour)), action.GetDecisionId(), subject.GetRunId())
			if err != nil {
				return nil, false, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return action, false, nil
}

func (r *Repository) TransitionAction(ctx context.Context, actionID string, expected, next domainpb.WatchActionState, reason string) (*domainpb.WatchAction, error) {
	action, err := r.GetAction(ctx, actionID)
	if err != nil {
		return nil, err
	}
	if action.GetState() != expected {
		return nil, ErrConflict
	}
	updated := proto.Clone(action).(*domainpb.WatchAction)
	updated.State = next
	if next == domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED || next == domainpb.WatchActionState_WATCH_ACTION_STATE_SUPERSEDED {
		updated.RejectionReason = strings.TrimSpace(reason)
	}
	now := r.now().UTC()
	if next == domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED {
		updated.AcknowledgedAt = timestamppb.New(now)
	}
	encoded, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(updated)
	if err != nil {
		return nil, err
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if next == domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED {
		// Acquire the database write lock before checking/reserving the budget.
		// Accepted interventions consume capacity even while awaiting a safe turn.
		if _, err = tx.ExecContext(ctx, `UPDATE cohort_watches SET revision=revision WHERE watch_id=?`, action.GetWatchId()); err != nil {
			return nil, err
		}
		var reserved int
		var latest sql.NullString
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*),MAX(COALESCE(acknowledged_at,created_at)) FROM cohort_watch_actions WHERE watch_id=? AND kind=? AND target_run_id=? AND state IN (?,?)`, action.GetWatchId(), int32(action.GetKind()), action.GetTargetRunId(), int32(domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED), int32(domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED)).Scan(&reserved, &latest)
		if err != nil {
			return nil, err
		}
		denied := ""
		if action.GetMaximumCount() > 0 && reserved >= int(action.GetMaximumCount()) {
			denied = "maximum action count reserved"
		}
		if latest.Valid && action.GetCooldown().IsValid() {
			stamp, err := time.Parse(time.RFC3339Nano, latest.String)
			if err != nil {
				return nil, err
			}
			if now.Before(stamp.Add(action.GetCooldown().AsDuration())) {
				denied = "action cooldown is active"
			}
		}
		if denied != "" {
			next = domainpb.WatchActionState_WATCH_ACTION_STATE_REJECTED
			reason = denied
			updated.State = next
			updated.RejectionReason = denied
			encoded, err = protojson.MarshalOptions{UseProtoNames: true}.Marshal(updated)
			if err != nil {
				return nil, err
			}
		}
	}
	result, err := tx.ExecContext(ctx, `UPDATE cohort_watch_actions SET state=?,action_json=?,acknowledged_at=? WHERE action_id=? AND state=?`, int32(next), string(encoded), nullableTimestamp(updated.GetAcknowledgedAt()), actionID, int32(expected))
	if err != nil {
		return nil, err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return nil, ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO cohort_watch_action_transitions (action_id,sequence,state,reason,occurred_at) SELECT ?,COALESCE(MAX(sequence),0)+1,?,?,? FROM cohort_watch_action_transitions WHERE action_id=?`, actionID, int32(next), strings.TrimSpace(reason), formatTime(now), actionID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *Repository) GetAction(ctx context.Context, actionID string) (*domainpb.WatchAction, error) {
	return scanAction(r.db.QueryRowContext(ctx, `SELECT action_json FROM cohort_watch_actions WHERE action_id=?`, actionID))
}

func (r *Repository) GetActionByIdempotencyKey(ctx context.Context, key string) (*domainpb.WatchAction, error) {
	return scanAction(r.db.QueryRowContext(ctx, `SELECT action_json FROM cohort_watch_actions WHERE idempotency_key=?`, strings.TrimSpace(key)))
}

func (r *Repository) ListActions(ctx context.Context, watchID string, limit int) ([]*domainpb.WatchAction, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.QueryxContext(ctx, `SELECT action_json FROM cohort_watch_actions WHERE watch_id=? ORDER BY created_at DESC,action_id DESC LIMIT ?`, watchID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*domainpb.WatchAction, 0, limit)
	for rows.Next() {
		action, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, action)
	}
	return result, rows.Err()
}

func (r *Repository) ListPendingActions(ctx context.Context, limit int) ([]*domainpb.WatchAction, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.db.QueryxContext(ctx, `SELECT action_json FROM cohort_watch_actions WHERE state IN (?,?) ORDER BY created_at,action_id LIMIT ?`, int32(domainpb.WatchActionState_WATCH_ACTION_STATE_REQUESTED), int32(domainpb.WatchActionState_WATCH_ACTION_STATE_ACCEPTED), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*domainpb.WatchAction, 0, limit)
	for rows.Next() {
		action, err := scanAction(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, action)
	}
	return result, rows.Err()
}

func (r *Repository) AppliedActionStats(ctx context.Context, watchID string, kind domainpb.WatchActionKind, targetRunID string) (int, *time.Time, error) {
	var count int
	var latest sql.NullString
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*),MAX(acknowledged_at) FROM cohort_watch_actions WHERE watch_id=? AND kind=? AND target_run_id=? AND state=?`, watchID, int32(kind), targetRunID, int32(domainpb.WatchActionState_WATCH_ACTION_STATE_APPLIED)).Scan(&count, &latest); err != nil {
		return 0, nil, err
	}
	if !latest.Valid || latest.String == "" {
		return count, nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, latest.String)
	return count, &parsed, err
}

func scanAction(row scanner) (*domainpb.WatchAction, error) {
	var encoded string
	if err := row.Scan(&encoded); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	action := &domainpb.WatchAction{}
	if err := protojson.Unmarshal([]byte(encoded), action); err != nil {
		return nil, err
	}
	return action, nil
}

func nullableTimestamp(value *timestamppb.Timestamp) any {
	if value == nil || !value.IsValid() {
		return nil
	}
	return formatTime(value.AsTime())
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
