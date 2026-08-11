package transcript

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestAllP0StrategiesHaveProtocolTranscript(t *testing.T) {
	f, err := Load(filepath.Join("transcripts.json"))
	require.NoError(t, err)
	require.Len(t, f.Strategies, 5)
	for _, e := range f.Strategies {
		require.NotEmpty(t, e.StrategyID)
		require.NotEmpty(t, e.Observe)
		require.NotEmpty(t, e.Actuate)
		require.NotEmpty(t, e.UnavailableReason)
	}
}
