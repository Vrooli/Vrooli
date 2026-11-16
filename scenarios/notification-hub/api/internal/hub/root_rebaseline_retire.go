package hub

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// RetireRootRebaseline permanently prevents an expired, unused decision from
// being consumed. The unique consumption row serializes this with Consume.
// Its non-time marker can never be mistaken for a consumed approval.
func (s *Service) RetireRootRebaseline(ctx context.Context, requester, digest string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	ask, err := scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.source = ? AND a.source_event_type = ? AND a.requester_ref = ?`, "root-rebaseline", "root.rebaseline.decision_requested.v1", requester))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrRootDecisionUnavailable
		}
		return err
	}
	deadline, err := time.Parse(time.RFC3339Nano, ask.Deadline)
	if err != nil || s.clock.Now().UTC().Before(deadline) || ask.ProviderDecisionContext == nil || ask.ProviderDecisionContext.SchemaVersion != 1 || ask.ProviderDecisionContext.Scope != ProviderDecisionScopeRootRebaseline || ask.ProviderDecisionContext.CanonicalTermsSHA256 != digest || len(ask.Options) != 2 || ask.Options[0].Key != "approve" || ask.Options[1].Key != "reject" {
		return ErrRootDecisionUnavailable
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO root_rebaseline_consumptions (ask_id, consumed_at) VALUES (?, ?) ON CONFLICT (ask_id) DO NOTHING`, ask.ID, "retired")
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		var marker string
		if err := tx.QueryRowContext(ctx, `SELECT consumed_at FROM root_rebaseline_consumptions WHERE ask_id = ?`, ask.ID).Scan(&marker); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrRootDecisionUnavailable
			}
			return err
		}
		if marker != "retired" {
			return ErrRootDecisionUnavailable
		}
	}
	return tx.Commit()
}
