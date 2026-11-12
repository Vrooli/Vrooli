//go:build windows

package platform

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
)

func ownedServiceReleased(ctx context.Context, r ServiceRef, boot string) (bool, error) {
	return ownedWorkloadReleased(ctx, OperationRef(r), boot)
}
func ownedServiceIntentAbsent(ctx context.Context, i ServiceIntent) (bool, error) {
	return ownedWorkloadIntentAbsent(ctx, OperationIntent(i))
}
func ownedWorkloadReleased(ctx context.Context, r OperationRef, boot string) (bool, error) {
	absent, e := ownedWorkloadIntentAbsent(ctx, r.Intent)
	if e != nil || !absent {
		return false, e
	}
	if boot != "" {
		current, e := windowsOwnedBootID()
		if e != nil {
			return false, e
		}
		if current != boot {
			return true, nil
		}
	}
	stamp, alive, e := windowsOwnedProcessStart(r.RootPID)
	if e != nil {
		return false, e
	}
	return !alive || stamp != r.RootStartTime, nil
}
func ownedWorkloadIntentAbsent(ctx context.Context, i OperationIntent) (bool, error) {
	_, e := lookupOwnedOperation(ctx, i)
	if e == nil {
		return false, nil
	}
	if !errors.Is(e, ErrOperationNotFound) {
		if !errors.Is(e, ErrOperationIdentity) {
			return false, e
		}
		gone, inspectErr := windowsOwnedReplacedSlot(ctx, i)
		if inspectErr != nil || !gone {
			return false, inspectErr
		}
	}
	return windowsOwnedJobAbsent(windowsOwnedJob(i))
}

// Invocation-specific ledger keys cannot contain a replacement invocation.
func windowsOwnedReplacedSlot(context.Context, OperationIntent) (bool, error) {
	return false, ErrOperationIdentity
}

// A retained empty native object is quiescent, but not absent. Query leases
// must close before a retirement observation can claim namespace absence.
func windowsOwnedJobAbsent(name string) (bool, error) {
	h, e := openWindowsOwnedJob(name)
	if e != nil {
		return false, e
	}
	if h == 0 {
		return true, nil
	}
	windows.CloseHandle(h)
	return false, nil
}
