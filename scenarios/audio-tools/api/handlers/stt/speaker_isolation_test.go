package stt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/stt/egress"
	sttpipeline "audio-tools/internal/stt/pipeline"
	sttspeaker "audio-tools/internal/stt/speaker"
)

// fakeSpeakerResource stands up an httptest server shaped like the
// speaker-verification resource's enroll/verify/delete contract. score controls
// the verify result so tests can drive match/non-match without real ECAPA.
func fakeSpeakerResource(t *testing.T) *sttpipeline.SpeakerClient {
	t.Helper()
	return fakeSpeakerResourceWithScore(t, 0.9)
}

func fakeSpeakerResourceWithScore(t *testing.T, score float64) *sttpipeline.SpeakerClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/profiles":
			_ = r.ParseMultipartForm(1 << 20)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profile_id": r.FormValue("profile_id"), "clip_id": "clip-1", "label": r.FormValue("label"),
				"voiced_seconds": 1.5, "audio_seconds": 1.5, "clip_count": 1, "total_voiced_seconds": 1.5,
				"embedding_dim": 192, "sample_rate": 16000,
				"model_name": "speechbrain/spkrec-ecapa-voxceleb", "created_at": "2026-05-27T00:00:00Z",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/verify":
			_ = r.ParseMultipartForm(1 << 20)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profile_id": r.FormValue("profile_id"), "matched": score >= 0.35, "score": score,
				"threshold": 0.35, "sufficient": true, "voiced_seconds": 2.0,
				"duration_ms": 12.0, "backend": "speechbrain", "model": "ecapa", "audio_seconds": 1.0,
			})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return &sttpipeline.SpeakerClient{BaseURL: srv.URL, Doer: http.DefaultClient}
}

// pastWarmupCfg builds a filter-mode config whose warm-up window (1.0s) is
// crossed by a single fixture segment (voiced_seconds 2.0), so one Evaluate is
// past warm-up and the score governs the verdict.
func pastWarmupCfg(profiles ...string) sttpipeline.SpeakerConfig {
	return sttpipeline.SpeakerConfig{
		Enabled: true, Mode: "filter", Threshold: 0.35, ProfileIDs: profiles,
		MinDecisionSeconds: 1.0,
	}
}

// TestSpeakerVerificationIsolation_AllowsMatch proves the handler-layer
// verification adapter maps a matching ECAPA score to an allowed verdict and a
// non-match (filter mode) to a rejection — the audio-domain egress contract.
func TestSpeakerVerificationIsolation_AllowsMatch(t *testing.T) {
	iso := sttspeaker.NewIsolation(pastWarmupCfg("sp-1"), fakeSpeakerResourceWithScore(t, 0.9), nil)
	v := iso.Evaluate(context.Background(), []byte("pcm"))
	require.True(t, v.Allowed)
	require.False(t, v.FallbackUsed)

	// Drive the egress stage: an allowed verdict does not reject (the Gate
	// stamps Emit before stages run; a stage only flips to Drop/Reject).
	dec := egress.SpeakerStage{Isolation: iso}.Apply(context.Background(), egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, dec.Outcome)
}

func TestSpeakerVerificationIsolation_RejectsNonMatch(t *testing.T) {
	iso := sttspeaker.NewIsolation(pastWarmupCfg("sp-1"), fakeSpeakerResourceWithScore(t, 0.1), nil) // below threshold
	dec := egress.SpeakerStage{Isolation: iso}.Apply(context.Background(), egress.SegmentDecision{Text: "nickelback lyrics", Audio: []byte("pcm")})
	require.Equal(t, egress.Reject, dec.Outcome)
	require.NotEmpty(t, dec.Reason)
	// The real resource score (0.1) and configured threshold (0.35) must reach
	// the decision so the rejection banner shows them rather than 0.00/0.00.
	require.InDelta(t, 0.1, dec.Score, 1e-6)
	require.InDelta(t, 0.35, dec.Threshold, 1e-6)
}

// TestSpeakerVerificationIsolation_WarmupNeverRejects proves a below-threshold
// segment inside the warm-up window (not enough voiced audio accrued yet) is
// allowed through rather than rejected.
func TestSpeakerVerificationIsolation_WarmupNeverRejects(t *testing.T) {
	cfg := sttpipeline.SpeakerConfig{
		Enabled: true, Mode: "filter", Threshold: 0.35, ProfileIDs: []string{"sp-1"},
		MinDecisionSeconds: 10.0, // fixture's 2.0s/segment stays in warm-up
	}
	iso := sttspeaker.NewIsolation(cfg, fakeSpeakerResourceWithScore(t, 0.1), nil)
	dec := egress.SpeakerStage{Isolation: iso}.Apply(context.Background(), egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, dec.Outcome, "warm-up window must not reject")
}

// TestSpeakerVerificationIsolation_FallbackWhenNoProfile proves fallback
// semantics: with no enrolled profile and FallbackWithoutVerification, the
// segment is allowed but flagged as not-actually-verified.
func TestSpeakerVerificationIsolation_FallbackWhenNoProfile(t *testing.T) {
	cfg := pastWarmupCfg()
	cfg.FallbackWithoutVerification = true
	iso := sttspeaker.NewIsolation(cfg, fakeSpeakerResourceWithScore(t, 0.9), nil)
	v := iso.Evaluate(context.Background(), []byte("pcm"))
	require.True(t, v.Allowed)
	require.True(t, v.FallbackUsed, "no profile bound -> allowed via fallback, flagged unverified")
}

// TestCurrentSpeakerIsolation_NilWhenOff proves the audio-domain stage is
// omitted (nil isolation) when speaker isolation is disabled or off.
func TestCurrentSpeakerIsolation_NilWhenOff(t *testing.T) {
	resetSpeakerCfg()
	t.Cleanup(resetSpeakerCfg)
	require.Nil(t, currentSpeakerIsolation(Deps{SpeakerResource: fakeSpeakerResource(t)}),
		"default config is disabled -> no isolation stage")
}
