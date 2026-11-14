package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// BuiltArtifactMetadata is emitted by a desktop build. Registration consumes
// this artifact metadata instead of accepting hand-typed release values.
type BuiltArtifactMetadata struct {
	BundleKey           string `json:"bundle_key"`
	AppKey              string `json:"app_key"`
	Platform            string `json:"platform"`
	VariantKey          string `json:"variant_key"`
	ReleaseVersion      string `json:"release_version"`
	ArtifactURL         string `json:"artifact_url"`
	RequiresEntitlement bool   `json:"requires_entitlement"`
}

// RegisterBuiltArtifact verifies the artifact on disk and upserts the exact
// checksum/version/platform tuple emitted by the build.
func (s *CatalogService) RegisterBuiltArtifact(ctx context.Context, artifactPath, metadataPath string) (*Asset, error) {
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("read artifact metadata: %w", err)
	}
	var metadata BuiltArtifactMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("decode artifact metadata: %w", err)
	}
	for name, value := range map[string]string{
		"bundle_key": metadata.BundleKey, "app_key": metadata.AppKey, "platform": metadata.Platform, "release_version": metadata.ReleaseVersion,
	} {
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("artifact metadata %s is required", name)
		}
	}
	if _, err := s.GetApp(metadata.BundleKey, metadata.AppKey); err != nil {
		return nil, fmt.Errorf("artifact app is not registered: %w", err)
	}
	file, err := os.Open(artifactPath)
	if err != nil {
		return nil, fmt.Errorf("open built artifact: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("hash built artifact: %w", err)
	}
	checksum := hex.EncodeToString(hash.Sum(nil))
	if metadata.ArtifactURL == "" {
		metadata.ArtifactURL = "/downloads/" + metadata.AppKey + "/" + metadata.Platform
	}
	return s.UpsertAsset(ctx, Asset{
		BundleKey: metadata.BundleKey, AppKey: metadata.AppKey, Platform: metadata.Platform,
		VariantKey: metadata.VariantKey, ArtifactURL: metadata.ArtifactURL, ArtifactSource: "direct",
		ReleaseVersion: metadata.ReleaseVersion, Checksum: checksum, RequiresEntitlement: metadata.RequiresEntitlement,
		Metadata: map[string]interface{}{"artifact_path": artifactPath, "sha256": checksum},
	})
}
