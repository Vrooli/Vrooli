//go:build windows

package msvcruntime

import (
	"debug/pe"
	"errors"
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	crypt32          = windows.NewLazySystemDLL("crypt32.dll")
	cryptMsgGetParam = crypt32.NewProc("CryptMsgGetParam")
	cryptMsgClose    = crypt32.NewProc("CryptMsgClose")
)

// Check reads native System32 files and cached signature trust only. It does
// not execute/load a DLL, probe hardware, install software, use PATH or contact
// a certificate server. Microsoft v14 ABI compatibility and exact imports are
// independently checked before OpenRGB starts.
func Check() (string, error) {
	if windows.RtlGetVersion().MajorVersion < 10 {
		return "", errors.New("native OpenRGB requires Windows10+ system UCRT")
	}
	root, err := windows.GetSystemDirectory()
	if err != nil {
		return "", err
	}
	var version string
	for _, name := range requiredRuntimeLibraries() {
		path := filepath.Join(root, name)
		f, err := pe.Open(path)
		if err != nil {
			return "", fmt.Errorf("Microsoft C++ x64 runtime missing %s; install system/msvc-runtime through Scenario Dependency Analyzer: %w", name, err)
		}
		valid := f.Machine == pe.IMAGE_FILE_MACHINE_AMD64
		f.Close()
		if !valid {
			return "", errors.New("Microsoft C++ runtime requires x64 PE ABI")
		}
		major, minor, build, revision, err := fileVersion(path)
		if err != nil {
			return "", err
		}
		if major != 14 || minor < 44 {
			return "", fmt.Errorf("Microsoft v14 runtime >=14.44 required; found %d.%d", major, minor)
		}
		if err := verifyMicrosoftSignature(path); err != nil {
			return "", fmt.Errorf("Microsoft runtime provenance %s: %w", name, err)
		}
		if version == "" {
			version = fmt.Sprintf("%d.%d.%d.%d", major, minor, build, revision)
		}
	}
	// UCRT is an OS-serviced dependency, distinct from the redistributable.
	crt, err := pe.Open(filepath.Join(root, "ucrtbase.dll"))
	if err != nil {
		return "", fmt.Errorf("Windows system UCRT unavailable: %w", err)
	}
	defer crt.Close()
	if crt.Machine != pe.IMAGE_FILE_MACHINE_AMD64 {
		return "", errors.New("system UCRT has another ABI")
	}
	return version, nil
}

func fileVersion(path string) (uint32, uint32, uint32, uint32, error) {
	size, err := windows.GetFileVersionInfoSize(path, nil)
	if err != nil || size < 52 || size > 1<<20 {
		return 0, 0, 0, 0, errors.Join(errors.New("runtime has no bounded native version resource"), err)
	}
	data := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&data[0])); err != nil {
		return 0, 0, 0, 0, err
	}
	var fixed *windows.VS_FIXEDFILEINFO
	var n uint32
	if err := windows.VerQueryValue(unsafe.Pointer(&data[0]), "\\", unsafe.Pointer(&fixed), &n); err != nil {
		return 0, 0, 0, 0, err
	}
	if fixed == nil || n < uint32(unsafe.Sizeof(*fixed)) || fixed.Signature != 0xfeef04bd {
		return 0, 0, 0, 0, errors.New("invalid runtime fixed version resource")
	}
	return fixed.FileVersionMS >> 16, fixed.FileVersionMS & 0xffff, fixed.FileVersionLS >> 16, fixed.FileVersionLS & 0xffff, nil
}

func verifyMicrosoftSignature(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	info := windows.WinTrustFileInfo{FilePath: p}
	info.Size = uint32(unsafe.Sizeof(info))
	trust := windows.WinTrustData{UIChoice: windows.WTD_UI_NONE, UnionChoice: windows.WTD_CHOICE_FILE, FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(&info), StateAction: windows.WTD_STATEACTION_VERIFY, ProvFlags: windows.WTD_CACHE_ONLY_URL_RETRIEVAL | windows.WTD_REVOCATION_CHECK_NONE}
	trust.Size = uint32(unsafe.Sizeof(trust))
	action := windows.GUID{Data1: 0x00aac56b, Data2: 0xcd44, Data3: 0x11d0, Data4: [8]byte{0x8c, 0xc2, 0x00, 0xc0, 0x4f, 0xc2, 0x95, 0xee}}
	err = windows.WinVerifyTrustEx(0, &action, &trust)
	defer func() {
		trust.StateAction = windows.WTD_STATEACTION_CLOSE
		_ = windows.WinVerifyTrustEx(0, &action, &trust)
	}()
	if err != nil {
		return err
	}
	var store, msg windows.Handle
	var encoding uint32
	if err := windows.CryptQueryObject(windows.CERT_QUERY_OBJECT_FILE, unsafe.Pointer(p), windows.CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED, windows.CERT_QUERY_FORMAT_FLAG_BINARY, 0, &encoding, nil, nil, &store, &msg, nil); err != nil {
		return err
	}
	defer windows.CertCloseStore(store, 0)
	defer cryptMsgClose.Call(uintptr(msg))
	// CMSG_SIGNER_CERT_INFO_PARAM binds the actual signer, not an arbitrary
	// Microsoft certificate included as a decoy in a PKCS7 certificate set.
	var n uint32
	r, _, e := cryptMsgGetParam.Call(uintptr(msg), 7, 0, 0, uintptr(unsafe.Pointer(&n)))
	if r == 0 || n == 0 || n > 1<<20 {
		return fmt.Errorf("read runtime signer identity: %v", e)
	}
	data := make([]byte, n)
	r, _, e = cryptMsgGetParam.Call(uintptr(msg), 7, 0, uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return fmt.Errorf("read runtime signer certificate: %v", e)
	}
	cert, err := windows.CertFindCertificateInStore(store, encoding, 0, windows.CERT_FIND_SUBJECT_CERT, unsafe.Pointer(&data[0]), nil)
	if err != nil {
		return err
	}
	defer windows.CertFreeCertificateContext(cert)
	count := windows.CertGetNameString(cert, windows.CERT_NAME_SIMPLE_DISPLAY_TYPE, 0, nil, nil, 0)
	if count == 0 || count > 4096 {
		return errors.New("runtime signer name unavailable")
	}
	name := make([]uint16, count)
	windows.CertGetNameString(cert, windows.CERT_NAME_SIMPLE_DISPLAY_TYPE, 0, nil, &name[0], count)
	if windows.UTF16ToString(name) != "Microsoft Corporation" {
		return errors.New("runtime signer is not Microsoft Corporation")
	}
	return nil
}

func CheckBundleImports(bundle string) error {
	root, err := windows.GetSystemDirectory()
	if err != nil {
		return err
	}
	return verifyBundleImports(bundle, root)
}
