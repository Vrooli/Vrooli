package bundleruntime

import (
	"crypto/sha256"
	"fmt"
	goruntime "runtime"
	"strings"

	"scenario-to-desktop-runtime/api"
	"scenario-to-desktop-runtime/manifest"
)

// sha256Sum computes the SHA256 hash of data.
func sha256Sum(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// ErrorSummary returns a human-readable summary of validation errors.
func errorSummary(r *api.BundleValidationResult) string {
	if r.Valid {
		return ""
	}
	var parts []string
	if len(r.MissingBinaries) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing binaries", len(r.MissingBinaries)))
	}
	if len(r.MissingAssets) > 0 {
		parts = append(parts, fmt.Sprintf("%d missing assets", len(r.MissingAssets)))
	}
	if len(r.InvalidChecksums) > 0 {
		parts = append(parts, fmt.Sprintf("%d checksum failures", len(r.InvalidChecksums)))
	}
	for _, e := range r.Errors {
		parts = append(parts, e.Message)
	}
	return strings.Join(parts, "; ")
}

// ValidateBundle performs pre-flight validation of the bundle manifest.
// This checks that all binaries and assets exist BEFORE starting any services.
// Returns a detailed validation result with specific failures.
func (s *Supervisor) ValidateBundle() *api.BundleValidationResult {
	result := &api.BundleValidationResult{Valid: true}
	targetOS := goruntime.GOOS
	targetArch := goruntime.GOARCH

	// Validate manifest structure.
	if err := s.opts.Manifest.Validate(targetOS, targetArch); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "manifest_invalid",
			Message: err.Error(),
		})
		return result
	}

	// Check all service binaries exist.
	for _, svc := range s.opts.Manifest.Services {
		s.validateServiceBinaries(svc, targetOS, targetArch, result)
		s.validateServiceAssets(svc, result)
	}

	return result
}

// validateServiceBinaries checks that the binary for a service exists.
func (s *Supervisor) validateServiceBinaries(svc manifest.Service, targetOS, targetArch string, result *api.BundleValidationResult) {
	bin, ok := s.opts.Manifest.ResolveBinary(svc)
	if !ok {
		result.Valid = false
		platform := manifest.PlatformKey(targetOS, targetArch)
		result.MissingBinaries = append(result.MissingBinaries, api.MissingBinary{
			ServiceID: svc.ID,
			Platform:  platform,
			Path:      "<no binary defined>",
		})
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "binary_not_defined",
			Service: svc.ID,
			Message: fmt.Sprintf("no binary defined for service %s on platform %s", svc.ID, platform),
		})
		return
	}

	// Resolve the full path.
	binPath := manifest.ResolvePath(s.opts.BundlePath, bin.Path)

	// Check if file exists.
	info, err := s.fs.Stat(binPath)
	if err != nil {
		result.Valid = false
		platform := manifest.PlatformKey(targetOS, targetArch)
		result.MissingBinaries = append(result.MissingBinaries, api.MissingBinary{
			ServiceID: svc.ID,
			Platform:  platform,
			Path:      bin.Path,
		})
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "binary_missing",
			Service: svc.ID,
			Path:    bin.Path,
			Message: fmt.Sprintf("binary not found for service %s: %s", svc.ID, bin.Path),
		})
		return
	}

	// Verify it's not a directory.
	if info.IsDir() {
		result.Valid = false
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "binary_is_directory",
			Service: svc.ID,
			Path:    bin.Path,
			Message: fmt.Sprintf("binary path is a directory for service %s: %s", svc.ID, bin.Path),
		})
	}
}

// validateServiceAssets checks that all assets for a service exist and have valid checksums.
func (s *Supervisor) validateServiceAssets(svc manifest.Service, result *api.BundleValidationResult) {
	for _, asset := range svc.Assets {
		assetPath := manifest.ResolvePath(s.opts.BundlePath, asset.Path)

		// Check existence.
		info, err := s.fs.Stat(assetPath)
		if err != nil {
			result.Valid = false
			result.MissingAssets = append(result.MissingAssets, api.MissingAsset{
				ServiceID: svc.ID,
				Path:      asset.Path,
			})
			result.Errors = append(result.Errors, api.BundleError{
				Code:    "asset_missing",
				Service: svc.ID,
				Path:    asset.Path,
				Message: fmt.Sprintf("asset not found for service %s: %s", svc.ID, asset.Path),
			})
			continue
		}

		// Verify it's a file.
		if info.IsDir() {
			result.Valid = false
			result.Errors = append(result.Errors, api.BundleError{
				Code:    "asset_is_directory",
				Service: svc.ID,
				Path:    asset.Path,
				Message: fmt.Sprintf("asset path is a directory for service %s: %s", svc.ID, asset.Path),
			})
			continue
		}

		// Validate checksum if specified.
		if asset.SHA256 != "" {
			if err := s.validateAssetChecksum(svc, asset, assetPath, result); err != nil {
				// Error already recorded in result.
				continue
			}
		}

		// Validate size budget if specified.
		if asset.SizeBytes > 0 {
			s.validateAssetSize(svc, asset, info.Size(), result)
		}
	}
}

// validateAssetChecksum verifies the SHA256 checksum of an asset.
func (s *Supervisor) validateAssetChecksum(svc manifest.Service, asset manifest.Asset, path string, result *api.BundleValidationResult) error {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "asset_unreadable",
			Service: svc.ID,
			Path:    asset.Path,
			Message: fmt.Sprintf("cannot read asset for service %s: %s", svc.ID, err),
		})
		return err
	}

	actual := fmt.Sprintf("%x", sha256Sum(data))
	if !strings.EqualFold(actual, asset.SHA256) {
		result.Valid = false
		result.InvalidChecksums = append(result.InvalidChecksums, api.InvalidChecksum{
			ServiceID: svc.ID,
			Path:      asset.Path,
			Expected:  asset.SHA256,
			Actual:    actual,
		})
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "checksum_mismatch",
			Service: svc.ID,
			Path:    asset.Path,
			Message: fmt.Sprintf("checksum mismatch for service %s asset %s", svc.ID, asset.Path),
		})
		return fmt.Errorf("checksum mismatch")
	}

	return nil
}

// validateAssetSize checks if an asset's size is within acceptable bounds.
func (s *Supervisor) validateAssetSize(svc manifest.Service, asset manifest.Asset, actual int64, result *api.BundleValidationResult) {
	expected := asset.SizeBytes
	slack := sizeBudgetSlack(expected)

	// Check for oversized assets.
	if actual > expected+slack {
		result.Valid = false
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "asset_size_exceeded",
			Service: svc.ID,
			Path:    asset.Path,
			Message: fmt.Sprintf("asset %s exceeds size budget: got %d bytes, budget %d (+%d slack)", asset.Path, actual, expected, slack),
		})
		return
	}

	// Check for suspiciously small assets.
	if actual < expected/2 {
		result.Valid = false
		result.Errors = append(result.Errors, api.BundleError{
			Code:    "asset_size_suspicious",
			Service: svc.ID,
			Path:    asset.Path,
			Message: fmt.Sprintf("asset %s is smaller than expected (%d bytes vs %d)", asset.Path, actual, expected),
		})
		return
	}

	// Warn for slightly oversized assets (within slack).
	if actual > expected {
		result.Warnings = append(result.Warnings, api.BundleWarning{
			Code:    "asset_size_warning",
			Service: svc.ID,
			Path:    asset.Path,
			Message: fmt.Sprintf("asset %s is %d bytes larger than expected", asset.Path, actual-expected),
		})
	}
}

// sizeBudgetSlack calculates the allowed size overage for an asset.
// Returns 5% of expected size or 1MB minimum.
func sizeBudgetSlack(expected int64) int64 {
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
