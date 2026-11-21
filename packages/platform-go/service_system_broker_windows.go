//go:build windows

package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func nativeWindowsSystemBrokerRoots() (nativeSystemBrokerRoots, error) {
	program, e := windows.KnownFolderPath(windows.FOLDERID_ProgramFiles, 0)
	if e != nil {
		return nativeSystemBrokerRoots{}, e
	}
	data, e := windows.KnownFolderPath(windows.FOLDERID_ProgramData, 0)
	if e != nil {
		return nativeSystemBrokerRoots{}, e
	}
	system, e := windows.GetWindowsDirectory()
	if e != nil {
		return nativeSystemBrokerRoots{}, e
	}
	return nativeSystemBrokerRoots{Install: filepath.Join(program, "Vrooli", "RuntimeSupervisor"), State: filepath.Join(data, "Vrooli", "RuntimeSupervisor"), Unit: runtimeSupervisorService, SystemRoot: system}, nil
}

func nativeWindowsBrokerSD(directory bool) (*windows.SECURITY_DESCRIPTOR, error) {
	s := "O:BAG:BAD:P(A;;FA;;;SY)(A;;FA;;;BA)"
	if directory {
		s = "O:BAG:BAD:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
	}
	return windows.SecurityDescriptorFromString(s)
}

func windowsBrokerTrustedOwner(sid *windows.SID) bool {
	v := sid.String()
	return v == "S-1-5-18" || v == "S-1-5-32-544" || v == "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464"
}

func windowsValidateBrokerHandle(h windows.Handle, directory bool, anchor bool) error {
	var info windows.ByHandleFileInformation
	if e := windows.GetFileInformationByHandle(h, &info); e != nil {
		return e
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 || (info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0) != directory {
		return ErrOperationIdentity
	}
	sd, e := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return e
	}
	owner, _, e := sd.Owner()
	if e != nil || owner == nil || !windowsBrokerTrustedOwner(owner) {
		return ErrOperationIdentity
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return ErrOperationIdentity
	}
	writes := uint32(windows.GENERIC_ALL | windows.GENERIC_WRITE | windows.DELETE | windows.WRITE_DAC | windows.WRITE_OWNER | windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA | windows.FILE_WRITE_ATTRIBUTES | 0x40)
	if anchor {
		writes &^= windows.GENERIC_WRITE | windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA | windows.FILE_WRITE_EA | windows.FILE_WRITE_ATTRIBUTES
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var entry *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &entry); e != nil {
			return e
		}
		header := &entry.Header
		if header.AceFlags&windows.INHERIT_ONLY_ACE != 0 {
			continue
		}
		if header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			if header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
				continue
			}
			return ErrUnsupported
		}
		ace := entry
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if uint32(ace.Mask)&writes != 0 && !windowsBrokerTrustedOwner(sid) {
			return fmt.Errorf("%w: broker path grants write authority outside SYSTEM/Administrators/TrustedInstaller", ErrOperationIdentity)
		}
	}
	return nil
}

func windowsOpenBrokerPath(path string, directory bool, access uint32) (windows.Handle, error) {
	name, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return 0, e
	}
	flags := uint32(windows.FILE_FLAG_OPEN_REPARSE_POINT)
	if directory {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
	}
	h, e := windows.CreateFile(name, access|windows.READ_CONTROL, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, flags, 0)
	if e != nil {
		return 0, e
	}
	anchor := directory && strings.EqualFold(filepath.Clean(path), filepath.VolumeName(path)+string(filepath.Separator))
	if directory {
		for _, id := range []*windows.KNOWNFOLDERID{windows.FOLDERID_ProgramFiles, windows.FOLDERID_ProgramData} {
			known, lookupErr := windows.KnownFolderPath(id, 0)
			if lookupErr != nil {
				windows.CloseHandle(h)
				return 0, lookupErr
			}
			if strings.EqualFold(known, path) {
				anchor = true
			}
		}
	}
	if e = windowsValidateBrokerHandle(h, directory, anchor); e != nil {
		windows.CloseHandle(h)
		return 0, e
	}
	return h, nil
}

// Every privileged parent is verified and pinned against rename through the
// operation. Creation supplies the protected owner/DACL atomically.
func windowsEnsureBrokerDirectory(path string) ([]windows.Handle, error) {
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if volume == "" || !filepath.IsAbs(path) {
		return nil, ErrOperationIdentity
	}
	parts := strings.Split(strings.TrimPrefix(path, volume+string(filepath.Separator)), string(filepath.Separator))
	current := volume + string(filepath.Separator)
	var held []windows.Handle
	rootHandle, e := windowsOpenBrokerPath(current, true, windows.FILE_READ_ATTRIBUTES)
	if e != nil {
		return nil, e
	}
	held = append(held, rootHandle)
	fail := func(e error) ([]windows.Handle, error) {
		for _, h := range held {
			windows.CloseHandle(h)
		}
		return nil, e
	}
	// Known Windows roots permit ordinary users to create unrelated siblings;
	// the install chain itself starts at the existing ProgramFiles/ProgramData
	// child, whose write ACL is verified below.
	for _, part := range parts {
		current = filepath.Join(current, part)
		name, _ := windows.UTF16PtrFromString(current)
		sd, e := nativeWindowsBrokerSD(true)
		if e != nil {
			return fail(e)
		}
		sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
		e = windows.CreateDirectory(name, &sa)
		if e != nil && !errors.Is(e, windows.ERROR_ALREADY_EXISTS) {
			return fail(e)
		}
		h, e := windowsOpenBrokerPath(current, true, windows.FILE_READ_ATTRIBUTES)
		if e != nil {
			return fail(e)
		}
		held = append(held, h)
	}
	return held, nil
}

func windowsCreateSealedBrokerFile(path string, write func(io.Writer) error) error {
	var nonce [16]byte
	if _, e := rand.Read(nonce[:]); e != nil {
		return e
	}
	staged := path + ".stage-" + hex.EncodeToString(nonce[:])
	defer os.Remove(staged)
	sd, e := nativeWindowsBrokerSD(false)
	if e != nil {
		return e
	}
	name, _ := windows.UTF16PtrFromString(staged)
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	h, e := windows.CreateFile(name, windows.GENERIC_WRITE|windows.GENERIC_READ, windows.FILE_SHARE_READ, &sa, windows.CREATE_NEW, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		return e
	}
	file := os.NewFile(uintptr(h), staged)
	writeErr := write(file)
	syncErr := file.Sync()
	closeErr := file.Close()
	if e = errors.Join(writeErr, syncErr, closeErr); e != nil {
		return e
	}
	source, _ := windows.UTF16PtrFromString(staged)
	dest, _ := windows.UTF16PtrFromString(path)
	// No replace flag: a competing publisher or existing immutable version wins.
	return windows.MoveFileEx(source, dest, 0)
}

func windowsValidateBrokerImports(path string) error {
	image, e := os.Open(path)
	if e != nil {
		return e
	}
	defer image.Close()
	return verifyWindowsMaintainedLauncherPE(image, runtime.GOARCH)
}

func stageWindowsSystemBroker(p nativeSystemBrokerPlan) error {
	var held []windows.Handle
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	for _, dir := range []string{filepath.Dir(p.ImagePath)} {
		pins, e := windowsEnsureBrokerDirectory(dir)
		if e != nil {
			return e
		}
		held = append(held, pins...)
	}
	source, e := os.Open(p.Source)
	if e != nil {
		return e
	}
	defer source.Close()
	info, e := source.Stat()
	if e != nil || !info.Mode().IsRegular() {
		return ErrOperationIdentity
	}
	if h, e := windowsOpenBrokerPath(p.ImagePath, false, windows.GENERIC_READ); e == nil {
		file := os.NewFile(uintptr(h), p.ImagePath)
		e = verifyNativeSystemBrokerImage(file, p.SHA256)
		file.Close()
		if e != nil {
			return e
		}
	} else if errors.Is(e, windows.ERROR_FILE_NOT_FOUND) {
		if e = windowsCreateSealedBrokerFile(p.ImagePath, func(w io.Writer) error { return copyNativeSystemBrokerImage(source, w, p.SHA256) }); e != nil {
			return e
		}
	} else {
		return e
	}
	if e = windowsValidateBrokerImports(p.ImagePath); e != nil {
		return e
	}
	if h, e := windowsOpenBrokerPath(p.DescriptorPath, false, windows.GENERIC_READ); e == nil {
		file := os.NewFile(uintptr(h), p.DescriptorPath)
		raw, readErr := io.ReadAll(io.LimitReader(file, 64<<10+1))
		file.Close()
		if readErr != nil {
			return readErr
		}
		if string(raw) != string(p.Descriptor) {
			return ErrOperationIdentity
		}
	} else if errors.Is(e, windows.ERROR_FILE_NOT_FOUND) {
		if e = windowsCreateSealedBrokerFile(p.DescriptorPath, func(w io.Writer) error { _, e := w.Write(p.Descriptor); return e }); e != nil {
			return e
		}
	} else {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// The filtered operator runs only this sealed CLI with ordinary user
	// authority. Grant image read/execute, never descriptor/directory writes.
	descriptor, e := decodeNativeSystemServiceBrokerDescriptor(p.Descriptor)
	if e != nil {
		return e
	}
	if e = grantImmutableNativeCodeReaders(ctx, filepath.Dir(p.ImagePath), []string{"sid:" + strings.TrimPrefix(descriptor.Options.OperatorPrincipal, "windows:")}); e != nil {
		return e
	}
	for _, path := range []string{p.ImagePath, p.DescriptorPath} {
		if e = grantWindowsOwnedHostRead(ctx, path, descriptor.Options.OperatorUsername); e != nil {
			return e
		}
	}
	return nil
}

func windowsReadBrokerDescriptor(path string) (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, error) {
	return windowsReadBrokerDescriptorWithImageProof(path, true)
}
func windowsReadBrokerDescriptorWithImageProof(path string, imageProof bool) (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, error) {
	var d nativeSystemBrokerDescriptor
	var p nativeSystemBrokerPlan
	roots, e := nativeWindowsSystemBrokerRoots()
	if e != nil {
		return d, p, e
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return d, p, ErrOperationIdentity
	}
	relative, e := filepath.Rel(filepath.Join(roots.Install, "versions"), path)
	if e != nil {
		return d, p, e
	}
	parts := strings.Split(relative, string(filepath.Separator))
	if len(parts) != 2 || !validNativeBrokerDigest(parts[0]) || !strings.HasPrefix(parts[1], "broker-") || !strings.HasSuffix(parts[1], ".json") || !validNativeBrokerDigest(strings.TrimSuffix(strings.TrimPrefix(parts[1], "broker-"), ".json")) {
		return d, p, ErrOperationIdentity
	}
	held, e := windowsEnsureExistingBrokerParents(filepath.Dir(path))
	if e != nil {
		return d, p, e
	}
	defer func() {
		for _, h := range held {
			windows.CloseHandle(h)
		}
	}()
	h, e := windowsOpenBrokerPath(path, false, windows.GENERIC_READ)
	if e != nil {
		return d, p, e
	}
	file := os.NewFile(uintptr(h), path)
	raw, e := io.ReadAll(io.LimitReader(file, 64<<10+1))
	file.Close()
	if e != nil {
		return d, p, e
	}
	d, e = decodeNativeSystemServiceBrokerDescriptor(raw)
	if e != nil {
		return d, p, e
	}
	p, e = validateNativeSystemBrokerDescriptor(d, "windows", path, roots)
	if e != nil {
		return d, p, e
	}
	if !imageProof {
		return d, p, nil
	}
	h, e = windowsOpenBrokerPath(p.ImagePath, false, windows.GENERIC_READ)
	if e != nil {
		return d, p, e
	}
	image := os.NewFile(uintptr(h), p.ImagePath)
	e = verifyNativeSystemBrokerImage(image, p.SHA256)
	image.Close()
	if e != nil {
		return d, p, e
	}
	if e = windowsValidateBrokerImports(p.ImagePath); e != nil {
		return d, p, e
	}
	return d, p, nil
}

func windowsEnsureExistingBrokerParents(path string) ([]windows.Handle, error) {
	volume := filepath.VolumeName(path)
	parts := strings.Split(strings.TrimPrefix(path, volume+string(filepath.Separator)), string(filepath.Separator))
	current := volume + string(filepath.Separator)
	var held []windows.Handle
	rootHandle, e := windowsOpenBrokerPath(current, true, windows.FILE_READ_ATTRIBUTES)
	if e != nil {
		return nil, e
	}
	held = append(held, rootHandle)
	for _, part := range parts {
		current = filepath.Join(current, part)
		h, e := windowsOpenBrokerPath(current, true, windows.FILE_READ_ATTRIBUTES)
		if e != nil {
			for _, v := range held {
				windows.CloseHandle(v)
			}
			return nil, e
		}
		held = append(held, h)
	}
	return held, nil
}

// windowsInstalledTaskPlan validates the protected native definition before
// accepting its descriptor/image, even for an ordinary same-operator reader.
func windowsInstalledTaskPlan() (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, WindowsTaskSnapshot, error) {
	snapshot, e := WindowsTaskStatus(runtimeSupervisorService)
	if e != nil {
		return nativeSystemBrokerDescriptor{}, nativeSystemBrokerPlan{}, snapshot, e
	}
	return windowsInstalledTaskPlanFromSnapshot(snapshot)
}

func windowsInstalledTaskPlanFromSnapshot(snapshot WindowsTaskSnapshot) (nativeSystemBrokerDescriptor, nativeSystemBrokerPlan, WindowsTaskSnapshot, error) {
	var d nativeSystemBrokerDescriptor
	var p nativeSystemBrokerPlan
	task, e := parseWindowsTaskDefinition(snapshot.XML)
	if e != nil {
		return d, p, snapshot, e
	}
	var args []string
	// Command is a separate XML field, not a token needing shell interpretation.
	tail, e := windows.DecomposeCommandLine(task.Actions[0].Arguments)
	if e != nil {
		return d, p, snapshot, e
	}
	args = append([]string{task.Actions[0].Command}, tail...)
	if len(args) != 3 || args[1] != NativeSystemSupervisorVerb {
		return d, p, snapshot, ErrOperationIdentity
	}
	d, p, e = windowsReadBrokerDescriptor(args[2])
	if e != nil {
		return d, p, snapshot, e
	}
	sid, e := windows.StringToSid(strings.TrimPrefix(d.Options.OperatorPrincipal, "windows:"))
	if e != nil {
		return d, p, snapshot, e
	}
	if e = verifyWindowsTaskDescriptor(snapshot.SecurityDescriptor, sid); e != nil {
		return d, p, snapshot, e
	}
	actual, e := resolveWindowsTaskUserSID(task.Principals[0].UserID)
	if e != nil {
		return d, p, snapshot, e
	}
	if !windows.EqualSid(actual, sid) || !strings.EqualFold(args[0], p.ImagePath) || task.Actions[0].Directory != d.Options.OperatorHome || !windowsOperatorBootPolicy(task) {
		return d, p, snapshot, ErrOperationIdentity
	}
	return d, p, snapshot, nil
}

func registerWindowsSystemBroker(p nativeSystemBrokerPlan) (ServiceInstallResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if e := retireWindowsLegacySupervisor(); e != nil {
		return ServiceInstallResult{}, e
	}
	if _, _, old, e := windowsInstalledTaskPlan(); e == nil {
		if e = stopWindowsOperatorTask(ctx, old, true); e != nil {
			return ServiceInstallResult{}, e
		}
	} else if !taskNativeErrorMissing(e) {
		return ServiceInstallResult{}, e
	}
	artifact, e := RenderWindowsTaskXML(p.Definition)
	if e != nil {
		return ServiceInstallResult{}, e
	}
	if e = registerWindowsTask(runtimeSupervisorService, artifact.Files[0].Content); e != nil {
		return ServiceInstallResult{}, e
	}
	if e = windowsTaskControl(runtimeSupervisorService, "start"); e != nil {
		return ServiceInstallResult{}, e
	}
	if e = awaitWindowsTaskAction(ctx); e != nil {
		return ServiceInstallResult{}, e
	}
	_, actual, snapshot, e := windowsInstalledTaskPlan()
	if e != nil {
		return ServiceInstallResult{}, e
	}
	if actual.DescriptorPath != p.DescriptorPath {
		return ServiceInstallResult{}, ErrOperationIdentity
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorService, UnitPath: p.DescriptorPath, SealedExecutablePath: p.ImagePath, StateRoot: p.StateRoot, Scope: "machine", Active: snapshot.State == 4 && len(snapshot.Instances) == 1, Verdict: Verdict{State: VerdictAccepted, Validator: "sealed-native-task-s4u-operator"}}, nil
}

func installWindowsSystemBroker(o ServiceInstallOptions) (ServiceInstallResult, error) {
	if o.SystemBroker == nil {
		return ServiceInstallResult{}, ErrOperationIdentity
	}
	roots, e := nativeWindowsSystemBrokerRoots()
	if e != nil {
		return ServiceInstallResult{}, e
	}
	if _, e = buildNativeSystemBrokerPlan("windows", o, roots); e != nil {
		return ServiceInstallResult{}, e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	unlock, e := lockWindowsNativeBootTask(ctx, o.SystemBroker.OperatorPrincipal)
	if e != nil {
		return ServiceInstallResult{}, e
	}
	defer unlock()
	if e = validateWindowsBrokerOperator(*o.SystemBroker, roots); e != nil {
		return ServiceInstallResult{}, e
	}
	return installNativeSystemBrokerWithOps("windows", o, roots, nativeSystemBrokerInstallOps{Stage: stageWindowsSystemBroker, Register: registerWindowsSystemBroker})
}

func readNativeSystemServiceBrokerDescriptor(path string) (NativeSystemServiceBrokerOptions, error) {
	var zero NativeSystemServiceBrokerOptions
	d, p, _, e := windowsInstalledTaskPlan()
	if e != nil {
		return zero, e
	}
	if p.DescriptorPath != path || len(os.Args) != 3 || os.Args[1] != NativeSystemSupervisorVerb || os.Args[2] != path {
		return zero, ErrOperationIdentity
	}
	security, e := ProcessSecurity(os.Getpid())
	if e != nil {
		return zero, e
	}
	if security.Elevated || "windows:"+security.Principal != d.Options.OperatorPrincipal {
		return zero, ErrOperationIdentity
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pid, e := nativeSystemSupervisorPID(ctx)
	if e != nil {
		return zero, e
	}
	if pid != os.Getpid() {
		return zero, ErrOperationIdentity
	}
	return d.Options, nil
}

func runNativeSystemSupervisor(ctx context.Context, o NativeSystemServiceBrokerOptions, run func(context.Context) error) error {
	roots, e := nativeWindowsSystemBrokerRoots()
	if e != nil {
		return e
	}
	descriptor := nativeSystemBrokerDescriptorPath(filepath.Join(roots.Install, "versions", o.ExecutableSHA256, "vrooli.exe"), o)
	actual, e := readNativeSystemServiceBrokerDescriptor(descriptor)
	if e != nil {
		return e
	}
	if actual != o || run == nil {
		return ErrOperationIdentity
	}
	if e = os.Setenv("HOME", o.OperatorHome); e != nil {
		return e
	}
	if e = os.Setenv("VROOLI_RUNTIME_SUPERVISOR", "on"); e != nil {
		return e
	}
	if e = os.Setenv("USERPROFILE", o.OperatorHome); e != nil {
		return e
	}
	if e = os.Chdir(o.OperatorHome); e != nil {
		return e
	}
	return run(ctx)
}

func uninstallWindowsSystemBroker(o ServiceInstallOptions) (ServiceInstallResult, error) {
	if o.SystemBroker == nil {
		return ServiceInstallResult{}, ErrOperationIdentity
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	unlock, lockErr := lockWindowsNativeBootTask(ctx, o.SystemBroker.OperatorPrincipal)
	if lockErr != nil {
		return ServiceInstallResult{}, lockErr
	}
	defer unlock()
	d, p, snapshot, e := windowsInstalledTaskPlan()
	if taskNativeErrorMissing(e) {
		return ServiceInstallResult{UnitName: runtimeSupervisorService, Scope: "machine"}, nil
	}
	if e != nil {
		return ServiceInstallResult{}, e
	}
	if d.Options != *o.SystemBroker {
		return ServiceInstallResult{}, ErrOperationIdentity
	}
	if e = stopWindowsOperatorTask(ctx, snapshot, true); e != nil {
		return ServiceInstallResult{}, e
	}
	if e = deleteWindowsQuiescentTask(func() (WindowsTaskSnapshot, error) {
		current, _, snapshot, err := windowsInstalledTaskPlan()
		if err == nil && current.Options != d.Options {
			err = ErrOperationIdentity
		}
		return snapshot, err
	}, windowsDeleteTask, taskNativeErrorMissing); e != nil {
		return ServiceInstallResult{}, e
	}
	return ServiceInstallResult{UnitName: runtimeSupervisorService, UnitPath: p.DescriptorPath, SealedExecutablePath: p.ImagePath, StateRoot: p.StateRoot, Scope: "machine"}, nil
}

func startWindowsSystemBroker(o ServiceInstallOptions) (bool, error) {
	if o.SystemBroker == nil {
		return false, ErrOperationIdentity
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	unlock, lockErr := lockWindowsNativeBootTask(ctx, o.SystemBroker.OperatorPrincipal)
	if lockErr != nil {
		return false, lockErr
	}
	defer unlock()
	d, _, snapshot, e := windowsInstalledTaskPlan()
	if taskNativeErrorMissing(e) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if d.Options != *o.SystemBroker {
		return false, ErrOperationIdentity
	}
	run, e := windowsTaskNeedsRun(snapshot)
	if e != nil {
		return false, e
	}
	if run {
		if e = windowsTaskControl(runtimeSupervisorService, "start"); e != nil {
			return false, e
		}
	}
	if e = awaitWindowsTaskAction(ctx); e != nil {
		return false, e
	}
	return true, nil
}

func validateWindowsBrokerOperator(o NativeSystemServiceBrokerOptions, roots nativeSystemBrokerRoots) error {
	if !canonicalNativeBrokerWindowsUserPrincipal(o.OperatorPrincipal) {
		return ErrOperationIdentity
	}
	sid, _, kind, e := windows.LookupSID("", o.OperatorUsername)
	if e != nil {
		return e
	}
	if kind != windows.SidTypeUser || "windows:"+sid.String() != o.OperatorPrincipal {
		return ErrOperationIdentity
	}
	key, e := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\`+sid.String(), registry.QUERY_VALUE)
	if e != nil {
		return fmt.Errorf("platform: broker operator profile must already be initialized: %w", e)
	}
	defer key.Close()
	home, _, e := key.GetStringValue("ProfileImagePath")
	if e != nil {
		return e
	}
	home = strings.ReplaceAll(home, "%SystemDrive%", filepath.VolumeName(roots.SystemRoot))
	home = strings.ReplaceAll(home, "%SystemRoot%", roots.SystemRoot)
	if strings.Contains(home, "%") || !filepath.IsAbs(home) || !strings.EqualFold(filepath.Clean(home), o.OperatorHome) {
		return fmt.Errorf("%w: broker operator home differs from native SID profile", ErrOperationIdentity)
	}
	return nil
}
