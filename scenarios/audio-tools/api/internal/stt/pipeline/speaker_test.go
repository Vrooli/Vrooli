package pipeline

import (
	"context"
	"strings"
	"testing"
)

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

func TestContainsRemoveStringHelpers(t *testing.T) {
	if !containsString([]string{"a", "b"}, "b") {
		t.Fatalf("containsString(true) failed")
	}
	if containsString([]string{"a", "b"}, "c") {
		t.Fatalf("containsString(false) failed")
	}
	got := removeString([]string{"a", "b", "a"}, "a")
	if len(got) != 1 || got[0] != "b" {
		t.Fatalf("removeString didn't remove all: %v", got)
	}
}
