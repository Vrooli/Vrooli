//go:build linux

package activityedge

import (
	"fmt"
	"net"
	"syscall"
)

func unixPeerCredentials(conn net.Conn) (int, int, error) {
	u, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, 0, fmt.Errorf("bridge peer is not a Unix connection")
	}
	raw, err := u.SyscallConn()
	if err != nil {
		return 0, 0, err
	}
	var cred *syscall.Ucred
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		cred, sockErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return 0, 0, err
	}
	if sockErr != nil {
		return 0, 0, sockErr
	}
	return int(cred.Pid), int(cred.Uid), nil
}
