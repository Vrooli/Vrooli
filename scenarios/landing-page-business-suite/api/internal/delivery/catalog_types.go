package delivery

import (
	"database/sql"
	"encoding/json"
	"strings"
)

// Asset represents a gated downloadable artifact.
type Asset struct {
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

// AssetScanTargets holds temporary nullable scan variables for a download_assets row.
type AssetScanTargets struct {
	asset          Asset
	artifactURL    sql.NullString
	artifactSource sql.NullString
	artifactID     sql.NullInt64
	metadataBytes  []byte
}

// ScanDest returns the ordered slice of scan destinations matching the standard asset SELECT columns:
// id, bundle_key, app_key, platform, artifact_url, artifact_source, artifact_id,
// release_version, release_notes, checksum, requires_entitlement, metadata
func (t *AssetScanTargets) ScanDest() []interface{} {
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

// Hydrate populates the asset struct from the scanned nullable values.
func (t *AssetScanTargets) Hydrate() Asset {
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

// Storefront represents an app store link for mobile/desktop stores.
type Storefront struct {
	Store string `json:"store"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Badge string `json:"badge,omitempty"`
}

// App models an install experience that can include multiple artifacts.
type App struct {
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
	Storefronts     []Storefront           `json:"storefronts,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	DisplayOrder    int                    `json:"display_order"`
	UpdateAPIKey    string                 `json:"update_api_key,omitempty"`
	UpdatePolicy    map[string]interface{} `json:"update_policy,omitempty"`
	// Platforms is a required array in the public landing configuration contract.
	// Keep it non-nil for apps without a published installer so JSON emits []
	// rather than omitting the field or returning null.
	Platforms []Asset `json:"platforms"`
}

// AppScanTargets holds temporary nullable scan variables for a download_apps row.
// Use ScanDest() to get the ordered scan destinations, then Hydrate() to populate the app.
type AppScanTargets struct {
	app               App
	iconURL           sql.NullString
	screenshotURL     sql.NullString
	updateAPIKey      sql.NullString
	installStepsBytes []byte
	storefrontBytes   []byte
	metadataBytes     []byte
	updatePolicyBytes []byte
}

// ScanDest returns the ordered slice of scan destinations matching the standard app SELECT columns:
// id, bundle_key, app_key, name, tagline, description, icon_url, screenshot_url,
// install_overview, install_steps, storefronts, metadata, display_order, update_api_key, update_policy
func (t *AppScanTargets) ScanDest() []interface{} {
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

// Hydrate populates the app struct from the scanned nullable values.
func (t *AppScanTargets) Hydrate() App {
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
		var storefronts []Storefront
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
