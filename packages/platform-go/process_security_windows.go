//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

func nativeProcessSecurity(pid int) (ProcessSecurityContext, error) {
	h, err := openProcessForObservation(pid, windows.PROCESS_QUERY_LIMITED_INFORMATION)
	if err != nil {
		return ProcessSecurityContext{}, err
	}
	defer windows.CloseHandle(h)
	var token windows.Token
	if err := windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token); err != nil {
		return ProcessSecurityContext{}, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return ProcessSecurityContext{}, err
	}
	var elevated uint32
	var size uint32
	if err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &size); err != nil {
		return ProcessSecurityContext{}, err
	}
	return ProcessSecurityContext{Principal: user.User.Sid.String(), EffectiveUID: -1, Elevated: elevated != 0}, nil
}
