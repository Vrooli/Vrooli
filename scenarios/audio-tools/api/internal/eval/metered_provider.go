package eval

import (
	"context"
	"sync"
	"time"

	"audio-tools/internal/ai/sttchain"
)

// MeteredProvider is a sttchain.Provider decorator that instruments the
// Transcribe path: it counts how many backend calls a strategy made, sums
// the input-audio-seconds it asked the backend to process, and sums the
// provider-reported latency. These are the compute-cost instruments the
// harness needs to compute RTF and the "Whisper-calls / audio-seconds"
// columns of the comparison report.
//
// It is a pure decorator — no behavior changes, every call delegates to
// the wrapped provider — so it can wrap the real LocalProviderWith in the
// replay harness without altering what the strategy sees. seam: this is a
// thin wrapper over the existing sttchain.Provider seam, adding no new
// core dependency.
//
// Audio-seconds are derived from the request byte length using
// bytesPerSecond (s16le mono 16 kHz = 32000). Strategies that consume
// canonical PCM (VADSegment, OverlapAgree) always hand the backend PCM, so
// this is exact for them. For an encoded-bytes batch path set
// bytesPerSecond=0 to disable byte-derived accounting and let the caller
// attribute audio-seconds from the known clip duration instead.
type MeteredProvider struct {
	inner          sttchain.Provider
	bytesPerSecond float64

	mu              sync.Mutex
	calls           int
	audioSeconds    float64
	providerLatency time.Duration
}

// NewMeteredProvider wraps inner. bytesPerSecond is the PCM byte rate used
// to derive audio-seconds from each request's byte length; pass 0 to skip
// byte-derived audio-second accounting (calls + latency are still counted).
func NewMeteredProvider(inner sttchain.Provider, bytesPerSecond float64) *MeteredProvider {
	return &MeteredProvider{inner: inner, bytesPerSecond: bytesPerSecond}
}

// MeterSnapshot is an immutable read of the accumulated meters.
type MeterSnapshot struct {
	Calls           int
	AudioSeconds    float64
	ProviderLatency time.Duration
}

// Snapshot returns the current meter values.
func (m *MeteredProvider) Snapshot() MeterSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return MeterSnapshot{
		Calls:           m.calls,
		AudioSeconds:    m.audioSeconds,
		ProviderLatency: m.providerLatency,
	}
}

// Reset zeroes the meters (so one provider instance can be reused across
// clips when the harness attributes per-clip totals via deltas).
func (m *MeteredProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = 0
	m.audioSeconds = 0
	m.providerLatency = 0
}

// Transcribe is the instrumented path: it records one call, the request's
// audio-seconds, and the result's provider-reported latency, then returns
// the wrapped provider's result verbatim.
func (m *MeteredProvider) Transcribe(ctx context.Context, req sttchain.Request) (*sttchain.Result, error) {
	res, err := m.inner.Transcribe(ctx, req)
	m.mu.Lock()
	m.calls++
	if m.bytesPerSecond > 0 {
		m.audioSeconds += float64(len(req.Audio)) / m.bytesPerSecond
	}
	if res != nil {
		m.providerLatency += res.Latency
	}
	m.mu.Unlock()
	return res, err
}

// The remaining Provider methods delegate unchanged.

func (m *MeteredProvider) Type() sttchain.ProviderTier          { return m.inner.Type() }
func (m *MeteredProvider) IsAvailable(ctx context.Context) bool { return m.inner.IsAvailable(ctx) }
func (m *MeteredProvider) Model() string                        { return m.inner.Model() }
func (m *MeteredProvider) Traits() sttchain.ProviderTraits      { return m.inner.Traits() }

func (m *MeteredProvider) TranscribeStreaming(ctx context.Context, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
	return m.inner.TranscribeStreaming(ctx, start, chunks)
}

// compile-time assertion that the decorator stays a full Provider.
var _ sttchain.Provider = (*MeteredProvider)(nil)
