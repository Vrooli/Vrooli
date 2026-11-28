package continuity

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"web-console/internal/dbx"
)

// Receipt is the durable audit result of one lifecycle command. OperationID
// is the caller's replay key, not a timestamp, so retries can return the same
// outcome without issuing a second destructive effect.
type Receipt struct {
	OperationID string
	SessionID   string
	ActorKind   string
	ActorID     string
	Command     string
	FromState   State
	ToState     State
	ReasonCode  string
	Status      string
	ErrorCode   string
	CreatedAt   time.Time
	CompletedAt time.Time
}

type Ledger interface {
	Put(ctx context.Context, receipt Receipt) (Receipt, error)
	Complete(ctx context.Context, operationID, status, errorCode string, completedAt time.Time) (Receipt, error)
	Get(ctx context.Context, operationID string) (Receipt, error)
}

// RetryableLedger optionally clears a recorded failure so the same operation
// id can be attempted again. A failed receipt describes an attempt that
// changed nothing, so replaying it forever turns one bad moment into a
// permanent refusal — a Close button that is dead for the rest of the
// session's life. Callers that own a retryable command reopen instead.
type RetryableLedger interface {
	Reopen(ctx context.Context, operationID string) (Receipt, error)
}

// ResultLedger optionally records the resource produced by a lifecycle
// operation. Recover uses this to make a successful replay return the same
// replacement session after the source row has been dismissed.
type ResultLedger interface {
	SetActorID(ctx context.Context, operationID, actorID string) (Receipt, error)
}

type SQLLedger struct {
	db dbx.Handle
	// SQLite permits one writer at a time. Serialize operations issued through
	// one ledger so concurrent lifecycle callers do not turn a valid idempotent
	// retry into SQLITE_BUSY before the database-level retry window can help.
	mu sync.Mutex
}

func NewSQLLedger(db dbx.Handle) *SQLLedger { return &SQLLedger{db: db} }

func (l *SQLLedger) Put(ctx context.Context, r Receipt) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if r.OperationID == "" || r.SessionID == "" || r.Command == "" {
		return Receipt{}, fmt.Errorf("operation_id, session_id and command are required")
	}
	_, err := execWithBusyRetry(ctx, l.db, `INSERT OR IGNORE INTO session_lifecycle_receipts
		(operation_id, session_id, actor_kind, actor_id, command, from_state,
		 to_state, reason_code, status, error_code, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.OperationID, r.SessionID, r.ActorKind, r.ActorID, r.Command,
		string(r.FromState), string(r.ToState), r.ReasonCode, r.Status,
		r.ErrorCode, r.CreatedAt.UTC().Format(time.RFC3339Nano), formatReceiptTime(r.CompletedAt))
	if err != nil {
		return Receipt{}, err
	}
	return getWithBusyRetry(ctx, l.db, r.OperationID)
}

func (l *SQLLedger) Get(ctx context.Context, operationID string) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return getWithBusyRetry(ctx, l.db, operationID)
}

func getWithBusyRetry(ctx context.Context, db dbx.Handle, operationID string) (Receipt, error) {
	var r Receipt
	var fromState, toState, createdAt string
	var completedAt sql.NullString
	var err error
	for attempt := 0; ; attempt++ {
		err = db.QueryRowContext(ctx, `SELECT operation_id, session_id, actor_kind,
		actor_id, command, from_state, to_state, reason_code, status, error_code,
		created_at, completed_at FROM session_lifecycle_receipts WHERE operation_id = ?`, operationID).
			Scan(&r.OperationID, &r.SessionID, &r.ActorKind, &r.ActorID, &r.Command,
				&fromState, &toState, &r.ReasonCode, &r.Status, &r.ErrorCode,
				&createdAt, &completedAt)
		if !isSQLiteBusy(err) || attempt >= ledgerBusyRetryLimit {
			break
		}
		if err := waitForLedgerRetry(ctx, attempt); err != nil {
			return Receipt{}, err
		}
	}
	if err != nil {
		return Receipt{}, err
	}
	r.FromState, r.ToState = State(fromState), State(toState)
	r.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Receipt{}, fmt.Errorf("parse receipt created_at: %w", err)
	}
	if completedAt.Valid && completedAt.String != "" {
		r.CompletedAt, err = time.Parse(time.RFC3339Nano, completedAt.String)
		if err != nil {
			return Receipt{}, fmt.Errorf("parse receipt completed_at: %w", err)
		}
	}
	return r, nil
}

const ledgerBusyRetryLimit = 20

func execWithBusyRetry(ctx context.Context, db dbx.Handle, query string, args ...any) (sql.Result, error) {
	for attempt := 0; ; attempt++ {
		result, err := db.ExecContext(ctx, query, args...)
		if !isSQLiteBusy(err) || attempt >= ledgerBusyRetryLimit {
			return result, err
		}
		if err := waitForLedgerRetry(ctx, attempt); err != nil {
			return nil, err
		}
	}
}

func isSQLiteBusy(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "sqlite_busy")
}

func waitForLedgerRetry(ctx context.Context, attempt int) error {
	delay := time.Duration(attempt+1) * 5 * time.Millisecond
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *SQLLedger) Complete(ctx context.Context, operationID, status, errorCode string, completedAt time.Time) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if strings.TrimSpace(operationID) == "" {
		return Receipt{}, fmt.Errorf("operation_id is required")
	}
	_, err := execWithBusyRetry(ctx, l.db, `UPDATE session_lifecycle_receipts SET status = ?, error_code = ?, completed_at = ? WHERE operation_id = ? AND completed_at IS NULL`, status, errorCode, formatReceiptTime(completedAt), operationID)
	if err != nil {
		return Receipt{}, err
	}
	return getWithBusyRetry(ctx, l.db, operationID)
}

// Reopen permits a failed resumable operation to retry from its durable
// progress cursor without changing its idempotency identity.
func (l *SQLLedger) Reopen(ctx context.Context, operationID string) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if strings.TrimSpace(operationID) == "" {
		return Receipt{}, fmt.Errorf("operation_id is required")
	}
	if _, err := execWithBusyRetry(ctx, l.db, `UPDATE session_lifecycle_receipts SET status='pending', error_code='', completed_at=NULL WHERE operation_id=? AND status='failed'`, operationID); err != nil {
		return Receipt{}, err
	}
	return getWithBusyRetry(ctx, l.db, operationID)
}

func (l *SQLLedger) SetActorID(ctx context.Context, operationID, actorID string) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if strings.TrimSpace(operationID) == "" || strings.TrimSpace(actorID) == "" {
		return Receipt{}, fmt.Errorf("operation_id and actor_id are required")
	}
	_, err := execWithBusyRetry(ctx, l.db, `UPDATE session_lifecycle_receipts SET actor_id = ? WHERE operation_id = ?`, actorID, operationID)
	if err != nil {
		return Receipt{}, err
	}
	return getWithBusyRetry(ctx, l.db, operationID)
}

func formatReceiptTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

var _ = sql.ErrNoRows
