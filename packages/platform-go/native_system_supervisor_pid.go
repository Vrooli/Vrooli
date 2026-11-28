package platform

import (
	"context"
	"errors"
)

// NativeSystemSupervisorPID reads the actual Running manager-owned machine
// broker slot. It is not a caller PID claim, executable grant, or permission to
// mutate the service. IPC peers must compare their kernel peer PID both before
// and after each exchange against this native observation.
func NativeSystemSupervisorPID(ctx context.Context) (int, error) {
	if ctx == nil {
		return 0, errors.New("platform: native supervisor PID requires context")
	}
	if e := ctx.Err(); e != nil {
		return 0, e
	}
	return nativeSystemSupervisorPID(ctx)
}

// NativeSystemSupervisorStateRoot returns the fixed private state root only
// from the actual privileged current broker process. It creates no paths.
func NativeSystemSupervisorStateRoot(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errors.New("platform: native supervisor state context required")
	}
	if e := ctx.Err(); e != nil {
		return "", e
	}
	return nativeSystemSupervisorStateRoot(ctx)
}
