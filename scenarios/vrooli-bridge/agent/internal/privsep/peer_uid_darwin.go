//go:build darwin

package privsep

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

// xucred is the Darwin LOCAL_PEERCRED socket option payload. Keep this small
// local definition so the cross-compiled agent does not need cgo or a runtime
// dependency solely for one credential lookup.
type xucred struct {
	Version uint32
	UID     uint32
	NGroups int16
	_       int16
	Groups  [16]uint32
}

func peerUID(conn net.Conn) (int, error) {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return -1, fmt.Errorf("IPC connection does not expose Darwin peer credentials")
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return -1, err
	}
	var uid int
	var innerErr error
	if err := raw.Control(func(fd uintptr) {
		cred := xucred{}
		length := uint32(unsafe.Sizeof(cred))
		_, _, errno := syscall.Syscall6(
			syscall.SYS_GETSOCKOPT,
			fd,
			0,                                // SOL_LOCAL
			1,                                // LOCAL_PEERCRED
			uintptr(unsafe.Pointer(&cred)),   // #nosec G103 -- syscall requires the kernel output buffer.
			uintptr(unsafe.Pointer(&length)), // #nosec G103 -- syscall requires the socklen pointer.
			0,
		)
		if errno != 0 {
			innerErr = errno
			return
		}
		uid = int(cred.UID)
	}); err != nil {
		return -1, err
	}
	if innerErr != nil {
		return -1, innerErr
	}
	return uid, nil
}
