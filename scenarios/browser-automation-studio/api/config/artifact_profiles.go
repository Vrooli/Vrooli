// Package config provides artifact collection configuration and preset profiles.
//
// Artifact collection can be configured via:
//  1. Preset profiles: "full", "standard", "minimal", "debug", "none"
//  2. Custom configuration: Set profile to "custom" and toggle individual artifacts
//  3. Environment variables: Override default limits globally
//
// The configuration flows from ExecutionParameters.ArtifactConfig through the
// executor to the FileWriter, which uses it to decide what to persist.
package config

import (
	"strings"

	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

// ArtifactCollectionSettings is the resolved configuration used by the FileWriter.
// It merges proto config, profile defaults, and system defaults into a single struct.
type ArtifactCollectionSettings struct {
	// Artifact toggles
	CollectScreenshots   bool
	CollectDOMSnapshots  bool
	CollectConsoleLogs   bool
	CollectNetworkEvents bool
	CollectExtractedData bool
	CollectAssertions    bool
	CollectCursorTrails  bool
	CollectTelemetry     bool

	// ScreenshotPolicy governs whether the driver CAPTURES a step screenshot,
	// which is a separate question from whether CollectScreenshots persists one.
	// Capture is the expensive half — a full-viewport PNG per step, encoded and
	// shipped over HTTP — so a run that only needs pass/fail evidence can skip
	// it on steps that carry no visual meaning.
	ScreenshotPolicy basexecution.ScreenshotCapturePolicy

	// Size limits (in bytes)
	MaxScreenshotBytes     int
	MaxDOMSnapshotBytes    int
	MaxConsoleEntryBytes   int
	MaxNetworkPreviewBytes int
}

// ArtifactProfile represents a named preset configuration.
type ArtifactProfile struct {
	Name        string
	Description string
	Settings    ArtifactCollectionSettings
}

// Default size limits (can be overridden via config or proto).
const (
	DefaultMaxScreenshotBytes     = 4 * 1024 * 1024 // 4MB
	DefaultMaxDOMSnapshotBytes    = 512 * 1024      // 512KB
	DefaultMaxConsoleEntryBytes   = 16 * 1024       // 16KB
	DefaultMaxNetworkPreviewBytes = 64 * 1024       // 64KB

	// Debug profile uses larger limits for troubleshooting.
	DebugMaxScreenshotBytes     = 4 * 1024 * 1024 // 4MB
	DebugMaxDOMSnapshotBytes    = 2 * 1024 * 1024 // 2MB
	DebugMaxConsoleEntryBytes   = 64 * 1024       // 64KB
	DebugMaxNetworkPreviewBytes = 256 * 1024      // 256KB
)

// Profile names as constants for type safety.
const (
	ProfileFull     = "full"
	ProfileStandard = "standard"
	ProfileMinimal  = "minimal"
	ProfileDebug    = "debug"
	ProfileNone     = "none"
	ProfileCustom   = "custom"
	// ProfileValidation is for automated suites that need pass/fail evidence
	// rather than a replayable storyboard. It keeps assertions and extracted
	// data — those ARE the test results — but captures screenshots only where
	// they carry diagnostic value, which is most of the run's cost.
	ProfileValidation = "validation"
)

// artifactProfiles defines the preset configurations.
// These are used when ExecutionParameters.ArtifactConfig.Profile is set.
var artifactProfiles = map[string]ArtifactCollectionSettings{
	ProfileFull: {
		// Collect everything - backward compatible default
		ScreenshotPolicy:       basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS,
		CollectScreenshots:     true,
		CollectDOMSnapshots:    true,
		CollectConsoleLogs:     true,
		CollectNetworkEvents:   true,
		CollectExtractedData:   true,
		CollectAssertions:      true,
		CollectCursorTrails:    true,
		CollectTelemetry:       true,
		MaxScreenshotBytes:     DefaultMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DefaultMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DefaultMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DefaultMaxNetworkPreviewBytes,
	},
	ProfileStandard: {
		// Most useful artifacts, skip verbose debugging data
		ScreenshotPolicy:       basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS,
		CollectScreenshots:     true,
		CollectDOMSnapshots:    false,
		CollectConsoleLogs:     true,
		CollectNetworkEvents:   false,
		CollectExtractedData:   true,
		CollectAssertions:      true,
		CollectCursorTrails:    false,
		CollectTelemetry:       true,
		MaxScreenshotBytes:     DefaultMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DefaultMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DefaultMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DefaultMaxNetworkPreviewBytes,
	},
	ProfileMinimal: {
		// Just screenshots and assertions for quick validation
		ScreenshotPolicy:       basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS,
		CollectScreenshots:     true,
		CollectDOMSnapshots:    false,
		CollectConsoleLogs:     false,
		CollectNetworkEvents:   false,
		CollectExtractedData:   true,
		CollectAssertions:      true,
		CollectCursorTrails:    false,
		CollectTelemetry:       false,
		MaxScreenshotBytes:     DefaultMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DefaultMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DefaultMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DefaultMaxNetworkPreviewBytes,
	},
	ProfileDebug: {
		// Everything enabled with larger size limits for troubleshooting
		ScreenshotPolicy:       basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS,
		CollectScreenshots:     true,
		CollectDOMSnapshots:    true,
		CollectConsoleLogs:     true,
		CollectNetworkEvents:   true,
		CollectExtractedData:   true,
		CollectAssertions:      true,
		CollectCursorTrails:    true,
		CollectTelemetry:       true,
		MaxScreenshotBytes:     DebugMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DebugMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DebugMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DebugMaxNetworkPreviewBytes,
	},
	ProfileValidation: {
		// Automated suites: keep the artifacts that ARE the result, drop the
		// per-step imagery nobody reads. Screenshots still persist when they
		// happen, so failure frames survive for debugging.
		ScreenshotPolicy:       basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ON_FAILURE,
		CollectScreenshots:     true,
		CollectDOMSnapshots:    false,
		CollectConsoleLogs:     false,
		CollectNetworkEvents:   false,
		CollectExtractedData:   true,
		CollectAssertions:      true,
		CollectCursorTrails:    false,
		CollectTelemetry:       true,
		MaxScreenshotBytes:     DefaultMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DefaultMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DefaultMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DefaultMaxNetworkPreviewBytes,
	},
	ProfileNone: {
		// Disable all artifact collection (execution status only)
		ScreenshotPolicy:       basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_NEVER,
		CollectScreenshots:     false,
		CollectDOMSnapshots:    false,
		CollectConsoleLogs:     false,
		CollectNetworkEvents:   false,
		CollectExtractedData:   false,
		CollectAssertions:      false,
		CollectCursorTrails:    false,
		CollectTelemetry:       false,
		MaxScreenshotBytes:     DefaultMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DefaultMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DefaultMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DefaultMaxNetworkPreviewBytes,
	},
}

// GetArtifactProfiles returns a copy of all available profiles for documentation/UI.
func GetArtifactProfiles() []ArtifactProfile {
	return []ArtifactProfile{
		{
			Name:        ProfileFull,
			Description: "Collect all artifacts (screenshots, DOM, console, network, etc.)",
			Settings:    artifactProfiles[ProfileFull],
		},
		{
			Name:        ProfileStandard,
			Description: "Screenshots, console logs, extracted data, and assertions",
			Settings:    artifactProfiles[ProfileStandard],
		},
		{
			Name:        ProfileMinimal,
			Description: "Screenshots and assertions only (fastest execution)",
			Settings:    artifactProfiles[ProfileMinimal],
		},
		{
			Name:        ProfileDebug,
			Description: "All artifacts with larger size limits for troubleshooting",
			Settings:    artifactProfiles[ProfileDebug],
		},
		{
			Name:        ProfileValidation,
			Description: "Assertions and extracted data, screenshots only on failure (automated suites)",
			Settings:    artifactProfiles[ProfileValidation],
		},
		{
			Name:        ProfileNone,
			Description: "No artifacts collected (execution status only)",
			Settings:    artifactProfiles[ProfileNone],
		},
	}
}

// DefaultArtifactSettings returns the "collect everything" configuration (full
// profile). This is the hard fallback used by in-memory recorders that always
// want every artifact. The normal per-execution default is the operator-
// configured profile (BAS_ARTIFACT_DEFAULT_PROFILE, "standard" by default);
// resolve that via ResolveArtifactSettingsWithDefault, not this function.
func DefaultArtifactSettings() ArtifactCollectionSettings {
	return artifactProfiles[ProfileFull]
}

// DefaultArtifactSettingsForProfile returns the settings for the named profile,
// falling back to the standard profile when the name is empty or unknown.
func DefaultArtifactSettingsForProfile(profile string) ArtifactCollectionSettings {
	name := strings.ToLower(strings.TrimSpace(profile))
	if name == "" {
		name = ProfileStandard
	}
	if settings, ok := artifactProfiles[name]; ok {
		return settings
	}
	return artifactProfiles[ProfileStandard]
}

// ResolveArtifactSettingsWithDefault resolves a proto ArtifactCollectionConfig to
// concrete settings, applying defaultProfile when no per-execution profile is
// supplied. An explicit cfg.Profile always wins over defaultProfile; size-limit
// overrides on cfg are honored in both cases. defaultProfile is the operator-
// configured global default (BAS_ARTIFACT_DEFAULT_PROFILE).
func ResolveArtifactSettingsWithDefault(cfg *basexecution.ArtifactCollectionConfig, defaultProfile string) ArtifactCollectionSettings {
	if cfg == nil || strings.TrimSpace(cfg.GetProfile()) == "" {
		settings := DefaultArtifactSettingsForProfile(defaultProfile)
		// Allow per-execution size-limit overrides even when the profile is defaulted.
		return applyLimitOverrides(settings, cfg)
	}
	return ResolveArtifactSettings(cfg)
}

// ResolveArtifactSettings converts a proto ArtifactCollectionConfig to resolved settings.
// It handles profile lookup, custom configuration, and default fallbacks.
func ResolveArtifactSettings(cfg *basexecution.ArtifactCollectionConfig) ArtifactCollectionSettings {
	// No config provided - use full profile (backward compatible)
	if cfg == nil {
		return DefaultArtifactSettings()
	}

	// Get the profile name (default to "full")
	profileName := strings.ToLower(strings.TrimSpace(cfg.GetProfile()))
	if profileName == "" {
		profileName = ProfileFull
	}

	// Look up profile by name
	var settings ArtifactCollectionSettings
	if profileName == ProfileCustom {
		// Custom profile: use individual toggles from proto
		settings = buildCustomSettings(cfg)
	} else if profile, ok := artifactProfiles[profileName]; ok {
		// Known profile: use its settings
		settings = profile
	} else {
		// Unknown profile name: fall back to full
		settings = DefaultArtifactSettings()
	}

	// Apply size limit overrides from proto (if provided)
	settings = applyLimitOverrides(settings, cfg)

	return settings
}

// screenshotPolicyFor maps a persist toggle to the matching capture policy.
// Persisting and capturing are separate concerns, but "don't keep it" always
// implies "don't spend on it".
func screenshotPolicyFor(collect bool) basexecution.ScreenshotCapturePolicy {
	if !collect {
		return basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_NEVER
	}
	return basexecution.ScreenshotCapturePolicy_SCREENSHOT_CAPTURE_POLICY_ALWAYS
}

// buildCustomSettings creates settings from individual proto toggles.
// All toggles default to true for backward compatibility.
func buildCustomSettings(cfg *basexecution.ArtifactCollectionConfig) ArtifactCollectionSettings {
	return ArtifactCollectionSettings{
		// A custom profile that turns screenshots off should not pay to capture
		// them either; otherwise capture stays at the backward-compatible always.
		ScreenshotPolicy:       screenshotPolicyFor(getBoolWithDefault(cfg.CollectScreenshots, true)),
		CollectScreenshots:     getBoolWithDefault(cfg.CollectScreenshots, true),
		CollectDOMSnapshots:    getBoolWithDefault(cfg.CollectDomSnapshots, true),
		CollectConsoleLogs:     getBoolWithDefault(cfg.CollectConsoleLogs, true),
		CollectNetworkEvents:   getBoolWithDefault(cfg.CollectNetworkEvents, true),
		CollectExtractedData:   getBoolWithDefault(cfg.CollectExtractedData, true),
		CollectAssertions:      getBoolWithDefault(cfg.CollectAssertions, true),
		CollectCursorTrails:    getBoolWithDefault(cfg.CollectCursorTrails, true),
		CollectTelemetry:       getBoolWithDefault(cfg.CollectTelemetry, true),
		MaxScreenshotBytes:     DefaultMaxScreenshotBytes,
		MaxDOMSnapshotBytes:    DefaultMaxDOMSnapshotBytes,
		MaxConsoleEntryBytes:   DefaultMaxConsoleEntryBytes,
		MaxNetworkPreviewBytes: DefaultMaxNetworkPreviewBytes,
	}
}

// applyLimitOverrides applies size limit overrides from proto config.
func applyLimitOverrides(settings ArtifactCollectionSettings, cfg *basexecution.ArtifactCollectionConfig) ArtifactCollectionSettings {
	if cfg == nil {
		return settings
	}

	if cfg.MaxScreenshotBytes != nil && *cfg.MaxScreenshotBytes > 0 {
		settings.MaxScreenshotBytes = int(*cfg.MaxScreenshotBytes)
	}
	if cfg.MaxDomSnapshotBytes != nil && *cfg.MaxDomSnapshotBytes > 0 {
		settings.MaxDOMSnapshotBytes = int(*cfg.MaxDomSnapshotBytes)
	}
	if cfg.MaxConsoleEntryBytes != nil && *cfg.MaxConsoleEntryBytes > 0 {
		settings.MaxConsoleEntryBytes = int(*cfg.MaxConsoleEntryBytes)
	}
	if cfg.MaxNetworkPreviewBytes != nil && *cfg.MaxNetworkPreviewBytes > 0 {
		settings.MaxNetworkPreviewBytes = int(*cfg.MaxNetworkPreviewBytes)
	}

	return settings
}

// getBoolWithDefault returns the value of an optional bool pointer, or the default if nil.
func getBoolWithDefault(ptr *bool, defaultVal bool) bool {
	if ptr == nil {
		return defaultVal
	}
	return *ptr
}

// ValidateProfileName checks if a profile name is valid.
func ValidateProfileName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == ProfileCustom {
		return true
	}
	_, ok := artifactProfiles[name]
	return ok
}

// GetProfileNames returns all valid profile names.
func GetProfileNames() []string {
	return []string{ProfileFull, ProfileStandard, ProfileMinimal, ProfileDebug, ProfileNone, ProfileCustom}
}
