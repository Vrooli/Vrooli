package eval

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
)

// s16leMonoBytesPerSecond is the canonical PCM byte rate the strategies
// feed the backend: 16 kHz * 2 bytes/sample * 1 channel.
const s16leMonoBytesPerSecond = 16000 * 2

func TestMeteredProvider_CountsCallsAudioAndLatency(t *testing.T) {
	inner := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true})
	inner.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "x", Latency: 40 * time.Millisecond}, nil
	}
	m := NewMeteredProvider(inner, s16leMonoBytesPerSecond)

	// Two calls: 1s and 0.5s of PCM.
	oneSecond := make([]byte, s16leMonoBytesPerSecond)
	halfSecond := make([]byte, s16leMonoBytesPerSecond/2)
	_, err := m.Transcribe(context.Background(), sttchain.Request{Audio: oneSecond})
	require.NoError(t, err)
	_, err = m.Transcribe(context.Background(), sttchain.Request{Audio: halfSecond})
	require.NoError(t, err)

	snap := m.Snapshot()
	require.Equal(t, 2, snap.Calls)
	require.InDelta(t, 1.5, snap.AudioSeconds, 1e-9, "audio-seconds summed from PCM byte length")
	require.Equal(t, 80*time.Millisecond, snap.ProviderLatency, "provider-reported latency summed")
}

func TestMeteredProvider_DelegatesAndResets(t *testing.T) {
	inner := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true, Stream: false})
	inner.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "ok"}, nil
	}
	m := NewMeteredProvider(inner, s16leMonoBytesPerSecond)

	// Delegation: Type/Traits pass through.
	require.Equal(t, sttchain.TierLocal, m.Type())
	require.True(t, m.Traits().Batch)

	_, _ = m.Transcribe(context.Background(), sttchain.Request{Audio: make([]byte, s16leMonoBytesPerSecond)})
	require.Equal(t, 1, m.Snapshot().Calls)
	m.Reset()
	require.Equal(t, MeterSnapshot{}, m.Snapshot(), "reset zeroes all meters")
}

// TestMeteredProvider_ZeroBytesPerSecondSkipsAudioAccounting proves the
// encoded-bytes batch path can disable byte-derived audio-seconds.
func TestMeteredProvider_ZeroBytesPerSecondSkipsAudioAccounting(t *testing.T) {
	inner := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true})
	inner.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "x", Latency: time.Millisecond}, nil
	}
	m := NewMeteredProvider(inner, 0)
	_, _ = m.Transcribe(context.Background(), sttchain.Request{Audio: make([]byte, 9999)})
	snap := m.Snapshot()
	require.Equal(t, 1, snap.Calls)
	require.Equal(t, 0.0, snap.AudioSeconds, "byte-derived audio-seconds disabled when bytesPerSecond=0")
}
