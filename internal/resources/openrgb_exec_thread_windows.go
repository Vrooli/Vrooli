//go:build windows

package resources

import (
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
)

func confineOpenRGBExecThread() (func(), error) {
	runtime.LockOSThread()
	var token windows.Token
	err := windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &token)
	if err == nil {
		token.Close()
		runtime.UnlockOSThread()
		return nil, errors.New("OpenRGB wrapper must not run with a thread impersonation token")
	}
	if !errors.Is(err, windows.ERROR_NO_TOKEN) {
		runtime.UnlockOSThread()
		return nil, err
	}
	return runtime.UnlockOSThread, nil
}
