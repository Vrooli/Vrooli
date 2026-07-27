package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/database"
	delivery "landing-page-business-suite-api/internal/download"
)

// DownloadService provides helpers for retrieving bundle download metadata.
type DownloadService struct {
	db DownloadStore
}

// DownloadStore is the transaction-capable persistence boundary for download
// metadata and entitlement-gated artifact configuration.
type DownloadStore interface {
	Query(string, ...any) (*sql.Rows, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
	Exec(string, ...any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// routedDownloadStore adapts RoutedDB to the established download persistence
// boundary. Context-free compatibility operations intentionally use primary;
// request-aware reads and writes retain the test-mode routing marker.
type routedDownloadStore struct {
	db *database.RoutedDB
}

func newRoutedDownloadStore(db *database.RoutedDB) DownloadStore {
	return routedDownloadStore{db: db}
}

func (s routedDownloadStore) Query(query string, args ...any) (*sql.Rows, error) {
	return s.db.Primary().Query(query, args...)
}

func (s routedDownloadStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s routedDownloadStore) QueryRow(query string, args ...any) *sql.Row {
	// #nosec G701 -- DownloadService owns all call sites and supplies package-constant SQL.
	// Dynamic SQL is not accepted at this persistence boundary.
	return s.db.Primary().QueryRow(query, args...)
}

func (s routedDownloadStore) Exec(query string, args ...any) (sql.Result, error) {
	return s.db.Primary().Exec(query, args...)
}

func (s routedDownloadStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s routedDownloadStore) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}

var (
	// ErrDownloadNotFound indicates the requested artifact is not configured.
	ErrDownloadNotFound = errors.New("download not found")
	// ErrDownloadAppNotFound indicates the requested app is not configured.
	ErrDownloadAppNotFound = errors.New("download app not found")
	// ErrDownloadRequiresActiveSubscription indicates a gated download without active access.
	ErrDownloadRequiresActiveSubscription = errors.New("active subscription required for downloads")
	// ErrDownloadIdentityRequired indicates the caller must provide identity details before accessing gated assets.
	ErrDownloadIdentityRequired = errors.New("user identity required for gated downloads")
	// ErrDownloadPlatformRequired indicates the platform input was blank.
	ErrDownloadPlatformRequired = errors.New("platform is required")
	// ErrDownloadEntitlementsUnavailable indicates the entitlement provider returned an unusable response.
	ErrDownloadEntitlementsUnavailable = errors.New("entitlements unavailable")
)

// DownloadAsset represents a gated downloadable artifact.
type DownloadAsset struct {
	ID                  int64                  `json:"id"`
	BundleKey           string                 `json:"bundle_key"`
	AppKey              string                 `json:"app_key"`
	Platform            string                 `json:"platform"`
	ArtifactURL         string                 `json:"artifact_url"`
	ArtifactSource      string                 `json:"artifact_source"`
	ArtifactID          *int64                 `json:"artifact_id,omitempty"`
	ReleaseVersion      string                 `json:"release_version"`
	ReleaseNotes        string                 `json:"release_notes,omitempty"`
	Checksum            string                 `json:"checksum,omitempty"`
	VariantKey          string                 `json:"variant_key,omitempty"`
	RequiresEntitlement bool                   `json:"requires_entitlement"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	// Artifact info (populated when artifact_source is 'managed')
	ArtifactFilename  string `json:"artifact_filename,omitempty"`
	ArtifactSizeBytes int64  `json:"artifact_size_bytes,omitempty"`
	ArtifactCount     int    `json:"artifact_count,omitempty"`
}

// assetScanTargets holds temporary nullable scan variables for a download_assets row.
type assetScanTargets struct {
	asset          DownloadAsset
	artifactURL    sql.NullString
	artifactSource sql.NullString
	artifactID     sql.NullInt64
	metadataBytes  []byte
}

// scanDest returns the ordered slice of scan destinations matching the standard asset SELECT columns:
// id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id,
// release_version, release_notes, checksum, requires_entitlement, metadata
func (t *assetScanTargets) scanDest() []interface{} {
	return []interface{}{
		&t.asset.ID,
		&t.asset.BundleKey,
		&t.asset.AppKey,
		&t.asset.Platform,
		&t.artifactURL,
		&t.artifactSource,
		&t.artifactID,
		&t.asset.ReleaseVersion,
		&t.asset.ReleaseNotes,
		&t.asset.Checksum,
		&t.asset.RequiresEntitlement,
		&t.metadataBytes,
	}
}

// hydrate populates the asset struct from the scanned nullable values.
func (t *assetScanTargets) hydrate() DownloadAsset {
	t.asset.ArtifactURL = t.artifactURL.String
	if t.artifactSource.Valid && strings.TrimSpace(t.artifactSource.String) != "" {
		t.asset.ArtifactSource = t.artifactSource.String
	} else {
		t.asset.ArtifactSource = "direct"
	}
	if t.artifactID.Valid {
		id := t.artifactID.Int64
		t.asset.ArtifactID = &id
	}
	if len(t.metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(t.metadataBytes, &meta); err == nil {
			t.asset.Metadata = meta
		}
	}
	return t.asset
}

// DownloadStorefront represents an app store link for mobile/desktop stores.
type DownloadStorefront struct {
	Store string `json:"store"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Badge string `json:"badge,omitempty"`
}

// DownloadApp models an install experience that can include multiple artifacts.
type DownloadApp struct {
	ID              int64                  `json:"id"`
	BundleKey       string                 `json:"bundle_key"`
	AppKey          string                 `json:"app_key"`
	Name            string                 `json:"name"`
	Tagline         string                 `json:"tagline,omitempty"`
	Description     string                 `json:"description,omitempty"`
	IconURL         string                 `json:"icon_url,omitempty"`
	ScreenshotURL   string                 `json:"screenshot_url,omitempty"`
	InstallOverview string                 `json:"install_overview,omitempty"`
	InstallSteps    []string               `json:"install_steps,omitempty"`
	Storefronts     []DownloadStorefront   `json:"storefronts,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	DisplayOrder    int                    `json:"display_order"`
	UpdateAPIKey    string                 `json:"update_api_key,omitempty"`
	UpdatePolicy    map[string]interface{} `json:"update_policy,omitempty"`
	// Platforms is a required array in the public landing configuration contract.
	// Keep it non-nil for apps without a published installer so JSON emits []
	// rather than omitting the field or returning null.
	Platforms []DownloadAsset `json:"platforms"`
}

// appScanTargets holds temporary nullable scan variables for a download_apps row.
// Use scanDest() to get the ordered scan destinations, then hydrate() to populate the app.
type appScanTargets struct {
	app               DownloadApp
	iconURL           sql.NullString
	screenshotURL     sql.NullString
	updateAPIKey      sql.NullString
	installStepsBytes []byte
	storefrontBytes   []byte
	metadataBytes     []byte
	updatePolicyBytes []byte
}

// scanDest returns the ordered slice of scan destinations matching the standard app SELECT columns:
// id, bundle_key, app_key, name, tagline, description, icon_url, screenshot_url,
// install_overview, install_steps, storefronts, metadata, display_order, update_api_key, update_policy
func (t *appScanTargets) scanDest() []interface{} {
	return []interface{}{
		&t.app.ID,
		&t.app.BundleKey,
		&t.app.AppKey,
		&t.app.Name,
		&t.app.Tagline,
		&t.app.Description,
		&t.iconURL,
		&t.screenshotURL,
		&t.app.InstallOverview,
		&t.installStepsBytes,
		&t.storefrontBytes,
		&t.metadataBytes,
		&t.app.DisplayOrder,
		&t.updateAPIKey,
		&t.updatePolicyBytes,
	}
}

// hydrate populates the app struct from the scanned nullable values.
func (t *appScanTargets) hydrate() DownloadApp {
	t.app.IconURL = t.iconURL.String
	t.app.ScreenshotURL = t.screenshotURL.String
	t.app.UpdateAPIKey = t.updateAPIKey.String
	if len(t.installStepsBytes) > 0 {
		var steps []string
		if err := json.Unmarshal(t.installStepsBytes, &steps); err == nil {
			t.app.InstallSteps = steps
		}
	}
	if len(t.storefrontBytes) > 0 {
		var storefronts []DownloadStorefront
		if err := json.Unmarshal(t.storefrontBytes, &storefronts); err == nil {
			t.app.Storefronts = storefronts
		}
	}
	if len(t.metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(t.metadataBytes, &meta); err == nil {
			t.app.Metadata = meta
		}
	}
	if len(t.updatePolicyBytes) > 0 {
		var policy map[string]interface{}
		if err := json.Unmarshal(t.updatePolicyBytes, &policy); err == nil {
			t.app.UpdatePolicy = policy
		}
	}
	return t.app
}

func NewDownloadService(db DownloadStore) *DownloadService {
	return &DownloadService{db: db}
}

func validateDirectArtifactURL(raw string) error {
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
func (s *DownloadService) ListAssets(bundleKey string) ([]DownloadAsset, error) {
	return s.ListAssetsContext(context.Background(), bundleKey)
}

// ListAssetsContext reads assets from the pool selected for ctx. Requests
// marked by api-core test mode therefore stay inside their leased database.
func (s *DownloadService) ListAssetsContext(ctx context.Context, bundleKey string) ([]DownloadAsset, error) {
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

	assets := make([]DownloadAsset, 0)
	for rows.Next() {
		var t assetScanTargets
		var artifactFilename sql.NullString
		var artifactSizeBytes sql.NullInt64
		var artifactCount sql.NullInt64
		dest := append(t.scanDest(), &artifactFilename, &artifactSizeBytes, &artifactCount)
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		asset := t.hydrate()
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
func (s *DownloadService) ListApps(bundleKey string) ([]DownloadApp, error) {
	return s.ListAppsContext(context.Background(), bundleKey)
}

// ListAppsContext reads app metadata and its assets from the pool selected for
// ctx. It is the request-aware counterpart used by public landing rendering.
func (s *DownloadService) ListAppsContext(ctx context.Context, bundleKey string) ([]DownloadApp, error) {
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

	apps := make([]DownloadApp, 0)
	for rows.Next() {
		var t appScanTargets
		if err := rows.Scan(t.scanDest()...); err != nil {
			return nil, err
		}
		apps = append(apps, t.hydrate())
	}

	if len(apps) == 0 {
		return apps, nil
	}

	assets, err := s.ListAssetsContext(ctx, bundleKey)
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]DownloadAsset)
	for _, asset := range assets {
		grouped[asset.AppKey] = append(grouped[asset.AppKey], asset)
	}

	for i := range apps {
		app := &apps[i]
		app.Platforms = grouped[app.AppKey]
		if app.Platforms == nil {
			app.Platforms = make([]DownloadAsset, 0)
		}
	}

	return apps, nil
}

// GetApp fetches a single download app with its assets.
func (s *DownloadService) GetApp(bundleKey, appKey string) (*DownloadApp, error) {
	query := `
		SELECT id, bundle_key, app_key, name, tagline, description,
		       icon_url, screenshot_url, install_overview, install_steps, storefronts, metadata, display_order, update_api_key, update_policy
		FROM download_apps
		WHERE bundle_key = $1 AND app_key = $2
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey)
	var t appScanTargets
	if err := row.Scan(t.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDownloadAppNotFound
		}
		return nil, err
	}
	app := t.hydrate()

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
		var t assetScanTargets
		if err := assetRows.Scan(t.scanDest()...); err != nil {
			return nil, err
		}
		app.Platforms = append(app.Platforms, t.hydrate())
	}

	return &app, nil
}

// UpsertDownloadApp creates or updates an app definition along with its platform installers.
func (s *DownloadService) UpsertDownloadApp(app DownloadApp) (*DownloadApp, error) {
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
				if err := validateDirectArtifactURL(asset.ArtifactURL); err != nil {
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
func (s *DownloadService) DeleteApp(bundleKey, appKey string) error {
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
		return ErrDownloadAppNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete download app: %w", err)
	}

	return nil
}

// GetAsset fetches a download artifact by platform.
func (s *DownloadService) GetAsset(bundleKey, appKey, platform string) (*DownloadAsset, error) {
	query := `
		SELECT id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2 AND platform = $3
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey, platform)
	var t assetScanTargets
	if err := row.Scan(t.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s/%s/%s", ErrDownloadNotFound, bundleKey, appKey, platform)
		}
		return nil, err
	}
	asset := t.hydrate()

	return &asset, nil
}

// GetAssetByVariant fetches a download asset by platform and variant_key.
func (s *DownloadService) GetAssetByVariant(bundleKey, appKey, platform, variantKey string) (*DownloadAsset, error) {
	query := `
		SELECT id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2 AND platform = $3 AND variant_key = $4
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey, platform, variantKey)
	var t assetScanTargets
	if err := row.Scan(t.scanDest()...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s/%s/%s/%s", ErrDownloadNotFound, bundleKey, appKey, platform, variantKey)
		}
		return nil, err
	}
	asset := t.hydrate()

	return &asset, nil
}

// UpsertAsset creates or updates a single download asset row (does not touch other platforms).
func (s *DownloadService) UpsertAsset(ctx context.Context, asset DownloadAsset) (*DownloadAsset, error) {
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
		if err := validateDirectArtifactURL(asset.ArtifactURL); err != nil {
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
func (s *DownloadService) ListChannels(bundleKey, appKey string) ([]ChannelInfo, error) {
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
func (s *DownloadService) UpdateAppPolicy(bundleKey, appKey string, policy UpdatePolicy) error {
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
		return ErrDownloadAppNotFound
	}
	return nil
}

type downloadAssetLookup interface {
	GetAsset(bundleKey, appKey, platform string) (*DownloadAsset, error)
}

type entitlementProvider interface {
	GetEntitlementsContext(context.Context, string) (*EntitlementPayload, error)
}

type entitlementStatusLookup struct {
	ctx      context.Context
	provider entitlementProvider
}

func (l entitlementStatusLookup) GetStatus(userIdentity string) (string, error) {
	entitlements, err := l.provider.GetEntitlementsContext(l.ctx, userIdentity)
	if err != nil {
		return "", err
	}
	if entitlements == nil {
		return "", delivery.ErrEntitlementsUnavailable
	}
	return entitlements.Status, nil
}

// DownloadAuthorizer coordinates entitlement checks before returning assets.
type DownloadAuthorizer struct {
	downloads    downloadAssetLookup
	entitlements entitlementProvider
	bundleKey    string
}

// NewDownloadAuthorizer wires the dependencies required for download gating.
func NewDownloadAuthorizer(downloads downloadAssetLookup, entitlements entitlementProvider, bundleKey string) *DownloadAuthorizer {
	return &DownloadAuthorizer{
		downloads:    downloads,
		entitlements: entitlements,
		bundleKey:    bundleKey,
	}
}

// Authorize ensures the caller can access the requested download asset.
func (a *DownloadAuthorizer) Authorize(ctx context.Context, appKey string, platform string, userIdentity string) (*DownloadAsset, error) {
	trimmedApp := strings.TrimSpace(appKey)
	if trimmedApp == "" {
		return nil, ErrDownloadAppNotFound
	}
	trimmedPlatform := strings.TrimSpace(platform)
	if trimmedPlatform == "" {
		return nil, ErrDownloadPlatformRequired
	}

	asset, err := a.downloads.GetAsset(a.bundleKey, trimmedApp, trimmedPlatform)
	if err != nil {
		return nil, err
	}

	err = delivery.Authorize(delivery.Request{
		AppKey:              trimmedApp,
		Platform:            trimmedPlatform,
		UserIdentity:        userIdentity,
		RequiresEntitlement: asset.RequiresEntitlement,
	}, entitlementStatusLookup{ctx: ctx, provider: a.entitlements})
	if err != nil {
		switch {
		case errors.Is(err, delivery.ErrIdentityRequired):
			return nil, ErrDownloadIdentityRequired
		case errors.Is(err, delivery.ErrRequiresActiveSubscription):
			return nil, ErrDownloadRequiresActiveSubscription
		case errors.Is(err, delivery.ErrEntitlementsUnavailable):
			return nil, fmt.Errorf("retrieve entitlements: %w", ErrDownloadEntitlementsUnavailable)
		default:
			return nil, err
		}
	}

	return asset, nil
}
