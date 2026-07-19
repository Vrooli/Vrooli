//go:build linux

package privilegebroker

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

func peerUID(conn *net.UnixConn) (uint32, error) {
	var credential *unix.Ucred
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("access peer socket: %w", err)
	}
	err = raw.Control(func(fd uintptr) {
		credential, _ = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if err != nil || credential == nil {
		return 0, fmt.Errorf("read peer credentials")
	}
	return credential.Uid, nil
}
