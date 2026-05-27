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
				"profile_id": r.FormValue("profile_id"), "display_name": r.FormValue("display_name"),
				"embedding_dim": 192, "sample_rate": 16000, "enrollment_audio_seconds": 1.5,
				"model_name": "speechbrain/spkrec-ecapa-voxceleb", "created_at": "2026-05-27T00:00:00Z",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/verify":
			_ = r.ParseMultipartForm(1 << 20)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profile_id": r.FormValue("profile_id"), "matched": score >= 0.35, "score": score,
				"threshold": 0.35, "duration_ms": 12.0, "backend": "speechbrain",
				"model": "ecapa", "audio_seconds": 1.0,
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

// TestSpeakerVerificationIsolation_AllowsMatch proves the handler-layer
// verification adapter maps a matching ECAPA score to an allowed verdict and a
// non-match (filter mode) to a rejection — the audio-domain egress contract.
func TestSpeakerVerificationIsolation_AllowsMatch(t *testing.T) {
	iso := speakerVerification{
		cfg:    sttpipeline.SpeakerConfig{Enabled: true, Mode: "filter", Threshold: 0.35, ProfileIDs: []string{"sp-1"}},
		client: fakeSpeakerResourceWithScore(t, 0.9),
	}
	v := iso.Evaluate(context.Background(), []byte("pcm"))
	require.True(t, v.Allowed)
	require.False(t, v.FallbackUsed)

	// Drive the egress stage: an allowed verdict does not reject (the Gate
	// stamps Emit before stages run; a stage only flips to Drop/Reject).
	dec := egress.SpeakerStage{Isolation: iso}.Apply(context.Background(), egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, dec.Outcome)
}

func TestSpeakerVerificationIsolation_RejectsNonMatch(t *testing.T) {
	iso := speakerVerification{
		cfg:    sttpipeline.SpeakerConfig{Enabled: true, Mode: "filter", Threshold: 0.35, ProfileIDs: []string{"sp-1"}},
		client: fakeSpeakerResourceWithScore(t, 0.1), // below threshold
	}
	dec := egress.SpeakerStage{Isolation: iso}.Apply(context.Background(), egress.SegmentDecision{Text: "nickelback lyrics", Audio: []byte("pcm")})
	require.Equal(t, egress.Reject, dec.Outcome)
	require.NotEmpty(t, dec.Reason)
}

// TestSpeakerVerificationIsolation_FallbackWhenNoProfile proves fallback
// semantics: with no enrolled profile and FallbackWithoutVerification, the
// segment is allowed but flagged as not-actually-verified.
func TestSpeakerVerificationIsolation_FallbackWhenNoProfile(t *testing.T) {
	iso := speakerVerification{
		cfg:    sttpipeline.SpeakerConfig{Enabled: true, Mode: "filter", Threshold: 0.35, FallbackWithoutVerification: true},
		client: fakeSpeakerResourceWithScore(t, 0.9),
	}
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
