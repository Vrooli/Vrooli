//go:build !linux && !darwin && !windows

package network

func captureTCPListenerSnapshot() TCPListenerSnapshot {
	return TCPListenerSnapshot{Reason: "listener snapshot is not implemented on this platform"}
}
