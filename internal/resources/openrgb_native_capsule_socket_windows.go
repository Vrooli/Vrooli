//go:build windows

package resources

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"unsafe"

	platform "github.com/vrooli/platform-go"
	"golang.org/x/sys/windows"
)

// Windows AF_UNIX uses a socket reparse object. Modify only a held handle to
// that exact AF_UNIX tag, never a directory or arbitrary reparse target.
func configureOpenRGBNativeClaimSocket(ctx context.Context, c openRGBNativeCapsule) (os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	expected, err := openRGBNativeCapsuleFrozenEndpoint(c)
	if err != nil || expected != c.Endpoint {
		return nil, errors.New("native claim socket path differs from admitted capsule")
	}

	account, err := platform.ResolveOwnedServiceAccount(ctx, c.Account)
	if err != nil {
		return nil, err
	}
	return configureOpenRGBNativeSocketAccess(ctx, c.Endpoint, c.OwnerPrincipal, account.Principal)
}
func configureOpenRGBNativeSocketAccess(ctx context.Context, path, ownerPrincipal, runtimePrincipal string) (os.FileInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(name, windows.READ_CONTROL|windows.WRITE_DAC|windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return nil, err
	}
	held := os.NewFile(uintptr(handle), path)
	defer held.Close()
	pinned, err := held.Stat()
	if err != nil || !os.SameFile(info, pinned) {
		return nil, errors.New("native socket changed while opening")
	}
	var tag struct{ Attributes, Tag uint32 }
	if err = windows.GetFileInformationByHandleEx(handle, windows.FileAttributeTagInfo, (*byte)(unsafe.Pointer(&tag)), uint32(unsafe.Sizeof(tag))); err != nil {
		return nil, err
	}
	if tag.Attributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 || tag.Attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 || tag.Tag != 0x80000023 {
		return nil, errors.New("native claim endpoint is not an AF_UNIX socket object")
	}
	current, err := windows.GetCurrentThreadEffectiveToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	if ownerPrincipal != "windows:"+current.User.Sid.String() {
		return nil, errors.New("native socket owner differs from current kernel principal")
	}
	sid, err := windows.StringToSid(strings.TrimPrefix(runtimePrincipal, "sid:"))
	if err != nil || runtimePrincipal != "sid:"+sid.String() || !windows.EqualSid(sid, current.User.Sid) {
		return nil, errors.New("native socket account differs from the actual operator SID")
	}
	actual, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return nil, err
	}
	owner, _, err := actual.Owner()
	if err != nil || owner == nil || !windows.EqualSid(owner, current.User.Sid) {
		return nil, errors.New("native socket object has a foreign owner")
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + owner.String() + ")")
	if err != nil {
		return info, err
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return info, err
	}
	if err = windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); err != nil {
		return info, err
	}
	observed, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return info, err
	}
	observedOwner, _, err := observed.Owner()
	if err != nil || observedOwner == nil || !windows.EqualSid(owner, observedOwner) {
		return info, errors.New("native socket owner changed")
	}
	have, _, err := observed.DACL()
	if err != nil || !equalNativeCapsuleSocketACL(dacl, have) {
		return info, errors.New("native socket ACL changed during setup")
	}
	after, err := os.Lstat(path)
	if err != nil || !os.SameFile(info, after) {
		return info, errors.New("native socket path changed during setup")
	}
	return after, nil
}
func equalNativeCapsuleSocketACL(a, b *windows.ACL) bool {
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
