// Package mocks holds in-memory fakes for the git seam.
package mocks

import (
	"context"
	"sync/atomic"

	"architecture-cartographer/internal/git"
)

// FakeRunner is the in-memory git.Runner.
type FakeRunner struct {
	Available bool
	LogOutput string
	LogErr    error

	LogCalls atomic.Int64
}

func (f *FakeRunner) IsAvailable(_ context.Context) bool { return f.Available }
func (f *FakeRunner) Log(_ context.Context, _ ...string) (string, error) {
	f.LogCalls.Add(1)
	if f.LogErr != nil {
		return "", f.LogErr
	}
	return f.LogOutput, nil
}

var _ git.Runner = (*FakeRunner)(nil)
