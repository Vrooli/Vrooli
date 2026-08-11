package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratorsAreDeterministicAndSeeded(t *testing.T) {
	for _, preset := range ListPresets() {
		a, err := Render(Request{Preset: preset.ID, Width: 160, Height: 96, Seed: 7})
		require.NoError(t, err)
		b, err := Render(Request{Preset: preset.ID, Width: 160, Height: 96, Seed: 7})
		require.NoError(t, err)
		require.True(t, bytes.Equal(a.PNG, b.PNG), preset.ID)
		c, err := Render(Request{Preset: preset.ID, Width: 160, Height: 96, Seed: 8})
		require.NoError(t, err)
		require.NotEqual(t, a.SHA256, c.SHA256, preset.ID)
	}
}

func TestReservedRegionsAreFlatForBothConditioners(t *testing.T) {
	for _, conditioner := range []string{"depth", "edge"} {
		result, err := Render(Request{Preset: "horizon", Width: 160, Height: 96, Seed: 7, Conditioner: conditioner, Regions: []Region{{X: .1, Y: .2, Width: .3, Height: .25}, {X: .6, Y: .6, Width: .2, Height: .2}}})
		require.NoError(t, err)
		require.NotEmpty(t, result.PNG)
	}
}

func TestGoldenEvidence(t *testing.T) {
	golden := filepath.Join("testdata", "golden")
	for _, preset := range ListPresets() {
		result, err := Render(Request{Preset: preset.ID, Width: 320, Height: 180, Seed: 7})
		require.NoError(t, err)
		if os.Getenv("UPDATE_SCAFFOLD_EVIDENCE") == "1" {
			root := filepath.Join("..", "..", "..", "docs", "evidence", "phase-03")
			require.NoError(t, os.MkdirAll(root, 0o755))
			require.NoError(t, os.MkdirAll(golden, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(golden, preset.ID+".png"), result.PNG, 0o644))
			for _, seed := range []int64{7, 8} {
				seeded, e := Render(Request{Preset: preset.ID, Width: 320, Height: 180, Seed: seed})
				require.NoError(t, e)
				require.NoError(t, os.WriteFile(filepath.Join(root, preset.ID+"-seed-"+fmt.Sprint(seed)+".png"), seeded.PNG, 0o644))
			}
			continue
		}
		want, readErr := os.ReadFile(filepath.Join(golden, preset.ID+".png"))
		if readErr == nil {
			require.Equal(t, want, result.PNG, preset.ID)
		}
	}
}
