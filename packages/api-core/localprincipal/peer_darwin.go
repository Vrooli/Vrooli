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

// PeerPID reads the process identity captured by the kernel for the connected
// local socket. An unavailable kernel credential never becomes a caller PID.
func PeerPID(conn *net.UnixConn) (uint32, error) {
	if conn == nil {
		return 0, fmt.Errorf("local peer PID unavailable")
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var pid int
	var peerErr error
	err = raw.Control(func(fd uintptr) {
		pid, peerErr = unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
	})
	if err != nil || peerErr != nil || pid <= 0 {
		return 0, fmt.Errorf("local peer PID unavailable")
	}
	return uint32(pid), nil
}
