// Package mocks holds the hoisted FakeChecker test fake for the
// capabilities domain. capabilities.Checker is a one-method seam
// (`Check(ctx) (Status, string)`); this fake records call counts and
// returns canned (Status, Message) pairs so registry tests can assert
// caching and fan-out behavior.
package mocks

import (
	"context"
	"sync/atomic"

	"audio-tools/internal/capabilities"
)

// FakeChecker satisfies capabilities.Checker. Tests configure Status
// and Message; Calls reports the number of invocations as a typed
// atomic counter so parallel tests stay race-clean.
type FakeChecker struct {
	Status  capabilities.Status
	Message string
	Calls   atomic.Int64
}

// NewFakeChecker constructs a FakeChecker pre-seeded with status and
// message.
func NewFakeChecker(status capabilities.Status, message string) *FakeChecker {
	return &FakeChecker{Status: status, Message: message}
}

// Check increments Calls and returns the configured pair.
func (f *FakeChecker) Check(_ context.Context) (capabilities.Status, string) {
	f.Calls.Add(1)
	return f.Status, f.Message
}

// CallCount returns the recorded invocation count.
func (f *FakeChecker) CallCount() int64 { return f.Calls.Load() }

// ResetCalls zeroes the counter for sub-test reuse.
func (f *FakeChecker) ResetCalls() { f.Calls.Store(0) }

// Compile-time guarantee that *FakeChecker satisfies capabilities.Checker.
var _ capabilities.Checker = (*FakeChecker)(nil)
