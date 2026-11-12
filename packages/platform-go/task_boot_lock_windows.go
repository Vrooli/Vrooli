//go:build windows

package platform

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"strings"
	"unsafe"
)

// The setup and ordinary lifecycle caller serialize on the selected account,
// including an administrator installing for a different existing operator.
func lockWindowsNativeBootTask(ctx context.Context, principal string) (func(), error) {
	if !canonicalNativeBrokerWindowsUserPrincipal(principal) {
		return nil, ErrOperationIdentity
	}
	runtime.LockOSThread()
	locked := true
	defer func() {
		if locked {
			runtime.UnlockOSThread()
		}
	}()
	sid := strings.TrimPrefix(principal, "windows:")
	descriptor, e := windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;GA;;;" + sid + ")(A;;RC;;;OW)")
	if e != nil {
		return nil, e
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: descriptor}
	name, e := windows.UTF16PtrFromString(`Global\VrooliNativeBootTask-` + sid)
	if e != nil {
		return nil, e
	}
	handle, e := windows.CreateMutexEx(&sa, name, 0, windows.SYNCHRONIZE|windows.MUTEX_MODIFY_STATE|windows.READ_CONTROL)
	if e != nil && !errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
		return nil, e
	}
	fail := func(e error) (func(), error) { windows.CloseHandle(handle); return nil, e }
	observed, e := windows.GetSecurityInfo(handle, windows.SE_KERNEL_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return fail(e)
	}
	if e = verifyWindowsBootMutexDescriptor(observed, sid); e != nil {
		return fail(e)
	}
	for {
		if e = ctx.Err(); e != nil {
			return fail(e)
		}
		state, e := windows.WaitForSingleObject(handle, 100)
		if e != nil {
			return fail(e)
		}
		if state == windows.WAIT_OBJECT_0 || state == uint32(windows.WAIT_ABANDONED) {
			locked = false
			return func() { windows.ReleaseMutex(handle); windows.CloseHandle(handle); runtime.UnlockOSThread() }, nil
		}
		if state != uint32(windows.WAIT_TIMEOUT) {
			return fail(ErrOperationIdentity)
		}
	}
}

func verifyWindowsBootMutexDescriptor(sd *windows.SECURITY_DESCRIPTOR, operator string) error {
	control, _, e := sd.Control()
	if e != nil {
		return e
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return ErrOperationIdentity
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	if acl == nil || acl.AceCount != 4 {
		return ErrOperationIdentity
	}
	seen := map[string]bool{}
	for index := uint32(0); index < uint32(acl.AceCount); index++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, index, &ace); e != nil {
			return e
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != 0 {
			return ErrOperationIdentity
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if seen[sid] {
			return ErrOperationIdentity
		}
		seen[sid] = true
		if sid == "S-1-3-4" {
			if ace.Mask != windows.READ_CONTROL {
				return ErrOperationIdentity
			}
			continue
		}
		if sid != "S-1-5-18" && sid != "S-1-5-32-544" && sid != operator {
			return ErrOperationIdentity
		}
		if ace.Mask != windows.GENERIC_ALL && ace.Mask != windows.STANDARD_RIGHTS_REQUIRED|windows.SYNCHRONIZE|windows.MUTEX_MODIFY_STATE {
			return ErrOperationIdentity
		}
	}
	return nil
}
