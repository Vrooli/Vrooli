//go:build windows

package lightingbinding

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Listen uses Windows AF_UNIX with a protected, explicitly owned directory.
// Client connections also require the kernel's SIO_AF_UNIX_GETPEERPID proof.
// Older Windows kernels lacking that facility cannot establish a binding.
func Listen(directory string, identity Identity) (net.Listener, error) {
	if err := identity.Validate(); err != nil {
		return nil, err
	}
	if identity.Scenario != "lighting-control" || identity.Variant != "live" || identity.PID != os.Getpid() || !filepath.IsAbs(directory) || filepath.Clean(directory) != directory {
		return nil, errors.New("native lighting requires its own exact default scenario identity and canonical absolute directory")
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	sid := user.User.Sid
	// SYSTEM is the maintained host control-plane authority; other interactive
	// principals and inherited groups receive no access to the binding endpoint.
	descriptor, err := windows.SecurityDescriptorFromString("O:" + sid.String() + "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;" + sid.String() + ")")
	if err != nil {
		return nil, err
	}
	path, err := windows.UTF16PtrFromString(directory)
	if err != nil {
		return nil, err
	}
	attributes := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: descriptor}
	if err := windows.CreateDirectory(path, &attributes); err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil, err
	}
	// Deny replacement of the directory while the socket is live. Refuse a
	// reparse-point directory rather than modifying someone else's ACL.
	handle, err := windows.CreateFile(path, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	closeHandle := true
	defer func() {
		if closeHandle {
			_ = windows.CloseHandle(handle)
		}
	}()
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return nil, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return nil, errors.New("native lighting directory must be a real directory")
	}
	actual, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return nil, err
	}
	owner, _, err := actual.Owner()
	if err != nil || owner == nil || !windows.EqualSid(owner, sid) {
		return nil, errors.New("native lighting directory owner differs from current principal")
	}
	control, _, err := actual.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return nil, errors.New("native lighting directory must have a protected ACL")
	}
	wantACL, _, err := descriptor.DACL()
	if err != nil {
		return nil, err
	}
	haveACL, _, err := actual.DACL()
	if err != nil || !equalLightingACL(wantACL, haveACL) {
		return nil, errors.New("native lighting directory ACL permits an unqualified principal")
	}
	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: SocketPath(directory, identity), Net: "unix"})
	if err != nil {
		return nil, fmt.Errorf("private native lighting listener: %w", err)
	}
	closeHandle = false
	return &windowsLightingListener{Listener: listener, directory: handle}, nil
}

func equalLightingACL(a, b *windows.ACL) bool {
	if a == nil || b == nil || a.AceCount != b.AceCount {
		return false
	}
	for i := uint32(0); i < uint32(a.AceCount); i++ {
		var x, y *windows.ACCESS_ALLOWED_ACE
		if windows.GetAce(a, i, &x) != nil || windows.GetAce(b, i, &y) != nil || x == nil || y == nil || x.Header.AceSize != y.Header.AceSize {
			return false
		}
		if !bytes.Equal(unsafe.Slice((*byte)(unsafe.Pointer(x)), int(x.Header.AceSize)), unsafe.Slice((*byte)(unsafe.Pointer(y)), int(y.Header.AceSize))) {
			return false
		}
	}
	return true
}

type windowsLightingListener struct {
	net.Listener
	directory windows.Handle
	once      sync.Once
}

func (l *windowsLightingListener) Close() error {
	var err error
	l.once.Do(func() { err = errors.Join(l.Listener.Close(), windows.CloseHandle(l.directory)) })
	return err
}
