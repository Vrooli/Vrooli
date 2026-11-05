//go:build windows

package securestore

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// nativeDefault uses the Windows Credential Manager through CredWriteW/CredReadW/
// CredDeleteW. The value travels inside a CREDENTIALW structure, so it never
// appears in a process argument — which is why cmdkey, whose value path is
// argv, is not used here.
func nativeDefault() Store { return credentialManagerStore{} }

var (
	advapi32       = windows.NewLazySystemDLL("advapi32.dll")
	procCredWriteW = advapi32.NewProc("CredWriteW")
	procCredReadW  = advapi32.NewProc("CredReadW")
	procCredFree   = advapi32.NewProc("CredFree")
	procCredDelete = advapi32.NewProc("CredDeleteW")
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

// credentialW mirrors CREDENTIALW from wincred.h. Field order and width are
// load-bearing: the struct is passed to the API by pointer.
type credentialW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type credentialManagerStore struct{}

func (credentialManagerStore) AdapterName() string { return "windows-credential-manager" }

// targetName namespaces every Vrooli credential inside the shared per-user
// Credential Manager store so it cannot collide with an unrelated entry.
func targetName(service, key string) string {
	return "Vrooli:" + service + ":" + key
}

func (credentialManagerStore) Put(service, key, value string) error {
	target, err := windows.UTF16PtrFromString(targetName(service, key))
	if err != nil {
		return fmt.Errorf("%w: encode credential target: %v", ErrUnavailable, err)
	}
	userName, err := windows.UTF16PtrFromString(key)
	if err != nil {
		return fmt.Errorf("%w: encode credential user name: %v", ErrUnavailable, err)
	}
	blob := []byte(value)
	credential := credentialW{
		Type:               credTypeGeneric,
		TargetName:         target,
		Persist:            credPersistLocalMachine,
		CredentialBlobSize: uint32(len(blob)),
		UserName:           userName,
	}
	if len(blob) > 0 {
		credential.CredentialBlob = &blob[0]
	}
	result, _, callErr := procCredWriteW.Call(uintptr(unsafe.Pointer(&credential)), 0)
	if result == 0 {
		return classifyWindowsCredentialError("store secure resource material", callErr)
	}
	return nil
}

func (credentialManagerStore) Get(service, key string) (string, error) {
	target, err := windows.UTF16PtrFromString(targetName(service, key))
	if err != nil {
		return "", fmt.Errorf("%w: encode credential target: %v", ErrUnavailable, err)
	}
	var pointer *credentialW
	result, _, callErr := procCredReadW.Call(
		uintptr(unsafe.Pointer(target)),
		credTypeGeneric,
		0,
		uintptr(unsafe.Pointer(&pointer)),
	)
	if result == 0 {
		return "", classifyWindowsCredentialError("read secure resource material", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(pointer)))
	if pointer.CredentialBlobSize == 0 || pointer.CredentialBlob == nil {
		return "", nil
	}
	blob := unsafe.Slice(pointer.CredentialBlob, pointer.CredentialBlobSize)
	return string(blob), nil
}

func (credentialManagerStore) Delete(service, key string) error {
	target, err := windows.UTF16PtrFromString(targetName(service, key))
	if err != nil {
		return fmt.Errorf("%w: encode credential target: %v", ErrUnavailable, err)
	}
	result, _, callErr := procCredDelete.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0)
	if result == 0 {
		err := classifyWindowsCredentialError("delete secure resource material", callErr)
		if errors.Is(err, ErrNotFound) {
			// Deleting an absent credential is the state the caller asked for.
			return nil
		}
		return err
	}
	return nil
}

// classifyWindowsCredentialError keeps all three operations answering
// identically for identical host conditions, exactly as the libsecret adapter
// does.
func classifyWindowsCredentialError(stage string, err error) error {
	if errors.Is(err, windows.ERROR_NOT_FOUND) {
		return fmt.Errorf("%w: %s", ErrNotFound, stage)
	}
	if errors.Is(err, windows.ERROR_MOD_NOT_FOUND) || errors.Is(err, windows.ERROR_PROC_NOT_FOUND) {
		return fmt.Errorf("%w: advapi32 credential API is unavailable on this host", ErrAbsent)
	}
	detail := "unknown error"
	if err != nil {
		detail = err.Error()
	}
	return fmt.Errorf("%w: %s: %s", ErrUnavailable, stage, strings.TrimSpace(detail))
}
