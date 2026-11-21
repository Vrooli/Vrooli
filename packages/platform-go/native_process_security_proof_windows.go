//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

func windowsOperatorTokenProof(token windows.Token) (string, error) {
	user, e := token.GetTokenUser()
	if e != nil {
		return "", e
	}
	principal := "sid:" + user.User.Sid.String()
	if !validNativeObserverPrincipal("windows", principal) {
		return "", ErrOperationIdentity
	}
	var elevated, size uint32
	if e = windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), 4, &size); e != nil {
		return "", e
	}
	if size != 4 || elevated != 0 {
		return "", ErrOperationIdentity
	}
	return windowsOperatorSecurityFactsProof(principal, false)
}
func windowsProcessOperatorSecurityProof(process windows.Handle, expected string) (string, error) {
	var token windows.Token
	if e := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); e != nil {
		return "", e
	}
	defer token.Close()
	user, e := token.GetTokenUser()
	if e != nil {
		return "", e
	}
	if expected != "sid:"+user.User.Sid.String() {
		return "", ErrOperationIdentity
	}
	return windowsOperatorTokenProof(token)
}
func nativeProcessSecurityProof(pid int) (string, error) {
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if e != nil {
		return "", e
	}
	defer windows.CloseHandle(h)
	var token windows.Token
	if e = windows.OpenProcessToken(h, windows.TOKEN_QUERY, &token); e != nil {
		return "", e
	}
	defer token.Close()
	return windowsOperatorTokenProof(token)
}
