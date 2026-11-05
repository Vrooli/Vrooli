package strategy

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
)

func TestStrategyKinds(t *testing.T) {
	require.Equal(t, sttchain.StrategyBuffered, (&BufferedFallback{}).Kind())
	require.Equal(t, sttchain.StrategyOverlapAgree, (&OverlapAgree{}).Kind())
	require.Equal(t, sttchain.StrategyPassthrough, (&Passthrough{}).Kind())
	require.Equal(t, sttchain.StrategyVADSegment, (&VADSegmenter{}).Kind())
}
