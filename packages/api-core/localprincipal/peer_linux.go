//go:build linux

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
	var credential *unix.Ucred
	var peerErr error
	raw, err := conn.SyscallConn()
	if err != nil {
		return "", fmt.Errorf("access peer socket: %w", err)
	}
	err = raw.Control(func(fd uintptr) {
		credential, peerErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
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

// PeerPID reports the process credential captured by the kernel for this Unix
// connection. It must not be replaced with a request-supplied PID.
func PeerPID(conn *net.UnixConn) (uint32, error) {
	if conn == nil {
		return 0, fmt.Errorf("local peer PID unavailable")
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var credential *unix.Ucred
	var peerErr error
	err = raw.Control(func(fd uintptr) {
		credential, peerErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	})
	if err != nil || peerErr != nil || credential == nil || credential.Pid <= 0 {
		return 0, fmt.Errorf("local peer PID unavailable")
	}
	return uint32(credential.Pid), nil
}
