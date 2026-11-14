package mocks

import (
	"fmt"
	"sync"

	"client-rehearsal/internal/logx"
)

// FakeLogger records formatted log lines for deterministic API tests.
type FakeLogger struct {
	mu    sync.Mutex
	Lines []string
}

func (f *FakeLogger) Printf(format string, values ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Lines = append(f.Lines, fmt.Sprintf(format, values...))
}

func (f *FakeLogger) Fatalf(format string, values ...any) {
	f.Printf(format, values...)
}

var _ logx.Logger = (*FakeLogger)(nil)
