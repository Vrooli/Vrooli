//go:build audio_lab

package evaldeps

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/testutil/mocks"

	"github.com/vrooli/api-core/schedule"
)

// TestLoadClipsMapsCorpusMetadataAndDefaultsSampleRate covers the adapter that
// turns corpus rows into eval clips. A wrong sample rate silently corrupts
// every replay measurement, so the 16 kHz default is asserted explicitly.
func TestLoadClipsMapsCorpusMetadataAndDefaultsSampleRate(t *testing.T) {
	ctx := context.Background()
	blobs := mocks.NewFakeBlobStore()
	require.NoError(t, blobs.Put(ctx, "blob-a", []byte{1, 2, 3}, "audio/wav"))
	require.NoError(t, blobs.Put(ctx, "blob-b", []byte{4, 5}, "audio/wav"))

	repo := &fakeClipRepo{
		order: []string{"a", "b"},
		clips: map[string]intcorpus.Clip{
			"a": {ID: "a", BlobKey: "blob-a", SampleRateHz: 0, ReferenceText: "hello", Format: "wav", TimedReference: &intcorpus.TimedReference{
				Words:      []intcorpus.TimedWord{{Word: "hello", StartSeconds: 0.1, EndSeconds: 0.4}},
				Provenance: intcorpus.TimingProvenance{Status: "human_reviewed", SourceReference: "ref-a", AlignmentTool: "aligner", AlignmentModel: "model-a", Reviewer: "reviewer-a", ReviewMethod: "listen", ReviewedAt: "2026-11-20T12:00:00Z"},
			}},
			"b": {ID: "b", BlobKey: "blob-b", SampleRateHz: 22050, ReferenceText: "world", Format: "wav"},
		},
	}
	service := intcorpus.NewService(repo, blobs, schedule.System())
	deps := New(nil, service, func(string) sttchain.Provider { return nil }, sttpkg.Defaults(), nil)

	all, err := deps.LoadClips(ctx, nil)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, []byte{1, 2, 3}, all[0].PCM)
	require.Equal(t, "a", all[0].ID)
	require.Equal(t, 16000, all[0].SampleRate, "a missing corpus sample rate defaults to 16 kHz")
	require.Equal(t, "hello", all[0].Reference)
	require.Equal(t, "wav", all[0].Format)
	require.Equal(t, []inteval.TimedWord{{Word: "hello", Start: 0.1, End: 0.4}}, all[0].TimedReference)
	require.Equal(t, &inteval.TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "ref-a", AlignmentTool: "aligner", AlignmentModel: "model-a", Reviewer: "reviewer-a", ReviewMethod: "listen", ReviewedAt: "2026-11-20T12:00:00Z"}, all[0].TimingProvenance)
	require.Equal(t, 22050, all[1].SampleRate, "an explicit corpus sample rate is preserved")

	subset, err := deps.LoadClips(ctx, []string{"b"})
	require.NoError(t, err)
	require.Len(t, subset, 1)
	require.Equal(t, "b", subset[0].ID, "explicit clip ids select only that clip")
	require.Equal(t, []byte{4, 5}, subset[0].PCM)
}

type fakeClipRepo struct {
	order []string
	clips map[string]intcorpus.Clip
}

var errFakeRepoUnused = errors.New("fake repository: method not used by this test")

func (r *fakeClipRepo) Create(context.Context, intcorpus.Clip) (intcorpus.Clip, error) {
	return intcorpus.Clip{}, errFakeRepoUnused
}

func (r *fakeClipRepo) Get(_ context.Context, id string) (intcorpus.Clip, error) {
	clip, ok := r.clips[id]
	if !ok {
		return intcorpus.Clip{}, intcorpus.ErrClipNotFound{ID: id}
	}
	return clip, nil
}

func (r *fakeClipRepo) List(context.Context, intcorpus.ListFilter) ([]intcorpus.Clip, error) {
	clips := make([]intcorpus.Clip, 0, len(r.order))
	for _, id := range r.order {
		clips = append(clips, r.clips[id])
	}
	return clips, nil
}

func (r *fakeClipRepo) ListBySourceID(context.Context, string) ([]intcorpus.Clip, error) {
	return nil, errFakeRepoUnused
}

func (r *fakeClipRepo) Delete(context.Context, string) error { return errFakeRepoUnused }

func (r *fakeClipRepo) CountCreatedInWindow(context.Context, time.Time, time.Time) (int64, error) {
	return 0, errFakeRepoUnused
}

func (r *fakeClipRepo) BackfillSourceLineage(context.Context, []intcorpus.LineageEntry, bool) (int, error) {
	return 0, errFakeRepoUnused
}

func (r *fakeClipRepo) ReviewReference(context.Context, string, string, string, intcorpus.ReferenceReview) (intcorpus.Clip, error) {
	return intcorpus.Clip{}, errFakeRepoUnused
}

func (r *fakeClipRepo) ReviewReferenceGroup(context.Context, []intcorpus.ReviewMember, string, string, string, string, string, string, string, intcorpus.ReferenceReview) ([]intcorpus.Clip, error) {
	return nil, errFakeRepoUnused
}
