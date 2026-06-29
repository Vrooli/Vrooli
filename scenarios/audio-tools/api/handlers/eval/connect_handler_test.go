package eval

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	intcorpus "audio-tools/internal/corpus"
	localdb "audio-tools/internal/database"
	"audio-tools/internal/logx"
	"audio-tools/internal/stt"
	"audio-tools/internal/testutil/db"
	"audio-tools/internal/testutil/mocks"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

type memBlobs struct{ m map[string][]byte }

func (b *memBlobs) Put(_ context.Context, key string, data []byte, _ string) error {
	b.m[key] = append([]byte(nil), data...)
	return nil
}

func (b *memBlobs) Get(_ context.Context, key string) ([]byte, error) {
	return append([]byte(nil), b.m[key]...), nil
}
func (b *memBlobs) Delete(_ context.Context, key string) error { delete(b.m, key); return nil }

func newCorpusWithClip(t *testing.T, reference string, pcmLen int) *intcorpus.Service {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(intcorpus.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	svc := intcorpus.NewService(intcorpus.NewSQLiteRepository(d, clk), &memBlobs{m: map[string][]byte{}}, clk)
	_, err := svc.CreateClip(context.Background(), intcorpus.CreateClipInput{
		Audio:         make([]byte, pcmLen),
		ReferenceText: reference,
		SampleRateHz:  16000,
		Format:        "pcm_s16le",
	})
	require.NoError(t, err)
	return svc
}

// fakeProvider returns the canned text for every Transcribe call.
func fakeProvider(text string) sttchain.Provider {
	p := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true})
	p.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: text, Tier: sttchain.TierLocal, Latency: time.Millisecond}, nil
	}
	return p
}

func TestRunEval_BatchAndOverlapOverFakeProvider(t *testing.T) {
	// 500ms of PCM so overlap-agree gets several settle attempts.
	corpus := newCorpusWithClip(t, "hello world", 16000*2/2)
	h := NewConnectHandler(Deps{
		Logger:      logx.Std{},
		Clock:       mocks.NewFakeClock(time.Now()),
		Corpus:      corpus,
		NewProvider: func() sttchain.Provider { return fakeProvider("hello world") },
		Defaults:    stt.Defaults(),
	})

	resp, err := h.RunEval(context.Background(), connect.NewRequest(&evalv1.RunEvalRequest{
		Strategies: []*evalv1.EvalStrategy{
			{Kind: "batch", OverlapMaxStallRejects: -1},
			{Kind: "overlap_agree", OverlapMaxStallRejects: -1},
		},
	}))
	require.NoError(t, err)
	report := resp.Msg.GetReport()
	require.True(t, report.GetQualityMeasured())
	require.False(t, report.GetLatencyMeasured())
	require.Len(t, report.GetPerStrategy(), 2)

	for _, s := range report.GetPerStrategy() {
		require.InDelta(t, 0.0, s.GetWer(), 1e-9, "fake provider returns the exact reference -> zero WER (%s)", s.GetLabel())
		require.GreaterOrEqual(t, s.GetWhisperCalls(), int32(1), "calls metered for %s", s.GetLabel())
		require.Len(t, s.GetPerClip(), 1)
	}
}

func TestRunEval_NoProviderFailsPrecondition(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger: logx.Std{}, Clock: mocks.NewFakeClock(time.Now()),
		Corpus: newCorpusWithClip(t, "x", 100), NewProvider: nil,
	})
	_, err := h.RunEval(context.Background(), connect.NewRequest(&evalv1.RunEvalRequest{}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestRunEval_UnknownStrategyRejected(t *testing.T) {
	h := NewConnectHandler(Deps{
		Logger: logx.Std{}, Clock: mocks.NewFakeClock(time.Now()),
		Corpus:      newCorpusWithClip(t, "x", 100),
		NewProvider: func() sttchain.Provider { return fakeProvider("x") },
		Defaults:    stt.Defaults(),
	})
	_, err := h.RunEval(context.Background(), connect.NewRequest(&evalv1.RunEvalRequest{
		Strategies: []*evalv1.EvalStrategy{{Kind: "nonsense"}},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
