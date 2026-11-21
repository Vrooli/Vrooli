package journal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Appender is the only event mutation seam. It deliberately exposes no update
// or delete operation; corrections are additional reversal events.
type Appender interface {
	Append(context.Context, Event) (Event, error)
}

type TransactionExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type transactionalAppender struct {
	db       TransactionExecutor
	resolver ProvenanceResolver
}

// NewTransactionalAppender binds the only journal write primitive to a caller-
// owned transaction. The caller remains responsible for commit or rollback.
func NewTransactionalAppender(db TransactionExecutor) Appender {
	return NewTransactionalAppenderWithResolver(db, ContextProvenanceResolver{})
}

// NewTransactionalAppenderWithResolver keeps provenance deterministic in
// tests while production uses the shared api-core/cli-core verification path.
func NewTransactionalAppenderWithResolver(db TransactionExecutor, resolver ProvenanceResolver) Appender {
	if resolver == nil {
		resolver = ContextProvenanceResolver{}
	}
	return &transactionalAppender{db: db, resolver: resolver}
}

// BalanceInTransaction projects balance from append-only events using the
// caller's transaction. It deliberately does not read balance_projections.
func BalanceInTransaction(ctx context.Context, db TransactionExecutor, holderID, tokenTypeID string) (Balance, error) {
	events, err := readEvents(ctx, db, holderID, tokenTypeID)
	if err != nil {
		return Balance{}, err
	}
	amount, err := projectEvents(events)
	if err != nil {
		return Balance{}, err
	}
	return Balance{HolderID: holderID, TokenTypeID: tokenTypeID, Amount: amount}, nil
}

func (a *transactionalAppender) Append(ctx context.Context, event Event) (Event, error) {
	event = stampEventAttribution(ctx, event, a.resolver)
	if event.Kind != EventKindReversal && strings.TrimSpace(event.Reason) == "" {
		event.Reason = event.CauseReference
	}
	if err := validateEvent(event); err != nil {
		return Event{}, err
	}
	if event.Kind == EventKindReversal {
		original, err := getEvent(ctx, a.db, event.CauseReference)
		if err != nil {
			return Event{}, fmt.Errorf("resolve reversal cause %q: %w", event.CauseReference, err)
		}
		if original.HolderID != event.HolderID || original.TokenTypeID != event.TokenTypeID {
			return Event{}, fmt.Errorf("%w: reversal cause must belong to the same holder and token type", ErrInvalidJournalEvent)
		}
		if original.Amount != event.Amount {
			return Event{}, fmt.Errorf("%w: reversal amount must equal the original event amount", ErrInvalidJournalEvent)
		}
		var existingID string
		err = a.db.QueryRowContext(ctx, `SELECT id FROM journal_events WHERE kind = 'reversal' AND cause_reference = ?`, original.ID).Scan(&existingID)
		if err == nil {
			return Event{}, ErrEventAlreadyReversed
		}
		if err != nil && err != sql.ErrNoRows {
			return Event{}, fmt.Errorf("check existing journal reversal: %w", err)
		}
	}

	_, err := a.db.ExecContext(ctx, `
		INSERT INTO journal_events (
			id, token_type_id, holder_id, amount, kind, cause_reference, reason,
			actor_identity, actor_kind, actor_verification_status, actor_run_id, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.TokenTypeID, event.HolderID, event.Amount, event.Kind,
		event.CauseReference, event.Reason, event.ActorIdentity, event.ActorKind,
		event.ActorVerificationStatus, event.ActorRunID, formatTime(event.CreatedAt))
	if err != nil {
		return Event{}, fmt.Errorf("insert journal event: %w", err)
	}

	events, err := readEvents(ctx, a.db, event.HolderID, event.TokenTypeID)
	if err != nil {
		return Event{}, err
	}
	amount, err := projectEvents(events)
	if err != nil {
		return Event{}, err
	}
	if err := writeCachedBalance(ctx, a.db, Balance{HolderID: event.HolderID, TokenTypeID: event.TokenTypeID, Amount: amount}, event.CreatedAt); err != nil {
		return Event{}, err
	}
	return event, nil
}
