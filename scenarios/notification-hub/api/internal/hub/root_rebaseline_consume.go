package hub

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// RootRebaselineConsumedAt returns the durable consumption marker for an ask.
func (s *Service) RootRebaselineConsumedAt(ctx context.Context, askID string) (string, error) {
	var consumed string
	err := s.db.QueryRowContext(ctx, `SELECT consumed_at FROM root_rebaseline_consumptions WHERE ask_id = ?`, askID).Scan(&consumed)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return consumed, err
}

// ErrRootDecisionUnavailable does not disclose which approval condition failed.
var ErrRootDecisionUnavailable = errors.New("root rebaseline decision unavailable")

// ConsumeRootRebaseline atomically checks the persisted answer and records its
// one-time use. A successful commit consumes approval even if the response is
// lost; the caller must request a new decision instead of replaying it.
func (s *Service) ConsumeRootRebaseline(ctx context.Context, requester, digest string) (AskRecord, string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AskRecord{}, "", err
	}
	defer func() { _ = tx.Rollback() }()
	ask, err := scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.source = ? AND a.source_event_type = ? AND a.requester_ref = ?`, "root-rebaseline", "root.rebaseline.decision_requested.v1", requester))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return AskRecord{}, "", ErrRootDecisionUnavailable
		}
		return AskRecord{}, "", err
	}
	now := s.clock.Now().UTC()
	deadline, deadlineErr := time.Parse(time.RFC3339Nano, ask.Deadline)
	answered, answeredErr := time.Parse(time.RFC3339Nano, ask.AnsweredAt)
	if deadlineErr != nil || answeredErr != nil || !now.Before(deadline) || !answered.Before(deadline) || answered.After(now) || ask.State != AskAnswered || ask.Answer != "approve" || ask.AnsweredByIdentityKind != AnswerIdentityKindHuman || (ask.AnsweredByIdentitySource != AnswerIdentitySourceAccess && ask.AnsweredByIdentitySource != AnswerIdentitySourcePassword) || ask.ProviderDecisionContext == nil || ask.ProviderDecisionContext.SchemaVersion != 1 || ask.ProviderDecisionContext.Scope != ProviderDecisionScopeRootRebaseline || ask.ProviderDecisionContext.CanonicalTermsSHA256 != digest || len(ask.Options) != 2 || ask.Options[0].Key != "approve" || ask.Options[1].Key != "reject" {
		return AskRecord{}, "", ErrRootDecisionUnavailable
	}
	consumed := storedTime(now)
	result, err := tx.ExecContext(ctx, `INSERT INTO root_rebaseline_consumptions (ask_id, consumed_at) VALUES (?, ?) ON CONFLICT (ask_id) DO NOTHING`, ask.ID, consumed)
	if err != nil {
		return AskRecord{}, "", err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return AskRecord{}, "", err
	}
	if count != 1 {
		return AskRecord{}, "", ErrRootDecisionUnavailable
	}
	if err := tx.Commit(); err != nil {
		return AskRecord{}, "", err
	}
	return ask, consumed, nil
}
