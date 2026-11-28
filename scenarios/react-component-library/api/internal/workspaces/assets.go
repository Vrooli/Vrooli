package workspaces

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"

	"react-component-library/internal/imageheader"
)

// MaxAssetBytes bounds one decoded upload. The parser and the MIME allowlist
// live in internal/imageheader so this store and any importer share one answer.
const MaxAssetBytes = imageheader.MaxBytes

// Asset fidelities describe how faithful an upload is to the rendered page.
const (
	AssetFidelityReference = "reference"
	AssetFidelityCaptured  = "captured"
)

// Asset is one uploaded reference image's metadata. Bytes are served by the
// byte route; this shape is the JSON list/create payload.
type Asset struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Mime        string `json:"mime"`
	Bytes       int64  `json:"bytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Fidelity    string `json:"fidelity"`
	AddedAt     string `json:"addedAt"`
	URL         string `json:"url"`
}

// AssetInput is a decoded upload. The handler decodes base64; this package
// validates the name, MIME type, size and fidelity, and reads the dimensions.
type AssetInput struct {
	WorkspaceID string
	Name        string
	Mime        string
	Data        []byte
	Fidelity    string
}

// AssetURL is the browser path for one asset's bytes. The handler prefixes the
// API origin; the client builds an absolute URL from this value.
func AssetURL(workspaceID, assetID string) string {
	return "/api/workspaces/" + url.PathEscape(workspaceID) + "/assets/" + url.PathEscape(assetID)
}

func normalizeAssetInput(input AssetInput) (AssetInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return AssetInput{}, invalidAsset("name", "asset name is required")
	}
	input.Mime = imageheader.NormalizeMime(input.Mime)
	if _, ok := imageheader.AllowedMimes[input.Mime]; !ok {
		return AssetInput{}, invalidAsset("mime", "asset mime must be image/png, image/jpeg, image/webp, image/gif or image/svg+xml")
	}
	if len(input.Data) == 0 {
		return AssetInput{}, invalidAsset("data", "asset data is required")
	}
	if len(input.Data) > MaxAssetBytes {
		return AssetInput{}, invalidAsset("data", "asset exceeds the 8 MiB limit")
	}
	input.Fidelity = strings.ToLower(strings.TrimSpace(input.Fidelity))
	if input.Fidelity == "" {
		input.Fidelity = AssetFidelityReference
	}
	if input.Fidelity != AssetFidelityReference && input.Fidelity != AssetFidelityCaptured {
		return AssetInput{}, invalidAsset("fidelity", "asset fidelity must be reference or captured")
	}
	return input, nil
}

// invalidAsset produces the same *InvalidError the aggregate validator does,
// so the handler maps every invalid upload and every invalid composition to one
// 400 envelope.
func invalidAsset(path, message string) error {
	return &InvalidError{Issues: []*workspacev1.ValidationIssue{{Path: path, Message: message}}}
}

// ListWorkspaceAssets returns a workspace's uploads newest first.
func (r *SQLiteRepository) ListWorkspaceAssets(ctx context.Context, workspaceID string) ([]Asset, error) {
	if err := r.requireWorkspace(ctx, r.db, workspaceID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+assetColumns+`
		FROM workspace_assets WHERE workspace_id = ?
		ORDER BY created_at DESC, rowid DESC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace assets: %w", err)
	}
	defer rows.Close()
	assets := make([]Asset, 0)
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace assets: %w", err)
	}
	return assets, nil
}

// ListAllWorkspaceAssets returns upload metadata across every workspace so a
// person can reuse a reference they already uploaded elsewhere. Bytes are not
// part of this read; the byte route stays scoped to the owning workspace.
func (r *SQLiteRepository) ListAllWorkspaceAssets(ctx context.Context) ([]Asset, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+assetColumns+`
		FROM workspace_assets ORDER BY created_at DESC, rowid DESC LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("list all workspace assets: %w", err)
	}
	defer rows.Close()
	assets := make([]Asset, 0)
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all workspace assets: %w", err)
	}
	return assets, nil
}

// GetWorkspaceAsset returns one upload's metadata and bytes, scoped to its
// workspace so an id from another workspace is a miss, never a cross-read.
func (r *SQLiteRepository) GetWorkspaceAsset(ctx context.Context, workspaceID, assetID string) (Asset, []byte, error) {
	var (
		asset Asset
		data  []byte
	)
	err := r.db.QueryRowContext(ctx, `SELECT `+assetColumns+`, data
		FROM workspace_assets WHERE id = ? AND workspace_id = ?`, assetID, workspaceID).
		Scan(&asset.ID, &asset.WorkspaceID, &asset.Name, &asset.Mime, &asset.Bytes,
			&asset.Width, &asset.Height, &asset.Fidelity, &asset.AddedAt, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, nil, ErrNotFound
	}
	if err != nil {
		return Asset{}, nil, fmt.Errorf("get workspace asset %q: %w", assetID, err)
	}
	asset.URL = AssetURL(asset.WorkspaceID, asset.ID)
	return asset, data, nil
}

// CreateWorkspaceAsset validates and stores one upload. The existence check and
// the insert share a transaction so a workspace deleted concurrently cannot
// leave an orphan row behind a foreign-key error.
func (r *SQLiteRepository) CreateWorkspaceAsset(ctx context.Context, input AssetInput) (Asset, error) {
	clean, err := normalizeAssetInput(input)
	if err != nil {
		return Asset{}, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Asset{}, fmt.Errorf("begin workspace asset upload: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := r.requireWritableWorkspace(ctx, tx, clean.WorkspaceID); err != nil {
		return Asset{}, err
	}

	width, height := imageheader.Dimensions(clean.Mime, clean.Data)
	asset := Asset{
		ID:          uuid.NewString(),
		WorkspaceID: clean.WorkspaceID,
		Name:        clean.Name,
		Mime:        clean.Mime,
		Bytes:       int64(len(clean.Data)),
		Width:       width,
		Height:      height,
		Fidelity:    clean.Fidelity,
		AddedAt:     r.timestamp(),
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO workspace_assets
		(id, workspace_id, name, mime, bytes, width, height, fidelity, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		asset.ID, asset.WorkspaceID, asset.Name, asset.Mime, asset.Bytes,
		asset.Width, asset.Height, asset.Fidelity, clean.Data, asset.AddedAt); err != nil {
		return Asset{}, fmt.Errorf("create workspace asset: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Asset{}, fmt.Errorf("create workspace asset commit: %w", err)
	}
	asset.URL = AssetURL(asset.WorkspaceID, asset.ID)
	return asset, nil
}

// DeleteWorkspaceAsset removes one upload. It is scoped to the workspace and
// idempotent for an existing workspace: a re-sent delete is not an error, but a
// missing workspace is a 404 rather than a silent no-op.
func (r *SQLiteRepository) DeleteWorkspaceAsset(ctx context.Context, workspaceID, assetID string) error {
	if err := r.requireWritableWorkspace(ctx, r.db, workspaceID); err != nil {
		return err
	}
	var referenced int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM workspace_state_source_baselines WHERE workspace_id=? AND asset_id=? LIMIT 1`, workspaceID, assetID).Scan(&referenced); err == nil {
		return ErrSourceBaselineAsset
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check source baseline asset reference: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM workspace_assets WHERE id = ? AND workspace_id = ?`,
		assetID, workspaceID); err != nil {
		return fmt.Errorf("delete workspace asset: %w", err)
	}
	return nil
}

// requireWritableWorkspace also refuses a curated template: its assets are part
// of the reviewed snapshot, so an upload or delete would let the shelf drift.
func (r *SQLiteRepository) requireWritableWorkspace(ctx context.Context, q queryer, id string) error {
	var provenance string
	if strings.TrimSpace(id) == "" {
		return ErrNotFound
	}
	err := q.QueryRowContext(ctx, `SELECT provenance FROM workspaces WHERE id = ?`, id).Scan(&provenance)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect workspace %q: %w", id, err)
	}
	if provenance == provenanceCurated {
		return ErrCurated
	}
	return nil
}

func (r *SQLiteRepository) requireWorkspace(ctx context.Context, q queryer, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrNotFound
	}
	var exists int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM workspaces WHERE id = ?`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect workspace %q: %w", id, err)
	}
	return nil
}

const assetColumns = `id, workspace_id, name, mime, bytes, width, height, fidelity, created_at`

type assetScanner interface {
	Scan(dest ...any) error
}

func scanAsset(row assetScanner) (Asset, error) {
	var asset Asset
	if err := row.Scan(&asset.ID, &asset.WorkspaceID, &asset.Name, &asset.Mime, &asset.Bytes,
		&asset.Width, &asset.Height, &asset.Fidelity, &asset.AddedAt); err != nil {
		return Asset{}, fmt.Errorf("scan workspace asset: %w", err)
	}
	asset.URL = AssetURL(asset.WorkspaceID, asset.ID)
	return asset, nil
}
