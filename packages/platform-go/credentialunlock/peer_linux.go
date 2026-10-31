//go:build linux

package credentialunlock

import (
	"fmt"
	"net"
	"syscall"
)

// PeerUID returns the kernel-reported UID of the process on the other end of a
// unix socket (SO_PEERCRED).
func PeerUID(conn net.Conn) (int, error) {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return -1, fmt.Errorf("connection does not expose peer credentials")
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return -1, err
	}
	uid := -1
	var inner error
	if err := raw.Control(func(fd uintptr) {
		cred, err := syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
		if err != nil {
			inner = err
			return
		}
		uid = int(cred.Uid)
	}); err != nil {
		return -1, err
	}
	return uid, inner
}
