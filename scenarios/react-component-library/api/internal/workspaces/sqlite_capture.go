package workspaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

type renderedState struct {
	ID       string `json:"id"`
	ArgsJSON string `json:"argsJson"`
}
type renderedDisplay struct {
	ID     string          `json:"id"`
	Device int32           `json:"device"`
	Width  int32           `json:"width"`
	Height int32           `json:"height"`
	States []renderedState `json:"states"`
}
type renderedPage struct {
	ID       string            `json:"id"`
	SpecJSON string            `json:"specJson"`
	Displays []renderedDisplay `json:"displays"`
}
type renderedWorkspace struct {
	ThemeJSON   string         `json:"themeJson"`
	OptionsJSON string         `json:"optionsJson"`
	Pages       []renderedPage `json:"pages"`
}

func workspaceCompositionChanged(before, after *workspacev1.Workspace) (bool, error) {
	project := func(workspace *workspacev1.Workspace) renderedWorkspace {
		out := renderedWorkspace{ThemeJSON: workspace.GetThemeJson(), OptionsJSON: workspace.GetOptionsJson()}
		for _, page := range workspace.GetPages() {
			p := renderedPage{ID: page.GetId(), SpecJSON: page.GetSpecJson()}
			for _, display := range page.GetDisplays() {
				d := renderedDisplay{ID: display.GetId(), Device: int32(display.GetDevice()), Width: display.GetWidth(), Height: display.GetHeight()}
				for _, state := range display.GetStates() {
					d.States = append(d.States, renderedState{ID: state.GetId(), ArgsJSON: state.GetArgsJson()})
				}
				p.Displays = append(p.Displays, d)
			}
			out.Pages = append(out.Pages, p)
		}
		return out
	}
	left, err := json.Marshal(project(before))
	if err != nil {
		return false, fmt.Errorf("encode previous workspace composition: %w", err)
	}
	right, err := json.Marshal(project(after))
	if err != nil {
		return false, fmt.Errorf("encode next workspace composition: %w", err)
	}
	return string(left) != string(right), nil
}

func invalidateWorkspaceCaptures(workspace *workspacev1.Workspace) {
	for _, page := range workspace.GetPages() {
		for _, display := range page.GetDisplays() {
			for _, state := range display.GetStates() {
				if state.GetStatus() == workspacev1.StateStatus_STATE_STATUS_CAPTURED || state.GetCaptureId() != "" {
					state.Status = workspacev1.StateStatus_STATE_STATUS_READY
					state.CaptureId = ""
				}
			}
		}
	}
}

// AttachCaptureEvidence persists the BAS artifact references, captured state
// marker, revision snapshot and workspace revision in one SQLite transaction.
func (r *SQLiteRepository) AttachCaptureEvidence(ctx context.Context, workspaceID, stateID string, baseRevision int32, attemptID, requestToken, captureID, evidenceJSON, capturedAt string) (*workspacev1.Workspace, error) {
	if workspaceID == "" || stateID == "" || requestToken == "" || captureID == "" || !json.Valid([]byte(evidenceJSON)) {
		return nil, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "capture_evidence", Message: "workspace, state, token, capture identity, and JSON evidence are required"}}}
	}
	receiptID, err := newID("cr_")
	if err != nil {
		return nil, fmt.Errorf("assign capture receipt identity: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin workspace capture commit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	current, err := r.get(ctx, tx, workspaceID)
	if err != nil {
		return nil, err
	}
	if current.GetProvenance() == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
		return nil, ErrCurated
	}
	if current.GetRevision() != baseRevision {
		return nil, &ConflictError{Current: current}
	}
	found := false
	for _, page := range current.GetPages() {
		for _, display := range page.GetDisplays() {
			for _, state := range display.GetStates() {
				if state.GetId() == stateID {
					found = true
				}
			}
		}
	}
	if !found {
		return nil, ErrNotFound
	}
	now := r.timestamp()
	nextRevision := current.Revision + 1
	result, err := tx.ExecContext(ctx, `UPDATE workspaces SET revision=?, updated_at=? WHERE id=? AND revision=?`, nextRevision, now, workspaceID, baseRevision)
	if err != nil {
		return nil, fmt.Errorf("advance workspace capture revision: %w", err)
	}
	updatedRows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("check workspace capture revision update: %w", err)
	}
	if updatedRows != 1 {
		raced, err := r.get(ctx, tx, workspaceID)
		if err != nil {
			return nil, err
		}
		return nil, &ConflictError{Current: raced}
	}
	stateResult, err := tx.ExecContext(ctx, `UPDATE workspace_states SET status='captured', capture_id=?
		WHERE id=? AND display_id IN (
			SELECT d.id FROM workspace_displays d JOIN workspace_pages p ON p.id=d.page_id WHERE p.workspace_id=?
		)`, captureID, stateID, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("mark workspace state captured: %w", err)
	}
	stateRows, err := stateResult.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("check captured workspace state update: %w", err)
	}
	if stateRows != 1 {
		return nil, ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_state_capture_evidence
		(workspace_id,state_id,request_token,source_revision,capture_id,evidence_json,captured_at)
		VALUES (?,?,?,?,?,?,?) ON CONFLICT(workspace_id,state_id) DO UPDATE SET
		request_token=excluded.request_token,source_revision=excluded.source_revision,capture_id=excluded.capture_id,evidence_json=excluded.evidence_json,captured_at=excluded.captured_at`,
		workspaceID, stateID, requestToken, baseRevision, captureID, evidenceJSON, capturedAt); err != nil {
		return nil, fmt.Errorf("persist workspace state capture artifacts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_capture_receipts
		(id,workspace_id,state_id,source_revision,capture_id,attempt_id,request_token,evidence_json,captured_at)
		VALUES (?,?,?,?,?,?,?,?,?)`, receiptID, workspaceID, stateID, baseRevision, captureID, attemptID, requestToken, evidenceJSON, capturedAt); err != nil {
		return nil, fmt.Errorf("append immutable workspace capture receipt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE workspace_state_capture_attempts SET state='completed',error='',updated_at=? WHERE workspace_id=? AND state_id=? AND source_revision=? AND attempt_id=? AND state='in_progress'`, now, workspaceID, stateID, baseRevision, attemptID); err != nil {
		return nil, fmt.Errorf("complete workspace state capture attempt: %w", err)
	}
	updated, err := r.get(ctx, tx, workspaceID)
	if err != nil {
		return nil, err
	}
	snapshot, err := marshalWorkspace(updated)
	if err != nil {
		return nil, err
	}
	if err := insertRevision(ctx, tx, workspaceID, updated.Revision, snapshot, "capture", now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit workspace capture evidence: %w", err)
	}
	if err := r.record(ctx, "workspace.state_captured", workspaceID, current, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (r *SQLiteRepository) BeginCaptureAttempt(ctx context.Context, workspaceID, stateID string, sourceRevision int32, attemptID, requestToken string) (CaptureAttempt, bool, error) {
	if workspaceID == "" || stateID == "" || attemptID == "" || requestToken == "" || sourceRevision <= 0 {
		return CaptureAttempt{}, false, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "capture_attempt", Message: "workspace, state, revision, attempt identity, and request token are required"}}}
	}
	var created CaptureAttempt
	err := r.db.QueryRowContext(ctx, `INSERT INTO workspace_state_capture_attempts
		(workspace_id,state_id,source_revision,attempt_id,request_token,state,error,created_at,updated_at)
		VALUES (?,?,?,?,?,'in_progress','',?,?) ON CONFLICT DO NOTHING RETURNING workspace_id,state_id,source_revision,attempt_id,request_token,state,error,created_at,updated_at`,
		workspaceID, stateID, sourceRevision, attemptID, requestToken, r.timestamp(), r.timestamp()).Scan(
		&created.WorkspaceID, &created.StateID, &created.SourceRevision, &created.AttemptID, &created.RequestToken, &created.State, &created.Error, &created.CreatedAt, &created.UpdatedAt)
	if err == nil {
		return created, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CaptureAttempt{}, false, fmt.Errorf("reserve workspace state capture attempt: %w", err)
	}
	var existing CaptureAttempt
	err = r.db.QueryRowContext(ctx, `SELECT workspace_id,state_id,source_revision,attempt_id,request_token,state,error,created_at,updated_at FROM workspace_state_capture_attempts WHERE workspace_id=? AND state_id=? AND source_revision=? AND attempt_id=?`, workspaceID, stateID, sourceRevision, attemptID).Scan(
		&existing.WorkspaceID, &existing.StateID, &existing.SourceRevision, &existing.AttemptID, &existing.RequestToken, &existing.State, &existing.Error, &existing.CreatedAt, &existing.UpdatedAt)
	if err == nil {
		if existing.RequestToken != requestToken {
			return CaptureAttempt{}, false, fmt.Errorf("capture attempt identity was reused with a different request token")
		}
		return existing, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return CaptureAttempt{}, false, fmt.Errorf("read workspace state capture attempt: %w", err)
	}
	return CaptureAttempt{}, false, ErrCaptureInProgress
}

func (r *SQLiteRepository) GetActiveCaptureAttempt(ctx context.Context, workspaceID, stateID string, sourceRevision int32) (CaptureAttempt, error) {
	var attempt CaptureAttempt
	err := r.db.QueryRowContext(ctx, `SELECT workspace_id,state_id,source_revision,attempt_id,request_token,state,error,created_at,updated_at
		FROM workspace_state_capture_attempts WHERE workspace_id=? AND state_id=? AND source_revision=? AND state='in_progress'`,
		workspaceID, stateID, sourceRevision).Scan(
		&attempt.WorkspaceID, &attempt.StateID, &attempt.SourceRevision, &attempt.AttemptID, &attempt.RequestToken, &attempt.State, &attempt.Error, &attempt.CreatedAt, &attempt.UpdatedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return CaptureAttempt{}, fmt.Errorf("load active workspace capture attempt: %w", err)
	}
	return attempt, err
}

func (r *SQLiteRepository) FinishCaptureAttempt(ctx context.Context, workspaceID, stateID string, sourceRevision int32, attemptID, state, detail string) error {
	if state != "failed" && state != "completed" {
		return fmt.Errorf("capture attempt terminal state must be failed or completed")
	}
	if len(detail) > 1000 {
		detail = detail[:1000]
	}
	_, err := r.db.ExecContext(ctx, `UPDATE workspace_state_capture_attempts SET state=?,error=?,updated_at=? WHERE workspace_id=? AND state_id=? AND source_revision=? AND attempt_id=? AND state='in_progress'`, state, detail, r.timestamp(), workspaceID, stateID, sourceRevision, attemptID)
	if err != nil {
		return fmt.Errorf("finish workspace state capture attempt: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetCaptureEvidence(ctx context.Context, workspaceID, stateID string) (CaptureEvidence, error) {
	var evidence CaptureEvidence
	err := r.db.QueryRowContext(ctx, `SELECT workspace_id,state_id,request_token,source_revision,capture_id,evidence_json,captured_at FROM workspace_state_capture_evidence WHERE workspace_id=? AND state_id=?`, workspaceID, stateID).Scan(&evidence.WorkspaceID, &evidence.StateID, &evidence.RequestToken, &evidence.SourceRevision, &evidence.CaptureID, &evidence.EvidenceJSON, &evidence.CapturedAt)
	if err != nil {
		return CaptureEvidence{}, err
	}
	return evidence, nil
}

func (r *SQLiteRepository) ListCaptureReceipts(ctx context.Context, workspaceID, stateID string) ([]CaptureReceipt, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,state_id,source_revision,capture_id,attempt_id,evidence_json,captured_at
		FROM workspace_capture_receipts WHERE workspace_id=? AND state_id=? ORDER BY captured_at DESC,id DESC`, workspaceID, stateID)
	if err != nil {
		return nil, fmt.Errorf("list workspace capture receipts: %w", err)
	}
	defer rows.Close()
	receipts := make([]CaptureReceipt, 0)
	for rows.Next() {
		var receipt CaptureReceipt
		if err := rows.Scan(&receipt.ID, &receipt.WorkspaceID, &receipt.StateID, &receipt.SourceRevision, &receipt.CaptureID, &receipt.AttemptID, &receipt.EvidenceJSON, &receipt.CapturedAt); err != nil {
			return nil, fmt.Errorf("scan workspace capture receipt: %w", err)
		}
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace capture receipts: %w", err)
	}
	return receipts, nil
}

func (r *SQLiteRepository) GetCaptureReceipt(ctx context.Context, workspaceID, stateID, receiptID string) (CaptureReceipt, error) {
	var receipt CaptureReceipt
	err := r.db.QueryRowContext(ctx, `SELECT id,workspace_id,state_id,source_revision,capture_id,attempt_id,evidence_json,captured_at
		FROM workspace_capture_receipts WHERE workspace_id=? AND state_id=? AND id=?`, workspaceID, stateID, receiptID).
		Scan(&receipt.ID, &receipt.WorkspaceID, &receipt.StateID, &receipt.SourceRevision, &receipt.CaptureID, &receipt.AttemptID, &receipt.EvidenceJSON, &receipt.CapturedAt)
	if err != nil {
		return CaptureReceipt{}, err
	}
	return receipt, nil
}

func (r *SQLiteRepository) CreateVisualComparison(ctx context.Context, workspaceID, stateID, baselineReceiptID, currentReceiptID, resultJSON, heatmapRef, createdAt string) (VisualComparison, error) {
	if workspaceID == "" || stateID == "" || baselineReceiptID == "" || currentReceiptID == "" || baselineReceiptID == currentReceiptID || !json.Valid([]byte(resultJSON)) {
		return VisualComparison{}, &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: "visual_comparison", Message: "workspace, state, distinct receipt identities, and JSON result are required"}}}
	}
	id, err := newID("vc_")
	if err != nil {
		return VisualComparison{}, fmt.Errorf("assign visual comparison identity: %w", err)
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO workspace_visual_comparisons
		(id,workspace_id,state_id,baseline_receipt_id,current_receipt_id,result_json,heatmap_ref,created_at)
		SELECT ?,?,?,?,?,?,?,? WHERE
		(SELECT COUNT(*) FROM workspace_capture_receipts WHERE workspace_id=? AND state_id=? AND id IN (?,?))=2`,
		id, workspaceID, stateID, baselineReceiptID, currentReceiptID, resultJSON, heatmapRef, createdAt,
		workspaceID, stateID, baselineReceiptID, currentReceiptID)
	if err != nil {
		return VisualComparison{}, fmt.Errorf("persist immutable workspace visual comparison: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return VisualComparison{}, fmt.Errorf("check workspace visual comparison insert: %w", err)
	}
	if count != 1 {
		return VisualComparison{}, sql.ErrNoRows
	}
	return VisualComparison{ID: id, WorkspaceID: workspaceID, StateID: stateID, BaselineReceiptID: baselineReceiptID, CurrentReceiptID: currentReceiptID, ResultJSON: resultJSON, HeatmapRef: heatmapRef, CreatedAt: createdAt}, nil
}

func (r *SQLiteRepository) ListVisualComparisons(ctx context.Context, workspaceID, stateID string) ([]VisualComparison, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,state_id,baseline_receipt_id,current_receipt_id,result_json,heatmap_ref,created_at
		FROM workspace_visual_comparisons WHERE workspace_id=? AND state_id=? ORDER BY created_at DESC,id DESC`, workspaceID, stateID)
	if err != nil {
		return nil, fmt.Errorf("list workspace visual comparisons: %w", err)
	}
	defer rows.Close()
	comparisons := make([]VisualComparison, 0)
	for rows.Next() {
		var item VisualComparison
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.BaselineReceiptID, &item.CurrentReceiptID, &item.ResultJSON, &item.HeatmapRef, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan workspace visual comparison: %w", err)
		}
		comparisons = append(comparisons, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace visual comparisons: %w", err)
	}
	return comparisons, nil
}

func (r *SQLiteRepository) GetVisualComparison(ctx context.Context, workspaceID, stateID, comparisonID string) (VisualComparison, error) {
	var item VisualComparison
	err := r.db.QueryRowContext(ctx, `SELECT id,workspace_id,state_id,baseline_receipt_id,current_receipt_id,result_json,heatmap_ref,created_at
		FROM workspace_visual_comparisons WHERE workspace_id=? AND state_id=? AND id=?`, workspaceID, stateID, comparisonID).
		Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.BaselineReceiptID, &item.CurrentReceiptID, &item.ResultJSON, &item.HeatmapRef, &item.CreatedAt)
	if err != nil {
		return VisualComparison{}, err
	}
	return item, nil
}
