package designs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

const assetColumns = `id, design_id, name, mime, bytes, width, height, fidelity, created_at`

func (r *sqliteRepository) ListAssets(ctx context.Context, designID string) ([]Asset, error) {
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM studio_designs WHERE id = ?`, designID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("inspect design %q: %w", designID, err)
	}
	if exists == 0 {
		return nil, ErrNotFound
	}
	rows, err := r.db.QueryContext(ctx, `SELECT `+assetColumns+` FROM studio_design_assets WHERE design_id = ? ORDER BY created_at ASC, rowid ASC`, designID)
	if err != nil {
		return nil, fmt.Errorf("list design assets: %w", err)
	}
	defer rows.Close()
	return scanAssets(rows)
}

func (r *sqliteRepository) ListAllAssets(ctx context.Context) ([]Asset, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+assetColumns+` FROM studio_design_assets ORDER BY created_at DESC, rowid DESC`)
	if err != nil {
		return nil, fmt.Errorf("list all design assets: %w", err)
	}
	defer rows.Close()
	return scanAssets(rows)
}

func (r *sqliteRepository) GetAsset(ctx context.Context, designID, assetID string) (Asset, []byte, error) {
	var (
		asset Asset
		data  []byte
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT `+assetColumns+`, data FROM studio_design_assets WHERE id = ? AND design_id = ?`,
		assetID, designID,
	).Scan(&asset.ID, &asset.DesignID, &asset.Name, &asset.Mime, &asset.Bytes, &asset.Width, &asset.Height,
		&asset.Fidelity, &asset.AddedAt, &data)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, nil, ErrNotFound
	}
	if err != nil {
		return Asset{}, nil, fmt.Errorf("get design asset %q: %w", assetID, err)
	}
	asset.URL = AssetURL(asset.DesignID, asset.ID)
	return asset, data, nil
}

func (r *sqliteRepository) CreateAsset(ctx context.Context, input AssetInput) (Asset, error) {
	clean, err := normalizeAssetInput(input)
	if err != nil {
		return Asset{}, err
	}
	width, height := imageDimensions(clean.Mime, clean.Data)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Asset{}, fmt.Errorf("create design asset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM studio_designs WHERE id = ?`, clean.DesignID).Scan(&exists); err != nil {
		return Asset{}, fmt.Errorf("inspect design %q: %w", clean.DesignID, err)
	}
	if exists == 0 {
		return Asset{}, ErrNotFound
	}

	addedAt := clean.AddedAt
	if addedAt == "" {
		addedAt = r.timestamp()
	}
	asset := Asset{
		ID:       "asset-" + uuid.NewString(),
		DesignID: clean.DesignID,
		Name:     clean.Name,
		Mime:     clean.Mime,
		Bytes:    int64(len(clean.Data)),
		Width:    width,
		Height:   height,
		Fidelity: clean.Fidelity,
		AddedAt:  addedAt,
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO studio_design_assets (id, design_id, name, mime, bytes, width, height, fidelity, data, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		asset.ID, asset.DesignID, asset.Name, asset.Mime, asset.Bytes, asset.Width, asset.Height,
		asset.Fidelity, clean.Data, asset.AddedAt); err != nil {
		return Asset{}, fmt.Errorf("create design asset: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Asset{}, fmt.Errorf("create design asset commit: %w", err)
	}
	asset.URL = AssetURL(asset.DesignID, asset.ID)
	return asset, nil
}

func (r *sqliteRepository) DeleteAsset(ctx context.Context, designID, assetID string) error {
	// Scoped by design_id so an id from another design is a no-op, never a
	// cross-design delete. Idempotent for the owning design.
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM studio_design_assets WHERE id = ? AND design_id = ?`, assetID, designID); err != nil {
		return fmt.Errorf("delete design asset: %w", err)
	}
	return nil
}

// AssetURL is the browser path for one asset's bytes. The handler prefixes
// the API origin; the client builds the absolute URL from this value.
func AssetURL(designID, assetID string) string {
	return fmt.Sprintf("/api/designs/%s/assets/%s", designID, assetID)
}

func scanAssets(rows *sql.Rows) ([]Asset, error) {
	assets := make([]Asset, 0)
	for rows.Next() {
		var asset Asset
		if err := rows.Scan(&asset.ID, &asset.DesignID, &asset.Name, &asset.Mime, &asset.Bytes,
			&asset.Width, &asset.Height, &asset.Fidelity, &asset.AddedAt); err != nil {
			return nil, fmt.Errorf("scan design asset: %w", err)
		}
		asset.URL = AssetURL(asset.DesignID, asset.ID)
		assets = append(assets, asset)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate design assets: %w", err)
	}
	return assets, nil
}
