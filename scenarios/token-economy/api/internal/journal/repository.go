// Package journal owns append-only token events and balance projections.
package journal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

type EventKind string

const (
	EventKindMint     EventKind = "mint"
	EventKindCredit   EventKind = "credit"
	EventKindDebit    EventKind = "debit"
	EventKindReversal EventKind = "reversal"
	EventKindExpiry   EventKind = "expiry"
)

type Event struct {
	ID                      string
	TokenTypeID             string
	HolderID                string
	Amount                  int64
	Kind                    EventKind
	CauseReference          string
	Reason                  string
	ActorIdentity           string
	ActorKind               string
	ActorVerificationStatus string
	ActorRunID              string
	CreatedAt               time.Time
}

type Reversal struct {
	ID              string
	OriginalEventID string
	Reason          string
	IdempotencyKey  string
	ActorIdentity   string
	CreatedAt       time.Time
}

type Balance struct {
	HolderID    string
	TokenTypeID string
	Amount      int64
}

var (
	ErrEventNotFound        = errors.New("journal event not found")
	ErrEventAlreadyReversed = errors.New("journal event already reversed")
	ErrInvalidJournalEvent  = errors.New("invalid journal event")
	ErrInvalidEventSequence = errors.New("invalid journal event sequence")
)

type Reader interface {
	Read(context.Context, string, string) ([]Event, error)
}

type HolderEventReader interface {
	ReadHolder(context.Context, string) ([]Event, error)
}

type Repository interface {
	Appender
	Reader
	Reverse(context.Context, Reversal) (Event, error)
}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) *sqliteRepository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Append(ctx context.Context, event Event) (Event, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("begin journal append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	appended, err := NewTransactionalAppender(tx).Append(ctx, event)
	if err != nil {
		return Event{}, err
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit journal append: %w", err)
	}
	return appended, nil
}

func (r *sqliteRepository) Reverse(ctx context.Context, reversal Reversal) (Event, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("begin journal reversal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var priorID string
	err = tx.QueryRowContext(ctx, `SELECT reversal_event_id FROM journal_reversal_receipts WHERE idempotency_key = ?`, reversal.IdempotencyKey).Scan(&priorID)
	if err == nil {
		return getEvent(ctx, tx, priorID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Event{}, fmt.Errorf("read journal reversal receipt: %w", err)
	}

	original, err := getEvent(ctx, tx, reversal.OriginalEventID)
	if err != nil {
		return Event{}, err
	}
	if original.Kind == EventKindReversal {
		return Event{}, fmt.Errorf("%w: reversal events cannot be reversed", ErrInvalidJournalEvent)
	}
	var existingID string
	err = tx.QueryRowContext(ctx, `SELECT id FROM journal_events WHERE kind = 'reversal' AND cause_reference = ?`, original.ID).Scan(&existingID)
	if err == nil {
		return Event{}, ErrEventAlreadyReversed
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Event{}, fmt.Errorf("check existing journal reversal: %w", err)
	}

	event := Event{
		ID: reversal.ID, TokenTypeID: original.TokenTypeID, HolderID: original.HolderID,
		Amount: original.Amount, Kind: EventKindReversal, CauseReference: original.ID,
		Reason: reversal.Reason, ActorIdentity: reversal.ActorIdentity, CreatedAt: reversal.CreatedAt,
	}
	appended, err := NewTransactionalAppender(tx).Append(ctx, event)
	if err != nil {
		return Event{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO journal_reversal_receipts (idempotency_key, original_event_id, reversal_event_id, created_at)
		VALUES (?, ?, ?, ?)`, reversal.IdempotencyKey, original.ID, appended.ID, formatTime(reversal.CreatedAt)); err != nil {
		return Event{}, fmt.Errorf("insert journal reversal receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit journal reversal: %w", err)
	}
	return appended, nil
}

func validateEvent(event Event) error {
	switch {
	case strings.TrimSpace(event.ID) == "":
		return fmt.Errorf("%w: id is required", ErrInvalidJournalEvent)
	case strings.TrimSpace(event.TokenTypeID) == "":
		return fmt.Errorf("%w: token type id is required", ErrInvalidJournalEvent)
	case strings.TrimSpace(event.HolderID) == "":
		return fmt.Errorf("%w: holder id is required", ErrInvalidJournalEvent)
	case event.Amount <= 0:
		return fmt.Errorf("%w: amount must be positive", ErrInvalidJournalEvent)
	case strings.TrimSpace(event.CauseReference) == "":
		return fmt.Errorf("%w: cause reference is required", ErrInvalidJournalEvent)
	case event.Kind == EventKindReversal && strings.TrimSpace(event.Reason) == "":
		return fmt.Errorf("%w: reversal reason is required", ErrInvalidJournalEvent)
	case strings.TrimSpace(event.ActorIdentity) == "":
		return fmt.Errorf("%w: actor identity is required", ErrInvalidJournalEvent)
	case strings.TrimSpace(event.ActorKind) == "":
		return fmt.Errorf("%w: actor kind is required", ErrInvalidJournalEvent)
	case strings.TrimSpace(event.ActorVerificationStatus) == "":
		return fmt.Errorf("%w: actor verification status is required", ErrInvalidJournalEvent)
	case event.CreatedAt.IsZero():
		return fmt.Errorf("%w: created time is required", ErrInvalidJournalEvent)
	}
	switch event.Kind {
	case EventKindMint, EventKindCredit, EventKindDebit, EventKindReversal, EventKindExpiry:
		return nil
	default:
		return fmt.Errorf("%w: unsupported kind %q", ErrInvalidJournalEvent, event.Kind)
	}
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getEvent(ctx context.Context, db queryer, id string) (Event, error) {
	var event Event
	var createdAt string
	err := db.QueryRowContext(ctx, `
		SELECT id, token_type_id, holder_id, amount, kind, cause_reference, reason,
		       actor_identity, actor_kind, actor_verification_status, actor_run_id, created_at
		FROM journal_events WHERE id = ?`, id).Scan(
		&event.ID, &event.TokenTypeID, &event.HolderID, &event.Amount,
		&event.Kind, &event.CauseReference, &event.Reason, &event.ActorIdentity,
		&event.ActorKind, &event.ActorVerificationStatus, &event.ActorRunID, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, ErrEventNotFound
	}
	if err != nil {
		return Event{}, fmt.Errorf("read journal event: %w", err)
	}
	event.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Event{}, fmt.Errorf("parse journal event creation time: %w", err)
	}
	return event, nil
}

func (r *sqliteRepository) Read(ctx context.Context, holderID, tokenTypeID string) ([]Event, error) {
	if strings.TrimSpace(holderID) == "" || strings.TrimSpace(tokenTypeID) == "" {
		return nil, fmt.Errorf("%w: holder id and token type id are required", ErrInvalidJournalEvent)
	}
	return readEvents(ctx, r.db, holderID, tokenTypeID)
}

func (r *sqliteRepository) ReadHolder(ctx context.Context, holderID string) ([]Event, error) {
	if strings.TrimSpace(holderID) == "" {
		return nil, fmt.Errorf("%w: holder id is required", ErrInvalidJournalEvent)
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, token_type_id, holder_id, amount, kind, cause_reference, reason,
		       actor_identity, actor_kind, actor_verification_status, actor_run_id, created_at
		FROM journal_events
		WHERE holder_id = ?
		ORDER BY created_at ASC, id ASC`, holderID)
	if err != nil {
		return nil, fmt.Errorf("read holder journal events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func readEvents(ctx context.Context, db queryer, holderID, tokenTypeID string) ([]Event, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, token_type_id, holder_id, amount, kind, cause_reference, reason,
		       actor_identity, actor_kind, actor_verification_status, actor_run_id, created_at
		FROM journal_events
		WHERE holder_id = ? AND token_type_id = ?
		ORDER BY created_at ASC, id ASC`, holderID, tokenTypeID)
	if err != nil {
		return nil, fmt.Errorf("read journal events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func readAllEvents(ctx context.Context, db queryer) ([]Event, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, token_type_id, holder_id, amount, kind, cause_reference, reason,
		       actor_identity, actor_kind, actor_verification_status, actor_run_id, created_at
		FROM journal_events ORDER BY created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("read all journal events: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		var createdAt string
		if err := rows.Scan(
			&event.ID, &event.TokenTypeID, &event.HolderID, &event.Amount, &event.Kind,
			&event.CauseReference, &event.Reason, &event.ActorIdentity, &event.ActorKind,
			&event.ActorVerificationStatus, &event.ActorRunID, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan journal event: %w", err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse journal event creation time: %w", err)
		}
		event.CreatedAt = parsed
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate journal events: %w", err)
	}
	return events, nil
}

func (r *sqliteRepository) BalanceAt(ctx context.Context, holderID, tokenTypeID string) (Balance, error) {
	events, err := r.Read(ctx, holderID, tokenTypeID)
	if err != nil {
		return Balance{}, err
	}
	amount, err := projectEvents(events)
	if err != nil {
		return Balance{}, err
	}
	return Balance{HolderID: holderID, TokenTypeID: tokenTypeID, Amount: amount}, nil
}

func projectEvents(events []Event) (int64, error) {
	var amount int64
	deltas := make(map[string]int64, len(events))
	for _, event := range events {
		var delta int64
		switch event.Kind {
		case EventKindMint, EventKindCredit:
			delta = event.Amount
		case EventKindDebit, EventKindExpiry:
			delta = -event.Amount
		case EventKindReversal:
			original, ok := deltas[event.CauseReference]
			if !ok || original == math.MinInt64 {
				return 0, fmt.Errorf("%w: reversal %q references unavailable event %q", ErrInvalidEventSequence, event.ID, event.CauseReference)
			}
			delta = -original
		default:
			return 0, fmt.Errorf("%w: unsupported kind %q", ErrInvalidEventSequence, event.Kind)
		}
		if (delta > 0 && amount > math.MaxInt64-delta) || (delta < 0 && amount < math.MinInt64-delta) {
			return 0, fmt.Errorf("%w: balance overflow at event %q", ErrInvalidEventSequence, event.ID)
		}
		amount += delta
		deltas[event.ID] = delta
	}
	return amount, nil
}

func (r *sqliteRepository) Rebuild(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin projection rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	events, err := readAllEvents(ctx, tx)
	if err != nil {
		return err
	}
	type key struct{ holderID, tokenTypeID string }
	grouped := make(map[key][]Event)
	for _, event := range events {
		k := key{holderID: event.HolderID, tokenTypeID: event.TokenTypeID}
		grouped[k] = append(grouped[k], event)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM balance_projections`); err != nil {
		return fmt.Errorf("truncate balance projection cache: %w", err)
	}
	keys := make([]key, 0, len(grouped))
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].holderID == keys[j].holderID {
			return keys[i].tokenTypeID < keys[j].tokenTypeID
		}
		return keys[i].holderID < keys[j].holderID
	})
	rebuiltAt := time.Now().UTC()
	for _, k := range keys {
		amount, projectErr := projectEvents(grouped[k])
		if projectErr != nil {
			return projectErr
		}
		if err := writeCachedBalance(ctx, tx, Balance{HolderID: k.holderID, TokenTypeID: k.tokenTypeID, Amount: amount}, rebuiltAt); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit projection rebuild: %w", err)
	}
	return nil
}

type sqlExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func writeCachedBalance(ctx context.Context, db sqlExecer, balance Balance, rebuiltAt time.Time) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO balance_projections (holder_id, token_type_id, amount, rebuilt_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(holder_id, token_type_id) DO UPDATE SET
			amount = excluded.amount,
			rebuilt_at = excluded.rebuilt_at`,
		balance.HolderID, balance.TokenTypeID, balance.Amount, formatTime(rebuiltAt))
	if err != nil {
		return fmt.Errorf("write derived balance projection: %w", err)
	}
	return nil
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
