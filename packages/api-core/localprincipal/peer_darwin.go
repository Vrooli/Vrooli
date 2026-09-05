//go:build darwin

package localprincipal

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/sys/unix"
)

func peer(conn *net.UnixConn) (Principal, error) {
	if conn == nil {
		return "", fmt.Errorf("local peer credentials: nil connection")
	}
	var credential *unix.Xucred
	var peerErr error
	raw, err := conn.SyscallConn()
	if err != nil {
		return "", fmt.Errorf("access peer socket: %w", err)
	}
	err = raw.Control(func(fd uintptr) {
		credential, peerErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	})
	if err != nil {
		return "", fmt.Errorf("read peer credentials: %w", err)
	}
	if peerErr != nil {
		return "", fmt.Errorf("read peer credentials: %w", peerErr)
	}
	if credential == nil {
		return "", fmt.Errorf("read peer credentials: empty result")
	}
	return UnixUID(credential.Uid), nil
}

func current() (Principal, error) { return UnixUID(uint32(os.Getuid())), nil }
