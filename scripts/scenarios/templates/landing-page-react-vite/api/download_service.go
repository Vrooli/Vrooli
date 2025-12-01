package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// DownloadService provides helpers for retrieving bundle download metadata.
type DownloadService struct {
	db *sql.DB
}

var (
	// ErrDownloadNotFound indicates the requested artifact is not configured.
	ErrDownloadNotFound = errors.New("download not found")
	// ErrDownloadRequiresActiveSubscription indicates a gated download without active access.
	ErrDownloadRequiresActiveSubscription = errors.New("active subscription required for downloads")
)

// DownloadAsset represents a gated downloadable artifact.
type DownloadAsset struct {
	ID                  int64                  `json:"id"`
	BundleKey           string                 `json:"bundle_key"`
	Platform            string                 `json:"platform"`
	ArtifactURL         string                 `json:"artifact_url"`
	ReleaseVersion      string                 `json:"release_version"`
	ReleaseNotes        string                 `json:"release_notes,omitempty"`
	Checksum            string                 `json:"checksum,omitempty"`
	RequiresEntitlement bool                   `json:"requires_entitlement"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

func NewDownloadService(db *sql.DB) *DownloadService {
	return &DownloadService{db: db}
}

// ListAssets returns all download assets for a bundle.
func (s *DownloadService) ListAssets(bundleKey string) ([]DownloadAsset, error) {
	query := `
		SELECT id, bundle_key, platform, artifact_url, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1
		ORDER BY platform
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

// GetAsset fetches a download artifact by platform.
func (s *DownloadService) GetAsset(bundleKey, platform string) (*DownloadAsset, error) {
	query := `
		SELECT id, bundle_key, platform, artifact_url, release_version,
		       release_notes, checksum, requires_entitlement, metadata
		FROM download_assets
		WHERE bundle_key = $1 AND platform = $2
		LIMIT 1
	`

	row := s.db.QueryRow(query, bundleKey, platform)
	var asset DownloadAsset
	var metadataBytes []byte
	if err := row.Scan(
		&asset.ID,
		&asset.BundleKey,
		&asset.Platform,
		&asset.ArtifactURL,
		&asset.ReleaseVersion,
		&asset.ReleaseNotes,
		&asset.Checksum,
		&asset.RequiresEntitlement,
		&metadataBytes,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s/%s", ErrDownloadNotFound, bundleKey, platform)
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
	GetAsset(bundleKey, platform string) (*DownloadAsset, error)
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
func (a *DownloadAuthorizer) Authorize(platform string, userIdentity string) (*DownloadAsset, error) {
	asset, err := a.downloads.GetAsset(a.bundleKey, platform)
	if err != nil {
		return nil, err
	}

	if !asset.RequiresEntitlement {
		return asset, nil
	}

	entitlements, err := a.entitlements.GetEntitlements(userIdentity)
	if err != nil {
		return nil, fmt.Errorf("retrieve entitlements: %w", err)
	}

	status := entitlements.Status
	if status != "active" && status != "trialing" {
		return nil, ErrDownloadRequiresActiveSubscription
	}

	return asset, nil
}
