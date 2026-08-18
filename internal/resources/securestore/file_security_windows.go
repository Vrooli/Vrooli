//go:build windows

package securestore

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/windows"
)

// RestrictCredentialFile replaces inherited permissions with a protected DACL
// granting access only to the current user, LocalSystem, and Administrators.
// FileMode is not a meaningful confidentiality boundary on Windows.
func RestrictCredentialFile(path string) error {
	return restrictCredentialPath(path, windows.SE_FILE_OBJECT)
}

func restrictCredentialDirectory(path string) error {
	return restrictCredentialPath(path, windows.SE_FILE_OBJECT)
}

func restrictCredentialPath(path string, objectType windows.SE_OBJECT_TYPE) error {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return fmt.Errorf("open current Windows token: %w", err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return fmt.Errorf("resolve current Windows user SID: %w", err)
	}
	system, err := windows.CreateWellKnownSid(windows.WinLocalSystemSid)
	if err != nil {
		return fmt.Errorf("resolve LocalSystem SID: %w", err)
	}
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return fmt.Errorf("resolve Administrators SID: %w", err)
	}
	var pin runtime.Pinner
	defer pin.Unpin()
	for _, sid := range []*windows.SID{user.User.Sid, system, admins} {
		pin.Pin(sid)
	}
	const permissions = windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE
	entries := []windows.EXPLICIT_ACCESS{
		{AccessPermissions: permissions, AccessMode: windows.SET_ACCESS, Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_USER, TrusteeValue: windows.TrusteeValueFromSID(user.User.Sid)}},
		{AccessPermissions: permissions, AccessMode: windows.SET_ACCESS, Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_GROUP, TrusteeValue: windows.TrusteeValueFromSID(system)}},
		{AccessPermissions: permissions, AccessMode: windows.SET_ACCESS, Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_GROUP, TrusteeValue: windows.TrusteeValueFromSID(admins)}},
	}
	acl, err := windows.ACLFromEntries(entries, nil)
	if err != nil {
		return fmt.Errorf("build credential DACL: %w", err)
	}
	if err := windows.SetNamedSecurityInfo(path, objectType, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return fmt.Errorf("set credential DACL: %w", err)
	}
	return nil
}
