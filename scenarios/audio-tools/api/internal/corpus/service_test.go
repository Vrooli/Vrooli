package corpus_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/corpus"
	"audio-tools/internal/testutil/mocks"
)

// memBlobs is an in-memory BlobBytes for service tests.
type memBlobs struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMemBlobs() *memBlobs { return &memBlobs{m: map[string][]byte{}} }

func (b *memBlobs) Put(_ context.Context, key string, data []byte, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.m[key] = append([]byte(nil), data...)
	return nil
}

func (b *memBlobs) Get(_ context.Context, key string) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	d, ok := b.m[key]
	if !ok {
		return nil, errors.New("blob not found")
	}
	return append([]byte(nil), d...), nil
}

func (b *memBlobs) Delete(_ context.Context, key string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.m, key)
	return nil
}

func (b *memBlobs) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.m)
}

func newService(t *testing.T) (*corpus.Service, *memBlobs) {
	t.Helper()
	clk := mocks.NewFakeClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	repo := corpus.NewSQLiteRepository(newSchemaDB(t), clk)
	blobs := newMemBlobs()
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
	require.Equal(t, 1, blobs.count(), "audio bytes landed in the blob store")

	gotAudio, gotClip, err := svc.GetClipAudio(ctx, clip.ID)
	require.NoError(t, err)
	require.Equal(t, audio, gotAudio, "audio round-trips byte-for-byte")
	require.Equal(t, "hello world", gotClip.ReferenceText)

	require.NoError(t, svc.DeleteClip(ctx, clip.ID))
	require.Equal(t, 0, blobs.count(), "delete removes the blob too")
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
	blobs := newMemBlobs()
	svc := corpus.NewService(failingRepo{}, blobs, clk)

	_, err := svc.CreateClip(context.Background(), corpus.CreateClipInput{
		Audio: []byte{1, 2, 3}, ReferenceText: "x", Format: "pcm_s16le",
	})
	require.Error(t, err)
	require.Equal(t, 0, blobs.count(), "a failed metadata write must not leak an orphan blob")
}
