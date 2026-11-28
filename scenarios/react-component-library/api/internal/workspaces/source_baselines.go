package workspaces

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (r *SQLiteRepository) CreateSourceBaseline(ctx context.Context, workspaceID, stateID, assetID, digest, locator, createdAt string) (SourceBaseline, error) {
	if err := r.requireWritableWorkspace(ctx, r.db, workspaceID); err != nil {
		return SourceBaseline{}, err
	}
	asset, data, err := r.GetWorkspaceAsset(ctx, workspaceID, assetID)
	if err != nil {
		return SourceBaseline{}, err
	}
	if asset.Fidelity != AssetFidelityReference || (asset.Mime != "image/png" && asset.Mime != "image/jpeg") {
		return SourceBaseline{}, invalidAsset("assetId", "source baseline must use a PNG or JPEG reference image")
	}
	actual := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(actual[:]), digest) {
		return SourceBaseline{}, invalidAsset("sha256", "source image digest does not match the stored asset")
	}
	if strings.TrimSpace(locator) == "" {
		locator = "uploaded reference image"
	}
	id := uuid.NewString()
	result, err := r.db.ExecContext(ctx, `INSERT INTO workspace_state_source_baselines (id,workspace_id,state_id,asset_id,sha256,source_locator,created_at)
		SELECT ?,?,?,?,?,?,? WHERE EXISTS (SELECT 1 FROM workspace_states s JOIN workspace_displays d ON d.id=s.display_id JOIN workspace_pages p ON p.id=d.page_id WHERE s.id=? AND p.workspace_id=?)`,
		id, workspaceID, stateID, assetID, strings.ToLower(digest), locator, createdAt, stateID, workspaceID)
	if err != nil {
		return SourceBaseline{}, fmt.Errorf("create workspace source baseline: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return SourceBaseline{}, err
	}
	if count != 1 {
		return SourceBaseline{}, sql.ErrNoRows
	}
	return r.getSourceBaseline(ctx, r.db, workspaceID, stateID, id)
}

func (r *SQLiteRepository) GetCurrentSourceBaseline(ctx context.Context, workspaceID, stateID string) (SourceBaseline, error) {
	var item SourceBaseline
	err := r.db.QueryRowContext(ctx, `SELECT b.id,b.workspace_id,b.state_id,b.asset_id,b.sha256,b.source_locator,b.created_at,
		COALESCE(a.accepted_at,''),COALESCE(a.accepted_by,'')
		FROM workspace_state_source_baselines b LEFT JOIN workspace_state_source_baseline_acceptances a ON a.baseline_id=b.id
		WHERE b.workspace_id=? AND b.state_id=? ORDER BY b.created_at DESC,b.rowid DESC LIMIT 1`, workspaceID, stateID).
		Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.AssetID, &item.SHA256, &item.SourceLocator, &item.CreatedAt, &item.AcceptedAt, &item.AcceptedBy)
	if err != nil {
		return SourceBaseline{}, err
	}
	return item, nil
}

func (r *SQLiteRepository) ListSourceBaselines(ctx context.Context, workspaceID, stateID string) ([]SourceBaseline, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT b.id,b.workspace_id,b.state_id,b.asset_id,b.sha256,b.source_locator,b.created_at,COALESCE(a.accepted_at,''),COALESCE(a.accepted_by,'') FROM workspace_state_source_baselines b LEFT JOIN workspace_state_source_baseline_acceptances a ON a.baseline_id=b.id WHERE b.workspace_id=? AND b.state_id=? ORDER BY b.created_at DESC,b.rowid DESC`, workspaceID, stateID)
	if err != nil {
		return nil, fmt.Errorf("list source baselines: %w", err)
	}
	defer rows.Close()
	items := make([]SourceBaseline, 0)
	for rows.Next() {
		var item SourceBaseline
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.AssetID, &item.SHA256, &item.SourceLocator, &item.CreatedAt, &item.AcceptedAt, &item.AcceptedBy); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) GetAcceptedSourceBaseline(ctx context.Context, workspaceID, stateID string) (SourceBaseline, error) {
	var item SourceBaseline
	err := r.db.QueryRowContext(ctx, `SELECT b.id,b.workspace_id,b.state_id,b.asset_id,b.sha256,b.source_locator,b.created_at,a.accepted_at,a.accepted_by FROM workspace_state_source_baselines b JOIN workspace_state_source_baseline_acceptances a ON a.baseline_id=b.id WHERE b.workspace_id=? AND b.state_id=? ORDER BY a.rowid DESC LIMIT 1`, workspaceID, stateID).Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.AssetID, &item.SHA256, &item.SourceLocator, &item.CreatedAt, &item.AcceptedAt, &item.AcceptedBy)
	return item, err
}

func (r *SQLiteRepository) AcceptSourceBaseline(ctx context.Context, workspaceID, stateID, baselineID, acceptedBy, acceptedAt string) (SourceBaseline, error) {
	acceptedBy = strings.TrimSpace(acceptedBy)
	if acceptedBy == "" {
		return SourceBaseline{}, invalidAsset("acceptedBy", "a human acceptance label is required")
	}
	if err := r.requireWritableWorkspace(ctx, r.db, workspaceID); err != nil {
		return SourceBaseline{}, err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO workspace_state_source_baseline_acceptances (baseline_id,workspace_id,state_id,accepted_by,accepted_at)
		SELECT b.id,b.workspace_id,b.state_id,?,? FROM workspace_state_source_baselines b
		WHERE b.id=? AND b.workspace_id=? AND b.state_id=?
		ON CONFLICT(baseline_id) DO NOTHING`, acceptedBy, acceptedAt, baselineID, workspaceID, stateID)
	if err != nil {
		return SourceBaseline{}, fmt.Errorf("accept workspace source baseline: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return SourceBaseline{}, err
	}
	if count == 0 {
		existing, e := r.getSourceBaseline(ctx, r.db, workspaceID, stateID, baselineID)
		if e != nil || existing.AcceptedAt == "" {
			return SourceBaseline{}, sql.ErrNoRows
		}
		return existing, nil
	}
	return r.getSourceBaseline(ctx, r.db, workspaceID, stateID, baselineID)
}

func (r *SQLiteRepository) getSourceBaseline(ctx context.Context, q queryer, workspaceID, stateID, baselineID string) (SourceBaseline, error) {
	var item SourceBaseline
	err := q.QueryRowContext(ctx, `SELECT b.id,b.workspace_id,b.state_id,b.asset_id,b.sha256,b.source_locator,b.created_at,COALESCE(a.accepted_at,''),COALESCE(a.accepted_by,'') FROM workspace_state_source_baselines b LEFT JOIN workspace_state_source_baseline_acceptances a ON a.baseline_id=b.id WHERE b.workspace_id=? AND b.state_id=? AND b.id=?`, workspaceID, stateID, baselineID).Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.AssetID, &item.SHA256, &item.SourceLocator, &item.CreatedAt, &item.AcceptedAt, &item.AcceptedBy)
	return item, err
}

func (r *SQLiteRepository) CreateSourceVisualComparison(ctx context.Context, item SourceVisualComparison) (SourceVisualComparison, error) {
	if !json.Valid([]byte(item.ResultJSON)) {
		return SourceVisualComparison{}, invalidAsset("result", "visual comparison result must be valid JSON")
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO workspace_source_visual_comparisons (id,workspace_id,state_id,source_baseline_id,current_receipt_id,result_json,heatmap_ref,created_at)
		SELECT ?,?,?,?,?,?,?,? WHERE EXISTS (SELECT 1 FROM workspace_state_source_baselines b JOIN workspace_state_source_baseline_acceptances a ON a.baseline_id=b.id WHERE b.id=? AND b.workspace_id=? AND b.state_id=?)
		AND EXISTS (SELECT 1 FROM workspace_capture_receipts c WHERE c.id=? AND c.workspace_id=? AND c.state_id=?)`,
		item.ID, item.WorkspaceID, item.StateID, item.SourceBaselineID, item.CurrentReceiptID, item.ResultJSON, item.HeatmapRef, item.CreatedAt,
		item.SourceBaselineID, item.WorkspaceID, item.StateID, item.CurrentReceiptID, item.WorkspaceID, item.StateID)
	if err != nil {
		return SourceVisualComparison{}, fmt.Errorf("persist source visual comparison: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return SourceVisualComparison{}, err
	}
	if count != 1 {
		return SourceVisualComparison{}, sql.ErrNoRows
	}
	return item, nil
}

func (r *SQLiteRepository) ListSourceVisualComparisons(ctx context.Context, workspaceID, stateID string) ([]SourceVisualComparison, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT c.id,c.workspace_id,c.state_id,c.source_baseline_id,b.asset_id,c.current_receipt_id,c.result_json,c.heatmap_ref,c.created_at FROM workspace_source_visual_comparisons c JOIN workspace_state_source_baselines b ON b.id=c.source_baseline_id WHERE c.workspace_id=? AND c.state_id=? ORDER BY c.created_at DESC,c.rowid DESC`, workspaceID, stateID)
	if err != nil {
		return nil, fmt.Errorf("list source visual comparisons: %w", err)
	}
	defer rows.Close()
	items := make([]SourceVisualComparison, 0)
	for rows.Next() {
		var item SourceVisualComparison
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.SourceBaselineID, &item.SourceAssetID, &item.CurrentReceiptID, &item.ResultJSON, &item.HeatmapRef, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *SQLiteRepository) GetSourceVisualComparison(ctx context.Context, workspaceID, stateID, comparisonID string) (SourceVisualComparison, error) {
	var item SourceVisualComparison
	err := r.db.QueryRowContext(ctx, `SELECT c.id,c.workspace_id,c.state_id,c.source_baseline_id,b.asset_id,c.current_receipt_id,c.result_json,c.heatmap_ref,c.created_at FROM workspace_source_visual_comparisons c JOIN workspace_state_source_baselines b ON b.id=c.source_baseline_id WHERE c.workspace_id=? AND c.state_id=? AND c.id=?`, workspaceID, stateID, comparisonID).Scan(&item.ID, &item.WorkspaceID, &item.StateID, &item.SourceBaselineID, &item.SourceAssetID, &item.CurrentReceiptID, &item.ResultJSON, &item.HeatmapRef, &item.CreatedAt)
	return item, err
}
