package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// DownloadService provides helpers for retrieving bundle download metadata.
type DownloadService struct {
	db *sql.DB
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
	ReleaseVersion      string                 `json:"release_version"`
	ReleaseNotes        string                 `json:"release_notes,omitempty"`
	Checksum            string                 `json:"checksum,omitempty"`
	RequiresEntitlement bool                   `json:"requires_entitlement"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
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
	InstallOverview string                 `json:"install_overview,omitempty"`
	InstallSteps    []string               `json:"install_steps,omitempty"`
	Storefronts     []DownloadStorefront   `json:"storefronts,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	DisplayOrder    int                    `json:"display_order"`
	Platforms       []DownloadAsset        `json:"platforms,omitempty"`
}

func NewDownloadService(db *sql.DB) *DownloadService {
	return &DownloadService{db: db}
}

// ListAssets returns all download assets for a bundle.
func (s *DownloadService) ListAssets(bundleKey string) ([]DownloadAsset, error) {
	query := `
		SELECT id, bundle_key, app_key, platform, artifact_url, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1
		ORDER BY app_key, platform
	`

	rows, err := s.db.Query(query, bundleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []DownloadAsset
	for rows.Next() {
		var asset DownloadAsset
		var metadataBytes []byte
		if err := rows.Scan(
			&asset.ID,
			&asset.BundleKey,
			&asset.AppKey,
			&asset.Platform,
			&asset.ArtifactURL,
			&asset.ReleaseVersion,
			&asset.ReleaseNotes,
			&asset.Checksum,
			&asset.RequiresEntitlement,
			&metadataBytes,
		); err != nil {
			return nil, err
		}

		if len(metadataBytes) > 0 {
			var meta map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &meta); err == nil {
				asset.Metadata = meta
			}
		}

		assets = append(assets, asset)
	}

	return assets, nil
}

// ListApps returns download apps with their associated platform assets.
func (s *DownloadService) ListApps(bundleKey string) ([]DownloadApp, error) {
	query := `
		SELECT id, bundle_key, app_key, name, tagline, description,
		       install_overview, install_steps, storefronts, metadata, display_order
		FROM download_apps
		WHERE bundle_key = $1
		ORDER BY display_order, name
	`

	rows, err := s.db.Query(query, bundleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []DownloadApp
	for rows.Next() {
		var app DownloadApp
		var installStepsBytes, storefrontBytes, metadataBytes []byte
		if err := rows.Scan(
			&app.ID,
			&app.BundleKey,
			&app.AppKey,
			&app.Name,
			&app.Tagline,
			&app.Description,
			&app.InstallOverview,
			&installStepsBytes,
			&storefrontBytes,
			&metadataBytes,
			&app.DisplayOrder,
		); err != nil {
			return nil, err
		}

		if len(installStepsBytes) > 0 {
			var steps []string
			if err := json.Unmarshal(installStepsBytes, &steps); err == nil {
				app.InstallSteps = steps
			}
		}

		if len(storefrontBytes) > 0 {
			var storefronts []DownloadStorefront
			if err := json.Unmarshal(storefrontBytes, &storefronts); err == nil {
				app.Storefronts = storefronts
			}
		}

		if len(metadataBytes) > 0 {
			var meta map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &meta); err == nil {
				app.Metadata = meta
			}
		}

		apps = append(apps, app)
	}

	if len(apps) == 0 {
		return apps, nil
	}

	assets, err := s.ListAssets(bundleKey)
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
	}

	return apps, nil
}

// GetApp fetches a single download app with its assets.
func (s *DownloadService) GetApp(bundleKey, appKey string) (*DownloadApp, error) {
	query := `
		SELECT id, bundle_key, app_key, name, tagline, description,
		       install_overview, install_steps, storefronts, metadata, display_order
		FROM download_apps
		WHERE bundle_key = $1 AND app_key = $2
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey)
	var app DownloadApp
	var installStepsBytes, storefrontBytes, metadataBytes []byte
	if err := row.Scan(
		&app.ID,
		&app.BundleKey,
		&app.AppKey,
		&app.Name,
		&app.Tagline,
		&app.Description,
		&app.InstallOverview,
		&installStepsBytes,
		&storefrontBytes,
		&metadataBytes,
		&app.DisplayOrder,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDownloadAppNotFound
		}
		return nil, err
	}

	if len(installStepsBytes) > 0 {
		var steps []string
		if err := json.Unmarshal(installStepsBytes, &steps); err == nil {
			app.InstallSteps = steps
		}
	}
	if len(storefrontBytes) > 0 {
		var storefronts []DownloadStorefront
		if err := json.Unmarshal(storefrontBytes, &storefronts); err == nil {
			app.Storefronts = storefronts
		}
	}
	if len(metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			app.Metadata = meta
		}
	}

	assetRows, err := s.db.Query(`
		SELECT id, bundle_key, app_key, platform, artifact_url, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2
		ORDER BY platform
	`, bundleKey, appKey)
	if err != nil {
		return nil, err
	}
	defer assetRows.Close()

	for assetRows.Next() {
		var asset DownloadAsset
		var metadataBytes []byte
		if err := assetRows.Scan(
			&asset.ID,
			&asset.BundleKey,
			&asset.AppKey,
			&asset.Platform,
			&asset.ArtifactURL,
			&asset.ReleaseVersion,
			&asset.ReleaseNotes,
			&asset.Checksum,
			&asset.RequiresEntitlement,
			&metadataBytes,
		); err != nil {
			return nil, err
		}

		if len(metadataBytes) > 0 {
			var meta map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &meta); err == nil {
				asset.Metadata = meta
			}
		}

		app.Platforms = append(app.Platforms, asset)
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
	defer tx.Rollback()

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

	_, err = tx.Exec(`
		INSERT INTO download_apps (
			bundle_key, app_key, name, tagline, description,
			install_overview, install_steps, storefronts, metadata, display_order
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (bundle_key, app_key) DO UPDATE SET
			name = EXCLUDED.name,
			tagline = EXCLUDED.tagline,
			description = EXCLUDED.description,
			install_overview = EXCLUDED.install_overview,
			install_steps = EXCLUDED.install_steps,
			storefronts = EXCLUDED.storefronts,
			metadata = EXCLUDED.metadata,
			display_order = EXCLUDED.display_order,
			updated_at = NOW()
	`, app.BundleKey, app.AppKey, app.Name, app.Tagline, app.Description, app.InstallOverview, installStepsBytes, storefrontBytes, metadataBytes, app.DisplayOrder)
	if err != nil {
		return nil, fmt.Errorf("upsert download app: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM download_assets WHERE bundle_key = $1 AND app_key = $2`, app.BundleKey, app.AppKey); err != nil {
		return nil, fmt.Errorf("clear existing assets: %w", err)
	}

	if len(app.Platforms) > 0 {
		stmt, err := tx.Prepare(`
			INSERT INTO download_assets (
				bundle_key, app_key, platform, artifact_url,
				release_version, release_notes, checksum, requires_entitlement, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
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
			metadataBytes, err := json.Marshal(asset.Metadata)
			if err != nil {
				return nil, fmt.Errorf("marshal asset metadata: %w", err)
			}
			if _, err := stmt.Exec(
				app.BundleKey,
				app.AppKey,
				platform,
				strings.TrimSpace(asset.ArtifactURL),
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

	result, err := s.db.Exec(`DELETE FROM download_apps WHERE bundle_key = $1 AND app_key = $2`, bundleKey, appKey)
	if err != nil {
		return fmt.Errorf("delete download app: %w", err)
	}

	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return ErrDownloadAppNotFound
	}

	return nil
}

// GetAsset fetches a download artifact by platform.
func (s *DownloadService) GetAsset(bundleKey, appKey, platform string) (*DownloadAsset, error) {
	query := `
		SELECT id, bundle_key, app_key, platform, artifact_url, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2 AND platform = $3
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, appKey, platform)
	var asset DownloadAsset
	var metadataBytes []byte
	if err := row.Scan(
		&asset.ID,
		&asset.BundleKey,
		&asset.AppKey,
		&asset.Platform,
		&asset.ArtifactURL,
		&asset.ReleaseVersion,
		&asset.ReleaseNotes,
		&asset.Checksum,
		&asset.RequiresEntitlement,
		&metadataBytes,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s/%s/%s", ErrDownloadNotFound, bundleKey, appKey, platform)
		}
		return nil, err
	}

	if len(metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			asset.Metadata = meta
		}
	}

	return &asset, nil
}

type downloadAssetLookup interface {
	GetAsset(bundleKey, appKey, platform string) (*DownloadAsset, error)
}

type entitlementProvider interface {
	GetEntitlements(userIdentity string) (*EntitlementPayload, error)
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
func (a *DownloadAuthorizer) Authorize(appKey string, platform string, userIdentity string) (*DownloadAsset, error) {
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

	if !asset.RequiresEntitlement {
		return asset, nil
	}

	userIdentity = strings.TrimSpace(userIdentity)
	if userIdentity == "" {
		return nil, ErrDownloadIdentityRequired
	}

	entitlements, err := a.entitlements.GetEntitlements(userIdentity)
	if err != nil {
		return nil, fmt.Errorf("retrieve entitlements: %w", err)
	}
	if entitlements == nil {
		return nil, fmt.Errorf("retrieve entitlements: %w", ErrDownloadEntitlementsUnavailable)
	}

	status := entitlements.Status
	if status != "active" && status != "trialing" {
		return nil, ErrDownloadRequiresActiveSubscription
	}

	return asset, nil
}
