//go:build linux

package privsep

import (
	"fmt"
	"net"
	"syscall"
)

func peerUID(conn net.Conn) (int, error) {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return -1, fmt.Errorf("IPC connection does not expose peer credentials")
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return -1, err
	}
	var uid int
	var innerErr error
	if err := raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err != nil {
			innerErr = err
			return
		}
		uid = int(cred.Uid)
	}); err != nil {
		return -1, err
	}
	if innerErr != nil {
		return -1, innerErr
	}
	return uid, nil
}
