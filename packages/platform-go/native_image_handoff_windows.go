//go:build windows

package platform

import (
	"context"
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"path/filepath"
	"strings"
	"sync"
)

func verifyImmutableNativePath(ctx context.Context, path string, directory bool) error {
	held, e := windowsEnsureExistingBrokerParents(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	h, e := windowsOpenBrokerPath(path, directory, windows.FILE_READ_ATTRIBUTES)
	if e != nil {
		return e
	}
	defer windows.CloseHandle(h)
	// Ancestors may permit unrelated sibling creation, but the exact execution
	// object/directory must grant no foreign mutation at all.
	if e = windowsValidateBrokerHandle(h, directory, false); e != nil {
		return e
	}
	return ctx.Err()
}
func prepareImmutableNativeDirectory(ctx context.Context, path string) (func() error, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	held, e := windowsEnsureBrokerDirectory(path)
	if e != nil {
		return nil, e
	}
	var once sync.Once
	var result error
	return func() error {
		once.Do(func() {
			for _, h := range held {
				if e := windows.CloseHandle(h); e != nil {
					result = e
				}
			}
		})
		return result
	}, nil
}
func writeImmutableNativeFile(ctx context.Context, path string, write func(io.Writer) error) error {
	held, e := windowsEnsureExistingBrokerParents(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	if e = ctx.Err(); e != nil {
		return e
	}
	return windowsCreateSealedBrokerFile(path, func(w io.Writer) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := write(w); e != nil {
			return e
		}
		return ctx.Err()
	})
}
func frozenOwnedServiceEnvironment(ctx context.Context, ref ServiceRef) ([]string, error) {
	p, e := readWindowsOperatorPayload(ref.Intent)
	if e != nil {
		return nil, e
	}
	if !windowsSameNative(windowsRef(p), OperationRef(ref)) {
		return nil, ErrOperationIdentity
	}
	record, e := readWindowsOwnedRecord(ref.Intent)
	if e != nil {
		return nil, e
	}
	if ref.RootPID <= 0 || record.PID != ref.RootPID || record.Start != ref.RootStartTime || record.Incarnation != ref.Native.RootIncarnation {
		return nil, ErrOperationIdentity
	}
	stamp, alive, e := windowsOwnedProcessStart(ref.RootPID)
	if e != nil {
		return nil, e
	}
	if !alive || stamp != ref.RootStartTime {
		return nil, ErrOperationIdentity
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	return append([]string(nil), p.Spec.Env...), nil
}

func grantImmutableNativeCodeReaders(ctx context.Context, path string, readers []string) error {
	program, e := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if e != nil {
		return e
	}
	codeRoot := filepath.Join(program, "Vrooli")
	relative, e := filepath.Rel(codeRoot, path)
	if e != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ErrOperationIdentity
	}
	var sids []*windows.SID
	for _, reader := range readers {
		if !strings.HasPrefix(reader, "sid:") {
			return ErrOperationIdentity
		}
		sid, e := windows.StringToSid(strings.TrimPrefix(reader, "sid:"))
		if e != nil || "sid:"+sid.String() != reader {
			return ErrOperationIdentity
		}
		name, domain, kind, e := sid.LookupAccount("")
		if e != nil {
			return e
		}
		if kind != windows.SidTypeUser || !validNativeObserverPrincipal("windows", reader) {
			return ErrOperationIdentity
		}
		account := name
		if domain != "" {
			account = domain + `\` + name
		}
		actual, _, _, e := windows.LookupSID("", account)
		if e != nil || !windows.EqualSid(actual, sid) {
			return ErrOperationIdentity
		}
		sids = append(sids, sid)
	}
	unlock, e := lockWindowsOwnedSubmission(ctx, OperationIntent{Kind: ownedServiceKind, Owner: "code-metadata-acl"})
	if e != nil {
		return e
	}
	defer unlock()
	held, e := windowsEnsureExistingBrokerParents(path)
	if e != nil {
		return e
	}
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	targets := []string{codeRoot}
	current := codeRoot
	if relative != "." {
		for _, part := range strings.Split(relative, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			targets = append(targets, current)
		}
	}
	for _, target := range targets {
		if e = ctx.Err(); e != nil {
			return e
		}
		h, e := windowsOpenBrokerPath(target, true, windows.FILE_READ_ATTRIBUTES|windows.WRITE_DAC)
		if e != nil {
			return e
		}
		sd, e := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if e != nil {
			windows.CloseHandle(h)
			return e
		}
		entries := make([]windows.EXPLICIT_ACCESS, 0, len(sids))
		for _, sid := range sids {
			entries = append(entries, windows.EXPLICIT_ACCESS{AccessPermissions: 0x1200A0, AccessMode: windows.GRANT_ACCESS, Trustee: windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_USER, TrusteeValue: windows.TrusteeValueFromSID(sid)}})
		}
		merged, e := windows.BuildSecurityDescriptor(nil, nil, entries, nil, sd)
		if e != nil {
			windows.CloseHandle(h)
			return e
		}
		acl, _, e := merged.DACL()
		if e == nil {
			e = windows.SetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
		}
		if e == nil {
			e = windowsValidateBrokerHandle(h, true, false)
		}
		windows.CloseHandle(h)
		if e != nil {
			return fmt.Errorf("platform: code ancestry metadata grant: %w", e)
		}
	}
	return nil
}
