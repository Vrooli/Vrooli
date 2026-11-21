//go:build windows

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func windowsPrivatePrincipals(p PrivatePathPolicy) (owner, runtime string, err error) {
	if p.RuntimeReadOnly && p.RuntimeTraverseOnly {
		return "", "", errors.New("ambiguous private runtime access policy")
	}
	user, e := windows.GetCurrentThreadEffectiveToken().GetTokenUser()
	if e != nil {
		return "", "", e
	}
	current := user.User.Sid.String()
	if p.OwnerPrincipal == "" {
		owner = current
	} else {
		owner = strings.TrimPrefix(p.OwnerPrincipal, "sid:")
	}
	if p.RuntimePrincipal != "" {
		runtime = strings.TrimPrefix(p.RuntimePrincipal, "sid:")
	}
	for _, sid := range []string{owner, runtime} {
		if sid != "" {
			parsed, e := windows.StringToSid(sid)
			if e != nil || parsed.String() != sid {
				return "", "", errors.New("invalid private path principal")
			}
		}
	}
	if current != owner && current != runtime && current != "S-1-5-18" {
		return "", "", errors.New("private path operation is outside its explicit owner/runtime principals")
	}
	return owner, runtime, nil
}

func nativePrivatePathOwner(path string, _ os.FileInfo) (string, error) {
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return "", err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil {
		return "", errors.Join(errors.New("Windows private path has no owner"), err)
	}
	return "sid:" + owner.String(), nil
}

func windowsPrivateDescriptor(p PrivatePathPolicy) (*windows.SECURITY_DESCRIPTOR, error) {
	owner, runtime, e := windowsPrivatePrincipals(p)
	if e != nil {
		return nil, e
	}
	inherit := "OICI"
	if p.NoInherit {
		inherit = ""
	}
	sddl := "D:P(A;" + inherit + ";FA;;;" + owner + ")(A;" + inherit + ";FA;;;SY)"
	if runtime != "" && runtime != owner {
		sddl += fmt.Sprintf("(A;%s;0x%x;;;%s)", inherit, windowsRuntimePrivateRights(p), runtime)
	}
	return windows.SecurityDescriptorFromString(sddl)
}

func verifyNativePathAccess(path string, info os.FileInfo, p PrivatePathPolicy) error {
	return verifyNativePrivatePath(path, info, p)
}
func configureNativeReadOnlyPathAccess(path string, info os.FileInfo, p PrivatePathPolicy) error {
	prior := p
	prior.NoInherit = false
	if err := verifyNativePathAccess(path, info, prior); err != nil {
		return err
	}
	descriptor, err := windowsPrivateDescriptor(p)
	if err != nil {
		return err
	}
	acl, _, err := descriptor.DACL()
	if err != nil {
		return err
	}
	handle, release, err := openWindowsPrivateObject(path, windows.READ_CONTROL|windows.WRITE_DAC, info)
	if err != nil {
		return err
	}
	defer release()
	return windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}
func verifyNativePrivatePath(path string, _ os.FileInfo, p PrivatePathPolicy) error {
	owner, runtime, e := windowsPrivatePrincipals(p)
	if e != nil {
		return e
	}
	sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return e
	}
	actual, _, e := sd.Owner()
	if e != nil || actual == nil {
		return errors.Join(errors.New("private Windows path has no owner"), e)
	}
	sid := actual.String()
	if err := verifyPrivateOwner(sid, owner, runtime, "S-1-5-18", p); err != nil {
		return err
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return errors.Join(errors.New("private Windows path requires an explicit non-null DACL"), e)
	}
	if acl.AceCount == 0 {
		return errors.New("private Windows path DACL has no owner access")
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			return e
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return errors.New("private Windows path has an unsupported ACE type")
		}
		if p.NoInherit && ace.Header.AceFlags&(windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE) != 0 {
			return errors.New("private mixed-policy root must not propagate ACLs into its children")
		}
		allowed := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if allowed != owner && allowed != runtime && allowed != "S-1-5-18" {
			return fmt.Errorf("private Windows path grants an unapproved principal %s", allowed)
		}
		if allowed == runtime && runtime != owner && ace.Mask&^(windowsRuntimePrivateRights(p)) != 0 {
			return errors.New("private Windows runtime principal has administrative file rights")
		}
	}
	return nil
}

func windowsRuntimePrivateRights(p PrivatePathPolicy) windows.ACCESS_MASK {
	if p.RuntimeTraverseOnly {
		return 0x1200a0
	}
	if p.RuntimeReadOnly {
		return 0x1200a9
	}
	return 0x1301bf
}

func ensureNativePrivateDirectory(path string, p PrivatePathPolicy) error {
	info, e := os.Lstat(path)
	if e == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("private Windows directory is an alias or non-directory")
		}
		return verifyNativePrivatePath(path, info, p)
	}
	if !os.IsNotExist(e) {
		return e
	}
	parent := filepath.Dir(path)
	if parent == path {
		return e
	}
	// Missing ancestors receive only current-owner/SYSTEM rights, never an
	// implicit service grant to the operator's whole home or runtime root.
	if _, e = os.Lstat(parent); os.IsNotExist(e) {
		if e = ensureNativePrivateDirectory(parent, PrivatePathPolicy{}); e != nil {
			return e
		}
	} else if e != nil {
		return e
	}
	sd, e := windowsPrivateDescriptor(p)
	if e != nil {
		return e
	}
	native, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return e
	}
	attrs := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	e = windows.CreateDirectory(native, &attrs)
	if e != nil && !errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
		return e
	}
	info, e = os.Lstat(path)
	if e != nil {
		return e
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("private Windows directory changed during creation")
	}
	return verifyNativePrivatePath(path, info, p)
}

// ConfigurePrivateDirectoryAccess is an explicit provisioning operation.
// Existing unknown grants are refused rather than silently repaired.
func ConfigurePrivateDirectoryAccess(path string, p PrivatePathPolicy) error {
	if e := EnsurePrivateDirectory(path, p); e != nil {
		return e
	}
	sd, e := windowsPrivateDescriptor(p)
	if e != nil {
		return e
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	info, e := os.Lstat(path)
	if e != nil {
		return e
	}
	handle, release, e := openWindowsPrivateObject(path, windows.READ_CONTROL|windows.WRITE_DAC, info)
	if e != nil {
		return e
	}
	e = windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
	release()
	if e != nil {
		return e
	}
	return configureWindowsPrivateOwner(path, p)
}

func configureWindowsPrivateOwner(path string, p PrivatePathPolicy) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	return configureWindowsPrivateOwnerWithInfo(path, p, info)
}
func configureWindowsPrivateOwnerWithInfo(path string, p PrivatePathPolicy, info os.FileInfo) error {
	owner, _, err := windowsPrivatePrincipals(p)
	if err != nil {
		return err
	}
	actual, err := nativePrivatePathOwner(path, info)
	if err != nil {
		return err
	}
	if actual == "sid:"+owner {
		return nil
	}
	sid, err := windows.StringToSid(owner)
	if err != nil {
		return err
	}
	// Use a dedicated goroutine/thread. If reverting impersonation fails, the
	// goroutine exits while still locked, so Go destroys that thread instead of
	// returning a privileged thread to the pool.
	result := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		impersonated := false
		e := func() error {
			var prior windows.Token
			if err := windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &prior); err == nil {
				prior.Close()
				return errors.New("private owner setup refuses an already impersonating thread")
			} else if !errors.Is(err, windows.ERROR_NO_TOKEN) {
				return err
			}
			if err := windows.ImpersonateSelf(windows.SecurityImpersonation); err != nil {
				return err
			}
			impersonated = true
			var token windows.Token
			if err := windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, true, &token); err != nil {
				return err
			}
			defer token.Close()
			name, err := windows.UTF16PtrFromString("SeRestorePrivilege")
			if err != nil {
				return err
			}
			var luid windows.LUID
			if err = windows.LookupPrivilegeValue(nil, name, &luid); err != nil {
				return err
			}
			privileges := windows.Tokenprivileges{PrivilegeCount: 1, Privileges: [1]windows.LUIDAndAttributes{{Luid: luid, Attributes: windows.SE_PRIVILEGE_ENABLED}}}
			if err = windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil); err != nil {
				return fmt.Errorf("explicit native owner provisioning requires restore privilege: %w", err)
			}
			handle, release, err := openWindowsPrivateObject(path, windows.READ_CONTROL|windows.WRITE_OWNER, info)
			if err != nil {
				return err
			}
			defer release()
			return windows.SetSecurityInfo(handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION, sid, nil, nil, nil)
		}()
		if impersonated {
			revertErr := windows.RevertToSelf()
			e = errors.Join(e, revertErr)
			if revertErr == nil {
				runtime.UnlockOSThread()
			}
		} else {
			runtime.UnlockOSThread()
		}
		result <- e
	}()
	return <-result
}

// Pin every ancestor without delete sharing and reject reparse points before
// privileged ACL/owner changes. A junction replacement cannot redirect setup
// from its exact resource subtree to a different host object.
func openWindowsPrivateObject(path string, access uint32, expected os.FileInfo) (windows.Handle, func(), error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.HasPrefix(filepath.VolumeName(path), `\\`) {
		return 0, nil, errors.New("private Windows setup requires a canonical local path")
	}
	var ancestors []string
	for parent := filepath.Dir(path); ; parent = filepath.Dir(parent) {
		ancestors = append(ancestors, parent)
		if filepath.Dir(parent) == parent {
			break
		}
		if len(ancestors) > 128 {
			return 0, nil, errors.New("private Windows path depth exceeds bound")
		}
	}
	var held []windows.Handle
	closeParents := func() {
		for i := len(held) - 1; i >= 0; i-- {
			windows.CloseHandle(held[i])
		}
	}
	open := func(p string, rights uint32) (windows.Handle, error) {
		native, err := windows.UTF16PtrFromString(p)
		if err != nil {
			return 0, err
		}
		h, err := windows.CreateFile(native, rights, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		if err != nil {
			return 0, err
		}
		var info windows.ByHandleFileInformation
		if err = windows.GetFileInformationByHandle(h, &info); err != nil || info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			windows.CloseHandle(h)
			return 0, errors.Join(errors.New("private Windows path has a reparse point or unavailable identity"), err)
		}
		return h, nil
	}
	for i := len(ancestors) - 1; i >= 0; i-- {
		h, err := open(ancestors[i], windows.FILE_READ_ATTRIBUTES)
		if err != nil {
			closeParents()
			return 0, nil, err
		}
		held = append(held, h)
	}
	h, err := open(path, access)
	if err != nil {
		closeParents()
		return 0, nil, err
	}
	file := os.NewFile(uintptr(h), path)
	actual, err := file.Stat()
	if err != nil || expected != nil && !os.SameFile(expected, actual) {
		file.Close()
		closeParents()
		return 0, nil, errors.Join(errors.New("private Windows setup object identity changed"), err)
	}
	return h, func() { file.Close(); closeParents() }, nil
}

func assignPrivateWriterOwner(file *os.File, p PrivatePathPolicy) error {
	current, err := windows.GetCurrentThreadEffectiveToken().GetTokenUser()
	if err != nil {
		return err
	}
	if current.User.Sid.String() != "S-1-5-18" || p.OwnerPrincipal == "" {
		return nil
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return configureWindowsPrivateOwnerWithInfo(file.Name(), p, info)
}

func configurePrivateWriterAccess(*os.File, PrivatePathPolicy) error { return nil }

func createNativePrivateTemp(parent string, p PrivatePathPolicy) (privateFileTemporary, error) {
	info, err := os.Lstat(parent)
	if err != nil {
		return privateFileTemporary{}, err
	}
	_, release, err := openWindowsPrivateObject(parent, windows.FILE_READ_ATTRIBUTES, info)
	if err != nil {
		return privateFileTemporary{}, err
	}
	name, err := privateTemporaryName()
	if err != nil {
		release()
		return privateFileTemporary{}, err
	}
	path := filepath.Join(parent, name)
	sd, err := windowsPrivateDescriptor(p)
	if err != nil {
		release()
		return privateFileTemporary{}, err
	}
	native, err := windows.UTF16PtrFromString(path)
	if err != nil {
		release()
		return privateFileTemporary{}, err
	}
	attrs := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	handle, err := windows.CreateFile(native, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, &attrs, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		release()
		return privateFileTemporary{}, err
	}
	file := os.NewFile(uintptr(handle), path)
	return privateFileTemporary{File: file, Name: path, Publish: func(to string) error {
		if filepath.Dir(to) != parent {
			return errors.New("private publication target changed its pinned directory")
		}
		return publishNativePrivateFile(path, to)
	}, Cleanup: func() { file.Close(); os.Remove(path); release() }}, nil
}

func openNativePrivateLock(path string, p PrivatePathPolicy) (*os.File, func(), error) {
	parent := filepath.Dir(path)
	info, err := os.Lstat(parent)
	if err != nil {
		return nil, nil, err
	}
	_, release, err := openWindowsPrivateObject(parent, windows.FILE_READ_ATTRIBUTES, info)
	if err != nil {
		return nil, nil, err
	}
	sd, err := windowsPrivateDescriptor(p)
	if err != nil {
		release()
		return nil, nil, err
	}
	native, err := windows.UTF16PtrFromString(path)
	if err != nil {
		release()
		return nil, nil, err
	}
	attrs := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	handle, err := windows.CreateFile(native, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, &attrs, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	created := err == nil
	if errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		handle, err = windows.CreateFile(native, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	}
	if err != nil {
		release()
		return nil, nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	closeAll := func() { file.Close(); release() }
	var facts windows.ByHandleFileInformation
	if err = windows.GetFileInformationByHandle(handle, &facts); err != nil || facts.FileAttributes&(windows.FILE_ATTRIBUTE_DIRECTORY|windows.FILE_ATTRIBUTE_REPARSE_POINT) != 0 || facts.NumberOfLinks != 1 {
		closeAll()
		return nil, nil, errors.Join(errors.New("private lock must be a regular single-link file"), err)
	}
	if created {
		if err = assignPrivateWriterOwner(file, p); err != nil {
			closeAll()
			return nil, nil, err
		}
	}
	return file, closeAll, nil
}

func publishNativePrivateFile(from, to string) error {
	source, e := windows.UTF16PtrFromString(from)
	if e != nil {
		return e
	}
	target, e := windows.UTF16PtrFromString(to)
	if e != nil {
		return e
	}
	return windows.MoveFileEx(source, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func removeNativePrivateFile(path string) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-retired-*")
	if err != nil {
		return err
	}
	tombstone := tmp.Name()
	if err = tmp.Close(); err != nil {
		os.Remove(tombstone)
		return err
	}
	if err = publishNativePrivateFile(path, tombstone); err != nil {
		os.Remove(tombstone)
		return err
	}
	return os.Remove(tombstone)
}
