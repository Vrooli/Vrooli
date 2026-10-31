//go:build windows

package hostsem

import (
	"crypto/sha256"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	waitObject0 = 0
	waitTimeout = 258
)

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	createMutexW  = kernel32.NewProc("CreateMutexW")
	waitForSingle = kernel32.NewProc("WaitForSingleObject")
	releaseMutex  = kernel32.NewProc("ReleaseMutex")
	closeHandle   = kernel32.NewProc("CloseHandle")
)

// Windows named mutexes retain the same cross-process semantics as flock.
// The hash prevents arbitrary filesystem characters from becoming a mutex name.
func tryAcquireSlot(dir string, slot int) (func(), bool, error) {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s/slot-%d", dir, slot)))
	name, err := syscall.UTF16PtrFromString("Global\\vrooli-hostsem-" + fmt.Sprintf("%x", sum[:]))
	if err != nil {
		return nil, false, fmt.Errorf("hostsem: mutex name: %w", err)
	}
	handle, _, callErr := createMutexW.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return nil, false, fmt.Errorf("hostsem: create mutex: %w", callErr)
	}
	result, _, waitErr := waitForSingle.Call(handle, 0)
	if result == waitTimeout {
		_, _, _ = closeHandle.Call(handle)
		return nil, false, nil
	}
	if result != waitObject0 {
		_, _, _ = closeHandle.Call(handle)
		return nil, false, fmt.Errorf("hostsem: wait mutex: %w", waitErr)
	}
	return func() {
		_, _, _ = releaseMutex.Call(handle)
		_, _, _ = closeHandle.Call(handle)
	}, true, nil
}
