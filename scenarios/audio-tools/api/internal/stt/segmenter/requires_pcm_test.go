package segmenter

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/sttengine"
)

// TestRequiresPCM_ManifestDriven proves the PCM-normalization decision is
// manifest-driven for Passthrough: a native-streaming engine that declares
// requires.pcm16kMono (Kyutai) gets its inbound chunks normalized to canonical
// PCM before the Passthrough strategy forwards them, while VAD/Overlap always
// need PCM and BufferedFallback never does.
func TestRequiresPCM_ManifestDriven(t *testing.T) {
	reg := sttengine.Default() // whisper-local + kyutai
	s := &Segmenter{deps: Deps{Registry: reg}}

	// VAD/Overlap always require client-side PCM regardless of engine.
	require.True(t, s.requiresPCM(sttchain.StrategyVADSegment, "whisper-local"))
	require.True(t, s.requiresPCM(sttchain.StrategyOverlapAgree, "whisper-local"))

	// Passthrough to Kyutai (requires.pcm16kMono=true) needs normalization.
	require.True(t, s.requiresPCM(sttchain.StrategyPassthrough, "kyutai"))

	// Passthrough to an engine the manifest doesn't know (a BYOK vendor that
	// decodes for itself) does NOT get normalized.
	require.False(t, s.requiresPCM(sttchain.StrategyPassthrough, "deepgram-byok"))

	// BufferedFallback never pre-normalizes (Whisper decodes the whole file).
	require.False(t, s.requiresPCM(sttchain.StrategyBuffered, "whisper-local"))
}

// TestRequiresPCM_NoRegistry falls back to strategy-only behavior.
func TestRequiresPCM_NoRegistry(t *testing.T) {
	s := &Segmenter{deps: Deps{}}
	require.True(t, s.requiresPCM(sttchain.StrategyVADSegment, "whisper-local"))
	require.False(t, s.requiresPCM(sttchain.StrategyPassthrough, "kyutai"))
}
