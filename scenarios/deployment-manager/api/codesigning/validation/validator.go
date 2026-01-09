// Package validation provides signing configuration validation for the codesigning package.
//
// The package separates structural validation (checking config values are correct)
// from prerequisite validation (checking tools and certificates are available).
package validation

import (
	"context"

	"deployment-manager/codesigning"
)

// Validator validates signing configurations structurally.
// It does not perform I/O operations - for that, use PrerequisiteChecker.
type Validator interface {
	// ValidateConfig checks structural validity of a SigningConfig.
	// Returns validation results with errors for any invalid configuration.
	ValidateConfig(config *codesigning.SigningConfig) *codesigning.ValidationResult

	// ValidateForPlatform checks config is valid for a specific target platform.
	// This is stricter than ValidateConfig - it ensures the platform has all required fields.
	ValidateForPlatform(config *codesigning.SigningConfig, platform string) *codesigning.ValidationResult
}

// validator implements the Validator interface.
type validator struct {
	rules []ValidationRule
}

// ValidatorOption configures a validator.
type ValidatorOption func(*validator)

// WithRules adds custom validation rules.
func WithRules(rules ...ValidationRule) ValidatorOption {
	return func(v *validator) {
		v.rules = append(v.rules, rules...)
	}
}

// NewValidator creates a validator with the given options.
// By default, it includes all standard validation rules.
func NewValidator(opts ...ValidatorOption) Validator {
	v := &validator{
		rules: DefaultRules(),
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// ValidateConfig checks structural validity of a SigningConfig.
func (v *validator) ValidateConfig(config *codesigning.SigningConfig) *codesigning.ValidationResult {
	result := codesigning.NewValidationResult()

	// If signing is disabled, no validation needed
	if config == nil || !config.Enabled {
		return result
	}

	// Run all validation rules
	for _, rule := range v.rules {
		rule.Validate(config, result)
	}

	// Initialize platform validation entries
	v.initializePlatformValidation(config, result)

	return result
}

// ValidateForPlatform checks config is valid for a specific target platform.
func (v *validator) ValidateForPlatform(config *codesigning.SigningConfig, platform string) *codesigning.ValidationResult {
	result := codesigning.NewValidationResult()

	// Validate platform name
	if !codesigning.IsValidPlatform(platform) {
		result.AddError(codesigning.ValidationError{
			Code:        "INVALID_PLATFORM",
			Platform:    platform,
			Message:     "Unknown platform: " + platform,
			Remediation: "Valid platforms are: windows, macos, linux",
		})
		return result
	}

	// If signing is disabled, no validation needed
	if config == nil || !config.Enabled {
		return result
	}

	// Check that the requested platform is configured
	platformConfig := v.getPlatformConfig(config, platform)
	if platformConfig == nil {
		result.AddError(codesigning.ValidationError{
			Code:        "PLATFORM_NOT_CONFIGURED",
			Platform:    platform,
			Message:     "No signing configuration for platform: " + platform,
			Remediation: "Add " + platform + " signing configuration or disable signing",
		})
		return result
	}

	// Run all validation rules
	for _, rule := range v.rules {
		rule.Validate(config, result)
	}

	// Filter to only include errors/warnings for the requested platform
	result = v.filterResultsForPlatform(result, platform)

	return result
}

// initializePlatformValidation creates platform entries in the validation result.
func (v *validator) initializePlatformValidation(config *codesigning.SigningConfig, result *codesigning.ValidationResult) {
	if config.Windows != nil {
		result.Platforms[codesigning.PlatformWindows] = codesigning.PlatformValidation{
			Configured: true,
			Errors:     []string{},
			Warnings:   []string{},
		}
	}
	if config.MacOS != nil {
		result.Platforms[codesigning.PlatformMacOS] = codesigning.PlatformValidation{
			Configured: true,
			Errors:     []string{},
			Warnings:   []string{},
		}
	}
	if config.Linux != nil {
		result.Platforms[codesigning.PlatformLinux] = codesigning.PlatformValidation{
			Configured: true,
			Errors:     []string{},
			Warnings:   []string{},
		}
	}
}

// getPlatformConfig returns the platform-specific config or nil.
// NOTE: We explicitly check for nil before returning to avoid the Go interface nil trap.
// Returning a nil pointer as interface{} results in a non-nil interface (with nil value).
func (v *validator) getPlatformConfig(config *codesigning.SigningConfig, platform string) interface{} {
	switch platform {
	case codesigning.PlatformWindows:
		if config.Windows == nil {
			return nil
		}
		return config.Windows
	case codesigning.PlatformMacOS:
		if config.MacOS == nil {
			return nil
		}
		return config.MacOS
	case codesigning.PlatformLinux:
		if config.Linux == nil {
			return nil
		}
		return config.Linux
	default:
		return nil
	}
}

// filterResultsForPlatform filters validation results to only include the specified platform.
func (v *validator) filterResultsForPlatform(result *codesigning.ValidationResult, platform string) *codesigning.ValidationResult {
	filtered := codesigning.NewValidationResult()

	// Copy platform validation for the requested platform
	if pv, ok := result.Platforms[platform]; ok {
		filtered.Platforms[platform] = pv
	}

	// Filter errors
	for _, err := range result.Errors {
		if err.Platform == platform || err.Platform == "" {
			filtered.AddError(err)
		}
	}

	// Filter warnings
	for _, warn := range result.Warnings {
		if warn.Platform == platform || warn.Platform == "" {
			filtered.AddWarning(warn)
		}
	}

	return filtered
}

// PrerequisiteChecker verifies external signing prerequisites.
// Unlike Validator, this performs I/O operations to check tools and certificates.
type PrerequisiteChecker interface {
	// CheckPrerequisites validates tools and certificates are available.
	// This performs actual file system and command execution checks.
	CheckPrerequisites(ctx context.Context, config *codesigning.SigningConfig) *codesigning.ValidationResult

	// CheckPlatformPrerequisites checks prerequisites for a specific platform.
	CheckPlatformPrerequisites(ctx context.Context, config *codesigning.SigningConfig, platform string) *codesigning.ValidationResult

	// DetectTools returns available signing tools on the current system.
	DetectTools(ctx context.Context) ([]codesigning.ToolDetectionResult, error)
}
