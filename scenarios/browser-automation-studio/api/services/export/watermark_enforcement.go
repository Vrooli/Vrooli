package export

import (
	"strings"
)

const (
	// BuiltInAssetPrefix is the prefix for built-in asset IDs
	BuiltInAssetPrefix = "builtin:"
	// VrooliAscensionAssetID is the default Vrooli logo asset ID
	VrooliAscensionAssetID = "builtin:vrooli-ascension"
)

// WatermarkEnforcementResult contains the result of watermark enforcement
type WatermarkEnforcementResult struct {
	// WasEnforced indicates if any changes were made to enforce watermark requirements
	WasEnforced bool
	// OriginalEnabled was the original watermark.enabled value
	OriginalEnabled bool
	// OriginalAssetID was the original watermark.asset_id value
	OriginalAssetID string
}

// IsBuiltInAssetID checks if an asset ID refers to a built-in asset
func IsBuiltInAssetID(assetID string) bool {
	return strings.HasPrefix(assetID, BuiltInAssetPrefix)
}

// EnforceWatermarkRequirements enforces watermark requirements on a movie spec
// based on entitlement restrictions.
//
// If requiresWatermark is true:
// - Watermark.Enabled will be set to true
// - If no watermark asset is set, VrooliAscensionAssetID will be used
// - If a non-built-in asset is set, it will be replaced with VrooliAscensionAssetID
//
// Returns information about what was enforced.
func EnforceWatermarkRequirements(spec *ReplayMovieSpec, requiresWatermark bool) WatermarkEnforcementResult {
	result := WatermarkEnforcementResult{}

	if !requiresWatermark {
		return result
	}

	// Ensure watermark struct exists
	if spec.Watermark == nil {
		spec.Watermark = &ExportWatermark{
			Enabled:  true,
			AssetID:  VrooliAscensionAssetID,
			Position: "bottom-right",
			Size:     15,
			Opacity:  80,
			Margin:   16,
		}
		result.WasEnforced = true
		return result
	}

	result.OriginalEnabled = spec.Watermark.Enabled
	result.OriginalAssetID = spec.Watermark.AssetID

	// Enforce watermark enabled
	if !spec.Watermark.Enabled {
		spec.Watermark.Enabled = true
		result.WasEnforced = true
	}

	// Enforce built-in asset or set default
	if spec.Watermark.AssetID == "" {
		spec.Watermark.AssetID = VrooliAscensionAssetID
		result.WasEnforced = true
	} else if !IsBuiltInAssetID(spec.Watermark.AssetID) {
		spec.Watermark.AssetID = VrooliAscensionAssetID
		result.WasEnforced = true
	}

	// Ensure reasonable defaults if missing
	if spec.Watermark.Position == "" {
		spec.Watermark.Position = "bottom-right"
	}
	if spec.Watermark.Size <= 0 {
		spec.Watermark.Size = 15
	}
	if spec.Watermark.Opacity <= 0 {
		spec.Watermark.Opacity = 80
	}
	if spec.Watermark.Margin <= 0 {
		spec.Watermark.Margin = 16
	}

	return result
}

// ValidateWatermarkForExport checks if the watermark settings in a spec
// are valid for export given the entitlement restrictions.
//
// Returns an error if the settings violate entitlement requirements.
func ValidateWatermarkForExport(spec *ReplayMovieSpec, requiresWatermark bool) error {
	if !requiresWatermark {
		return nil // No restrictions
	}

	if spec.Watermark == nil || !spec.Watermark.Enabled {
		return &WatermarkValidationError{
			Code:    "WATERMARK_REQUIRED",
			Message: "Watermark is required for your subscription tier",
		}
	}

	if spec.Watermark.AssetID != "" && !IsBuiltInAssetID(spec.Watermark.AssetID) {
		return &WatermarkValidationError{
			Code:    "CUSTOM_LOGO_NOT_ALLOWED",
			Message: "Custom logos are not available on your subscription tier. Upgrade to use your own logo.",
		}
	}

	return nil
}

// WatermarkValidationError represents a watermark validation error
type WatermarkValidationError struct {
	Code    string
	Message string
}

func (e *WatermarkValidationError) Error() string {
	return e.Message
}
