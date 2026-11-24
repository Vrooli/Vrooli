package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	"google.golang.org/protobuf/proto"
)

// CreateWorkspaceWithConversionReceipt commits the converted aggregate and
// its recovery evidence together. It preserves the converter-assigned ID so
// the target hash remains valid, and a repeat with identical evidence returns
// the already committed pair.
func (r *SQLiteRepository) CreateWorkspaceWithConversionReceipt(ctx context.Context, workspace *workspacev1.Workspace, input ConversionReceiptInput) (*workspacev1.Workspace, ConversionReceipt, error) {
	if workspace == nil || workspace.GetId() == "" || workspace.GetBindingScenario() == "" || workspace.GetBindingPage() == "" {
		return nil, ConversionReceipt{}, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "workspace", Message: "a bound workspace with an ID is required"}}}
	}
	input.Scenario = strings.TrimSpace(input.Scenario)
	input.Page = strings.TrimSpace(input.Page)
	if input.Scenario != workspace.GetBindingScenario() || input.Page != workspace.GetBindingPage() || input.WorkspaceID != workspace.GetId() || input.SourceHash == "" || input.TargetHash == "" || !json.Valid([]byte(input.ReceiptJSON)) {
		return nil, ConversionReceipt{}, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "conversion_receipt", Message: "receipt identity must match the bound workspace and include hashes and valid JSON"}}}
	}
	// A previously committed pair is the normal retry path after a lost reply.
	if stored, err := r.GetConversionReceipt(ctx, input.Scenario, input.Page); err == nil {
		if stored.WorkspaceID == input.WorkspaceID && stored.SourceHash == input.SourceHash && stored.TargetHash == input.TargetHash && stored.ReceiptJSON == input.ReceiptJSON {
			current, getErr := r.Get(ctx, stored.WorkspaceID)
			return current, stored, getErr
		}
		return nil, ConversionReceipt{}, &ConversionReceiptConflictError{Existing: stored}
	} else if !errors.Is(err, ErrNotFound) {
		return nil, ConversionReceipt{}, err
	}
	prepared := proto.Clone(workspace).(*workspacev1.Workspace)
	now := r.timestamp()
	if prepared.CreatedAt == "" {
		prepared.CreatedAt = now
	}
	if prepared.UpdatedAt == "" {
		prepared.UpdatedAt = now
	}
	prepared.Revision = 1
	if err := normalizeAggregate(prepared, true); err != nil {
		return nil, ConversionReceipt{}, err
	}
	if err := invalidFromIssues(validationIssues(prepared)); err != nil {
		return nil, ConversionReceipt{}, err
	}
	if input.CreatedAt == "" {
		input.CreatedAt = now
	}
	receiptID, err := newID("ir_")
	if err != nil {
		return nil, ConversionReceipt{}, fmt.Errorf("assign conversion receipt identity: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ConversionReceipt{}, fmt.Errorf("begin workspace conversion commit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertWorkspace(ctx, tx, prepared); err != nil {
		var existingID string
		lookupErr := tx.QueryRowContext(ctx, `SELECT id FROM workspaces WHERE binding_scenario=? AND binding_page=?`, input.Scenario, input.Page).Scan(&existingID)
		if lookupErr == nil {
			// A concurrent request may have committed the exact workspace and
			// receipt after our initial read. Resolve that race to the committed
			// pair instead of exposing a binding conflict for a safe retry.
			existing, receiptErr := scanConversionReceipt(ctx, tx, input.Scenario, input.Page)
			if receiptErr == nil {
				if existing.WorkspaceID == input.WorkspaceID && existing.SourceHash == input.SourceHash && existing.TargetHash == input.TargetHash && existing.ReceiptJSON == input.ReceiptJSON {
					_ = tx.Rollback()
					current, getErr := r.Get(ctx, existing.WorkspaceID)
					return current, existing, getErr
				}
				return nil, ConversionReceipt{}, &ConversionReceiptConflictError{Existing: existing}
			}
			if !errors.Is(receiptErr, sql.ErrNoRows) {
				return nil, ConversionReceipt{}, fmt.Errorf("read conversion receipt after binding conflict: %w", receiptErr)
			}
			return nil, ConversionReceipt{}, fmt.Errorf("%w: %s", ErrBindingConflict, existingID)
		}
		return nil, ConversionReceipt{}, err
	}
	if err := insertNested(ctx, tx, prepared); err != nil {
		return nil, ConversionReceipt{}, err
	}
	snapshot, err := marshalWorkspace(prepared)
	if err != nil {
		return nil, ConversionReceipt{}, err
	}
	if err := insertRevision(ctx, tx, prepared.Id, prepared.Revision, snapshot, "", now); err != nil {
		return nil, ConversionReceipt{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_conversion_receipts
		(id,scenario,page,workspace_id,source_hash,target_hash,receipt_json,created_at)
		VALUES (?,?,?,?,?,?,?,?)`, receiptID, input.Scenario, input.Page, input.WorkspaceID, input.SourceHash, input.TargetHash, input.ReceiptJSON, input.CreatedAt); err != nil {
		return nil, ConversionReceipt{}, fmt.Errorf("insert conversion receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, ConversionReceipt{}, fmt.Errorf("commit workspace conversion: %w", err)
	}
	if err := r.record(ctx, "workspace.created", prepared.Id, nil, prepared); err != nil {
		return nil, ConversionReceipt{}, err
	}
	receipt := ConversionReceipt{ID: receiptID, Scenario: input.Scenario, Page: input.Page, WorkspaceID: input.WorkspaceID, SourceHash: input.SourceHash, TargetHash: input.TargetHash, ReceiptJSON: input.ReceiptJSON, CreatedAt: input.CreatedAt}
	return prepared, receipt, nil
}

// CreateConversionReceipt stores immutable recovery evidence. Repeating the
// same request is safe; a different conversion for the same source identity
// returns the stored receipt in a typed conflict for explicit reconciliation.
func (r *SQLiteRepository) CreateConversionReceipt(ctx context.Context, input ConversionReceiptInput) (ConversionReceipt, error) {
	input.Scenario = strings.TrimSpace(input.Scenario)
	input.Page = strings.TrimSpace(input.Page)
	if input.Scenario == "" || input.Page == "" || input.WorkspaceID == "" || input.SourceHash == "" || input.TargetHash == "" || !json.Valid([]byte(input.ReceiptJSON)) {
		return ConversionReceipt{}, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "conversion_receipt", Message: "scenario, page, workspace, hashes, and valid receipt JSON are required"}}}
	}
	if input.CreatedAt == "" {
		input.CreatedAt = r.timestamp()
	}
	id, err := newID("ir_")
	if err != nil {
		return ConversionReceipt{}, fmt.Errorf("assign conversion receipt identity: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO workspace_conversion_receipts
		(id,scenario,page,workspace_id,source_hash,target_hash,receipt_json,created_at)
		VALUES (?,?,?,?,?,?,?,?)`, id, input.Scenario, input.Page, input.WorkspaceID, input.SourceHash, input.TargetHash, input.ReceiptJSON, input.CreatedAt)
	if err == nil {
		return ConversionReceipt{ID: id, Scenario: input.Scenario, Page: input.Page, WorkspaceID: input.WorkspaceID, SourceHash: input.SourceHash, TargetHash: input.TargetHash, ReceiptJSON: input.ReceiptJSON, CreatedAt: input.CreatedAt}, nil
	}
	// A concurrent insert can win between the insert and lookup. The unique
	// key establishes a single durable outcome, then this branch reconciles it.
	existing, lookupErr := r.GetConversionReceipt(ctx, input.Scenario, input.Page)
	if lookupErr == nil {
		if existing.WorkspaceID == input.WorkspaceID && existing.SourceHash == input.SourceHash && existing.TargetHash == input.TargetHash && existing.ReceiptJSON == input.ReceiptJSON {
			return existing, nil
		}
		return ConversionReceipt{}, &ConversionReceiptConflictError{Existing: existing}
	}
	if !errors.Is(lookupErr, ErrNotFound) {
		return ConversionReceipt{}, fmt.Errorf("insert conversion receipt: %w (lookup existing: %v)", err, lookupErr)
	}
	return ConversionReceipt{}, fmt.Errorf("insert conversion receipt: %w", err)
}

func (r *SQLiteRepository) GetConversionReceipt(ctx context.Context, scenario, page string) (ConversionReceipt, error) {
	receipt, err := scanConversionReceipt(ctx, r.db, scenario, page)
	if errors.Is(err, sql.ErrNoRows) {
		return ConversionReceipt{}, ErrNotFound
	}
	if err != nil {
		return ConversionReceipt{}, fmt.Errorf("get conversion receipt: %w", err)
	}
	return receipt, nil
}

func scanConversionReceipt(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, scenario, page string) (ConversionReceipt, error) {
	var receipt ConversionReceipt
	err := queryer.QueryRowContext(ctx, `SELECT id,scenario,page,workspace_id,source_hash,target_hash,receipt_json,created_at
		FROM workspace_conversion_receipts WHERE scenario=? AND page=?`, scenario, page).Scan(
		&receipt.ID, &receipt.Scenario, &receipt.Page, &receipt.WorkspaceID, &receipt.SourceHash,
		&receipt.TargetHash, &receipt.ReceiptJSON, &receipt.CreatedAt)
	return receipt, err
}
