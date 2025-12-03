package main

// ProfileDefaults defines the default values applied to newly created profiles.
// These defaults ensure profiles have sensible starting configurations.
type ProfileDefaults struct {
	// Tiers specifies which deployment tiers are enabled by default.
	// Default: [2] (desktop tier only)
	Tiers []int

	// Swaps contains default resource swaps applied to profiles.
	// Default: empty map (no swaps)
	Swaps map[string]interface{}

	// Secrets contains default secret configurations.
	// Default: empty map (no pre-configured secrets)
	Secrets map[string]interface{}

	// Settings contains default profile settings.
	// Default: empty map (use system defaults)
	Settings map[string]interface{}
}

// DefaultProfileDefaults returns the standard defaults for new profiles.
//
// Decision rationale:
// - Desktop tier (2) is the default because it's the primary bundling target
// - Empty swaps/secrets/settings allow scenario-specific configuration
func DefaultProfileDefaults() ProfileDefaults {
	return ProfileDefaults{
		Tiers:    []int{TierDesktop}, // Default to desktop tier
		Swaps:    map[string]interface{}{},
		Secrets:  map[string]interface{}{},
		Settings: map[string]interface{}{},
	}
}

// ApplyProfileDefaults fills in missing fields on a profile with default values.
// Only nil fields are replaced; explicit empty values are preserved.
func ApplyProfileDefaults(profile *Profile) {
	defaults := DefaultProfileDefaults()

	if profile.Tiers == nil {
		profile.Tiers = defaults.Tiers
	}
	if profile.Swaps == nil {
		profile.Swaps = defaults.Swaps
	}
	if profile.Secrets == nil {
		profile.Secrets = defaults.Secrets
	}
	if profile.Settings == nil {
		profile.Settings = defaults.Settings
	}
}

// ShouldApplyDefault decides whether a field should receive a default value.
// Returns true if the value is nil (unset), false if explicitly set (even if empty).
func ShouldApplyDefault(value interface{}) bool {
	return value == nil
}
