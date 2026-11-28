package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const stripeEventBackfillScope = "stripe-events"

type stripeEventBackfillDB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type stripeEventBackfillState struct {
	Active        bool
	Complete      bool
	StartingAfter string
}

type stripeEventBackfillRef struct {
	Cursor   PaymentReconciliationCursor
	EventID  string
	Type     string
	ObjectID string
}

// SQLStripeEventBackfillStore persists only provider event identity metadata.
// Raw Stripe event payloads are deliberately never stored by the backfill.
type SQLStripeEventBackfillStore struct{ db stripeEventBackfillDB }

func (s *SQLStripeEventBackfillStore) State(ctx context.Context) (stripeEventBackfillState, error) {
	if s == nil || s.db == nil {
		return stripeEventBackfillState{}, errors.New("Stripe event backfill store is unavailable")
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO payment_reconciliation_backfill_state(scope_id) VALUES($1) ON CONFLICT(scope_id) DO NOTHING`, stripeEventBackfillScope); err != nil {
		return stripeEventBackfillState{}, fmt.Errorf("initialize Stripe event backfill state: %w", err)
	}
	var state stripeEventBackfillState
	if err := s.db.QueryRowContext(ctx, `SELECT scan_active,scan_complete,starting_after FROM payment_reconciliation_backfill_state WHERE scope_id=$1`, stripeEventBackfillScope).Scan(&state.Active, &state.Complete, &state.StartingAfter); err != nil {
		return stripeEventBackfillState{}, fmt.Errorf("read Stripe event backfill state: %w", err)
	}
	return state, nil
}

func (s *SQLStripeEventBackfillStore) Begin(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO payment_reconciliation_backfill_state(scope_id,scan_active,scan_complete,starting_after,updated_at) VALUES($1,TRUE,FALSE,'',CURRENT_TIMESTAMP) ON CONFLICT(scope_id) DO UPDATE SET scan_active=TRUE,scan_complete=FALSE,starting_after='',updated_at=CURRENT_TIMESTAMP`, stripeEventBackfillScope); err != nil {
		return fmt.Errorf("begin Stripe event backfill scan: %w", err)
	}
	return nil
}

func (s *SQLStripeEventBackfillStore) AddPage(ctx context.Context, refs []stripeEventBackfillRef, startingAfter string, complete bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Stripe event backfill page transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, ref := range refs {
		if !ref.Cursor.Valid() || ref.EventID != ref.Cursor.EventID || ref.Type == "" || ref.ObjectID == "" {
			return ErrInvalidReconciliationEvent
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO payment_reconciliation_backfill_events(scope_id,event_id,created_at,event_type,object_id) VALUES($1,$2,$3,$4,$5) ON CONFLICT(scope_id,event_id) DO NOTHING`, stripeEventBackfillScope, ref.EventID, ref.Cursor.CreatedAt.UTC(), ref.Type, ref.ObjectID); err != nil {
			return fmt.Errorf("stage Stripe event reference: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payment_reconciliation_backfill_state SET starting_after=$2,scan_complete=$3,scan_active=$4,updated_at=CURRENT_TIMESTAMP WHERE scope_id=$1`, stripeEventBackfillScope, startingAfter, complete, !complete); err != nil {
		return fmt.Errorf("checkpoint Stripe event backfill page: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Stripe event backfill page: %w", err)
	}
	return nil
}

func (s *SQLStripeEventBackfillStore) Pending(ctx context.Context, cursor PaymentReconciliationCursor, hasCursor bool, limit int) ([]stripeEventBackfillRef, error) {
	query := `SELECT event_id,created_at,event_type,object_id FROM payment_reconciliation_backfill_events WHERE scope_id=$1 ORDER BY created_at,event_id LIMIT $2`
	args := []any{stripeEventBackfillScope, limit}
	if hasCursor {
		query = `SELECT event_id,created_at,event_type,object_id FROM payment_reconciliation_backfill_events WHERE scope_id=$1 AND (created_at>$2 OR (created_at=$2 AND event_id>$3)) ORDER BY created_at,event_id LIMIT $4`
		args = []any{stripeEventBackfillScope, cursor.CreatedAt.UTC(), cursor.EventID, limit}
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list staged Stripe event references: %w", err)
	}
	defer rows.Close()
	refs := make([]stripeEventBackfillRef, 0, limit)
	for rows.Next() {
		var ref stripeEventBackfillRef
		if err := rows.Scan(&ref.EventID, &ref.Cursor.CreatedAt, &ref.Type, &ref.ObjectID); err != nil {
			return nil, fmt.Errorf("read staged Stripe event reference: %w", err)
		}
		ref.Cursor.CreatedAt = ref.Cursor.CreatedAt.UTC().Truncate(time.Microsecond)
		ref.Cursor.EventID = ref.EventID
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staged Stripe event references: %w", err)
	}
	return refs, nil
}

func (s *SQLStripeEventBackfillStore) PruneThrough(ctx context.Context, cursor PaymentReconciliationCursor, hasCursor bool) error {
	if !hasCursor {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM payment_reconciliation_backfill_events WHERE scope_id=$1 AND (created_at<$2 OR (created_at=$2 AND event_id<=$3))`, stripeEventBackfillScope, cursor.CreatedAt.UTC(), cursor.EventID); err != nil {
		return fmt.Errorf("prune checkpointed Stripe event references: %w", err)
	}
	return nil
}

func (s *SQLStripeEventBackfillStore) Finish(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE payment_reconciliation_backfill_state SET scan_active=FALSE,scan_complete=FALSE,starting_after='',updated_at=CURRENT_TIMESTAMP WHERE scope_id=$1`, stripeEventBackfillScope); err != nil {
		return fmt.Errorf("finish Stripe event backfill scan: %w", err)
	}
	return nil
}
