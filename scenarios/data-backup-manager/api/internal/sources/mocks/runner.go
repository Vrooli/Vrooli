// Package mocks holds test doubles for the sources domain seams. Lives in a
// mocks/ directory (no _test.go suffix) so sibling _test.go files in other
// packages can import it; never linked into production.
package mocks

import (
	"context"

	"data-backup-manager/internal/sources"
)

// FakeCommandRunner satisfies sources.CommandRunner for tests. It records
// every call and returns programmable (stdout []byte, err error) pairs.
// When no result is programmed the fake returns (nil, nil).
type FakeCommandRunner struct {
	// Calls records every invocation in order.
	Calls []CommandCall
	// Results is indexed by call order. If a call index has no entry the
	// fake returns (nil, nil).
	Results []CommandResult
}

// CommandCall captures a single invocation.
type CommandCall struct {
	Name string
	Args []string
}

// CommandResult is the programmable output for a single call.
type CommandResult struct {
	Stdout []byte
	Err    error
}

// Compile-time guarantee.
var _ sources.CommandRunner = (*FakeCommandRunner)(nil)

// Run records the call and returns the pre-programmed result (or nil/nil if
// no result is configured for this call index).
func (f *FakeCommandRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	idx := len(f.Calls)
	f.Calls = append(f.Calls, CommandCall{Name: name, Args: args})
	if idx < len(f.Results) {
		return f.Results[idx].Stdout, f.Results[idx].Err
	}
	return nil, nil
}

// LastCall returns the most recently recorded call, panicking if no calls
// have been made yet (test assertion convenience).
func (f *FakeCommandRunner) LastCall() CommandCall {
	if len(f.Calls) == 0 {
		panic("FakeCommandRunner: no calls recorded")
	}
	return f.Calls[len(f.Calls)-1]
}
