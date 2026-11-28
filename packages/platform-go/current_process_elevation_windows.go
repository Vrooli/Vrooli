//go:build windows

package platform

import "golang.org/x/sys/windows"

func nativeCurrentProcessElevated() (bool, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return false, err
	}
	defer token.Close()
	return token.IsElevated(), nil
}
