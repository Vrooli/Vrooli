//go:build linux && (amd64 || arm64)

package agentharness

import (
	"syscall"
	"time"
	"unsafe"
)

type statxTimestamp struct {
	Sec  int64
	Nsec uint32
	_    int32
}

// statxBuffer is struct statx from the kernel ABI: 256 bytes, with the birth
// time at offset 80.
type statxBuffer struct {
	Mask           uint32
	Blksize        uint32
	Attributes     uint64
	Nlink          uint32
	UID            uint32
	GID            uint32
	Mode           uint16
	_              uint16
	Ino            uint64
	Size           uint64
	Blocks         uint64
	AttributesMask uint64
	Atime          statxTimestamp
	Btime          statxTimestamp
	Ctime          statxTimestamp
	Mtime          statxTimestamp
	_              [16]uint64
}

const (
	atSymlinkNoFollow = 0x100
	statxBtime        = 0x800
)

// atFDCWD is -100 as the unsigned argument the kernel reads.
const atFDCWD = ^uintptr(99)

// birthTime is when the file system created the path (a link is measured as a
// link). ok is false when the file system does not record it, which callers
// treat as "unknown", never as "old" or "new".
func birthTime(path string) (time.Time, bool) {
	p, err := syscall.BytePtrFromString(path)
	if err != nil {
		return time.Time{}, false
	}
	var st statxBuffer
	_, _, errno := syscall.Syscall6(sysStatx, atFDCWD, uintptr(unsafe.Pointer(p)), uintptr(atSymlinkNoFollow), uintptr(statxBtime), uintptr(unsafe.Pointer(&st)), 0)
	if errno != 0 || st.Mask&statxBtime == 0 {
		return time.Time{}, false
	}
	return time.Unix(st.Btime.Sec, int64(st.Btime.Nsec)).UTC(), true
}
