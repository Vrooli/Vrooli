package duplex

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Journal stores only correlation and provider-receipt metadata. It never
// stores bearer tokens, credentials, audio, or transcripts. LPBS remains the
// authoritative ledger and its lifecycle API remains the recovery authority.
type Journal interface {
	Create(context.Context, JournalOperation) error
	Transition(context.Context, string, string, string, ProviderUsage) error
	Recoverable(context.Context, string) ([]JournalOperation, error)
}

type RecoveryJournal interface {
	Journal
	MarkRecovered(context.Context, JournalOperation, string, string) error
}

type JournalOperation struct {
	OperationID, OwnerKey, ProviderID, ModelID, Phase, ProviderReceiptID string
	BudgetMicros, CostMicros, ActiveSeconds, PausedSeconds               int64
	CreatedAt, UpdatedAt                                                 time.Time
}

// SQLJournal is owned by the duplex domain and can use the scenario's
// single-writer SQLite pool. The durable record is a recovery index, not a
// second account balance.
type SQLJournal struct{ db *sql.DB }

func NewSQLJournal(db *sql.DB) (*SQLJournal, error) {
	if db == nil {
		return nil, errors.New("duplex: SQL journal database is required")
	}
	return &SQLJournal{db: db}, nil
}

func (j *SQLJournal) Create(ctx context.Context, op JournalOperation) error {
	if ctx == nil || op.OperationID == "" || op.OwnerKey == "" || op.ProviderID == "" || op.ModelID == "" || op.BudgetMicros <= 0 {
		return errors.New("duplex: invalid accounting journal intent")
	}
	now := time.Now().UTC()
	if op.CreatedAt.IsZero() {
		op.CreatedAt = now
	}
	op.UpdatedAt = now
	_, err := j.db.ExecContext(ctx, `INSERT INTO duplex_accounting_operations(operation_id,owner_key,provider_id,model_id,budget_micros,phase,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(operation_id) DO NOTHING`, op.OperationID, op.OwnerKey, op.ProviderID, op.ModelID, op.BudgetMicros, "reserve_pending", op.CreatedAt.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("journal duplex reservation intent: %w", err)
	}
	// Retried intent creation is safe only when the existing operation is the
	// same owner's exact reservation. In particular, never let an operation ID
	// collision rebind an existing hold to another owner or budget.
	var owner, provider, model string
	var budget int64
	if err := j.db.QueryRowContext(ctx, `SELECT owner_key,provider_id,model_id,budget_micros FROM duplex_accounting_operations WHERE operation_id=?`, op.OperationID).Scan(&owner, &provider, &model, &budget); err != nil {
		return fmt.Errorf("verify duplex reservation intent: %w", err)
	}
	if owner != op.OwnerKey || provider != op.ProviderID || model != op.ModelID || budget != op.BudgetMicros {
		return errors.New("duplex: accounting operation ID is already bound to a different reservation")
	}
	return nil
}

// CountCreatedBetween returns only the aggregate number of journal operations
// created in the half-open interval [from,to). It deliberately exposes no
// operation, owner, provider, or receipt metadata to the measures surface.
func (j *SQLJournal) CountCreatedBetween(ctx context.Context, from, to time.Time) (int64, error) {
	// SQLite text ordering differs for RFC3339 values with and without
	// fractional seconds ("Z" sorts after "."). Read only the timestamp column
	// from a safely widened lexical range, then apply the exact half-open window
	// with Go's RFC3339 parser.
	rows, err := j.db.QueryContext(ctx, `SELECT created_at FROM duplex_accounting_operations WHERE created_at >= ? AND created_at <= ?`, from.UTC().Add(-time.Second).Format(time.RFC3339Nano), to.UTC().Add(time.Second).Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("query duplex accounting operation timestamps: %w", err)
	}
	defer rows.Close()
	var count int64
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, fmt.Errorf("scan duplex accounting operation timestamp: %w", err)
		}
		created, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return 0, fmt.Errorf("parse duplex accounting operation timestamp: %w", err)
		}
		if !created.Before(from) && created.Before(to) {
			count++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate duplex accounting operation timestamps: %w", err)
	}
	return count, nil
}

// Transition updates one known operation. Pending phases are written before
// remote calls, so a restart can distinguish a lost response from no attempt.
func (j *SQLJournal) Transition(ctx context.Context, id, from, to string, usage ProviderUsage) error {
	if ctx == nil || id == "" || !validJournalTransition(from, to) {
		return errors.New("duplex: invalid accounting journal transition")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := j.db.ExecContext(ctx, `UPDATE duplex_accounting_operations SET phase=?, provider_receipt_id=CASE WHEN ?='' THEN provider_receipt_id ELSE ? END, cost_micros=CASE WHEN ?='' THEN cost_micros ELSE ? END, active_seconds=CASE WHEN ?='' THEN active_seconds ELSE ? END, paused_seconds=CASE WHEN ?='' THEN paused_seconds ELSE ? END, updated_at=? WHERE operation_id=? AND phase=?`, to, usage.ProviderReceiptID, usage.ProviderReceiptID, usage.ProviderReceiptID, usage.CostMicros, usage.ProviderReceiptID, usage.ActiveSeconds, usage.ProviderReceiptID, usage.PausedSeconds, now, id, from)
	if err != nil {
		return fmt.Errorf("update duplex accounting journal: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("duplex: accounting journal phase changed or operation missing")
	}
	return nil
}

// MarkRecovered applies only an owner-confirmed terminal phase after LPBS's
// authenticated recovery endpoint has reconciled its wallet and lifecycle.
func (j *SQLJournal) MarkRecovered(ctx context.Context, op JournalOperation, phase, receipt string) error {
	if phase != "settled" && phase != "released" && phase != "expired" {
		return errors.New("duplex: recovery did not confirm a terminal LPBS phase")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if op.OwnerKey == "" {
		return errors.New("duplex: recovered operation is missing its owner key")
	}
	res, err := j.db.ExecContext(ctx, `UPDATE duplex_accounting_operations SET phase=?, provider_receipt_id=CASE WHEN ?='' THEN provider_receipt_id ELSE ? END, updated_at=? WHERE operation_id=? AND owner_key=? AND phase=?`, phase, receipt, receipt, now, op.OperationID, op.OwnerKey, op.Phase)
	if err != nil {
		return fmt.Errorf("mark recovered duplex accounting operation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("duplex: recovery journal phase changed or operation missing")
	}
	return nil
}

func validJournalTransition(from, to string) bool {
	switch from {
	case "reserve_pending":
		return to == "reserved" || to == "uncertain"
	case "reserved":
		return to == "dispatch_pending" || to == "released" || to == "expired"
	case "dispatch_pending":
		return to == "dispatched" || to == "uncertain"
	case "dispatched":
		return to == "settlement_pending" || to == "uncertain"
	case "settlement_pending":
		return to == "settled" || to == "uncertain"
	default:
		return false
	}
}

func (j *SQLJournal) Recoverable(ctx context.Context, ownerKey string) ([]JournalOperation, error) {
	if ctx == nil || ownerKey == "" {
		return nil, errors.New("duplex: verified owner key required for recovery listing")
	}
	rows, err := j.db.QueryContext(ctx, `SELECT operation_id,owner_key,provider_id,model_id,budget_micros,phase,provider_receipt_id,cost_micros,active_seconds,paused_seconds,created_at,updated_at FROM duplex_accounting_operations WHERE owner_key=? AND phase NOT IN ('settled','released','expired') ORDER BY created_at`, ownerKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JournalOperation{}
	for rows.Next() {
		var op JournalOperation
		var created, updated string
		if err := rows.Scan(&op.OperationID, &op.OwnerKey, &op.ProviderID, &op.ModelID, &op.BudgetMicros, &op.Phase, &op.ProviderReceiptID, &op.CostMicros, &op.ActiveSeconds, &op.PausedSeconds, &created, &updated); err != nil {
			return nil, err
		}
		op.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("parse duplex accounting operation creation time: %w", err)
		}
		op.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return nil, fmt.Errorf("parse duplex accounting operation update time: %w", err)
		}
		out = append(out, op)
	}
	return out, rows.Err()
}
