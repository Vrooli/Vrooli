package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	require.NoError(t, c.Validate())
	// Accuracy-biased defaults — see ProfilePresets[ProfileBalanced].
	require.Equal(t, 2500, c.SegmentSilenceMs)
	require.Equal(t, 8192, c.OverlapBytes)
}

func TestConfigValidate_Errors(t *testing.T) {
	cases := []Config{
		{FlushIntervalMs: 0, MinDeltaBytes: 4096, OverlapBytes: 0},
		{FlushIntervalMs: 500, MinDeltaBytes: 100, OverlapBytes: 0},
		{FlushIntervalMs: 500, MinDeltaBytes: 4096, OverlapBytes: -1},
		{FlushIntervalMs: 500, MinDeltaBytes: 4096, OverlapBytes: 0, SegmentSilenceMs: 100},
		{FlushIntervalMs: 500, MinDeltaBytes: 4096, OverlapBytes: 0, WakeWordThreshold: 0.99},
	}
	for _, c := range cases {
		require.Error(t, c.Validate())
	}
}

func TestConfigPatch_Apply(t *testing.T) {
	b := DefaultConfig()
	fl, md, ov, ss := 800, 8192, 4096, 1200
	wwe, persist := true, true
	thr := 0.8
	p := ConfigPatch{
		FlushIntervalMs: &fl, MinDeltaBytes: &md, OverlapBytes: &ov,
		PersistentMode: &persist, WakeWordEnabled: &wwe,
		WakeWordThreshold: &thr, SegmentSilenceMs: &ss,
	}
	got := p.Apply(b)
	require.Equal(t, 800, got.FlushIntervalMs)
	require.Equal(t, 8192, got.MinDeltaBytes)
	require.True(t, got.PersistentMode)
	require.Equal(t, 0.8, got.WakeWordThreshold)
}

func TestLoadConfig_Missing(t *testing.T) {
	c, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	require.NoError(t, err)
	require.Equal(t, DefaultConfig(), c)
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	require.NoError(t, SaveConfig(p, DefaultConfig()))
	got, err := LoadConfig(p)
	require.NoError(t, err)
	require.Equal(t, DefaultConfig().FlushIntervalMs, got.FlushIntervalMs)
}

func TestLoadConfig_Invalid(t *testing.T) {
	// Invalid JSON triggers parse failure.
	p := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, SaveConfig(p, DefaultConfig()))
	// Overwrite with garbage.
	err := overwriteFile(p, "{not json")
	require.NoError(t, err)
	_, err = LoadConfig(p)
	require.Error(t, err)
}

func overwriteFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0o644)
}
