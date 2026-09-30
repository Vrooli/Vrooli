package agentharness

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// The standard-library syscall loader avoids an additional dependency while
// preserving native DACLs. POSIX chmod alone does not protect Windows files.
var (
	permissionAdvapi           = syscall.NewLazyDLL("advapi32.dll")
	permissionKernel           = syscall.NewLazyDLL("kernel32.dll")
	getNamedPermissionSecurity = permissionAdvapi.NewProc("GetNamedSecurityInfoW")
	securityToString           = permissionAdvapi.NewProc("ConvertSecurityDescriptorToStringSecurityDescriptorW")
	stringToSecurity           = permissionAdvapi.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	getSecurityControl         = permissionAdvapi.NewProc("GetSecurityDescriptorControl")
	setPermissionSecurity      = permissionAdvapi.NewProc("SetNamedSecurityInfoW")
	freePermissionSecurity     = permissionKernel.NewProc("LocalFree")
	getPermissionDACL          = permissionAdvapi.NewProc("GetSecurityDescriptorDacl")
)

const permissionDACL = uint32(4)

func fileSecurityDescriptor(path string) (uintptr, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var descriptor uintptr
	result, _, _ := getNamedPermissionSecurity.Call(uintptr(unsafe.Pointer(name)), 1, 4, 0, 0, 0, 0, uintptr(unsafe.Pointer(&descriptor)))
	if result != 0 {
		return 0, fmt.Errorf("read file security %s: %w", path, syscall.Errno(result))
	}
	return descriptor, nil
}

func permissionFileSecurity(path string) (string, error) {
	descriptor, err := fileSecurityDescriptor(path)
	if err != nil {
		return "", err
	}
	defer freePermissionSecurity.Call(descriptor)
	var text *uint16
	var length uint32
	result, _, callErr := securityToString.Call(descriptor, 1, 4, uintptr(unsafe.Pointer(&text)), uintptr(unsafe.Pointer(&length)))
	if result == 0 {
		return "", fmt.Errorf("encode file security: %w", callErr)
	}
	defer freePermissionSecurity.Call(uintptr(unsafe.Pointer(text)))
	return syscall.UTF16ToString(unsafe.Slice(text, length)), nil
}

// SecurePermissionFile preserves the existing access ACL or creates a protected
// owner-only ACL on a private new file. Native failures abort publication.
func SecurePermissionFile(path, security string) error {
	var descriptor uintptr
	var err error
	if security != "" {
		descriptor, err = permissionDescriptorFromString(security)
	} else {
		descriptor, err = privatePermissionDescriptor()
	}
	if err != nil {
		return err
	}
	defer freePermissionSecurity.Call(descriptor)
	var control uint16
	var revision uint32
	result, _, callErr := getSecurityControl.Call(descriptor, uintptr(unsafe.Pointer(&control)), uintptr(unsafe.Pointer(&revision)))
	if result == 0 {
		return fmt.Errorf("read file ACL control: %w", callErr)
	}
	flags := permissionDACL | 0x20000000 // retain inheritance when source permits it
	if control&0x1000 != 0 {
		flags = permissionDACL | 0x80000000
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	var present, defaulted uint32
	var dacl uintptr
	result, _, callErr = getPermissionDACL.Call(descriptor, uintptr(unsafe.Pointer(&present)), uintptr(unsafe.Pointer(&dacl)), uintptr(unsafe.Pointer(&defaulted)))
	if result == 0 || present == 0 {
		return fmt.Errorf("native access ACL unavailable: %v", callErr)
	}
	result, _, _ = setPermissionSecurity.Call(uintptr(unsafe.Pointer(name)), 1, uintptr(flags), 0, 0, dacl, 0)
	if result != 0 {
		return fmt.Errorf("apply native file ACL: %w", syscall.Errno(result))
	}
	return nil
}

func privatePermissionDescriptor() (uintptr, error) {
	// OW is the owner-rights SID; the new file's owner is the current caller.
	return permissionDescriptorFromString("D:P(A;;FA;;;OW)")
}

func permissionDescriptorFromString(security string) (uintptr, error) {
	sddl, err := syscall.UTF16PtrFromString(security)
	if err != nil {
		return 0, err
	}
	var descriptor uintptr
	result, _, callErr := stringToSecurity.Call(uintptr(unsafe.Pointer(sddl)), 1, uintptr(unsafe.Pointer(&descriptor)), 0)
	if result == 0 {
		return 0, fmt.Errorf("decode permission ACL: %w", callErr)
	}
	return descriptor, nil
}

// CreatePermissionTemp applies its private DACL at creation, so another process
// cannot acquire a readable handle before a later ACL update protects the data.
func CreatePermissionTemp(directory, prefix string) (*os.File, error) {
	if strings.ContainsAny(prefix, `/\`) {
		return nil, fmt.Errorf("temporary permission prefix must not contain separators")
	}
	descriptor, err := privatePermissionDescriptor()
	if err != nil {
		return nil, err
	}
	defer freePermissionSecurity.Call(descriptor)
	attributes := syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(syscall.SecurityAttributes{})), SecurityDescriptor: descriptor}
	for attempt := 0; attempt < 16; attempt++ {
		var random [16]byte
		if _, err = rand.Read(random[:]); err != nil {
			return nil, err
		}
		path := filepath.Join(directory, prefix+hex.EncodeToString(random[:]))
		name, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			return nil, err
		}
		handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE, &attributes, syscall.CREATE_NEW, syscall.FILE_ATTRIBUTE_NORMAL, 0)
		if err == syscall.ERROR_FILE_EXISTS || err == syscall.ERROR_ALREADY_EXISTS {
			continue
		}
		if err != nil {
			return nil, err
		}
		return os.NewFile(uintptr(handle), path), nil
	}
	return nil, fmt.Errorf("could not allocate a unique private permission file")
}
