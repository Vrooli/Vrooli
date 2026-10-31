package pipeline

import "fmt"

// Profile is a curated preset for the accuracy/latency tradeoff on the
// VAD-batch streaming path. Operators pick a profile to set
// SegmentSilenceMs and OverlapBytes in one shot; raw knobs remain
// editable for fine tuning.
type Profile string

const (
	ProfileLatency  Profile = "latency"
	ProfileBalanced Profile = "balanced"
	ProfileAccuracy Profile = "accuracy"
)

// ProfilePreset is the field-scoped overlay a profile applies to a base
// Config. Only the listed fields are touched.
type ProfilePreset struct {
	SegmentSilenceMs int
	OverlapBytes     int
}

// ProfilePresets returns the canonical preset map. Balanced matches
// DefaultConfig() so selecting "balanced" is a no-op against a fresh
// install.
func ProfilePresets() map[Profile]ProfilePreset {
	return map[Profile]ProfilePreset{
		ProfileLatency:  {SegmentSilenceMs: 1200, OverlapBytes: 2048},
		ProfileBalanced: {SegmentSilenceMs: 2500, OverlapBytes: 8192},
		ProfileAccuracy: {SegmentSilenceMs: 3000, OverlapBytes: 16384},
	}
}

// ApplyProfile overlays the profile-controlled fields onto base. Unknown
// profile names return an error and leave base untouched. Idempotent.
func ApplyProfile(p Profile, base Config) (Config, error) {
	presets := ProfilePresets()
	preset, ok := presets[p]
	if !ok {
		return base, fmt.Errorf("unknown profile %q", p)
	}
	base.SegmentSilenceMs = preset.SegmentSilenceMs
	base.OverlapBytes = preset.OverlapBytes
	return base, nil
}
