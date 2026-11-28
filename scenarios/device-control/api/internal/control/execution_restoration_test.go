package control

import (
	"testing"

	"device-control/strategy"
	"github.com/stretchr/testify/require"
)

func TestMergeRestorationSnapshotCarriesKnownScreenState(t *testing.T) {
	for _, screen := range []string{"on", "off"} {
		t.Run(screen, func(t *testing.T) {
			merged := mergeRestorationSnapshot(strategy.DeviceState{}, strategy.DeviceState{ScreenState: screen})
			require.Equal(t, screen, merged.ScreenState)
		})
	}
	merged := mergeRestorationSnapshot(strategy.DeviceState{}, strategy.DeviceState{ScreenState: "unknown"})
	require.Empty(t, merged.ScreenState, "unknown observations must not cause an invented screen transition")
}
