package mocks

import (
	"context"
	"io"
	"strings"
	"sync"

	"audio-tools/internal/capabilities"
)

// FakeController is the lifecycle seam test double. It records each
// invocation in-order and returns canned errors keyed by method. Tests
// in handlers/provider_lifecycle and internal/capabilities both consume
// this fake so the implementation stays in lockstep with the interface.
type FakeController struct {
	mu           sync.Mutex
	StartCalls   []string
	StopCalls    []string
	RestartCalls []string
	PullCalls    []string
	LogCalls     []FakeLogCall
	StartErr     error
	StopErr      error
	RestartErr   error
	PullErr      error
	LogsErr      error
	LogsReader   io.ReadCloser
	LogsBody     string
}

// FakeLogCall records the arguments passed to FakeController.Logs.
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
	if f.LogsBody != "" {
		return io.NopCloser(strings.NewReader(f.LogsBody)), nil
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
