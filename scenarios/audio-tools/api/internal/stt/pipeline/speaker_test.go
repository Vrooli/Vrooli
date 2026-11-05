package pipeline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// fakeVerifyClient stands up a /v1/verify endpoint that reports matched =
// (score >= threshold), so EvaluateSpeaker's mode semantics can be exercised
// without real ECAPA.
func fakeVerifyClient(t *testing.T, score float64) *SpeakerClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		threshold, _ := strconv.ParseFloat(r.FormValue("threshold"), 64)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"profile_id":     r.FormValue("profile_id"),
			"matched":        score >= threshold,
			"score":          score,
			"threshold":      threshold,
			"sufficient":     true,
			"voiced_seconds": 2.0,
		})
	}))
	t.Cleanup(srv.Close)
	return &SpeakerClient{BaseURL: srv.URL, Doer: http.DefaultClient}
}

// TestEvaluateSpeakerFilterMatchAndNonMatch proves the filter-mode gate: a
// matching score allows the segment; a non-matching score blocks it (the
// non-target voice is dropped).
func TestEvaluateSpeakerFilterMatchAndNonMatch(t *testing.T) {
	cfg := SpeakerConfig{Enabled: true, Mode: "filter", Threshold: 0.5, ProfileIDs: []string{"p1"}}

	match := EvaluateSpeaker(context.Background(), cfg, fakeVerifyClient(t, 0.9), []byte("pcm"))
	if !match.Allowed || !match.Matched || !match.Applied {
		t.Fatalf("matching score must allow + mark matched/applied: %+v", match)
	}

	nonMatch := EvaluateSpeaker(context.Background(), cfg, fakeVerifyClient(t, 0.1), []byte("pcm"))
	if nonMatch.Allowed {
		t.Fatalf("filter mode must block a non-matching voice: %+v", nonMatch)
	}
	if !nonMatch.Applied {
		t.Fatalf("verification ran, so Applied must be true even on non-match: %+v", nonMatch)
	}
}

// TestEvaluateSpeakerAdvisoryAllowsNonMatch proves advisory mode scores but
// never blocks: a non-matching segment is still Allowed, with Applied=true so
// consumers can see the gate ran and the score it produced.
func TestEvaluateSpeakerAdvisoryAllowsNonMatch(t *testing.T) {
	cfg := SpeakerConfig{Enabled: true, Mode: "advisory", Threshold: 0.5, ProfileIDs: []string{"p1"}}
	got := EvaluateSpeaker(context.Background(), cfg, fakeVerifyClient(t, 0.1), []byte("pcm"))
	if !got.Allowed {
		t.Fatalf("advisory mode must always allow: %+v", got)
	}
	if !got.Applied {
		t.Fatalf("advisory still RUNS verification -> Applied=true: %+v", got)
	}
	if got.Matched {
		t.Fatalf("non-matching score must not report Matched: %+v", got)
	}
}

// fakeInsufficientClient stands up a /v1/verify that reports the segment had
// too little voiced audio to judge (sufficient:false, score:0).
func fakeInsufficientClient(t *testing.T) *SpeakerClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"profile_id":     r.FormValue("profile_id"),
			"matched":        false,
			"score":          0.0,
			"sufficient":     false,
			"voiced_seconds": 0.3,
		})
	}))
	t.Cleanup(srv.Close)
	return &SpeakerClient{BaseURL: srv.URL, Doer: http.DefaultClient}
}

// TestEvaluateSpeakerInsufficientPassesThrough proves an insufficient-audio
// segment is undetermined evidence: it is never rejected (even in filter mode)
// and is flagged Applied=false / Sufficient=false so the session verifier
// ignores it during warm-up.
func TestEvaluateSpeakerInsufficientPassesThrough(t *testing.T) {
	cfg := SpeakerConfig{Enabled: true, Mode: "filter", Threshold: 0.5, ProfileIDs: []string{"p1"}}
	got := EvaluateSpeaker(context.Background(), cfg, fakeInsufficientClient(t), []byte("pcm"))
	if !got.Allowed {
		t.Fatalf("insufficient audio must never be rejected: %+v", got)
	}
	if got.Applied {
		t.Fatalf("insufficient audio is not an applied verification: %+v", got)
	}
	if got.Sufficient {
		t.Fatalf("expected Sufficient=false: %+v", got)
	}
	if got.VoicedSeconds != 0.3 {
		t.Fatalf("expected VoicedSeconds propagated: %+v", got)
	}
}

func TestEvaluateSpeakerDisabled(t *testing.T) {
	cfg := SpeakerConfig{Enabled: false}
	got := EvaluateSpeaker(context.Background(), cfg, nil, []byte("x"))
	if got.Enabled {
		t.Fatalf("expected Enabled=false")
	}
	if !got.Allowed {
		t.Fatalf("disabled should default Allowed=true")
	}
}

func TestEvaluateSpeakerOffMode(t *testing.T) {
	cfg := SpeakerConfig{Enabled: true, Mode: "off"}
	got := EvaluateSpeaker(context.Background(), cfg, nil, []byte("x"))
	if got.Enabled {
		t.Fatalf("mode=off should keep Enabled=false")
	}
}

func TestEvaluateSpeakerNoProfiles(t *testing.T) {
	cfg := SpeakerConfig{Enabled: true, Mode: "enforce", ProfileIDs: nil, FallbackWithoutVerification: true}
	got := EvaluateSpeaker(context.Background(), cfg, nil, []byte("x"))
	if !got.Allowed {
		t.Fatalf("FallbackWithoutVerification=true should allow even with no profiles")
	}
	if !strings.Contains(got.ErrorMessage, "no speaker profiles") {
		t.Fatalf("expected diagnostic, got %q", got.ErrorMessage)
	}

	cfg.FallbackWithoutVerification = false
	got = EvaluateSpeaker(context.Background(), cfg, nil, []byte("x"))
	if got.Allowed {
		t.Fatalf("FallbackWithoutVerification=false should reject")
	}
}

func TestEvaluateSpeakerNoClient(t *testing.T) {
	cfg := SpeakerConfig{Enabled: true, Mode: "enforce", ProfileIDs: []string{"p1"}, FallbackWithoutVerification: true}
	got := EvaluateSpeaker(context.Background(), cfg, nil, []byte("x"))
	if !got.Allowed {
		t.Fatalf("nil client + FallbackWithoutVerification=true should allow")
	}
}

func TestFormatSpeakerDecisionError(t *testing.T) {
	d := SpeakerDecision{ErrorMessage: "boom"}
	if got := FormatSpeakerDecisionError(d); !strings.Contains(got, "boom") {
		t.Fatalf("expected error message threaded through, got %q", got)
	}
}
