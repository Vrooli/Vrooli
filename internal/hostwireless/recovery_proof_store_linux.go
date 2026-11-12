//go:build linux

package hostwireless

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

func recoveryRecordName(r RecoveryProofRequest) string {
	b, _ := json.Marshal(r)
	return digest(b) + ".json"
}
func (o *recoveryObserver) recordDirectory(create bool) (*os.File, error) {
	parent, e := o.root.Open(".")
	if e != nil {
		return nil, e
	}
	for _, component := range strings.Split(recoveryRecordDir, "/") {
		var st unix.Stat_t
		if e = unix.Fstat(int(parent.Fd()), &st); e != nil || st.Uid != o.uid || st.Mode&0022 != 0 {
			parent.Close()
			return nil, fmt.Errorf("unprotected recovery state ancestor %s (uid %d, mode %o, stat error %v)", parent.Name(), st.Uid, st.Mode, e)
		}
		if create {
			e = unix.Mkdirat(int(parent.Fd()), component, 0700)
			if e != nil && !errors.Is(e, unix.EEXIST) {
				parent.Close()
				return nil, e
			}
			if e = parent.Sync(); e != nil {
				parent.Close()
				return nil, e
			}
		}
		fd, err := unix.Openat(int(parent.Fd()), component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		parent.Close()
		if err != nil {
			return nil, err
		}
		parent = os.NewFile(uintptr(fd), component)
	}
	var st unix.Stat_t
	if e = unix.Fstat(int(parent.Fd()), &st); e != nil || st.Uid != o.uid || st.Mode&0077 != 0 {
		parent.Close()
		return nil, errors.New("recovery state directory must be private and root owned")
	}
	return parent, nil
}
func (o *recoveryObserver) protectedRecord(r RecoveryProofRequest) (*os.File, error) {
	dir, e := o.recordDirectory(false)
	if e != nil {
		return nil, e
	}
	defer dir.Close()
	fd, e := unix.Openat(int(dir.Fd()), recoveryRecordName(r), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if e != nil {
		return nil, e
	}
	f := os.NewFile(uintptr(fd), "recovery-proof")
	var st unix.Stat_t
	if e = unix.Fstat(fd, &st); e != nil || st.Mode&unix.S_IFMT != unix.S_IFREG || st.Uid != o.uid || st.Mode&0077 != 0 || st.Nlink != 1 {
		f.Close()
		return nil, errors.New("recovery record must be a private root-owned regular single-link file")
	}
	return f, nil
}
func (o *recoveryObserver) save(r RecoveryProofRequest, data []byte) error {
	dir, e := o.recordDirectory(true)
	if e != nil {
		return e
	}
	defer dir.Close()
	var nonce [16]byte
	if _, e = rand.Read(nonce[:]); e != nil {
		return e
	}
	temporary := ".pending-" + hex.EncodeToString(nonce[:])
	fd, e := unix.Openat(int(dir.Fd()), temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0600)
	if e != nil {
		return e
	}
	f := os.NewFile(uintptr(fd), temporary)
	defer f.Close()
	defer unix.Unlinkat(int(dir.Fd()), temporary, 0)
	if _, e = f.Write(data); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	// A new independently observed fallback boot may refresh stale evidence.
	// Validate an existing leaf before replacing it; rename never follows it.
	previous, checkErr := o.protectedRecord(r)
	if checkErr == nil {
		previous.Close()
	} else if !errors.Is(checkErr, os.ErrNotExist) {
		return checkErr
	}
	if e = unix.Renameat2(int(dir.Fd()), temporary, int(dir.Fd()), recoveryRecordName(r), 0); e != nil {
		return fmt.Errorf("publish observed recovery proof: %w", e)
	}
	return dir.Sync()
}

// activePhysicalVT observes the caller's controlling terminal, and asks the
// kernel whether it is the currently active virtual console. Pseudo terminals,
// SSH sessions and a user supplied assertion cannot produce this evidence.
func activePhysicalVT() (string, error) {
	f, e := os.OpenFile("/dev/tty", os.O_RDONLY|unix.O_NOCTTY, 0)
	if e != nil {
		return "", errors.New("run recovery proof at a physically reachable active Linux VT")
	}
	defer f.Close()
	var st unix.Stat_t
	if e = unix.Fstat(int(f.Fd()), &st); e != nil {
		return "", e
	}
	// /dev/tty itself is major 5 minor 0; TIOCGDEV returns the actual terminal.
	var device uint32
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.TIOCGDEV, uintptr(unsafe.Pointer(&device)))
	if errno != 0 {
		return "", errno
	}
	if unix.Major(uint64(device)) != 4 || unix.Minor(uint64(device)) < 1 || unix.Minor(uint64(device)) > 63 {
		return "", errors.New("recovery console is not a physical Linux virtual terminal")
	}
	vt := struct{ Active, Signal, State uint16 }{}
	_, _, errno = unix.Syscall(unix.SYS_IOCTL, f.Fd(), 0x5603 /* VT_GETSTATE, linux/vt.h */, uintptr(unsafe.Pointer(&vt)))
	if errno != 0 {
		return "", errno
	}
	if uint32(vt.Active) != unix.Minor(uint64(device)) {
		return "", errors.New("recovery VT is not currently active")
	}
	return fmt.Sprintf("tty%d", vt.Active), nil
}
