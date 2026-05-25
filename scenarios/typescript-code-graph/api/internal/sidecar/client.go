package sidecar

import "context"

// SidecarClient is the seam between domain packages (internal/graph,
// internal/rewrite) and the Node sidecar process.
//
// seam: production wires *Supervisor; tests wire
// mocks.FakeSidecarClient. Domain code MUST take this interface, never
// the concrete supervisor.
type SidecarClient interface {
	// Extract requests a graph for the project rooted at scenarioPath.
	// ctx.Done() triggers a best-effort cancel IPC and resolves the
	// pending future locally with ctx.Err(). The returned
	// ExtractResult.RequestID is the supervisor-minted UUID for the
	// underlying IPC request (non-empty whenever the sidecar was reached).
	Extract(ctx context.Context, scenarioPath string) (ExtractResult, error)

	// RewriteApply executes the given operations against the project at
	// scenarioPath. The result slice matches ops 1:1.
	RewriteApply(ctx context.Context, scenarioPath string, ops []Operation) ([]OperationResult, error)

	// Status reports the supervisor's view of the child process. It is
	// safe to call concurrently and never blocks.
	Status() Status

	// Shutdown sends a graceful shutdown IPC and waits for the child to
	// exit, killing it if ctx expires first.
	Shutdown(ctx context.Context) error
}

// Compile-time guarantee.
var _ SidecarClient = (*Supervisor)(nil)
