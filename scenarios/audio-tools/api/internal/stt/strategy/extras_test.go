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

func TestFindWebMInitEnd(t *testing.T) {
	cluster := []byte{0x1F, 0x43, 0xB6, 0x75}
	require.Equal(t, 0, FindWebMInitEnd([]byte("no cluster here")))
	buf := append([]byte("header"), cluster...)
	require.Equal(t, 6, FindWebMInitEnd(buf))
}
