//go:build audio_lab

package corpus_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/corpus"
)

func TestAssessQualityCohortRejectsLocalCleanOnlyCorpus(t *testing.T) {
	got := corpus.AssessQualityCohort([]corpus.Clip{{
		ID: "local-1", ReferenceText: "hello world", CorpusRevision: "local", LicenseReference: "fixture",
		SpeakerPseudonym: "speaker-1", Split: corpus.SplitDev, NoiseCondition: corpus.NoiseClean, ContentSHA256: "hash-1",
	}}, "voice-pilot-en-v1")
	require.False(t, got.Eligible)
	require.Contains(t, got.Reasons, "requires at least 60 clips, got 1")
	require.Contains(t, got.Reasons, "clip \"local-1\" has a missing or mismatched corpus revision")
	require.Contains(t, got.Reasons, "clip \"local-1\" is not in the held-out split")
}

func TestAssessQualityCohortAcceptsCompleteRecipe(t *testing.T) {
	clips := make([]corpus.Clip, 0, corpus.RequiredQualityClips)
	for i := 0; i < corpus.RequiredQualityClips; i++ {
		condition := corpus.NoiseClean
		snr := float64(0)
		if i%3 == 1 {
			condition, snr = corpus.NoiseNoisy, 20
		} else if i%3 == 2 {
			condition, snr = corpus.NoiseNoisy, 10
		}
		clips = append(clips, corpus.Clip{
			ID: fmt.Sprintf("clip-%d", i), ReferenceText: strings.Repeat("one two three four five six seven eight nine ten ", 10), CorpusRevision: "voice-pilot-en-v1",
			LicenseReference: "license-1", SpeakerPseudonym: fmt.Sprintf("speaker-%d", i%corpus.RequiredQualitySpeakers),
			Split: corpus.SplitHeldOut, NoiseCondition: condition, SNRDB: snr, ContentSHA256: fmt.Sprintf("hash-%d", i),
			ReferenceReview: &corpus.ReferenceReview{Reviewer: "auditor@example.test", Protocol: "listen-and-correct-v1", ReviewedAt: "2026-06-29T12:00:00Z", ReferenceSHA256: corpus.ReferenceDigest(strings.Repeat("one two three four five six seven eight nine ten ", 10))},
		})
	}
	got := corpus.AssessQualityCohort(clips, "voice-pilot-en-v1")
	require.True(t, got.Eligible, "reasons: %v", got.Reasons)
	require.Equal(t, corpus.RequiredQualityClips, got.ClipCount)
	require.Equal(t, corpus.RequiredQualitySpeakers, got.SpeakerCount)
}

func TestAssessQualityCohortRejectsUnreviewedImportedReferences(t *testing.T) {
	got := corpus.AssessQualityCohort([]corpus.Clip{{ID: "imported", ReferenceText: "silently supplied labels", CorpusRevision: "voice-pilot-en-v1", Split: corpus.SplitHeldOut}}, "voice-pilot-en-v1")
	require.False(t, got.Eligible)
	require.Contains(t, got.Reasons, `clip "imported" has no human-reviewed reference`)
}

func TestReferenceReviewMustBindTranscriptRevision(t *testing.T) {
	review := &corpus.ReferenceReview{Reviewer: "auditor", Protocol: "listen-v1", ReviewedAt: "2026-06-29T12:00:00Z", ReferenceSHA256: corpus.ReferenceDigest("reviewed transcript")}
	require.NoError(t, review.Validate("reviewed transcript"))
	require.Error(t, review.Validate("edited transcript"))
}
