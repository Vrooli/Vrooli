package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// DownloadService provides helpers for retrieving bundle download metadata.
type DownloadService struct {
	db *sql.DB
}

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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("download not found for %s/%s", bundleKey, platform)
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
