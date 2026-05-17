package mocks

import (
	"fmt"
	"sync"

	"audio-tools/internal/logx"
)

// FakeLogger records every Printf call so tests can assert what the
// system under test logged without writing to stderr or scraping output.
type FakeLogger struct {
	mu      sync.Mutex
	entries []string
}

// NewFakeLogger constructs an empty FakeLogger.
func NewFakeLogger() *FakeLogger { return &FakeLogger{} }

// Printf formats the line with fmt.Sprintf and records it.
func (l *FakeLogger) Printf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, fmt.Sprintf(format, args...))
}

// Entries returns a snapshot of recorded log lines.
func (l *FakeLogger) Entries() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, len(l.entries))
	copy(out, l.entries)
	return out
}

// Reset clears recorded entries (useful between sub-tests).
func (l *FakeLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = nil
}

// Compile-time guarantee that *FakeLogger satisfies logx.Logger.
var _ logx.Logger = (*FakeLogger)(nil)
