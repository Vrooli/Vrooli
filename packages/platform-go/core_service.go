package platform

import "context"

// CoreServiceState is an observation of the fixed runtime-supervisor unit.
// It grants no authority to change the service or another unit.
type CoreServiceState struct {
	Unit, LoadState, ActiveState, SubState                string
	ControlGroup, InvocationID, FragmentPath, DropInPaths string
	KillMode, NeedDaemonReload                            string
	PID                                                   int
	PendingJob                                            bool
}

func InspectCoreSupervisor(ctx context.Context) (CoreServiceState, error) {
	if ctx == nil {
		return CoreServiceState{}, ErrOperationIdentity
	}
	return inspectCoreSupervisor(ctx)
}

// CoreSupervisorFence pins the kernel tree before its owner requests stop.
// Stopped must prove native job exclusion and emptiness of that exact tree;
// a successful stop acknowledgement alone is insufficient.
type CoreSupervisorFence struct {
	stopped func(context.Context) error
	close   func() error
}

func PinCoreSupervisor(ctx context.Context, expected CoreServiceState) (*CoreSupervisorFence, error) {
	if ctx == nil {
		return nil, ErrOperationIdentity
	}
	return pinCoreSupervisor(ctx, expected)
}

func (f *CoreSupervisorFence) Stopped(ctx context.Context) error {
	if f == nil || f.stopped == nil || ctx == nil {
		return ErrOperationIdentity
	}
	return f.stopped(ctx)
}

func (f *CoreSupervisorFence) Close() error {
	if f == nil || f.close == nil {
		return nil
	}
	return f.close()
}
