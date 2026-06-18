package gate

import (
	"context"
	"time"
)

// NodeRef is the minimal node shape gate needs to select a validation target per
// OS: its id, OS, and whether it is revoked. The handler adapter projects a
// registry node down to this.
type NodeRef struct {
	ID      string
	OS      string
	Arch    string
	Revoked bool
}

// NodeLister is the registry enumeration seam: the gate needs every registered
// node so it can pick one eligible node per target OS. The handler adapter wraps
// the registry service.
type NodeLister interface {
	// ListNodes returns every registered node. An empty fleet returns an empty
	// slice (not an error).
	ListNodes(ctx context.Context) ([]NodeRef, error)
}

// Presence is the live seam node selection gates on: online state + the
// protocol-compatibility verdict. The presence hub satisfies it. A node that is
// offline or version-drifted is not an eligible validation target.
type Presence interface {
	IsOnline(nodeID string) bool
	// Dispatchable reports online AND protocol-compatible (not flagged).
	Dispatchable(nodeID string) bool
}

// Runner is the dispatch + runs delegation seam: dispatch one node's validation
// run and read/await its verdict. The handler adapter wraps the dispatch service
// (which enforces the allowlist + per-node scopes + audit) and the runs service
// (the durable run lifecycle). Gate NEVER reimplements dispatch or run
// management (or imports the dispatch / runs domain / proto).
type Runner interface {
	// Dispatch validates + dispatches the validation run to the node and returns
	// its durable run id. It surfaces the dispatch domain's typed rejection on a
	// disallowed verb / offline / incompatible node.
	Dispatch(ctx context.Context, in DispatchRequest) (runID string, err error)

	// Verdict returns the current (possibly non-terminal) verdict of a run
	// without blocking.
	Verdict(ctx context.Context, runID string) (RunVerdict, error)

	// Wait blocks once until the run reaches a terminal verdict or the timeout
	// elapses, returning the latest verdict (Terminal is false on timeout).
	Wait(ctx context.Context, runID string, timeout time.Duration) (RunVerdict, error)
}

// DispatchRequest is the gate-local DTO for one OS's validation dispatch.
type DispatchRequest struct {
	Actor          string
	NodeID         string
	Scenario       string
	Verb           string
	Args           []string
	TimeoutSeconds int64
}

// RunVerdict is the gate-local projection of a durable run's state: whether it
// is terminal, and (when terminal) how it ended.
type RunVerdict struct {
	// Terminal is true once the run has settled (passed / failed / aborted).
	Terminal bool
	// Passed is true only for a terminal run that exited 0.
	Passed bool
	// Aborted is true for a terminal run that was aborted (vs exited non-zero).
	Aborted bool
	// ExitCode is the run's exit code (meaningful once terminal).
	ExitCode int32
	// Detail is a human-readable status note.
	Detail string
}

// disposition maps a refreshed RunVerdict onto the OS disposition for a target
// that was previously PENDING.
func (v RunVerdict) disposition() OSDisposition {
	switch {
	case !v.Terminal:
		return OSDispositionPending
	case v.Aborted:
		return OSDispositionAborted
	case v.Passed:
		return OSDispositionPassed
	default:
		return OSDispositionFailed
	}
}
