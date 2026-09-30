package inventory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"nutrition-planner/internal/decimalx"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Append(ctx context.Context, workspaceID string, event Event) error {
	if workspaceID == "" {
		return errors.New("workspace_id is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if err := validateEvent(event); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO inventory_events(workspace_id,event_id,kind,item_id,batch_id,amount,unit,recipe_id,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, workspaceID, event.ID, event.Kind, event.ItemID, event.BatchID, event.Amount.String(), event.Unit, event.RecipeID, event.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err == nil {
		return nil
	}
	var existing Event
	row := r.db.QueryRowContext(ctx, `SELECT kind,item_id,batch_id,amount,unit,recipe_id,created_at FROM inventory_events WHERE workspace_id=? AND event_id=?`, workspaceID, event.ID)
	var existingAmount, created string
	if scanErr := row.Scan(&existing.Kind, &existing.ItemID, &existing.BatchID, &existingAmount, &existing.Unit, &existing.RecipeID, &created); scanErr != nil {
		return fmt.Errorf("append inventory event: %w", err)
	}
	existing.Amount, _ = decimalx.Parse(existingAmount)
	if existing.Kind != event.Kind || existing.ItemID != event.ItemID || existing.BatchID != event.BatchID || existing.Amount.String() != event.Amount.String() || existing.Unit != event.Unit || existing.RecipeID != event.RecipeID {
		return fmt.Errorf("idempotency key %q reused with different payload", event.ID)
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id,kind,item_id,batch_id,amount,unit,recipe_id,created_at FROM inventory_events WHERE workspace_id=? ORDER BY created_at,event_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var event Event
		var amount, created string
		if err := rows.Scan(&event.ID, &event.Kind, &event.ItemID, &event.BatchID, &amount, &event.Unit, &event.RecipeID, &created); err != nil {
			return nil, err
		}
		var parseErr error
		event.Amount, parseErr = parseAmount(amount)
		if parseErr != nil {
			return nil, parseErr
		}
		event.CreatedAt, parseErr = time.Parse(time.RFC3339Nano, created)
		if parseErr != nil {
			return nil, parseErr
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) PrepareBatch(ctx context.Context, workspaceID, eventID, batchID, recipeID string, recipeRevision int64, yield decimalx.Decimal, unit string, requirements []Event) (Batch, error) {
	if workspaceID == "" || eventID == "" || batchID == "" || yield.IsUnknown() || yield.IsZero() || unit == "" {
		return Batch{}, errors.New("preparation requires workspace, event, batch, known yield, and unit")
	}
	if existing, err := r.batch(ctx, workspaceID, batchID); err == nil {
		return existing, nil
	}
	for _, requirement := range requirements {
		if requirement.Kind == "" {
			requirement.Kind = Preparation
		}
		if requirement.ID == "" {
			requirement.ID = eventID + ":" + requirement.ItemID
		}
		if err := r.Append(ctx, workspaceID, requirement); err != nil {
			return Batch{}, err
		}
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO inventory_batches(workspace_id,batch_id,recipe_id,recipe_revision,yield_amount,available_amount,unit) VALUES(?,?,?,?,?,?,?)`, workspaceID, batchID, recipeID, recipeRevision, yield.String(), yield.String(), unit)
	if err != nil {
		if existing, readErr := r.batch(ctx, workspaceID, batchID); readErr == nil {
			return existing, nil
		}
		return Batch{}, err
	}
	return Batch{ID: batchID, RecipeID: recipeID, RecipeRevision: recipeRevision, Yield: yield, Available: yield, Unit: unit}, nil
}

func (r *sqliteRepository) ConsumeBatchPortion(ctx context.Context, workspaceID, eventID, batchID string, amount decimalx.Decimal, unit, recipeID string, undo bool) (Batch, error) {
	if eventID == "" || batchID == "" || amount.IsUnknown() || amount.IsZero() || unit == "" {
		return Batch{}, errors.New("batch portion requires event, batch, known amount, and unit")
	}
	if existing, err := r.findEvent(ctx, workspaceID, eventID); err == nil {
		if existing.BatchID != batchID || existing.Amount.String() != amount.String() || existing.Unit != unit {
			return Batch{}, fmt.Errorf("idempotency key %q reused with different payload", eventID)
		}
		return r.batch(ctx, workspaceID, batchID)
	}
	batch, err := r.batch(ctx, workspaceID, batchID)
	if err != nil {
		return Batch{}, err
	}
	if batch.Unit != unit {
		return Batch{}, errors.New("batch portion unit does not match batch")
	}
	if !undo {
		comparison, _ := decimalx.Compare(amount, batch.Available)
		if comparison > 0 {
			return Batch{}, errors.New("batch portion exceeds available yield")
		}
		batch.Available, _ = decimalx.Sub(batch.Available, amount)
	} else {
		batch.Available, _ = decimalx.Add(batch.Available, amount)
	}
	if _, err = r.db.ExecContext(ctx, `UPDATE inventory_batches SET available_amount=? WHERE workspace_id=? AND batch_id=?`, batch.Available.String(), workspaceID, batchID); err != nil {
		return Batch{}, err
	}
	kind := BatchPortion
	if undo {
		kind = PortionUndo
	}
	if err = r.Append(ctx, workspaceID, Event{ID: eventID, Kind: kind, BatchID: batchID, Amount: amount, Unit: unit, RecipeID: recipeID, CreatedAt: time.Now().UTC()}); err != nil {
		return Batch{}, err
	}
	return batch, nil
}

func (r *sqliteRepository) batch(ctx context.Context, workspaceID, batchID string) (Batch, error) {
	var out Batch
	var yield, available string
	err := r.db.QueryRowContext(ctx, `SELECT recipe_id,recipe_revision,yield_amount,available_amount,unit FROM inventory_batches WHERE workspace_id=? AND batch_id=?`, workspaceID, batchID).Scan(&out.RecipeID, &out.RecipeRevision, &yield, &available, &out.Unit)
	if err != nil {
		return Batch{}, fmt.Errorf("batch %q: %w", batchID, err)
	}
	out.ID = batchID
	out.Yield, err = decimalx.Parse(yield)
	if err != nil {
		return Batch{}, err
	}
	out.Available, err = decimalx.Parse(available)
	return out, err
}

func (r *sqliteRepository) findEvent(ctx context.Context, workspaceID, eventID string) (Event, error) {
	var event Event
	var amount, created string
	err := r.db.QueryRowContext(ctx, `SELECT kind,item_id,batch_id,amount,unit,recipe_id,created_at FROM inventory_events WHERE workspace_id=? AND event_id=?`, workspaceID, eventID).Scan(&event.Kind, &event.ItemID, &event.BatchID, &amount, &event.Unit, &event.RecipeID, &created)
	if err != nil {
		return Event{}, err
	}
	event.ID = eventID
	event.Amount, err = decimalx.Parse(amount)
	if err != nil {
		return Event{}, err
	}
	event.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return event, err
}

func validateEvent(event Event) error {
	if event.ID == "" || event.Amount.IsUnknown() || event.Amount.IsZero() || event.Unit == "" {
		return errors.New("event requires id, known non-zero amount, and unit")
	}
	return nil
}

func parseAmount(value string) (decimalx.Decimal, error) { return decimalx.Parse(value) }

func (r *sqliteRepository) StageReceiptProposal(ctx context.Context, workspaceID string, proposal ReceiptProposal) (ReceiptProposal, error) {
	if workspaceID == "" || proposal.SourceID == "" || proposal.TransactionID == "" || proposal.ItemID == "" || proposal.Amount.IsUnknown() || proposal.Amount.IsZero() || proposal.Unit == "" {
		return ReceiptProposal{}, errors.New("receipt proposal requires workspace, source, transaction, item, known amount, and unit")
	}
	proposal.LineKey = strings.TrimSpace(proposal.LineKey)
	if proposal.LineKey == "" {
		proposal.LineKey = normalizeReceiptLine(proposal)
	}
	proposal.ID = receiptProposalID(proposal.SourceID, proposal.TransactionID, proposal.LineKey)
	proposal.EventID = "receipt-purchase:" + proposal.ID
	proposal.Status = ReceiptProposalPending
	if proposal.CreatedAt.IsZero() {
		proposal.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO inventory_receipt_proposals(workspace_id,proposal_id,source_id,transaction_id,line_key,description,item_id,amount,unit,price,status,event_id,created_at,applied_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, workspaceID, proposal.ID, proposal.SourceID, proposal.TransactionID, proposal.LineKey, proposal.Description, proposal.ItemID, proposal.Amount.String(), proposal.Unit, proposal.Price, proposal.Status, proposal.EventID, proposal.CreatedAt.UTC().Format(time.RFC3339Nano), "")
	if err != nil {
		existing, readErr := r.receiptProposal(ctx, workspaceID, proposal.ID)
		if readErr != nil {
			return ReceiptProposal{}, err
		}
		if existing.SourceID != proposal.SourceID || existing.TransactionID != proposal.TransactionID || existing.LineKey != proposal.LineKey || existing.Description != proposal.Description || existing.ItemID != proposal.ItemID || existing.Amount.String() != proposal.Amount.String() || existing.Unit != proposal.Unit || existing.Price != proposal.Price {
			return ReceiptProposal{}, fmt.Errorf("receipt proposal idempotency key %q reused with different payload", proposal.ID)
		}
		return existing, nil
	}
	return proposal, nil
}

func (r *sqliteRepository) ListReceiptProposals(ctx context.Context, workspaceID string) ([]ReceiptProposal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT proposal_id,source_id,transaction_id,line_key,description,item_id,amount,unit,price,status,event_id,created_at,applied_at FROM inventory_receipt_proposals WHERE workspace_id=? ORDER BY created_at,proposal_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReceiptProposal
	for rows.Next() {
		proposal, err := scanReceiptProposal(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, proposal)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) ApplyReceiptProposal(ctx context.Context, workspaceID, proposalID string) (ReceiptProposal, error) {
	proposal, err := r.receiptProposal(ctx, workspaceID, proposalID)
	if err != nil {
		return ReceiptProposal{}, err
	}
	if proposal.Status == ReceiptProposalApplied {
		return proposal, nil
	}
	if err := r.Append(ctx, workspaceID, Event{ID: proposal.EventID, Kind: Purchase, ItemID: proposal.ItemID, Amount: proposal.Amount, Unit: proposal.Unit, CreatedAt: time.Now().UTC()}); err != nil {
		return ReceiptProposal{}, err
	}
	proposal.Status = ReceiptProposalApplied
	proposal.AppliedAt = time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `UPDATE inventory_receipt_proposals SET status=?,applied_at=? WHERE workspace_id=? AND proposal_id=? AND status=?`, proposal.Status, proposal.AppliedAt.Format(time.RFC3339Nano), workspaceID, proposalID, ReceiptProposalPending); err != nil {
		// The event is already durable. A retry will observe the idempotent event.
		return ReceiptProposal{}, err
	}
	return proposal, nil
}

type rowScanner interface{ Scan(...any) error }

func (r *sqliteRepository) receiptProposal(ctx context.Context, workspaceID, proposalID string) (ReceiptProposal, error) {
	row := r.db.QueryRowContext(ctx, `SELECT proposal_id,source_id,transaction_id,line_key,description,item_id,amount,unit,price,status,event_id,created_at,applied_at FROM inventory_receipt_proposals WHERE workspace_id=? AND proposal_id=?`, workspaceID, proposalID)
	return scanReceiptProposal(row)
}

func scanReceiptProposal(row rowScanner) (ReceiptProposal, error) {
	var proposal ReceiptProposal
	var amount, status, created, applied string
	if err := row.Scan(&proposal.ID, &proposal.SourceID, &proposal.TransactionID, &proposal.LineKey, &proposal.Description, &proposal.ItemID, &amount, &proposal.Unit, &proposal.Price, &status, &proposal.EventID, &created, &applied); err != nil {
		return ReceiptProposal{}, err
	}
	var err error
	proposal.Amount, err = decimalx.Parse(amount)
	if err != nil {
		return ReceiptProposal{}, err
	}
	proposal.Status = ReceiptProposalStatus(status)
	proposal.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return ReceiptProposal{}, err
	}
	if applied != "" {
		proposal.AppliedAt, err = time.Parse(time.RFC3339Nano, applied)
		if err != nil {
			return ReceiptProposal{}, err
		}
	}
	return proposal, nil
}

func normalizeReceiptLine(proposal ReceiptProposal) string {
	return strings.Join([]string{strings.ToLower(strings.TrimSpace(proposal.ItemID)), strings.ToLower(strings.TrimSpace(proposal.Description)), proposal.Amount.String(), strings.ToLower(strings.TrimSpace(proposal.Unit)), strings.TrimSpace(proposal.Price)}, "\x1f")
}

func receiptProposalID(sourceID, transactionID, lineKey string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{sourceID, transactionID, lineKey}, "|")))
	return hex.EncodeToString(h[:])
}
