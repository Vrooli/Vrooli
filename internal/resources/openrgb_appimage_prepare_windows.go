//go:build windows

package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	platform "github.com/vrooli/platform-go"
	"golang.org/x/sys/windows"
)

func openRGBStageReadPrincipal(ctx context.Context) (string, error) {
	owner, err := readOpenRGBNativeOwnerContext(ctx)
	if err != nil {
		return "", err
	}
	operator := strings.TrimPrefix(owner.OwnerPrincipal, "sid:")
	sid, err := windows.StringToSid(operator)
	if err != nil || sid.String() != operator || operator == "S-1-5-18" || operator == "S-1-5-32-544" || owner.RuntimePrincipal != owner.OwnerPrincipal || owner.NativeAccount != owner.OwnerPrincipal {
		return "", errors.New("native artifact setup requires the exact ordinary operator as backend identity")
	}
	return operator, nil
}

func openRGBStagePreparationOwner() (openRGBStagePreparation, error) {
	p, err := openRGBReleaseProfileForCurrentHost()
	if err != nil {
		return openRGBStagePreparation{}, err
	}
	return openRGBStagePreparation{payloadRelative: p.Payload, extract: func(ctx context.Context, f *os.File, path string) error {
		return extractOpenRGBZipWithIO(ctx, f, path, p, openRGBSealedTreeIO())
	}, seal: sealOpenRGBStage, verifySeal: verifyOpenRGBAppDirSeal, publish: func(a, b string) error {
		from, err := windows.UTF16PtrFromString(a)
		if err != nil {
			return err
		}
		to, err := windows.UTF16PtrFromString(b)
		if err != nil {
			return err
		}
		return windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH)
	}, syncDir: syncOpenRGBStageDirectory}, nil
}

func syncOpenRGBStageDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func openRGBStageHandle(path string) (windows.Handle, *windows.ByHandleFileInformation, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, nil, err
	}
	h, err := windows.CreateFile(p, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return 0, nil, err
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		windows.CloseHandle(h)
		return 0, nil, err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 && info.NumberOfLinks != 1 {
		windows.CloseHandle(h)
		return 0, nil, errors.New("native artifact rejects reparse points and multiply-linked files")
	}
	return h, &info, nil
}

func openRGBStageSecurity(path string) (*windows.SECURITY_DESCRIPTOR, error) {
	h, _, err := openRGBStageHandle(path)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(h)
	return windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
}

func verifyOpenRGBStageWindowsACL(path, operatorSID string, exact bool) error {
	if err := platform.VerifyImmutableNativePath(context.Background(), path, func() bool { info, err := os.Lstat(path); return err == nil && info.IsDir() }()); err != nil {
		return err
	}
	if !exact {
		return nil
	}
	sd, err := openRGBStageSecurity(path)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil || owner == nil || owner.String() != "S-1-5-18" && owner.String() != "S-1-5-32-544" {
		return errors.Join(errors.New("sealed artifact is not machine-owned"), err)
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		return errors.Join(errors.New("sealed artifact lacks protected finite DACL"), err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		return errors.Join(errors.New("sealed artifact DACL inherits authority"), err)
	}
	seen := map[string]bool{}
	expected := map[string]uint32{"S-1-5-18": 0x1f01ff, "S-1-5-32-544": 0x1f01ff, operatorSID: 0x1200a9}
	if len(expected) != 3 || acl.AceCount != 3 {
		return errors.New("sealed artifact identities are not three distinct finite principals")
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(acl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags != 0 {
			return errors.New("sealed artifact contains inherited or unsupported ACE")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		mask, ok := expected[sid]
		if !ok || seen[sid] || uint32(ace.Mask) != mask {
			return errors.New("sealed artifact grants exceed exact privileged-owner and readonly-runtime roles")
		}
		seen[sid] = true
	}
	return nil
}

func configureOpenRGBSealedAncestors(ctx context.Context, parent, root string) error {
	operator, err := openRGBStageReadPrincipal(ctx)
	if err != nil {
		return err
	}
	program, err := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if err != nil {
		return err
	}
	codeBase := filepath.Join(program, "Vrooli")
	// Explicit test-class roots cannot widen this privileged code namespace.
	canonical := filepath.Join(codeBase, "OpenRGB")
	if !strings.EqualFold(root, canonical) {
		return errors.New("Windows machine artifact publication requires the canonical ProgramFiles code root")
	}
	return platform.GrantImmutableNativeCodeReaders(ctx, parent, []string{"sid:" + operator})
}

func ensureOpenRGBStageParent(ctx context.Context, path string) error {
	release, err := openRGBStageParentLease(ctx, path)
	if err != nil {
		return err
	}
	return release()
}

func verifyOpenRGBPendingOwner(path string) error {
	return platform.VerifyImmutableNativePath(context.Background(), path, true)
}

func sealOpenRGBStage(ctx context.Context, root string) error {
	if err := requireOpenRGBNativeStagePublisher(); err != nil {
		return err
	}
	operator, err := openRGBStageReadPrincipal(ctx)
	if err != nil {
		return err
	}
	count := 0
	var total int64
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		count++
		if count > 1024 {
			return errors.New("OpenRGB stage exceeds entry bound")
		}
		h, info, err := openRGBStageHandle(path)
		if err != nil {
			return err
		}
		windows.CloseHandle(h)
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		if info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0 {
			if !stat.Mode().IsRegular() || stat.Size() > (256<<20)-total {
				return errors.New("OpenRGB stage has unsafe types or sizes")
			}
			total += stat.Size()
			if err := os.Chmod(path, 0o600); err != nil {
				return err
			}
			if err := syncOpenRGBStageDirectory(path); err != nil {
				return err
			}
		}
		// Objects were created atomically as BA/SYSTEM-owned, never chowned from
		// mutable operator inodes. Add readonly roles only after their bytes exist.
		sd, err := windows.SecurityDescriptorFromString(fmt.Sprintf("O:BAG:BAD:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FRFX;;;%s)", operator))
		if err != nil {
			return err
		}
		acl, _, err := sd.DACL()
		if err != nil {
			return err
		}
		return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
	})
	if err != nil {
		return err
	}
	return filepath.WalkDir(root, func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, 0o555)
	})
}

func verifyOpenRGBAppDirSeal(ctx context.Context, root string) error {
	// Serving verification runs with the admitted actual operator token and a
	// frozen capsule environment. It must not rediscover owner.json through an
	// ambient global-root variable that the wrapper deliberately omits.
	security, err := platform.ProcessSecurity(os.Getpid())
	if err != nil {
		return err
	}
	operator := security.Principal
	sid, err := windows.StringToSid(operator)
	if err != nil || sid.String() != operator || operator == "S-1-5-18" || operator == "S-1-5-32-544" {
		return errors.Join(errors.New("sealed artifact verification requires the actual ordinary operator principal"), err)
	}
	if err := platform.VerifyImmutableNativePath(ctx, filepath.Dir(root), true); err != nil {
		return err
	}
	count := 0
	var size int64
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		count++
		if count > 512 {
			return errors.New("OpenRGB sealed bundle exceeds entry bound")
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if !info.Mode().IsRegular() || info.Size() > (128<<20)-size {
				return errors.New("OpenRGB bundle has unsafe types or size")
			}
			size += info.Size()
		}
		return verifyOpenRGBStageWindowsACL(path, operator, true)
	})
}
