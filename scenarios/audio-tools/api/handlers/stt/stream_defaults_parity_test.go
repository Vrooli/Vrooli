package stt

import (
	"testing"

	sttpkg "audio-tools/internal/stt"
)

// clientVADFallbackMs mirrors VAD_FALLBACK_SILENCE_TIMEOUT_MS in the
// audio-integration UI package
// (scenarios/*/ui/src/audio-integration/hooks/voice/vad.ts). The browser uses
// that constant as the auto-stop silence timeout until it has hydrated the
// real config from this server. If the two drift, the mic-button ring (and the
// client RMS-VAD fallback) fire at a different time than the server actually
// cuts the segment — the "button shows off while transcription keeps coming"
// desync. Keep this literal equal to vad.ts; the matching vad.test.ts assertion
// guards the other direction.
const clientVADFallbackMs = 1200

// TestVADSilenceDefaultsSingleSource locks every server-side VAD-silence
// default to the single DefaultVADSilenceMs source of truth, and to the client
// fallback. This converts the fragile "keep these aligned" comment on
// stt.Defaults() into a build-time guarantee.
func TestVADSilenceDefaultsSingleSource(t *testing.T) {
	const want = sttpkg.DefaultVADSilenceMs

	// The strategy default consumed by the segmenter.
	if got := sttpkg.Defaults().VADSilenceMs; got != want {
		t.Errorf("stt.Defaults().VADSilenceMs = %d, want DefaultVADSilenceMs (%d)", got, want)
	}

	// The persisted-doc default shipped to the browser via GetStreamConfig.
	if got := defaultStreamCfg().VadSilenceMs; int(got) != want {
		t.Errorf("defaultStreamCfg().VadSilenceMs = %d, want DefaultVADSilenceMs (%d) — "+
			"a divergence here makes the mic ring disagree with the server cut", got, want)
	}

	// The client-side fallback (vad.ts). Mirrored here so a server change that
	// is not reflected in the UI package trips this test.
	if want != clientVADFallbackMs {
		t.Errorf("DefaultVADSilenceMs (%d) != client VAD_FALLBACK_SILENCE_TIMEOUT_MS (%d) — "+
			"update vad.ts in every audio-integration copy to match", want, clientVADFallbackMs)
	}
}
