// Package mocks provides deterministic claim-domain test doubles.
package mocks

import (
	"context"
	"sync"

	"content-desk/internal/claims"
)

// FakeRunner records checks and returns arranged results without spawning a process.
type FakeRunner struct {
	mu     sync.Mutex
	Checks []claims.EvidenceCheck
	Result claims.CheckResult
	Err    error
}

var _ claims.Runner = (*FakeRunner)(nil)

func (f *FakeRunner) Run(_ context.Context, check claims.EvidenceCheck) (claims.CheckResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Checks = append(f.Checks, check)
	return f.Result, f.Err
}
