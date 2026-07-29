package delivery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// CatalogStore is the transaction-capable persistence boundary for the
// delivery catalog and its entitlement-gated artifact configuration.
//
// seam: CatalogStore
type CatalogStore interface {
	Query(string, ...any) (*sql.Rows, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
	Exec(string, ...any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

var (
	ErrAssetNotFound = errors.New("download not found")
	ErrAppNotFound   = errors.New("download app not found")
)

type ChannelInfo struct {
	Channel   string `json:"channel"`
	Platform  string `json:"platform"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

type UpdatePolicy struct {
	CheckIntervalHours int    `json:"check_interval_hours"`
	UpdateMode         string `json:"update_mode"`
	AllowDowngrade     bool   `json:"allow_downgrade"`
}

// CatalogService owns the durable application and asset catalog used by
// storefront rendering, admin operations, and update manifests.
type CatalogService struct {
	db CatalogStore
}

func NewCatalogService(db CatalogStore) *CatalogService {
	return &CatalogService{db: db}
}

// ValidateDirectArtifactURL accepts only relative paths or absolute HTTP(S)
// addresses for direct delivery artifacts.
func ValidateDirectArtifactURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("artifact_url is required for direct downloads")
	}
	if strings.HasPrefix(raw, "/") {
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid artifact_url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("artifact_url must be http(s) or a relative path")
	}
	if parsed.Host == "" {
		return fmt.Errorf("artifact_url must include a host")
	}
	return nil
}

// ListAssets returns all download assets for a bundle with artifact info.
func (s *CatalogService) ListAssets(bundleKey string) ([]Asset, error) {
	return s.ListAssetsContext(context.Background(), bundleKey)
}

// ListAssetsContext reads assets from the pool selected for ctx. Requests
// marked by api-core test mode therefore stay inside their leased database.
func (s *CatalogService) ListAssetsContext(ctx context.Context, bundleKey string) ([]Asset, error) {
	// Query assets with artifact info via LEFT JOIN
	query := `
		SELECT da.id, da.bundle_key, da.app_key, da.platform, da.artifact_url, da.artifact_source, da.artifact_id, da.release_version,
		       da.release_notes, da.checksum, da.requires_entitlement, da.metadata,
		       art.original_filename, art.size_bytes,
		       (SELECT COUNT(*) FROM download_artifacts WHERE bundle_key = da.bundle_key AND app_key = da.app_key AND platform = da.platform) AS artifact_count
		FROM download_assets da
		LEFT JOIN download_artifacts art ON da.artifact_id = art.id
		WHERE da.bundle_key = $1
		ORDER BY da.app_key, da.platform, COALESCE(da.display_order, 0), da.id
	`

	rows, err := s.db.QueryContext(ctx, query, bundleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]Asset, 0)
	for rows.Next() {
		var t AssetScanTargets
		var artifactFilename sql.NullString
		var artifactSizeBytes sql.NullInt64
		var artifactCount sql.NullInt64
		dest := append(t.ScanDest(), &artifactFilename, &artifactSizeBytes, &artifactCount)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		asset := t.Hydrate()
		if artifactFilename.Valid {
			asset.ArtifactFilename = artifactFilename.String
		}
		if artifactSizeBytes.Valid {
			asset.ArtifactSizeBytes = artifactSizeBytes.Int64
		}
		if artifactCount.Valid {
			asset.ArtifactCount = int(artifactCount.Int64)
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

// ListApps returns download apps with their associated platform assets.
func (s *CatalogService) ListApps(bundleKey string) ([]App, error) {
	return s.ListAppsContext(context.Background(), bundleKey)
}

// ListAppsContext reads app metadata and its assets from the pool selected for
// ctx. It is the request-aware counterpart used by public landing rendering.
func (s *CatalogService) ListAppsContext(ctx context.Context, bundleKey string) ([]App, error) {
	query := `
		SELECT id, bundle_key, app_key, name, tagline, description,
		       icon_url, screenshot_url, install_overview, install_steps, storefronts, metadata, display_order, update_api_key, update_policy
		FROM download_apps
		WHERE bundle_key = $1
		ORDER BY display_order, name
	`

	rows, err := s.db.QueryContext(ctx, query, bundleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	apps := make([]App, 0)
	for rows.Next() {
		var t AppScanTargets
		if err := rows.Scan(t.ScanDest()...); err != nil {
			return nil, err
		}
		apps = append(apps, t.Hydrate())
	}

	if len(apps) == 0 {
		return apps, nil
	}

	assets, err := s.ListAssetsContext(ctx, bundleKey)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]Asset)
	for _, asset := range assets {
		grouped[asset.AppKey] = append(grouped[asset.AppKey], asset)
	}

	for i := range apps {
		app := &apps[i]
		app.Platforms = grouped[app.AppKey]
		if app.Platforms == nil {
			app.Platforms = make([]Asset, 0)
		}
	}

	return apps, nil
}

// GetApp fetches a single download app with its assets.
func (s *CatalogService) GetApp(bundleKey, appKey string) (*App, error) {
	query := `
		SELECT id, bundle_key, app_key, name, tagline, description,
		       icon_url, screenshot_url, install_overview, install_steps, storefronts, metadata, display_order, update_api_key, update_policy
		FROM download_apps
		WHERE bundle_key = $1 AND app_key = $2
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey)
	var t AppScanTargets
	if err := row.Scan(t.ScanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	app := t.Hydrate()

	assetRows, err := s.db.Query(`
		SELECT id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2
		ORDER BY platform, COALESCE(display_order, 0), id
	`, bundleKey, appKey)
	if err != nil {
		return nil, err
	}
	defer assetRows.Close()

	for assetRows.Next() {
		var t AssetScanTargets
		if err := assetRows.Scan(t.ScanDest()...); err != nil {
			return nil, err
		}
		app.Platforms = append(app.Platforms, t.Hydrate())
	}

	return &app, nil
}

// UpsertApp creates or updates an app definition along with its platform installers.
func (s *CatalogService) UpsertApp(app App) (*App, error) {
	app.BundleKey = strings.TrimSpace(app.BundleKey)
	app.AppKey = strings.TrimSpace(app.AppKey)

	if app.BundleKey == "" || app.AppKey == "" {
		return nil, fmt.Errorf("bundle_key and app_key are required")
	}
	if strings.TrimSpace(app.Name) == "" {
		return nil, fmt.Errorf("app name is required")
	}

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	installStepsBytes, err := json.Marshal(app.InstallSteps)
	if err != nil {
		return nil, fmt.Errorf("marshal install steps: %w", err)
	}
	storefrontBytes, err := json.Marshal(app.Storefronts)
	if err != nil {
		return nil, fmt.Errorf("marshal storefronts: %w", err)
	}
	metadataBytes, err := json.Marshal(app.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal metadata: %w", err)
	}

	// Convert empty strings to nil for nullable columns
	var iconURL, screenshotURL, updateAPIKey interface{}
	if strings.TrimSpace(app.IconURL) != "" {
		iconURL = app.IconURL
	}
	if strings.TrimSpace(app.ScreenshotURL) != "" {
		screenshotURL = app.ScreenshotURL
	}
	if strings.TrimSpace(app.UpdateAPIKey) != "" {
		updateAPIKey = app.UpdateAPIKey
	}

	_, err = tx.Exec(`
		INSERT INTO download_apps (
			bundle_key, app_key, name, tagline, description,
			icon_url, screenshot_url, install_overview, install_steps, storefronts, metadata, display_order, update_api_key
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (bundle_key, app_key) DO UPDATE SET
			name = EXCLUDED.name,
			tagline = EXCLUDED.tagline,
			description = EXCLUDED.description,
			icon_url = EXCLUDED.icon_url,
			screenshot_url = EXCLUDED.screenshot_url,
			install_overview = EXCLUDED.install_overview,
			install_steps = EXCLUDED.install_steps,
			storefronts = EXCLUDED.storefronts,
			metadata = EXCLUDED.metadata,
			display_order = EXCLUDED.display_order,
			update_api_key = EXCLUDED.update_api_key,
			updated_at = NOW()
	`, app.BundleKey, app.AppKey, app.Name, app.Tagline, app.Description, iconURL, screenshotURL, app.InstallOverview, installStepsBytes, storefrontBytes, metadataBytes, app.DisplayOrder, updateAPIKey)
	if err != nil {
		return nil, fmt.Errorf("upsert download app: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM download_assets WHERE bundle_key = $1 AND app_key = $2`, app.BundleKey, app.AppKey); err != nil {
		return nil, fmt.Errorf("clear existing assets: %w", err)
	}

	if len(app.Platforms) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO download_assets (
				bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id,
				release_version, release_notes, checksum, requires_entitlement, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		`)
		if err != nil {
			return nil, fmt.Errorf("prepare asset insert: %w", err)
		}
		defer stmt.Close()

		for _, asset := range app.Platforms {
			platform := strings.TrimSpace(asset.Platform)
			if platform == "" {
				return nil, fmt.Errorf("platform is required for all assets")
			}
			assetSource := strings.TrimSpace(asset.ArtifactSource)
			if assetSource == "" {
				assetSource = "direct"
			}
			if assetSource != "direct" && assetSource != "managed" {
				return nil, fmt.Errorf("artifact_source must be 'direct' or 'managed'")
			}
			if assetSource == "direct" {
				if err := ValidateDirectArtifactURL(asset.ArtifactURL); err != nil {
					return nil, err
				}
			} else if asset.ArtifactID == nil || *asset.ArtifactID == 0 {
				return nil, fmt.Errorf("artifact_id is required when artifact_source is managed")
			}
			metadataBytes, err := json.Marshal(asset.Metadata)
			if err != nil {
				return nil, fmt.Errorf("marshal asset metadata: %w", err)
			}
			var artifactID interface{}
			if asset.ArtifactID != nil && *asset.ArtifactID != 0 {
				artifactID = *asset.ArtifactID
			}
			if _, err := stmt.Exec(
				app.BundleKey,
				app.AppKey,
				platform,
				strings.TrimSpace(asset.ArtifactURL),
				assetSource,
				artifactID,
				strings.TrimSpace(asset.ReleaseVersion),
				strings.TrimSpace(asset.ReleaseNotes),
				strings.TrimSpace(asset.Checksum),
				asset.RequiresEntitlement,
				metadataBytes,
			); err != nil {
				return nil, fmt.Errorf("insert asset for platform %s: %w", platform, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit download app upsert: %w", err)
	}

	return s.GetApp(app.BundleKey, app.AppKey)
}

// DeleteApp removes a download app and its associated assets.
func (s *CatalogService) DeleteApp(bundleKey, appKey string) error {
	bundleKey = strings.TrimSpace(bundleKey)
	appKey = strings.TrimSpace(appKey)
	if bundleKey == "" || appKey == "" {
		return fmt.Errorf("bundle_key and app_key are required")
	}

	// Do not rely solely on the database foreign-key cascade here. Existing
	// deployments created before the constraint was introduced must still remove
	// entitlement-bearing assets when an app is deleted.
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin delete download app transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM download_assets WHERE bundle_key = $1 AND app_key = $2`, bundleKey, appKey); err != nil {
		return fmt.Errorf("delete download app assets: %w", err)
	}

	result, err := tx.Exec(`DELETE FROM download_apps WHERE bundle_key = $1 AND app_key = $2`, bundleKey, appKey)
	if err != nil {
		return fmt.Errorf("delete download app: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return ErrAppNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete download app: %w", err)
	}

	return nil
}

// GetAsset fetches a download artifact by platform.
func (s *CatalogService) GetAsset(bundleKey, appKey, platform string) (*Asset, error) {
	query := `
		SELECT id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2 AND platform = $3
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey, platform)
	var t AssetScanTargets
	if err := row.Scan(t.ScanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s/%s/%s", ErrAssetNotFound, bundleKey, appKey, platform)
		}
		return nil, err
	}
	asset := t.Hydrate()

	return &asset, nil
}

// GetAssetByVariant fetches a download asset by platform and variant_key.
func (s *CatalogService) GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*Asset, error) {
	query := `
		SELECT id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2 AND platform = $3 AND variant_key = $4
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey, platform, variantKey)
	var t AssetScanTargets
	if err := row.Scan(t.ScanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s/%s/%s/%s", ErrAssetNotFound, bundleKey, appKey, platform, variantKey)
		}
		return nil, err
	}
	asset := t.Hydrate()

	return &asset, nil
}

// UpsertAsset creates or updates a single download asset row (does not touch other platforms).
func (s *CatalogService) UpsertAsset(ctx context.Context, asset Asset) (*Asset, error) {
	asset.BundleKey = strings.TrimSpace(asset.BundleKey)
	asset.AppKey = strings.TrimSpace(asset.AppKey)
	asset.Platform = strings.TrimSpace(asset.Platform)
	asset.ArtifactURL = strings.TrimSpace(asset.ArtifactURL)
	asset.ReleaseVersion = strings.TrimSpace(asset.ReleaseVersion)
	asset.ReleaseNotes = strings.TrimSpace(asset.ReleaseNotes)
	asset.Checksum = strings.TrimSpace(asset.Checksum)
	asset.VariantKey = strings.TrimSpace(asset.VariantKey)
	if asset.VariantKey == "" {
		asset.VariantKey = "default"
	}

	assetSource := strings.TrimSpace(asset.ArtifactSource)
	if assetSource == "" {
		assetSource = "direct"
	}

	if asset.BundleKey == "" || asset.AppKey == "" || asset.Platform == "" {
		return nil, fmt.Errorf("bundle_key, app_key, and platform are required")
	}
	if asset.ReleaseVersion == "" {
		return nil, fmt.Errorf("release_version is required")
	}
	if assetSource != "direct" && assetSource != "managed" {
		return nil, fmt.Errorf("artifact_source must be 'direct' or 'managed'")
	}
	if assetSource == "direct" {
		if err := ValidateDirectArtifactURL(asset.ArtifactURL); err != nil {
			return nil, err
		}
	} else if asset.ArtifactID == nil || *asset.ArtifactID == 0 {
		return nil, fmt.Errorf("artifact_id is required when artifact_source is managed")
	}

	metadataBytes, err := json.Marshal(asset.Metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal asset metadata: %w", err)
	}
	var artifactID interface{}
	if asset.ArtifactID != nil && *asset.ArtifactID != 0 {
		artifactID = *asset.ArtifactID
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO download_assets (
			bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id,
			release_version, release_notes, checksum, requires_entitlement, metadata, variant_key, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12, NOW())
		ON CONFLICT (bundle_key, app_key, platform, variant_key) DO UPDATE SET
			artifact_url = EXCLUDED.artifact_url,
			artifact_source = EXCLUDED.artifact_source,
			artifact_id = EXCLUDED.artifact_id,
			release_version = EXCLUDED.release_version,
			release_notes = EXCLUDED.release_notes,
			checksum = EXCLUDED.checksum,
			requires_entitlement = EXCLUDED.requires_entitlement,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
	`, asset.BundleKey, asset.AppKey, asset.Platform, asset.ArtifactURL, assetSource, artifactID,
		asset.ReleaseVersion, asset.ReleaseNotes, asset.Checksum, asset.RequiresEntitlement, metadataBytes, asset.VariantKey)
	if err != nil {
		return nil, fmt.Errorf("upsert download asset: %w", err)
	}

	return s.GetAsset(asset.BundleKey, asset.AppKey, asset.Platform)
}

// ListChannels returns available channels with latest version per platform for an app.
func (s *CatalogService) ListChannels(bundleKey, appKey string) ([]ChannelInfo, error) {
	rows, err := s.db.Query(`
		SELECT
			da.variant_key,
			da.platform,
			COALESCE(art.release_version, da.release_version) AS version,
			COALESCE(art.updated_at, NOW()) AS updated_at
		FROM download_assets da
		LEFT JOIN download_artifacts art ON da.artifact_id = art.id
		WHERE da.bundle_key = $1 AND da.app_key = $2
		ORDER BY da.variant_key, da.platform
	`, bundleKey, appKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []ChannelInfo
	for rows.Next() {
		var variantKey, platform, version string
		var updatedAt time.Time
		if err := rows.Scan(&variantKey, &platform, &version, &updatedAt); err != nil {
			return nil, err
		}
		channel := variantKey
		if variantKey == "default" {
			channel = "stable"
		}
		channels = append(channels, ChannelInfo{
			Channel:   channel,
			Platform:  platform,
			Version:   version,
			UpdatedAt: updatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	if channels == nil {
		channels = []ChannelInfo{}
	}
	return channels, nil
}

// UpdateAppPolicy updates the update_policy JSONB column for an app.
func (s *CatalogService) UpdateAppPolicy(bundleKey, appKey string, policy UpdatePolicy) error {
	policyBytes, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("marshal policy: %w", err)
	}
	result, err := s.db.Exec(`
		UPDATE download_apps SET update_policy = $3, updated_at = NOW()
		WHERE bundle_key = $1 AND app_key = $2
	`, bundleKey, appKey, policyBytes)
	if err != nil {
		return fmt.Errorf("update policy: %w", err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrAppNotFound
	}
	return nil
}
