//go:build windows || plan9

package main

import (
	"errors"
	"os"
)

// These platforms do not share the Unix flock syscall. The loop's native
// scheduler still owns restart behavior there; this fallback keeps the lock
// seam buildable until a platform-specific non-blocking primitive is needed.
func acquireNativeLoopLock(path string) (func(), error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, errNativeLockUnavailable
		}
		return nil, err
	}
	return func() {
		_ = file.Close()
		_ = os.Remove(path)
	}, nil
}
