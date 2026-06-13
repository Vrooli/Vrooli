//go:build !unix && !windows

package process

import (
	"fmt"
	"os"
)

func processIsAlive(process *os.Process) bool {
	// No liveness primitive on this platform; report dead so callers treat
	// the evidence as absent rather than trusting a stale record.
	return false
}

func readProcessEnvironment(pid int) (map[string]string, error) {
	return nil, fmt.Errorf("process environment inspection is not supported on this platform")
}
