//go:build audio_lab

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
		Audio:            []byte{1, 2, 3, 4},
		ReferenceText:    "hello world",
		Tags:             []string{"smoke"},
		SampleRateHz:     16000,
		Format:           "pcm_s16le",
		Source:           corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED,
		CorpusRevision:   "voice-pilot-en-v1",
		LicenseReference: "license-1",
		SpeakerPseudonym: "speaker-01",
		Split:            corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT,
		NoiseCondition:   corpusv1.NoiseCondition_NOISE_CONDITION_NOISY,
		SnrDb:            10,
		TimedReference: &corpusv1.TimedReference{
			Words:      []*corpusv1.TimedWord{{Word: "hello", StartSeconds: 0.1, EndSeconds: 0.3}, {Word: "world", StartSeconds: 0.31, EndSeconds: 0.6}},
			Provenance: &corpusv1.TimingProvenance{Status: "human_reviewed", SourceReference: "source-1", AlignmentTool: "aligner", AlignmentModel: "model-1", Reviewer: "reviewer-1", ReviewMethod: "listen-and-correct", ReviewedAt: "2026-06-29T12:00:00Z"},
		},
		ReferenceReview: &corpusv1.ReferenceReview{Reviewer: "reviewer-1", Protocol: "listen-and-correct-v1", ReviewedAt: "2026-06-29T12:00:00Z"},
	}))
	require.NoError(t, err)
	id := created.Msg.GetClip().GetId()
	require.NotEmpty(t, id)
	require.Equal(t, corpusv1.ClipSource_CLIP_SOURCE_SCRIPTED, created.Msg.GetClip().GetSource())
	require.Equal(t, "voice-pilot-en-v1", created.Msg.GetClip().GetCorpusRevision())
	require.Equal(t, corpusv1.CorpusSplit_CORPUS_SPLIT_HELD_OUT, created.Msg.GetClip().GetSplit())
	require.Equal(t, corpusv1.NoiseCondition_NOISE_CONDITION_NOISY, created.Msg.GetClip().GetNoiseCondition())
	require.Equal(t, float64(10), created.Msg.GetClip().GetSnrDb())
	require.Len(t, created.Msg.GetClip().GetContentSha256(), 64)
	require.Equal(t, "reviewer-1", created.Msg.GetClip().GetTimedReference().GetProvenance().GetReviewer())
	require.Equal(t, "reviewer-1", created.Msg.GetClip().GetReferenceReview().GetReviewer())
	require.Equal(t, intcorpus.ReferenceDigest("hello world"), created.Msg.GetClip().GetReferenceReview().GetReferenceSha256())

	list, err := h.ListClips(ctx, connect.NewRequest(&corpusv1.ListClipsRequest{}))
	require.NoError(t, err)
	require.Len(t, list.Msg.GetClips(), 1)
	require.Equal(t, created.Msg.GetClip().GetTimedReference(), list.Msg.GetClips()[0].GetTimedReference())
	require.Equal(t, created.Msg.GetClip().GetReferenceReview(), list.Msg.GetClips()[0].GetReferenceReview())
	got, err := h.GetClip(ctx, connect.NewRequest(&corpusv1.GetClipRequest{Id: id}))
	require.NoError(t, err)
	require.Equal(t, created.Msg.GetClip().GetTimedReference(), got.Msg.GetClip().GetTimedReference())

	audio, err := h.GetClipAudio(ctx, connect.NewRequest(&corpusv1.GetClipAudioRequest{Id: id}))
	require.NoError(t, err)
	require.Equal(t, []byte{1, 2, 3, 4}, audio.Msg.GetAudio())

	_, err = h.DeleteClip(ctx, connect.NewRequest(&corpusv1.DeleteClipRequest{Id: id}))
	require.NoError(t, err)

	_, err = h.GetClip(ctx, connect.NewRequest(&corpusv1.GetClipRequest{Id: id}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "deleted clip is not found")
}

func TestConnectHandler_RejectsInvalidTimedReference(t *testing.T) {
	h := newHandler(t)
	_, err := h.CreateClip(context.Background(), connect.NewRequest(&corpusv1.CreateClipRequest{
		Audio: []byte{1}, DurationMs: 100, Format: "pcm_s16le",
		TimedReference: &corpusv1.TimedReference{Words: []*corpusv1.TimedWord{{Word: "late", StartSeconds: .2, EndSeconds: .3}}, Provenance: &corpusv1.TimingProvenance{Status: "human_reviewed", SourceReference: "src", AlignmentTool: "tool", AlignmentModel: "model", Reviewer: "r", ReviewMethod: "listen", ReviewedAt: "bad"}},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnectHandler_CreateRequiresAudio(t *testing.T) {
	h := newHandler(t)
	_, err := h.CreateClip(context.Background(), connect.NewRequest(&corpusv1.CreateClipRequest{ReferenceText: "x"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestConnectHandler_ReviewReferenceRequiresHumanFieldsAndRejectsStaleForms(t *testing.T) {
	h := newHandler(t)
	created, err := h.CreateClip(context.Background(), connect.NewRequest(&corpusv1.CreateClipRequest{Audio: []byte{1, 2}, ReferenceText: "old reference", Format: "pcm_s16le"}))
	require.NoError(t, err)
	id := created.Msg.GetClip().GetId()
	_, err = h.ReviewReference(context.Background(), connect.NewRequest(&corpusv1.ReviewReferenceRequest{Id: id, ExpectedReferenceSha256: intcorpus.ReferenceDigest("old reference"), ReferenceText: "corrected reference"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	saved, err := h.ReviewReference(context.Background(), connect.NewRequest(&corpusv1.ReviewReferenceRequest{Id: id, ExpectedReferenceSha256: intcorpus.ReferenceDigest("old reference"), ReferenceText: "corrected reference", Reviewer: "human-1", Protocol: "listen-v1"}))
	require.NoError(t, err)
	require.Equal(t, "corrected reference", saved.Msg.GetClip().GetReferenceText())
	require.Equal(t, intcorpus.ReferenceDigest("corrected reference"), saved.Msg.GetClip().GetReferenceReview().GetReferenceSha256())
	require.Equal(t, "human-1", saved.Msg.GetClip().GetReferenceReview().GetReviewer())
	require.NotEmpty(t, saved.Msg.GetClip().GetReferenceReview().GetReviewedAt())

	_, err = h.ReviewReference(context.Background(), connect.NewRequest(&corpusv1.ReviewReferenceRequest{Id: id, ExpectedReferenceSha256: intcorpus.ReferenceDigest("old reference"), ReferenceText: "stale write", Reviewer: "human-2", Protocol: "listen-v1"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
}

func TestConnectHandler_NoServiceFailsPrecondition(t *testing.T) {
	clk := scheduletest.New(time.Now())
	h := NewConnectHandler(Deps{Logger: logx.Std{}, Clock: clk, Service: nil})
	_, err := h.ListClips(context.Background(), connect.NewRequest(&corpusv1.ListClipsRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
