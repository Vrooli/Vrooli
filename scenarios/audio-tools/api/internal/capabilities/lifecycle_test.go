package capabilities_test

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"audio-tools/internal/capabilities"

	"github.com/stretchr/testify/require"
)

// FakeController is the lifecycle seam test double. It records each
// invocation in-order and returns canned errors keyed by method. Tests
// in handlers/provider_lifecycle wire this fake; we keep the
// implementation here so it lives next to its production counterpart
// and stays in lockstep with the interface.
type FakeController struct {
	mu              sync.Mutex
	StartCalls      []string
	StopCalls       []string
	RestartCalls    []string
	PullCalls       []string
	LogCalls        []FakeLogCall
	StartErr        error
	StopErr         error
	RestartErr      error
	PullErr         error
	LogsErr         error
	LogsReader      io.ReadCloser
}

type FakeLogCall struct {
	Slug      string
	Follow    bool
	TailLines int
}

func (f *FakeController) Start(_ context.Context, slug string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StartCalls = append(f.StartCalls, slug)
	return f.StartErr
}

func (f *FakeController) Stop(_ context.Context, slug string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.StopCalls = append(f.StopCalls, slug)
	return f.StopErr
}

func (f *FakeController) Restart(_ context.Context, slug string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.RestartCalls = append(f.RestartCalls, slug)
	return f.RestartErr
}

func (f *FakeController) Logs(_ context.Context, slug string, follow bool, tailLines int) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.LogCalls = append(f.LogCalls, FakeLogCall{Slug: slug, Follow: follow, TailLines: tailLines})
	if f.LogsErr != nil {
		return nil, f.LogsErr
	}
	if f.LogsReader != nil {
		return f.LogsReader, nil
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (f *FakeController) PullModel(_ context.Context, model string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.PullCalls = append(f.PullCalls, model)
	return f.PullErr
}

// Counts is a convenience snapshot for handler tests asserting "exactly
// one Start call" etc.
func (f *FakeController) Counts() (startN, stopN, restartN, pullN int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.StartCalls), len(f.StopCalls), len(f.RestartCalls), len(f.PullCalls)
}

// Compile-time guarantee that *FakeController satisfies the seam.
var _ capabilities.ResourceController = (*FakeController)(nil)

func TestResourceSlugForProviderID(t *testing.T) {
	cases := []struct {
		id   string
		slug string
		ok   bool
	}{
		{"whisper-stt", "whisper", true},
		{"kokoro-tts", "kokoro", true},
		{"speaker-verification", "speaker-verification", true},
		{"ollama", "ollama", true},
		{"openrouter", "", false},
		{"audio-tools", "", false},
		{"unknown", "", false},
	}
	for _, tc := range cases {
		got, ok := capabilities.ResourceSlugForProviderID(tc.id)
		require.Equal(t, tc.ok, ok, "id=%q", tc.id)
		require.Equal(t, tc.slug, got, "id=%q", tc.id)
	}
}

func TestSupportsPullModel(t *testing.T) {
	require.True(t, capabilities.SupportsPullModel("ollama"))
	require.False(t, capabilities.SupportsPullModel("whisper-stt"))
	require.False(t, capabilities.SupportsPullModel("openrouter"))
}

func TestCLIController_NoBinary_ReturnsUnavailable(t *testing.T) {
	// We can't reliably unset PATH in a unit test that may itself need
	// to spawn helpers; instead instantiate a controller with no
	// resolved binary and assert the sentinel.
	c := &capabilities.CLIController{}
	err := c.Start(context.Background(), "whisper")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	err = c.Stop(context.Background(), "whisper")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	err = c.Restart(context.Background(), "whisper")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	err = c.PullModel(context.Background(), "phi3")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	_, err = c.Logs(context.Background(), "whisper", false, 0)
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
}
