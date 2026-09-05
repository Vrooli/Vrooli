// Package mocks provides the test double for the kexec.Runner seam. It records
// every Call (argv + env overlay) so unit tests can assert on the kopia argv
// the resource would have produced — without a real kopia binary.
package mocks

import (
	"context"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
)

// FakeRunner records calls and returns canned output keyed by a matcher, so
// tests can both assert on produced argv and simulate kopia responses.
type FakeRunner struct {
	// Calls captures every invocation in order.
	Calls []kexec.Call
	// Output is returned from Run when Responder is nil.
	Output []byte
	// Err is returned from Run when Responder is nil.
	Err error
	// Responder, when set, computes the response per-call (e.g. to return
	// different JSON for `snapshot create` vs `snapshot list`).
	Responder func(c kexec.Call) ([]byte, error)
}

var _ kexec.Runner = (*FakeRunner)(nil)

// Run records the call and returns the configured response.
func (f *FakeRunner) Run(_ context.Context, c kexec.Call) ([]byte, error) {
	f.Calls = append(f.Calls, c)
	if f.Responder != nil {
		return f.Responder(c)
	}
	return f.Output, f.Err
}

// LastCall returns the most recent recorded call and whether one exists.
func (f *FakeRunner) LastCall() (kexec.Call, bool) {
	if len(f.Calls) == 0 {
		return kexec.Call{}, false
	}
	return f.Calls[len(f.Calls)-1], true
}

// AllArgs returns the argv of every recorded call, for invariant scans.
func (f *FakeRunner) AllArgs() [][]string {
	out := make([][]string, 0, len(f.Calls))
	for _, c := range f.Calls {
		out = append(out, c.Args)
	}
	return out
}
