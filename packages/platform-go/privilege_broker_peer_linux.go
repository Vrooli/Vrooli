//go:build linux

package platform

import (
	"fmt"
	"net"
	"syscall"
)

func verifyPrivilegeBrokerPeer(conn net.Conn) error {
	unix, ok := conn.(*net.UnixConn)
	if !ok {
		return fmt.Errorf("privilege broker requires a Unix peer")
	}
	raw, err := unix.SyscallConn()
	if err != nil {
		return err
	}
	var credential *syscall.Ucred
	var credentialErr error
	if err := raw.Control(func(fd uintptr) {
		credential, credentialErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	}); err != nil {
		return err
	}
	if credentialErr != nil {
		return credentialErr
	}
	if credential == nil || credential.Uid != 0 {
		return fmt.Errorf("privilege broker peer is not root; run vrooli setup --sudo-mode=ask")
	}
	return nil
}
