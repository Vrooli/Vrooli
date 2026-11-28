//go:build windows

package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows cannot replace the current process from an open file. Native
// wrappers use the explicit suspended child handoff instead.
func replaceProcessImage(*os.File, []string, []string) error { return ErrUnsupported }

func lockedWindowsImage(file *os.File) (*os.File, error) {
	info, e := file.Stat()
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() || !filepath.IsAbs(file.Name()) {
		return nil, errors.New("platform: verified image must be regular with an absolute path")
	}
	name, e := windows.UTF16PtrFromString(file.Name())
	if e != nil {
		return nil, e
	}
	h, e := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		return nil, e
	}
	locked := os.NewFile(uintptr(h), file.Name())
	observed, e := locked.Stat()
	if e != nil || !os.SameFile(info, observed) {
		locked.Close()
		if e != nil {
			return nil, e
		}
		return nil, ErrOperationIdentity
	}
	return locked, nil
}
func windowsImageID(file *os.File) (string, error) {
	var info windows.ByHandleFileInformation
	if e := windows.GetFileInformationByHandle(windows.Handle(file.Fd()), &info); e != nil {
		return "", e
	}
	return fmt.Sprintf("windows-file:%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow), nil
}

var windowsIsProcessInJob = windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob")

func windowsProcessInExactJob(process, job windows.Handle) (bool, error) {
	var found uint32
	ok, _, e := windowsIsProcessInJob.Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&found)))
	if ok == 0 {
		return false, e
	}
	return found != 0, nil
}
func currentWindowsOwnedJob() (windowsOwnedPayload, error) {
	p, e := bootstrapWindowsWrapperCustody()
	if e != nil {
		return p, e
	}

	record, e := readWindowsOwnedRecord(p.Intent)
	if e != nil {
		return p, e
	}
	stamp, alive, e := windowsOwnedProcessStart(os.Getpid())
	if e != nil {
		return p, e
	}
	if !alive || record.PID != os.Getpid() || record.Start != stamp {
		return p, ErrOperationIdentity
	}
	// Custody is retained by this trusted wrapper, never by the SDK child.
	// Query the exact held Job, not implicit hJob=0 (which is ambiguous under
	// nested Task Scheduler/parent Jobs).
	windowsWrapperCustody.Lock()
	job := windowsWrapperCustody.job
	windowsWrapperCustody.Unlock()
	if uint64(job) != record.JobHandle {
		return p, ErrOperationIdentity
	}
	if e = validateWindowsOwnedJobHandle(job, p); e != nil {
		return p, e
	}
	inside, e := windowsProcessInExactJob(windows.CurrentProcess(), job)
	if e != nil {
		return p, e
	}
	if !inside {
		return p, ErrOperationIdentity
	}

	return p, nil
}
func prepareOwnedImageChild(file *os.File, argv, env []string, dir string) (*OwnedImageChild, error) {
	if e := validateWindowsFrozenText(file.Name(), dir, argv, env); e != nil {
		return nil, e
	}
	locked, e := lockedWindowsImage(file)
	if e != nil {
		return nil, e
	}
	fail := func(e error) (*OwnedImageChild, error) { locked.Close(); return nil, e }
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return fail(errors.New("platform: child working directory must be clean and absolute"))
	}
	// Validate fixed arguments and case-insensitive explicit environment before
	// any process is created. A shell or interpreter is never inserted.
	spec := OperationSpec{Path: locked.Name(), Args: argv[1:], Env: env, Dir: dir, Limits: OperationLimits{MemoryMaxBytes: 1, TasksMax: minimumTasksMax, CPUWeight: 1}}
	if e = validateOwnedProcessSpec(spec); e != nil {
		return fail(e)
	}
	seen := map[string]bool{}
	for _, entry := range env {
		k, _, _ := strings.Cut(entry, "=")
		k = strings.ToUpper(k)
		if seen[k] {
			return fail(errors.New("platform: duplicate Windows environment key"))
		}
		seen[k] = true
	}
	p, e := currentWindowsOwnedJob()
	if e != nil {
		return fail(e)
	}
	windowsWrapperCustody.Lock()
	job := windowsWrapperCustody.job
	windowsWrapperCustody.Unlock()
	if job == 0 {
		return fail(ErrOperationIdentity)
	}
	command := syscall.EscapeArg(locked.Name())
	for _, a := range argv[1:] {
		command += " " + syscall.EscapeArg(a)
	}
	if len(command) > 30000 {
		return fail(errors.New("platform: Windows command line exceeds bound"))
	}
	app, _ := windows.UTF16PtrFromString(locked.Name())
	cmd, _ := windows.UTF16PtrFromString(command)
	cwd, _ := windows.UTF16PtrFromString(dir)
	entries := append([]string(nil), env...)
	sort.Slice(entries, func(i, j int) bool { return strings.ToUpper(entries[i]) < strings.ToUpper(entries[j]) })
	block := []uint16{}
	for _, entry := range entries {
		v, _ := windows.UTF16FromString(entry)
		block = append(block, v...)
	}
	block = append(block, 0)
	if len(block) == 1 {
		block = append(block, 0)
	}
	var pi windows.ProcessInformation
	expectedSecurity, e := windowsProcessOperatorSecurityProof(windows.CurrentProcess(), p.Spec.NativeAccount)
	if e != nil {
		return fail(e)
	}
	// The child retains its own read-only, no-write/no-delete image lock even
	// if this wrapper crashes. Only this handle is inherited, never Job handles.
	lockHandle := windows.Handle(locked.Fd())
	if e = windows.SetHandleInformation(lockHandle, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); e != nil {
		return fail(e)
	}
	defer windows.SetHandleInformation(lockHandle, windows.HANDLE_FLAG_INHERIT, 0)
	attrs, e := windows.NewProcThreadAttributeList(1)
	if e != nil {
		return fail(e)
	}
	defer attrs.Delete()
	if e = attrs.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&lockHandle), unsafe.Sizeof(lockHandle)); e != nil {
		return fail(e)
	}
	si := windows.StartupInfoEx{StartupInfo: windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{}))}, ProcThreadAttributeList: attrs.List()}
	// The wrapper is already inside the no-breakaway job. Windows inherits
	// membership atomically at birth, before any suspended child is visible.
	if e = windows.CreateProcess(app, cmd, nil, nil, true, windows.CREATE_SUSPENDED|windows.CREATE_UNICODE_ENVIRONMENT|windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_NO_WINDOW, &block[0], cwd, &si.StartupInfo, &pi); e != nil {
		return fail(e)
	}
	abortFailure := func(e error) (*OwnedImageChild, error) {
		windows.TerminateProcess(pi.Process, 1)
		windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		windows.CloseHandle(pi.Thread)
		windows.CloseHandle(pi.Process)
		return fail(e)
	}
	actualSecurity, e := windowsProcessOperatorSecurityProof(pi.Process, p.Spec.NativeAccount)
	if e != nil {
		return abortFailure(e)
	}
	if actualSecurity != expectedSecurity {
		return abortFailure(ErrOperationIdentity)
	}
	inside, e := windowsProcessInExactJob(pi.Process, job)
	if e != nil || !inside {
		if e == nil {
			e = ErrOperationIdentity
		}
		return abortFailure(e)
	}
	mappedPath, e := processExecutablePath(int(pi.ProcessId))
	if e != nil {
		return abortFailure(e)
	}
	mapped, e := os.Open(mappedPath)
	if e != nil {
		return abortFailure(e)
	}
	observed, e := mapped.Stat()
	mapped.Close()
	original, statErr := locked.Stat()
	if e != nil || statErr != nil || !os.SameFile(original, observed) {
		if e == nil {
			e = ErrOperationIdentity
		}
		return abortFailure(e)
	}
	stamp, alive, e := windowsOwnedProcessStart(int(pi.ProcessId))
	if e != nil || !alive {
		if e == nil {
			e = ErrOperationIdentity
		}
		return abortFailure(e)
	}
	ticks, _ := strconv.ParseUint(stamp, 10, 64)
	creation := (&windows.Filetime{HighDateTime: uint32(ticks >> 32), LowDateTime: uint32(ticks)}).Nanoseconds()
	imageID, e := windowsImageID(locked)
	if e != nil {
		return abortFailure(e)
	}
	child := &OwnedImageChild{PID: int(pi.ProcessId), Incarnation: fmt.Sprintf("windows:%d:%d", pi.ProcessId, creation), ImageID: imageID, ImageHandle: uint64(lockHandle), NativeAccount: windowsOwnedAccount(p.Intent), ContainmentID: windowsOwnedJob(p.Intent), SecurityProof: actualSecurity}
	var mu sync.Mutex
	resumed, closed := false, false
	var waitErr error
	closeHandles := func() { windows.CloseHandle(pi.Thread); windows.CloseHandle(pi.Process); locked.Close(); closed = true }
	child.resume = func() error {
		mu.Lock()
		defer mu.Unlock()
		if closed || resumed {
			return ErrOperationIdentity
		}
		proof, e := windowsProcessOperatorSecurityProof(pi.Process, p.Spec.NativeAccount)
		if e != nil {
			return e
		}
		if proof != child.SecurityProof {
			return ErrOperationIdentity
		}
		if _, e := windows.ResumeThread(pi.Thread); e != nil {
			return e
		}
		resumed = true
		return nil
	}
	child.abort = func() error {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return waitErr
		}
		if resumed {
			return ErrOperationNotQuiescent
		}
		if e := windows.TerminateProcess(pi.Process, 1); e != nil {
			return e
		}
		if _, e := windows.WaitForSingleObject(pi.Process, windows.INFINITE); e != nil {
			return e
		}
		closeHandles()
		return nil
	}
	child.wait = func() error {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return waitErr
		}
		if !resumed {
			return errors.New("platform: child claim must be persisted and resumed before wait")
		}
		if _, e := windows.WaitForSingleObject(pi.Process, windows.INFINITE); e != nil {
			return e
		}
		var exit uint32
		if e := windows.GetExitCodeProcess(pi.Process, &exit); e != nil {
			return e
		}
		if exit != 0 {
			waitErr = fmt.Errorf("platform: image child exited with code %d", exit)
		}
		closeHandles()
		return waitErr
	}
	return child, nil
}

func verifyWindowsMappedImageHandle(process windows.Handle, locked *os.File) error {
	var name [32768]uint16
	size := uint32(len(name))
	if e := windows.QueryFullProcessImageName(process, 0, &name[0], &size); e != nil {
		return e
	}
	mapped, e := os.Open(windows.UTF16ToString(name[:size]))
	if e != nil {
		return e
	}
	defer mapped.Close()
	a, e := mapped.Stat()
	if e != nil {
		return e
	}
	b, e := locked.Stat()
	if e != nil {
		return e
	}
	if !os.SameFile(a, b) {
		return ErrOperationIdentity
	}
	return nil
}

// This narrow observer requires the sealed operator Job root record and inherited
// birth-lock handle. A pathname by itself never proves mapped image identity.
func openOwnedLauncherImage(ref ServiceRef) (*os.File, error) {
	p, e := currentWindowsOwnedJob()
	if e != nil {
		return nil, e
	}
	if p.Intent != ref.Intent {
		return nil, ErrOperationIdentity
	}
	actual, e := lookupOwnedOperation(context.Background(), p.Intent)
	if e != nil {
		return nil, e
	}
	if !actual.Equal(OperationRef(ref)) {
		return nil, ErrOperationIdentity
	}
	record, e := readWindowsOwnedRecord(p.Intent)
	if e != nil {
		return nil, e
	}
	if record.ImageHandle == 0 || record.ImageID == "" {
		return nil, ErrOperationIdentity
	}
	var duplicate windows.Handle
	if e = windows.DuplicateHandle(windows.CurrentProcess(), windows.Handle(record.ImageHandle), windows.CurrentProcess(), &duplicate, 0, false, windows.DUPLICATE_SAME_ACCESS); e != nil {
		return nil, e
	}
	image := os.NewFile(uintptr(duplicate), p.Spec.Path)
	fail := func(e error) (*os.File, error) { image.Close(); return nil, e }
	id, e := windowsImageID(image)
	if e != nil {
		return fail(e)
	}
	if id != record.ImageID {
		return fail(ErrOperationIdentity)
	}
	if e = verifyWindowsMappedImageHandle(windows.CurrentProcess(), image); e != nil {
		return fail(e)
	}
	expected, e := os.Open(p.Spec.Path)
	if e != nil {
		return fail(e)
	}
	a, e := expected.Stat()
	expected.Close()
	if e != nil {
		return fail(e)
	}
	b, e := image.Stat()
	if e != nil {
		return fail(e)
	}
	if !os.SameFile(a, b) {
		return fail(ErrOperationIdentity)
	}
	return image, nil
}

func verifyOwnedImageChild(ctx context.Context, ref ServiceRef, id OwnedImageChildIdentity, image *os.File) error {
	if id.ImageHandle == 0 {
		return ErrOperationIdentity
	}
	if ref.Native == nil || ref.Native.Platform != "windows" || id.NativeAccount != ref.Native.Account || id.ContainmentID != ref.Native.ContainmentID {
		return ErrOperationIdentity
	}
	observeRoot := func() error {
		o, e := InspectOwnedService(ctx, ref)
		if e != nil {
			return e
		}
		if !o.Ref.Equal(OperationRef(ref)) || o.State != OperationRunning || !o.Populated || !o.RootAlive {
			return ErrOperationIdentity
		}
		return nil
	}
	if e := observeRoot(); e != nil {
		return e
	}
	locked, e := lockedWindowsImage(image)
	if e != nil {
		return e
	}
	defer locked.Close()
	imageID, e := windowsImageID(locked)
	if e != nil {
		return e
	}
	if id.ImageID != imageID {
		return ErrOperationIdentity
	}
	process, e := windowsServingProcess(id.PID, id.Incarnation)
	if e != nil {
		return e
	}
	defer windows.CloseHandle(process)
	birth, e := windowsServingBirthImage(process, id.ImageHandle, id.ImageID)
	if e != nil {
		return e
	}
	birthInfo, e := birth.Stat()
	birth.Close()
	if e != nil {
		return e
	}
	expectedInfo, e := locked.Stat()
	if e != nil {
		return e
	}
	if !os.SameFile(birthInfo, expectedInfo) {
		return ErrOperationIdentity
	}
	alive := func() error {
		state, e := windows.WaitForSingleObject(process, 0)
		if e != nil {
			return e
		}
		if state != uint32(windows.WAIT_TIMEOUT) {
			return ErrOperationIdentity
		}
		return ctx.Err()
	}
	if e = alive(); e != nil {
		return e
	}
	var c, x, k, u windows.Filetime
	if e = windows.GetProcessTimes(process, &c, &x, &k, &u); e != nil {
		return e
	}
	if id.Incarnation != fmt.Sprintf("windows:%d:%d", id.PID, c.Nanoseconds()) {
		return ErrOperationIdentity
	}
	var token windows.Token
	if e = windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); e != nil {
		return e
	}
	defer token.Close()
	if !validNativeBrokerDigest(id.SecurityProof) {
		return ErrOperationIdentity
	}
	securityProof, e := windowsOperatorTokenProof(token)
	if e != nil {
		return e
	}
	if securityProof != id.SecurityProof {
		return ErrOperationIdentity
	}
	user, e := token.GetTokenUser()
	if e != nil {
		return e
	}
	expected, e := windows.StringToSid(strings.TrimPrefix(ref.Native.Account, "sid:"))
	if e != nil {
		return e
	}
	if !windows.EqualSid(user.User.Sid, expected) {
		return ErrOperationIdentity
	}
	// The verifier is the authorized owner, not the payload. Its ephemeral named
	// Job handle is closed before the final root inspection/claim boundary.
	job, e := openWindowsOwnedJob(ref.Native.ContainmentID)
	if e != nil {
		return e
	}
	if job == 0 {
		return ErrOperationIdentity
	}
	contained, e := windowsProcessInExactJob(process, job)
	windows.CloseHandle(job)
	if e != nil {
		return e
	}
	if !contained {
		return ErrOperationIdentity
	}
	if e = verifyWindowsMappedImageHandle(process, locked); e != nil {
		return e
	}
	if e = alive(); e != nil {
		return e
	}
	if e = observeRoot(); e != nil {
		return e
	}
	return alive()
}
