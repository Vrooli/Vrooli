package platform

import "fmt"

// ProcessAliveObserved distinguishes native absence from a failed/forbidden
// observation. A bare false from a best-effort liveness helper is insufficient
// for retiring durable ownership or a recovered endpoint.
func ProcessAliveObserved(pid int) (bool, error) {
	if pid <= 0 || uint64(pid) > 1<<31-1 {
		return false, fmt.Errorf("platform: invalid process id %d", pid)
	}
	return processAliveObserved(pid)
}
