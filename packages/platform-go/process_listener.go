package platform

import "fmt"

// ProcessTCPListener is a kernel-observed TCP listening endpoint owned by a
// process. Darwin global snapshots report the kernel last/effective owner PID;
// the per-process descriptor witness establishes actual FD possession. Callers
// must bind the observation to an independently checked process
// incarnation before and after use. Address includes wildcard addresses; callers
// requiring loopback must reject them, rather than treating them as localhost.
type ProcessTCPListener struct {
	Address string
	Port    uint16
	PID     int
}

func ProcessTCPListeners(pid int) ([]ProcessTCPListener, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("platform: invalid pid %d", pid)
	}
	return nativeProcessTCPListeners(pid)
}

// TCPListeners observes every TCP listener at a port. Inaccessible ownership
// data is an error, never evidence of absence. A zero port is invalid.
func TCPListeners(port uint16) ([]ProcessTCPListener, error) {
	if port == 0 {
		return nil, fmt.Errorf("platform: TCP listener port is required")
	}
	return nativeTCPListeners(port)
}
