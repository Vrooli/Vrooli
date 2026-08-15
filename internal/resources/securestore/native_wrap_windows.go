//go:build windows

package securestore

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	crypt32            = windows.NewLazySystemDLL("crypt32.dll")
	cryptProtectData   = crypt32.NewProc("CryptProtectData")
	cryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	localFree          = windows.NewLazySystemDLL("kernel32.dll").NewProc("LocalFree")
)

func nativeWrapAvailable() (string, error) {
	if err := cryptProtectData.Find(); err != nil {
		return "", fmt.Errorf("%w: Windows DPAPI is unavailable: %v", errKeyProviderUnavailable, err)
	}
	if err := cryptUnprotectData.Find(); err != nil {
		return "", fmt.Errorf("%w: Windows DPAPI unprotect is unavailable: %v", errKeyProviderUnavailable, err)
	}
	return keyStoreDPAPI, nil
}

func nativeWrapProtect(value []byte) ([]byte, error) {
	if _, err := nativeWrapAvailable(); err != nil {
		return nil, err
	}
	in := windows.DataBlob{Size: uint32(len(value)), Data: &value[0]}
	var out windows.DataBlob
	r, _, err := cryptProtectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, fmt.Errorf("%w: CryptProtectData: %v", errKeyProviderUnavailable, err)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func nativeWrapUnprotect(value []byte) ([]byte, error) {
	if _, err := nativeWrapAvailable(); err != nil {
		return nil, err
	}
	in := windows.DataBlob{Size: uint32(len(value)), Data: &value[0]}
	var out windows.DataBlob
	r, _, err := cryptUnprotectData.Call(uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0, 0, uintptr(unsafe.Pointer(&out)))
	if r == 0 {
		return nil, fmt.Errorf("%w: CryptUnprotectData: %v", errKeyProviderUnavailable, err)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
