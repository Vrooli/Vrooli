package corpus_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/corpus"
	"audio-tools/internal/testutil/mocks"
)

func newService(t *testing.T) (*corpus.Service, *mocks.FakeBlobStore) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	repo := corpus.NewSQLiteRepository(newSchemaDB(t), clk)
	blobs := mocks.NewFakeBlobStore()
	return corpus.NewService(repo, blobs, clk), blobs
}

func TestService_CreateGetAudioDelete(t *testing.T) {
	ctx := context.Background()
	svc, blobs := newService(t)

	audio := []byte{1, 2, 3, 4, 5, 6}
	clip, err := svc.CreateClip(ctx, corpus.CreateClipInput{
		Audio:         audio,
		ReferenceText: "hello world",
		Tags:          []string{"smoke"},
		DurationMs:    320,
		SampleRateHz:  16000,
		Format:        "pcm_s16le",
		Source:        corpus.SourceFreeForm,
	})
	require.NoError(t, err)
	require.NotEmpty(t, clip.BlobKey)
	require.Contains(t, clip.BlobKey, "clips/2026-06/", "blob key is the dated hierarchical scheme")
	require.Equal(t, 1, blobs.Count(), "audio bytes landed in the blob store")

	gotAudio, gotClip, err := svc.GetClipAudio(ctx, clip.ID)
	require.NoError(t, err)
	require.Equal(t, audio, gotAudio, "audio round-trips byte-for-byte")
	require.Equal(t, "hello world", gotClip.ReferenceText)

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

// failingRepo errors on Create so the service's blob-rollback can be tested.
type failingRepo struct{ corpus.Repository }

func (failingRepo) Create(context.Context, corpus.Clip) (corpus.Clip, error) {
	return corpus.Clip{}, errors.New("metadata write failed")
}

func TestService_RollsBackBlobOnMetadataFailure(t *testing.T) {
	clk := mocks.NewFakeClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	blobs := mocks.NewFakeBlobStore()
	svc := corpus.NewService(failingRepo{}, blobs, clk)

	_, err := svc.CreateClip(context.Background(), corpus.CreateClipInput{
		Audio: []byte{1, 2, 3}, ReferenceText: "x", Format: "pcm_s16le",
	})
	require.Error(t, err)
	require.Equal(t, 0, blobs.Count(), "a failed metadata write must not leak an orphan blob")
}
