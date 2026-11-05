package stt

import (
	"strings"
	"testing"

	"audio-tools/internal/sttengine"
)

func engineCandidate(id string, native bool, verdict, reason string) sttengine.Candidate {
	c := sttengine.Candidate{Verdict: verdict, Reason: reason}
	c.Engine.ID = id
	c.Engine.Provides.NativeStreaming = native
	return c
}

// The outage this notice exists for: kyutai-stt failed artifact verification,
// the resolver seated CPU whisper for every dictation session, and the only
// symptom a person could observe was that their words came out wrong. A
// degradation that changes transcription quality has to be on the wire.
func TestStreamingDegradationNoticeNamesTheRejectedStreamingEngine(t *testing.T) {
	resolution := &sttengine.Resolution{
		Selected: "whisper-local",
		Candidates: []sttengine.Candidate{
			engineCandidate("whisper-local", false, "selected", "tiebreak"),
			engineCandidate("kyutai", true, "rejected", "not_installed: backing resource is not installed"),
		},
	}
	notice := streamingDegradationNotice(resolution, "whisper-local")
	if notice == "" {
		t.Fatal("streamingDegradationNotice() = \"\", want a degradation notice")
	}
	for _, want := range []string{"whisper-local", "kyutai", "not_installed"} {
		if !strings.Contains(notice, want) {
			t.Fatalf("notice = %q, want it to contain %q", notice, want)
		}
	}
}

func TestStreamingDegradationNoticeSilentWhenStreamingEngineSeated(t *testing.T) {
	resolution := &sttengine.Resolution{
		Selected: "kyutai",
		Candidates: []sttengine.Candidate{
			engineCandidate("whisper-local", false, "candidate", ""),
			engineCandidate("kyutai", true, "selected", ""),
		},
	}
	if notice := streamingDegradationNotice(resolution, "kyutai"); notice != "" {
		t.Fatalf("streamingDegradationNotice() = %q, want \"\" when the streaming engine is seated", notice)
	}
}

// Buffered transcription is the design when nothing streams. That is not a
// degradation and must not be reported as one, or the signal becomes noise.
func TestStreamingDegradationNoticeSilentWhenNoStreamingEngineExists(t *testing.T) {
	resolution := &sttengine.Resolution{
		Selected: "whisper-local",
		Candidates: []sttengine.Candidate{
			engineCandidate("whisper-local", false, "selected", ""),
		},
	}
	if notice := streamingDegradationNotice(resolution, "whisper-local"); notice != "" {
		t.Fatalf("streamingDegradationNotice() = %q, want \"\"", notice)
	}
}

func TestStreamingDegradationNoticeHandlesMissingResolution(t *testing.T) {
	if notice := streamingDegradationNotice(nil, "whisper-local"); notice != "" {
		t.Fatalf("streamingDegradationNotice(nil) = %q, want \"\"", notice)
	}
}
