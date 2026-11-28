//go:build audio_lab

package corpus_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/corpus"
)

func TestTimedReferenceAllowsManualReviewedBoundariesWithoutAligner(t *testing.T) {
	reference := &corpus.TimedReference{
		Words: []corpus.TimedWord{{Word: "reviewed", StartSeconds: 0.1, EndSeconds: 0.5}},
		Provenance: corpus.TimingProvenance{
			Status: "human_reviewed", SourceReference: "clip:sha256:abc123",
			Reviewer: "reviewer-1", ReviewMethod: "listen-and-correct",
			ReviewedAt: "2026-11-20T22:00:00Z",
		},
	}
	require.NoError(t, reference.Validate(1000))
}

func TestTimedReferenceRequiresCompleteAlignmentProvenanceWhenPresent(t *testing.T) {
	reference := &corpus.TimedReference{
		Words: []corpus.TimedWord{{Word: "reviewed", StartSeconds: 0.1, EndSeconds: 0.5}},
		Provenance: corpus.TimingProvenance{
			Status: "human_reviewed", SourceReference: "clip:sha256:abc123",
			AlignmentTool: "forced-aligner-v1", Reviewer: "reviewer-1",
			ReviewMethod: "listen-and-correct", ReviewedAt: "2026-11-20T22:00:00Z",
		},
	}
	require.ErrorContains(t, reference.Validate(1000), "both alignment_tool and alignment_model")
}

func TestTimedReferenceAlignerOnlyRequiresAlignmentModelAndTool(t *testing.T) {
	reference := &corpus.TimedReference{
		Words:      []corpus.TimedWord{{Word: "aligned", StartSeconds: 0.1, EndSeconds: 0.5}},
		Provenance: corpus.TimingProvenance{Status: "aligner_only", SourceReference: "clip:sha256:abc123"},
	}
	require.ErrorContains(t, reference.Validate(1000), "alignment_tool and alignment_model")
}

func TestTimedReferenceTranscriptJoinIgnoresPunctuationAndCase(t *testing.T) {
	reference := &corpus.TimedReference{Words: []corpus.TimedWord{{Word: "Hello,"}, {Word: "WORLD!"}}}
	require.NoError(t, reference.ValidateTranscript("hello world."))
}

func TestTimedReferenceTranscriptJoinRejectsDifferentWords(t *testing.T) {
	reference := &corpus.TimedReference{Words: []corpus.TimedWord{{Word: "hello"}, {Word: "there"}}}
	require.ErrorContains(t, reference.ValidateTranscript("hello world"), "word 1")
	require.ErrorContains(t, reference.ValidateTranscript("hello"), "has 2 words but transcript has 1")
}
