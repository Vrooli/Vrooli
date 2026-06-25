package mocks

import (
	"context"
	"sync"
)

// FakeWaiter is a controllable orchestration.Waiter for await-registry tests.
// Wait blocks until the test signals a result (Resolve), an error (Fail), or the
// context is done — so tests can exercise resolve / producer-error / deadline /
// cancel paths deterministically without shelling real producer CLIs.
type FakeWaiter struct {
	ProducerName string

	mu       sync.Mutex
	resultCh chan waiterResult
	calls    []string // keys passed to Wait, in order
}

type waiterResult struct {
	result string
	err    error
}

// NewFakeWaiter builds a fake for the given producer key.
func NewFakeWaiter(producer string) *FakeWaiter {
	return &FakeWaiter{
		ProducerName: producer,
		resultCh:     make(chan waiterResult, 1),
	}
}

// Producer returns the producer key this fake answers for.
func (f *FakeWaiter) Producer() string { return f.ProducerName }

// Wait records the key and blocks until Resolve/Fail is called or ctx is done.
func (f *FakeWaiter) Wait(ctx context.Context, key string) (string, error) {
	f.mu.Lock()
	f.calls = append(f.calls, key)
	ch := f.resultCh
	f.mu.Unlock()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.result, res.err
	}
}

// Resolve unblocks a pending Wait with a successful result.
func (f *FakeWaiter) Resolve(result string) {
	f.resultCh <- waiterResult{result: result}
}

// Fail unblocks a pending Wait with a producer-side error.
func (f *FakeWaiter) Fail(err error) {
	f.resultCh <- waiterResult{err: err}
}

// Calls returns the keys passed to Wait so far.
func (f *FakeWaiter) Calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}
