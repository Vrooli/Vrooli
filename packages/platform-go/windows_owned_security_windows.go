//go:build windows

package platform

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Retained retired-host descriptor for historical broker fixtures/decoder code.
// Actual operator Jobs use their explicit operator DACL, never this descriptor.
const windowsNoPayloadJobSD = "D:P(A;;GA;;;SY)(A;;GA;;;BA)(A;;0x00000000;;;OW)"

func windowsObserverACE(observer, rights string) string {
	if observer == "" {
		return ""
	}
	return "(A;;" + rights + ";;;" + strings.TrimPrefix(observer, "sid:") + ")"
}

func resolveWindowsObserverSID(observer string) (*windows.SID, error) {
	if observer == "" {
		return nil, nil
	}
	if !validNativeObserverPrincipal("windows", observer) {
		return nil, ErrOperationIdentity
	}
	sid, e := windows.StringToSid(strings.TrimPrefix(observer, "sid:"))
	if e != nil {
		return nil, e
	}
	if "sid:"+sid.String() != observer {
		return nil, ErrOperationIdentity
	}
	name, domain, kind, e := sid.LookupAccount("")
	if e != nil {
		return nil, e
	}
	if kind != windows.SidTypeUser {
		return nil, ErrOperationIdentity
	}
	account := name
	if domain != "" {
		account = domain + `\` + name
	}
	confirmed, _, confirmedKind, e := windows.LookupSID("", account)
	if e != nil {
		return nil, e
	}
	if confirmedKind != windows.SidTypeUser || !windows.EqualSid(sid, confirmed) {
		return nil, ErrOperationIdentity
	}
	return sid, nil
}

// Explicit deletion makes privileged operations non-reenableable. The SDK's
// DISABLE_MAX flag wording is not used as evidence of privilege absence.
func nativePrivilegeDeletionList(token windows.Token) ([]windows.LUIDAndAttributes, error) {
	data := make([]byte, 8192)
	var size uint32
	if e := windows.GetTokenInformation(token, windows.TokenPrivileges, &data[0], uint32(len(data)), &size); e != nil {
		return nil, e
	}
	p := (*windows.Tokenprivileges)(unsafe.Pointer(&data[0]))
	if p.PrivilegeCount > 128 || uint64(unsafe.Offsetof(p.Privileges))+uint64(p.PrivilegeCount)*uint64(unsafe.Sizeof(p.Privileges[0])) > uint64(size) {
		return nil, ErrOperationIdentity
	}
	var traversal windows.LUID
	name, _ := windows.UTF16PtrFromString("SeChangeNotifyPrivilege")
	if e := windows.LookupPrivilegeValue(nil, name, &traversal); e != nil {
		return nil, e
	}
	out := make([]windows.LUIDAndAttributes, 0, p.PrivilegeCount)
	for _, privilege := range p.AllPrivileges() {
		if privilege.Luid != traversal {
			out = append(out, windows.LUIDAndAttributes{Luid: privilege.Luid})
		}
	}
	return out, nil
}
