// Package config provides artifact collection configuration and preset profiles.
//
// Artifact collection can be configured via:
//   1. Preset profiles: "full", "standard", "minimal", "debug", "none"
//   2. Custom configuration: Set profile to "custom" and toggle individual artifacts
//   3. Environment variables: Override default limits globally
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
	CollectScreenshots    bool
	CollectDOMSnapshots   bool
	CollectConsoleLogs    bool
	CollectNetworkEvents  bool
	CollectExtractedData  bool
	CollectAssertions     bool
	CollectCursorTrails   bool
	CollectTelemetry      bool

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
	DefaultMaxScreenshotBytes     = 512 * 1024  // 512KB
	DefaultMaxDOMSnapshotBytes    = 512 * 1024  // 512KB
	DefaultMaxConsoleEntryBytes   = 16 * 1024   // 16KB
	DefaultMaxNetworkPreviewBytes = 64 * 1024   // 64KB

	// Debug profile uses larger limits for troubleshooting.
	DebugMaxScreenshotBytes     = 2 * 1024 * 1024  // 2MB
	DebugMaxDOMSnapshotBytes    = 2 * 1024 * 1024  // 2MB
	DebugMaxConsoleEntryBytes   = 64 * 1024        // 64KB
	DebugMaxNetworkPreviewBytes = 256 * 1024       // 256KB
)

// Profile names as constants for type safety.
const (
	ProfileFull     = "full"
	ProfileStandard = "standard"
	ProfileMinimal  = "minimal"
	ProfileDebug    = "debug"
	ProfileNone     = "none"
	ProfileCustom   = "custom"
)

// artifactProfiles defines the preset configurations.
// These are used when ExecutionParameters.ArtifactConfig.Profile is set.
var artifactProfiles = map[string]ArtifactCollectionSettings{
	ProfileFull: {
		// Collect everything - backward compatible default
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
	ProfileNone: {
		// Disable all artifact collection (execution status only)
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
			Name:        ProfileNone,
			Description: "No artifacts collected (execution status only)",
			Settings:    artifactProfiles[ProfileNone],
		},
	}
}

// DefaultArtifactSettings returns the default configuration (full profile).
func DefaultArtifactSettings() ArtifactCollectionSettings {
	return artifactProfiles[ProfileFull]
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

// buildCustomSettings creates settings from individual proto toggles.
// All toggles default to true for backward compatibility.
func buildCustomSettings(cfg *basexecution.ArtifactCollectionConfig) ArtifactCollectionSettings {
	return ArtifactCollectionSettings{
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
