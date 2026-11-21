package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"agent-manager/internal/runmodel"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

// TerminalEventAcceptor is supplied only by an explicitly authorized composition.
// It must preserve these exact bytes/ID and return a durable Events-owner ack.
// A best-effort publisher or generic 409 is not an implementation of this seam.
// Production installs no acceptor, transport, worker, route, or timer.
type TerminalEventAcceptor interface {
	AcceptTerminalEventIntent(context.Context, string, string) (eventbus.DomainEventAcknowledgment, error)
}

// TerminalOutbox uses the immutable receipt itself as pending work. Only a
// separate immutable matching acknowledgment removes it from dispatch selection.
// Cross-owner atomicity is deliberately not claimed: uncertainty replays the same
// intent, and the Events owner must deduplicate even after canonical-event prune.
type TerminalOutbox struct {
	runs     *runRepository
	acceptor TerminalEventAcceptor
}

func NewTerminalOutbox(repos *Repositories, acceptor TerminalEventAcceptor) (*TerminalOutbox, error) {
	if repos == nil || acceptor == nil {
		return nil, errors.New("terminal outbox requires explicit repositories and acceptor")
	}
	runs, ok := repos.Runs.(*runRepository)
	if !ok || runs == nil || runs.db == nil || runs.terminalReceiptScope == nil {
		return nil, errors.New("terminal receipt capture is disabled")
	}
	if err := runs.terminalReceiptScope.Validate(); err != nil {
		return nil, err
	}
	return &TerminalOutbox{runs, acceptor}, nil
}

// Dispatch is finite and caller-driven. It releases all DB readers before the
// acceptor call and never sends within the Run/Task transaction. Return count is
// locally acknowledged receipts, not newly published events or subscriber work.
// Failure leaves this and later intents pending. Concurrent dispatchers may send
// duplicates; identity and both immutable ledgers make those replays harmless.
func (o *TerminalOutbox) Dispatch(ctx context.Context, limit int) (int, error) {
	if o == nil || o.runs == nil || o.acceptor == nil || ctx == nil || limit < 1 || limit > 100 {
		return 0, errors.New("invalid finite terminal outbox dispatch")
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	bound, err := o.sourceBound(ctx)
	if err != nil || !bound {
		return 0, err
	}
	scope := *o.runs.terminalReceiptScope
	rows, err := o.runs.db.QueryContext(ctx, `SELECT r.run_id,r.lifecycle_version FROM run_terminal_receipts r LEFT JOIN run_terminal_event_acknowledgments a ON a.event_intent_id=r.event_intent_id WHERE a.event_intent_id IS NULL AND r.source_instance=? AND r.source_epoch=? ORDER BY r.rowid LIMIT ?`, scope.SourceInstance, scope.SourceEpoch, limit)
	if err != nil {
		return 0, err
	}
	type key struct {
		run     uuid.UUID
		version int64
	}
	var keys []key
	for rows.Next() {
		var k key
		if err := rows.Scan(&k.run, &k.version); err != nil {
			_ = rows.Close()
			return 0, err
		}
		keys = append(keys, k)
	}
	rowErr := rows.Err()
	closeErr := rows.Close()
	if rowErr != nil || closeErr != nil {
		return 0, errors.Join(rowErr, closeErr)
	}
	completed := 0
	for _, k := range keys {
		receipt, err := o.runs.TerminalReceipt(ctx, k.run, k.version)
		if err != nil {
			return completed, err
		}
		// Reject damaged/malformed retained intent before calling any external owner.
		if _, _, err := eventbus.RetainedDomainEventEnvelope(receipt.EventIntentJSON, receipt.EventIntentID); err != nil {
			return completed, err
		}
		ack, err := o.acceptor.AcceptTerminalEventIntent(ctx, receipt.EventIntentJSON, receipt.EventIntentID)
		if err != nil {
			return completed, fmt.Errorf("terminal event acceptance uncertain: %w", err)
		}
		if err := o.retainAcknowledgment(ctx, receipt, ack); err != nil {
			return completed, err
		}
		completed++
	}
	return completed, nil
}

// ValidateSourceBinding fails explicit composition before serving if this
// database already belongs to another immutable installation/epoch. It does not
// bind an empty database or authorize publication.
func (o *TerminalOutbox) ValidateSourceBinding(ctx context.Context) error {
	_, err := o.sourceBound(ctx)
	return err
}

func (o *TerminalOutbox) sourceBound(ctx context.Context) (bool, error) {
	if o == nil || o.runs == nil || o.runs.terminalReceiptScope == nil || ctx == nil {
		return false, errors.New("terminal source binding unavailable")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	scope := *o.runs.terminalReceiptScope
	var instance, epoch string
	err := o.runs.db.QueryRowContext(ctx, `SELECT source_instance,source_epoch FROM run_terminal_receipt_source WHERE singleton=1`).Scan(&instance, &epoch)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if instance != scope.SourceInstance || epoch != scope.SourceEpoch {
		return false, errors.New("terminal outbox source binding mismatch")
	}
	return true, nil
}

func (o *TerminalOutbox) retainAcknowledgment(ctx context.Context, receipt runmodel.TerminalReceipt, ack eventbus.DomainEventAcknowledgment) error {
	if err := eventbus.ValidateDomainEventAcknowledgment(receipt.EventIntentJSON, receipt.EventIntentID, ack); err != nil {
		return err
	}
	encoded, err := json.Marshal(ack)
	if err != nil {
		return err
	}
	tx, err := o.runs.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO run_terminal_event_acknowledgments (event_intent_id,acknowledgment_json) VALUES (?,?) ON CONFLICT (event_intent_id) DO NOTHING`, receipt.EventIntentID, string(encoded)); err != nil {
		return err
	}
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT acknowledgment_json FROM run_terminal_event_acknowledgments WHERE event_intent_id=?`, receipt.EventIntentID).Scan(&stored); err != nil {
		return err
	}
	if stored != string(encoded) {
		return errors.New("conflicting terminal event acknowledgment")
	}
	return tx.Commit()
}

// TerminalEventAcknowledgment returns separately retained Events acceptance. It
// is not a current selected-event read, freshness/permission proof, Work binding,
// Flywheel delivery, or authority to act on the underlying terminal observation.
func (r *runRepository) TerminalEventAcknowledgment(ctx context.Context, runID uuid.UUID, version int64) (eventbus.DomainEventAcknowledgment, error) {
	receipt, err := r.TerminalReceipt(ctx, runID, version)
	if err != nil {
		return eventbus.DomainEventAcknowledgment{}, err
	}
	var encoded string
	if err := r.db.QueryRowContext(ctx, `SELECT acknowledgment_json FROM run_terminal_event_acknowledgments WHERE event_intent_id=?`, receipt.EventIntentID).Scan(&encoded); err != nil {
		return eventbus.DomainEventAcknowledgment{}, err
	}
	var ack eventbus.DomainEventAcknowledgment
	if err := json.Unmarshal([]byte(encoded), &ack); err != nil {
		return eventbus.DomainEventAcknowledgment{}, err
	}
	if err := eventbus.ValidateDomainEventAcknowledgment(receipt.EventIntentJSON, receipt.EventIntentID, ack); err != nil {
		return eventbus.DomainEventAcknowledgment{}, err
	}
	return ack, nil
}
