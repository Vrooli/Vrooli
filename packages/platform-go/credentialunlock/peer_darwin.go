//go:build darwin

package credentialunlock

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// xucred is the Darwin LOCAL_PEERCRED payload; defined locally so the agent
// cross-compiles without cgo.
type xucred struct {
	Version uint32
	UID     uint32
	NGroups int16
	_       int16
	Groups  [16]uint32
}

// PeerUID returns the kernel-reported UID of the process on the other end of a
// unix socket (LOCAL_PEERCRED).
func PeerUID(conn net.Conn) (int, error) {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return -1, fmt.Errorf("connection does not expose Darwin peer credentials")
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return -1, err
	}
	uid := -1
	var inner error
	if err := raw.Control(func(fd uintptr) {
		cred := xucred{}
		length := uint32(unsafe.Sizeof(cred))
		_, _, errno := syscall.Syscall6(syscall.SYS_GETSOCKOPT, fd,
			0,                                // SOL_LOCAL
			1,                                // LOCAL_PEERCRED
			uintptr(unsafe.Pointer(&cred)),   // #nosec G103 -- kernel output buffer.
			uintptr(unsafe.Pointer(&length)), // #nosec G103 -- socklen pointer.
			0)
		if errno != 0 {
			inner = errno
			return
		}
		uid = int(cred.UID)
	}); err != nil {
		return -1, err
	}
	return uid, inner
}
