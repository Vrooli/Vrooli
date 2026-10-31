package sttchain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeLocalEngine is a minimal Provider used to assert engine-id resolution
// without standing up real backends.
type fakeLocalEngine struct {
	id        string
	available bool
}

func (f *fakeLocalEngine) Type() ProviderTier               { return TierLocal }
func (f *fakeLocalEngine) IsAvailable(context.Context) bool { return f.available }
func (f *fakeLocalEngine) Model() string                    { return f.id }
func (f *fakeLocalEngine) Traits() ProviderTraits {
	return ProviderTraits{Stream: true, Strategies: []StrategyKind{StrategyPassthrough}}
}

func (f *fakeLocalEngine) Transcribe(context.Context, Request) (*Result, error) { return nil, nil }
func (f *fakeLocalEngine) TranscribeStreaming(context.Context, StreamStart, <-chan AudioChunk) (<-chan StreamEvent, error) {
	return nil, nil
}

func TestResolveLocalEngine_ByEngineID(t *testing.T) {
	whisper := &fakeLocalEngine{id: "whisper-local", available: true}
	kyutai := &fakeLocalEngine{id: "kyutai", available: true}
	c := &Chain{
		localEngines: map[string]Provider{"whisper-local": whisper, "kyutai": kyutai},
		enableLocal:  true,
	}

	require.Equal(t, Provider(kyutai), c.resolveLocalEngine("kyutai"))
	require.Equal(t, Provider(whisper), c.resolveLocalEngine("whisper-local"))
	// Unknown / empty id falls back to the default local provider (c.local).
	c.local = NewLocalProvider(nil)
	require.Equal(t, Provider(c.local), c.resolveLocalEngine(""))
	require.Equal(t, Provider(c.local), c.resolveLocalEngine("does-not-exist"))
}

func TestStreamCandidates_PicksEngineSpecificLocalProvider(t *testing.T) {
	whisper := &fakeLocalEngine{id: "whisper-local", available: true}
	kyutai := &fakeLocalEngine{id: "kyutai", available: true}
	c := &Chain{
		localEngines: map[string]Provider{"whisper-local": whisper, "kyutai": kyutai},
		enableLocal:  true,
	}

	got := c.StreamCandidates(context.Background(), StreamStart{EngineID: "kyutai"})
	require.Len(t, got, 1)
	require.Equal(t, "kyutai", got[0].Model())

	got = c.StreamCandidates(context.Background(), StreamStart{EngineID: "whisper-local"})
	require.Len(t, got, 1)
	require.Equal(t, "whisper-local", got[0].Model())
}

func TestStreamCandidates_DropsUnavailableEngine(t *testing.T) {
	kyutai := &fakeLocalEngine{id: "kyutai", available: false} // resource down
	c := &Chain{
		localEngines: map[string]Provider{"kyutai": kyutai},
		enableLocal:  true,
	}
	got := c.StreamCandidates(context.Background(), StreamStart{EngineID: "kyutai"})
	require.Empty(t, got, "an unavailable engine must not be a candidate")
}

func TestLocalEngineAvailable(t *testing.T) {
	kyutai := &fakeLocalEngine{id: "kyutai", available: true}
	c := &Chain{localEngines: map[string]Provider{"kyutai": kyutai}, enableLocal: true}
	require.True(t, c.LocalEngineAvailable(context.Background(), "kyutai"))

	c.enableLocal = false
	require.False(t, c.LocalEngineAvailable(context.Background(), "kyutai"), "disabled local tier is never available")
}
