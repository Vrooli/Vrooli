package hub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const agentManagerRunDecisionRequestEventType = "agent_manager.run.decision_requested.v1"

// DecisionWithdrawal identifies the exact decision request that is obsolete.
// Source and RequesterRef identify one request; SourceEventType prevents a
// producer from closing a different kind of request that reuses that ref.
type DecisionWithdrawal struct {
	Source          string
	RequesterRef    string
	SourceEventType string
	Reason          string
}

// reserveDecisionRequestTx serializes request creation with withdrawal intake.
// The lifecycle row and notification/Ask are written in the same transaction,
// so either event delivery order has one durable winner.
func reserveDecisionRequestTx(ctx context.Context, q dbTx, source, ref, typeName, now string) (bool, error) {
	if _, err := q.ExecContext(ctx, `INSERT INTO decision_request_lifecycle (source, requester_ref, source_event_type, state, reason, created_at, updated_at) VALUES (?, ?, ?, 'requested', '', ?, ?) ON CONFLICT(source, requester_ref) DO NOTHING`, source, ref, typeName, now, now); err != nil {
		return false, err
	}
	var existingType, state string
	if err := q.QueryRowContext(ctx, `SELECT source_event_type, state FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, source, ref).Scan(&existingType, &state); err != nil {
		return false, err
	}
	if existingType != typeName {
		return false, ErrAskIdentityConflict
	}
	return state == "withdrawn", nil
}

// checkDecisionRequestLifecycleTx prevents an Agent Manager request from
// reusing a ref that is already bound by a withdrawal, without creating
// lifecycle rows for other decision types.
func checkDecisionRequestLifecycleTx(ctx context.Context, q dbTx, source, ref, typeName string) (bool, error) {
	var existingType, state string
	err := q.QueryRowContext(ctx, `SELECT source_event_type, state FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, source, ref).Scan(&existingType, &state)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if existingType != typeName {
		return false, ErrAskIdentityConflict
	}
	return state == "withdrawn", nil
}

// WithdrawDecisionAsk records exact producer lifecycle intent and closes only
// the matching still-open Ask. It is not an operator answer: no answers row or
// answered_by attribution is written. The lifecycle ledger also records a
// withdrawal that arrives before its request, preventing delayed delivery from
// creating a notification after the cancellation was acknowledged.
func (s *Service) WithdrawDecisionAsk(ctx context.Context, in DecisionWithdrawal) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("ask service is unavailable")
	}
	source := strings.TrimSpace(in.Source)
	ref := strings.TrimSpace(in.RequesterRef)
	typeName := strings.TrimSpace(in.SourceEventType)
	reason := strings.TrimSpace(in.Reason)
	if source != "agent-manager" || ref == "" || len(ref) > 128 || typeName != agentManagerRunDecisionRequestEventType || (reason != "verified_empty_diff" && reason != "superseded_by_new_review") {
		return false, fmt.Errorf("%w: withdrawal identity or reason is invalid", ErrInvalidArgument)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	ask, askErr := askByRequesterTx(ctx, tx, source, ref)
	askExists := askErr == nil
	if askErr != nil && !errors.Is(askErr, ErrNotFound) {
		return false, askErr
	}
	if askExists && ask.SourceEventType != typeName {
		return false, ErrAskIdentityConflict
	}
	if askExists && ask.State != AskPending && ask.State != AskEscalated {
		// A human answer or another terminal outcome already won. Keep that
		// outcome and its attribution intact.
		return false, nil
	}

	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	inserted, err := tx.ExecContext(ctx, `INSERT INTO decision_request_lifecycle (source, requester_ref, source_event_type, state, reason, created_at, updated_at) VALUES (?, ?, ?, 'withdrawn', ?, ?, ?) ON CONFLICT(source, requester_ref) DO NOTHING`, source, ref, typeName, reason, now, now)
	if err != nil {
		return false, err
	}
	insertCount, err := inserted.RowsAffected()
	if err != nil {
		return false, err
	}
	var lifecycleType, lifecycleState string
	if err := tx.QueryRowContext(ctx, `SELECT source_event_type, state FROM decision_request_lifecycle WHERE source=? AND requester_ref=?`, source, ref).Scan(&lifecycleType, &lifecycleState); err != nil {
		return false, err
	}
	if lifecycleType != typeName {
		return false, ErrAskIdentityConflict
	}
	lifecycleChanged := insertCount == 1
	if lifecycleState != "withdrawn" {
		if _, err := tx.ExecContext(ctx, `UPDATE decision_request_lifecycle SET state='withdrawn', reason=?, updated_at=? WHERE source=? AND requester_ref=? AND source_event_type=? AND state='requested'`, reason, now, source, ref, typeName); err != nil {
			return false, err
		}
		lifecycleChanged = true
	}
	if !askExists {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return lifecycleChanged, nil
	}

	withdrawalReason := "requester confirmed no decision is needed"
	if reason == "superseded_by_new_review" {
		withdrawalReason = "a newer review request superseded this request"
	}
	result, err := tx.ExecContext(ctx, `UPDATE asks SET state='withdrawn', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND source=? AND requester_ref=? AND source_event_type=? AND state IN ('pending','escalated')`, withdrawalReason, now, now, ask.ID, source, ref, typeName)
	if err != nil {
		return false, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	if changed == 1 {
		s.kickResolutions(ctx)
		return true, nil
	}
	return false, nil
}
