//go:build !linux

package smoketest

import (
	"fmt"
	"runtime"
	"time"
)

func cleanupProcessSession(sessionID, _ string, _ time.Duration) ([]int, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("empty process session marker")
	}
	// Other platforms still get process-group/tree cleanup. Marker-based escaped
	// scope enumeration is currently implemented by the Linux /proc adapter.
	if runtime.GOOS == "windows" {
		return nil, nil
	}
	return nil, nil
}
