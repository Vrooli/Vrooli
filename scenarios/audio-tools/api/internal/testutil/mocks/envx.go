package mocks

import (
	"sync"

	"audio-tools/internal/envx"
)

// FakeEnv is the testutil counterpart to envx.OS. It serves values from
// an in-memory map so tests can avoid t.Setenv (which races under
// t.Parallel) and assert exactly which keys were read.
type FakeEnv struct {
	mu     sync.Mutex
	values map[string]string
	reads  []string
}

// NewFakeEnv constructs a FakeEnv seeded with the provided values.
// Pass nil for an empty environment.
func NewFakeEnv(values map[string]string) *FakeEnv {
	cp := make(map[string]string, len(values))
	for k, v := range values {
		cp[k] = v
	}
	return &FakeEnv{values: cp}
}

// Get returns the configured value for key, or empty string if unset.
// Every call is recorded for assertions via Reads().
func (e *FakeEnv) Get(key string) string {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.reads = append(e.reads, key)
	return e.values[key]
}

// Set adds or overwrites a value in the fake environment.
func (e *FakeEnv) Set(key, value string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.values == nil {
		e.values = map[string]string{}
	}
	e.values[key] = value
}

// Reads returns a snapshot of keys that have been read in order.
func (e *FakeEnv) Reads() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, len(e.reads))
	copy(out, e.reads)
	return out
}

// Compile-time guarantee that *FakeEnv satisfies envx.Reader.
var _ envx.Reader = (*FakeEnv)(nil)
