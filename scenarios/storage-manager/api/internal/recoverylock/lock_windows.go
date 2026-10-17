//go:build windows

package recoverylock

import "fmt"

// Acquire is the platform seam for Windows LockFileEx. The Windows adapter
// is deliberately explicit until the control-plane Windows host package is
// available; callers receive a clear unsupported result, never a false lock.
func Acquire(path string) (func(), error) {
	return AcquireFor(path, "windows")
}

func AcquireFor(path, _ string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	return nil, fmt.Errorf("recovery lock is not implemented on windows")
}
