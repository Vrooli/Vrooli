//go:build linux

package platform

import (
	"fmt"
	"net"
	"syscall"
)

func peerProcessID(conn net.Conn) (int, error) {
	if conn == nil {
		return 0, ErrPeerProcessIdentityUnavailable
	}
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return 0, fmt.Errorf("%w: connection has no raw socket", ErrPeerProcessIdentityUnavailable)
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrPeerProcessIdentityUnavailable, err)
	}
	pid := 0
	var peerErr error
	if err := raw.Control(func(fd uintptr) {
		credentials, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err != nil {
			peerErr = err
			return
		}
		pid = int(credentials.Pid)
	}); err != nil {
		return 0, fmt.Errorf("%w: %v", ErrPeerProcessIdentityUnavailable, err)
	}
	if peerErr != nil {
		return 0, fmt.Errorf("%w: %v", ErrPeerProcessIdentityUnavailable, peerErr)
	}
	if pid <= 0 {
		return 0, fmt.Errorf("%w: kernel returned invalid PID %d", ErrPeerProcessIdentityUnavailable, pid)
	}
	return pid, nil
}
