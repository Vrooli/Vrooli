// Package download is the application layer for platform-artifact downloads:
// listing/upserting install experiences (apps) and their per-platform assets,
// and gating asset access behind entitlement checks. The Connect handler in
// handlers/download is a thin proto<->domain adapter over this Service and its
// Authorizer.
package download

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Domain errors surfaced to the handler, which maps them to Connect codes.
var (
	ErrNotFound            = errors.New("download not found")
	ErrAppNotFound         = errors.New("download app not found")
	ErrRequiresActive      = errors.New("active subscription required for downloads")
	ErrIdentityRequired    = errors.New("user identity required for gated downloads")
	ErrPlatformRequired    = errors.New("platform is required")
	ErrEntitlementsUnavail = errors.New("entitlements unavailable")
)

// Asset is a gated downloadable artifact for one platform.
type Asset struct {
	ID                  int64
	BundleKey           string
	AppKey              string
	Platform            string
	ArtifactURL         string
	ReleaseVersion      string
	ReleaseNotes        string
	Checksum            string
	RequiresEntitlement bool
	Metadata            map[string]interface{}
}

// Storefront is an app-store link for an install experience.
type Storefront struct {
	Store string `json:"store"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Badge string `json:"badge,omitempty"`
}

// App models an install experience with one or more platform artifacts.
type App struct {
	ID              int64
	BundleKey       string
	AppKey          string
	Name            string
	Tagline         string
	Description     string
	InstallOverview string
	InstallSteps    []string
	Storefronts     []Storefront
	Metadata        map[string]interface{}
	DisplayOrder    int
	Platforms       []Asset
}

// Service reads and writes the download catalog.
type Service struct {
	db *sql.DB
}

// NewService constructs the download Service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

const assetColumns = `id, bundle_key, app_key, platform, artifact_url, release_version,
	COALESCE(release_notes, ''), COALESCE(checksum, ''), requires_entitlement, metadata`

func scanAsset(row interface{ Scan(...any) error }) (Asset, error) {
	var a Asset
	var metadataBytes []byte
	if err := row.Scan(&a.ID, &a.BundleKey, &a.AppKey, &a.Platform, &a.ArtifactURL,
		&a.ReleaseVersion, &a.ReleaseNotes, &a.Checksum, &a.RequiresEntitlement, &metadataBytes); err != nil {
		return Asset{}, err
	}
	a.Metadata = decodeMap(metadataBytes)
	return a, nil
}

// ListAssets returns all assets for a bundle, ordered by app then platform.
func (s *Service) ListAssets(bundleKey string) ([]Asset, error) {
	rows, err := s.db.Query(`SELECT `+assetColumns+` FROM download_assets WHERE bundle_key = $1 ORDER BY app_key, platform`, bundleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var assets []Asset
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

const appColumns = `id, bundle_key, app_key, name, COALESCE(tagline, ''), COALESCE(description, ''),
	COALESCE(install_overview, ''), install_steps, storefronts, metadata, display_order`

func scanApp(row interface{ Scan(...any) error }) (App, error) {
	var app App
	var stepsBytes, storefrontBytes, metadataBytes []byte
	if err := row.Scan(&app.ID, &app.BundleKey, &app.AppKey, &app.Name, &app.Tagline, &app.Description,
		&app.InstallOverview, &stepsBytes, &storefrontBytes, &metadataBytes, &app.DisplayOrder); err != nil {
		return App{}, err
	}
	if len(stepsBytes) > 0 {
		_ = json.Unmarshal(stepsBytes, &app.InstallSteps)
	}
	if len(storefrontBytes) > 0 {
		_ = json.Unmarshal(storefrontBytes, &app.Storefronts)
	}
	app.Metadata = decodeMap(metadataBytes)
	return app, nil
}

// ListApps returns download apps with their grouped platform assets.
func (s *Service) ListApps(bundleKey string) ([]App, error) {
	rows, err := s.db.Query(`SELECT `+appColumns+` FROM download_apps WHERE bundle_key = $1 ORDER BY display_order, name`, bundleKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []App
	for rows.Next() {
		app, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return apps, nil
	}

	assets, err := s.ListAssets(bundleKey)
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]Asset)
	for _, asset := range assets {
		grouped[asset.AppKey] = append(grouped[asset.AppKey], asset)
	}
	for i := range apps {
		apps[i].Platforms = grouped[apps[i].AppKey]
	}
	return apps, nil
}

// GetApp fetches a single app with its assets.
func (s *Service) GetApp(bundleKey, appKey string) (*App, error) {
	app, err := scanApp(s.db.QueryRow(`SELECT `+appColumns+` FROM download_apps WHERE bundle_key = $1 AND app_key = $2 LIMIT 1`, bundleKey, appKey))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAppNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT `+assetColumns+` FROM download_assets WHERE bundle_key = $1 AND app_key = $2 ORDER BY platform`, bundleKey, appKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		app.Platforms = append(app.Platforms, asset)
	}
	return &app, rows.Err()
}

// GetAsset fetches one artifact by platform.
func (s *Service) GetAsset(bundleKey, appKey, platform string) (*Asset, error) {
	asset, err := scanAsset(s.db.QueryRow(`SELECT `+assetColumns+` FROM download_assets WHERE bundle_key = $1 AND app_key = $2 AND platform = $3 LIMIT 1`, bundleKey, appKey, platform))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s/%s/%s", ErrNotFound, bundleKey, appKey, platform)
	}
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// UpsertDownloadApp creates or replaces an app and its platform installers.
func (s *Service) UpsertDownloadApp(app App) (*App, error) {
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

	stepsBytes, _ := json.Marshal(app.InstallSteps)
	storefrontBytes, _ := json.Marshal(app.Storefronts)
	metadataBytes, _ := json.Marshal(app.Metadata)

	if _, err := tx.Exec(`
		INSERT INTO download_apps (
			bundle_key, app_key, name, tagline, description,
			install_overview, install_steps, storefronts, metadata, display_order
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (bundle_key, app_key) DO UPDATE SET
			name = EXCLUDED.name, tagline = EXCLUDED.tagline, description = EXCLUDED.description,
			install_overview = EXCLUDED.install_overview, install_steps = EXCLUDED.install_steps,
			storefronts = EXCLUDED.storefronts, metadata = EXCLUDED.metadata,
			display_order = EXCLUDED.display_order, updated_at = NOW()
	`, app.BundleKey, app.AppKey, app.Name, app.Tagline, app.Description, app.InstallOverview,
		stepsBytes, storefrontBytes, metadataBytes, app.DisplayOrder); err != nil {
		return nil, fmt.Errorf("upsert download app: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM download_assets WHERE bundle_key = $1 AND app_key = $2`, app.BundleKey, app.AppKey); err != nil {
		return nil, fmt.Errorf("clear existing assets: %w", err)
	}

	for _, asset := range app.Platforms {
		platform := strings.TrimSpace(asset.Platform)
		if platform == "" {
			return nil, fmt.Errorf("platform is required for all assets")
		}
		assetMeta, _ := json.Marshal(asset.Metadata)
		if _, err := tx.Exec(`
			INSERT INTO download_assets (
				bundle_key, app_key, platform, artifact_url,
				release_version, release_notes, checksum, requires_entitlement, metadata
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`, app.BundleKey, app.AppKey, platform, strings.TrimSpace(asset.ArtifactURL),
			strings.TrimSpace(asset.ReleaseVersion), strings.TrimSpace(asset.ReleaseNotes),
			strings.TrimSpace(asset.Checksum), asset.RequiresEntitlement, assetMeta); err != nil {
			return nil, fmt.Errorf("insert asset for platform %s: %w", platform, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit download app upsert: %w", err)
	}
	return s.GetApp(app.BundleKey, app.AppKey)
}

// Entitlements is the minimal entitlement view the authorizer gates on.
type Entitlements struct {
	Status string
}

// EntitlementProvider resolves a caller's entitlement status.
type EntitlementProvider interface {
	GetEntitlements(userIdentity string) (*Entitlements, error)
}

type assetLookup interface {
	GetAsset(bundleKey, appKey, platform string) (*Asset, error)
}

// Authorizer gates asset access behind entitlement checks (order-sensitive: app
// key, platform, asset existence, then entitlement).
type Authorizer struct {
	downloads    assetLookup
	entitlements EntitlementProvider
	bundleKey    string
}

// NewAuthorizer wires the download lookup, entitlement provider, and bundle key.
func NewAuthorizer(downloads assetLookup, entitlements EntitlementProvider, bundleKey string) *Authorizer {
	return &Authorizer{downloads: downloads, entitlements: entitlements, bundleKey: bundleKey}
}

// Authorize returns the requested asset if the caller may access it.
func (a *Authorizer) Authorize(appKey, platform, userIdentity string) (*Asset, error) {
	trimmedApp := strings.TrimSpace(appKey)
	if trimmedApp == "" {
		return nil, ErrAppNotFound
	}
	trimmedPlatform := strings.TrimSpace(platform)
	if trimmedPlatform == "" {
		return nil, ErrPlatformRequired
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
		return nil, ErrIdentityRequired
	}

	entitlements, err := a.entitlements.GetEntitlements(userIdentity)
	if err != nil {
		return nil, fmt.Errorf("retrieve entitlements: %w", err)
	}
	if entitlements == nil {
		return nil, fmt.Errorf("retrieve entitlements: %w", ErrEntitlementsUnavail)
	}
	if entitlements.Status != "active" && entitlements.Status != "trialing" {
		return nil, ErrRequiresActive
	}
	return asset, nil
}

func decodeMap(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}
