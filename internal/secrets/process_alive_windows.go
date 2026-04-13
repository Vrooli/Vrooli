//go:build windows

package secrets

func processAlive(pid int) bool {
	// Conservative placeholder until a Windows-specific handle-based liveness
	// check is needed. The stale-lock contract is still safe because callers will
	// time out rather than forcibly reclaim a lock held by an unknown process.
	_ = pid
	return true
}
