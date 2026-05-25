// Package mocks holds test doubles for the sidecar seam.
//
// Domain packages (internal/graph, internal/rewrite) wire
// FakeSidecarClient in unit tests; only the supervisor integration
// tests in the sidecar package itself spawn a real Node child.
package mocks

import (
	"context"
	"sync"

	"typescript-code-graph/internal/sidecar"
)

// FakeSidecarClient is a programmable double satisfying
// sidecar.SidecarClient. Set the Extract/RewriteApply/Shutdown fields
// to functions matching the desired test behavior.
type FakeSidecarClient struct {
	mu sync.Mutex

	ExtractFn      func(ctx context.Context, scenarioPath string) (sidecar.ExtractResult, error)
	RewriteApplyFn func(ctx context.Context, scenarioPath string, ops []sidecar.Operation) ([]sidecar.OperationResult, error)
	ShutdownFn     func(ctx context.Context) error
	StatusValue    sidecar.Status

	ExtractCalls      int
	RewriteApplyCalls int
	ShutdownCalls     int
}

// Extract records the call and dispatches to ExtractFn.
func (f *FakeSidecarClient) Extract(ctx context.Context, scenarioPath string) (sidecar.ExtractResult, error) {
	f.mu.Lock()
	f.ExtractCalls++
	fn := f.ExtractFn
	f.mu.Unlock()
	if fn == nil {
		return sidecar.ExtractResult{}, nil
	}
	return fn(ctx, scenarioPath)
}

// RewriteApply records the call and dispatches to RewriteApplyFn.
func (f *FakeSidecarClient) RewriteApply(ctx context.Context, scenarioPath string, ops []sidecar.Operation) ([]sidecar.OperationResult, error) {
	f.mu.Lock()
	f.RewriteApplyCalls++
	fn := f.RewriteApplyFn
	f.mu.Unlock()
	if fn == nil {
		return nil, nil
	}
	return fn(ctx, scenarioPath, ops)
}

// Status returns the configured status (defaults to READY).
func (f *FakeSidecarClient) Status() sidecar.Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.StatusValue == "" {
		return sidecar.StatusReady
	}
	return f.StatusValue
}

// Shutdown records the call and dispatches to ShutdownFn.
func (f *FakeSidecarClient) Shutdown(ctx context.Context) error {
	f.mu.Lock()
	f.ShutdownCalls++
	fn := f.ShutdownFn
	f.mu.Unlock()
	if fn == nil {
		return nil
	}
	return fn(ctx)
}

// Compile-time guarantee.
var _ sidecar.SidecarClient = (*FakeSidecarClient)(nil)
