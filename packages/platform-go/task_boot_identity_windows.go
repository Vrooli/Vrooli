//go:build windows

package platform

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unsafe"
)

func openInstalledNativeSupervisorImage(ctx context.Context) (*os.File, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	_, p, snapshot, e := windowsInstalledTaskPlan()
	if e != nil {
		return nil, e
	}
	h, e := windowsOpenBrokerPath(p.ImagePath, false, windows.GENERIC_READ)
	if e != nil {
		return nil, e
	}
	source := os.NewFile(uintptr(h), p.ImagePath)
	image, e := lockedWindowsImage(source)
	source.Close()
	if e != nil {
		return nil, e
	}
	fail := func(e error) (*os.File, error) { image.Close(); return nil, e }
	if e = verifyNativeSystemBrokerImage(image, p.SHA256); e != nil {
		return fail(e)
	}
	_, confirmed, after, e := windowsInstalledTaskPlan()
	if e != nil {
		return fail(e)
	}
	if confirmed.DescriptorPath != p.DescriptorPath || snapshot.XML != after.XML || snapshot.SecurityDescriptor != after.SecurityDescriptor {
		return fail(ErrOperationIdentity)
	}
	if e = ctx.Err(); e != nil {
		return fail(e)
	}
	return image, nil
}

// nativeSystemSupervisorPID binds a native task instance to its actual Exec
// process. EnginePID alone is not an action identity: the held action must
// either be that verified image itself or have the held engine as its kernel
// parent, with ordered birth times. All action fields and the manager GUID are
// confirmed again after inspection. Uncertain/ambiguous mappings fail closed.
func nativeSystemSupervisorPID(ctx context.Context) (int, error) {
	if e := ctx.Err(); e != nil {
		return 0, e
	}
	d, p, before, e := windowsInstalledTaskPlan()
	if e != nil {
		return 0, e
	}
	if before.State != 4 || len(before.Instances) != 1 {
		return 0, ErrOperationNotFound
	}
	instance := before.Instances[0]
	if instance.EnginePID == 0 || instance.State != 4 {
		return 0, ErrOperationNotFound
	}
	if instance.GUID == "" {
		return 0, ErrOperationIdentity
	}
	engine, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, instance.EnginePID)
	if e != nil {
		return 0, e
	}
	defer windows.CloseHandle(engine)
	var engineBirth, exit, kernel, user windows.Filetime
	if e = windows.GetProcessTimes(engine, &engineBirth, &exit, &kernel, &user); e != nil {
		return 0, e
	}
	imageHandle, e := windowsOpenBrokerPath(p.ImagePath, false, windows.GENERIC_READ)
	if e != nil {
		return 0, e
	}
	source := os.NewFile(uintptr(imageHandle), p.ImagePath)
	image, e := lockedWindowsImage(source)
	source.Close()
	if e != nil {
		return 0, e
	}
	defer image.Close()
	snapshot, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return 0, e
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	var held windows.Handle
	var found int
	defer func() {
		if held != 0 {
			windows.CloseHandle(held)
		}
	}()
	for e = windows.Process32First(snapshot, &entry); e == nil; e = windows.Process32Next(snapshot, &entry) {
		if !strings.EqualFold(windows.UTF16ToString(entry.ExeFile[:]), filepath.Base(p.ImagePath)) {
			continue
		}
		if entry.ProcessID != instance.EnginePID && entry.ParentProcessID != instance.EnginePID {
			continue
		}
		process, openErr := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ|windows.SYNCHRONIZE, false, entry.ProcessID)
		if errors.Is(openErr, windows.ERROR_INVALID_PARAMETER) {
			continue
		}
		if openErr != nil {
			return 0, openErr
		}
		matches, matchErr := windowsTaskActionMatches(process, int(entry.ProcessID), instance, engineBirth, d, p, image)
		if matchErr != nil {
			windows.CloseHandle(process)
			return 0, matchErr
		}
		if !matches {
			windows.CloseHandle(process)
			continue
		}
		if found != 0 {
			windows.CloseHandle(process)
			return 0, ErrOperationIdentity
		}
		found = int(entry.ProcessID)
		held = process
	}
	if !errors.Is(e, windows.ERROR_NO_MORE_FILES) {
		return 0, e
	}
	if found == 0 {
		return 0, ErrOperationNotFound
	}
	_, afterPlan, after, e := windowsInstalledTaskPlan()
	if e != nil {
		return 0, e
	}
	if afterPlan.DescriptorPath != p.DescriptorPath || !windowsTaskSameIncarnation(before, after) {
		return 0, ErrOperationIdentity
	}
	if status, e := windows.WaitForSingleObject(held, 0); e != nil || status != uint32(windows.WAIT_TIMEOUT) {
		return 0, ErrOperationNotFound
	}
	if status, e := windows.WaitForSingleObject(engine, 0); e != nil || status != uint32(windows.WAIT_TIMEOUT) {
		return 0, ErrOperationNotFound
	}
	return found, ctx.Err()
}
func windowsTaskActionMatches(process windows.Handle, pid int, instance WindowsTaskInstance, engineBirth windows.Filetime, d nativeSystemBrokerDescriptor, p nativeSystemBrokerPlan, image *os.File) (bool, error) {
	if e := verifyWindowsMappedImageHandle(process, image); e != nil {
		if errors.Is(e, ErrOperationIdentity) {
			return false, nil
		}
		return false, e
	}
	var basic windows.PROCESS_BASIC_INFORMATION
	if e := windows.NtQueryInformationProcess(process, windows.ProcessBasicInformation, unsafe.Pointer(&basic), uint32(unsafe.Sizeof(basic)), nil); e != nil {
		return false, e
	}
	if basic.UniqueProcessId != uintptr(pid) {
		return false, nil
	}
	var birth, exit, kernel, user windows.Filetime
	if e := windows.GetProcessTimes(process, &birth, &exit, &kernel, &user); e != nil {
		return false, e
	}
	if !windowsTaskActionRelation(uint32(pid), uint32(basic.InheritedFromUniqueProcessId), uint64(birth.HighDateTime)<<32|uint64(birth.LowDateTime), instance, uint64(engineBirth.HighDateTime)<<32|uint64(engineBirth.LowDateTime)) {
		return false, ErrOperationIdentity
	}
	security, e := ProcessSecurity(pid)
	if e != nil {
		return false, e
	}
	if security.Elevated || "windows:"+security.Principal != d.Options.OperatorPrincipal {
		return false, ErrOperationIdentity
	}
	args, e := processArguments(pid)
	if e != nil {
		return false, e
	}
	if len(args) != 3 || !strings.EqualFold(args[0], p.ImagePath) || !slices.Equal(args[1:], p.Definition.Args) {
		return false, nil
	}
	return true, nil
}
func verifyWindowsSystemBrokerDispatch(ctx context.Context, o NativeSystemServiceBrokerOptions) error {
	pid, e := nativeSystemSupervisorPID(ctx)
	if e != nil {
		return e
	}
	if pid != os.Getpid() {
		return ErrOperationIdentity
	}
	_, p, _, e := windowsInstalledTaskPlan()
	if e != nil {
		return e
	}
	actual, e := readNativeSystemServiceBrokerDescriptor(p.DescriptorPath)
	if e != nil {
		return e
	}
	if actual != o {
		return ErrOperationIdentity
	}
	return nil
}
func nativeSystemSupervisorStateRoot(ctx context.Context) (string, error) {
	pid, e := nativeSystemSupervisorPID(ctx)
	if e != nil {
		return "", e
	}
	if pid != os.Getpid() {
		return "", ErrOperationIdentity
	}
	d, _, _, e := windowsInstalledTaskPlan()
	if e != nil {
		return "", e
	}
	return d.Options.OperatorHome, ctx.Err()
}

func awaitWindowsTaskQuiescent(ctx context.Context) error {
	return waitWindowsTaskQuiescent(ctx, func() (WindowsTaskSnapshot, error) {
		_, _, snapshot, err := windowsInstalledTaskPlan()
		return snapshot, err
	})
}
func awaitWindowsTaskAction(ctx context.Context) error {
	for {
		_, e := nativeSystemSupervisorPID(ctx)
		if e == nil {
			return nil
		}
		if !errors.Is(e, ErrOperationNotFound) {
			return e
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func stopWindowsOperatorTask(ctx context.Context, snapshot WindowsTaskSnapshot, disableBoot bool) error {
	if len(snapshot.Instances) > 1 {
		return ErrOperationIdentity
	}
	var process windows.Handle
	if len(snapshot.Instances) == 1 {
		pid, e := nativeSystemSupervisorPID(ctx)
		if e != nil {
			return e
		}
		process, e = windows.OpenProcess(windows.SYNCHRONIZE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
		if e != nil {
			return e
		}
		defer windows.CloseHandle(process)
		confirmed, e := nativeSystemSupervisorPID(ctx)
		if e != nil {
			return e
		}
		current, e := WindowsTaskStatus(runtimeSupervisorService)
		if e != nil {
			return e
		}
		if confirmed != pid || !windowsTaskSameIncarnation(snapshot, current) {
			return ErrOperationIdentity
		}
		// The opened handle must still denote the live process just proven.
		// A PID recycled between observations must not leave us waiting on an
		// already-exited predecessor while a replacement action continues.
		if state, e := windows.WaitForSingleObject(process, 0); e != nil || state != uint32(windows.WAIT_TIMEOUT) {
			return ErrOperationIdentity
		}
	}
	if disableBoot {
		if e := windowsSetTaskEnabled(runtimeSupervisorService, false); e != nil {
			return e
		}
	}
	// GetInstances omits queued work. Cancel the exact registered task on every
	// attempt, including a queued or currently unenumerated action; only its
	// native S_OK acknowledges cancellation.
	if e := windowsStopRegisteredTask(runtimeSupervisorService); e != nil {
		return e
	}
	if e := awaitWindowsTaskQuiescent(ctx); e != nil {
		return e
	}
	if disableBoot {
		_, _, current, e := windowsInstalledTaskPlan()
		if e != nil {
			return e
		}
		quiet, e := windowsTaskQuiescent(current)
		if e != nil {
			return e
		}
		if current.Enabled || !quiet {
			return ErrOperationNotQuiescent
		}
	}
	if process != 0 {
		for {
			status, e := windows.WaitForSingleObject(process, 50)
			if e != nil {
				return e
			}
			if status == windows.WAIT_OBJECT_0 {
				break
			}
			if status != uint32(windows.WAIT_TIMEOUT) {
				return ErrOperationIdentity
			}
			if e = ctx.Err(); e != nil {
				return e
			}
		}
	}
	return nil
}

// Ordinary stop has read/execute task authority only. Definition mutation is
// reserved for setup/removal; runtime policy owns whether managed jobs run.
func controlWindowsInstalledOperatorTask(ctx context.Context, verb string) error {
	d, _, _, e := windowsInstalledTaskPlan()
	if e != nil {
		return e
	}
	unlock, e := lockWindowsNativeBootTask(ctx, d.Options.OperatorPrincipal)
	if e != nil {
		return e
	}
	defer unlock()
	confirmed, _, snapshot, e := windowsInstalledTaskPlan()
	if e != nil {
		return e
	}
	if confirmed.Options != d.Options {
		return ErrOperationIdentity
	}
	return controlWindowsOperatorTaskWith(ctx, verb, snapshot, windowsOperatorTaskControlEffects{
		stop: func(ctx context.Context, snapshot WindowsTaskSnapshot) error {
			return stopWindowsOperatorTask(ctx, snapshot, false)
		},
		query: func() (WindowsTaskSnapshot, error) {
			current, _, snapshot, err := windowsInstalledTaskPlan()
			if err == nil && current.Options != d.Options {
				err = ErrOperationIdentity
			}
			return snapshot, err
		},
		run: func() error { return windowsTaskControl(runtimeSupervisorService, "start") }, await: awaitWindowsTaskAction,
	})
}
