//go:build audio_lab

package corpus_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/corpus"
	"audio-tools/internal/testutil/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

func newService(t *testing.T) (*corpus.Service, *mocks.FakeBlobStore) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	repo := corpus.NewSQLiteRepository(newSchemaDB(t), clk)
	blobs := mocks.NewFakeBlobStore()
	return corpus.NewService(repo, blobs, clk), blobs
}

func TestService_CreateGetAudioDelete(t *testing.T) {
	ctx := context.Background()
	svc, blobs := newService(t)

	audio := []byte{1, 2, 3, 4, 5, 6}
	clip, err := svc.CreateClip(ctx, corpus.CreateClipInput{
		Audio:            audio,
		ReferenceText:    "hello world",
		Tags:             []string{"smoke"},
		DurationMs:       320,
		SampleRateHz:     16000,
		Format:           "pcm_s16le",
		Source:           corpus.SourceFreeForm,
		CorpusRevision:   "voice-pilot-en-v1",
		LicenseReference: "fixture-license-1",
		SpeakerPseudonym: "speaker-01",
		Split:            corpus.SplitHeldOut,
		NoiseCondition:   corpus.NoiseNoisy,
		SNRDB:            20,
	})
	require.NoError(t, err)
	require.NotEmpty(t, clip.BlobKey)
	require.Contains(t, clip.BlobKey, "clips/2026-06/", "blob key is the dated hierarchical scheme")
	require.Equal(t, 1, blobs.Count(), "audio bytes landed in the blob store")

	gotAudio, gotClip, err := svc.GetClipAudio(ctx, clip.ID)
	require.NoError(t, err)
	require.Equal(t, audio, gotAudio, "audio round-trips byte-for-byte")
	require.Equal(t, "hello world", gotClip.ReferenceText)
	require.Equal(t, "voice-pilot-en-v1", gotClip.CorpusRevision)
	require.Equal(t, "fixture-license-1", gotClip.LicenseReference)
	require.Equal(t, "speaker-01", gotClip.SpeakerPseudonym)
	require.Equal(t, corpus.SplitHeldOut, gotClip.Split)
	require.Equal(t, corpus.NoiseNoisy, gotClip.NoiseCondition)
	require.Equal(t, float64(20), gotClip.SNRDB)
	require.Len(t, gotClip.ContentSHA256, 64, "stored clip must carry a content digest")

	require.NoError(t, svc.DeleteClip(ctx, clip.ID))
	require.Equal(t, 0, blobs.Count(), "delete removes the blob too")
	_, err = svc.GetClip(ctx, clip.ID)
	require.ErrorAs(t, err, &corpus.ErrClipNotFound{})
}

func TestService_CreateRequiresAudio(t *testing.T) {
	svc, _ := newService(t)
	_, err := svc.CreateClip(context.Background(), corpus.CreateClipInput{ReferenceText: "no audio"})
	require.Error(t, err)
}

func TestService_ReviewReferenceBindsExactSavedTranscriptAndRejectsStaleDigest(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService(t)
	clip, err := svc.CreateClip(ctx, corpus.CreateClipInput{Audio: []byte{1, 2}, ReferenceText: "old words", Format: "pcm_s16le"})
	require.NoError(t, err)
	_, err = svc.ReviewReference(ctx, clip.ID, corpus.ReferenceDigest("stale"), "corrected words", "reviewer-1", "listen-v1")
	require.ErrorAs(t, err, &corpus.ErrStaleReference{})
	unchanged, err := svc.GetClip(ctx, clip.ID)
	require.NoError(t, err)
	require.Equal(t, "old words", unchanged.ReferenceText)
	require.Nil(t, unchanged.ReferenceReview)

	saved, err := svc.ReviewReference(ctx, clip.ID, corpus.ReferenceDigest("old words"), "corrected words", "reviewer-1", "listen-v1")
	require.NoError(t, err)
	require.Equal(t, "corrected words", saved.ReferenceText)
	require.NotNil(t, saved.ReferenceReview)
	require.Equal(t, corpus.ReferenceDigest(saved.ReferenceText), saved.ReferenceReview.ReferenceSHA256)
	require.Equal(t, "reviewer-1", saved.ReferenceReview.Reviewer)
	require.Equal(t, "listen-v1", saved.ReferenceReview.Protocol)
	require.NotEmpty(t, saved.ReferenceReview.ReviewedAt)
}

func makeReviewGroup(t *testing.T, alter func(index int, in *corpus.CreateClipInput), lineageAlter func(index int, entry *corpus.LineageEntry)) (*corpus.Service, []corpus.Clip) {
	t.Helper()
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	repo := corpus.NewSQLiteRepository(newSchemaDB(t), clk)
	svc := corpus.NewService(repo, mocks.NewFakeBlobStore(), clk)
	text := "the source transcript"
	clips := make([]corpus.Clip, 0, 3)
	for i, condition := range []corpus.NoiseCondition{corpus.NoiseClean, corpus.NoiseNoisy, corpus.NoiseNoisy} {
		in := corpus.CreateClipInput{ReferenceText: text, CorpusRevision: "vctk-v1", SpeakerPseudonym: "vctk-p230", LicenseReference: "license", Split: corpus.SplitHeldOut, NoiseCondition: condition}
		if i == 2 {
			in.SNRDB = 10
		} else if i == 1 {
			in.SNRDB = 20
		}
		if alter != nil {
			alter(i, &in)
		}
		entry := corpus.LineageEntry{SourceID: "p230_001", SpeakerPseudonym: in.SpeakerPseudonym, AccentGroup: "uk"}
		if lineageAlter != nil {
			lineageAlter(i, &entry)
		}
		id := []string{"variant-0", "variant-1", "variant-2"}[i]
		clip, err := repo.Create(context.Background(), corpus.Clip{ID: id, ReferenceText: in.ReferenceText, BlobKey: id, CorpusRevision: in.CorpusRevision, SpeakerPseudonym: entry.SpeakerPseudonym, LicenseReference: in.LicenseReference, Split: in.Split, NoiseCondition: condition, SourceID: entry.SourceID, AccentGroup: entry.AccentGroup, ManifestTranscriptSHA256: corpus.ReferenceDigest(in.ReferenceText), LineageManifestSHA256: strings.Repeat("f", 64)})
		require.NoError(t, err)
		clips = append(clips, clip)
	}
	return svc, clips
}

func reviewGroupMembers(clips []corpus.Clip) []corpus.ReviewMember {
	members := make([]corpus.ReviewMember, 0, len(clips))
	for _, clip := range clips {
		members = append(members, corpus.ReviewMember{ID: clip.ID, ExpectedReferenceSHA256: corpus.ReferenceDigest(clip.ReferenceText)})
	}
	return members
}

func TestService_ReviewReferenceGroupAuditsEveryExactVariant(t *testing.T) {
	svc, clips := makeReviewGroup(t, nil, nil)
	saved, err := svc.ReviewReferenceGroup(context.Background(), reviewGroupMembers(clips), "corrected source transcript", "human-1", "listen-and-correct-each-source-variant-v1")
	require.NoError(t, err)
	require.Len(t, saved, 3)
	groupID := saved[0].ReferenceReview.ReviewGroupID
	require.NotEmpty(t, groupID)
	for _, clip := range saved {
		require.Equal(t, "corrected source transcript", clip.ReferenceText)
		require.NotNil(t, clip.ReferenceReview)
		require.Equal(t, corpus.ReferenceDigest(clip.ReferenceText), clip.ReferenceReview.ReferenceSHA256)
		require.Equal(t, "listen-and-correct-each-source-variant-v1", clip.ReferenceReview.Protocol)
		require.Equal(t, groupID, clip.ReferenceReview.ReviewGroupID)
	}
}

func TestService_ReviewReferenceGroupRejectsUnsafeScopesAndStaleMembers(t *testing.T) {
	tests := []struct {
		name         string
		alter        func(int, *corpus.CreateClipInput)
		lineageAlter func(int, *corpus.LineageEntry)
		mutate       func([]corpus.ReviewMember)
		want         string
	}{
		{name: "revision mismatch", alter: func(i int, in *corpus.CreateClipInput) {
			if i == 2 {
				in.CorpusRevision = "other"
			}
		}, want: "mismatched revision"},
		{name: "speaker mismatch", alter: func(i int, in *corpus.CreateClipInput) {
			if i == 2 {
				in.SpeakerPseudonym = "other"
			}
		}, want: "mismatched revision"},
		{name: "accent mismatch", lineageAlter: func(i int, entry *corpus.LineageEntry) {
			if i == 2 {
				entry.AccentGroup = "us"
			}
		}, want: "mismatched revision"},
		{name: "source mismatch", lineageAlter: func(i int, entry *corpus.LineageEntry) {
			if i == 2 {
				entry.SourceID = "other-source"
			}
		}, want: "missing clip"},
		{name: "transcript mismatch", alter: func(i int, in *corpus.CreateClipInput) {
			if i == 2 {
				in.ReferenceText = "a different transcript"
			}
		}, want: "mismatched transcripts"},
		{name: "stale digest", mutate: func(m []corpus.ReviewMember) { m[1].ExpectedReferenceSHA256 = corpus.ReferenceDigest("stale") }, want: "changed since review began"},
		{name: "partial group", want: "every source variant"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, clips := makeReviewGroup(t, tt.alter, tt.lineageAlter)
			members := reviewGroupMembers(clips)
			if tt.mutate != nil {
				tt.mutate(members)
			}
			if tt.name == "partial group" {
				members = members[:2]
			}
			_, err := svc.ReviewReferenceGroup(context.Background(), members, "corrected", "human-1", "listen-each-v1")
			require.ErrorContains(t, err, tt.want)
			for _, before := range clips {
				after, getErr := svc.GetClip(context.Background(), before.ID)
				require.NoError(t, getErr)
				require.Equal(t, before.ReferenceText, after.ReferenceText)
				require.Nil(t, after.ReferenceReview)
			}
		})
	}
}

// failingRepo errors on Create so the service's blob-rollback can be tested.
type failingRepo struct{ corpus.Repository }

func (failingRepo) Create(context.Context, corpus.Clip) (corpus.Clip, error) {
	return corpus.Clip{}, errors.New("metadata write failed")
}

func TestService_RollsBackBlobOnMetadataFailure(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	blobs := mocks.NewFakeBlobStore()
	svc := corpus.NewService(failingRepo{}, blobs, clk)

	_, err := svc.CreateClip(context.Background(), corpus.CreateClipInput{
		Audio: []byte{1, 2, 3}, ReferenceText: "x", Format: "pcm_s16le",
	})
	require.Error(t, err)
	require.Equal(t, 0, blobs.Count(), "a failed metadata write must not leak an orphan blob")
}
