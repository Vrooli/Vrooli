package stt_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt"
	"audio-tools/internal/sttengine"
)

const whisperOnlyManifest = `{
  "engines": [{
    "id": "whisper-local", "displayName": "Whisper", "kind": "local_resource", "resource": "whisper",
    "provides": {"nativeStreaming": false, "builtinVad": true, "confidenceSignals": ["no_speech_prob","avg_logprob"], "wordTimestamps": true},
    "requires": {"pcm16kMono": true},
    "strategies": ["vad_segment", "overlap_agree", "buffered_fallback"]
  }],
  "speakerIsolation": {"active": "verification", "methods": {"verification": {"backendResource": "speaker-verification"}}}
}`

const streamingEngineManifest = `{
  "engines": [{
    "id": "kyutai", "displayName": "Kyutai", "kind": "local_resource", "resource": "kyutai-stt",
    "provides": {"nativeStreaming": true, "builtinVad": true, "confidenceSignals": [], "wordTimestamps": true},
    "requires": {"pcm16kMono": true},
    "strategies": ["passthrough"]
  }],
  "speakerIsolation": {"active": "verification", "methods": {"verification": {"backendResource": "speaker-verification"}}}
}`

func mustRegistry(t *testing.T, raw string) *sttengine.Registry {
	t.Helper()
	r, err := sttengine.Load([]byte(raw))
	require.NoError(t, err)
	return r
}

// TestSelectorManifestDrivenEligibility proves the Local tier's eligible
// strategies come from the engine manifest, NOT the provider's hardcoded
// ProviderTraits.Strategies (here deliberately empty). The manifest whitelist
// gates even capabilities the provider otherwise advertises (Stream=true +
// passthrough excluded by the whisper manifest -> incompatible).
func TestSelectorManifestDrivenEligibility(t *testing.T) {
	cases := []struct {
		name     string
		manifest string
		engineID string
		traits   sttchain.ProviderTraits
		pref     stt.StrategyPreference
		wantKind sttchain.StrategyKind
		wantErr  error
	}{
		{
			// Auto resolves to overlap_agree when the manifest allows it.
			// The growing-buffer LocalAgreement-N + VAD-anchored trigger
			// gives incremental commits mid-utterance.
			name:     "whisper auto -> overlap_agree (resolved 2026-05-28)",
			manifest: whisperOnlyManifest, engineID: "whisper-local",
			traits:   sttchain.ProviderTraits{Batch: true}, // empty Strategies: manifest is authority
			pref:     stt.PreferenceAuto,
			wantKind: sttchain.StrategyOverlapAgree,
		},
		{
			name:     "whisper overlap stays eligible",
			manifest: whisperOnlyManifest, engineID: "whisper-local",
			traits:   sttchain.ProviderTraits{Batch: true},
			pref:     stt.PreferenceOverlap,
			wantKind: sttchain.StrategyOverlapAgree,
		},
		{
			name:     "manifest excludes passthrough even when provider can stream",
			manifest: whisperOnlyManifest, engineID: "whisper-local",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true},
			pref:     stt.PreferencePassthrough,
			wantKind: sttchain.StrategyBuffered,
			wantErr:  stt.ErrIncompatibleStrategyProvider,
		},
		{
			name:     "native-streaming engine auto -> passthrough",
			manifest: streamingEngineManifest, engineID: "kyutai",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true},
			pref:     stt.PreferenceAuto,
			wantKind: sttchain.StrategyPassthrough,
		},
		{
			name:     "native-streaming engine refuses vad",
			manifest: streamingEngineManifest, engineID: "kyutai",
			traits:   sttchain.ProviderTraits{Batch: true, Stream: true},
			pref:     stt.PreferenceVAD,
			wantKind: sttchain.StrategyBuffered,
			wantErr:  stt.ErrIncompatibleStrategyProvider,
		},
		{
			name:     "empty engine_id resolves to manifest default",
			manifest: whisperOnlyManifest, engineID: "",
			traits:   sttchain.ProviderTraits{Batch: true},
			pref:     stt.PreferenceAuto,
			wantKind: sttchain.StrategyOverlapAgree,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sel := stt.NewSelectorWithRegistry(&sttmocks.FakeBatchExecutor{}, nil, mustRegistry(t, tc.manifest))
			provs := []stt.ProviderEligibility{
				{Provider: sttmocks.NewFakeProvider(sttchain.TierLocal, tc.traits), Tier: sttchain.TierLocal, Available: true},
			}
			cfg := stt.StreamConfig{Mode: stt.ModeAuto, StrategyPreference: tc.pref, EngineID: tc.engineID}
			out, err := sel.Select(context.Background(), cfg, sttchain.StreamStart{}, provs)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantKind, out.Kind)
		})
	}
}
