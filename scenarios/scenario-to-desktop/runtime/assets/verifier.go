// Package assets provides asset verification for the bundle runtime.
package assets

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/telemetry"
)

// Verifier verifies bundle assets.
type Verifier struct {
	BundlePath string
	FS         infra.FileSystem
	Telemetry  telemetry.Recorder
}

// NewVerifier creates a new asset verifier.
func NewVerifier(bundlePath string, fs infra.FileSystem, telem telemetry.Recorder) *Verifier {
	return &Verifier{
		BundlePath: bundlePath,
		FS:         fs,
		Telemetry:  telem,
	}
}

// EnsureAssets verifies all required assets for a service exist and are valid.
// It checks file existence, size budgets, and SHA256 checksums.
func (v *Verifier) EnsureAssets(svc manifest.Service) error {
	for _, asset := range svc.Assets {
		if err := v.verifyAsset(svc, asset); err != nil {
			return err
		}
	}
	return nil
}

// verifyAsset checks a single asset for existence, size, and checksum.
func (v *Verifier) verifyAsset(svc manifest.Service, asset manifest.Asset) error {
	path := manifest.ResolvePath(v.BundlePath, asset.Path)

	// Check existence.
	info, err := v.FS.Stat(path)
	if err != nil {
		_ = v.Telemetry.Record("asset_missing", map[string]interface{}{
			"service_id": svc.ID,
			"path":       asset.Path,
		})
		return fmt.Errorf("asset missing for service %s: %s", svc.ID, asset.Path)
	}

	// Verify it's a file, not a directory.
	if info.IsDir() {
		_ = v.Telemetry.Record("asset_missing", map[string]interface{}{
			"service_id": svc.ID,
			"path":       asset.Path,
			"reason":     "expected file",
		})
		return fmt.Errorf("asset path is a directory: %s", asset.Path)
	}

	// Check size budget if specified.
	if asset.SizeBytes > 0 {
		if err := v.checkSizeBudget(svc, asset, info.Size()); err != nil {
			return err
		}
	}

	// Verify checksum if specified.
	if asset.SHA256 != "" {
		if err := v.verifyChecksum(svc, asset, path); err != nil {
			return err
		}
	}

	return nil
}

// verifyChecksum computes and compares the SHA256 hash of an asset.
func (v *Verifier) verifyChecksum(svc manifest.Service, asset manifest.Asset, path string) error {
	if strings.EqualFold(asset.SHA256, "pending") {
		// Placeholder checksum from generator; skip strict validation.
		return nil
	}

	data, err := v.FS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read asset %s: %w", asset.Path, err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	if !strings.EqualFold(sum, asset.SHA256) {
		_ = v.Telemetry.Record("asset_checksum_mismatch", map[string]interface{}{
			"service_id": svc.ID,
			"path":       asset.Path,
			"expected":   asset.SHA256,
			"actual":     sum,
		})
		return fmt.Errorf("asset %s checksum mismatch", asset.Path)
	}
	return nil
}

// checkSizeBudget verifies an asset's size is within acceptable bounds.
// Assets can exceed their expected size by up to 5% (minimum 1MB slack).
// Assets smaller than half the expected size are considered suspicious.
func (v *Verifier) checkSizeBudget(svc manifest.Service, asset manifest.Asset, actual int64) error {
	expected := asset.SizeBytes
	slack := SizeBudgetSlack(expected)

	// Check for oversized assets.
	if actual > expected+slack {
		_ = v.Telemetry.Record("asset_size_exceeded", map[string]interface{}{
			"service_id":     svc.ID,
			"path":           asset.Path,
			"expected_bytes": expected,
			"actual_bytes":   actual,
			"slack_bytes":    slack,
		})
		return fmt.Errorf("asset %s exceeds size budget: got %d bytes, budget %d (+%d slack)", asset.Path, actual, expected, slack)
	}

	// Check for suspiciously small assets.
	if actual < expected/2 {
		_ = v.Telemetry.Record("asset_size_suspicious", map[string]interface{}{
			"service_id":     svc.ID,
			"path":           asset.Path,
			"expected_bytes": expected,
			"actual_bytes":   actual,
		})
		return fmt.Errorf("asset %s is smaller than expected (%d bytes vs %d)", asset.Path, actual, expected)
	}

	// Log warning for slightly oversized assets (within slack).
	if actual > expected {
		_ = v.Telemetry.Record("asset_size_warning", map[string]interface{}{
			"service_id":     svc.ID,
			"path":           asset.Path,
			"expected_bytes": expected,
			"actual_bytes":   actual,
		})
	}

	return nil
}

// SizeBudgetSlack calculates the allowed size overage for an asset.
// Returns 5% of expected size or 1MB minimum.
func SizeBudgetSlack(expected int64) int64 {
	if expected <= 0 {
		return 0
	}
	percent := expected / 20 // 5%
	const minSlack = int64(1 * 1024 * 1024)
	if percent < minSlack {
		return minSlack
	}
	return percent
}
