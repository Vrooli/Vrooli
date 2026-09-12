// Package mocks holds test doubles for the runnability seam. It carries no
// _test.go suffix so sibling packages' tests can import it, and lives in a
// mocks/ directory so the production-import guard exempts it.
package mocks

import "test-genie/internal/orchestrator/runnability"

// FakeResolver is a programmable runnability.Resolver. Tests set Func to drive
// per-call decisions, or set Verdict for a constant answer; calls are recorded.
type FakeResolver struct {
	// Func, when set, computes the verdict for each call.
	Func func(caps runnability.PhaseCapabilities, rc runnability.RunContext) runnability.Verdict
	// Verdict is the constant returned when Func is nil.
	Verdict runnability.Verdict
	// Calls records every (caps, ctx) the resolver was asked about, in order.
	Calls []FakeCall
}

// FakeCall captures one Resolve invocation.
type FakeCall struct {
	Caps    runnability.PhaseCapabilities
	Context runnability.RunContext
}

// Resolve records the call and returns the programmed verdict.
func (f *FakeResolver) Resolve(caps runnability.PhaseCapabilities, rc runnability.RunContext) runnability.Verdict {
	f.Calls = append(f.Calls, FakeCall{Caps: caps, Context: rc})
	if f.Func != nil {
		return f.Func(caps, rc)
	}
	return f.Verdict
}

var _ runnability.Resolver = (*FakeResolver)(nil)
