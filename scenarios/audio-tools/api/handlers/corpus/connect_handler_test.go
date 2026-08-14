package corpus

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	intcorpus "audio-tools/internal/corpus"
	localdb "audio-tools/internal/database"
	"audio-tools/internal/logx"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	corpusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/corpus"
)

type memBlobs struct{ m map[string][]byte }

func (b *memBlobs) Put(_ context.Context, key string, data []byte, _ string) error {
	b.m[key] = append([]byte(nil), data...)
	return nil
}

func (b *memBlobs) Get(_ context.Context, key string) ([]byte, error) {
	d, ok := b.m[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return append([]byte(nil), d...), nil
}

func (b *memBlobs) Delete(_ context.Context, key string) error { delete(b.m, key); return nil }

func newHandler(t *testing.T) *connectHandler {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(intcorpus.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	svc := intcorpus.NewService(intcorpus.NewSQLiteRepository(d, clk), &memBlobs{m: map[string][]byte{}}, clk)
	return NewConnectHandler(Deps{Logger: logx.Std{}, Clock: clk, Service: svc})
}

func TestConnectHandler_ClipRoundTrip(t *testing.T) {
	ctx := context.Background()
	h := newHandler(t)

	created, err := h.CreateClip(ctx, connect.NewRequest(&corpusv1.CreateClipRequest{
		Audio:         []byte{1, 2, 3, 4},
		ReferenceText: "hello world",
		Tags:          []string{"smoke"},
		SampleRateHz:  16000,
		Format:        "pcm_s16le",
		Source:        corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED,
	}))
	require.NoError(t, err)
	id := created.Msg.GetClip().GetId()
	require.NotEmpty(t, id)
	require.Equal(t, corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED, created.Msg.GetClip().GetSource())

	list, err := h.ListClips(ctx, connect.NewRequest(&corpusv1.ListClipsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetClips(), 1)

	audio, err := h.GetClipAudio(ctx, connect.NewRequest(&corpusv1.GetClipAudioRequest{Id: id}))
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, audio.Msg.GetAudio())

	_, err = h.DeleteClip(ctx, connect.NewRequest(&corpusv1.DeleteClipRequest{Id: id}))
	require.NoError(t, err)

	_, err = h.GetClip(ctx, connect.NewRequest(&corpusv1.GetClipRequest{Id: id}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "deleted clip is not found")
}

func TestConnectHandler_CreateRequiresAudio(t *testing.T) {
	h := newHandler(t)
	_, err := h.CreateClip(context.Background(), connect.NewRequest(&corpusv1.CreateClipRequest{ReferenceText: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnectHandler_NoServiceFailsPrecondition(t *testing.T) {
	clk := scheduletest.New(time.Now())
	h := NewConnectHandler(Deps{Logger: logx.Std{}, Clock: clk, Service: nil})
	_, err := h.ListClips(context.Background(), connect.NewRequest(&corpusv1.ListClipsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
