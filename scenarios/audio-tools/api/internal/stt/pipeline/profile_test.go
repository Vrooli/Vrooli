package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfilePresets_AllValid(t *testing.T) {
	base := DefaultConfig()
	for name := range ProfilePresets() {
		got, err := ApplyProfile(name, base)
		require.NoError(t, err, "profile %s", name)
		require.NoError(t, got.Validate(), "profile %s validates", name)
	}
}

func TestApplyProfile_FieldScoped(t *testing.T) {
	base := DefaultConfig()
	// Mutate fields that profiles should NOT touch.
	base.FlushIntervalMs = 750
	base.WakeWordThreshold = 0.42
	base.PersistentMode = true

	got, err := ApplyProfile(ProfileLatency, base)
	require.NoError(t, err)
	require.Equal(t, 750, got.FlushIntervalMs)
	require.InDelta(t, 0.42, got.WakeWordThreshold, 0.0001)
	require.True(t, got.PersistentMode)
	require.Equal(t, 1200, got.SegmentSilenceMs)
	require.Equal(t, 2048, got.OverlapBytes)
}

func TestApplyProfile_Idempotent(t *testing.T) {
	base := DefaultConfig()
	once, err := ApplyProfile(ProfileAccuracy, base)
	require.NoError(t, err)
	twice, err := ApplyProfile(ProfileAccuracy, once)
	require.NoError(t, err)
	require.Equal(t, once, twice)
}

func TestApplyProfile_UnknownReturnsError(t *testing.T) {
	base := DefaultConfig()
	got, err := ApplyProfile(Profile("nonsense"), base)
	require.Error(t, err)
	require.Equal(t, base, got)
}

func TestDefaultConfig_MatchesBalancedPreset(t *testing.T) {
	d := DefaultConfig()
	preset := ProfilePresets()[ProfileBalanced]
	require.Equal(t, preset.SegmentSilenceMs, d.SegmentSilenceMs)
	require.Equal(t, preset.OverlapBytes, d.OverlapBytes)
}
