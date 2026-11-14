//go:build windows

package localprincipal

import (
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SIO_AF_UNIX_GETPEERPID is _WSAIOR(IOC_VENDOR, 256), defined in the
// Windows SDK afunix.h. Windows builds without this kernel facility fail
// closed; there is no request-supplied PID or TCP/token downgrade.
const sioAFUnixGetPeerPID = uint32(0x58000100)

func PeerPID(conn *net.UnixConn) (uint32, error) {
	if conn == nil {
		return 0, fmt.Errorf("local peer PID unavailable")
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, err
	}
	var pid, count uint32
	var peerErr error
	err = raw.Control(func(fd uintptr) {
		peerErr = windows.WSAIoctl(windows.Handle(fd), sioAFUnixGetPeerPID, nil, 0, (*byte)(unsafe.Pointer(&pid)), uint32(unsafe.Sizeof(pid)), &count, nil, 0)
	})
	if err != nil || peerErr != nil || count != uint32(unsafe.Sizeof(pid)) || pid == 0 {
		return 0, fmt.Errorf("local peer PID unavailable")
	}
	return pid, nil
}

func current() (Principal, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("current local principal unavailable: %w", err)
	}
	return Principal("windows:" + user.User.Sid.String()), nil
}

func peer(conn *net.UnixConn) (Principal, error) {
	pid, err := PeerPID(conn)
	if err != nil {
		return "", err
	}
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", fmt.Errorf("local peer process unavailable: %w", err)
	}
	defer windows.CloseHandle(process)
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return "", fmt.Errorf("local peer token unavailable: %w", err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("local peer principal unavailable: %w", err)
	}
	return Principal("windows:" + user.User.Sid.String()), nil
}
