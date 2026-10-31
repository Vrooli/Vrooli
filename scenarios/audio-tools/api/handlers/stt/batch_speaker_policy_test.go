package stt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/stt/pipeline"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func TestSpeakerCodecMapsDeclaredBatchFormats(t *testing.T) {
	require.Equal(t, "wav", speakerCodec(" WAV ").String())
	require.Equal(t, "pcm_s16le", speakerCodec("pcm").String())
	require.Equal(t, "", speakerCodec("unknown").String())
}

func TestApplyBatchSpeakerPolicyReportsAppliedVerification(t *testing.T) {
	setBatchSpeakerConfig(t, pastWarmupCfg("profile-1"))
	h := &connectHandler{deps: Deps{SpeakerResource: fakeSpeakerResourceWithScore(t, 0.9)}}
	pcm := make([]byte, 32_000)
	got, outcome, err := h.applyBatchSpeakerPolicy(context.Background(), pcm, speakerCodec("pcm_s16le"))
	require.NoError(t, err)
	require.Equal(t, pcm, got)
	require.NotNil(t, outcome)
	require.True(t, outcome.Allowed)
	require.True(t, outcome.Applied)
	require.False(t, outcome.FallbackUsed)
	require.InDelta(t, 0.9, outcome.Score, 1e-6)
}

func TestApplyBatchSpeakerPolicyFailsClosedOnNonMatch(t *testing.T) {
	setBatchSpeakerConfig(t, pastWarmupCfg("profile-1"))
	h := &connectHandler{deps: Deps{SpeakerResource: fakeSpeakerResourceWithScore(t, 0.1)}}
	_, outcome, err := h.applyBatchSpeakerPolicy(context.Background(), make([]byte, 32_000), speakerCodec("pcm_s16le"))
	require.NoError(t, err)
	require.NotNil(t, outcome)
	require.False(t, outcome.Allowed)
	require.True(t, outcome.Applied)
	require.Contains(t, outcome.Reason, "speaker")
}

func TestAddBatchSpeakerPolicyExposesDegradedState(t *testing.T) {
	response := &sttv1.TranscribeResponse{}
	h := &connectHandler{}
	h.addBatchSpeakerPolicy(response, &batchSpeakerOutcome{Allowed: true, FallbackUsed: true, Reason: "resource unavailable"})
	require.Equal(t, "degraded", response.GetPolicyDetails()["speaker_policy"])
	require.Equal(t, "true", response.GetPolicyDetails()["speaker_fallback_used"])
}

func setBatchSpeakerConfig(t *testing.T, cfg pipeline.SpeakerConfig) {
	t.Helper()
	t.Cleanup(resetSpeakerCfg)
	speakerCfgMu.Lock()
	speakerCfg = speakerCfgDoc{
		Enabled: cfg.Enabled, Mode: cfg.Mode, Threshold: cfg.Threshold,
		ProfileIDs: cfg.ProfileIDs, FallbackWithoutVerification: cfg.FallbackWithoutVerification,
		MinDecisionSeconds: cfg.MinDecisionSeconds,
	}
	speakerCfgMu.Unlock()
}
